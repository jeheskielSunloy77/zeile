package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zeile/tui/internal/domain"
)

func (m *model) handleLibraryKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "q":
		return tea.Quit
	case "up", "k":
		if m.librarySelected > 0 {
			m.librarySelected--
		}
	case "down", "j":
		if m.librarySelected < len(m.libraryBooks)-1 {
			m.librarySelected++
		}
	case "/":
		m.promptFor(promptLibrarySearch, "Library Search", "Search title/author/file name", "query", m.libraryQuery)
	case "a":
		m.startAddFlow()
		m.clearStatus()
	case "enter":
		book, ok := m.selectedBook()
		if !ok {
			return nil
		}
		if err := m.openBook(book.ID, nil); err != nil {
			m.setStatusDestructive(fmt.Sprintf("Open failed: %v", err))
		}
	case "r":
		book, ok := m.selectedBook()
		if ok {
			m.openRemoveModal(book)
		}
	case "?":
		m.setStatusDefault("Library: Tab/Shift+Tab switch views  / search  a add  Enter open  r remove")
	}
	return nil
}

func (m *model) handleAddKey(msg tea.KeyMsg) tea.Cmd {
	if m.importing {
		switch msg.String() {
		case "esc", "c":
			if m.importCancel != nil {
				m.importCancel()
			}
		}
		return nil
	}

	switch msg.String() {
	case "q":
		m.currentView = viewLibrary
		return nil
	case "b":
		m.backAddStep()
		return nil
	}

	switch m.addStep {
	case addStepChooseSource:
		switch msg.String() {
		case "up", "k", "down", "j", "left", "h", "right", "l":
			if m.addSourceMethod == addSourcePath {
				m.addSourceMethod = addSourceSelector
			} else {
				m.addSourceMethod = addSourcePath
			}
		case "enter":
			if m.addSourceMethod == addSourcePath {
				m.addStep = addStepPathInput
			} else {
				m.loadBrowserHome()
				m.addStep = addStepFileSelector
			}
		}
		return nil

	case addStepPathInput:
		switch msg.String() {
		case "enter":
			path := strings.TrimSpace(m.addPath)
			if path == "" {
				m.setStatusDestructive("Enter a file path")
				return nil
			}
			m.addStep = addStepManagedCopy
		case "backspace":
			if len(m.addPath) > 0 {
				runes := []rune(m.addPath)
				m.addPath = string(runes[:len(runes)-1])
			}
		default:
			if len(msg.Runes) > 0 {
				m.addPath += string(msg.Runes)
			}
		}
		return nil

	case addStepFileSelector:
		switch msg.String() {
		case "u":
			parent := filepath.Dir(m.browserDir)
			m.loadBrowser(parent)
			return nil
		}
		if len(m.browserEntries) == 0 {
			return nil
		}

		switch msg.String() {
		case "up", "k":
			if m.browserSelected > 0 {
				m.browserSelected--
			}
		case "down", "j":
			if m.browserSelected < len(m.browserEntries)-1 {
				m.browserSelected++
			}
		case "enter", "i":
			entry := m.browserEntries[m.browserSelected]
			if entry.isDir {
				m.loadBrowser(entry.path)
				return nil
			}
			m.addPath = entry.path
			m.addStep = addStepManagedCopy
		}
		return nil

	case addStepManagedCopy:
		switch msg.String() {
		case "left", "h", "down", "j", "n":
			m.addManagedCopy = false
			return nil
		case "right", "l", "up", "k", "y":
			m.addManagedCopy = true
			return nil
		case "m":
			m.addManagedCopy = !m.addManagedCopy
			return nil
		case "enter":
			path := strings.TrimSpace(m.addPath)
			if path == "" {
				m.setStatusDestructive("Select or enter a file path first")
				return nil
			}
			cmd := m.startImport(path)
			if m.importing {
				m.addStep = addStepImporting
			}
			return cmd
		}
		return nil
	}

	return nil
}

func (m *model) startAddFlow() {
	m.currentView = viewAdd
	m.addStep = addStepChooseSource
	m.addSourceMethod = addSourcePath
	m.addPath = ""
	m.importing = false
	m.importCancel = nil
	m.importEvents = nil
	m.importStage = ""
	m.importPercent = 0
	if m.container != nil {
		m.addManagedCopy = m.container.Config.ManagedCopyDefault
	}
	m.loadBrowserHome()
}

func (m *model) loadBrowserHome() {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return
	}
	m.loadBrowser(home)
}

func (m *model) backAddStep() {
	switch m.addStep {
	case addStepPathInput, addStepFileSelector:
		m.addStep = addStepChooseSource
	case addStepManagedCopy:
		if m.addSourceMethod == addSourceSelector {
			m.addStep = addStepFileSelector
		} else {
			m.addStep = addStepPathInput
		}
	}
}

