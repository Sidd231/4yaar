package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role constants
const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// User represents an authenticated account on 4Yaar.
type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name         string         `gorm:"type:varchar(100);not null" json:"name"`
	Email        string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"type:varchar(255);not null" json:"-"`
	Role         string         `gorm:"type:varchar(20);not null;default:'member'" json:"role"`
	CreatedAt    time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate assigns a new UUID to the User if not already provided.
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
