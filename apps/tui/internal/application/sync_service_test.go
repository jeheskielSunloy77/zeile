package application

import (
	"context"
	"testing"
	"time"

	"github.com/zeile/tui/internal/domain"
	"github.com/zeile/tui/internal/infrastructure/remote"
	"github.com/zeile/tui/internal/infrastructure/repository"
)

type mockSyncLibrary struct {
	books  []domain.Book
	states map[string][]domain.ReadingState
}

func (m *mockSyncLibrary) ListBooks(ctx context.Context) ([]domain.Book, error) {
	return m.books, nil
}

func (m *mockSyncLibrary) StatesForBook(ctx context.Context, bookID string) ([]domain.ReadingState, error) {
	return m.states[bookID], nil
}

type mockSyncAccountRepo struct {
	accounts []domain.SyncAccount
}

func (m *mockSyncAccountRepo) Upsert(ctx context.Context, account domain.SyncAccount) error {
	m.accounts = append(m.accounts, account)
	return nil
}

type mockSyncLinkRepo struct {
	links    map[string]domain.SyncBookLink
	upserted []domain.SyncBookLink
}

func (m *mockSyncLinkRepo) GetByLocalBookID(ctx context.Context, localBookID string) (domain.SyncBookLink, error) {
	if link, ok := m.links[localBookID]; ok {
		return link, nil
	}
	return domain.SyncBookLink{}, repository.ErrNotFound
}

func (m *mockSyncLinkRepo) UpsertBookLink(ctx context.Context, link domain.SyncBookLink) error {
	if m.links == nil {
		m.links = make(map[string]domain.SyncBookLink)
	}
	m.links[link.LocalBookID] = link
	m.upserted = append(m.upserted, link)
	return nil
}

type mockSyncRemote struct {
	createCatalogCalls int
	upsertLibraryCalls int
	upsertStateCalls   int

	catalogID string
	libraryID string
}

func (m *mockSyncRemote) CreateCatalogBook(ctx context.Context, accessToken, title, authors string) (remote.BookCatalog, error) {
	m.createCatalogCalls++
	return remote.BookCatalog{ID: m.catalogID}, nil
}

func (m *mockSyncRemote) UpsertLibraryBook(ctx context.Context, accessToken, catalogBookID string) (remote.UserLibraryBook, error) {
	m.upsertLibraryCalls++
	return remote.UserLibraryBook{ID: m.libraryID}, nil
}

func (m *mockSyncRemote) UpsertReadingState(ctx context.Context, accessToken, libraryBookID, mode string, locator map[string]any, progressPercent float64) error {
	m.upsertStateCalls++
	return nil
}

