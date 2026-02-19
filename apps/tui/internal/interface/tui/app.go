package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zeile/tui/internal/application"
)

type model struct {
	container   *application.Container
	width       int
	height      int
	startupInfo string
}

func New(container *application.Container) tea.Model {
	m := model{container: container}
	if container != nil && container.Library != nil {
		books, err := container.Library.ListBooks(context.Background())
		if err != nil {
			m.startupInfo = fmt.Sprintf("Library unavailable: %v", err)
		} else {
			m.startupInfo = fmt.Sprintf("Library ready: %d book(s)", len(books))
		}
	}
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	title := lipgloss.NewStyle().Bold(true).Render("Zeile TUI MVP (V1)")
	body := strings.Join([]string{
		title,
		"",
		"Scaffold ready. MVP features are being implemented.",
		m.startupInfo,
		"",
		"Press q to quit.",
	}, "\n")

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
	}

	return body
}
