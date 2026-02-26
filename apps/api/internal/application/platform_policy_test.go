package application

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jeheskielSunloy77/zeile/internal/app/errs"
	applicationdto "github.com/jeheskielSunloy77/zeile/internal/application/dto"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/repository"
	internaltesting "github.com/jeheskielSunloy77/zeile/internal/testing"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSharingServiceUpsertBookSharePolicy_RequiresVerifiedCatalog(t *testing.T) {
	testDB, cleanup := internaltesting.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	err := internaltesting.WithRollbackTransaction(ctx, testDB, func(tx *gorm.DB) error {
		require.NoError(t, tx.Create(&domain.User{ID: uuid.New(), Email: "owner@example.com", Username: "owner"}).Error)
		var owner domain.User
		require.NoError(t, tx.First(&owner, "email = ?", "owner@example.com").Error)

		libraryRepo := repository.NewLibraryRepository(tx)
		sharingRepo := repository.NewSharingRepository(tx)
		communityRepo := repository.NewCommunityRepository(tx)

		catalog := &domain.BookCatalog{
			Title:              "Book One",
			Authors:            "Author",
			VerificationStatus: domain.VerificationStatusPending,
			SourceType:         "user_upload",
		}
		require.NoError(t, libraryRepo.CreateCatalogBook(ctx, catalog))

		libraryBook, err := libraryRepo.UpsertUserLibraryBook(ctx, &domain.UserLibraryBook{
			UserID:              owner.ID,
			CatalogBookID:       catalog.ID,
			State:               domain.UserLibraryBookStateActive,
			VisibilityInProfile: true,
		})
		require.NoError(t, err)

		service := NewSharingService(sharingRepo, libraryRepo, communityRepo)

		_, err = service.UpsertBookSharePolicy(ctx, owner.ID, applicationdto.UpsertBookSharePolicyInput{
			UserLibraryBookID:    libraryBook.ID,
			RawFileSharing:       domain.RawFileSharingPublicLink,
			AllowMetadataSharing: true,
		})
		require.Error(t, err)

		var httpErr *errs.ErrorResponse
		require.ErrorAs(t, err, &httpErr)
		require.Equal(t, http.StatusBadRequest, httpErr.Status)
		require.Equal(t, "verification_required", httpErr.Message)

		_, err = libraryRepo.UpdateCatalogVerification(ctx, catalog.ID, domain.VerificationStatusVerifiedPublicDomain)
		require.NoError(t, err)

		policy, err := service.UpsertBookSharePolicy(ctx, owner.ID, applicationdto.UpsertBookSharePolicyInput{
			UserLibraryBookID:    libraryBook.ID,
			RawFileSharing:       domain.RawFileSharingPublicLink,
			AllowMetadataSharing: true,
		})
		require.NoError(t, err)
		require.Equal(t, domain.RawFileSharingPublicLink, policy.RawFileSharing)
		return nil
	})
	require.NoError(t, err)
}

func TestLibraryServiceUpsertReadingState_ConflictVersionMismatch(t *testing.T) {
	testDB, cleanup := internaltesting.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	err := internaltesting.WithRollbackTransaction(ctx, testDB, func(tx *gorm.DB) error {
		require.NoError(t, tx.Create(&domain.User{ID: uuid.New(), Email: "reader@example.com", Username: "reader"}).Error)
		var reader domain.User
		require.NoError(t, tx.First(&reader, "email = ?", "reader@example.com").Error)

		libraryRepo := repository.NewLibraryRepository(tx)
		communityRepo := repository.NewCommunityRepository(tx)
		service := NewLibraryService(libraryRepo, communityRepo, nil)

		catalog := &domain.BookCatalog{
			Title:              "Reading Book",
			Authors:            "Author",
			VerificationStatus: domain.VerificationStatusPending,
			SourceType:         "user_upload",
		}
		require.NoError(t, libraryRepo.CreateCatalogBook(ctx, catalog))

		libraryBook, err := libraryRepo.UpsertUserLibraryBook(ctx, &domain.UserLibraryBook{
			UserID:              reader.ID,
			CatalogBookID:       catalog.ID,
			State:               domain.UserLibraryBookStateActive,
			VisibilityInProfile: true,
		})
		require.NoError(t, err)

		state, err := service.UpsertReadingState(ctx, reader.ID, libraryBook.ID, applicationdto.UpsertReadingStateInput{
			Mode:            domain.ReadingModeEPUB,
			LocatorJSON:     []byte(`{"chapter":1,"offset":42}`),
			ProgressPercent: 12.5,
		})
		require.NoError(t, err)
		require.Equal(t, int64(1), state.Version)

		wrongVersion := int64(99)
		_, err = service.UpsertReadingState(ctx, reader.ID, libraryBook.ID, applicationdto.UpsertReadingStateInput{
			Mode:            domain.ReadingModeEPUB,
			LocatorJSON:     []byte(`{"chapter":1,"offset":99}`),
			ProgressPercent: 22,
			IfMatchVersion:  &wrongVersion,
		})
		require.Error(t, err)

		var httpErr *errs.ErrorResponse
		require.ErrorAs(t, err, &httpErr)
		require.Equal(t, 409, httpErr.Status)
		require.Equal(t, "conflict_version_mismatch", httpErr.Message)
		return nil
	})
	require.NoError(t, err)
}

func TestModerationServiceDecideReview_UpdatesCatalogVerification(t *testing.T) {
	testDB, cleanup := internaltesting.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	err := internaltesting.WithRollbackTransaction(ctx, testDB, func(tx *gorm.DB) error {
		require.NoError(t, tx.Create(&domain.User{ID: uuid.New(), Email: "submitter@example.com", Username: "submitter"}).Error)
		require.NoError(t, tx.Create(&domain.User{ID: uuid.New(), Email: "admin@example.com", Username: "admin", IsAdmin: true}).Error)

		var submitter domain.User
		var admin domain.User
		require.NoError(t, tx.First(&submitter, "email = ?", "submitter@example.com").Error)
		require.NoError(t, tx.First(&admin, "email = ?", "admin@example.com").Error)

		libraryRepo := repository.NewLibraryRepository(tx)
		moderationRepo := repository.NewModerationRepository(tx)
		service := NewModerationService(moderationRepo, libraryRepo)

		catalog := &domain.BookCatalog{
			Title:              "Moderated Book",
			Authors:            "Author",
			VerificationStatus: domain.VerificationStatusPending,
			SourceType:         "user_upload",
		}
		require.NoError(t, libraryRepo.CreateCatalogBook(ctx, catalog))

		review, err := service.CreateReview(ctx, submitter.ID, applicationdto.CreateModerationReviewInput{
			CatalogBookID: catalog.ID,
			EvidenceJSON:  []byte(`{"source":"public-domain-proof"}`),
		})
		require.NoError(t, err)
		require.Equal(t, domain.ModerationStatusPending, review.Status)

		decided, err := service.DecideReview(ctx, admin.ID, review.ID, applicationdto.DecideModerationReviewInput{Decision: "approved"})
		require.NoError(t, err)
		require.Equal(t, domain.ModerationStatusApproved, decided.Status)

		updatedCatalog, err := libraryRepo.GetCatalogBookByID(ctx, catalog.ID)
		require.NoError(t, err)
		require.Equal(t, domain.VerificationStatusVerifiedPublicDomain, updatedCatalog.VerificationStatus)
		return nil
	})
	require.NoError(t, err)
}
