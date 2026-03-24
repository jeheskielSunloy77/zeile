package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jeheskielSunloy77/kern/tui/internal/domain"
	"github.com/jeheskielSunloy77/kern/tui/internal/infrastructure/remote"
	"github.com/jeheskielSunloy77/kern/tui/internal/infrastructure/repository"
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

func (m *mockSyncLinkRepo) ListBookLinks(ctx context.Context) ([]domain.SyncBookLink, error) {
	links := make([]domain.SyncBookLink, 0, len(m.links))
	for _, link := range m.links {
		links = append(links, link)
	}
	return links, nil
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
	uploadAssetCalls   int
	updateAssetCalls   int

	catalogID string
	libraryID string
	assetID   string

	preferredAssetID *string
	uploadErr        error
}

func (m *mockSyncRemote) CreateCatalogBook(ctx context.Context, accessToken, title, authors string) (remote.BookCatalog, error) {
	m.createCatalogCalls++
	return remote.BookCatalog{ID: m.catalogID}, nil
}

func (m *mockSyncRemote) UpsertLibraryBook(ctx context.Context, accessToken, catalogBookID string) (remote.UserLibraryBook, error) {
	m.upsertLibraryCalls++
	return remote.UserLibraryBook{
		ID:               m.libraryID,
		CatalogBookID:    catalogBookID,
		PreferredAssetID: m.preferredAssetID,
	}, nil
}

func (m *mockSyncRemote) ListLibraryBooks(ctx context.Context, accessToken string) ([]remote.UserLibraryBook, error) {
	book := remote.UserLibraryBook{
		ID:               m.libraryID,
		CatalogBookID:    m.catalogID,
		PreferredAssetID: m.preferredAssetID,
	}
	if book.ID == "" {
		return nil, nil
	}
	return []remote.UserLibraryBook{book}, nil
}

func (m *mockSyncRemote) UploadBookAsset(ctx context.Context, accessToken, catalogBookID, filePath string) (remote.BookAsset, error) {
	m.uploadAssetCalls++
	if m.uploadErr != nil {
		return remote.BookAsset{}, m.uploadErr
	}
	return remote.BookAsset{
		ID:            m.assetID,
		CatalogBookID: catalogBookID,
	}, nil
}

func (m *mockSyncRemote) UpdateLibraryBookPreferredAsset(ctx context.Context, accessToken, libraryBookID, preferredAssetID string) (remote.UserLibraryBook, error) {
	m.updateAssetCalls++
	m.preferredAssetID = &preferredAssetID
	return remote.UserLibraryBook{
		ID:               libraryBookID,
		PreferredAssetID: &preferredAssetID,
	}, nil
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
	filePath := seedSyncFile(t, "book-1.epub")
	book.ManagedPath = filePath
	library := &mockSyncLibrary{
		books: []domain.Book{book},
		states: map[string][]domain.ReadingState{
			"book-1": {
				{BookID: "book-1", Mode: domain.ReadingModeEPUB, Locator: domain.Locator{Offset: 15}, ProgressPercent: 12.5},
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
		assetID:   "asset-1",
	}

	service := NewSyncService(auth, library, accountRepo, linkRepo, remoteClient)
	result, err := service.ReconcileNow(context.Background())
	if err != nil {
		t.Fatalf("reconcile now: %v", err)
	}

	if result.SyncedBooks != 1 {
		t.Fatalf("expected 1 synced book, got %d", result.SyncedBooks)
	}
	if result.SyncedStates != 1 {
		t.Fatalf("expected 1 synced state, got %d", result.SyncedStates)
	}
	if result.SkippedBooks != 0 {
		t.Fatalf("expected 0 skipped books, got %d", result.SkippedBooks)
	}
	if result.UploadedFiles != 1 {
		t.Fatalf("expected 1 uploaded file, got %d", result.UploadedFiles)
	}
	if result.UploadFailures != 0 {
		t.Fatalf("expected 0 upload failures, got %d", result.UploadFailures)
	}
	if remoteClient.createCatalogCalls != 1 {
		t.Fatalf("expected 1 catalog call, got %d", remoteClient.createCatalogCalls)
	}
	if remoteClient.upsertLibraryCalls != 1 {
		t.Fatalf("expected 1 upsert library call, got %d", remoteClient.upsertLibraryCalls)
	}
	if remoteClient.upsertStateCalls != 1 {
		t.Fatalf("expected 1 upsert state call, got %d", remoteClient.upsertStateCalls)
	}
	if remoteClient.uploadAssetCalls != 1 {
		t.Fatalf("expected 1 upload asset call, got %d", remoteClient.uploadAssetCalls)
	}
	if remoteClient.updateAssetCalls != 1 {
		t.Fatalf("expected 1 preferred asset update call, got %d", remoteClient.updateAssetCalls)
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
	filePath := seedSyncFile(t, "book-1.epub")
	book.ManagedPath = filePath
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
		catalogID:        "catalog-1",
		libraryID:        "library-1",
		preferredAssetID: ptr("asset-existing"),
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
	if remoteClient.uploadAssetCalls != 0 {
		t.Fatalf("expected no upload asset calls, got %d", remoteClient.uploadAssetCalls)
	}
}

func TestSyncServiceReconcileNow_UploadFailureDoesNotAbortSync(t *testing.T) {
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
		ManagedPath: seedSyncFile(t, "book-1.epub"),
	}
	library := &mockSyncLibrary{
		books: []domain.Book{book},
		states: map[string][]domain.ReadingState{
			"book-1": {
				{BookID: "book-1", Mode: domain.ReadingModeEPUB, Locator: domain.Locator{Offset: 10}, ProgressPercent: 10},
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
		uploadErr: errors.New("upload failed"),
	}

	service := NewSyncService(auth, library, accountRepo, linkRepo, remoteClient)
	result, err := service.ReconcileNow(context.Background())
	if err != nil {
		t.Fatalf("reconcile now: %v", err)
	}

	if result.UploadFailures != 1 {
		t.Fatalf("expected 1 upload failure, got %d", result.UploadFailures)
	}
	if result.SyncedStates != 1 {
		t.Fatalf("expected reading states to keep syncing, got %d", result.SyncedStates)
	}
	if result.LastUploadError == "" {
		t.Fatalf("expected upload error summary")
	}
	if remoteClient.upsertStateCalls != 1 {
		t.Fatalf("expected one upsert state call, got %d", remoteClient.upsertStateCalls)
	}
}

func seedSyncFile(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("book-bytes"), 0o644); err != nil {
		t.Fatalf("write sync file: %v", err)
	}
	return path
}

func ptr(value string) *string {
	return &value
}
