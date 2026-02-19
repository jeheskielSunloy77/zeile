package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	internaltesting "github.com/jeheskielSunloy77/zeile/internal/testing"
	"gorm.io/gorm"

	"github.com/stretchr/testify/require"
)

// Ensures auth repository creates users and can lookup by email, username, and Google ID.
func TestAuthRepository_CreateAndLookupUser(t *testing.T) {
	testDB, cleanup := internaltesting.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	err := internaltesting.WithRollbackTransaction(ctx, testDB, func(tx *gorm.DB) error {
		repo := NewAuthRepository(tx)
		googleID := "google-123"
		user := &domain.User{
			Email:        "Test@Example.com",
			Username:     "TestUser",
			PasswordHash: "hash",
			GoogleID:     &googleID,
		}

		require.NoError(t, repo.CreateUser(ctx, user))
		require.NotEqual(t, uuid.Nil, user.ID)

		byEmail, err := repo.GetByEmail(ctx, "TEST@EXAMPLE.COM")
		require.NoError(t, err)
		require.Equal(t, user.ID, byEmail.ID)

		byUsername, err := repo.GetByUsername(ctx, "testuser")
		require.NoError(t, err)
		require.Equal(t, user.ID, byUsername.ID)

		byGoogle, err := repo.GetByGoogleID(ctx, googleID)
		require.NoError(t, err)
		require.Equal(t, user.ID, byGoogle.ID)

		loginAt := time.Now().UTC().Truncate(time.Second)
		require.NoError(t, repo.UpdateLoginAt(ctx, user.ID, loginAt))

		verifiedAt := time.Now().UTC().Truncate(time.Second)
		require.NoError(t, repo.UpdateEmailVerifiedAt(ctx, user.ID, verifiedAt))

		updated, err := repo.GetByID(ctx, user.ID)
		require.NoError(t, err)
		require.NotNil(t, updated.LastLoginAt)
		require.WithinDuration(t, loginAt, *updated.LastLoginAt, time.Second)
		require.NotNil(t, updated.EmailVerifiedAt)
		require.WithinDuration(t, verifiedAt, *updated.EmailVerifiedAt, time.Second)

		return nil
	})
	require.NoError(t, err)
}
