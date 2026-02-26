package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type settingsFieldView struct {
	Label    string
	Value    string
	Action   bool
	Disabled bool
}

func (m model) renderSettings() string {
	header := m.renderMainNavHeader(viewSettings)
	subheader := "Global settings (live apply, auto-save)"

	bodyWidth := m.bodyContentWidth()
	if bodyWidth <= 0 {
		bodyWidth = 96
	}
	body := m.renderSettingsBody(bodyWidth)
	hints := m.renderFooterHints([]footerHint{
		{key: "Tab", action: "next view"},
		{key: "Shift+Tab", action: "prev view"},
		{key: "[/]", action: "section"},
		{key: "↑/↓", action: "field"},
		{key: "←/→", action: "adjust"},
		{key: "Enter", action: "apply"},
		{key: "r", action: "reset section"},
		{key: "Esc", action: "back"},
	})
	status := m.renderStatusToast("Ready")
	footerLines := []string{"", m.renderFooterRow(hints, status)}
	headerLines := []string{header, subheader, ""}

	body = m.centerBodyVertically(body, len(headerLines), len(footerLines))

	return m.renderPinnedLayout(
		headerLines,
		body,
		footerLines,
	)
}

func (m model) centerBodyVertically(body string, headerLines, footerLines int) string {
	if m.height <= 0 {
		return body
	}

	contentHeight := m.mainLayoutHeight() - headerLines - footerLines
	if contentHeight <= 0 {
		return body
	}

	lines := strings.Split(body, "\n")
	if len(lines) >= contentHeight {
		return body
	}

	topPad := (contentHeight - len(lines)) / 2
	if topPad <= 0 {
		return body
	}

	padded := make([]string, 0, topPad+len(lines))
	for i := 0; i < topPad; i++ {
		padded = append(padded, "")
	}
	padded = append(padded, lines...)
	return strings.Join(padded, "\n")
}

func (m model) renderSettingsBody(contentWidth int) string {
	theme := m.activeTheme()
	if contentWidth < 56 {
		contentWidth = 56
	}

	const paneFrameExtra = 4 // rounded border + horizontal padding(1,1)
	const joinExtra = 3      // " " + "│" + " "
	leftWidth := 22
	if contentWidth < 80 {
		leftWidth = 18
	}
	rightWidth := contentWidth - leftWidth - (paneFrameExtra*2 + joinExtra)
	if rightWidth < 24 {
		rightWidth = 24
		leftWidth = contentWidth - rightWidth - (paneFrameExtra*2 + joinExtra)
	}
	if leftWidth < 14 {
		leftWidth = 14
	}

	leftPane := m.renderSettingsSectionList(leftWidth)
	rightPane := m.renderSettingsFields(rightWidth)

	separator := lipgloss.NewStyle().Foreground(theme.Divider).Render("│")
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " ", separator, " ", rightPane)
}

func (m model) renderSettingsSectionList(width int) string {
	theme := m.activeTheme()
	lines := make([]string, 0, len(settingsSections)+2)
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Sections"), "")
	for _, section := range settingsSections {
		label := settingsSectionLabel(section)
		marker := " "
		style := lipgloss.NewStyle()
		if section == m.settingsSection {
			marker = m.renderSelectorMarker()
			style = style.Bold(true).Foreground(theme.Primary)
		}
		line := fmt.Sprintf("%s %s", marker, label)
		lines = append(lines, style.Render(line))
	}

	return lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		Render(strings.Join(lines, "\n"))
}

