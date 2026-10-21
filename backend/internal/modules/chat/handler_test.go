package chat

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

type mockChatService struct {
	canAccessRoomFunc func(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
	addMemberFunc     func(ctx context.Context, roomID, targetUserID, requestingUserID uuid.UUID, requestingUserRole string) (*models.RoomMember, error)
}

func (m *mockChatService) CreateRoom(ctx context.Context, name, roomType string, creatorID uuid.UUID) (*models.Room, error) {
	return nil, nil
}

func (m *mockChatService) GetUserRooms(ctx context.Context, userID uuid.UUID) ([]models.Room, error) {
	return nil, nil
}

func (m *mockChatService) GetRoomMessages(ctx context.Context, roomID, userID uuid.UUID, cursor *time.Time, limit int) ([]models.Message, error) {
	return nil, nil
}

func (m *mockChatService) SaveMessage(ctx context.Context, roomID, senderID uuid.UUID, content, msgType string, fileID *uuid.UUID) (*models.Message, error) {
	return nil, nil
}

func (m *mockChatService) CanAccessRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	if m.canAccessRoomFunc != nil {
		return m.canAccessRoomFunc(ctx, roomID, userID)
	}
	return false, nil
}

func (m *mockChatService) AddMember(ctx context.Context, roomID, targetUserID, requestingUserID uuid.UUID, requestingUserRole string) (*models.RoomMember, error) {
	if m.addMemberFunc != nil {
		return m.addMemberFunc(ctx, roomID, targetUserID, requestingUserID, requestingUserRole)
	}
	return nil, nil
}

const testChatJWTSecret = "test-chat-super-secret-key-at-least-32-bytes!!"

func createChatToken(userID uuid.UUID, email, tokenType string) string {
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
	tokenStr, _ := token.SignedString([]byte(testChatJWTSecret))
	return tokenStr
}

func TestChatHandler_ServeWS(t *testing.T) {
	roomID := uuid.New()
	memberID := uuid.New()
	nonMemberID := uuid.New()

	mockSvc := &mockChatService{
		canAccessRoomFunc: func(ctx context.Context, rID, uID uuid.UUID) (bool, error) {
			return rID == roomID && uID == memberID, nil
		},
	}

	hub := &Hub{
		rooms:      make(map[uuid.UUID]map[*Client]bool),
		Register:   make(chan *Client, 10),
		Unregister: make(chan *Client, 10),
	}

	allowedOrigins := []string{"http://localhost:3000"}
	handler := NewHandler(mockSvc, hub, testChatJWTSecret, allowedOrigins)

	r := chi.NewRouter()
	r.Get("/ws/chat/{roomID}", handler.ServeWS)

	server := httptest.NewServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/chat/" + roomID.String()

	t.Run("member with access token and allowed origin connects successfully", func(t *testing.T) {
		token := createChatToken(memberID, "member@example.com", middleware.TokenTypeAccess)
		header := http.Header{}
		header.Set("Origin", "http://localhost:3000")
		header.Set("Authorization", "Bearer "+token)

		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err != nil {
			t.Fatalf("expected websocket connection to succeed, got: %v", err)
		}
		defer conn.Close()
		if resp.StatusCode != http.StatusSwitchingProtocols {
			t.Fatalf("expected 101, got %d", resp.StatusCode)
		}
	})

	t.Run("non-member rejected with 403 Forbidden before upgrade", func(t *testing.T) {
		token := createChatToken(nonMemberID, "nonmember@example.com", middleware.TokenTypeAccess)
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
		refreshToken := createChatToken(memberID, "member@example.com", middleware.TokenTypeRefresh)
		header := http.Header{}
		header.Set("Origin", "http://localhost:3000")
		header.Set("Authorization", "Bearer "+refreshToken)

		_, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err == nil {
			t.Fatal("expected connection to fail with refresh token")
		}
		if resp == nil || resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %v", resp)
		}
	})

	t.Run("disallowed origin rejected by CheckOrigin", func(t *testing.T) {
		token := createChatToken(memberID, "member@example.com", middleware.TokenTypeAccess)
		header := http.Header{}
		header.Set("Origin", "https://attacker.com")
		header.Set("Authorization", "Bearer "+token)

		_, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err == nil {
			t.Fatal("expected connection to fail for disallowed origin")
		}
		if resp == nil || resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %v", resp)
		}
	})
}

