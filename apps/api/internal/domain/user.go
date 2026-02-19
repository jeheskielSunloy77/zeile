package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents an application user with local and federated auth support.
type User struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"deletedAt"`

	Email           string     `json:"email" gorm:"uniqueIndex;not null"`
	Username        string     `json:"username" gorm:"not null"`
	PasswordHash    string     `json:"-"`
	GoogleID        *string    `json:"googleId,omitempty" gorm:"uniqueIndex"`
	EmailVerifiedAt *time.Time `json:"emailVerifiedAt,omitempty"`
	LastLoginAt     *time.Time `json:"lastLoginAt,omitempty"`
	IsAdmin         bool       `json:"isAdmin" gorm:"not null;default:false"`
}

func (m User) GetID() uuid.UUID {
	return m.ID
}
