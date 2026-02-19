package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
	internaltesting "github.com/jeheskielSunloy77/zeile/internal/testing"
	"gorm.io/gorm"

	"github.com/stretchr/testify/require"
)

// Ensures email verification records can be created, retrieved, and marked verified.
func TestEmailVerificationRepository_Lifecycle(t *testing.T) {
	testDB, cleanup := internaltesting.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	err := internaltesting.WithRollbackTransaction(ctx, testDB, func(tx *gorm.DB) error {
		userRepo := NewUserRepository(&config.Config{}, tx, nil)
		repo := NewEmailVerificationRepository(tx)

		user := &domain.User{
			ID:       uuid.New(),
			Email:    "user@example.com",
			Username: "user",
		}
		require.NoError(t, userRepo.Store(ctx, user))

		now := time.Now().UTC()
		verification := &domain.EmailVerification{
			UserID:    user.ID,
			Email:     user.Email,
			CodeHash:  "hash",
			ExpiresAt: now.Add(time.Hour),
		}
		require.NoError(t, repo.Create(ctx, verification))

		fetched, err := repo.GetActiveByUserIDAndCodeHash(ctx, user.ID, "hash", now)
		require.NoError(t, err)
		require.Equal(t, verification.ID, fetched.ID)

		require.NoError(t, repo.MarkVerified(ctx, verification.ID, now))
		_, err = repo.GetActiveByUserIDAndCodeHash(ctx, user.ID, "hash", now)
		require.Error(t, err)

		return nil
	})
	require.NoError(t, err)
}
