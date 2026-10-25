package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Sidd231/4yaar/backend/internal/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const (
	UserContextKey contextKey = "user_claims"

	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// UserClaims defines the JWT claims payload for authenticated users.
type UserClaims struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	TokenType string    `json:"token_type"`
	jwt.RegisteredClaims
}

// AuthMiddleware creates an HTTP middleware verifying the Bearer token in the Authorization header.
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// Allow unauthenticated access for login and refresh endpoints
				if strings.HasPrefix(r.URL.Path, "/auth/login") || strings.HasPrefix(r.URL.Path, "/auth/refresh") {
					next.ServeHTTP(w, r)
					return
				}
				slog.Warn("AuthMiddleware: Missing Authorization header", "path", r.URL.Path)
				utils.JSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				slog.Warn("AuthMiddleware: Invalid header format", "header", authHeader)
				utils.JSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			tokenString := parts[1]
			claims, err := ParseAndValidateToken(tokenString, jwtSecret)
			if err != nil {
				slog.Warn("AuthMiddleware: Token validation failed", "error", err, "path", r.URL.Path)
				if errors.Is(err, jwt.ErrTokenExpired) {
					utils.JSONError(w, http.StatusUnauthorized, "token_expired")
					return
				}
				utils.JSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			// Protected API endpoints must only accept access tokens
			if claims.TokenType != TokenTypeAccess {
				slog.Warn("AuthMiddleware: Invalid token type", "type", claims.TokenType)
				utils.JSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			// Store claims in request context
			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole ensures that the authenticated user possesses the required role (e.g., admin).
func RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetUserFromContext(r.Context())
			if !ok {
				utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			if claims.Role != requiredRole {
				utils.JSONError(w, http.StatusForbidden, "Forbidden: insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ParseAndValidateToken parses a HS256 JWT string and validates signature and expiration.
func ParseAndValidateToken(tokenString, secret string) (*UserClaims, error) {
	claims := &UserClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// GetUserFromContext extracts UserClaims from the given context.
func GetUserFromContext(ctx context.Context) (*UserClaims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*UserClaims)
	return claims, ok
}