func (m model) renderSettingsFields(width int) string {
	theme := m.activeTheme()
	fields := m.settingsFieldsForSection(m.settingsSection)
	if len(fields) == 0 {
		return lipgloss.NewStyle().Width(width).Render("No settings")
	}

	if m.settingsField >= len(fields) {
		m.settingsField = len(fields) - 1
	}
	if m.settingsField < 0 {
		m.settingsField = 0
	}

	lines := make([]string, 0, len(fields)+3)
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render(settingsSectionLabel(m.settingsSection)), "")
	labelWidth := 24
	if width < 40 {
		labelWidth = 16
	}
	valueWidth := width - labelWidth - 5
	if valueWidth < 8 {
		valueWidth = 8
	}

	for i, field := range fields {
		marker := " "
		style := lipgloss.NewStyle()
		valueStyle := lipgloss.NewStyle().Foreground(theme.Muted)
		if i == m.settingsField {
			marker = m.renderSelectorMarker()
			style = style.Bold(true).Foreground(theme.PrimaryAlt)
			valueStyle = valueStyle.Foreground(theme.PrimaryAlt)
		}
		if field.Disabled {
			style = style.Faint(true)
			valueStyle = valueStyle.Faint(true)
		}

		label := truncateRunes(field.Label, labelWidth)
		value := truncateRunes(field.Value, valueWidth)
		labelText := style.Render(padRight(label, labelWidth))
		valueText := valueStyle.Render(value)
		lines = append(lines, fmt.Sprintf("%s %s %s", marker, labelText, valueText))
	}

	if m.settingsSection == settingsSectionAdvanced {
		pathValue := truncateRunes(m.settingsTransferPath(), width-8)
		lines = append(lines, "", lipgloss.NewStyle().Faint(true).Render("Path: "+pathValue))
	}

	return lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		Render(strings.Join(lines, "\n"))
}

func (m model) settingsFieldsForSection(section settingsSectionID) []settingsFieldView {
	cfg := m.currentConfig()

	switch section {
	case settingsSectionTheme:
		colorValue := cfg.PrimaryOverrideColor
		if !cfg.PrimaryOverrideEnabled {
			colorValue = colorValue + " (disabled)"
		}
		return []settingsFieldView{
			{Label: "Theme pack", Value: titleCaseToken(cfg.ThemePack)},
			{Label: "Primary override", Value: boolLabel(cfg.PrimaryOverrideEnabled)},
			{Label: "Primary color", Value: colorValue, Disabled: !cfg.PrimaryOverrideEnabled},
			{Label: "Reset theme", Value: "Enter", Action: true},
		}
	case settingsSectionReading:
		return []settingsFieldView{
			{Label: "Content width", Value: fmt.Sprintf("%d", cfg.ContentWidth)},
			{Label: "Horizontal margin", Value: fmt.Sprintf("%d", cfg.MarginHorizontal)},
			{Label: "Line spacing", Value: fmt.Sprintf("%d", cfg.LineSpacing)},
			{Label: "Paragraph spacing", Value: fmt.Sprintf("%d", cfg.ParagraphSpacing)},
			{Label: "Spread threshold", Value: fmt.Sprintf("%d", cfg.SpreadThreshold)},
		}
	case settingsSectionBehavior:
		return []settingsFieldView{
			{Label: "Startup mode", Value: titleCaseToken(cfg.StartupMode)},
			{Label: "Managed copy default", Value: boolLabel(cfg.ManagedCopyDefault)},
			{Label: "Delete confirmation", Value: boolLabel(cfg.DeleteConfirmation)},
			{Label: "Key hints density", Value: titleCaseToken(cfg.KeyHintsDensity)},
		}
	case settingsSectionAccessibility:
		return []settingsFieldView{
			{Label: "High contrast", Value: boolLabel(cfg.HighContrast)},
			{Label: "Highlight style", Value: titleCaseToken(cfg.HighlightStyle)},
		}
	case settingsSectionAdvanced:
		return []settingsFieldView{
			{Label: "Export settings", Value: "Enter", Action: true},
			{Label: "Import settings", Value: "Enter", Action: true},
			{Label: "Reset reading", Value: "Enter", Action: true},
			{Label: "Reset all", Value: "Enter", Action: true},
		}
	default:
		return nil
	}
}

func settingsSectionLabel(section settingsSectionID) string {
	switch section {
	case settingsSectionTheme:
		return "Theme"
	case settingsSectionReading:
		return "Reading Layout"
	case settingsSectionBehavior:
		return "Behavior"
	case settingsSectionAccessibility:
		return "Accessibility"
	case settingsSectionAdvanced:
		return "Advanced"
	default:
		return "Unknown"
	}
}

func padRight(value string, width int) string {
	current := lipgloss.Width(value)
	if current >= width {
		return value
	}
	return value + strings.Repeat(" ", width-current)
}
