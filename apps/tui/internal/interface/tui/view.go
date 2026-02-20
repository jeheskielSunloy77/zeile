package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zeile/tui/internal/reader"
)

func (m model) renderLibrary() string {
	header := lipgloss.NewStyle().Bold(true).Render("Zeile - Library")
	query := "(none)"
	if strings.TrimSpace(m.libraryQuery) != "" {
		query = m.libraryQuery
	}
	subheader := fmt.Sprintf("Books: %d | Search: %s", len(m.libraryBooks), query)

	rows := make([]string, 0, len(m.libraryBooks)+2)
	if len(m.libraryBooks) == 0 {
		rows = append(rows, "No books yet. Press 'a' to import EPUB/PDF.")
	} else {
		for idx, book := range m.libraryBooks {
			marker := "  "
			if idx == m.librarySelected {
				marker = "->"
			}
			progress := m.libraryProgress[book.ID]
			status := fmt.Sprintf("%.1f%%", progress)
			if m.libraryFinished[book.ID] {
				status = "Finished"
			}

			row := fmt.Sprintf(
				"%s %s - %s | %s | Last opened: %s",
				marker,
				book.Title,
				book.Author,
				status,
				formatTime(book.LastOpened),
			)
			if idx == m.librarySelected {
				row = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Render(row)
			}
			rows = append(rows, row)
		}
	}

	hints := "Keys: / search | a add | Enter open | r remove | D delete disk | q quit"
	status := m.status
	if status == "" {
		status = "Ready"
	}

	return m.renderPinnedLayout(
		[]string{header, subheader, ""},
		strings.Join(rows, "\n"),
		[]string{"", m.renderFooterRow(hints, "Status: "+status)},
	)
}

func (m model) renderAdd() string {
	header := lipgloss.NewStyle().Bold(true).Render("Zeile - Add Book")
	focusPath := " "
	focusBrowser := " "
	if m.addFocus == addFocusPath {
		focusPath = "*"
	} else {
		focusBrowser = "*"
	}

	managed := "ON"
	if !m.addManagedCopy {
		managed = "OFF"
	}

	pathLine := fmt.Sprintf("%s Path: %s", focusPath, m.addPath)
	managedLine := fmt.Sprintf("Managed Copy: %s (toggle with 'm')", managed)

	browserLines := []string{fmt.Sprintf("%s File Browser: %s", focusBrowser, m.browserDir)}
	if len(m.browserEntries) == 0 {
		browserLines = append(browserLines, "  (empty)")
	} else {
		maxItems := 14
		start := 0
		if m.browserSelected >= maxItems {
			start = m.browserSelected - maxItems + 1
		}
		end := start + maxItems
		if end > len(m.browserEntries) {
			end = len(m.browserEntries)
		}

		for idx := start; idx < end; idx++ {
			entry := m.browserEntries[idx]
			marker := "  "
			if idx == m.browserSelected {
				marker = "->"
			}
			suffix := ""
			if entry.isDir {
				suffix = "/"
			}
			line := fmt.Sprintf("%s %s%s", marker, entry.name, suffix)
			if idx == m.browserSelected {
				line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Render(line)
			}
			browserLines = append(browserLines, line)
		}
	}

	progressLine := ""
	if m.importing {
		barWidth := 40
		filled := int(m.importPercent * float64(barWidth))
		if filled < 0 {
			filled = 0
		}
		if filled > barWidth {
			filled = barWidth
		}
		bar := strings.Repeat("#", filled) + strings.Repeat("-", barWidth-filled)
		progressLine = fmt.Sprintf("Importing: [%s] %.0f%% - %s (Esc to cancel)", bar, m.importPercent*100, m.importStage)
	}

	hints := "Keys: Tab focus | Enter import/open | i import from browser | u parent | m managed copy | q back"
	status := m.status
	if status == "" {
		status = "Ready"
	}

	bodyLines := []string{pathLine, managedLine}
	if progressLine != "" {
		bodyLines = append(bodyLines, progressLine)
	}
	bodyLines = append(bodyLines, "", strings.Join(browserLines, "\n"))

	return m.renderPinnedLayout(
		[]string{header, ""},
		strings.Join(bodyLines, "\n"),
		[]string{"", m.renderFooterRow(hints, "Status: "+status)},
	)
}

