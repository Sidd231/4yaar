package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Message type constants
const (
	MessageTypeText  = "text"
	MessageTypeImage = "image"
	MessageTypeFile  = "file"
)

// Message represents a chat message stored in the database.
type Message struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	RoomID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"room_id"`
	Room      *Room      `gorm:"foreignKey:RoomID;references:ID;constraint:OnDelete:CASCADE" json:"room,omitempty"`
	SenderID  uuid.UUID  `gorm:"type:uuid;not null;index" json:"sender_id"`
	Sender    *User      `gorm:"foreignKey:SenderID;references:ID;constraint:OnDelete:CASCADE" json:"sender,omitempty"`
	Content   string     `gorm:"type:text;not null" json:"content"`
	Type      string     `gorm:"type:varchar(20);not null;default:'text'" json:"type"`
	FileID    *uuid.UUID `gorm:"type:uuid;index" json:"file_id"`
	File      *File      `gorm:"foreignKey:FileID;references:ID;constraint:OnDelete:SET NULL" json:"file,omitempty"`
	CreatedAt time.Time  `gorm:"not null;index" json:"created_at"`
}

// BeforeCreate assigns a new UUID to Message if not already provided.
func (m *Message) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
