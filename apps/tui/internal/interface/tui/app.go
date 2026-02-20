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
	"github.com/zeile/tui/internal/infrastructure/repository"
	"github.com/zeile/tui/internal/reader"
)

type viewID int

const (
	viewLibrary viewID = iota
	viewAdd
	viewReader
)

type addFocus int

const (
	addFocusPath addFocus = iota
	addFocusBrowser
)

type promptKind int

const (
	promptNone promptKind = iota
	promptLibrarySearch
	promptReaderSearch
	promptGotoPage
	promptGotoPercent
	promptRemoveConfirm
	promptDeleteDiskConfirm
)

type promptState struct {
	kind        promptKind
	title       string
	description string
	value       string
	placeholder string
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

type model struct {
	container *application.Container

	width  int
	height int

	currentView viewID
	status      string

	prompt *promptState

	libraryBooks     []domain.Book
	librarySelected  int
	libraryQuery     string
	libraryProgress  map[string]float64
	libraryFinished  map[string]bool
	startupCompleted bool

	addPath         string
	addManagedCopy  bool
	addFocus        addFocus
	browserDir      string
	browserEntries  []browserEntry
	browserSelected int
	importing       bool
	importStage     string
	importPercent   float64
	importCancel    context.CancelFunc
	importEvents    <-chan tea.Msg

	readerBook          domain.Book
	readerMode          domain.ReadingMode
	readerTextDocument  reader.TextDocument
	readerPagination    reader.TextPagination
	readerSectionStarts []int
	readerLayoutPages   []string
	readerPage          int
	readerSearchQuery   string
	readerSearchMatches []int
	readerSearchIndex   int
	readerZen           bool
	readerHelp          bool
	readerFinished      bool
}

func New(container *application.Container) tea.Model {
	m := model{
		container:       container,
		currentView:     viewLibrary,
		addManagedCopy:  true,
		addFocus:        addFocusPath,
		libraryProgress: map[string]float64{},
		libraryFinished: map[string]bool{},
	}

	if container != nil {
		m.addManagedCopy = container.Config.ManagedCopyDefault
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
		book, err := container.Library.MostRecentUnfinishedBook(ctx)
		if err == nil {
			resumeBookID = book.ID
		} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return startupLoadedMsg{books: books, err: err}
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
			m.status = fmt.Sprintf("Startup error: %v", msg.err)
			m.startupCompleted = true
			return m, nil
		}

		m.libraryBooks = msg.books
		m.refreshProgressSummary()
		m.startupCompleted = true
		if msg.resumeBookID != "" {
			if err := m.openBook(msg.resumeBookID, nil); err != nil {
				m.status = fmt.Sprintf("Failed to auto-resume: %v", err)
			}
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
			if errors.Is(msg.err, context.Canceled) {
				m.status = "Import canceled"
			} else {
				m.status = fmt.Sprintf("Import failed: %v", msg.err)
			}
			return m, nil
		}

		m.status = fmt.Sprintf("Imported: %s", msg.book.Title)
		m.addPath = ""
		if err := m.refreshLibrary(); err != nil {
			m.status = fmt.Sprintf("Imported, but failed to reload library: %v", err)
		}
		return m, nil

	case importChannelClosedMsg:
		m.importEvents = nil
		if m.importing {
			m.importing = false
			m.importCancel = nil
		}
		return m, nil

	case tea.QuitMsg:
		if m.currentView == viewReader {
			if err := m.saveReaderState(); err != nil {
				m.status = fmt.Sprintf("Failed to save progress before quit: %v", err)
				return m, nil
			}
		}
		return m, tea.Quit

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, m.requestQuitCmd()
		}

		if m.prompt != nil {
			m.handlePromptKey(msg)
			return m, nil
		}

		switch m.currentView {
		case viewLibrary:
			cmd := m.handleLibraryKey(msg)
			return m, cmd
		case viewAdd:
			cmd := m.handleAddKey(msg)
			return m, cmd
		case viewReader:
			m.handleReaderKey(msg)
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
			m.status = fmt.Sprintf("Search failed: %v", err)
		}
	case promptReaderSearch:
		m.applyReaderSearch(value)
	case promptGotoPage:
		m.applyGoToPage(value)
	case promptGotoPercent:
		m.applyGoToPercent(value)
	case promptRemoveConfirm:
		if strings.EqualFold(value, "REMOVE") {
			book, ok := m.selectedBook()
			if ok {
				if err := m.container.Library.RemoveFromLibrary(context.Background(), book.ID); err != nil {
					m.status = fmt.Sprintf("Remove failed: %v", err)
				} else {
					m.status = fmt.Sprintf("Removed from library: %s", book.Title)
					_ = m.refreshLibrary()
				}
			}
		} else {
			m.status = "Removal canceled: type REMOVE to confirm"
		}
	case promptDeleteDiskConfirm:
		if value == "DELETE" {
			book, ok := m.selectedBook()
			if ok {
				if err := m.container.Library.DeleteFromDisk(context.Background(), book.ID); err != nil {
					m.status = fmt.Sprintf("Delete failed: %v", err)
				} else {
					m.status = fmt.Sprintf("Deleted from disk: %s", book.Title)
					_ = m.refreshLibrary()
				}
			}
		} else {
			m.status = "Delete canceled: type DELETE exactly to confirm"
		}
	}
}

func (m *model) requestQuitCmd() tea.Cmd {
	if m.currentView == viewReader {
		if err := m.saveReaderState(); err != nil {
			m.status = fmt.Sprintf("Failed to save progress before quit: %v", err)
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
	case viewAdd:
		body = m.renderAdd()
	case viewReader:
		body = m.renderReader()
	default:
		body = "Unknown view"
	}

	if m.prompt != nil {
		body += "\n" + m.renderPrompt()
	}

	return body
}

func (m model) renderLoading() string {
	message := "Loading local library..."
	if m.status != "" {
		message = m.status
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
