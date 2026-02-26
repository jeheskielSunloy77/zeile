package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zeile/tui/internal/infrastructure/config"
)

type settingsSectionID int

const (
	settingsSectionTheme settingsSectionID = iota
	settingsSectionReading
	settingsSectionBehavior
	settingsSectionAccessibility
	settingsSectionAdvanced
)

var settingsSections = []settingsSectionID{
	settingsSectionTheme,
	settingsSectionReading,
	settingsSectionBehavior,
	settingsSectionAccessibility,
	settingsSectionAdvanced,
}

var primaryOverridePalette = []string{"33", "39", "81", "117", "172", "205", "220"}

func (m *model) openSettings(from viewID) {
	m.settingsReturnView = from
	m.currentView = viewSettings
	m.settingsSection = settingsSectionTheme
	m.settingsField = 0
}

func (m *model) closeSettings() {
	target := m.settingsReturnView
	if target != viewReader && target != viewLibrary && target != viewAdd && target != viewCommunities && target != viewAccount {
		target = viewLibrary
	}
	m.currentView = target
	if target == viewReader && m.isReaderTextMode() {
		anchor := m.readerAnchorOffset()
		m.repaginateReader(anchor)
	}
}

func (m *model) handleSettingsKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "q":
		m.closeSettings()
		return nil
	case "]":
		m.stepSettingsSection(1)
		return nil
	case "[":
		m.stepSettingsSection(-1)
		return nil
	case "up", "k":
		if m.settingsField > 0 {
			m.settingsField--
		}
		return nil
	case "down", "j":
		maxField := m.settingsFieldCount(m.settingsSection) - 1
		if maxField < 0 {
			maxField = 0
		}
		if m.settingsField < maxField {
			m.settingsField++
		}
		return nil
	case "left", "h":
		return m.adjustCurrentSetting(-1)
	case "right", "l":
		return m.adjustCurrentSetting(1)
	case "enter":
		return m.activateCurrentSetting()
	case "r":
		return m.resetSettingsSection(m.settingsSection)
	}

	return nil
}

func (m *model) settingsFieldCount(section settingsSectionID) int {
	switch section {
	case settingsSectionTheme:
		return 4
	case settingsSectionReading:
		return 5
	case settingsSectionBehavior:
		return 4
	case settingsSectionAccessibility:
		return 2
	case settingsSectionAdvanced:
		return 4
	default:
		return 0
	}
}

func (m *model) stepSettingsSection(delta int) {
	idx := 0
	for i, section := range settingsSections {
		if section == m.settingsSection {
			idx = i
			break
		}
	}
	idx += delta
	if idx < 0 {
		idx = len(settingsSections) - 1
	}
	if idx >= len(settingsSections) {
		idx = 0
	}
	m.settingsSection = settingsSections[idx]
	m.settingsField = 0
}

