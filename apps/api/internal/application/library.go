package application

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/kern/internal/app/errs"
	"github.com/jeheskielSunloy77/kern/internal/app/sqlerr"
	applicationdto "github.com/jeheskielSunloy77/kern/internal/application/dto"
	"github.com/jeheskielSunloy77/kern/internal/application/port"
	"github.com/jeheskielSunloy77/kern/internal/domain"
	"github.com/jeheskielSunloy77/kern/internal/infrastructure/lib/storage"
)

type LibraryService interface {
	CreateCatalogBook(ctx context.Context, input applicationdto.CreateCatalogBookInput) (*domain.BookCatalog, error)
	ListCatalogBooks(ctx context.Context, limit, offset int) ([]domain.BookCatalog, int64, error)
	UploadBookAsset(ctx context.Context, userID uuid.UUID, input applicationdto.UploadBookAssetInput, reader io.Reader) (*domain.BookAsset, error)

	UpsertLibraryBook(ctx context.Context, userID uuid.UUID, input applicationdto.CreateLibraryBookInput) (*domain.UserLibraryBook, error)
	ListLibraryBooks(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.UserLibraryBook, int64, error)
	UpdateLibraryBook(ctx context.Context, userID, libraryBookID uuid.UUID, input applicationdto.UpdateLibraryBookInput) (*domain.UserLibraryBook, error)
	DeleteLibraryBook(ctx context.Context, userID, libraryBookID uuid.UUID) error

	GetReadingState(ctx context.Context, userID, libraryBookID uuid.UUID, mode string) (*domain.ReadingState, error)
	UpsertReadingState(ctx context.Context, userID, libraryBookID uuid.UUID, input applicationdto.UpsertReadingStateInput) (*domain.ReadingState, error)

	CreateHighlight(ctx context.Context, userID, libraryBookID uuid.UUID, input applicationdto.CreateHighlightInput) (*domain.Highlight, error)
	ListHighlights(ctx context.Context, userID, libraryBookID uuid.UUID) ([]domain.Highlight, error)
	UpdateHighlight(ctx context.Context, userID, highlightID uuid.UUID, input applicationdto.UpdateHighlightInput) (*domain.Highlight, error)
	DeleteHighlight(ctx context.Context, userID, highlightID uuid.UUID) error
}

type libraryService struct {
	repo    port.LibraryRepository
	storage storage.Storage
}

func NewLibraryService(repo port.LibraryRepository, storageProvider storage.Storage) LibraryService {
	return &libraryService{
		repo:    repo,
		storage: storageProvider,
	}
}

func (s *libraryService) CreateCatalogBook(ctx context.Context, input applicationdto.CreateCatalogBookInput) (*domain.BookCatalog, error) {
	if strings.TrimSpace(input.Title) == "" {
		return nil, errs.NewBadRequestError("Validation failed", true, []errs.FieldError{{Field: "title", Error: "is required"}}, nil)
	}

	identifiers := []byte("{}")
	if len(input.Identifiers) > 0 {
		encoded, err := jsonMarshal(input.Identifiers)
		if err != nil {
			return nil, errs.NewBadRequestError("identifiers must be a valid object", true, nil, nil)
		}
		identifiers = encoded
	}

	sourceType := "user_upload"
	if input.SourceType != nil && strings.TrimSpace(*input.SourceType) != "" {
		sourceType = strings.TrimSpace(*input.SourceType)
	}

	book := &domain.BookCatalog{
		Title:       strings.TrimSpace(input.Title),
		Authors:     strings.TrimSpace(input.Authors),
		Identifiers: identifiers,
		Language:    input.Language,
		SourceType:  sourceType,
	}
	if err := s.repo.CreateCatalogBook(ctx, book); err != nil {
		return nil, sqlerr.HandleError(err)
	}
	return book, nil
}

