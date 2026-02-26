package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zeile/tui/internal/application"
	"github.com/zeile/tui/internal/domain"
	"github.com/zeile/tui/internal/infrastructure/config"
	"github.com/zeile/tui/internal/infrastructure/repository"
	"github.com/zeile/tui/internal/reader"
)

type viewID int

const (
	viewLibrary viewID = iota
	viewCommunities
	viewAdd
	viewReader
	viewSettings
)

type addStep int

const (
	addStepChooseSource addStep = iota
	addStepPathInput
	addStepFileSelector
	addStepManagedCopy
	addStepImporting
)

type addSourceMethod int

const (
	addSourcePath addSourceMethod = iota
	addSourceSelector
)

type promptKind int

const (
	promptNone promptKind = iota
	promptLibrarySearch
	promptReaderSearch
	promptGotoPage
	promptGotoPercent
)

type promptState struct {
	kind        promptKind
	title       string
	description string
	value       string
	placeholder string
}

type removeAction int

const (
	removeActionLibrary removeAction = iota
	removeActionDeleteDisk
)

type removeStep int

const (
	removeStepChooseAction removeStep = iota
	removeStepConfirm
)

type removeState struct {
	bookID      string
	bookTitle   string
	managedPath string
	step        removeStep
	action      removeAction
	value       string
}

type browserEntry struct {
	name  string
	path  string
	isDir bool
}

type importProgressMsg struct {
	stage   string
	percent float64
}

type importDoneMsg struct {
	book domain.Book
	err  error
}

type importChannelClosedMsg struct{}

type startupLoadedMsg struct {
	books        []domain.Book
	resumeBookID string
	err          error
}

type statusVariant string

const (
	statusDefault     statusVariant = "default"
	statusSuccess     statusVariant = "success"
	statusDestructive statusVariant = "destructive"
)

type settingsSaveMsg struct {
	sequence int
}

type model struct {
	container *application.Container

	width  int
	height int

	currentView viewID
	statusText  string
	statusKind  statusVariant
	statusSetAt time.Time

	connectionLabel string
	deviceAuth      *deviceAuthState
	syncing         bool
	syncInterval    time.Duration

	prompt *promptState

	libraryBooks     []domain.Book
	librarySelected  int
	libraryQuery     string
	libraryProgress  map[string]float64
	libraryFinished  map[string]bool
	startupCompleted bool

	addPath         string
	addManagedCopy  bool
	addStep         addStep
	addSourceMethod addSourceMethod
	browserDir      string
	browserEntries  []browserEntry
	browserSelected int
	importing       bool
	importStage     string
	importPercent   float64
	importCancel    context.CancelFunc
	importEvents    <-chan tea.Msg

	remove *removeState

	readerBook          domain.Book
	readerMode          domain.ReadingMode
	readerTextDocument  reader.TextDocument
	readerPagination    reader.TextPagination
	readerSectionStarts []int
	readerChapterStarts map[int]struct{}
	readerLayoutPages   []string
	readerPage          int
	readerSearchQuery   string
	readerSearchMatches []int
	readerSearchIndex   int
	readerZen           bool
	readerHelp          bool
	readerFinished      bool

	settingsReturnView viewID
	settingsSection    settingsSectionID
	settingsField      int
	settingsSaveSeq    int
}

func New(container *application.Container) tea.Model {
	m := model{
		container:       container,
		currentView:     viewLibrary,
		addManagedCopy:  true,
		addStep:         addStepChooseSource,
		addSourceMethod: addSourcePath,
		libraryProgress: map[string]float64{},
		libraryFinished: map[string]bool{},
		connectionLabel: "Local-only",
		syncInterval:    2 * time.Minute,
	}

	if container != nil {
		container.Config = container.Config.Normalized()
		m.addManagedCopy = container.Config.ManagedCopyDefault
		if container.Auth != nil {
			m.connectionLabel = container.Auth.ConnectionLabel()
		}
	}

	cwd, err := os.Getwd()
	if err == nil {
		m.loadBrowser(filepath.Clean(cwd))
	}

	return m
}

func (m model) Init() tea.Cmd {
	return m.loadStartupCmd()
}

