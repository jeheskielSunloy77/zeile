package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/kern/internal/application/port"
	"github.com/jeheskielSunloy77/kern/internal/domain"
	"gorm.io/gorm"
)

type LibraryRepository = port.LibraryRepository

type libraryRepository struct {
	db *gorm.DB
}

type communityBookRow struct {
	ID               uuid.UUID
	CatalogBookID    uuid.UUID
	PreferredAssetID uuid.UUID
	OwnerID          uuid.UUID
	OwnerUsername    string
	OwnerAvatarURL   *string
	Title            string
	Authors          string
	Identifiers      []byte
	Language         *string
	SourceType       string
	AddedAt          time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	AssetID          uuid.UUID
	AssetMimeType    string
	AssetSizeBytes   int64
	AssetChecksum    string
	AssetPublicURL   *string
}

func NewLibraryRepository(db *gorm.DB) LibraryRepository {
	return &libraryRepository{db: db}
}

func (r *libraryRepository) CreateCatalogBook(ctx context.Context, book *domain.BookCatalog) error {
	if book.ID == uuid.Nil {
		book.ID = uuid.New()
	}
	if len(book.Identifiers) == 0 {
		book.Identifiers = []byte("{}")
	}
	if book.SourceType == "" {
		book.SourceType = "user_upload"
	}
	return r.db.WithContext(ctx).Create(book).Error
}

