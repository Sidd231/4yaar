package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Invite represents an invitation sent to a user to join the platform.
type Invite struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Email     string     `gorm:"type:varchar(255);not null;index" json:"email"`
	Token     uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"token"`
	InvitedBy uuid.UUID  `gorm:"type:uuid;not null;index" json:"invited_by"`
	Inviter   *User      `gorm:"foreignKey:InvitedBy;references:ID;constraint:OnDelete:CASCADE" json:"inviter,omitempty"`
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	UsedAt    *time.Time `json:"used_at"`
	CreatedAt time.Time  `gorm:"not null" json:"created_at"`
}

// BeforeCreate assigns a new UUID to Invite and its Token if not already provided.
func (i *Invite) BeforeCreate(tx *gorm.DB) (err error) {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	if i.Token == uuid.Nil {
		i.Token = uuid.New()
	}
	return nil
}

// IsExpired checks if the invitation token has expired.
func (i *Invite) IsExpired() bool {
	return time.Now().After(i.ExpiresAt)
}

// IsUsed checks if the invite has already been redeemed.
func (i *Invite) IsUsed() bool {
	return i.UsedAt != nil
}