func (s *libraryService) ListCatalogBooks(ctx context.Context, limit, offset int) ([]domain.BookCatalog, int64, error) {
	limit, offset = normalizePagination(limit, offset)
	books, total, err := s.repo.ListCatalogBooks(ctx, limit, offset)
	if err != nil {
		return nil, 0, sqlerr.HandleError(err)
	}
	return books, total, nil
}

func (s *libraryService) UploadBookAsset(ctx context.Context, userID uuid.UUID, input applicationdto.UploadBookAssetInput, reader io.Reader) (*domain.BookAsset, error) {
	if s.storage == nil {
		return nil, errs.NewInternalServerError()
	}
	if _, err := s.repo.GetCatalogBookByID(ctx, input.CatalogBookID); err != nil {
		return nil, sqlerr.HandleError(err)
	}
	if strings.TrimSpace(input.MimeType) == "" {
		return nil, errs.NewBadRequestError("Validation failed", true, []errs.FieldError{{Field: "mimeType", Error: "is required"}}, nil)
	}
	if input.SizeBytes <= 0 {
		return nil, errs.NewBadRequestError("Validation failed", true, []errs.FieldError{{Field: "sizeBytes", Error: "must be greater than 0"}}, nil)
	}

	assetID := uuid.New()
	cleanName := strings.TrimSpace(input.FileName)
	if cleanName == "" {
		cleanName = "upload.bin"
	}
	cleanName = path.Base(cleanName)
	storageKey := fmt.Sprintf("books/%s/%s-%s", input.CatalogBookID.String(), assetID.String(), cleanName)

	hasher := sha256.New()
	tee := io.TeeReader(reader, hasher)
	object, err := s.storage.Save(ctx, storageKey, tee, input.SizeBytes, input.MimeType)
	if err != nil {
		return nil, errs.NewInternalServerError()
	}

	checksum := strings.TrimSpace(input.Checksum)
	if checksum == "" {
		checksum = hex.EncodeToString(hasher.Sum(nil))
	}

	asset := &domain.BookAsset{
		ID:             assetID,
		CatalogBookID:  input.CatalogBookID,
		UploaderUserID: userID,
		StoragePath:    object.Path,
		MimeType:       input.MimeType,
		SizeBytes:      input.SizeBytes,
		Checksum:       checksum,
		IngestStatus:   domain.BookAssetIngestStatusCompleted,
	}
	if object.URL != "" {
		asset.PublicURL = &object.URL
	}

	if err := s.repo.CreateBookAsset(ctx, asset); err != nil {
		_ = s.storage.Delete(ctx, object.Path)
		return nil, sqlerr.HandleError(err)
	}
	return asset, nil
}

func (s *libraryService) UpsertLibraryBook(ctx context.Context, userID uuid.UUID, input applicationdto.CreateLibraryBookInput) (*domain.UserLibraryBook, error) {
	if _, err := s.repo.GetCatalogBookByID(ctx, input.CatalogBookID); err != nil {
		return nil, sqlerr.HandleError(err)
	}
	if input.PreferredAssetID != nil {
		if _, err := s.repo.GetBookAssetByID(ctx, *input.PreferredAssetID); err != nil {
			return nil, sqlerr.HandleError(err)
		}
	}

	isPublic := false
	if input.IsPublic != nil {
		isPublic = *input.IsPublic
	}
	if isPublic && input.PreferredAssetID == nil {
		return nil, errs.NewBadRequestError("Validation failed", true, []errs.FieldError{{Field: "isPublic", Error: "preferredAssetId is required for public books"}}, nil)
	}

	book, err := s.repo.UpsertUserLibraryBook(ctx, &domain.UserLibraryBook{
		UserID:           userID,
		CatalogBookID:    input.CatalogBookID,
		PreferredAssetID: input.PreferredAssetID,
		State:            domain.UserLibraryBookStateActive,
		IsPublic:         isPublic,
		AddedAt:          time.Now().UTC(),
	})
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}

	return book, nil
}