func (m *model) adjustCurrentSetting(delta int) tea.Cmd {
	if delta == 0 {
		return nil
	}

	cfg := m.currentConfig()
	before := cfg

	switch m.settingsSection {
	case settingsSectionTheme:
		switch m.settingsField {
		case 0:
			cfg.ThemePack = cycleEnum(cfg.ThemePack, []string{
				config.ThemePackDefault,
				config.ThemePackDracula,
				config.ThemePackGruvbox,
				config.ThemePackNord,
				config.ThemePackCatppuccin,
			}, delta)
		case 1:
			cfg.PrimaryOverrideEnabled = !cfg.PrimaryOverrideEnabled
		case 2:
			if cfg.PrimaryOverrideEnabled {
				cfg.PrimaryOverrideColor = cycleEnum(cfg.PrimaryOverrideColor, primaryOverridePalette, delta)
			}
		}
	case settingsSectionReading:
		switch m.settingsField {
		case 0:
			cfg.ContentWidth += 2 * delta
		case 1:
			cfg.MarginHorizontal += delta
		case 2:
			cfg.LineSpacing += delta
		case 3:
			cfg.ParagraphSpacing += delta
		case 4:
			cfg.SpreadThreshold += 2 * delta
			cfg.MinSpreadWidth = cfg.SpreadThreshold
		}
	case settingsSectionBehavior:
		switch m.settingsField {
		case 0:
			cfg.StartupMode = cycleEnum(cfg.StartupMode, []string{config.StartupModeResume, config.StartupModeLibrary}, delta)
		case 1:
			cfg.ManagedCopyDefault = !cfg.ManagedCopyDefault
		case 2:
			cfg.DeleteConfirmation = !cfg.DeleteConfirmation
		case 3:
			cfg.KeyHintsDensity = cycleEnum(cfg.KeyHintsDensity, []string{
				config.KeyHintsDensityFull,
				config.KeyHintsDensityCompact,
				config.KeyHintsDensityHidden,
			}, delta)
		}
	case settingsSectionAccessibility:
		switch m.settingsField {
		case 0:
			cfg.HighContrast = !cfg.HighContrast
		case 1:
			cfg.HighlightStyle = cycleEnum(cfg.HighlightStyle, []string{
				config.HighlightStyleUnderline,
				config.HighlightStyleReverse,
				config.HighlightStyleBlock,
			}, delta)
		}
	}

	return m.applySettingsUpdate(before, cfg)
}

func (m *model) activateCurrentSetting() tea.Cmd {
	switch m.settingsSection {
	case settingsSectionTheme:
		if m.settingsField == 3 {
			return m.resetThemeSettings()
		}
		return m.adjustCurrentSetting(1)
	case settingsSectionReading, settingsSectionBehavior, settingsSectionAccessibility:
		return m.adjustCurrentSetting(1)
	case settingsSectionAdvanced:
		switch m.settingsField {
		case 0:
			if err := m.exportSettings(); err != nil {
				m.setStatusDestructive(fmt.Sprintf("Export failed: %v", err))
				return nil
			}
			m.setStatusSuccess(fmt.Sprintf("Exported settings to %s", m.settingsTransferPath()))
			return nil
		case 1:
			before := m.currentConfig()
			imported, err := config.LoadFile(m.settingsTransferPath())
			if err != nil {
				m.setStatusDestructive(fmt.Sprintf("Import failed: %v", err))
				return nil
			}
			after := config.MergeSettings(before, imported)
			cmd := m.applySettingsUpdate(before, after)
			if cmd != nil {
				m.setStatusSuccess(fmt.Sprintf("Imported settings from %s", m.settingsTransferPath()))
			}
			return cmd
		case 2:
			return m.resetReadingSettings()
		case 3:
			return m.resetAllSettings()
		}
	}

	return nil
}

func (m *model) resetSettingsSection(section settingsSectionID) tea.Cmd {
	switch section {
	case settingsSectionTheme:
		return m.resetThemeSettings()
	case settingsSectionReading:
		return m.resetReadingSettings()
	case settingsSectionBehavior:
		return m.resetBehaviorSettings()
	case settingsSectionAccessibility:
		return m.resetAccessibilitySettings()
	case settingsSectionAdvanced:
		return nil
	default:
		return nil
	}
}

func (m *model) resetThemeSettings() tea.Cmd {
	before := m.currentConfig()
	after := before
	d := config.Default()
	after.ThemePack = d.ThemePack
	after.PrimaryOverrideEnabled = d.PrimaryOverrideEnabled
	after.PrimaryOverrideColor = d.PrimaryOverrideColor
	m.setStatusSuccess("Reset theme settings")
	return m.applySettingsUpdate(before, after)
}