func (m model) renderReader() string {
	if m.readerBook.ID == "" {
		return "Reader has no active book"
	}

	spread := m.isSpreadMode()
	pageWidth, pageHeight := m.readerPageSize()
	pageCount := m.readerPageCount()
	if pageCount == 0 {
		return "No pages to display"
	}

	leftIndex := clamp(m.readerPage, 0, pageCount-1)
	rightIndex := leftIndex + 1

	leftContent := m.readerPageContent(leftIndex, pageWidth, pageHeight)
	leftPage := m.renderPageBox(leftContent, leftIndex+1, pageCount, pageWidth, pageHeight)

	var pagesView string
	if spread {
		rightContent := ""
		rightPageNum := 0
		if rightIndex < pageCount {
			rightContent = m.readerPageContent(rightIndex, pageWidth, pageHeight)
			rightPageNum = rightIndex + 1
		}
		rightPage := m.renderPageBox(rightContent, rightPageNum, pageCount, pageWidth, pageHeight)
		divider := m.renderSpreadDivider(pageHeight)
		pagesView = lipgloss.JoinHorizontal(lipgloss.Top, leftPage, "   ", divider, "   ", rightPage)
	} else {
		pagesView = leftPage
	}

	if m.readerZen {
		body := pagesView
		if m.readerHelp {
			body += "\n" + m.renderReaderHelp()
		}
		return m.renderPinnedLayout(nil, body, nil)
	}

	header := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf(
		"%s - %s | Mode: %s | %s",
		m.readerBook.Title,
		m.readerBook.Author,
		m.readerMode,
		finishedLabel(m.readerFinished),
	))

	hints := "Reader keys: arrows/hjkl page | z zen | / search | n/N next/prev | g/G go-to | m pdf mode | f finished | q back"
	status := m.status
	if status == "" {
		status = "Reading"
	}

	body := pagesView
	if m.readerHelp {
		body += "\n\n" + m.renderReaderHelp()
	}

	return m.renderPinnedLayout(
		[]string{header, ""},
		body,
		[]string{"", m.renderFooterRow(hints, "Status: "+status)},
	)
}

func (m model) readerPageContent(pageIndex, pageWidth, pageHeight int) string {
	if pageIndex < 0 {
		return ""
	}

	if m.isReaderTextMode() {
		if pageIndex >= len(m.readerPagination.Pages) {
			return ""
		}
		return m.readerPagination.Pages[pageIndex]
	}

	if pageIndex >= len(m.readerLayoutPages) {
		return ""
	}
	return reader.RenderLayoutPage(m.readerLayoutPages[pageIndex], pageWidth, pageHeight)
}

func (m model) renderPageBox(content string, pageNumber, totalPages, pageWidth, pageHeight int) string {
	lines := strings.Split(content, "\n")
	if len(lines) > pageHeight-1 {
		lines = lines[:pageHeight-1]
	}
	for len(lines) < pageHeight-1 {
		lines = append(lines, "")
	}

	footer := ""
	if pageNumber > 0 {
		footer = fmt.Sprintf("Page %d/%d", pageNumber, totalPages)
	}
	lines = append(lines, footer)
	pageText := strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Width(pageWidth).
		Height(pageHeight).
		Render(pageText)
}

func (m model) renderSpreadDivider(height int) string {
	if height < 1 {
		height = 1
	}
	lines := make([]string, height)
	for i := range lines {
		lines[i] = "│"
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Join(lines, "\n"))
}