func (s *libraryService) ListLibraryBooks(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.UserLibraryBook, int64, error) {
	limit, offset = normalizePagination(limit, offset)
	books, total, err := s.repo.ListUserLibraryBooks(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, sqlerr.HandleError(err)
	}
	return books, total, nil
}

func (s *libraryService) UpdateLibraryBook(ctx context.Context, userID, libraryBookID uuid.UUID, input applicationdto.UpdateLibraryBookInput) (*domain.UserLibraryBook, error) {
	current, err := s.repo.GetUserLibraryBookByID(ctx, userID, libraryBookID)
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}

	updates := make(map[string]any)
	if input.State != nil {
		state := strings.TrimSpace(*input.State)
		if state != domain.UserLibraryBookStateActive && state != domain.UserLibraryBookStateArchived {
			return nil, errs.NewBadRequestError("Validation failed", true, []errs.FieldError{{Field: "state", Error: "must be active or archived"}}, nil)
		}
		updates["state"] = state
		if state == domain.UserLibraryBookStateArchived {
			now := time.Now().UTC()
			updates["archived_at"] = &now
		} else {
			updates["archived_at"] = nil
		}
	}
	if input.PreferredAssetID != nil {
		if _, err := s.repo.GetBookAssetByID(ctx, *input.PreferredAssetID); err != nil {
			return nil, sqlerr.HandleError(err)
		}
		updates["preferred_asset_id"] = input.PreferredAssetID
	}
	if input.IsPublic != nil {
		preferredAssetID := current.PreferredAssetID
		if input.PreferredAssetID != nil {
			preferredAssetID = input.PreferredAssetID
		}
		if *input.IsPublic && preferredAssetID == nil {
			return nil, errs.NewBadRequestError("Validation failed", true, []errs.FieldError{{Field: "isPublic", Error: "preferredAssetId is required for public books"}}, nil)
		}
		updates["is_public"] = *input.IsPublic
	}
	if len(updates) == 0 {
		return current, nil
	}
	book, err := s.repo.UpdateUserLibraryBook(ctx, userID, libraryBookID, updates)
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}
	return book, nil
}

func (s *libraryService) DeleteLibraryBook(ctx context.Context, userID, libraryBookID uuid.UUID) error {
	if err := s.repo.DeleteUserLibraryBook(ctx, userID, libraryBookID); err != nil {
		return sqlerr.HandleError(err)
	}
	return nil
}

func (s *libraryService) GetReadingState(ctx context.Context, userID, libraryBookID uuid.UUID, mode string) (*domain.ReadingState, error) {
	if !isValidReadingMode(mode) {
		return nil, errs.NewBadRequestError("Validation failed", true, []errs.FieldError{{Field: "mode", Error: "unsupported reading mode"}}, nil)
	}
	state, err := s.repo.GetReadingState(ctx, userID, libraryBookID, mode)
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}
	return state, nil
}

func (s *libraryService) UpsertReadingState(ctx context.Context, userID, libraryBookID uuid.UUID, input applicationdto.UpsertReadingStateInput) (*domain.ReadingState, error) {
	if !isValidReadingMode(input.Mode) {
		return nil, errs.NewBadRequestError("Validation failed", true, []errs.FieldError{{Field: "mode", Error: "unsupported reading mode"}}, nil)
	}
	if input.ProgressPercent < 0 || input.ProgressPercent > 100 {
		return nil, errs.NewBadRequestError("Validation failed", true, []errs.FieldError{{Field: "progressPercent", Error: "must be between 0 and 100"}}, nil)
	}
	if _, err := s.repo.GetUserLibraryBookByID(ctx, userID, libraryBookID); err != nil {
		return nil, sqlerr.HandleError(err)
	}

	state, err := s.repo.UpsertReadingState(ctx, &domain.ReadingState{
		UserID:            userID,
		UserLibraryBookID: libraryBookID,
		Mode:              input.Mode,
		LocatorJSON:       emptyJSONIfNil(input.LocatorJSON),
		ProgressPercent:   input.ProgressPercent,
	}, input.IfMatchVersion)
	if err != nil {
		if err == port.ErrVersionConflict {
			return nil, &errs.ErrorResponse{
				Message:  "conflict_version_mismatch",
				Status:   409,
				Success:  false,
				Override: true,
			}
		}
		return nil, sqlerr.HandleError(err)
	}

	return state, nil
}