func (m model) loadStartupCmd() tea.Cmd {
	container := m.container
	return func() tea.Msg {
		if container == nil || container.Library == nil {
			return startupLoadedMsg{err: errors.New("app container is not initialized")}
		}

		ctx := context.Background()
		books, err := container.Library.ListBooks(ctx)
		if err != nil {
			return startupLoadedMsg{err: err}
		}

		resumeBookID := ""
		startupMode := container.Config.StartupMode
		if startupMode == "" {
			startupMode = config.StartupModeResume
		}
		if startupMode == config.StartupModeResume {
			book, err := container.Library.MostRecentUnfinishedBook(ctx)
			if err == nil {
				resumeBookID = book.ID
			} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
				return startupLoadedMsg{books: books, err: err}
			}
		}

		return startupLoadedMsg{books: books, resumeBookID: resumeBookID}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.currentView == viewReader && m.isReaderTextMode() {
			anchor := m.readerAnchorOffset()
			m.repaginateReader(anchor)
		}
		return m, nil

	case startupLoadedMsg:
		if msg.err != nil {
			m.setStatusDestructive(fmt.Sprintf("Startup error: %v", msg.err))
			m.startupCompleted = true
			return m, nil
		}

		m.libraryBooks = msg.books
		m.refreshProgressSummary()
		m.startupCompleted = true
		if msg.resumeBookID != "" {
			if err := m.openBook(msg.resumeBookID, nil); err != nil {
				m.setStatusDestructive(fmt.Sprintf("Failed to auto-resume: %v", err))
			}
		}
		if m.shouldRunSync() {
			return m, waitSyncTick(m.syncInterval)
		}
		return m, nil

	case importProgressMsg:
		m.importStage = msg.stage
		m.importPercent = msg.percent
		if m.importEvents != nil {
			return m, waitForImportEvent(m.importEvents)
		}
		return m, nil

	case importDoneMsg:
		m.importing = false
		m.importCancel = nil
		m.importEvents = nil
		m.importStage = ""
		m.importPercent = 0
		if msg.err != nil {
			m.addStep = addStepManagedCopy
			if errors.Is(msg.err, context.Canceled) {
				m.setStatusDefault("Import canceled")
			} else {
				m.setStatusDestructive(fmt.Sprintf("Import failed: %v", msg.err))
			}
			return m, nil
		}

		m.setStatusSuccess(fmt.Sprintf("Imported: %s", msg.book.Title))
		m.addPath = ""
		m.addStep = addStepChooseSource
		if err := m.refreshLibrary(); err != nil {
			m.setStatusDestructive(fmt.Sprintf("Imported, but failed to reload library: %v", err))
		} else {
			m.currentView = viewLibrary
		}
		return m, nil

	case importChannelClosedMsg:
		m.importEvents = nil
		if m.importing {
			m.importing = false
			m.importCancel = nil
			m.addStep = addStepManagedCopy
		}
		return m, nil

	case settingsSaveMsg:
		if msg.sequence != m.settingsSaveSeq {
			return m, nil
		}
		if err := m.persistSettingsConfig(); err != nil {
			m.setStatusDestructive(fmt.Sprintf("Failed to save settings: %v", err))
		}
		return m, nil

	case syncTickMsg:
		if !m.shouldRunSync() {
			return m, nil
		}
		if m.syncing {
			return m, waitSyncTick(m.syncInterval)
		}
		m.syncing = true
		return m, tea.Batch(
			m.syncNowCmd(false),
			waitSyncTick(m.syncInterval),
		)

	case syncDoneMsg:
		m.syncing = false
		if msg.err != nil {
			if msg.triggeredByUser {
				m.setStatusDestructive(fmt.Sprintf("Sync failed: %v", msg.err))
			}
			return m, nil
		}

		if msg.triggeredByUser {
			m.setStatusSuccess(fmt.Sprintf(
				"Synced %d books, %d states (%d already linked)",
				msg.result.SyncedBooks,
				msg.result.SyncedStates,
				msg.result.SkippedBooks,
			))
		}
		return m, nil

	case deviceAuthStartMsg:
		if msg.err != nil {
			m.deviceAuth = nil
			m.setStatusDestructive(fmt.Sprintf("Connect failed: %v", msg.err))
			return m, nil
		}
		interval := time.Duration(msg.start.IntervalSeconds) * time.Second
		if interval <= 0 {
			interval = 5 * time.Second
		}
		m.deviceAuth = &deviceAuthState{
			DeviceCode:      msg.start.DeviceCode,
			UserCode:        msg.start.UserCode,
			VerificationURI: msg.start.VerificationURI,
			ExpiresAt:       msg.start.ExpiresAt,
			Interval:        interval,
		}
		m.setStatusDefault("Waiting for browser approval")
		return m, waitDeviceAuthPoll(interval)

	case deviceAuthPollTickMsg:
		if m.deviceAuth == nil {
			return m, nil
		}
		return m, m.pollDeviceAuthCmd()

	case deviceAuthPollMsg:
		if m.deviceAuth == nil {
			return m, nil
		}
		if msg.err != nil {
			m.deviceAuth = nil
			m.setStatusDestructive(fmt.Sprintf("Device auth failed: %v", msg.err))
			return m, nil
		}
		if msg.result.Status == "approved" {
			m.deviceAuth = nil
			if m.container != nil && m.container.Auth != nil {
				m.connectionLabel = m.container.Auth.ConnectionLabel()
			} else {
				m.connectionLabel = "Connected"
			}
			m.setStatusSuccess("Connected successfully")
			if m.shouldRunSync() {
				m.syncing = true
				return m, tea.Batch(
					m.syncNowCmd(true),
					waitSyncTick(m.syncInterval),
				)
			}
			return m, nil
		}

		if time.Now().UTC().After(m.deviceAuth.ExpiresAt) {
			m.deviceAuth = nil
			m.setStatusDestructive("Device code expired")
			return m, nil
		}
		return m, waitDeviceAuthPoll(m.deviceAuth.Interval)

	case authDisconnectedMsg:
		if msg.err != nil {
			m.setStatusDestructive(fmt.Sprintf("Disconnect failed: %v", msg.err))
			return m, nil
		}
		m.connectionLabel = "Local-only"
		m.deviceAuth = nil
		m.syncing = false
		m.setStatusSuccess("Disconnected")
		return m, nil

	case tea.QuitMsg:
		if m.currentView == viewReader {
			if err := m.saveReaderState(); err != nil {
				m.setStatusDestructive(fmt.Sprintf("Failed to save progress before quit: %v", err))
				return m, nil
			}
		}
		return m, tea.Quit

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, m.requestQuitCmd()
		}

		if m.deviceAuth != nil {
			switch msg.String() {
			case "esc", "q":
				m.deviceAuth = nil
				m.setStatusDefault("Device auth canceled")
			}
			return m, nil
		}

		if m.remove != nil {
			m.handleRemoveKey(msg)
			return m, nil
		}

		if m.prompt != nil {
			m.handlePromptKey(msg)
			return m, nil
		}

		if m.isMainNavView(m.currentView) {
			switch msg.String() {
			case "tab":
				m.stepMainView(1)
				return m, nil
			case "shift+tab", "backtab":
				m.stepMainView(-1)
				return m, nil
			}
		}

		switch m.currentView {
		case viewLibrary:
			cmd := m.handleLibraryKey(msg)
			return m, cmd
		case viewCommunities:
			cmd := m.handleCommunitiesKey(msg)
			return m, cmd
		case viewAdd:
			cmd := m.handleAddKey(msg)
			return m, cmd
		case viewReader:
			m.handleReaderKey(msg)
		case viewSettings:
			return m, m.handleSettingsKey(msg)
		}
	}

	return m, nil
}

