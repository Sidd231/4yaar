package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// File stores metadata of uploaded assets in S3.
type File struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UploaderID uuid.UUID      `gorm:"type:uuid;not null;index" json:"uploader_id"`
	Uploader   *User          `gorm:"foreignKey:UploaderID;references:ID;constraint:OnDelete:CASCADE" json:"uploader,omitempty"`
	RoomID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"room_id"`
	Room       *Room          `gorm:"foreignKey:RoomID;references:ID;constraint:OnDelete:CASCADE" json:"room,omitempty"`
	S3Key      string         `gorm:"type:varchar(512);not null" json:"s3_key"`
	FileName   string         `gorm:"type:varchar(255);not null" json:"file_name"`
	FileSize   int64          `gorm:"not null" json:"file_size"`
	MimeType   string         `gorm:"type:varchar(100);not null" json:"mime_type"`
	CreatedAt  time.Time      `gorm:"not null" json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// BeforeCreate assigns a new UUID to File if not already provided.
func (f *File) BeforeCreate(tx *gorm.DB) (err error) {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}
