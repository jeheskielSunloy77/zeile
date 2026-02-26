package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zeile/tui/internal/application"
)

type syncTickMsg struct{}

type syncDoneMsg struct {
	triggeredByUser bool
	result          application.SyncResult
	err             error
}

func waitSyncTick(interval time.Duration) tea.Cmd {
	if interval <= 0 {
		interval = 2 * time.Minute
	}
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return syncTickMsg{}
	})
}

func (m *model) syncNowCmd(triggeredByUser bool) tea.Cmd {
	return func() tea.Msg {
		if m.container == nil || m.container.Sync == nil {
			return syncDoneMsg{triggeredByUser: triggeredByUser, err: fmt.Errorf("sync service unavailable")}
		}
		result, err := m.container.Sync.ReconcileNow(context.Background())
		return syncDoneMsg{
			triggeredByUser: triggeredByUser,
			result:          result,
			err:             err,
		}
	}
}
