package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Sidd231/4yaar/backend/internal/middleware"
	"github.com/Sidd231/4yaar/backend/internal/models"
	"github.com/google/uuid"
)

type mockAuthService struct {
	getMeFunc func(ctx context.Context, userID uuid.UUID) (*UserDTO, error)
}

func (m *mockAuthService) Register(ctx context.Context, inviteToken uuid.UUID, name, password string) (*AuthResponse, error) {
	return nil, nil
}

func (m *mockAuthService) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	return nil, nil
}

func (m *mockAuthService) Refresh(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	return nil, nil
}

func (m *mockAuthService) Logout(ctx context.Context, refreshToken string) error {
	return nil
}

func (m *mockAuthService) GetMe(ctx context.Context, userID uuid.UUID) (*UserDTO, error) {
	if m.getMeFunc != nil {
		return m.getMeFunc(ctx, userID)
	}
	return nil, nil
}

func TestAuthHandler_GetMe(t *testing.T) {
	userID := uuid.New()
	mockSvc := &mockAuthService{
		getMeFunc: func(ctx context.Context, uID uuid.UUID) (*UserDTO, error) {
			if uID == userID {
				return &UserDTO{
					ID:        userID,
					Name:      "Alice",
					Email:     "alice@example.com",
					Role:      models.RoleMember,
					CreatedAt: time.Now(),
				}, nil
			}
			return nil, ErrUserNotFound
		},
	}

	handler := NewHandler(mockSvc)

	t.Run("returns authenticated user profile without password_hash", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		claims := &middleware.UserClaims{
			UserID:    userID,
			Email:     "alice@example.com",
			Role:      models.RoleMember,
			TokenType: middleware.TokenTypeAccess,
		}
		ctx := context.WithValue(req.Context(), middleware.UserContextKey, claims)
		rec := httptest.NewRecorder()

		handler.GetMe(rec, req.WithContext(ctx))

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Data  map[string]any `json:"data"`
			Error *string        `json:"error"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("expected error to be nil, got: %v", *resp.Error)
		}

		if resp.Data["email"] != "alice@example.com" || resp.Data["name"] != "Alice" {
			t.Fatalf("unexpected data: %+v", resp.Data)
		}

		if _, exists := resp.Data["password_hash"]; exists {
			t.Fatal("password_hash must never be present in the response")
		}
		if _, exists := resp.Data["password"]; exists {
			t.Fatal("password must never be present in the response")
		}
	})

	t.Run("returns 401 when context has no user claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		rec := httptest.NewRecorder()

		handler.GetMe(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("returns 404 when user is not found in database", func(t *testing.T) {
		unknownID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		claims := &middleware.UserClaims{
			UserID:    unknownID,
			Email:     "unknown@example.com",
			Role:      models.RoleMember,
			TokenType: middleware.TokenTypeAccess,
		}
		ctx := context.WithValue(req.Context(), middleware.UserContextKey, claims)
		rec := httptest.NewRecorder()

		handler.GetMe(rec, req.WithContext(ctx))

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got %d", rec.Code)
		}
	})
}
