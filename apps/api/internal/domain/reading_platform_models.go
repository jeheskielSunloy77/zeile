package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	VerificationStatusPending              = "pending"
	VerificationStatusVerifiedPublicDomain = "verified_public_domain"
	VerificationStatusRejected             = "rejected"
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
	ReadingModeEPUB      = "epub"
	ReadingModePDFText   = "pdf_text"
	ReadingModePDFLayout = "pdf_layout"
)

const (
	VisibilityPrivate       = "private"
	VisibilityAuthenticated = "authenticated"
)

const (
	ShareResourceTypeList      = "list"
	ShareResourceTypeHighlight = "highlight"
	ShareResourceTypeBookFile  = "book_file"
)

const (
	RawFileSharingPrivate    = "private"
	RawFileSharingPublicLink = "public_link"
)

const (
	ModerationStatusPending  = "pending"
	ModerationStatusApproved = "approved"
	ModerationStatusRejected = "rejected"
)

type BookCatalog struct {
	ID                 uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey"`
	Title              string          `json:"title" gorm:"not null"`
	Authors            string          `json:"authors" gorm:"not null"`
	Identifiers        json.RawMessage `json:"identifiers" gorm:"type:jsonb;not null"`
	Language           *string         `json:"language,omitempty"`
	VerificationStatus string          `json:"verificationStatus" gorm:"not null"`
	SourceType         string          `json:"sourceType" gorm:"not null"`
	CreatedAt          time.Time       `json:"createdAt"`
	UpdatedAt          time.Time       `json:"updatedAt"`
	DeletedAt          gorm.DeletedAt  `json:"deletedAt"`
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
	State               string         `json:"state" gorm:"not null"`
	VisibilityInProfile bool           `json:"visibilityInProfile" gorm:"not null"`
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
	ListID            *uuid.UUID      `json:"listId,omitempty" gorm:"type:uuid"`
	IsDeleted         bool            `json:"isDeleted"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
	DeletedAt         gorm.DeletedAt  `json:"deletedAt"`
}

func (m Highlight) GetID() uuid.UUID {
	return m.ID
}

type ShareList struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	UserID      uuid.UUID      `json:"userId" gorm:"type:uuid;not null;index"`
	Name        string         `json:"name" gorm:"not null"`
	Description *string        `json:"description,omitempty"`
	Visibility  string         `json:"visibility" gorm:"not null"`
	IsPublished bool           `json:"isPublished"`
	PublishedAt *time.Time     `json:"publishedAt,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `json:"deletedAt"`
}

func (m ShareList) GetID() uuid.UUID {
	return m.ID
}

type ShareListItem struct {
	ID                uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	ListID            uuid.UUID  `json:"listId" gorm:"type:uuid;not null;index"`
	ItemType          string     `json:"itemType" gorm:"not null"`
	UserLibraryBookID *uuid.UUID `json:"userLibraryBookId,omitempty" gorm:"type:uuid"`
	HighlightID       *uuid.UUID `json:"highlightId,omitempty" gorm:"type:uuid"`
	Position          int        `json:"position"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

func (m ShareListItem) GetID() uuid.UUID {
	return m.ID
}

type BookSharePolicy struct {
	ID                   uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	UserID               uuid.UUID `json:"userId" gorm:"type:uuid;not null;index"`
	UserLibraryBookID    uuid.UUID `json:"userLibraryBookId" gorm:"type:uuid;not null;index"`
	RawFileSharing       string    `json:"rawFileSharing" gorm:"not null"`
	AllowMetadataSharing bool      `json:"allowMetadataSharing"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

func (m BookSharePolicy) GetID() uuid.UUID {
	return m.ID
}

type ShareLink struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	UserID       uuid.UUID  `json:"userId" gorm:"type:uuid;not null;index"`
	ResourceType string     `json:"resourceType" gorm:"not null"`
	ResourceID   uuid.UUID  `json:"resourceId" gorm:"type:uuid;not null"`
	Token        string     `json:"token" gorm:"not null"`
	RequiresAuth bool       `json:"requiresAuth"`
	IsActive     bool       `json:"isActive"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

func (m ShareLink) GetID() uuid.UUID {
	return m.ID
}

type CommunityProfile struct {
	UserID              uuid.UUID `json:"userId" gorm:"type:uuid;primaryKey"`
	DisplayName         *string   `json:"displayName,omitempty"`
	Bio                 *string   `json:"bio,omitempty"`
	AvatarURL           *string   `json:"avatarUrl,omitempty"`
	ShowReadingActivity bool      `json:"showReadingActivity"`
	ShowHighlights      bool      `json:"showHighlights"`
	ShowLists           bool      `json:"showLists"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

func (m CommunityProfile) GetID() uuid.UUID {
	return m.UserID
}

type ActivityEvent struct {
	ID           uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey"`
	UserID       uuid.UUID       `json:"userId" gorm:"type:uuid;not null;index"`
	EventType    string          `json:"eventType" gorm:"not null"`
	ResourceType string          `json:"resourceType" gorm:"not null"`
	ResourceID   *uuid.UUID      `json:"resourceId,omitempty" gorm:"type:uuid"`
	PayloadJSON  json.RawMessage `json:"payloadJson" gorm:"type:jsonb;not null"`
	Visibility   string          `json:"visibility" gorm:"not null"`
	CreatedAt    time.Time       `json:"createdAt"`
}

func (m ActivityEvent) GetID() uuid.UUID {
	return m.ID
}

type ModerationReview struct {
	ID                uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey"`
	CatalogBookID     uuid.UUID       `json:"catalogBookId" gorm:"type:uuid;not null;index"`
	SubmittedByUserID uuid.UUID       `json:"submittedByUserId" gorm:"type:uuid;not null"`
	Status            string          `json:"status" gorm:"not null"`
	Decision          *string         `json:"decision,omitempty"`
	EvidenceJSON      json.RawMessage `json:"evidenceJson" gorm:"type:jsonb;not null"`
	ReviewerUserID    *uuid.UUID      `json:"reviewerUserId,omitempty" gorm:"type:uuid"`
	ReviewedAt        *time.Time      `json:"reviewedAt,omitempty"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
}

func (m ModerationReview) GetID() uuid.UUID {
	return m.ID
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