func (r *libraryRepository) ListCatalogBooks(ctx context.Context, limit, offset int) ([]domain.BookCatalog, int64, error) {
	var (
		books []domain.BookCatalog
		total int64
	)

	query := r.db.WithContext(ctx).Model(&domain.BookCatalog{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&books).Error; err != nil {
		return nil, 0, err
	}

	return books, total, nil
}

func (r *libraryRepository) GetCatalogBookByID(ctx context.Context, id uuid.UUID) (*domain.BookCatalog, error) {
	var book domain.BookCatalog
	if err := r.db.WithContext(ctx).First(&book, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &book, nil
}

func (r *libraryRepository) CreateBookAsset(ctx context.Context, asset *domain.BookAsset) error {
	if asset.ID == uuid.Nil {
		asset.ID = uuid.New()
	}
	if asset.IngestStatus == "" {
		asset.IngestStatus = domain.BookAssetIngestStatusPending
	}
	return r.db.WithContext(ctx).Create(asset).Error
}

func (r *libraryRepository) GetBookAssetByID(ctx context.Context, id uuid.UUID) (*domain.BookAsset, error) {
	var asset domain.BookAsset
	if err := r.db.WithContext(ctx).First(&asset, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &asset, nil
}

func (r *libraryRepository) UpsertUserLibraryBook(ctx context.Context, book *domain.UserLibraryBook) (*domain.UserLibraryBook, error) {
	if book.ID == uuid.Nil {
		book.ID = uuid.New()
	}
	if book.State == "" {
		book.State = domain.UserLibraryBookStateActive
	}

	query := r.db.WithContext(ctx).
		Where("user_id = ? AND catalog_book_id = ?", book.UserID, book.CatalogBookID)
	if book.SourceLibraryBookID == nil {
		query = query.Where("source_library_book_id IS NULL")
	} else {
		query = query.Where("source_library_book_id = ?", *book.SourceLibraryBookID)
	}

	var existing domain.UserLibraryBook
	err := query.First(&existing).Error
	switch {
	case err == nil:
		updates := map[string]any{
			"preferred_asset_id":     book.PreferredAssetID,
			"source_library_book_id": book.SourceLibraryBookID,
			"state":                  book.State,
			"is_public":              book.IsPublic,
			"updated_at":             time.Now().UTC(),
		}
		if book.ArchivedAt != nil {
			updates["archived_at"] = book.ArchivedAt
		}
		if err := r.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
			return nil, err
		}
		return r.GetUserLibraryBookByID(ctx, book.UserID, existing.ID)
	case errors.Is(err, gorm.ErrRecordNotFound):
		if err := r.db.WithContext(ctx).Create(book).Error; err != nil {
			return nil, err
		}
		return book, nil
	default:
		return nil, err
	}
}

func (r *libraryRepository) GetUserLibraryBookByID(ctx context.Context, userID, id uuid.UUID) (*domain.UserLibraryBook, error) {
	var book domain.UserLibraryBook
	if err := r.db.WithContext(ctx).First(&book, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		return nil, err
	}
	return &book, nil
}

func (r *libraryRepository) FindUserLibraryBookBySourceID(ctx context.Context, userID, sourceLibraryBookID uuid.UUID) (*domain.UserLibraryBook, error) {
	var book domain.UserLibraryBook
	if err := r.db.WithContext(ctx).
		First(&book, "user_id = ? AND source_library_book_id = ?", userID, sourceLibraryBookID).Error; err != nil {
		return nil, err
	}
	return &book, nil
}

func (r *libraryRepository) ListUserLibraryBooks(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.UserLibraryBook, int64, error) {
	var (
		books []domain.UserLibraryBook
		total int64
	)
	query := r.db.WithContext(ctx).Model(&domain.UserLibraryBook{}).Where("user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("added_at DESC").Limit(limit).Offset(offset).Find(&books).Error; err != nil {
		return nil, 0, err
	}
	return books, total, nil
}

func (r *libraryRepository) UpdateUserLibraryBook(ctx context.Context, userID, id uuid.UUID, updates map[string]any) (*domain.UserLibraryBook, error) {
	updates["updated_at"] = time.Now().UTC()
	result := r.db.WithContext(ctx).Model(&domain.UserLibraryBook{}).Where("id = ? AND user_id = ?", id, userID).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return r.GetUserLibraryBookByID(ctx, userID, id)
}

func (r *libraryRepository) DeleteUserLibraryBook(ctx context.Context, userID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&domain.UserLibraryBook{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *libraryRepository) ListPublicCommunityBooks(ctx context.Context, query, ownerUsername string, limit, offset int) ([]domain.CommunityBook, int64, error) {
	var (
		rows  []communityBookRow
		total int64
	)

	baseQuery := r.communityBooksBaseQuery(ctx)
	if query = strings.TrimSpace(query); query != "" {
		search := "%" + query + "%"
		baseQuery = baseQuery.Where(
			"bc.title ILIKE ? OR bc.authors ILIKE ? OR u.username ILIKE ?",
			search,
			search,
			search,
		)
	}
	if ownerUsername = strings.TrimSpace(ownerUsername); ownerUsername != "" {
		baseQuery = baseQuery.Where("LOWER(u.username) = LOWER(?)", ownerUsername)
	}

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	result := baseQuery.
		Select(r.communityBookSelect()).
		Order("ulb.added_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows)
	if result.Error != nil {
		return nil, 0, result.Error
	}

	books := make([]domain.CommunityBook, 0, len(rows))
	for _, row := range rows {
		books = append(books, mapCommunityBookRow(row))
	}
	return books, total, nil
}

func (r *libraryRepository) GetPublicCommunityBookByID(ctx context.Context, id uuid.UUID) (*domain.CommunityBook, error) {
	var row communityBookRow
	result := r.communityBooksBaseQuery(ctx).
		Where("ulb.id = ?", id).
		Select(r.communityBookSelect()).
		Limit(1).
		Scan(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	book := mapCommunityBookRow(row)
	return &book, nil
}

func (r *libraryRepository) UpsertReadingState(ctx context.Context, state *domain.ReadingState, expectedVersion *int64) (*domain.ReadingState, error) {
	if len(state.LocatorJSON) == 0 {
		state.LocatorJSON = []byte("{}")
	}

	var existing domain.ReadingState
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND user_library_book_id = ? AND mode = ?", state.UserID, state.UserLibraryBookID, state.Mode).
		First(&existing).Error
	switch {
	case err == nil:
		if expectedVersion != nil && existing.Version != *expectedVersion {
			return nil, port.ErrVersionConflict
		}
		updates := map[string]any{
			"locator_json":     state.LocatorJSON,
			"progress_percent": state.ProgressPercent,
			"version":          existing.Version + 1,
			"updated_at":       time.Now().UTC(),
		}
		if err := r.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
			return nil, err
		}
		return r.GetReadingState(ctx, state.UserID, state.UserLibraryBookID, state.Mode)
	case errors.Is(err, gorm.ErrRecordNotFound):
		if state.ID == uuid.Nil {
			state.ID = uuid.New()
		}
		if state.Version <= 0 {
			state.Version = 1
		}
		if err := r.db.WithContext(ctx).Create(state).Error; err != nil {
			return nil, err
		}
		return state, nil
	default:
		return nil, err
	}
}

func (r *libraryRepository) GetReadingState(ctx context.Context, userID, userLibraryBookID uuid.UUID, mode string) (*domain.ReadingState, error) {
	var state domain.ReadingState
	if err := r.db.WithContext(ctx).
		First(&state, "user_id = ? AND user_library_book_id = ? AND mode = ?", userID, userLibraryBookID, mode).Error; err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *libraryRepository) CreateHighlight(ctx context.Context, highlight *domain.Highlight) error {
	if highlight.ID == uuid.Nil {
		highlight.ID = uuid.New()
	}
	if len(highlight.LocatorJSON) == 0 {
		highlight.LocatorJSON = []byte("{}")
	}
	if highlight.Visibility == "" {
		highlight.Visibility = domain.VisibilityPrivate
	}
	return r.db.WithContext(ctx).Create(highlight).Error
}

func (r *libraryRepository) ListHighlights(ctx context.Context, userID, userLibraryBookID uuid.UUID) ([]domain.Highlight, error) {
	var highlights []domain.Highlight
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND user_library_book_id = ? AND is_deleted = ?", userID, userLibraryBookID, false).
		Order("created_at DESC").
		Find(&highlights).Error
	if err != nil {
		return nil, err
	}
	return highlights, nil
}

func (r *libraryRepository) GetHighlightByID(ctx context.Context, userID, id uuid.UUID) (*domain.Highlight, error) {
	var highlight domain.Highlight
	if err := r.db.WithContext(ctx).First(&highlight, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		return nil, err
	}
	return &highlight, nil
}

func (r *libraryRepository) UpdateHighlight(ctx context.Context, userID, id uuid.UUID, updates map[string]any) (*domain.Highlight, error) {
	updates["updated_at"] = time.Now().UTC()
	result := r.db.WithContext(ctx).Model(&domain.Highlight{}).Where("id = ? AND user_id = ?", id, userID).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return r.GetHighlightByID(ctx, userID, id)
}

func (r *libraryRepository) DeleteHighlight(ctx context.Context, userID, id uuid.UUID) error {
	updates := map[string]any{
		"is_deleted": true,
		"deleted_at": time.Now().UTC(),
		"updated_at": time.Now().UTC(),
	}
	result := r.db.WithContext(ctx).Model(&domain.Highlight{}).Where("id = ? AND user_id = ?", id, userID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *libraryRepository) GetIdempotencyKey(ctx context.Context, userID uuid.UUID, operation, key string) (*domain.IdempotencyKey, error) {
	var idempotency domain.IdempotencyKey
	if err := r.db.WithContext(ctx).
		First(&idempotency, "user_id = ? AND operation = ? AND key = ?", userID, operation, key).Error; err != nil {
		return nil, err
	}
	return &idempotency, nil
}

func (r *libraryRepository) CreateIdempotencyKey(ctx context.Context, idempotency *domain.IdempotencyKey) error {
	if idempotency.ID == uuid.Nil {
		idempotency.ID = uuid.New()
	}
	if len(idempotency.ResponseJSON) == 0 {
		idempotency.ResponseJSON = []byte("{}")
	}
	return r.db.WithContext(ctx).Create(idempotency).Error
}

func (r *libraryRepository) communityBooksBaseQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Table("user_library_books AS ulb").
		Joins("JOIN users AS u ON u.id = ulb.user_id AND u.deleted_at IS NULL").
		Joins("JOIN books_catalog AS bc ON bc.id = ulb.catalog_book_id AND bc.deleted_at IS NULL").
		Joins("JOIN book_assets AS ba ON ba.id = ulb.preferred_asset_id AND ba.deleted_at IS NULL").
		Where("ulb.deleted_at IS NULL").
		Where("ulb.is_public = ?", true).
		Where("ulb.preferred_asset_id IS NOT NULL").
		Where("ulb.state = ?", domain.UserLibraryBookStateActive)
}

func (r *libraryRepository) communityBookSelect() string {
	return strings.Join([]string{
		"ulb.id AS id",
		"ulb.catalog_book_id AS catalog_book_id",
		"ulb.preferred_asset_id AS preferred_asset_id",
		"u.id AS owner_id",
		"u.username AS owner_username",
		"u.avatar_url AS owner_avatar_url",
		"bc.title AS title",
		"bc.authors AS authors",
		"bc.identifiers AS identifiers",
		"bc.language AS language",
		"bc.source_type AS source_type",
		"ulb.added_at AS added_at",
		"ulb.created_at AS created_at",
		"ulb.updated_at AS updated_at",
		"ba.id AS asset_id",
		"ba.mime_type AS asset_mime_type",
		"ba.size_bytes AS asset_size_bytes",
		"ba.checksum AS asset_checksum",
		"ba.public_url AS asset_public_url",
	}, ", ")
}

func mapCommunityBookRow(row communityBookRow) domain.CommunityBook {
	return domain.CommunityBook{
		ID:               row.ID,
		CatalogBookID:    row.CatalogBookID,
		PreferredAssetID: row.PreferredAssetID,
		Owner: domain.CommunityBookOwner{
			ID:        row.OwnerID,
			Username:  row.OwnerUsername,
			AvatarURL: row.OwnerAvatarURL,
		},
		Title:       row.Title,
		Authors:     row.Authors,
		Identifiers: row.Identifiers,
		Language:    row.Language,
		SourceType:  row.SourceType,
		AddedAt:     row.AddedAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		PreferredAsset: domain.CommunityBookAsset{
			ID:        row.AssetID,
			MimeType:  row.AssetMimeType,
			SizeBytes: row.AssetSizeBytes,
			Checksum:  row.AssetChecksum,
			PublicURL: row.AssetPublicURL,
		},
	}
}
