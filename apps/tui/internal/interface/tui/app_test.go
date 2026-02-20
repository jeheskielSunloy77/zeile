package tui

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zeile/tui/internal/application"
	"github.com/zeile/tui/internal/domain"
	"github.com/zeile/tui/internal/infrastructure/config"
	"github.com/zeile/tui/internal/infrastructure/database"
	"github.com/zeile/tui/internal/infrastructure/repository"
	"github.com/zeile/tui/internal/infrastructure/storage"
	"github.com/zeile/tui/internal/preprocessing"
)

func TestStartupAutoResumeOpensMostRecentUnfinishedBook(t *testing.T) {
	container := newTestContainer(t)

	older := time.Now().UTC().Add(-2 * time.Hour)
	newer := time.Now().UTC().Add(-1 * time.Hour)

	resumeBook := seedBookWithEPUBCache(t, container, "resume-book", "/tmp/resume.epub")
	otherBook := seedBookWithEPUBCache(t, container, "other-book", "/tmp/other.epub")

	if err := container.Library.UpdateReadingState(context.Background(), domain.ReadingState{
		BookID:          resumeBook.ID,
		Mode:            domain.ReadingModeEPUB,
		Locator:         domain.Locator{Offset: 12, SectionIndex: 1},
		ProgressPercent: 24,
		UpdatedAt:       newer,
		IsFinished:      false,
	}); err != nil {
		t.Fatalf("seed resume reading state: %v", err)
	}

	if err := container.Library.UpdateReadingState(context.Background(), domain.ReadingState{
		BookID:          otherBook.ID,
		Mode:            domain.ReadingModeEPUB,
		Locator:         domain.Locator{Offset: 8},
		ProgressPercent: 60,
		UpdatedAt:       older,
		IsFinished:      false,
	}); err != nil {
		t.Fatalf("seed other reading state: %v", err)
	}

	if err := container.Library.UpdateReadingState(context.Background(), domain.ReadingState{
		BookID:          otherBook.ID,
		Mode:            domain.ReadingModePDFText,
		Locator:         domain.Locator{Offset: 0},
		ProgressPercent: 100,
		UpdatedAt:       time.Now().UTC(),
		IsFinished:      true,
	}); err != nil {
		t.Fatalf("seed finished reading state: %v", err)
	}

	m := New(container).(model)
	startupMsg, ok := m.loadStartupCmd()().(startupLoadedMsg)
	if !ok {
		t.Fatalf("expected startupLoadedMsg")
	}

	updated, _ := m.Update(startupMsg)
	after := updated.(model)

	if !after.startupCompleted {
		t.Fatalf("expected startup completed")
	}
	if after.currentView != viewReader {
		t.Fatalf("expected reader view, got %v", after.currentView)
	}
	if after.readerBook.ID != resumeBook.ID {
		t.Fatalf("expected resumed book %s, got %s", resumeBook.ID, after.readerBook.ID)
	}
	if after.readerMode != domain.ReadingModeEPUB {
		t.Fatalf("expected epub mode, got %s", after.readerMode)
	}
}

func TestDeleteFromDiskRequiresExactDELETEConfirmation(t *testing.T) {
	container := newTestContainer(t)
	book := seedBookWithEPUBCache(t, container, "delete-me", "")

	if err := os.WriteFile(book.ManagedPath, []byte("content"), 0o644); err != nil {
		t.Fatalf("write managed file: %v", err)
	}

	m := New(container).(model)
	m.startupCompleted = true
	if err := m.refreshLibrary(); err != nil {
		t.Fatalf("refresh library: %v", err)
	}
	if len(m.libraryBooks) != 1 {
		t.Fatalf("expected 1 library book, got %d", len(m.libraryBooks))
	}

	m.handleLibraryKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	if m.prompt == nil || m.prompt.kind != promptDeleteDiskConfirm {
		t.Fatalf("expected delete confirmation prompt")
	}
	if !strings.Contains(m.prompt.description, book.ManagedPath) {
		t.Fatalf("expected prompt to show full file path")
	}

	m.prompt.value = "delete"
	m.applyPrompt()
	m.closePrompt()

	if _, err := os.Stat(book.ManagedPath); err != nil {
		t.Fatalf("expected file to remain after wrong confirmation, stat error: %v", err)
	}
	if !strings.Contains(m.status, "Delete canceled") {
		t.Fatalf("expected canceled status, got %q", m.status)
	}

	m.handleLibraryKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	m.prompt.value = "DELETE"
	m.applyPrompt()
	m.closePrompt()

	if _, err := os.Stat(book.ManagedPath); !os.IsNotExist(err) {
		t.Fatalf("expected managed file deleted, stat err: %v", err)
	}
	if len(m.libraryBooks) != 0 {
		t.Fatalf("expected book removed from library, got %d", len(m.libraryBooks))
	}
}