func (s *libraryService) CreateHighlight(ctx context.Context, userID, libraryBookID uuid.UUID, input applicationdto.CreateHighlightInput) (*domain.Highlight, error) {
	if !isValidReadingMode(input.Mode) {
		return nil, errs.NewBadRequestError("Validation failed", true, []errs.FieldError{{Field: "mode", Error: "unsupported reading mode"}}, nil)
	}
	if _, err := s.repo.GetUserLibraryBookByID(ctx, userID, libraryBookID); err != nil {
		return nil, sqlerr.HandleError(err)
	}

	visibility := domain.VisibilityPrivate
	if input.Visibility != nil {
		visibility = strings.TrimSpace(*input.Visibility)
	}
	if visibility != domain.VisibilityPrivate && visibility != domain.VisibilityAuthenticated {
		return nil, errs.NewBadRequestError("Validation failed", true, []errs.FieldError{{Field: "visibility", Error: "must be private or authenticated"}}, nil)
	}

	highlight := &domain.Highlight{
		UserID:            userID,
		UserLibraryBookID: libraryBookID,
		Mode:              input.Mode,
		LocatorJSON:       emptyJSONIfNil(input.LocatorJSON),
		Excerpt:           input.Excerpt,
		Visibility:        visibility,
		IsDeleted:         false,
	}
	if err := s.repo.CreateHighlight(ctx, highlight); err != nil {
		return nil, sqlerr.HandleError(err)
	}

	return highlight, nil
}

func (s *libraryService) ListHighlights(ctx context.Context, userID, libraryBookID uuid.UUID) ([]domain.Highlight, error) {
	if _, err := s.repo.GetUserLibraryBookByID(ctx, userID, libraryBookID); err != nil {
		return nil, sqlerr.HandleError(err)
	}
	highlights, err := s.repo.ListHighlights(ctx, userID, libraryBookID)
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}
	return highlights, nil
}

func (s *libraryService) UpdateHighlight(ctx context.Context, userID, highlightID uuid.UUID, input applicationdto.UpdateHighlightInput) (*domain.Highlight, error) {
	updates := make(map[string]any)
	if input.LocatorJSON != nil {
		updates["locator_json"] = emptyJSONIfNil(*input.LocatorJSON)
	}
	if input.Excerpt != nil {
		updates["excerpt"] = input.Excerpt
	}
	if input.Visibility != nil {
		visibility := strings.TrimSpace(*input.Visibility)
		if visibility != domain.VisibilityPrivate && visibility != domain.VisibilityAuthenticated {
			return nil, errs.NewBadRequestError("Validation failed", true, []errs.FieldError{{Field: "visibility", Error: "must be private or authenticated"}}, nil)
		}
		updates["visibility"] = visibility
	}
	if len(updates) == 0 {
		return s.repo.GetHighlightByID(ctx, userID, highlightID)
	}
	highlight, err := s.repo.UpdateHighlight(ctx, userID, highlightID, updates)
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}
	return highlight, nil
}

func (s *libraryService) DeleteHighlight(ctx context.Context, userID, highlightID uuid.UUID) error {
	if err := s.repo.DeleteHighlight(ctx, userID, highlightID); err != nil {
		return sqlerr.HandleError(err)
	}
	return nil
}

func normalizePagination(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func isValidReadingMode(mode string) bool {
	switch strings.TrimSpace(mode) {
	case domain.ReadingModeEPUB, domain.ReadingModePDFText, domain.ReadingModePDFLayout:
		return true
	default:
		return false
	}
}

func emptyJSONIfNil(raw []byte) []byte {
	if len(raw) == 0 {
		return []byte("{}")
	}
	return raw
}

func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}
