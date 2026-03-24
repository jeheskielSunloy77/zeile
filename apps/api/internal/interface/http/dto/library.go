package dto

import (
	"encoding/json"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	applicationdto "github.com/jeheskielSunloy77/kern/internal/application/dto"
)

type CreateCatalogBookRequest struct {
	Title       string            `json:"title" validate:"required,min=1,max=255"`
	Authors     string            `json:"authors" validate:"max=1024"`
	Identifiers map[string]string `json:"identifiers"`
	Language    *string           `json:"language" validate:"omitempty,max=32"`
	SourceType  *string           `json:"sourceType" validate:"omitempty,max=64"`
}

func (d *CreateCatalogBookRequest) Validate() error {
	return validator.New().Struct(d)
}

func (d *CreateCatalogBookRequest) ToUsecase() applicationdto.CreateCatalogBookInput {
	return applicationdto.CreateCatalogBookInput{
		Title:       d.Title,
		Authors:     d.Authors,
		Identifiers: d.Identifiers,
		Language:    d.Language,
		SourceType:  d.SourceType,
	}
}

type CreateLibraryBookRequest struct {
	CatalogBookID    string  `json:"catalogBookId" validate:"required,uuid"`
	PreferredAssetID *string `json:"preferredAssetId" validate:"omitempty,uuid"`
	IsPublic         *bool   `json:"isPublic"`
}

func (d *CreateLibraryBookRequest) Validate() error {
	return validator.New().Struct(d)
}

func (d *CreateLibraryBookRequest) ToUsecase() applicationdto.CreateLibraryBookInput {
	catalogID, _ := uuid.Parse(d.CatalogBookID)
	var preferred *uuid.UUID
	if d.PreferredAssetID != nil {
		if parsed, err := uuid.Parse(*d.PreferredAssetID); err == nil {
			preferred = &parsed
		}
	}
	return applicationdto.CreateLibraryBookInput{
		CatalogBookID:    catalogID,
		PreferredAssetID: preferred,
		IsPublic:         d.IsPublic,
	}
}

type UpdateLibraryBookRequest struct {
	State            *string `json:"state" validate:"omitempty,oneof=active archived"`
	PreferredAssetID *string `json:"preferredAssetId" validate:"omitempty,uuid"`
	IsPublic         *bool   `json:"isPublic"`
}

func (d *UpdateLibraryBookRequest) Validate() error {
	return validator.New().Struct(d)
}

func (d *UpdateLibraryBookRequest) ToUsecase() applicationdto.UpdateLibraryBookInput {
	var preferred *uuid.UUID
	if d.PreferredAssetID != nil {
		if parsed, err := uuid.Parse(*d.PreferredAssetID); err == nil {
			preferred = &parsed
		}
	}
	return applicationdto.UpdateLibraryBookInput{
		State:            d.State,
		PreferredAssetID: preferred,
		IsPublic:         d.IsPublic,
	}
}

type UpsertReadingStateRequest struct {
	LocatorJSON     map[string]any `json:"locatorJson"`
	ProgressPercent float64        `json:"progressPercent" validate:"min=0,max=100"`
	IfMatchVersion  *int64         `json:"ifMatchVersion" validate:"omitempty,min=1"`
}

func (d *UpsertReadingStateRequest) Validate() error {
	return validator.New().Struct(d)
}

func (d *UpsertReadingStateRequest) ToUsecase(mode string) applicationdto.UpsertReadingStateInput {
	encoded, _ := json.Marshal(d.LocatorJSON)
	return applicationdto.UpsertReadingStateInput{
		Mode:            mode,
		LocatorJSON:     encoded,
		ProgressPercent: d.ProgressPercent,
		IfMatchVersion:  d.IfMatchVersion,
	}
}

type CreateHighlightRequest struct {
	Mode        string         `json:"mode" validate:"required,oneof=epub"`
	LocatorJSON map[string]any `json:"locatorJson"`
	Excerpt     *string        `json:"excerpt" validate:"omitempty,max=2000"`
	Visibility  *string        `json:"visibility" validate:"omitempty,oneof=private authenticated"`
}

func (d *CreateHighlightRequest) Validate() error {
	return validator.New().Struct(d)
}

func (d *CreateHighlightRequest) ToUsecase() applicationdto.CreateHighlightInput {
	encoded, _ := json.Marshal(d.LocatorJSON)
	return applicationdto.CreateHighlightInput{
		Mode:        d.Mode,
		LocatorJSON: encoded,
		Excerpt:     d.Excerpt,
		Visibility:  d.Visibility,
	}
}

type UpdateHighlightRequest struct {
	LocatorJSON *map[string]any `json:"locatorJson"`
	Excerpt     *string         `json:"excerpt" validate:"omitempty,max=2000"`
	Visibility  *string         `json:"visibility" validate:"omitempty,oneof=private authenticated"`
}

func (d *UpdateHighlightRequest) Validate() error {
	return validator.New().Struct(d)
}

func (d *UpdateHighlightRequest) ToUsecase() applicationdto.UpdateHighlightInput {
	var locator *json.RawMessage
	if d.LocatorJSON != nil {
		encoded, _ := json.Marshal(d.LocatorJSON)
		raw := json.RawMessage(encoded)
		locator = &raw
	}
	return applicationdto.UpdateHighlightInput{
		LocatorJSON: locator,
		Excerpt:     d.Excerpt,
		Visibility:  d.Visibility,
	}
}