func waitForImportEvent(events <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-events
		if !ok {
			return importChannelClosedMsg{}
		}
		return msg
	}
}

func (m *model) refreshLibrary() error {
	if m.container == nil || m.container.Library == nil {
		return errors.New("library service unavailable")
	}

	ctx := context.Background()
	var (
		books []domain.Book
		err   error
	)
	if strings.TrimSpace(m.libraryQuery) == "" {
		books, err = m.container.Library.ListBooks(ctx)
	} else {
		books, err = m.container.Library.SearchBooks(ctx, m.libraryQuery)
	}
	if err != nil {
		return err
	}

	m.libraryBooks = books
	if len(m.libraryBooks) == 0 {
		m.librarySelected = 0
	} else if m.librarySelected >= len(m.libraryBooks) {
		m.librarySelected = len(m.libraryBooks) - 1
	}

	m.refreshProgressSummary()
	return nil
}

func (m *model) refreshProgressSummary() {
	m.libraryProgress = map[string]float64{}
	m.libraryFinished = map[string]bool{}

	if m.container == nil || m.container.Library == nil {
		return
	}

	ctx := context.Background()
	for _, book := range m.libraryBooks {
		states, err := m.container.Library.StatesForBook(ctx, book.ID)
		if err != nil || len(states) == 0 {
			continue
		}

		sort.SliceStable(states, func(i, j int) bool {
			return states[i].UpdatedAt.After(states[j].UpdatedAt)
		})
		latest := states[0]
		m.libraryProgress[book.ID] = latest.ProgressPercent

		allFinished := true
		for _, state := range states {
			if !state.IsFinished {
				allFinished = false
				break
			}
		}
		m.libraryFinished[book.ID] = allFinished
	}
}

