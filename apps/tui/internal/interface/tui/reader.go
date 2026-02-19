package tui

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zeile/tui/internal/domain"
	"github.com/zeile/tui/internal/infrastructure/repository"
	"github.com/zeile/tui/internal/reader"
)

func (m *model) handleReaderKey(msg tea.KeyMsg) {
	switch msg.String() {
	case "q":
		if err := m.saveReaderState(); err != nil {
			m.status = fmt.Sprintf("Failed to save progress: %v", err)
		}
		m.currentView = viewLibrary
		m.readerHelp = false
		_ = m.refreshLibrary()
		return
	case "?":
		m.readerHelp = !m.readerHelp
		return
	case "z":
		m.readerZen = !m.readerZen
		if m.isReaderTextMode() {
			anchor := m.readerAnchorOffset()
			m.repaginateReader(anchor)
		}
		return
	case "left", "h", "up", "k":
		m.moveReaderPage(-m.readerPageStep())
		return
	case "right", "l", "down", "j":
		m.moveReaderPage(m.readerPageStep())
		return
	case "/":
		if !m.isReaderTextMode() {
			m.status = "In-book search is available in EPUB and PDF text mode"
			return
		}
		m.promptFor(promptReaderSearch, "In-Book Search", "Find text in current reading mode", "search", m.readerSearchQuery)
		return
	case "n":
		m.jumpToSearchResult(1)
		return
	case "N":
		m.jumpToSearchResult(-1)
		return
	case "g":
		m.promptFor(promptGotoPage, "Go To Page", "Jump to page number (1-based)", "page", "")
		return
	case "G":
		m.promptFor(promptGotoPercent, "Go To Percent", "Jump to reading percent (0-100)", "percent", "")
		return
	case "m":
		m.togglePDFMode()
		return
	case "f":
		m.toggleFinished()
		return
	}
}

func (m *model) openBook(bookID string, preferredMode *domain.ReadingMode) error {
	if m.container == nil || m.container.Library == nil || m.container.Reader == nil {
		return errors.New("app services unavailable")
	}

	book, err := m.container.Library.BookByID(context.Background(), bookID)
	if err != nil {
		return err
	}

	mode := domain.DefaultModeForFormat(book.Format)
	if preferredMode != nil {
		mode = *preferredMode
	} else {
		states, err := m.container.Library.StatesForBook(context.Background(), book.ID)
		if err == nil && len(states) > 0 {
			sort.SliceStable(states, func(i, j int) bool {
				return states[i].UpdatedAt.After(states[j].UpdatedAt)
			})
			mode = states[0].Mode
		}
	}

	if mode == domain.ReadingModePDFLayout {
		return m.openLayoutMode(bookID)
	}
	return m.openTextMode(bookID, mode)
}

func (m *model) openTextMode(bookID string, mode domain.ReadingMode) error {
	session, err := m.container.Reader.LoadTextSession(context.Background(), bookID, mode)
	if err != nil {
		return err
	}

	m.readerBook = session.Book
	m.readerMode = mode
	m.readerTextDocument = session.Document
	m.readerLayoutPages = nil
	m.readerSearchMatches = nil
	m.readerSearchIndex = 0
	m.readerSearchQuery = ""
	m.readerFinished = session.State.IsFinished

	anchor := session.State.Locator.Offset
	if anchor < 0 {
		anchor = 0
	}
	m.repaginateReader(anchor)
	if anchor == 0 && session.State.Locator.PageIndex > 0 {
		m.readerPage = clamp(session.State.Locator.PageIndex, 0, m.readerPageCount()-1)
	}

	m.currentView = viewReader
	m.readerHelp = false
	m.status = ""
	if err := m.container.Library.MarkOpened(context.Background(), session.Book.ID, time.Now().UTC()); err != nil {
		m.status = fmt.Sprintf("Opened book, but failed to update last-opened: %v", err)
	}
	return nil
}

func (m *model) openLayoutMode(bookID string) error {
	session, err := m.container.Reader.LoadLayoutSession(context.Background(), bookID)
	if err != nil {
		return err
	}

	m.readerBook = session.Book
	m.readerMode = domain.ReadingModePDFLayout
	m.readerLayoutPages = session.Pages
	m.readerTextDocument = reader.TextDocument{}
	m.readerPagination = reader.TextPagination{}
	m.readerPage = clamp(session.State.Locator.PageIndex, 0, len(session.Pages)-1)
	m.readerSearchMatches = nil
	m.readerSearchIndex = 0
	m.readerSearchQuery = ""
	m.readerFinished = session.State.IsFinished
	m.currentView = viewReader
	m.readerHelp = false
	m.status = ""
	if err := m.container.Library.MarkOpened(context.Background(), session.Book.ID, time.Now().UTC()); err != nil {
		m.status = fmt.Sprintf("Opened book, but failed to update last-opened: %v", err)
	}
	return nil
}

