package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/zeile/tui/internal/domain"
	"github.com/zeile/tui/internal/infrastructure/remote"
	"github.com/zeile/tui/internal/infrastructure/repository"
)

type SyncResult struct {
	SyncedBooks  int
	SyncedStates int
	SkippedBooks int
}

type SyncRemoteClient interface {
	CreateCatalogBook(ctx context.Context, accessToken, title, authors string) (remote.BookCatalog, error)
	UpsertLibraryBook(ctx context.Context, accessToken, catalogBookID string) (remote.UserLibraryBook, error)
	UpsertReadingState(ctx context.Context, accessToken, libraryBookID, mode string, locator map[string]any, progressPercent float64) error
}

type SyncService struct {
	auth     *AuthService
	library  SyncLibraryService
	accounts SyncAccountRepository
	links    SyncBookLinkRepository
	remote   SyncRemoteClient

	mu sync.Mutex
}

type SyncLibraryService interface {
	ListBooks(ctx context.Context) ([]domain.Book, error)
	StatesForBook(ctx context.Context, bookID string) ([]domain.ReadingState, error)
}

func NewSyncService(
	auth *AuthService,
	library SyncLibraryService,
	accounts SyncAccountRepository,
	links SyncBookLinkRepository,
	remoteClient SyncRemoteClient,
) *SyncService {
	return &SyncService{
		auth:     auth,
		library:  library,
		accounts: accounts,
		links:    links,
		remote:   remoteClient,
	}
}

func (s *SyncService) ReconcileNow(ctx context.Context) (SyncResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s == nil || s.auth == nil || s.library == nil || s.links == nil || s.remote == nil {
		return SyncResult{}, errors.New("sync service is not configured")
	}

	session, ok := s.auth.Session()
	if !ok || strings.TrimSpace(session.AccessToken) == "" {
		return SyncResult{}, errors.New("not connected")
	}
	if !session.AccessExpiresAt.After(time.Now().UTC()) {
		return SyncResult{}, errors.New("session expired")
	}

	books, err := s.library.ListBooks(ctx)
	if err != nil {
		return SyncResult{}, fmt.Errorf("list local books: %w", err)
	}

	result := SyncResult{}
	accessToken := session.AccessToken
	for _, book := range books {
		link, err := s.links.GetByLocalBookID(ctx, book.ID)
		switch {
		case err == nil:
			pushedStates, pushErr := s.pushReadingStates(ctx, accessToken, book.ID, link.RemoteLibraryBookID)
			if pushErr != nil {
				return result, pushErr
			}
			result.SkippedBooks++
			result.SyncedStates += pushedStates
			continue
		case errors.Is(err, repository.ErrNotFound):
			// Continue with first-time reconciliation.
		default:
			return result, fmt.Errorf("get sync link for %s: %w", book.ID, err)
		}

		catalog, err := s.remote.CreateCatalogBook(ctx, accessToken, book.Title, book.Author)
		if err != nil {
			return result, fmt.Errorf("create catalog entry for %q: %w", book.Title, err)
		}

		libraryBook, err := s.remote.UpsertLibraryBook(ctx, accessToken, catalog.ID)
		if err != nil {
			return result, fmt.Errorf("upsert remote library book for %q: %w", book.Title, err)
		}

		now := time.Now().UTC()
		if err := s.links.UpsertBookLink(ctx, domain.SyncBookLink{
			LocalBookID:         book.ID,
			LocalFingerprint:    book.Fingerprint,
			RemoteCatalogBookID: catalog.ID,
			RemoteLibraryBookID: libraryBook.ID,
			UpdatedAt:           now,
		}); err != nil {
			return result, fmt.Errorf("persist sync link for %q: %w", book.Title, err)
		}

		pushedStates, pushErr := s.pushReadingStates(ctx, accessToken, book.ID, libraryBook.ID)
		if pushErr != nil {
			return result, pushErr
		}

		result.SyncedBooks++
		result.SyncedStates += pushedStates
	}

	if s.accounts != nil {
		now := time.Now().UTC()
		if err := s.accounts.Upsert(ctx, domain.SyncAccount{
			UserID:           session.User.ID,
			Email:            session.User.Email,
			Username:         session.User.Username,
			LastReconciledAt: &now,
			UpdatedAt:        now,
		}); err != nil {
			return result, fmt.Errorf("upsert sync account: %w", err)
		}
	}

	return result, nil
}

func (s *SyncService) pushReadingStates(ctx context.Context, accessToken, localBookID, remoteLibraryBookID string) (int, error) {
	states, err := s.library.StatesForBook(ctx, localBookID)
	if err != nil {
		return 0, fmt.Errorf("list reading states for %s: %w", localBookID, err)
	}
	if len(states) == 0 {
		return 0, nil
	}

	pushed := 0
	for _, state := range states {
		locator, err := locatorToMap(state.Locator)
		if err != nil {
			return pushed, fmt.Errorf("serialize locator for book %s mode %s: %w", localBookID, state.Mode, err)
		}
		if err := s.remote.UpsertReadingState(
			ctx,
			accessToken,
			remoteLibraryBookID,
			string(state.Mode),
			locator,
			state.ProgressPercent,
		); err != nil {
			return pushed, fmt.Errorf("upsert reading state for book %s mode %s: %w", localBookID, state.Mode, err)
		}
		pushed++
	}
	return pushed, nil
}

func locatorToMap(locator domain.Locator) (map[string]any, error) {
	encoded, err := json.Marshal(locator)
	if err != nil {
		return nil, err
	}

	decoded := make(map[string]any)
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}
