package files

import (
	"context"
	"errors"
	"time"

	"github.com/Sidd231/4yaar/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository handles database operations for file metadata.
type Repository interface {
	CreateFile(ctx context.Context, file *models.File) error
	GetFileByID(ctx context.Context, id uuid.UUID) (*models.File, error)
	SoftDeleteFile(ctx context.Context, id uuid.UUID) error
	IsUserInRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new file repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// CreateFile stores a new File record.
func (r *repository) CreateFile(ctx context.Context, file *models.File) error {
	return r.db.WithContext(ctx).Create(file).Error
}

// GetFileByID finds an active (non-soft-deleted) file by UUID.
func (r *repository) GetFileByID(ctx context.Context, id uuid.UUID) (*models.File, error) {
	var file models.File
	if err := r.db.WithContext(ctx).First(&file, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &file, nil
}

// SoftDeleteFile marks a file as deleted with current timestamp.
func (r *repository) SoftDeleteFile(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&models.File{}).
		Where("id = ?", id).
		Update("deleted_at", now).Error
}

// IsUserInRoom verifies if the given user is an active member of the room.
func (r *repository) IsUserInRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.RoomMember{}).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Count(&count).Error
	return count > 0, err
}
