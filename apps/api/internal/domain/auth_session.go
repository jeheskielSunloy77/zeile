package domain

import (
	"time"

	"github.com/google/uuid"
)

type AuthSession struct {
	ID               uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	UserID           uuid.UUID  `json:"userId" gorm:"type:uuid;not null;index"`
	RefreshTokenHash string     `json:"-" gorm:"not null;uniqueIndex"`
	UserAgent        *string    `json:"userAgent,omitempty"`
	IPAddress        *string    `json:"ipAddress,omitempty"`
	ExpiresAt        time.Time  `json:"expiresAt"`
	RevokedAt        *time.Time `json:"revokedAt,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func (m AuthSession) GetID() uuid.UUID {
	return m.ID
}
