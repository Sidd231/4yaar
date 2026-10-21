package chat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sidd231/4yaar/backend/internal/models"
	"github.com/google/uuid"
)

var (
	ErrRoomNotFound  = errors.New("room not found")
	ErrAccessDenied  = errors.New("user does not have access to this room")
	ErrAlreadyMember = errors.New("user is already a member of this room")
	ErrNotAuthorized = errors.New("only room creator or admin can add members")
)

// Service defines chat room and messaging business logic.
type Service interface {
	CreateRoom(ctx context.Context, name, roomType string, creatorID uuid.UUID) (*models.Room, error)
	GetUserRooms(ctx context.Context, userID uuid.UUID) ([]models.Room, error)
	GetRoomMessages(ctx context.Context, roomID, userID uuid.UUID, cursor *time.Time, limit int) ([]models.Message, error)
	SaveMessage(ctx context.Context, roomID, senderID uuid.UUID, content, msgType string, fileID *uuid.UUID) (*models.Message, error)
	CanAccessRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
	AddMember(ctx context.Context, roomID, targetUserID, requestingUserID uuid.UUID, requestingUserRole string) (*models.RoomMember, error)
}

type service struct {
	repo Repository
}

// NewService instantiates the chat service.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// CreateRoom creates a new direct or group chat room.
func (s *service) CreateRoom(ctx context.Context, name, roomType string, creatorID uuid.UUID) (*models.Room, error) {
	if roomType != models.RoomTypeDirect && roomType != models.RoomTypeGroup {
		roomType = models.RoomTypeGroup
	}

	room := &models.Room{
		ID:        uuid.New(),
		Name:      name,
		Type:      roomType,
		CreatedBy: creatorID,
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateRoom(ctx, room, creatorID); err != nil {
		return nil, fmt.Errorf("failed to create chat room: %w", err)
	}

	return room, nil
}

// GetUserRooms retrieves all chat rooms associated with the user.
func (s *service) GetUserRooms(ctx context.Context, userID uuid.UUID) ([]models.Room, error) {
	return s.repo.GetUserRooms(ctx, userID)
}

// GetRoomMessages retrieves cursor-based paginated chat message history after verifying access.
func (s *service) GetRoomMessages(ctx context.Context, roomID, userID uuid.UUID, cursor *time.Time, limit int) ([]models.Message, error) {
	allowed, err := s.CanAccessRoom(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrAccessDenied
	}

	return s.repo.GetRoomMessages(ctx, roomID, cursor, limit)
}

// SaveMessage stores a chat message in PostgreSQL.
func (s *service) SaveMessage(ctx context.Context, roomID, senderID uuid.UUID, content, msgType string, fileID *uuid.UUID) (*models.Message, error) {
	if msgType == "" {
		msgType = models.MessageTypeText
	}

	msg := &models.Message{
		ID:        uuid.New(),
		RoomID:    roomID,
		SenderID:  senderID,
		Content:   content,
		Type:      msgType,
		FileID:    fileID,
		CreatedAt: time.Now(),
	}

	if err := s.repo.SaveMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	return msg, nil
}

// CanAccessRoom checks if a user is a registered member of the room.
func (s *service) CanAccessRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	return s.repo.IsUserInRoom(ctx, roomID, userID)
}

// AddMember adds a user to an existing chat room.
// Only the room creator or an admin can add members.
func (s *service) AddMember(ctx context.Context, roomID, targetUserID, requestingUserID uuid.UUID, requestingUserRole string) (*models.RoomMember, error) {
	room, err := s.repo.GetRoomByID(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup room: %w", err)
	}
	if room == nil {
		return nil, ErrRoomNotFound
	}

	// Only creator or admin can add members
	if room.CreatedBy != requestingUserID && requestingUserRole != models.RoleAdmin {
		return nil, ErrNotAuthorized
	}

	// Check if target user is already a member
	isMember, err := s.repo.IsUserInRoom(ctx, roomID, targetUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check room membership: %w", err)
	}
	if isMember {
		return nil, ErrAlreadyMember
	}

	member := &models.RoomMember{
		RoomID:   roomID,
		UserID:   targetUserID,
		JoinedAt: time.Now(),
	}

	if err := s.repo.AddMember(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to add room member: %w", err)
	}

	return member, nil
}