func (m *model) isReaderTextMode() bool {
	switch m.readerMode {
	case domain.ReadingModeEPUB, domain.ReadingModePDFText:
		return true
	default:
		return false
	}
}

func (m *model) togglePDFMode() {
	if m.readerBook.Format != domain.BookFormatPDF {
		m.status = "PDF mode toggle is only available for PDF books"
		return
	}

	if err := m.saveReaderState(); err != nil {
		m.status = fmt.Sprintf("Failed to save before mode switch: %v", err)
	}

	if m.readerMode == domain.ReadingModePDFLayout {
		if err := m.openTextMode(m.readerBook.ID, domain.ReadingModePDFText); err != nil {
			m.status = fmt.Sprintf("Failed to switch mode: %v", err)
			return
		}
		m.status = "Switched to PDF text mode"
		return
	}

	if err := m.openLayoutMode(m.readerBook.ID); err != nil {
		m.status = fmt.Sprintf("Failed to switch mode: %v", err)
		return
	}
	m.status = "Switched to PDF layout mode"
}

func (m *model) toggleFinished() {
	if m.readerBook.ID == "" {
		return
	}

	next := !m.readerFinished
	if err := m.container.Reader.SetFinished(context.Background(), m.readerBook.ID, next); err != nil {
		m.status = fmt.Sprintf("Failed to update finished state: %v", err)
		return
	}
	m.readerFinished = next
	if err := m.saveReaderState(); err != nil {
		m.status = fmt.Sprintf("Finished state changed, but save failed: %v", err)
		return
	}

	if next {
		m.status = "Marked as finished"
	} else {
		m.status = "Marked as unfinished"
	}
}

func (m *model) repaginateReader(anchorOffset int) {
	if !m.isReaderTextMode() {
		return
	}
	pageWidth, pageHeight := m.readerPageSize()
	m.readerPagination = m.readerTextDocument.Paginate(pageWidth, pageHeight)
	m.readerPage = clamp(m.readerPagination.PageForOffset(anchorOffset), 0, m.readerPageCount()-1)
}

func (m *model) readerPageSize() (int, int) {
	width := m.width
	height := m.height
	if width <= 0 {
		width = 120
	}
	if height <= 0 {
		height = 40
	}

	chromeRows := 5
	if m.readerZen {
		chromeRows = 1
	}
	pageHeight := height - chromeRows
	if pageHeight < 8 {
		pageHeight = 8
	}

	if m.isSpreadMode() {
		gap := 4
		pageWidth := (width - gap - 6) / 2
		if pageWidth < 20 {
			pageWidth = 20
		}
		return pageWidth, pageHeight
	}

	pageWidth := width - 8
	if pageWidth < 30 {
		pageWidth = 30
	}
	return pageWidth, pageHeight
}

func (m *model) isSpreadMode() bool {
	if m.container == nil {
		return m.width >= 120
	}
	return m.width >= m.container.Config.MinSpreadWidth
}

func (m *model) moveReaderPage(delta int) {
	if m.readerPageCount() == 0 {
		return
	}

	next := clamp(m.readerPage+delta, 0, m.readerPageCount()-1)
	if next == m.readerPage {
		return
	}
	m.readerPage = next
	if err := m.saveReaderState(); err != nil {
		m.status = fmt.Sprintf("Failed to save progress: %v", err)
	}
}

func (m *model) readerPageStep() int {
	if m.isSpreadMode() {
		return 2
	}
	return 1
}

func (m *model) readerPageCount() int {
	if m.isReaderTextMode() {
		return len(m.readerPagination.Pages)
	}
	return len(m.readerLayoutPages)
}

func (m *model) readerAnchorOffset() int {
	if !m.isReaderTextMode() {
		return 0
	}
	return m.readerPagination.OffsetForPage(m.readerPage)
}

