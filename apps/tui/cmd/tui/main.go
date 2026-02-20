package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zeile/tui/internal/application"
	"github.com/zeile/tui/internal/interface/tui"
)

func main() {
	container, err := application.NewContainer(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize app: %v\n", err)
		os.Exit(1)
	}
	defer container.Close()

	program := tea.NewProgram(tui.New(container), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start tui: %v\n", err)
		os.Exit(1)
	}
}
