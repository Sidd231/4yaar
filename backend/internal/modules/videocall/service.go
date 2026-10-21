package videocall

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sidd231/4yaar/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrVideoRoomNotFound     = errors.New("video room not found")
	ErrVideoRoomUnauthorized = errors.New("unauthorized: not a member of this video room")
)

// VideoRoomInfo encapsulates room metadata along with real-time active peer count.
type VideoRoomInfo struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	CreatedBy       uuid.UUID `json:"created_by"`
	ActivePeerCount int       `json:"active_peer_count"`
	CreatedAt       time.Time `json:"created_at"`
}

// Service defines video call room operations.
type Service interface {
	CreateRoom(ctx context.Context, name string, creatorID uuid.UUID) (*models.Room, error)
	GetRoomInfo(ctx context.Context, roomID, userID uuid.UUID) (*VideoRoomInfo, error)
	IsUserInRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
}

type service struct {
	db  *gorm.DB
	hub *Hub
}

// NewService instantiates a new video call service.
func NewService(db *gorm.DB, hub *Hub) Service {
	return &service{
		db:  db,
		hub: hub,
	}
}

// CreateRoom persists a new video conference room in the database.
func (s *service) CreateRoom(ctx context.Context, name string, creatorID uuid.UUID) (*models.Room, error) {
	room := &models.Room{
		ID:        uuid.New(),
		Name:      name,
		Type:      models.RoomTypeGroup,
		CreatedBy: creatorID,
		CreatedAt: time.Now(),
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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

	if err != nil {
		return nil, fmt.Errorf("failed to create video room: %w", err)
	}

	return room, nil
}

// IsUserInRoom verifies whether a user is an active member of the given room.
func (s *service) IsUserInRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&models.RoomMember{}).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Count(&count).Error
	return count > 0, err
}

// GetRoomInfo queries database metadata and augments it with real-time connected peer metrics from the hub.
// Access requires that the requesting user is a member of the room.
func (s *service) GetRoomInfo(ctx context.Context, roomID, userID uuid.UUID) (*VideoRoomInfo, error) {
	var room models.Room
	if err := s.db.WithContext(ctx).First(&room, "id = ?", roomID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVideoRoomNotFound
		}
		return nil, err
	}

	isMember, err := s.IsUserInRoom(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify room membership: %w", err)
	}
	if !isMember {
		return nil, ErrVideoRoomUnauthorized
	}

	peerCount := s.hub.GetPeerCount(roomID)

	return &VideoRoomInfo{
		ID:              room.ID,
		Name:            room.Name,
		CreatedBy:       room.CreatedBy,
		ActivePeerCount: peerCount,
		CreatedAt:       room.CreatedAt,
	}, nil
}
