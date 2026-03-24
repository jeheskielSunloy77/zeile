package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	BookAssetIngestStatusPending   = "pending"
	BookAssetIngestStatusCompleted = "completed"
	BookAssetIngestStatusFailed    = "failed"
)

const (
	UserLibraryBookStateActive   = "active"
	UserLibraryBookStateArchived = "archived"
)

const (
	ReadingModeEPUB = "epub"
)

const (
	VisibilityPrivate       = "private"
	VisibilityAuthenticated = "authenticated"
)

type BookCatalog struct {
	ID          uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey"`
	Title       string          `json:"title" gorm:"not null"`
	Authors     string          `json:"authors" gorm:"not null"`
	Identifiers json.RawMessage `json:"identifiers" gorm:"type:jsonb;not null"`
	Language    *string         `json:"language,omitempty"`
	SourceType  string          `json:"sourceType" gorm:"not null"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt  `json:"deletedAt"`
}

func (m BookCatalog) GetID() uuid.UUID {
	return m.ID
}

func (BookCatalog) TableName() string {
	return "books_catalog"
}

type BookAsset struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	CatalogBookID  uuid.UUID      `json:"catalogBookId" gorm:"type:uuid;not null;index"`
	UploaderUserID uuid.UUID      `json:"uploaderUserId" gorm:"type:uuid;not null;index"`
	SourceAssetID  *uuid.UUID     `json:"sourceAssetId,omitempty" gorm:"type:uuid"`
	StoragePath    string         `json:"storagePath" gorm:"not null"`
	PublicURL      *string        `json:"publicUrl,omitempty"`
	MimeType       string         `json:"mimeType" gorm:"not null"`
	SizeBytes      int64          `json:"sizeBytes" gorm:"not null"`
	Checksum       string         `json:"checksum" gorm:"not null"`
	IngestStatus   string         `json:"ingestStatus" gorm:"not null"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `json:"deletedAt"`
}

func (m BookAsset) GetID() uuid.UUID {
	return m.ID
}

type UserLibraryBook struct {
	ID                  uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	UserID              uuid.UUID      `json:"userId" gorm:"type:uuid;not null;index"`
	CatalogBookID       uuid.UUID      `json:"catalogBookId" gorm:"type:uuid;not null;index"`
	PreferredAssetID    *uuid.UUID     `json:"preferredAssetId,omitempty" gorm:"type:uuid"`
	SourceLibraryBookID *uuid.UUID     `json:"sourceLibraryBookId,omitempty" gorm:"type:uuid"`
	State               string         `json:"state" gorm:"not null"`
	IsPublic            bool           `json:"isPublic" gorm:"not null"`
	AddedAt             time.Time      `json:"addedAt"`
	ArchivedAt          *time.Time     `json:"archivedAt,omitempty"`
	CreatedAt           time.Time      `json:"createdAt"`
	UpdatedAt           time.Time      `json:"updatedAt"`
	DeletedAt           gorm.DeletedAt `json:"deletedAt"`
}

func (m UserLibraryBook) GetID() uuid.UUID {
	return m.ID
}

type ReadingState struct {
	ID                uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey"`
	UserID            uuid.UUID       `json:"userId" gorm:"type:uuid;not null;index"`
	UserLibraryBookID uuid.UUID       `json:"userLibraryBookId" gorm:"type:uuid;not null;index"`
	Mode              string          `json:"mode" gorm:"not null"`
	LocatorJSON       json.RawMessage `json:"locatorJson" gorm:"type:jsonb;not null"`
	ProgressPercent   float64         `json:"progressPercent"`
	Version           int64           `json:"version"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
	DeletedAt         gorm.DeletedAt  `json:"deletedAt"`
}

func (m ReadingState) GetID() uuid.UUID {
	return m.ID
}

type Highlight struct {
	ID                uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey"`
	UserID            uuid.UUID       `json:"userId" gorm:"type:uuid;not null;index"`
	UserLibraryBookID uuid.UUID       `json:"userLibraryBookId" gorm:"type:uuid;not null;index"`
	Mode              string          `json:"mode" gorm:"not null"`
	LocatorJSON       json.RawMessage `json:"locatorJson" gorm:"type:jsonb;not null"`
	Excerpt           *string         `json:"excerpt,omitempty"`
	Visibility        string          `json:"visibility" gorm:"not null"`
	IsDeleted         bool            `json:"isDeleted"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
	DeletedAt         gorm.DeletedAt  `json:"deletedAt"`
}

func (m Highlight) GetID() uuid.UUID {
	return m.ID
}

type CommunityBookOwner struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	AvatarURL *string   `json:"avatarUrl,omitempty"`
}

type CommunityBookAsset struct {
	ID        uuid.UUID `json:"id"`
	MimeType  string    `json:"mimeType"`
	SizeBytes int64     `json:"sizeBytes"`
	Checksum  string    `json:"checksum"`
	PublicURL *string   `json:"publicUrl,omitempty"`
}

type CommunityBook struct {
	ID               uuid.UUID          `json:"id"`
	CatalogBookID    uuid.UUID          `json:"catalogBookId"`
	PreferredAssetID uuid.UUID          `json:"preferredAssetId"`
	Owner            CommunityBookOwner `json:"owner"`
	Title            string             `json:"title"`
	Authors          string             `json:"authors"`
	Identifiers      json.RawMessage    `json:"identifiers"`
	Language         *string            `json:"language,omitempty"`
	SourceType       string             `json:"sourceType"`
	AddedAt          time.Time          `json:"addedAt"`
	CreatedAt        time.Time          `json:"createdAt"`
	UpdatedAt        time.Time          `json:"updatedAt"`
	PreferredAsset   CommunityBookAsset `json:"preferredAsset"`
}

type IdempotencyKey struct {
	ID           uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey"`
	UserID       uuid.UUID       `json:"userId" gorm:"type:uuid;not null"`
	Operation    string          `json:"operation" gorm:"not null"`
	Key          string          `json:"key" gorm:"not null"`
	RequestHash  *string         `json:"requestHash,omitempty"`
	ResponseJSON json.RawMessage `json:"responseJson" gorm:"type:jsonb;not null"`
	CreatedAt    time.Time       `json:"createdAt"`
	ExpiresAt    *time.Time      `json:"expiresAt,omitempty"`
}

func (m IdempotencyKey) GetID() uuid.UUID {
	return m.ID
}
