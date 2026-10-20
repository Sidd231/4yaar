package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Room type constants
const (
	RoomTypeDirect = "direct"
	RoomTypeGroup  = "group"
)

// Room represents a chat or video room.
type Room struct {
	ID        uuid.UUID    `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string       `gorm:"type:varchar(120);not null" json:"name"`
	Type      string       `gorm:"type:varchar(20);not null;default:'group'" json:"type"`
	CreatedBy uuid.UUID    `gorm:"type:uuid;not null;index" json:"created_by"`
	Creator   *User        `gorm:"foreignKey:CreatedBy;references:ID;constraint:OnDelete:CASCADE" json:"creator,omitempty"`
	Members   []RoomMember `gorm:"foreignKey:RoomID;references:ID" json:"members,omitempty"`
	CreatedAt time.Time    `gorm:"not null" json:"created_at"`
}

// BeforeCreate assigns a new UUID to Room if not already provided.
func (r *Room) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// RoomMember is the join table linking rooms and users.
type RoomMember struct {
	RoomID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"room_id"`
	UserID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	User     *User     `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	JoinedAt time.Time `gorm:"not null" json:"joined_at"`
}
