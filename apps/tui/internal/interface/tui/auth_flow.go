package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zeile/tui/internal/infrastructure/remote"
)

type deviceAuthState struct {
	DeviceCode      string
	UserCode        string
	VerificationURI string
	ExpiresAt       time.Time
	Interval        time.Duration
}

type deviceAuthStartMsg struct {
	start remote.DeviceAuthStartResponse
	err   error
}

type deviceAuthPollMsg struct {
	result remote.DeviceAuthPollResponse
	err    error
}

type deviceAuthPollTickMsg struct{}

type authDisconnectedMsg struct {
	err error
}

func (m *model) startDeviceAuthCmd() tea.Cmd {
	return func() tea.Msg {
		if m.container == nil || m.container.Auth == nil {
			return deviceAuthStartMsg{err: fmt.Errorf("auth service unavailable")}
		}
		start, err := m.container.Auth.StartDeviceAuth(context.Background())
		return deviceAuthStartMsg{start: start, err: err}
	}
}

func waitDeviceAuthPoll(interval time.Duration) tea.Cmd {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return deviceAuthPollTickMsg{}
	})
}

func (m *model) pollDeviceAuthCmd() tea.Cmd {
	return func() tea.Msg {
		if m.deviceAuth == nil {
			return deviceAuthPollMsg{}
		}
		if m.container == nil || m.container.Auth == nil {
			return deviceAuthPollMsg{err: fmt.Errorf("auth service unavailable")}
		}
		result, err := m.container.Auth.PollDeviceAuth(context.Background(), m.deviceAuth.DeviceCode)
		return deviceAuthPollMsg{result: result, err: err}
	}
}

func (m *model) disconnectCmd() tea.Cmd {
	return func() tea.Msg {
		if m.container == nil || m.container.Auth == nil {
			return authDisconnectedMsg{}
		}
		err := m.container.Auth.Disconnect(context.Background())
		return authDisconnectedMsg{err: err}
	}
}
