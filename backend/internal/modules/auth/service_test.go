package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Sidd231/4yaar/backend/internal/config"
	"github.com/Sidd231/4yaar/backend/internal/middleware"
	"github.com/Sidd231/4yaar/backend/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type mockAuthRepository struct {
	users             map[uuid.UUID]*models.User
	blacklistedTokens map[string]bool
	blacklistErr      error
}

func newMockAuthRepo() *mockAuthRepository {
	return &mockAuthRepository{
		users:             make(map[uuid.UUID]*models.User),
		blacklistedTokens: make(map[string]bool),
	}
}

func (m *mockAuthRepository) CreateUserWithInvite(ctx context.Context, user *models.User, inviteID uuid.UUID) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockAuthRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func (m *mockAuthRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockAuthRepository) GetInviteByToken(ctx context.Context, token uuid.UUID) (*models.Invite, error) {
	return nil, nil
}

func (m *mockAuthRepository) BlacklistToken(ctx context.Context, token string, expiration time.Duration) error {
	if m.blacklistErr != nil {
		return m.blacklistErr
	}
	m.blacklistedTokens[token] = true
	return nil
}

func (m *mockAuthRepository) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	return m.blacklistedTokens[token], nil
}

func testConfig() *config.Config {
	return &config.Config{
		JWTSecret:        "test-super-secret-key-at-least-32-bytes!!",
		JWTAccessExpiry:  15 * time.Minute,
		JWTRefreshExpiry: 7 * 24 * time.Hour,
	}
}

func generateToken(cfg *config.Config, userID uuid.UUID, email, role, tokenType string, expiry time.Duration) string {
	claims := &middleware.UserClaims{
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
	tokenString, _ := token.SignedString([]byte(cfg.JWTSecret))
	return tokenString
}

func TestAuthService_Refresh_Success(t *testing.T) {
	repo := newMockAuthRepo()
	cfg := testConfig()
	svc := NewService(repo, nil, cfg)

	userID := uuid.New()
	user := &models.User{
		ID:    userID,
		Email: "alice@example.com",
		Role:  models.RoleMember,
	}
	repo.users[userID] = user

	refreshToken := generateToken(cfg, userID, user.Email, user.Role, middleware.TokenTypeRefresh, time.Hour)

	resp, err := svc.Refresh(context.Background(), refreshToken)
	if err != nil {
		t.Fatalf("expected successful refresh, got: %v", err)
	}

	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("expected non-empty access and refresh tokens")
	}

	// Verify old refresh token was blacklisted
	isBlacklisted, _ := repo.IsTokenBlacklisted(context.Background(), refreshToken)
	if !isBlacklisted {
		t.Fatal("expected old refresh token to be blacklisted after rotation")
	}
}

func TestAuthService_Refresh_RejectsAccessToken(t *testing.T) {
	repo := newMockAuthRepo()
	cfg := testConfig()
	svc := NewService(repo, nil, cfg)

	userID := uuid.New()
	user := &models.User{
		ID:    userID,
		Email: "bob@example.com",
		Role:  models.RoleMember,
	}
	repo.users[userID] = user

	// Negative test: Access token must NOT be accepted for refresh
	accessToken := generateToken(cfg, userID, user.Email, user.Role, middleware.TokenTypeAccess, time.Hour)

	_, err := svc.Refresh(context.Background(), accessToken)
	if err == nil {
		t.Fatal("expected ErrInvalidToken when trying to refresh using an access token, got nil")
	}
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got: %v", err)
	}
}

func TestAuthService_Refresh_RejectsBlacklistedToken(t *testing.T) {
	repo := newMockAuthRepo()
	cfg := testConfig()
	svc := NewService(repo, nil, cfg)

	userID := uuid.New()
	user := &models.User{
		ID:    userID,
		Email: "carol@example.com",
		Role:  models.RoleMember,
	}
	repo.users[userID] = user

	refreshToken := generateToken(cfg, userID, user.Email, user.Role, middleware.TokenTypeRefresh, time.Hour)
	repo.blacklistedTokens[refreshToken] = true

	_, err := svc.Refresh(context.Background(), refreshToken)
	if err == nil {
		t.Fatal("expected ErrTokenRevoked for already blacklisted token, got nil")
	}
	if !errors.Is(err, ErrTokenRevoked) {
		t.Fatalf("expected ErrTokenRevoked, got: %v", err)
	}
}

func TestAuthService_Refresh_FailsWhenBlacklistFails(t *testing.T) {
	repo := newMockAuthRepo()
	cfg := testConfig()
	svc := NewService(repo, nil, cfg)

	userID := uuid.New()
	user := &models.User{
		ID:    userID,
		Email: "dave@example.com",
		Role:  models.RoleMember,
	}
	repo.users[userID] = user

	// Simulate Redis failure during blacklisting
	repo.blacklistErr = errors.New("redis connection refused")

	refreshToken := generateToken(cfg, userID, user.Email, user.Role, middleware.TokenTypeRefresh, time.Hour)

	_, err := svc.Refresh(context.Background(), refreshToken)
	if err == nil {
		t.Fatal("expected rotation to fail when old token cannot be blacklisted")
	}
}

func TestAuthService_Logout_Success(t *testing.T) {
	repo := newMockAuthRepo()
	cfg := testConfig()
	svc := NewService(repo, nil, cfg)

	userID := uuid.New()
	refreshToken := generateToken(cfg, userID, "user@example.com", models.RoleMember, middleware.TokenTypeRefresh, time.Hour)

	if err := svc.Logout(context.Background(), refreshToken); err != nil {
		t.Fatalf("unexpected error on logout: %v", err)
	}

	isBlacklisted, _ := repo.IsTokenBlacklisted(context.Background(), refreshToken)
	if !isBlacklisted {
		t.Fatal("expected refresh token to be blacklisted after logout")
	}
}

func TestAuthService_Logout_RejectsAccessToken(t *testing.T) {
	repo := newMockAuthRepo()
	cfg := testConfig()
	svc := NewService(repo, nil, cfg)

	userID := uuid.New()
	// Negative test: Access token must NOT be accepted for logout
	accessToken := generateToken(cfg, userID, "user@example.com", models.RoleMember, middleware.TokenTypeAccess, time.Hour)

	err := svc.Logout(context.Background(), accessToken)
	if err == nil {
		t.Fatal("expected ErrInvalidToken when attempting to logout with an access token")
	}
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got: %v", err)
	}
}
