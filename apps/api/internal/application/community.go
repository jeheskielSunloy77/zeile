package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/kern/internal/app/errs"
	"github.com/jeheskielSunloy77/kern/internal/app/sqlerr"
	"github.com/jeheskielSunloy77/kern/internal/application/port"
	"github.com/jeheskielSunloy77/kern/internal/domain"
	"gorm.io/gorm"
)

type CommunityService interface {
	ListBooks(ctx context.Context, query, ownerUsername string, limit, offset int) ([]domain.CommunityBook, int64, error)
	GetBook(ctx context.Context, libraryBookID uuid.UUID) (*domain.CommunityBook, error)
	SaveBook(ctx context.Context, userID, libraryBookID uuid.UUID) (*domain.UserLibraryBook, error)
}

type communityService struct {
	repo port.LibraryRepository
}

func NewCommunityService(repo port.LibraryRepository) CommunityService {
	return &communityService{repo: repo}
}

func (s *communityService) ListBooks(ctx context.Context, query, ownerUsername string, limit, offset int) ([]domain.CommunityBook, int64, error) {
	limit, offset = normalizePagination(limit, offset)
	books, total, err := s.repo.ListPublicCommunityBooks(ctx, query, ownerUsername, limit, offset)
	if err != nil {
		return nil, 0, sqlerr.HandleError(err)
	}
	return books, total, nil
}

func (s *communityService) GetBook(ctx context.Context, libraryBookID uuid.UUID) (*domain.CommunityBook, error) {
	book, err := s.repo.GetPublicCommunityBookByID(ctx, libraryBookID)
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}
	return book, nil
}

func (s *communityService) SaveBook(ctx context.Context, userID, libraryBookID uuid.UUID) (*domain.UserLibraryBook, error) {
	publicBook, err := s.repo.GetPublicCommunityBookByID(ctx, libraryBookID)
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}
	if publicBook.Owner.ID == userID {
		return nil, errs.NewForbiddenError("cannot save your own public book", true)
	}

	existing, err := s.repo.FindUserLibraryBookBySourceID(ctx, userID, libraryBookID)
	switch {
	case err == nil:
		return existing, nil
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return nil, sqlerr.HandleError(err)
	}

	sourceAsset, err := s.repo.GetBookAssetByID(ctx, publicBook.PreferredAssetID)
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}

	clonedSourceAssetID := sourceAsset.ID
	clonedAsset := &domain.BookAsset{
		CatalogBookID:  sourceAsset.CatalogBookID,
		UploaderUserID: userID,
		SourceAssetID:  &clonedSourceAssetID,
		StoragePath:    sourceAsset.StoragePath,
		PublicURL:      sourceAsset.PublicURL,
		MimeType:       sourceAsset.MimeType,
		SizeBytes:      sourceAsset.SizeBytes,
		Checksum:       sourceAsset.Checksum,
		IngestStatus:   sourceAsset.IngestStatus,
	}
	if err := s.repo.CreateBookAsset(ctx, clonedAsset); err != nil {
		return nil, sqlerr.HandleError(err)
	}

	sourceLibraryBookID := libraryBookID
	clonedAssetID := clonedAsset.ID
	book, err := s.repo.UpsertUserLibraryBook(ctx, &domain.UserLibraryBook{
		UserID:              userID,
		CatalogBookID:       publicBook.CatalogBookID,
		PreferredAssetID:    &clonedAssetID,
		SourceLibraryBookID: &sourceLibraryBookID,
		State:               domain.UserLibraryBookStateActive,
		IsPublic:            false,
		AddedAt:             time.Now().UTC(),
	})
	if err != nil {
		return nil, sqlerr.HandleError(err)
	}

	return book, nil
}