func (m model) renderReaderHelp() string {
	content := strings.Join([]string{
		"Help",
		"- / search inside book",
		"- n / N next or previous match",
		"- g go to page, G go to percent",
		"- z zen mode toggle",
		"- m toggle PDF text/layout mode",
		"- f mark finished",
		"- q back to library",
	}, "\n")

	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Render(content)
}

func (m model) renderPrompt() string {
	if m.prompt == nil {
		return ""
	}

	value := m.prompt.value
	if value == "" {
		value = m.prompt.placeholder
	}

	content := m.prompt.title
	if m.prompt.description != "" {
		content += "\n" + m.prompt.description
	}
	content += "\n> " + value
	content += "\n(Enter to confirm, Esc to cancel)"

	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Render(content)
}

func finishedLabel(done bool) string {
	if done {
		return "Finished"
	}
	return "Unfinished"
}

func (m model) renderPinnedLayout(headerLines []string, body string, footerLines []string) string {
	if m.width <= 0 || m.height <= 0 {
		parts := make([]string, 0, len(headerLines)+len(footerLines)+1)
		parts = append(parts, headerLines...)
		parts = append(parts, body)
		parts = append(parts, footerLines...)
		return strings.Join(parts, "\n")
	}

	bodyLines := strings.Split(body, "\n")
	if len(bodyLines) == 0 {
		bodyLines = []string{""}
	}

	contentHeight := m.height - len(headerLines) - len(footerLines)
	if contentHeight < 0 {
		contentHeight = 0
	}

	if len(bodyLines) > contentHeight {
		bodyLines = bodyLines[:contentHeight]
	}
	for len(bodyLines) < contentHeight {
		bodyLines = append(bodyLines, "")
	}
	bodyLines = m.applyBodyGutter(bodyLines)

	lines := make([]string, 0, len(headerLines)+len(bodyLines)+len(footerLines))
	lines = append(lines, headerLines...)
	lines = append(lines, bodyLines...)
	lines = append(lines, footerLines...)

	view := strings.Join(lines, "\n")
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, view)
}

func (m model) applyBodyGutter(lines []string) []string {
	if m.width <= 0 || len(lines) == 0 {
		return lines
	}

	gutter, innerWidth := m.bodyLayout()
	leftPad := strings.Repeat(" ", gutter)
	withGutter := make([]string, 0, len(lines))
	for _, line := range lines {
		if innerWidth == 0 {
			withGutter = append(withGutter, "")
			continue
		}
		if lipgloss.Width(line) > innerWidth {
			line = truncateRunes(line, innerWidth)
		}
		withGutter = append(withGutter, leftPad+line)
	}
	return withGutter
}

func (m model) bodyContentWidth() int {
	_, width := m.bodyLayout()
	return width
}

func (m model) bodyLayout() (leftGutter int, contentWidth int) {
	if m.width <= 0 {
		return 0, 0
	}

	const maxContentWidth = 200
	width := m.width
	if width > maxContentWidth {
		width = maxContentWidth
	}

	if width == m.width && m.width >= 72 {
		width = m.width - 4
	}
	if width < 24 {
		width = m.width
	}

	gutter := (m.width - width) / 2
	if gutter < 0 {
		gutter = 0
	}
	return gutter, width
}

func (m model) renderFooterRow(left, right string) string {
	if m.width <= 0 {
		return left + " | " + right
	}

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)

	if rightWidth >= m.width {
		return truncateRunes(right, m.width)
	}

	maxLeft := m.width - rightWidth - 1
	if maxLeft < 0 {
		maxLeft = 0
	}
	if leftWidth > maxLeft {
		left = truncateRunes(left, maxLeft)
		leftWidth = lipgloss.Width(left)
	}

	padding := m.width - leftWidth - rightWidth
	if padding < 1 {
		padding = 1
	}
	return left + strings.Repeat(" ", padding) + right
}

func truncateRunes(value string, width int) string {
	if width <= 0 {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}
