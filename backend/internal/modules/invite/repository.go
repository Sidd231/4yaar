package invite

import (
	"context"
	"errors"

	"github.com/Sidd231/4yaar/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository defines DB operations for invite records.
type Repository interface {
	CreateInvite(ctx context.Context, invite *models.Invite) error
	GetInviteByToken(ctx context.Context, token uuid.UUID) (*models.Invite, error)
	GetActiveInviteByEmail(ctx context.Context, email string) (*models.Invite, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository initializes an invite repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// CreateInvite persists a new invite entity.
func (r *repository) CreateInvite(ctx context.Context, invite *models.Invite) error {
	return r.db.WithContext(ctx).Create(invite).Error
}

// GetInviteByToken retrieves an invite and preloads the inviter user.
func (r *repository) GetInviteByToken(ctx context.Context, token uuid.UUID) (*models.Invite, error) {
	var inv models.Invite
	if err := r.db.WithContext(ctx).Preload("Inviter").First(&inv, "token = ?", token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &inv, nil
}

// GetActiveInviteByEmail finds an unused, unexpired invite for the email if one exists.
func (r *repository) GetActiveInviteByEmail(ctx context.Context, email string) (*models.Invite, error) {
	var inv models.Invite
	if err := r.db.WithContext(ctx).
		Where("email = ? AND used_at IS NULL AND expires_at > NOW()", email).
		First(&inv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &inv, nil
}
