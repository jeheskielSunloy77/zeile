package tui

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
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

	m.handleLibraryKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if m.remove == nil {
		t.Fatalf("expected remove modal")
	}
	if m.remove.step != removeStepChooseAction {
		t.Fatalf("expected action step")
	}

	m.handleRemoveKey(tea.KeyMsg{Type: tea.KeyDown})
	if m.remove.action != removeActionDeleteDisk {
		t.Fatalf("expected delete-from-disk action selected")
	}
	m.handleRemoveKey(tea.KeyMsg{Type: tea.KeyEnter})
	if m.remove.step != removeStepConfirm {
		t.Fatalf("expected confirmation step")
	}

	m.remove.value = "delete"
	m.applyRemove()
	m.closeRemoveModal()

	if _, err := os.Stat(book.ManagedPath); err != nil {
		t.Fatalf("expected file to remain after wrong confirmation, stat error: %v", err)
	}
	if !strings.Contains(m.statusText, "Delete canceled") {
		t.Fatalf("expected canceled status, got %q", m.statusText)
	}

	m.handleLibraryKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m.handleRemoveKey(tea.KeyMsg{Type: tea.KeyDown})
	m.handleRemoveKey(tea.KeyMsg{Type: tea.KeyEnter})
	m.remove.value = "DELETE"
	m.applyRemove()
	m.closeRemoveModal()

	if _, err := os.Stat(book.ManagedPath); !os.IsNotExist(err) {
		t.Fatalf("expected managed file deleted, stat err: %v", err)
	}
	if len(m.libraryBooks) != 0 {
		t.Fatalf("expected book removed from library, got %d", len(m.libraryBooks))
	}
}

func TestLibraryDeleteShortcutIsDisabled(t *testing.T) {
	container := newTestContainer(t)
	seedBookWithEPUBCache(t, container, "no-d-shortcut", "")

	m := New(container).(model)
	m.startupCompleted = true
	if err := m.refreshLibrary(); err != nil {
		t.Fatalf("refresh library: %v", err)
	}
	if len(m.libraryBooks) == 0 {
		t.Fatalf("expected seeded library book")
	}

	m.handleLibraryKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})

	if m.remove != nil {
		t.Fatalf("expected no remove modal on D key")
	}
	if m.prompt != nil {
		t.Fatalf("expected no prompt on D key")
	}
}

func TestManualSyncRequiresConnection(t *testing.T) {
	container := newTestContainer(t)
	seedBookWithEPUBCache(t, container, "needs-connect", "")

	m := New(container).(model)
	m.startupCompleted = true
	if err := m.refreshLibrary(); err != nil {
		t.Fatalf("refresh library: %v", err)
	}

	_ = m.handleLibraryKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if !strings.Contains(m.statusText, "Connect first to run sync") {
		t.Fatalf("expected connect warning, got %q", m.statusText)
	}
}

func TestAddFlowStartsFileSelectorAtHomeDirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		t.Skip("home directory unavailable")
	}

	container := newTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true

	m.handleLibraryKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.browserDir != filepath.Clean(home) {
		t.Fatalf("expected browserDir %q, got %q", filepath.Clean(home), m.browserDir)
	}

	m.addSourceMethod = addSourceSelector
	m.handleAddKey(tea.KeyMsg{Type: tea.KeyEnter})
	if m.addStep != addStepFileSelector {
		t.Fatalf("expected file selector step, got %v", m.addStep)
	}
	if m.browserDir != filepath.Clean(home) {
		t.Fatalf("expected selector to open at %q, got %q", filepath.Clean(home), m.browserDir)
	}
}

