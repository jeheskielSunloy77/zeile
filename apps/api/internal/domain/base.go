package domain

import "github.com/google/uuid"

type BaseModel interface {
	GetID() uuid.UUID
}
