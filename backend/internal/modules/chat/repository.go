package chat

import (
	"context"
	"errors"
	"time"

	"github.com/Sidd231/4yaar/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository handles database interactions for chat rooms and messages.
type Repository interface {
	CreateRoom(ctx context.Context, room *models.Room, creatorID uuid.UUID) error
	GetRoomByID(ctx context.Context, roomID uuid.UUID) (*models.Room, error)
	GetUserRooms(ctx context.Context, userID uuid.UUID) ([]models.Room, error)
	IsUserInRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
	SaveMessage(ctx context.Context, msg *models.Message) error
	GetRoomMessages(ctx context.Context, roomID uuid.UUID, cursor *time.Time, limit int) ([]models.Message, error)
	AddMember(ctx context.Context, member *models.RoomMember) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository creates a new chat repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// CreateRoom persists a new room and automatically creates the creator's RoomMember entry.
func (r *repository) CreateRoom(ctx context.Context, room *models.Room, creatorID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(room).Error; err != nil {
			return err
		}

		member := models.RoomMember{
			RoomID:   room.ID,
			UserID:   creatorID,
			JoinedAt: time.Now(),
		}
		return tx.Create(&member).Error
	})
}

// GetRoomByID retrieves a room by ID with member information.
func (r *repository) GetRoomByID(ctx context.Context, roomID uuid.UUID) (*models.Room, error) {
	var room models.Room
	if err := r.db.WithContext(ctx).Preload("Members.User").First(&room, "id = ?", roomID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &room, nil
}

// GetUserRooms lists all rooms a user has joined.
func (r *repository) GetUserRooms(ctx context.Context, userID uuid.UUID) ([]models.Room, error) {
	var rooms []models.Room
	err := r.db.WithContext(ctx).
		Joins("JOIN room_members rm ON rm.room_id = rooms.id").
		Where("rm.user_id = ?", userID).
		Preload("Creator").
		Preload("Members.User").
		Find(&rooms).Error
	return rooms, err
}

// IsUserInRoom verifies if the given user is a member of the room.
func (r *repository) IsUserInRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.RoomMember{}).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Count(&count).Error
	return count > 0, err
}

// SaveMessage stores a message in PostgreSQL.
func (r *repository) SaveMessage(ctx context.Context, msg *models.Message) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

// GetRoomMessages fetches paginated message history cursor-based by timestamp (ordered descending then returned).
func (r *repository) GetRoomMessages(ctx context.Context, roomID uuid.UUID, cursor *time.Time, limit int) ([]models.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Preload("Sender").
		Preload("File").
		Order("created_at DESC").
		Limit(limit)

	if cursor != nil && !cursor.IsZero() {
		query = query.Where("created_at < ?", *cursor)
	}

	var messages []models.Message
	if err := query.Find(&messages).Error; err != nil {
		return nil, err
	}

	return messages, nil
}

// AddMember inserts a new room member into the room_members table.
func (r *repository) AddMember(ctx context.Context, member *models.RoomMember) error {
	return r.db.WithContext(ctx).Create(member).Error
}
