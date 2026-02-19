package repository

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/zeile/internal/application/port"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	"gorm.io/gorm"
)

type AuthRepository = port.AuthRepository

type authRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &authRepository{db: db}
}

func (r *authRepository) Save(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *authRepository) CreateUser(ctx context.Context, user *domain.User) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *authRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, "LOWER(email) = ?", strings.ToLower(email)).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, "LOWER(username) = ?", strings.ToLower(username)).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, "google_id = ?", googleID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) UpdateLoginAt(ctx context.Context, id uuid.UUID, ts time.Time) error {
	return r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", id).
		Update("last_login_at", ts).
		Error
}

func (r *authRepository) UpdateEmailVerifiedAt(ctx context.Context, id uuid.UUID, ts time.Time) error {
	return r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", id).
		Update("email_verified_at", ts).
		Error
}
