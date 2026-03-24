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
	"github.com/jeheskielSunloy77/kern/tui/internal/domain"
	"github.com/jeheskielSunloy77/kern/tui/internal/infrastructure/repository"
	"github.com/jeheskielSunloy77/kern/tui/internal/reader"
)

func (m *model) handleReaderKey(msg tea.KeyMsg) {
	switch msg.String() {
	case "q":
		if err := m.saveReaderState(); err != nil {
			m.setStatusDestructive(fmt.Sprintf("Failed to save progress: %v", err))
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
	case "f":
		m.toggleFinished()
		return
	case "s":
		m.openSettings(viewReader)
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
	m.readerSectionStarts = session.SectionStarts
	m.readerChapterStarts = toOffsetSet(session.ChapterStarts)
	m.readerSearchMatches = nil
	m.readerSearchIndex = 0
	m.readerSearchQuery = ""
	m.readerFinished = session.State.IsFinished

	anchor := session.State.Locator.Offset
	if anchor < 0 {
		anchor = 0
	}
	if anchor == 0 && mode == domain.ReadingModeEPUB && session.State.Locator.SectionIndex > 0 {
		sectionIdx := clamp(session.State.Locator.SectionIndex, 0, len(m.readerSectionStarts)-1)
		if len(m.readerSectionStarts) > 0 {
			anchor = m.readerSectionStarts[sectionIdx]
		}
	}
	m.repaginateReader(anchor)
	if anchor == 0 && session.State.Locator.PageIndex > 0 {
		m.readerPage = clamp(session.State.Locator.PageIndex, 0, m.readerPageCount()-1)
	}

	m.currentView = viewReader
	m.readerHelp = false
	m.clearStatus()
	if err := m.container.Library.MarkOpened(context.Background(), session.Book.ID, time.Now().UTC()); err != nil {
		m.setStatusDestructive(fmt.Sprintf("Opened book, but failed to update last-opened: %v", err))
	}
	return nil
}

func (m *model) isReaderTextMode() bool {
	return true
}

func (m *model) toggleFinished() {
	if m.readerBook.ID == "" {
		return
	}

	next := !m.readerFinished
	if err := m.container.Reader.SetFinished(context.Background(), m.readerBook.ID, next); err != nil {
		m.setStatusDestructive(fmt.Sprintf("Failed to update finished state: %v", err))
		return
	}
	m.readerFinished = next
	if err := m.saveReaderState(); err != nil {
		m.setStatusDestructive(fmt.Sprintf("Finished state changed, but save failed: %v", err))
		return
	}

	if next {
		m.setStatusSuccess("Marked as finished")
	} else {
		m.setStatusSuccess("Marked as unfinished")
	}
}

func (m *model) repaginateReader(anchorOffset int) {
	if !m.isReaderTextMode() {
		return
	}
	pageWidth, pageHeight := m.readerPageSize()
	forcedStarts := map[int]struct{}(nil)
	if m.readerMode == domain.ReadingModeEPUB {
		forcedStarts = m.readerChapterStarts
	}
	m.readerPagination = m.readerTextDocument.PaginateWithForcedPageStarts(pageWidth, pageHeight, forcedStarts)
	m.readerPage = clamp(m.readerPagination.PageForOffset(anchorOffset), 0, m.readerPageCount()-1)
}

func (m *model) readerPageSize() (int, int) {
	width := m.bodyContentWidth()
	if width <= 0 {
		width = m.width
	}
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
	cfg := m.currentConfig()
	pageHeight := height - chromeRows
	pageHeight -= cfg.ParagraphSpacing
	pageHeight = pageHeight / cfg.LineSpacing
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
	return m.width >= m.currentConfig().SpreadThreshold
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
		m.setStatusDestructive(fmt.Sprintf("Failed to save progress: %v", err))
	}
}

func (m *model) readerPageStep() int {
	if m.isSpreadMode() {
		return 2
	}
	return 1
}

func (m *model) readerPageCount() int {
	return len(m.readerPagination.Pages)
}

func (m *model) readerAnchorOffset() int {
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

	offset := m.readerPagination.OffsetForPage(m.readerPage)
	state.Locator = domain.Locator{
		Offset:       offset,
		PageIndex:    m.readerPage,
		SectionIndex: m.readerSectionIndexForOffset(offset),
	}
	state.ProgressPercent = reader.ProgressPercent(offset, m.readerTextDocument.TokenCount())

	return m.container.Reader.SaveState(context.Background(), state)
}

func toOffsetSet(offsets []int) map[int]struct{} {
	if len(offsets) == 0 {
		return nil
	}
	result := make(map[int]struct{}, len(offsets))
	for _, offset := range offsets {
		if offset < 0 {
			continue
		}
		result[offset] = struct{}{}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func (m *model) readerSectionIndexForOffset(offset int) int {
	if m.readerMode != domain.ReadingModeEPUB || len(m.readerSectionStarts) == 0 {
		return 0
	}

	index := sort.Search(len(m.readerSectionStarts), func(i int) bool {
		return m.readerSectionStarts[i] > offset
	}) - 1
	if index < 0 {
		return 0
	}
	if index >= len(m.readerSectionStarts) {
		return len(m.readerSectionStarts) - 1
	}
	return index
}

func (m *model) applyReaderSearch(query string) {
	query = strings.TrimSpace(query)
	m.readerSearchQuery = query
	m.readerSearchMatches = nil
	m.readerSearchIndex = 0

	if query == "" {
		m.setStatusDefault("Cleared in-book search")
		return
	}

	matches := m.readerTextDocument.SearchTokenOffsets(query)
	if len(matches) == 0 {
		m.setStatusDefault(fmt.Sprintf("No matches for %q", query))
		return
	}

	m.readerSearchMatches = matches
	m.readerSearchIndex = 0
	offset := matches[0]
	m.readerPage = clamp(m.readerPagination.PageForOffset(offset), 0, m.readerPageCount()-1)
	if err := m.saveReaderState(); err != nil {
		m.setStatusDestructive(fmt.Sprintf("Match found, but failed to save position: %v", err))
		return
	}
	m.setStatusSuccess(fmt.Sprintf("Found %d match(es)", len(matches)))
}

func (m *model) jumpToSearchResult(direction int) {
	if len(m.readerSearchMatches) == 0 {
		m.setStatusDefault("No active search results")
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
		m.setStatusDestructive(fmt.Sprintf("Failed to save search position: %v", err))
		return
	}
	m.setStatusSuccess(fmt.Sprintf("Match %d/%d", m.readerSearchIndex+1, count))
}

func (m *model) applyGoToPage(value string) {
	pageNumber, err := parsePositiveInt(value)
	if err != nil {
		m.setStatusDestructive("Invalid page number")
		return
	}
	if m.readerPageCount() == 0 {
		m.setStatusDestructive("No pages available")
		return
	}

	m.readerPage = clamp(pageNumber-1, 0, m.readerPageCount()-1)
	if err := m.saveReaderState(); err != nil {
		m.setStatusDestructive(fmt.Sprintf("Jumped to page, but failed to save: %v", err))
		return
	}
	m.setStatusSuccess(fmt.Sprintf("Jumped to page %d", m.readerPage+1))
}

func (m *model) applyGoToPercent(value string) {
	percent, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		m.setStatusDestructive("Invalid percent value")
		return
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	if m.readerPageCount() == 0 {
		m.setStatusDestructive("No pages available")
		return
	}

	target := int((percent / 100.0) * float64(m.readerPageCount()-1))
	m.readerPage = clamp(target, 0, m.readerPageCount()-1)
	if err := m.saveReaderState(); err != nil {
		m.setStatusDestructive(fmt.Sprintf("Jumped to percent, but failed to save: %v", err))
		return
	}
	m.setStatusSuccess(fmt.Sprintf("Jumped to %.1f%%", percent))
}

func (m *model) tryAutoOpenResumeBook() {
	if m.container == nil || m.container.Library == nil {
		return
	}
	book, err := m.container.Library.MostRecentUnfinishedBook(context.Background())
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			m.setStatusDestructive(fmt.Sprintf("Resume check failed: %v", err))
		}
		return
	}
	if err := m.openBook(book.ID, nil); err != nil {
		m.setStatusDestructive(fmt.Sprintf("Failed to open resume book: %v", err))
	}
}