func (m *model) selectedBook() (domain.Book, bool) {
	if len(m.libraryBooks) == 0 {
		return domain.Book{}, false
	}
	if m.librarySelected < 0 || m.librarySelected >= len(m.libraryBooks) {
		return domain.Book{}, false
	}
	return m.libraryBooks[m.librarySelected], true
}

func (m *model) promptFor(kind promptKind, title, description, placeholder, initial string) {
	m.prompt = &promptState{
		kind:        kind,
		title:       title,
		description: description,
		placeholder: placeholder,
		value:       initial,
	}
}

func (m *model) closePrompt() {
	m.prompt = nil
}

func (m *model) handlePromptKey(msg tea.KeyMsg) {
	if m.prompt == nil {
		return
	}

	switch msg.String() {
	case "esc":
		m.closePrompt()
		return
	case "enter":
		m.applyPrompt()
		m.closePrompt()
		return
	case "backspace":
		if len(m.prompt.value) > 0 {
			runes := []rune(m.prompt.value)
			m.prompt.value = string(runes[:len(runes)-1])
		}
		return
	}

	if len(msg.Runes) > 0 {
		m.prompt.value += string(msg.Runes)
	}
}

func (m *model) applyPrompt() {
	if m.prompt == nil {
		return
	}

	value := strings.TrimSpace(m.prompt.value)
	switch m.prompt.kind {
	case promptLibrarySearch:
		m.libraryQuery = value
		if err := m.refreshLibrary(); err != nil {
			m.setStatusDestructive(fmt.Sprintf("Search failed: %v", err))
		}
	case promptReaderSearch:
		m.applyReaderSearch(value)
	case promptGotoPage:
		m.applyGoToPage(value)
	case promptGotoPercent:
		m.applyGoToPercent(value)
	}
}

func (m *model) requestQuitCmd() tea.Cmd {
	if m.currentView == viewReader {
		if err := m.saveReaderState(); err != nil {
			m.setStatusDestructive(fmt.Sprintf("Failed to save progress before quit: %v", err))
			return nil
		}
	}
	return tea.Quit
}

func (m model) View() string {
	if !m.startupCompleted {
		return m.renderLoading()
	}

	var body string
	switch m.currentView {
	case viewLibrary:
		body = m.renderLibrary()
	case viewCommunities:
		body = m.renderCommunities()
	case viewAdd:
		body = m.renderAdd()
	case viewReader:
		body = m.renderReader()
	case viewSettings:
		body = m.renderSettings()
	default:
		body = "Unknown view"
	}

	if m.prompt != nil {
		return m.renderPromptModal()
	}

	if m.remove != nil {
		return m.renderRemoveModal()
	}

	if m.deviceAuth != nil {
		return m.renderDeviceAuthModal()
	}

	return body
}

func (m model) renderLoading() string {
	message := "Loading local library..."
	if m.statusText != "" {
		message = m.statusText
	}

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, message)
	}
	return message
}

func formatTime(value *time.Time) string {
	if value == nil {
		return "-"
	}
	return value.Local().Format("2006-01-02 15:04")
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func parsePositiveInt(value string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, err
	}
	if parsed <= 0 {
		return 0, errors.New("value must be positive")
	}
	return parsed, nil
}

func (m *model) setStatusDefault(msg string) {
	m.statusText = strings.TrimSpace(msg)
	m.statusKind = statusDefault
	m.statusSetAt = time.Now()
}

func (m *model) setStatusSuccess(msg string) {
	m.statusText = strings.TrimSpace(msg)
	m.statusKind = statusSuccess
	m.statusSetAt = time.Now()
}

func (m *model) setStatusDestructive(msg string) {
	m.statusText = strings.TrimSpace(msg)
	m.statusKind = statusDestructive
	m.statusSetAt = time.Now()
}

func (m *model) clearStatus() {
	m.statusText = ""
	m.statusKind = ""
	m.statusSetAt = time.Time{}
}

func (m model) effectiveStatus(now time.Time, fallback string) (string, statusVariant, bool) {
	text := strings.TrimSpace(m.statusText)
	if text == "" {
		text = fallback
	}

	if text == "" || text == "Ready" || text == "Reading" {
		return "", statusDefault, false
	}

	if !m.statusSetAt.IsZero() && now.Sub(m.statusSetAt) >= 10*time.Second {
		return "", statusDefault, false
	}

	kind := m.statusKind
	if kind == "" {
		kind = statusDefault
	}
	return text, kind, true
}

func (m model) shouldRunSync() bool {
	if m.container == nil || m.container.Auth == nil || m.container.Sync == nil {
		return false
	}
	return m.container.Auth.IsConnected()
}
