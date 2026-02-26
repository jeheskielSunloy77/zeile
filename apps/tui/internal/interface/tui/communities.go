package tui

import tea "github.com/charmbracelet/bubbletea"

func (m *model) handleCommunitiesKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "q":
		return tea.Quit
	case "s":
		m.openSettings(viewCommunities)
	case "?":
		m.setStatusDefault("Communities: Tab/Shift+Tab switch views  s settings  q quit")
	}

	return nil
}
