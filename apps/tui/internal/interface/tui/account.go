package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type accountActionID int

const (
	accountActionLogin accountActionID = iota
	accountActionEditProfile
	accountActionManualSync
	accountActionLogout
)

type accountAction struct {
	ID    accountActionID
	Label string
}

func (m model) isAuthenticated() bool {
	return strings.HasPrefix(strings.TrimSpace(m.connectionLabel), "Connected")
}

func (m model) accountNavLabel() string {
	if m.isAuthenticated() {
		return "Account"
	}
	return "Login"
}

func (m model) accountActions() []accountAction {
	if !m.isAuthenticated() {
		return []accountAction{
			{ID: accountActionLogin, Label: "Login"},
		}
	}

	return []accountAction{
		{ID: accountActionEditProfile, Label: "Edit profile"},
		{ID: accountActionManualSync, Label: "Manual sync"},
		{ID: accountActionLogout, Label: "Logout"},
	}
}

func (m *model) normalizeAccountSelection() {
	actions := m.accountActions()
	if len(actions) == 0 {
		m.accountField = 0
		return
	}
	if m.accountField < 0 {
		m.accountField = 0
	}
	if m.accountField >= len(actions) {
		m.accountField = len(actions) - 1
	}
}

func (m *model) handleAccountKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "q":
		return tea.Quit
	case "up", "k":
		m.normalizeAccountSelection()
		if m.accountField > 0 {
			m.accountField--
		}
		return nil
	case "down", "j":
		m.normalizeAccountSelection()
		actions := m.accountActions()
		if m.accountField < len(actions)-1 {
			m.accountField++
		}
		return nil
	case "enter":
		m.normalizeAccountSelection()
		actions := m.accountActions()
		if len(actions) == 0 {
			return nil
		}

		switch actions[m.accountField].ID {
		case accountActionLogin:
			return m.startDeviceAuthCmd()
		case accountActionEditProfile:
			m.openProfileEditor()
			return nil
		case accountActionManualSync:
			if !m.shouldRunSync() {
				m.setStatusDefault("Connect first to run sync")
				return nil
			}
			if m.syncing {
				m.setStatusDefault("Sync already in progress")
				return nil
			}
			m.syncing = true
			return m.syncNowCmd(true)
		case accountActionLogout:
			return m.disconnectCmd()
		}
	case "?":
		if m.isAuthenticated() {
			m.setStatusDefault("Account: Tab/Shift+Tab switch views  Up/Down select  Enter apply")
		} else {
			m.setStatusDefault("Login: Tab/Shift+Tab switch views  Enter login")
		}
	}

	return nil
}
