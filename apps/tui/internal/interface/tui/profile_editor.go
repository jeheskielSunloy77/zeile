package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type profileEditorState struct {
	Username string
	Saving   bool
}

type profileUsernameUpdatedMsg struct {
	username string
	err      error
}

func (m *model) openProfileEditor() {
	username := ""
	if m.container != nil && m.container.Auth != nil {
		if session, ok := m.container.Auth.Session(); ok {
			username = strings.TrimSpace(session.User.Username)
		}
	}

	m.profileEditor = &profileEditorState{
		Username: username,
	}
}

func (m *model) closeProfileEditor() {
	m.profileEditor = nil
}

func (m *model) handleProfileEditorKey(msg tea.KeyMsg) tea.Cmd {
	if m.profileEditor == nil {
		return nil
	}

	if m.profileEditor.Saving {
		return nil
	}

	switch msg.String() {
	case "esc", "q":
		m.closeProfileEditor()
		return nil
	case "enter":
		username := strings.TrimSpace(m.profileEditor.Username)
		if len(username) < 3 || len(username) > 50 {
			m.setStatusDestructive("Username must be 3-50 characters")
			return nil
		}
		m.profileEditor.Saving = true
		return m.updateUsernameCmd(username)
	case "backspace":
		if len(m.profileEditor.Username) > 0 {
			runes := []rune(m.profileEditor.Username)
			m.profileEditor.Username = string(runes[:len(runes)-1])
		}
		return nil
	}

	if len(msg.Runes) > 0 {
		m.profileEditor.Username += string(msg.Runes)
	}

	return nil
}

func (m *model) updateUsernameCmd(username string) tea.Cmd {
	return func() tea.Msg {
		if m.container == nil || m.container.Auth == nil {
			return profileUsernameUpdatedMsg{err: fmt.Errorf("auth service unavailable")}
		}

		user, err := m.container.Auth.UpdateUsername(context.Background(), username)
		if err != nil {
			return profileUsernameUpdatedMsg{err: err}
		}

		return profileUsernameUpdatedMsg{username: user.Username}
	}
}