func TestLoadBrowserShowsOnlyDirectoriesAndSupportedFiles(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for _, name := range []string{"book.epub", "paper.PDF", "notes.txt", "archive.zip"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write file %s: %v", name, err)
		}
	}

	m.loadBrowser(dir)

	got := make(map[string]bool)
	for _, entry := range m.browserEntries {
		got[entry.name] = true
	}

	if !got["docs"] {
		t.Fatalf("expected directory entry")
	}
	if !got["book.epub"] {
		t.Fatalf("expected epub entry")
	}
	if !got["paper.PDF"] {
		t.Fatalf("expected pdf entry")
	}
	if got["notes.txt"] {
		t.Fatalf("expected txt file to be filtered out")
	}
	if got["archive.zip"] {
		t.Fatalf("expected zip file to be filtered out")
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

func TestRenderPageBoxHighlightsChapterLinesWithoutChangingTextStructure(t *testing.T) {
	container := newTestContainer(t)
	book := seedBookWithEPUBCache(t, container, "chapter-style", "")

	cacheDir := container.Paths.BookCacheDir(book.ID)
	cache := domain.EPUBCache{
		Title:    book.Title,
		Author:   book.Author,
		Sections: []string{"Chapter One\nalpha beta gamma\ndelta epsilon"},
		SectionChapterLineIndexes: [][]int{
			{0},
		},
	}
	cacheBytes, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("marshal epub cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "epub_cache.json"), cacheBytes, 0o644); err != nil {
		t.Fatalf("write epub cache: %v", err)
	}

	m := New(container).(model)
	m.startupCompleted = true
	m.width = 90
	m.height = 20

	if err := m.openBook(book.ID, nil); err != nil {
		t.Fatalf("open seeded book: %v", err)
	}

	content, lineStarts, lineRanges := m.readerPageContent(0, 60, 10)
	if content == "" {
		t.Fatalf("expected first page content")
	}

	unstyledModel := m
	unstyledModel.readerChapterStarts = nil
	unstyled := unstyledModel.renderPageBox(content, lineStarts, lineRanges, 1, 1, 60, 10)
	styled := m.renderPageBox(content, lineStarts, lineRanges, 1, 1, 60, 10)

	if len(lineStarts) == 0 {
		t.Fatalf("expected line starts for first page")
	}
	if _, ok := m.readerChapterStarts[lineStarts[0]]; !ok {
		t.Fatalf("expected first line to be marked as chapter heading")
	}

	plainStyled := stripANSI(styled)
	plainUnstyled := stripANSI(unstyled)
	if plainStyled != plainUnstyled {
		t.Fatalf("expected identical plain text structure after stripping ANSI\nunstyled:\n%s\nstyled:\n%s", plainUnstyled, plainStyled)
	}
}

func TestEPUBChapterStartsAtTopOfPage(t *testing.T) {
	container := newTestContainer(t)
	book := seedBookWithEPUBCache(t, container, "chapter-top", "")

	cacheDir := container.Paths.BookCacheDir(book.ID)
	cache := domain.EPUBCache{
		Title:  book.Title,
		Author: book.Author,
		Sections: []string{
			"line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nCHAPTER TWO\nafter chapter",
		},
		SectionChapterLineIndexes: [][]int{
			{8},
		},
	}
	cacheBytes, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("marshal epub cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "epub_cache.json"), cacheBytes, 0o644); err != nil {
		t.Fatalf("write epub cache: %v", err)
	}

	m := New(container).(model)
	m.startupCompleted = true
	m.width = 90
	m.height = 13

	if err := m.openBook(book.ID, nil); err != nil {
		t.Fatalf("open seeded book: %v", err)
	}
	if m.readerPageCount() < 2 {
		t.Fatalf("expected at least 2 pages, got %d", m.readerPageCount())
	}

	secondPage := strings.Split(m.readerPagination.Pages[1], "\n")
	if len(secondPage) == 0 || secondPage[0] != "CHAPTER TWO" {
		t.Fatalf("expected chapter on top of page 2, got %q", m.readerPagination.Pages[1])
	}
}

func TestEffectiveStatusHidesReadyAndReading(t *testing.T) {
	m := model{}

	if _, _, visible := m.effectiveStatus(time.Now(), "Ready"); visible {
		t.Fatalf("expected Ready fallback to be hidden")
	}
	if _, _, visible := m.effectiveStatus(time.Now(), "Reading"); visible {
		t.Fatalf("expected Reading fallback to be hidden")
	}
}

func TestRenderLibraryCentersEmptyStateInBody(t *testing.T) {
	m := model{
		currentView: viewLibrary,
		width:       100,
		height:      24,
	}

	rendered := stripANSI(m.renderLibrary())
	lines := strings.Split(rendered, "\n")
	message := "No books yet. Press 'a' to import EPUB/PDF."

	headerLine := -1
	messageLine := -1
	headerIndent := 0
	messageIndent := 0
	for idx, line := range lines {
		if strings.Contains(line, "Zeile") && strings.Contains(line, "Library") && strings.Contains(line, "Communities") && strings.Contains(line, "Settings") {
			headerLine = idx
			headerIndent = leadingSpaces(line)
		}
		if strings.Contains(line, message) {
			messageLine = idx
			messageIndent = leadingSpaces(line)
		}
	}

	if headerLine == -1 {
		t.Fatalf("expected library header in rendered output")
	}
	if messageLine == -1 {
		t.Fatalf("expected empty-state message in rendered output")
	}
	if messageIndent <= headerIndent+10 {
		t.Fatalf("expected centered empty state indentation > header indentation + 10, got header=%d message=%d", headerIndent, messageIndent)
	}

	minCenter := m.height / 3
	maxCenter := (2 * m.height) / 3
	if messageLine < minCenter || messageLine > maxCenter {
		t.Fatalf("expected empty-state line in vertical center band [%d,%d], got %d", minCenter, maxCenter, messageLine)
	}
}

func TestRenderLibraryRowsStayLeftAlignedWhenBooksExist(t *testing.T) {
	m := model{
		currentView: viewLibrary,
		width:       100,
		height:      24,
		libraryBooks: []domain.Book{
			{
				ID:     "book-1",
				Title:  "Book One",
				Author: "Author",
			},
		},
		libraryProgress: map[string]float64{},
		libraryFinished: map[string]bool{},
	}

	rendered := stripANSI(m.renderLibrary())
	rowText := "Book One - Author | 0.0% | Last opened: -"
	rowLine := -1
	rowIndent := 0
	for idx, line := range strings.Split(rendered, "\n") {
		if strings.Contains(line, rowText) {
			rowLine = idx
			rowIndent = leadingSpaces(line)
			break
		}
	}

	if rowLine == -1 {
		t.Fatalf("expected library row in rendered output")
	}
	if rowIndent > 12 {
		t.Fatalf("expected non-empty rows to remain left-aligned, got indentation %d", rowIndent)
	}
}

func TestEffectiveStatusAutoDismissesAfterTenSeconds(t *testing.T) {
	m := model{}
	m.setStatusSuccess("Imported: Book")

	soon := m.statusSetAt.Add(9 * time.Second)
	if _, _, visible := m.effectiveStatus(soon, "Ready"); !visible {
		t.Fatalf("expected status to still be visible before timeout")
	}

	late := m.statusSetAt.Add(10 * time.Second)
	if _, _, visible := m.effectiveStatus(late, "Ready"); visible {
		t.Fatalf("expected status to hide at timeout")
	}
}

func TestRenderStatusToastUsesVariantStyle(t *testing.T) {
	m := model{}

	m.setStatusSuccess("Saved")
	success := m.renderStatusToast("Ready")
	if !strings.Contains(success, "Saved") {
		t.Fatalf("expected success toast text, got %q", success)
	}
	if _, kind, visible := m.effectiveStatus(time.Now(), "Ready"); !visible || kind != statusSuccess {
		t.Fatalf("expected visible success status, got visible=%v kind=%q", visible, kind)
	}

	m.setStatusDestructive("Failed")
	destructive := m.renderStatusToast("Ready")
	if !strings.Contains(destructive, "Failed") {
		t.Fatalf("expected destructive toast text, got %q", destructive)
	}
	if _, kind, visible := m.effectiveStatus(time.Now(), "Ready"); !visible || kind != statusDestructive {
		t.Fatalf("expected visible destructive status, got visible=%v kind=%q", visible, kind)
	}
}

func TestLibrarySettingsKeyOpensSettingsView(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true
	m.currentView = viewLibrary

	_ = m.handleLibraryKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})

	if m.currentView != viewSettings {
		t.Fatalf("expected settings view, got %v", m.currentView)
	}
	if m.settingsReturnView != viewLibrary {
		t.Fatalf("expected return view library, got %v", m.settingsReturnView)
	}

	_ = m.handleSettingsKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.currentView != viewLibrary {
		t.Fatalf("expected return to library, got %v", m.currentView)
	}
}

