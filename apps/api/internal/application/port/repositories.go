package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
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

type Repositories struct {
	Auth              AuthRepository
	AuthSession       AuthSessionRepository
	User              UserRepository
	EmailVerification EmailVerificationRepository
}
