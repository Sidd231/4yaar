package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "test-secret-key-that-is-at-least-32-bytes-long!!"

func createSignedToken(userID uuid.UUID, email, role, tokenType string, secret string, expiry time.Duration) string {
	claims := &UserClaims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))
	return tokenStr
}

func TestAuthMiddleware_AccessTokenAccepted(t *testing.T) {
	userID := uuid.New()
	token := createSignedToken(userID, "user@example.com", "member", TokenTypeAccess, testSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	rec := httptest.NewRecorder()

	var handlerInvoked bool
	var capturedClaims *UserClaims

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerInvoked = true
		claims, ok := GetUserFromContext(r.Context())
		if !ok {
			t.Fatal("expected claims to be present in context")
		}
		capturedClaims = claims
		w.WriteHeader(http.StatusOK)
	})

	mw := AuthMiddleware(testSecret)(testHandler)
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !handlerInvoked {
		t.Fatal("expected next handler to be invoked")
	}
	if capturedClaims.UserID != userID || capturedClaims.TokenType != TokenTypeAccess {
		t.Fatalf("unexpected claims: %+v", capturedClaims)
	}
}

func TestAuthMiddleware_RefreshTokenRejected(t *testing.T) {
	userID := uuid.New()
	// Negative test: Refresh tokens must NOT authenticate protected API endpoints
	refreshToken := createSignedToken(userID, "user@example.com", "member", TokenTypeRefresh, testSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", refreshToken))
	rec := httptest.NewRecorder()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not have been called with a refresh token")
	})

	mw := AuthMiddleware(testSecret)(testHandler)
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for refresh token on protected route, got %d", rec.Code)
	}
}

func TestAuthMiddleware_MissingOrMalformedHeader(t *testing.T) {
	testCases := []struct {
		name       string
		authHeader string
	}{
		{"missing header", ""},
		{"malformed without Bearer prefix", "Token some-token"},
		{"malformed only Bearer", "Bearer"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rec := httptest.NewRecorder()

			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("handler should not have been called")
			})

			mw := AuthMiddleware(testSecret)(testHandler)
			mw.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", rec.Code)
			}
		})
	}
}

func TestAuthMiddleware_InvalidOrExpiredToken(t *testing.T) {
	// Expired token
	expiredToken := createSignedToken(uuid.New(), "user@example.com", "member", TokenTypeAccess, testSecret, -time.Hour)
	// Wrong secret
	wrongSecretToken := createSignedToken(uuid.New(), "user@example.com", "member", TokenTypeAccess, "wrong-secret-key-32-bytes-long!!", time.Hour)

	testCases := []struct {
		name  string
		token string
	}{
		{"expired token", expiredToken},
		{"wrong secret token", wrongSecretToken},
		{"garbage token", "not-a-valid-jwt-token"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)
			rec := httptest.NewRecorder()

			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("handler should not have been called")
			})

			mw := AuthMiddleware(testSecret)(testHandler)
			mw.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", rec.Code)
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	adminClaims := &UserClaims{
		UserID: uuid.New(),
		Role:   "admin",
	}
	memberClaims := &UserClaims{
		UserID: uuid.New(),
		Role:   "member",
	}

	t.Run("admin role matches requirement", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		ctx := req.Context()
		ctx = setContextClaims(ctx, adminClaims)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		RequireRole("admin")(testHandler).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("member role rejected for admin route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		ctx := req.Context()
		ctx = setContextClaims(ctx, memberClaims)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not have been called")
		})

		RequireRole("admin")(testHandler).ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d", rec.Code)
		}
	})
}

func setContextClaims(ctx context.Context, claims *UserClaims) context.Context {
	return context.WithValue(ctx, UserContextKey, claims)
}
