package users

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Sidd231/4yaar/backend/internal/middleware"
	"github.com/Sidd231/4yaar/backend/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "test-super-secret-key-at-least-32-bytes!!"

func createToken(userID uuid.UUID, role string) string {
	claims := &middleware.UserClaims{
		UserID:    userID,
		Email:     "user@example.com",
		Role:      role,
		TokenType: middleware.TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(testSecret))
	return tokenStr
}

func TestUsersRoutes_AccessControl(t *testing.T) {
	r := chi.NewRouter()

	called := false
	dummyHandler := &Handler{}
	// Override or create router with custom handler function to verify middleware chain
	r.Route("/users", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(testSecret))
		r.Use(middleware.RequireRole(models.RoleAdmin))
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})
	})

	_ = dummyHandler

	t.Run("unauthenticated request returns 401 Unauthorized", func(t *testing.T) {
		called = false
		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
		}
		if called {
			t.Fatal("handler should not have been called")
		}
	})

	t.Run("member role returns 403 Forbidden", func(t *testing.T) {
		called = false
		token := createToken(uuid.New(), models.RoleMember)
		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for member role, got %d", rec.Code)
		}
		if called {
			t.Fatal("handler should not have been called for member role")
		}
	})

	t.Run("admin role returns 200 OK and reaches handler", func(t *testing.T) {
		called = false
		token := createToken(uuid.New(), models.RoleAdmin)
		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for admin role, got %d", rec.Code)
		}
		if !called {
			t.Fatal("handler should have been called for admin role")
		}
	})
}