func (m *model) openRemoveModal(book domain.Book) {
	m.remove = &removeState{
		bookID:      book.ID,
		bookTitle:   book.Title,
		managedPath: book.ManagedPath,
		step:        removeStepChooseAction,
		action:      removeActionLibrary,
	}
}

func (m *model) closeRemoveModal() {
	m.remove = nil
}

func (m *model) handleRemoveKey(msg tea.KeyMsg) {
	if m.remove == nil {
		return
	}

	switch m.remove.step {
	case removeStepChooseAction:
		switch msg.String() {
		case "esc", "q":
			m.closeRemoveModal()
		case "up", "k", "down", "j", "left", "h", "right", "l":
			if m.remove.action == removeActionLibrary {
				m.remove.action = removeActionDeleteDisk
			} else {
				m.remove.action = removeActionLibrary
			}
		case "enter":
			if !m.currentConfig().DeleteConfirmation {
				if m.remove.action == removeActionLibrary {
					m.remove.value = "REMOVE"
				} else {
					m.remove.value = "DELETE"
				}
				m.applyRemove()
				m.closeRemoveModal()
				return
			}
			m.remove.step = removeStepConfirm
			m.remove.value = ""
		}
	case removeStepConfirm:
		switch msg.String() {
		case "esc":
			m.remove.step = removeStepChooseAction
			m.remove.value = ""
			return
		case "q":
			m.closeRemoveModal()
			return
		case "backspace":
			if len(m.remove.value) > 0 {
				runes := []rune(m.remove.value)
				m.remove.value = string(runes[:len(runes)-1])
			}
			return
		case "enter":
			m.applyRemove()
			m.closeRemoveModal()
			return
		}
		if len(msg.Runes) > 0 {
			m.remove.value += string(msg.Runes)
		}
	}
}

func (m *model) applyRemove() {
	if m.remove == nil {
		return
	}

	switch m.remove.action {
	case removeActionLibrary:
		if !strings.EqualFold(strings.TrimSpace(m.remove.value), "REMOVE") {
			m.setStatusDefault("Removal canceled: type REMOVE to confirm")
			return
		}
		if err := m.container.Library.RemoveFromLibrary(context.Background(), m.remove.bookID); err != nil {
			m.setStatusDestructive(fmt.Sprintf("Remove failed: %v", err))
			return
		}
		m.setStatusSuccess(fmt.Sprintf("Removed from library: %s", m.remove.bookTitle))
		_ = m.refreshLibrary()
	case removeActionDeleteDisk:
		if strings.TrimSpace(m.remove.value) != "DELETE" {
			m.setStatusDefault("Delete canceled: type DELETE exactly to confirm")
			return
		}
		if err := m.container.Library.DeleteFromDisk(context.Background(), m.remove.bookID); err != nil {
			m.setStatusDestructive(fmt.Sprintf("Delete failed: %v", err))
			return
		}
		m.setStatusSuccess(fmt.Sprintf("Deleted from disk: %s", m.remove.bookTitle))
		_ = m.refreshLibrary()
	}
}

func (m *model) startImport(path string) tea.Cmd {
	if m.container == nil || m.container.Library == nil {
		m.setStatusDestructive("Library service unavailable")
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan tea.Msg, 8)
	m.importing = true
	m.importCancel = cancel
	m.importEvents = events
	m.importStage = "Starting"
	m.importPercent = 0

	library := m.container.Library
	managed := m.addManagedCopy
	cleanPath := filepath.Clean(path)

	go func() {
		book, err := library.ImportBook(ctx, cleanPath, managed, func(stage string, percent float64) {
			select {
			case <-ctx.Done():
				return
			case events <- importProgressMsg{stage: stage, percent: percent}:
			}
		})
		if err != nil {
			events <- importDoneMsg{err: err}
			close(events)
			return
		}

		events <- importDoneMsg{book: book}
		close(events)
	}()

	return waitForImportEvent(events)
}

func (m *model) loadBrowser(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		m.setStatusDestructive(fmt.Sprintf("File browser error: %v", err))
		return
	}

	browserEntries := make([]browserEntry, 0, len(entries)+1)
	cleanDir := filepath.Clean(dir)
	parent := filepath.Dir(cleanDir)
	if parent != cleanDir {
		browserEntries = append(browserEntries, browserEntry{name: "..", path: parent, isDir: true})
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !entry.IsDir() {
			if _, err := domain.DetectFormat(name); err != nil {
				continue
			}
		}
		browserEntries = append(browserEntries, browserEntry{
			name:  name,
			path:  filepath.Join(cleanDir, name),
			isDir: entry.IsDir(),
		})
	}

	sort.SliceStable(browserEntries, func(i, j int) bool {
		if browserEntries[i].isDir == browserEntries[j].isDir {
			return strings.ToLower(browserEntries[i].name) < strings.ToLower(browserEntries[j].name)
		}
		return browserEntries[i].isDir && !browserEntries[j].isDir
	})

	m.browserDir = cleanDir
	m.browserEntries = browserEntries
	m.browserSelected = 0
}