func TestSyncServiceReconcileNow_FirstLinkCreation(t *testing.T) {
	auth := &AuthService{
		session: &remote.Session{
			User: remote.User{
				ID:       "user-1",
				Email:    "reader@example.com",
				Username: "reader",
			},
			AccessToken:     "access-token",
			AccessExpiresAt: time.Now().UTC().Add(30 * time.Minute),
		},
	}

	book := domain.Book{
		ID:          "book-1",
		Fingerprint: "fp-book-1",
		Title:       "Local Book",
		Author:      "Author",
	}
	library := &mockSyncLibrary{
		books: []domain.Book{book},
		states: map[string][]domain.ReadingState{
			"book-1": {
				{BookID: "book-1", Mode: domain.ReadingModeEPUB, Locator: domain.Locator{Offset: 15}, ProgressPercent: 12.5},
				{BookID: "book-1", Mode: domain.ReadingModePDFText, Locator: domain.Locator{PageIndex: 1}, ProgressPercent: 35},
			},
		},
	}

	accountRepo := &mockSyncAccountRepo{}
	linkRepo := &mockSyncLinkRepo{
		links: map[string]domain.SyncBookLink{},
	}
	remoteClient := &mockSyncRemote{
		catalogID: "catalog-1",
		libraryID: "library-1",
	}

	service := NewSyncService(auth, library, accountRepo, linkRepo, remoteClient)
	result, err := service.ReconcileNow(context.Background())
	if err != nil {
		t.Fatalf("reconcile now: %v", err)
	}

	if result.SyncedBooks != 1 {
		t.Fatalf("expected 1 synced book, got %d", result.SyncedBooks)
	}
	if result.SyncedStates != 2 {
		t.Fatalf("expected 2 synced states, got %d", result.SyncedStates)
	}
	if result.SkippedBooks != 0 {
		t.Fatalf("expected 0 skipped books, got %d", result.SkippedBooks)
	}
	if remoteClient.createCatalogCalls != 1 {
		t.Fatalf("expected 1 catalog call, got %d", remoteClient.createCatalogCalls)
	}
	if remoteClient.upsertLibraryCalls != 1 {
		t.Fatalf("expected 1 upsert library call, got %d", remoteClient.upsertLibraryCalls)
	}
	if remoteClient.upsertStateCalls != 2 {
		t.Fatalf("expected 2 upsert state calls, got %d", remoteClient.upsertStateCalls)
	}
	if len(linkRepo.upserted) != 1 {
		t.Fatalf("expected one persisted link, got %d", len(linkRepo.upserted))
	}
	if len(accountRepo.accounts) != 1 {
		t.Fatalf("expected one persisted sync account, got %d", len(accountRepo.accounts))
	}
}

func TestSyncServiceReconcileNow_ExistingLinkSkipsCatalogCreation(t *testing.T) {
	auth := &AuthService{
		session: &remote.Session{
			User: remote.User{
				ID:       "user-1",
				Email:    "reader@example.com",
				Username: "reader",
			},
			AccessToken:     "access-token",
			AccessExpiresAt: time.Now().UTC().Add(30 * time.Minute),
		},
	}

	book := domain.Book{
		ID:          "book-1",
		Fingerprint: "fp-book-1",
		Title:       "Local Book",
		Author:      "Author",
	}
	library := &mockSyncLibrary{
		books: []domain.Book{book},
		states: map[string][]domain.ReadingState{
			"book-1": {
				{BookID: "book-1", Mode: domain.ReadingModeEPUB, Locator: domain.Locator{Offset: 10}, ProgressPercent: 10},
			},
		},
	}

	linkRepo := &mockSyncLinkRepo{
		links: map[string]domain.SyncBookLink{
			"book-1": {
				LocalBookID:         "book-1",
				LocalFingerprint:    "fp-book-1",
				RemoteCatalogBookID: "catalog-1",
				RemoteLibraryBookID: "library-1",
			},
		},
	}
	accountRepo := &mockSyncAccountRepo{}
	remoteClient := &mockSyncRemote{
		catalogID: "catalog-ignored",
		libraryID: "library-ignored",
	}

	service := NewSyncService(auth, library, accountRepo, linkRepo, remoteClient)
	result, err := service.ReconcileNow(context.Background())
	if err != nil {
		t.Fatalf("reconcile now: %v", err)
	}

	if result.SyncedBooks != 0 {
		t.Fatalf("expected 0 synced books, got %d", result.SyncedBooks)
	}
	if result.SkippedBooks != 1 {
		t.Fatalf("expected 1 skipped book, got %d", result.SkippedBooks)
	}
	if result.SyncedStates != 1 {
		t.Fatalf("expected 1 synced state, got %d", result.SyncedStates)
	}
	if remoteClient.createCatalogCalls != 0 {
		t.Fatalf("expected no catalog creation calls, got %d", remoteClient.createCatalogCalls)
	}
	if remoteClient.upsertLibraryCalls != 0 {
		t.Fatalf("expected no upsert library calls, got %d", remoteClient.upsertLibraryCalls)
	}
	if remoteClient.upsertStateCalls != 1 {
		t.Fatalf("expected one upsert state call, got %d", remoteClient.upsertStateCalls)
	}
}
