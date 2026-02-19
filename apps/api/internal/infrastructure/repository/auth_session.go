package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/zeile/internal/application/port"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	"gorm.io/gorm"
)

type AuthSessionRepository = port.AuthSessionRepository

type authSessionRepository struct {
	db *gorm.DB
}

func NewAuthSessionRepository(db *gorm.DB) AuthSessionRepository {
	return &authSessionRepository{db: db}
}

func (r *authSessionRepository) Create(ctx context.Context, session *domain.AuthSession) error {
	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *authSessionRepository) GetByRefreshTokenHash(ctx context.Context, hash string) (*domain.AuthSession, error) {
	var session domain.AuthSession
	if err := r.db.WithContext(ctx).First(&session, "refresh_token_hash = ?", hash).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *authSessionRepository) RevokeByID(ctx context.Context, id uuid.UUID, revokedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&domain.AuthSession{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"revoked_at": revokedAt,
		}).
		Error
}

func (r *authSessionRepository) RevokeByUserID(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&domain.AuthSession{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Updates(map[string]any{
			"revoked_at": revokedAt,
		}).
		Error
}
