package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderAccount() string {
	headerLines := []string{
		m.renderMainNavHeader(viewAccount),
		"",
	}

	actions := m.accountActions()
	selected := m.accountField
	if selected < 0 {
		selected = 0
	}
	if len(actions) > 0 && selected >= len(actions) {
		selected = len(actions) - 1
	}

	theme := m.activeTheme()
	lines := make([]string, 0, len(actions)+10)
	if m.isAuthenticated() {
		lines = append(
			lines,
			lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render("Account"),
			"",
			lipgloss.NewStyle().Bold(true).Render(m.connectionLabel),
			"",
			"Your account is connected on this device.",
			"Choose an action below:",
			"",
		)
	} else {
		lines = append(
			lines,
			lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render("Login"),
			"",
			"Sign in to connect this app with your Zeile Cloud account.",
			"After login, you can run manual sync and manage your session.",
			"",
		)
	}

	for idx, action := range actions {
		marker := " "
		style := lipgloss.NewStyle().Foreground(theme.Muted)
		line := action.Label
		if m.isAuthenticated() {
			if action.ID == accountActionEditProfile {
				line += "  -  Update your username"
			}
			if action.ID == accountActionManualSync {
				line += "  -  Push local catalog and reading state now"
			}
			if action.ID == accountActionLogout {
				line += "  -  Disconnect this device session"
			}
		}
		if idx == selected {
			marker = m.renderSelectorMarker()
			style = lipgloss.NewStyle().Bold(true).Foreground(theme.Primary)
		}
		lines = append(lines, marker+" "+style.Render(line))
	}

	contentWidth := m.bodyContentWidth()
	if contentWidth <= 0 {
		contentWidth = 96
	}
	boxWidth := contentWidth
	if boxWidth > 92 {
		boxWidth = 92
	}
	if boxWidth < 56 {
		boxWidth = 56
	}
	body := lipgloss.NewStyle().
		Width(boxWidth).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))

	body = lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, body)

	hints := m.renderFooterHints([]footerHint{
		{key: "Tab", action: "next view"},
		{key: "Shift+Tab", action: "prev view"},
		{key: "↑/↓", action: "select"},
		{key: "Enter", action: "apply"},
	})
	status := m.renderStatusToast("Ready")
	footerLines := []string{"", m.renderFooterRow(hints, status)}
	body = m.centerBodyVertically(body, len(headerLines), len(footerLines))

	return m.renderPinnedLayout(
		headerLines,
		body,
		footerLines,
	)
}
