package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Sidd231/4yaar/backend/internal/config"
	"github.com/Sidd231/4yaar/backend/internal/middleware"
	"github.com/Sidd231/4yaar/backend/internal/models"
	"github.com/Sidd231/4yaar/backend/internal/modules/email"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidInvite       = errors.New("invalid or expired invitation token")
	ErrInviteAlreadyUsed   = errors.New("invitation token has already been used")
	ErrUserAlreadyExists   = errors.New("user with this email already exists")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidToken        = errors.New("invalid or expired refresh token")
	ErrTokenRevoked        = errors.New("token has been revoked")
	ErrUserNotFound        = errors.New("user not found")
)

// AuthResponse defines the payload returned on successful authentication.
type AuthResponse struct {
	User         *UserDTO `json:"user"`
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int64    `json:"expires_in"` // access token lifetime in seconds
}

// UserDTO provides a safe public projection of the User model.
type UserDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// Service defines auth business logic.
type Service interface {
	Register(ctx context.Context, inviteToken uuid.UUID, name, password string) (*AuthResponse, error)
	Login(ctx context.Context, email, password string) (*AuthResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*AuthResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	GetMe(ctx context.Context, userID uuid.UUID) (*UserDTO, error)
}

type service struct {
	repo         Repository
	emailService email.Service
	cfg          *config.Config
}

// NewService creates a new Auth service.
func NewService(repo Repository, emailService email.Service, cfg *config.Config) Service {
	return &service{
		repo:         repo,
		emailService: emailService,
		cfg:          cfg,
	}
}

// Register signs up a new user using a valid invite token.
func (s *service) Register(ctx context.Context, inviteToken uuid.UUID, name, password string) (*AuthResponse, error) {
	invite, err := s.repo.GetInviteByToken(ctx, inviteToken)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup invite: %w", err)
	}
	if invite == nil || invite.IsExpired() {
		return nil, ErrInvalidInvite
	}
	if invite.IsUsed() {
		return nil, ErrInviteAlreadyUsed
	}

	// Verify email is not already registered
	existingUser, err := s.repo.GetUserByEmail(ctx, invite.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		ID:           uuid.New(),
		Name:         name,
		Email:        invite.Email,
		PasswordHash: string(hashedPassword),
		Role:         models.RoleMember,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.CreateUserWithInvite(ctx, user, invite.ID); err != nil {
		return nil, fmt.Errorf("failed to persist user and update invite: %w", err)
	}

	// Dispatch welcome email asynchronously
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.emailService.SendWelcomeEmail(bgCtx, user.Email, user.Name); err != nil {
			slog.Warn("Failed to send welcome email", "email", user.Email, "error", err)
		}
	}()

	return s.generateAuthResponse(user)
}

// Login authenticates a user with email and password.
func (s *service) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.generateAuthResponse(user)
}

// Refresh validates and rotates the refresh token.
func (s *service) Refresh(ctx context.Context, refreshTokenString string) (*AuthResponse, error) {
	claims, err := middleware.ParseAndValidateToken(refreshTokenString, s.cfg.JWTSecret)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Only refresh tokens may be used to refresh
	if claims.TokenType != middleware.TokenTypeRefresh {
		return nil, ErrInvalidToken
	}

	// Check Redis blacklist
	blacklisted, err := s.repo.IsTokenBlacklisted(ctx, refreshTokenString)
	if err != nil {
		return nil, fmt.Errorf("failed checking blacklist: %w", err)
	}
	if blacklisted {
		return nil, ErrTokenRevoked
	}

	user, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil || user == nil {
		return nil, ErrInvalidToken
	}

	// Blacklist the old refresh token
	remainingTTL := time.Until(claims.ExpiresAt.Time)
	if remainingTTL <= 0 {
		remainingTTL = s.cfg.JWTRefreshExpiry
	}
	if err := s.repo.BlacklistToken(ctx, refreshTokenString, remainingTTL); err != nil {
		return nil, fmt.Errorf("failed to revoke old refresh token during rotation: %w", err)
	}

	// Issue new access and refresh tokens
	return s.generateAuthResponse(user)
}

// Logout blacklists the provided refresh token in Redis.
func (s *service) Logout(ctx context.Context, refreshTokenString string) error {
	claims, err := middleware.ParseAndValidateToken(refreshTokenString, s.cfg.JWTSecret)
	if err != nil {
		return ErrInvalidToken
	}

	// Only refresh tokens can be revoked via logout
	if claims.TokenType != middleware.TokenTypeRefresh {
		return ErrInvalidToken
	}

	ttl := s.cfg.JWTRefreshExpiry
	if claims.ExpiresAt != nil {
		remaining := time.Until(claims.ExpiresAt.Time)
		if remaining > 0 {
			ttl = remaining
		}
	}

	if err := s.repo.BlacklistToken(ctx, refreshTokenString, ttl); err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	return nil
}

// GetMe retrieves the user profile for the specified user ID.
func (s *service) GetMe(ctx context.Context, userID uuid.UUID) (*UserDTO, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	return &UserDTO{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}, nil
}

// generateAuthResponse creates JWT access and refresh tokens for a user.
func (s *service) generateAuthResponse(user *models.User) (*AuthResponse, error) {
	now := time.Now()

	// Access Token
	accessClaims := &middleware.UserClaims{
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		TokenType: middleware.TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.JWTAccessExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   user.ID.String(),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Refresh Token
	refreshClaims := &middleware.UserClaims{
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		TokenType: middleware.TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.JWTRefreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   user.ID.String(),
		},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &AuthResponse{
		User: &UserDTO{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.cfg.JWTAccessExpiry.Seconds()),
	}, nil
}
