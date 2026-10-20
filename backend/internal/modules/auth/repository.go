package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sidd231/4yaar/backend/internal/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Repository handles database and cache operations for user authentication.
type Repository interface {
	CreateUserWithInvite(ctx context.Context, user *models.User, inviteID uuid.UUID) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetInviteByToken(ctx context.Context, token uuid.UUID) (*models.Invite, error)
	BlacklistToken(ctx context.Context, token string, expiration time.Duration) error
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
}

type repository struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewRepository creates a new Auth repository instance.
func NewRepository(db *gorm.DB, redisClient *redis.Client) Repository {
	return &repository{
		db:    db,
		redis: redisClient,
	}
}

// CreateUserWithInvite creates the user and marks the invite as used within a single transaction.
func (r *repository) CreateUserWithInvite(ctx context.Context, user *models.User, inviteID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		// Mark invite as used
		if err := tx.Model(&models.Invite{}).
			Where("id = ? AND used_at IS NULL", inviteID).
			Update("used_at", &now).Error; err != nil {
			return fmt.Errorf("failed to mark invite as used: %w", err)
		}

		// Create user
		if err := tx.Create(user).Error; err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		return nil
	})
}

// GetUserByID retrieves a user record by primary key.
func (r *repository) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail finds a user record by email address.
func (r *repository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetInviteByToken retrieves an unused, valid invite matching the token.
func (r *repository) GetInviteByToken(ctx context.Context, token uuid.UUID) (*models.Invite, error) {
	var invite models.Invite
	if err := r.db.WithContext(ctx).First(&invite, "token = ?", token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &invite, nil
}

// BlacklistToken stores the given refresh token in Redis with a TTL.
func (r *repository) BlacklistToken(ctx context.Context, token string, expiration time.Duration) error {
	key := fmt.Sprintf("blacklist:%s", token)
	return r.redis.Set(ctx, key, "1", expiration).Err()
}

// IsTokenBlacklisted checks whether a token has been logged out or revoked.
func (r *repository) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	key := fmt.Sprintf("blacklist:%s", token)
	exists, err := r.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}
