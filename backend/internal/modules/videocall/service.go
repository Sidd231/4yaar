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
	ErrVideoRoomNotFound = errors.New("video room not found")
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
	GetRoomInfo(ctx context.Context, roomID uuid.UUID) (*VideoRoomInfo, error)
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

// GetRoomInfo queries database metadata and augments it with real-time connected peer metrics from the hub.
func (s *service) GetRoomInfo(ctx context.Context, roomID uuid.UUID) (*VideoRoomInfo, error) {
	var room models.Room
	if err := s.db.WithContext(ctx).First(&room, "id = ?", roomID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVideoRoomNotFound
		}
		return nil, err
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
