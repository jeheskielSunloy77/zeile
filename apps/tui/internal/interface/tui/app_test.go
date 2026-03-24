package tui

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jeheskielSunloy77/kern/tui/internal/application"
	"github.com/jeheskielSunloy77/kern/tui/internal/domain"
	"github.com/jeheskielSunloy77/kern/tui/internal/infrastructure/config"
	"github.com/jeheskielSunloy77/kern/tui/internal/infrastructure/database"
	"github.com/jeheskielSunloy77/kern/tui/internal/infrastructure/remote"
	"github.com/jeheskielSunloy77/kern/tui/internal/infrastructure/repository"
	"github.com/jeheskielSunloy77/kern/tui/internal/infrastructure/storage"
	"github.com/jeheskielSunloy77/kern/tui/internal/preprocessing"
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

func TestAccountViewShowsLoginWhenDisconnected(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true
	m.currentView = viewAccount

	rendered := stripANSI(m.renderAccount())
	if !strings.Contains(rendered, "Login") {
		t.Fatalf("expected login label in account view, got %q", rendered)
	}
	if strings.Contains(rendered, "Manual sync") || strings.Contains(rendered, "Logout") {
		t.Fatalf("expected authenticated actions to be hidden when disconnected, got %q", rendered)
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
	for _, name := range []string{"book.epub", "notes.txt", "archive.zip"} {
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
	message := "No books yet. Press 'a' to import EPUB."

	headerLine := -1
	messageLine := -1
	headerIndent := 0
	messageIndent := 0
	for idx, line := range lines {
		if strings.Contains(line, "Kern") && strings.Contains(line, "Library") && strings.Contains(line, "Communities") && strings.Contains(line, "Settings") && strings.Contains(line, "Login") {
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

func TestRenderLibraryRowsCenterWhenBooksExist(t *testing.T) {
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
	rowText := "Book One - Author | 0.0% | Cloud: Local only | Last opened: -"
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
	if rowIndent < 20 {
		t.Fatalf("expected non-empty rows to be centered, got indentation %d", rowIndent)
	}
	minCenter := m.height / 3
	maxCenter := (2 * m.height) / 3
	if rowLine < minCenter || rowLine > maxCenter {
		t.Fatalf("expected library row line in vertical center band [%d,%d], got %d", minCenter, maxCenter, rowLine)
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

func TestLibrarySettingsKeyDoesNothing(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true
	m.currentView = viewLibrary

	_ = m.handleLibraryKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})

	if m.currentView != viewLibrary {
		t.Fatalf("expected to stay in library, got %v", m.currentView)
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
	if m.currentView != viewAccount {
		t.Fatalf("expected account view after third Tab, got %v", m.currentView)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(model)
	if m.currentView != viewLibrary {
		t.Fatalf("expected library view after fourth Tab, got %v", m.currentView)
	}
}

func TestMainNavShiftTabCyclesViewsBackward(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true
	m.currentView = viewLibrary

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(model)
	if m.currentView != viewAccount {
		t.Fatalf("expected account view after Shift+Tab from library, got %v", m.currentView)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(model)
	if m.currentView != viewSettings {
		t.Fatalf("expected settings view after second Shift+Tab, got %v", m.currentView)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(model)
	if m.currentView != viewCommunities {
		t.Fatalf("expected communities view after third Shift+Tab, got %v", m.currentView)
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
	if m.currentView != viewAccount {
		t.Fatalf("expected Tab from settings to switch to account view, got %v", m.currentView)
	}
}

func TestAccountViewShowsAuthenticatedActions(t *testing.T) {
	m := model{
		currentView:      viewAccount,
		connectionLabel:  "Connected: @tester",
		startupCompleted: true,
	}

	rendered := stripANSI(m.renderAccount())
	if !strings.Contains(rendered, "Account") {
		t.Fatalf("expected account label in account view, got %q", rendered)
	}
	if strings.Contains(rendered, "Login") {
		t.Fatalf("expected login action hidden when connected, got %q", rendered)
	}
	if !strings.Contains(rendered, "Edit profile") || !strings.Contains(rendered, "Manual sync") || !strings.Contains(rendered, "Logout") {
		t.Fatalf("expected edit profile, manual sync, and logout actions, got %q", rendered)
	}
}

func TestAccountEnterStartsLoginFlowWhenDisconnected(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true
	m.currentView = viewAccount

	cmd := m.handleAccountKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected login command")
	}

	msg := cmd()
	if _, ok := msg.(deviceAuthStartMsg); !ok {
		t.Fatalf("expected deviceAuthStartMsg, got %T", msg)
	}
}

func TestAccountManualSyncShowsConnectWarningWhenUnavailable(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true
	m.currentView = viewAccount
	m.connectionLabel = "Connected: @tester"
	m.accountField = 1

	_ = m.handleAccountKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !strings.Contains(m.statusText, "Connect first to run sync") {
		t.Fatalf("expected connect warning, got %q", m.statusText)
	}
}

func TestAccountEnterOnLogoutRunsDisconnect(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true
	m.currentView = viewAccount
	m.connectionLabel = "Connected: @tester"
	m.accountField = 2

	cmd := m.handleAccountKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected logout command")
	}

	msg := cmd()
	if _, ok := msg.(authDisconnectedMsg); !ok {
		t.Fatalf("expected authDisconnectedMsg, got %T", msg)
	}
}

func TestAccountEnterOnEditProfileOpensModal(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true
	m.currentView = viewAccount
	m.connectionLabel = "Connected: @tester"
	m.accountField = 0

	cmd := m.handleAccountKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("expected no async command for opening profile editor")
	}
	if m.profileEditor == nil {
		t.Fatalf("expected profile editor modal to open")
	}
}

func TestImportDoneTriggersSyncWhenConnectedAndIdle(t *testing.T) {
	container := newConnectedSyncTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true

	updated, cmd := m.Update(importDoneMsg{book: domain.Book{Title: "Auto Sync"}})
	after := updated.(model)

	if !after.syncing {
		t.Fatalf("expected syncing state true after successful import while connected")
	}
	if cmd == nil {
		t.Fatalf("expected sync command after successful import while connected")
	}
}

func TestImportDoneDoesNotTriggerSyncWhenDisconnected(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true

	updated, cmd := m.Update(importDoneMsg{book: domain.Book{Title: "No Sync"}})
	after := updated.(model)

	if after.syncing {
		t.Fatalf("expected syncing state false while disconnected")
	}
	if cmd != nil {
		t.Fatalf("expected no sync command while disconnected")
	}
}

func TestImportDoneDoesNotEnqueueSyncWhenAlreadySyncing(t *testing.T) {
	container := newConnectedSyncTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true
	m.syncing = true

	updated, cmd := m.Update(importDoneMsg{book: domain.Book{Title: "Already Syncing"}})
	after := updated.(model)

	if !after.syncing {
		t.Fatalf("expected syncing state to remain true")
	}
	if cmd != nil {
		t.Fatalf("expected no new sync command when sync is already in progress")
	}
}

func TestProfileEditorEscClosesModal(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true
	m.currentView = viewAccount
	m.profileEditor = &profileEditorState{Username: "tester"}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(model)
	if m.profileEditor != nil {
		t.Fatalf("expected profile editor modal to close")
	}
}

func TestProfileEditorRejectsInvalidUsername(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true
	m.currentView = viewAccount
	m.profileEditor = &profileEditorState{Username: "ab"}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if cmd != nil {
		t.Fatalf("expected no save command for invalid username")
	}
	if m.profileEditor == nil {
		t.Fatalf("expected profile editor to remain open")
	}
	if !strings.Contains(m.statusText, "Username must be 3-50 characters") {
		t.Fatalf("expected validation status, got %q", m.statusText)
	}
}

func TestProfileUsernameUpdatedSuccessClosesModal(t *testing.T) {
	m := model{
		currentView:      viewAccount,
		connectionLabel:  "Connected: @old",
		startupCompleted: true,
		profileEditor: &profileEditorState{
			Username: "old",
			Saving:   true,
		},
	}

	updated, _ := m.Update(profileUsernameUpdatedMsg{username: "newname"})
	m = updated.(model)

	if m.profileEditor != nil {
		t.Fatalf("expected profile editor closed after success")
	}
	if !strings.Contains(m.connectionLabel, "@newname") {
		t.Fatalf("expected connection label to include updated username, got %q", m.connectionLabel)
	}
	if !strings.Contains(m.statusText, "Username updated") {
		t.Fatalf("expected success status, got %q", m.statusText)
	}
}

func TestProfileUsernameUpdatedFailureKeepsModalOpen(t *testing.T) {
	m := model{
		currentView:      viewAccount,
		connectionLabel:  "Connected: @old",
		startupCompleted: true,
		profileEditor: &profileEditorState{
			Username: "old",
			Saving:   true,
		},
	}

	updated, _ := m.Update(profileUsernameUpdatedMsg{err: errors.New("boom")})
	m = updated.(model)

	if m.profileEditor == nil {
		t.Fatalf("expected profile editor to remain open after failure")
	}
	if m.profileEditor.Saving {
		t.Fatalf("expected saving state cleared after failure")
	}
	if !strings.Contains(m.statusText, "Profile update failed") {
		t.Fatalf("expected failure status, got %q", m.statusText)
	}
}

func TestSettingsCloseReturnsToAccount(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true
	m.openSettings(viewAccount)

	_ = m.handleSettingsKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.currentView != viewAccount {
		t.Fatalf("expected return to account, got %v", m.currentView)
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

func TestRenderCommunitiesShowsDisconnectedState(t *testing.T) {
	m := model{
		currentView: viewCommunities,
		width:       100,
		height:      24,
	}

	rendered := stripANSI(m.renderCommunities())
	if !strings.Contains(rendered, "Connect your account to browse community books.") {
		t.Fatalf("expected disconnected communities state, got %q", rendered)
	}
}

func TestCommunitiesSearchOpensPrompt(t *testing.T) {
	container := newTestContainer(t)
	m := New(container).(model)
	m.startupCompleted = true
	m.currentView = viewCommunities

	_ = m.handleCommunitiesKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	if m.prompt == nil {
		t.Fatalf("expected community search prompt")
	}
	if m.prompt.kind != promptCommunitySearch {
		t.Fatalf("expected community search prompt kind, got %v", m.prompt.kind)
	}
}

func TestLibraryVisibilityKeyShowsDisconnectedStatus(t *testing.T) {
	container := newTestContainer(t)
	seedBookWithEPUBCache(t, container, "visibility-disconnected", "")

	m := New(container).(model)
	m.startupCompleted = true
	if err := m.refreshLibrary(); err != nil {
		t.Fatalf("refresh library: %v", err)
	}

	_ = m.handleLibraryKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	if !strings.Contains(m.statusText, "Connect first") {
		t.Fatalf("expected connect-first status, got %q", m.statusText)
	}
	if m.visibilityConfirm != nil {
		t.Fatalf("expected no visibility confirm while disconnected")
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
		DBPath:     filepath.Join(base, "kern.db"),
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

func newConnectedSyncTestContainer(t *testing.T) *application.Container {
	t.Helper()

	container := newTestContainer(t)
	session := remote.Session{
		User: remote.User{
			ID:       "user-1",
			Email:    "tester@example.com",
			Username: "tester",
		},
		AccessToken:      "access-token",
		AccessExpiresAt:  time.Now().UTC().Add(1 * time.Hour),
		RefreshToken:     "refresh-token",
		RefreshExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}
	payload, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("marshal test session: %v", err)
	}
	sessionPath := filepath.Join(container.Paths.BaseDir, "auth-session.json")
	if err := os.WriteFile(sessionPath, payload, 0o600); err != nil {
		t.Fatalf("write test session: %v", err)
	}

	auth, err := application.NewAuthService(container.Config, container.Paths)
	if err != nil {
		t.Fatalf("create auth service: %v", err)
	}
	syncRepo := repository.NewSyncRepository(container.DB)
	syncService := application.NewSyncService(
		auth,
		container.Library,
		syncRepo,
		syncRepo,
		noopSyncRemoteClient{},
	)

	container.Auth = auth
	container.Sync = syncService
	return container
}

type noopSyncRemoteClient struct{}

func (noopSyncRemoteClient) CreateCatalogBook(_ context.Context, _, _, _ string) (remote.BookCatalog, error) {
	return remote.BookCatalog{ID: "catalog-book"}, nil
}

func (noopSyncRemoteClient) UpsertLibraryBook(_ context.Context, _, catalogBookID string) (remote.UserLibraryBook, error) {
	return remote.UserLibraryBook{ID: "library-book", CatalogBookID: catalogBookID}, nil
}

func (noopSyncRemoteClient) ListLibraryBooks(_ context.Context, _ string) ([]remote.UserLibraryBook, error) {
	return []remote.UserLibraryBook{}, nil
}

func (noopSyncRemoteClient) UploadBookAsset(_ context.Context, _, catalogBookID, _ string) (remote.BookAsset, error) {
	return remote.BookAsset{ID: "asset", CatalogBookID: catalogBookID}, nil
}

func (noopSyncRemoteClient) UpdateLibraryBookPreferredAsset(_ context.Context, _, libraryBookID, _ string) (remote.UserLibraryBook, error) {
	return remote.UserLibraryBook{ID: libraryBookID, CatalogBookID: "catalog-book"}, nil
}

func (noopSyncRemoteClient) UpsertReadingState(_ context.Context, _, _, _ string, _ map[string]any, _ float64) error {
	return nil
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
