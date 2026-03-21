package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/kern/internal/domain"
)

type AuthRepository interface {
	Save(ctx context.Context, user *domain.User) error
	CreateUser(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error)
	UpdateLoginAt(ctx context.Context, id uuid.UUID, ts time.Time) error
	UpdateEmailVerifiedAt(ctx context.Context, id uuid.UUID, ts time.Time) error
}

type AuthSessionRepository interface {
	Create(ctx context.Context, session *domain.AuthSession) error
	GetByRefreshTokenHash(ctx context.Context, hash string) (*domain.AuthSession, error)
	RevokeByID(ctx context.Context, id uuid.UUID, revokedAt time.Time) error
	RevokeByUserID(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error
}

type EmailVerificationRepository interface {
	Create(ctx context.Context, verification *domain.EmailVerification) error
	GetActiveByUserIDAndCodeHash(ctx context.Context, userID uuid.UUID, codeHash string, now time.Time) (*domain.EmailVerification, error)
	ExpireActiveByUserID(ctx context.Context, userID uuid.UUID, now time.Time) error
	MarkVerified(ctx context.Context, id uuid.UUID, verifiedAt time.Time) error
}

type UserRepository interface {
	ResourceRepository[domain.User]
}

type LibraryRepository interface {
	CreateCatalogBook(ctx context.Context, book *domain.BookCatalog) error
	ListCatalogBooks(ctx context.Context, limit, offset int) ([]domain.BookCatalog, int64, error)
	GetCatalogBookByID(ctx context.Context, id uuid.UUID) (*domain.BookCatalog, error)

	CreateBookAsset(ctx context.Context, asset *domain.BookAsset) error
	GetBookAssetByID(ctx context.Context, id uuid.UUID) (*domain.BookAsset, error)

	UpsertUserLibraryBook(ctx context.Context, book *domain.UserLibraryBook) (*domain.UserLibraryBook, error)
	GetUserLibraryBookByID(ctx context.Context, userID, id uuid.UUID) (*domain.UserLibraryBook, error)
	ListUserLibraryBooks(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.UserLibraryBook, int64, error)
	UpdateUserLibraryBook(ctx context.Context, userID, id uuid.UUID, updates map[string]any) (*domain.UserLibraryBook, error)
	DeleteUserLibraryBook(ctx context.Context, userID, id uuid.UUID) error
	FindUserLibraryBookBySourceID(ctx context.Context, userID, sourceLibraryBookID uuid.UUID) (*domain.UserLibraryBook, error)
	ListPublicCommunityBooks(ctx context.Context, query, ownerUsername string, limit, offset int) ([]domain.CommunityBook, int64, error)
	GetPublicCommunityBookByID(ctx context.Context, id uuid.UUID) (*domain.CommunityBook, error)

	UpsertReadingState(ctx context.Context, state *domain.ReadingState, expectedVersion *int64) (*domain.ReadingState, error)
	GetReadingState(ctx context.Context, userID, userLibraryBookID uuid.UUID, mode string) (*domain.ReadingState, error)

	CreateHighlight(ctx context.Context, highlight *domain.Highlight) error
	ListHighlights(ctx context.Context, userID, userLibraryBookID uuid.UUID) ([]domain.Highlight, error)
	GetHighlightByID(ctx context.Context, userID, id uuid.UUID) (*domain.Highlight, error)
	UpdateHighlight(ctx context.Context, userID, id uuid.UUID, updates map[string]any) (*domain.Highlight, error)
	DeleteHighlight(ctx context.Context, userID, id uuid.UUID) error

	GetIdempotencyKey(ctx context.Context, userID uuid.UUID, operation, key string) (*domain.IdempotencyKey, error)
	CreateIdempotencyKey(ctx context.Context, idempotency *domain.IdempotencyKey) error
}

type Repositories struct {
	Auth              AuthRepository
	AuthSession       AuthSessionRepository
	User              UserRepository
	EmailVerification EmailVerificationRepository
	Library           LibraryRepository
}