func TestCtrlCSavesReaderProgressBeforeQuit(t *testing.T) {
	container := newTestContainer(t)
	book := seedBookWithEPUBCache(t, container, "ctrlc-save", "")

	cacheDir := container.Paths.BookCacheDir(book.ID)
	longSection := strings.Repeat("alpha beta gamma delta epsilon zeta eta theta iota kappa lambda mu ", 60)
	cache := domain.EPUBCache{
		Title:    book.Title,
		Author:   book.Author,
		Sections: []string{longSection},
	}
	cacheBytes, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("marshal long epub cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "epub_cache.json"), cacheBytes, 0o644); err != nil {
		t.Fatalf("write long epub cache: %v", err)
	}

	m := New(container).(model)
	m.startupCompleted = true
	m.width = 36
	m.height = 10

	if err := m.openBook(book.ID, nil); err != nil {
		t.Fatalf("open seeded book: %v", err)
	}
	if m.readerPageCount() < 2 {
		t.Fatalf("expected multiple pages, got %d", m.readerPageCount())
	}

	m.readerPage = 1
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	after := updated.(model)
	if cmd == nil {
		t.Fatalf("expected quit command")
	}

	saved, err := container.Library.ReadingStateForMode(context.Background(), book.ID, domain.ReadingModeEPUB)
	if err != nil {
		t.Fatalf("load saved state: %v", err)
	}
	if saved.Locator.PageIndex != after.readerPage {
		t.Fatalf("expected saved page %d, got %d", after.readerPage, saved.Locator.PageIndex)
	}
	if saved.Locator.Offset <= 0 {
		t.Fatalf("expected positive saved offset, got %d", saved.Locator.Offset)
	}
}

func newTestContainer(t *testing.T) *application.Container {
	t.Helper()

	base := t.TempDir()
	paths := storage.Paths{
		BaseDir:    base,
		LibraryDir: filepath.Join(base, "library"),
		CacheDir:   filepath.Join(base, "cache"),
		DBPath:     filepath.Join(base, "zeile.db"),
		ConfigPath: filepath.Join(base, "config.toml"),
	}
	if err := paths.Ensure(); err != nil {
		t.Fatalf("ensure test paths: %v", err)
	}

	db, err := database.Open(paths.DBPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := database.Migrate(context.Background(), db); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}

	bookRepo := repository.NewBookRepository(db)
	stateRepo := repository.NewReadingStateRepository(db)
	registry := preprocessing.NewRegistry()

	return &application.Container{
		Config: config.Config{
			DataDir:            base,
			ManagedCopyDefault: true,
			MinSpreadWidth:     120,
		},
		Paths:   paths,
		DB:      db,
		Library: application.NewLibraryService(bookRepo, stateRepo, registry, paths),
		Reader:  application.NewReaderService(bookRepo, stateRepo, paths),
	}
}

func seedBookWithEPUBCache(t *testing.T, container *application.Container, id string, sourcePath string) domain.Book {
	t.Helper()

	if sourcePath == "" {
		sourcePath = filepath.Join(container.Paths.BaseDir, id+".epub")
	}

	book := domain.Book{
		ID:          id,
		Fingerprint: "fp-" + id,
		Title:       strings.ReplaceAll(id, "-", " "),
		Author:      "Tester",
		Format:      domain.BookFormatEPUB,
		AddedAt:     time.Now().UTC(),
		SourcePath:  sourcePath,
		ManagedPath: filepath.Join(container.Paths.LibraryDir, id+".epub"),
		SizeBytes:   128,
	}

	if _, err := container.DB.ExecContext(
		context.Background(),
		`INSERT INTO books (
			id, fingerprint, title, author, format, added_at, source_path, managed_path, metadata_json, size_bytes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		book.ID,
		book.Fingerprint,
		book.Title,
		book.Author,
		book.Format,
		book.AddedAt.Unix(),
		book.SourcePath,
		book.ManagedPath,
		`{"seed":"true"}`,
		book.SizeBytes,
	); err != nil {
		t.Fatalf("seed book: %v", err)
	}

	cacheDir := container.Paths.BookCacheDir(book.ID)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
	cache := domain.EPUBCache{
		Title:    book.Title,
		Author:   book.Author,
		Sections: []string{"chapter one words", "chapter two words"},
	}
	cacheBytes, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("marshal epub cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "epub_cache.json"), cacheBytes, 0o644); err != nil {
		t.Fatalf("write epub cache: %v", err)
	}

	return book
}