func (m *model) resetReadingSettings() tea.Cmd {
	before := m.currentConfig()
	after := before
	d := config.Default()
	after.ContentWidth = d.ContentWidth
	after.MarginHorizontal = d.MarginHorizontal
	after.LineSpacing = d.LineSpacing
	after.ParagraphSpacing = d.ParagraphSpacing
	after.SpreadThreshold = d.SpreadThreshold
	after.MinSpreadWidth = d.SpreadThreshold
	m.setStatusSuccess("Reset reading layout settings")
	return m.applySettingsUpdate(before, after)
}

func (m *model) resetBehaviorSettings() tea.Cmd {
	before := m.currentConfig()
	after := before
	d := config.Default()
	after.StartupMode = d.StartupMode
	after.ManagedCopyDefault = d.ManagedCopyDefault
	after.DeleteConfirmation = d.DeleteConfirmation
	after.KeyHintsDensity = d.KeyHintsDensity
	m.setStatusSuccess("Reset behavior settings")
	return m.applySettingsUpdate(before, after)
}

func (m *model) resetAccessibilitySettings() tea.Cmd {
	before := m.currentConfig()
	after := before
	d := config.Default()
	after.HighContrast = d.HighContrast
	after.HighlightStyle = d.HighlightStyle
	m.setStatusSuccess("Reset accessibility settings")
	return m.applySettingsUpdate(before, after)
}

func (m *model) resetAllSettings() tea.Cmd {
	before := m.currentConfig()
	after := config.MergeSettings(before, config.Default())
	m.setStatusSuccess("Reset all settings")
	return m.applySettingsUpdate(before, after)
}

func (m *model) applySettingsUpdate(before, next config.Config) tea.Cmd {
	next = next.Normalized()
	if before == next {
		return nil
	}

	layoutChanged := before.ContentWidth != next.ContentWidth ||
		before.MarginHorizontal != next.MarginHorizontal ||
		before.LineSpacing != next.LineSpacing ||
		before.ParagraphSpacing != next.ParagraphSpacing ||
		before.SpreadThreshold != next.SpreadThreshold

	m.applyConfig(next)
	if layoutChanged && m.readerBook.ID != "" && m.isReaderTextMode() {
		anchor := m.readerAnchorOffset()
		m.repaginateReader(anchor)
	}

	return m.queueSettingsSaveCmd()
}

func (m *model) applyConfig(cfg config.Config) {
	if m.container == nil {
		return
	}
	m.container.Config = cfg.Normalized()
	m.addManagedCopy = m.container.Config.ManagedCopyDefault
}

func (m model) currentConfig() config.Config {
	if m.container == nil {
		return config.Default()
	}
	return m.container.Config.Normalized()
}

func (m *model) queueSettingsSaveCmd() tea.Cmd {
	m.settingsSaveSeq++
	seq := m.settingsSaveSeq
	return tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg {
		return settingsSaveMsg{sequence: seq}
	})
}

func (m *model) persistSettingsConfig() error {
	if m.container == nil {
		return nil
	}
	path := strings.TrimSpace(m.container.Paths.ConfigPath)
	if path == "" {
		return nil
	}
	return config.Save(path, m.container.Config)
}

func (m model) settingsTransferPath() string {
	if m.container == nil {
		return "settings-export.toml"
	}
	base := strings.TrimSpace(m.container.Config.DataDir)
	if base == "" {
		if strings.TrimSpace(m.container.Paths.ConfigPath) != "" {
			base = filepath.Dir(m.container.Paths.ConfigPath)
		} else {
			base = "."
		}
	}
	return filepath.Join(base, "settings-export.toml")
}

func (m *model) exportSettings() error {
	return config.Save(m.settingsTransferPath(), m.currentConfig())
}

func cycleEnum(current string, values []string, delta int) string {
	if len(values) == 0 {
		return current
	}
	idx := 0
	for i, value := range values {
		if value == current {
			idx = i
			break
		}
	}
	idx += delta
	for idx < 0 {
		idx += len(values)
	}
	idx %= len(values)
	return values[idx]
}

func boolLabel(value bool) string {
	if value {
		return "On"
	}
	return "Off"
}