func (m *model) saveReaderState() error {
	if m.container == nil || m.container.Reader == nil || m.readerBook.ID == "" {
		return nil
	}

	state := domain.ReadingState{
		BookID:     m.readerBook.ID,
		Mode:       m.readerMode,
		UpdatedAt:  time.Now().UTC(),
		IsFinished: m.readerFinished,
	}

	if m.isReaderTextMode() {
		offset := m.readerPagination.OffsetForPage(m.readerPage)
		state.Locator = domain.Locator{Offset: offset, PageIndex: m.readerPage}
		state.ProgressPercent = reader.ProgressPercent(offset, m.readerTextDocument.TokenCount())
	} else {
		totalPages := len(m.readerLayoutPages)
		if totalPages <= 0 {
			totalPages = 1
		}
		state.Locator = domain.Locator{PageIndex: m.readerPage}
		state.ProgressPercent = float64(m.readerPage+1) / float64(totalPages) * 100
	}

	return m.container.Reader.SaveState(context.Background(), state)
}

func (m *model) applyReaderSearch(query string) {
	query = strings.TrimSpace(query)
	m.readerSearchQuery = query
	m.readerSearchMatches = nil
	m.readerSearchIndex = 0

	if query == "" {
		m.status = "Cleared in-book search"
		return
	}
	if !m.isReaderTextMode() {
		m.status = "Search is available only in EPUB and PDF text mode"
		return
	}

	matches := m.readerTextDocument.SearchTokenOffsets(query)
	if len(matches) == 0 {
		m.status = fmt.Sprintf("No matches for %q", query)
		return
	}

	m.readerSearchMatches = matches
	m.readerSearchIndex = 0
	offset := matches[0]
	m.readerPage = clamp(m.readerPagination.PageForOffset(offset), 0, m.readerPageCount()-1)
	if err := m.saveReaderState(); err != nil {
		m.status = fmt.Sprintf("Match found, but failed to save position: %v", err)
		return
	}
	m.status = fmt.Sprintf("Found %d match(es)", len(matches))
}

func (m *model) jumpToSearchResult(direction int) {
	if len(m.readerSearchMatches) == 0 {
		m.status = "No active search results"
		return
	}
	if !m.isReaderTextMode() {
		m.status = "Search navigation is unavailable in layout mode"
		return
	}

	count := len(m.readerSearchMatches)
	m.readerSearchIndex = (m.readerSearchIndex + direction) % count
	if m.readerSearchIndex < 0 {
		m.readerSearchIndex += count
	}
	offset := m.readerSearchMatches[m.readerSearchIndex]
	m.readerPage = clamp(m.readerPagination.PageForOffset(offset), 0, m.readerPageCount()-1)
	if err := m.saveReaderState(); err != nil {
		m.status = fmt.Sprintf("Failed to save search position: %v", err)
		return
	}
	m.status = fmt.Sprintf("Match %d/%d", m.readerSearchIndex+1, count)
}

func (m *model) applyGoToPage(value string) {
	pageNumber, err := parsePositiveInt(value)
	if err != nil {
		m.status = "Invalid page number"
		return
	}
	if m.readerPageCount() == 0 {
		m.status = "No pages available"
		return
	}

	m.readerPage = clamp(pageNumber-1, 0, m.readerPageCount()-1)
	if err := m.saveReaderState(); err != nil {
		m.status = fmt.Sprintf("Jumped to page, but failed to save: %v", err)
		return
	}
	m.status = fmt.Sprintf("Jumped to page %d", m.readerPage+1)
}

func (m *model) applyGoToPercent(value string) {
	percent, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		m.status = "Invalid percent value"
		return
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	if m.readerPageCount() == 0 {
		m.status = "No pages available"
		return
	}

	target := int((percent / 100.0) * float64(m.readerPageCount()-1))
	m.readerPage = clamp(target, 0, m.readerPageCount()-1)
	if err := m.saveReaderState(); err != nil {
		m.status = fmt.Sprintf("Jumped to percent, but failed to save: %v", err)
		return
	}
	m.status = fmt.Sprintf("Jumped to %.1f%%", percent)
}

func (m *model) tryAutoOpenResumeBook() {
	if m.container == nil || m.container.Library == nil {
		return
	}
	book, err := m.container.Library.MostRecentUnfinishedBook(context.Background())
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			m.status = fmt.Sprintf("Resume check failed: %v", err)
		}
		return
	}
	if err := m.openBook(book.ID, nil); err != nil {
		m.status = fmt.Sprintf("Failed to open resume book: %v", err)
	}
}