func TestMainNavTabCyclesViews(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true
	m.currentView = viewLibrary

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(model)
	if m.currentView != viewCommunities {
		t.Fatalf("expected communities view after first Tab, got %v", m.currentView)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(model)
	if m.currentView != viewSettings {
		t.Fatalf("expected settings view after second Tab, got %v", m.currentView)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(model)
	if m.currentView != viewLibrary {
		t.Fatalf("expected library view after third Tab, got %v", m.currentView)
	}
}

func TestMainNavShiftTabCyclesViewsBackward(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true
	m.currentView = viewLibrary

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(model)
	if m.currentView != viewSettings {
		t.Fatalf("expected settings view after Shift+Tab from library, got %v", m.currentView)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(model)
	if m.currentView != viewCommunities {
		t.Fatalf("expected communities view after second Shift+Tab, got %v", m.currentView)
	}
}

func TestReaderSettingsKeyOpensSettingsView(t *testing.T) {
	container := newTestContainer(t)
	book := seedBookWithEPUBCache(t, container, "reader-settings", "")

	m := New(container).(model)
	m.startupCompleted = true
	m.width = 90
	m.height = 20

	if err := m.openBook(book.ID, nil); err != nil {
		t.Fatalf("open seeded book: %v", err)
	}

	m.handleReaderKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if m.currentView != viewSettings {
		t.Fatalf("expected settings view, got %v", m.currentView)
	}
	if m.settingsReturnView != viewReader {
		t.Fatalf("expected return view reader, got %v", m.settingsReturnView)
	}

	_ = m.handleSettingsKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.currentView != viewReader {
		t.Fatalf("expected return to reader, got %v", m.currentView)
	}
}

func TestSettingsTabSwitchesMainView(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true
	m.currentView = viewLibrary
	m.openSettings(viewLibrary)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(model)
	if m.currentView != viewLibrary {
		t.Fatalf("expected Tab from settings to switch to library view, got %v", m.currentView)
	}
}

func TestSettingsSectionSwitchUsesBrackets(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true
	m.openSettings(viewLibrary)

	if m.settingsSection != settingsSectionTheme {
		t.Fatalf("expected initial settings section theme, got %v", m.settingsSection)
	}

	_ = m.handleSettingsKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	if m.settingsSection != settingsSectionReading {
		t.Fatalf("expected next section after ], got %v", m.settingsSection)
	}

	_ = m.handleSettingsKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	if m.settingsSection != settingsSectionTheme {
		t.Fatalf("expected previous section after [, got %v", m.settingsSection)
	}
}

func TestRenderCommunitiesShowsPlaceholder(t *testing.T) {
	m := model{
		currentView: viewCommunities,
		width:       100,
		height:      24,
	}

	rendered := stripANSI(m.renderCommunities())
	if !strings.Contains(rendered, "Communities - Coming soon.") {
		t.Fatalf("expected communities placeholder, got %q", rendered)
	}
}

func TestCommunitiesSettingsReturnToCommunities(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true
	m.currentView = viewCommunities

	_ = m.handleCommunitiesKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if m.currentView != viewSettings {
		t.Fatalf("expected settings view, got %v", m.currentView)
	}
	if m.settingsReturnView != viewCommunities {
		t.Fatalf("expected return view communities, got %v", m.settingsReturnView)
	}

	_ = m.handleSettingsKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.currentView != viewCommunities {
		t.Fatalf("expected return to communities, got %v", m.currentView)
	}
}

func TestStartupModeLibrarySkipsAutoResume(t *testing.T) {
	container := newTestContainer(t)
	container.Config.StartupMode = config.StartupModeLibrary

	book := seedBookWithEPUBCache(t, container, "startup-library", "")
	if err := container.Library.UpdateReadingState(context.Background(), domain.ReadingState{
		BookID:          book.ID,
		Mode:            domain.ReadingModeEPUB,
		Locator:         domain.Locator{Offset: 10},
		ProgressPercent: 20,
		UpdatedAt:       time.Now().UTC(),
		IsFinished:      false,
	}); err != nil {
		t.Fatalf("seed reading state: %v", err)
	}

	m := New(container).(model)
	startupMsg, ok := m.loadStartupCmd()().(startupLoadedMsg)
	if !ok {
		t.Fatalf("expected startupLoadedMsg")
	}

	updated, _ := m.Update(startupMsg)
	after := updated.(model)

	if after.currentView != viewLibrary {
		t.Fatalf("expected library view, got %v", after.currentView)
	}
	if after.readerBook.ID != "" {
		t.Fatalf("expected no auto-resumed reader book, got %s", after.readerBook.ID)
	}
}

func TestDeleteConfirmationDisabledSkipsConfirmationStep(t *testing.T) {
	container := newTestContainer(t)
	seedBookWithEPUBCache(t, container, "no-confirm-remove", "")

	m := New(container).(model)
	m.startupCompleted = true
	if err := m.refreshLibrary(); err != nil {
		t.Fatalf("refresh library: %v", err)
	}

	cfg := m.currentConfig()
	cfg.DeleteConfirmation = false
	m.applyConfig(cfg)

	_ = m.handleLibraryKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if m.remove == nil {
		t.Fatalf("expected remove modal")
	}

	m.handleRemoveKey(tea.KeyMsg{Type: tea.KeyEnter})

	if m.remove != nil {
		t.Fatalf("expected remove modal to close after direct apply")
	}
	if len(m.libraryBooks) != 0 {
		t.Fatalf("expected library removal without confirmation, got %d books", len(m.libraryBooks))
	}
}

func TestRenderFooterHintsRespectsDensity(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)

	hints := []footerHint{
		{key: "1", action: "one"},
		{key: "2", action: "two"},
		{key: "3", action: "three"},
		{key: "4", action: "four"},
		{key: "5", action: "five"},
	}

	cfg := m.currentConfig()
	cfg.KeyHintsDensity = config.KeyHintsDensityHidden
	m.applyConfig(cfg)
	if got := stripANSI(m.renderFooterHints(hints)); got != "" {
		t.Fatalf("expected hidden hints, got %q", got)
	}

	cfg.KeyHintsDensity = config.KeyHintsDensityCompact
	m.applyConfig(cfg)
	got := stripANSI(m.renderFooterHints(hints))
	if strings.Contains(got, "five") {
		t.Fatalf("expected compact hints to truncate extra actions, got %q", got)
	}
	if !strings.Contains(got, "four") {
		t.Fatalf("expected compact hints to include first actions, got %q", got)
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
	defaultCfg := config.Default()
	defaultCfg.DataDir = base
	defaultCfg.ManagedCopyDefault = true
	defaultCfg.MinSpreadWidth = 120
	defaultCfg.SpreadThreshold = 120

	return &application.Container{
		Config:  defaultCfg.Normalized(),
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
		SectionChapterLineIndexes: [][]int{
			{0},
			{0},
		},
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

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(value string) string {
	return ansiPattern.ReplaceAllString(value, "")
}

func leadingSpaces(value string) int {
	count := 0
	for _, r := range value {
		if r != ' ' {
			break
		}
		count++
	}
	return count
}
