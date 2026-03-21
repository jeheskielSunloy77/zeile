package dto

import (
	"encoding/json"

	"github.com/google/uuid"
)

type CreateCatalogBookInput struct {
	Title       string
	Authors     string
	Identifiers map[string]string
	Language    *string
	SourceType  *string
}

type CreateLibraryBookInput struct {
	CatalogBookID    uuid.UUID
	PreferredAssetID *uuid.UUID
	IsPublic         *bool
}

type UpdateLibraryBookInput struct {
	State            *string
	PreferredAssetID *uuid.UUID
	IsPublic         *bool
}

type UploadBookAssetInput struct {
	CatalogBookID uuid.UUID
	FileName      string
	MimeType      string
	SizeBytes     int64
	Checksum      string
}

type UpsertReadingStateInput struct {
	Mode           string
	LocatorJSON    json.RawMessage
	ProgressPercent float64
	IfMatchVersion *int64
}

type CreateHighlightInput struct {
	Mode        string
	LocatorJSON json.RawMessage
	Excerpt     *string
	Visibility  *string
}

type UpdateHighlightInput struct {
	LocatorJSON *json.RawMessage
	Excerpt     *string
	Visibility  *string
}