func TestChatHandler_AddMember(t *testing.T) {
	roomID := uuid.New()
	creatorID := uuid.New()
	targetUserID := uuid.New()
	nonCreatorID := uuid.New()

	mockSvc := &mockChatService{
		addMemberFunc: func(ctx context.Context, rID, tID, reqID uuid.UUID, role string) (*models.RoomMember, error) {
			if rID != roomID {
				return nil, ErrRoomNotFound
			}
			if reqID != creatorID && role != models.RoleAdmin {
				return nil, ErrNotAuthorized
			}
			if tID == creatorID {
				return nil, ErrAlreadyMember
			}
			return &models.RoomMember{
				RoomID:   rID,
				UserID:   tID,
				JoinedAt: time.Now(),
			}, nil
		},
	}

	handler := NewHandler(mockSvc, nil, testChatJWTSecret, []string{"http://localhost:3000"})

	r := chi.NewRouter()
	r.Post("/chat/rooms/{roomID}/members", handler.AddMember)

	t.Run("room creator adds user successfully", func(t *testing.T) {
		body := `{"user_id":"` + targetUserID.String() + `"}`
		req := httptest.NewRequest(http.MethodPost, "/chat/rooms/"+roomID.String()+"/members", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		claims := &middleware.UserClaims{UserID: creatorID, Role: models.RoleMember, TokenType: middleware.TokenTypeAccess}
		ctx := context.WithValue(req.Context(), middleware.UserContextKey, claims)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Data struct {
				RoomID   uuid.UUID `json:"room_id"`
				UserID   uuid.UUID `json:"user_id"`
				JoinedAt time.Time `json:"joined_at"`
			} `json:"data"`
			Error *string `json:"error"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Error != nil {
			t.Fatalf("expected nil error, got: %s", *resp.Error)
		}
		if resp.Data.RoomID != roomID || resp.Data.UserID != targetUserID {
			t.Fatalf("unexpected member data: %+v", resp.Data)
		}
	})

	t.Run("non-creator member rejected with 403 Forbidden", func(t *testing.T) {
		body := `{"user_id":"` + targetUserID.String() + `"}`
		req := httptest.NewRequest(http.MethodPost, "/chat/rooms/"+roomID.String()+"/members", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		claims := &middleware.UserClaims{UserID: nonCreatorID, Role: models.RoleMember, TokenType: middleware.TokenTypeAccess}
		ctx := context.WithValue(req.Context(), middleware.UserContextKey, claims)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("adding existing member rejected with 409 Conflict", func(t *testing.T) {
		body := `{"user_id":"` + creatorID.String() + `"}`
		req := httptest.NewRequest(http.MethodPost, "/chat/rooms/"+roomID.String()+"/members", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		claims := &middleware.UserClaims{UserID: creatorID, Role: models.RoleMember, TokenType: middleware.TokenTypeAccess}
		ctx := context.WithValue(req.Context(), middleware.UserContextKey, claims)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("invalid user_id returns 400 Bad Request", func(t *testing.T) {
		body := `{"user_id":"not-a-valid-uuid"}`
		req := httptest.NewRequest(http.MethodPost, "/chat/rooms/"+roomID.String()+"/members", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		claims := &middleware.UserClaims{UserID: creatorID, Role: models.RoleMember, TokenType: middleware.TokenTypeAccess}
		ctx := context.WithValue(req.Context(), middleware.UserContextKey, claims)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
		}
	})
}

