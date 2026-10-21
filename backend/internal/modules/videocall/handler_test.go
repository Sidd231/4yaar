package videocall

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sidd231/4yaar/backend/internal/middleware"
	"github.com/Sidd231/4yaar/backend/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type mockVideoService struct {
	createRoomFunc   func(ctx context.Context, name string, creatorID uuid.UUID) (*models.Room, error)
	getRoomInfoFunc  func(ctx context.Context, roomID, userID uuid.UUID) (*VideoRoomInfo, error)
	isUserInRoomFunc func(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
}

func (m *mockVideoService) CreateRoom(ctx context.Context, name string, creatorID uuid.UUID) (*models.Room, error) {
	if m.createRoomFunc != nil {
		return m.createRoomFunc(ctx, name, creatorID)
	}
	return nil, nil
}

func (m *mockVideoService) GetRoomInfo(ctx context.Context, roomID, userID uuid.UUID) (*VideoRoomInfo, error) {
	if m.getRoomInfoFunc != nil {
		return m.getRoomInfoFunc(ctx, roomID, userID)
	}
	return nil, nil
}

func (m *mockVideoService) IsUserInRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	if m.isUserInRoomFunc != nil {
		return m.isUserInRoomFunc(ctx, roomID, userID)
	}
	return false, nil
}

const testJWTSecret = "test-video-super-secret-key-at-least-32-bytes!!"

func createTestToken(userID uuid.UUID, email, tokenType string) string {
	claims := &middleware.UserClaims{
		UserID:    userID,
		Email:     email,
		Role:      models.RoleMember,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(testJWTSecret))
	return tokenStr
}

func TestVideoCallHandler_GetRoomInfo(t *testing.T) {
	roomID := uuid.New()
	memberID := uuid.New()
	nonMemberID := uuid.New()

	mockSvc := &mockVideoService{
		getRoomInfoFunc: func(ctx context.Context, rID, uID uuid.UUID) (*VideoRoomInfo, error) {
			if rID != roomID {
				return nil, ErrVideoRoomNotFound
			}
			if uID != memberID {
				return nil, ErrVideoRoomUnauthorized
			}
			return &VideoRoomInfo{
				ID:              roomID,
				Name:            "Conference Alpha",
				CreatedBy:       memberID,
				ActivePeerCount: 1,
				CreatedAt:       time.Now(),
			}, nil
		},
	}

	hub := NewHub()
	handler := NewHandler(mockSvc, hub, testJWTSecret, []string{"http://localhost:3000"})

	r := chi.NewRouter()
	r.Get("/video/rooms/{roomID}", handler.GetRoomInfo)

	t.Run("member receives 200 OK with room details", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/video/rooms/"+roomID.String(), nil)
		claims := &middleware.UserClaims{UserID: memberID, TokenType: middleware.TokenTypeAccess}
		ctx := context.WithValue(req.Context(), middleware.UserContextKey, claims)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Data VideoRoomInfo `json:"data"`
		}
		_ = json.NewDecoder(rec.Body).Decode(&resp)
		if resp.Data.ID != roomID || resp.Data.Name != "Conference Alpha" {
			t.Fatalf("unexpected room info: %+v", resp.Data)
		}
	})

	t.Run("non-member receives 403 Forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/video/rooms/"+roomID.String(), nil)
		claims := &middleware.UserClaims{UserID: nonMemberID, TokenType: middleware.TokenTypeAccess}
		ctx := context.WithValue(req.Context(), middleware.UserContextKey, claims)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("non-existent room receives 404 Not Found", func(t *testing.T) {
		unknownID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/video/rooms/"+unknownID.String(), nil)
		claims := &middleware.UserClaims{UserID: memberID, TokenType: middleware.TokenTypeAccess}
		ctx := context.WithValue(req.Context(), middleware.UserContextKey, claims)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("unauthenticated request receives 401 Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/video/rooms/"+roomID.String(), nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestVideoCallHandler_ServeWS(t *testing.T) {
	roomID := uuid.New()
	memberID := uuid.New()
	nonMemberID := uuid.New()

	mockSvc := &mockVideoService{
		isUserInRoomFunc: func(ctx context.Context, rID, uID uuid.UUID) (bool, error) {
			return rID == roomID && uID == memberID, nil
		},
	}

	hub := NewHub()
	go hub.Run()

	allowedOrigins := []string{"http://localhost:3000"}
	handler := NewHandler(mockSvc, hub, testJWTSecret, allowedOrigins)

	r := chi.NewRouter()
	r.Get("/ws/video/{roomID}", handler.ServeWS)

	server := httptest.NewServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/video/" + roomID.String()

	t.Run("member with access token and allowed origin connects successfully", func(t *testing.T) {
		token := createTestToken(memberID, "member@example.com", middleware.TokenTypeAccess)
		header := http.Header{}
		header.Set("Origin", "http://localhost:3000")
		header.Set("Authorization", "Bearer "+token)

		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err != nil {
			t.Fatalf("expected websocket connection to succeed, got error: %v", err)
		}
		defer conn.Close()
		if resp.StatusCode != http.StatusSwitchingProtocols {
			t.Fatalf("expected 101 Switching Protocols, got %d", resp.StatusCode)
		}
	})

	t.Run("non-member with access token rejected with 403 Forbidden before upgrade", func(t *testing.T) {
		token := createTestToken(nonMemberID, "nonmember@example.com", middleware.TokenTypeAccess)
		header := http.Header{}
		header.Set("Origin", "http://localhost:3000")
		header.Set("Authorization", "Bearer "+token)

		_, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err == nil {
			t.Fatal("expected connection to fail for non-member")
		}
		if resp == nil || resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %v", resp)
		}
	})

	t.Run("refresh token rejected with 401 Unauthorized before upgrade", func(t *testing.T) {
		refreshToken := createTestToken(memberID, "member@example.com", middleware.TokenTypeRefresh)
		header := http.Header{}
		header.Set("Origin", "http://localhost:3000")
		header.Set("Authorization", "Bearer "+refreshToken)

		_, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err == nil {
			t.Fatal("expected connection to fail when refresh token is presented")
		}
		if resp == nil || resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %v", resp)
		}
	})

	t.Run("disallowed origin rejected by origin check", func(t *testing.T) {
		token := createTestToken(memberID, "member@example.com", middleware.TokenTypeAccess)
		header := http.Header{}
		header.Set("Origin", "https://malicious-site.com")
		header.Set("Authorization", "Bearer "+token)

		_, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err == nil {
			t.Fatal("expected connection from disallowed origin to fail")
		}
		if resp == nil || resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for disallowed origin, got %v", resp)
		}
	})

	t.Run("missing token rejected with 401 Unauthorized", func(t *testing.T) {
		header := http.Header{}
		header.Set("Origin", "http://localhost:3000")

		_, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err == nil {
			t.Fatal("expected connection to fail with missing token")
		}
		if resp == nil || resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %v", resp)
		}
	})
}
