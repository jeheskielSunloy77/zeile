package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
		m.currentView = viewAdd
		m.status = ""
	case "enter":
		book, ok := m.selectedBook()
		if !ok {
			return nil
		}
		if err := m.openBook(book.ID, nil); err != nil {
			m.status = fmt.Sprintf("Open failed: %v", err)
		}
	case "r":
		book, ok := m.selectedBook()
		if ok {
			m.promptFor(
				promptRemoveConfirm,
				"Remove From Library",
				fmt.Sprintf("Type REMOVE to remove '%s' from library only", book.Title),
				"REMOVE",
				"",
			)
		}
	case "D":
		book, ok := m.selectedBook()
		if ok {
			description := fmt.Sprintf("Scary action! Type DELETE to permanently delete file:\n%s", book.ManagedPath)
			m.promptFor(promptDeleteDiskConfirm, "Delete From Disk", description, "DELETE", "")
		}
	case "?":
		m.status = "Library: / search | a add | Enter open | r remove | D delete disk | q quit"
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
	case "tab":
		if m.addFocus == addFocusPath {
			m.addFocus = addFocusBrowser
		} else {
			m.addFocus = addFocusPath
		}
		return nil
	case "m":
		m.addManagedCopy = !m.addManagedCopy
		return nil
	case "u":
		parent := filepath.Dir(m.browserDir)
		m.loadBrowser(parent)
		return nil
	}

	if m.addFocus == addFocusPath {
		switch msg.String() {
		case "enter":
			path := strings.TrimSpace(m.addPath)
			if path == "" {
				m.status = "Enter a file path or use the file browser"
				return nil
			}
			return m.startImport(path)
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
		return m.startImport(entry.path)
	}

	return nil
}

func (m *model) startImport(path string) tea.Cmd {
	if m.container == nil || m.container.Library == nil {
		m.status = "Library service unavailable"
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
		m.status = fmt.Sprintf("File browser error: %v", err)
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
