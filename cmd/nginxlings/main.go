package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr, runExerciseCheck, startBubbleTea))
}

func startBubbleTea(opts cliOptions) error {
	m, err := newTUIModel(opts)
	if err != nil {
		return err
	}
	// Query the background before Bubble Tea owns stdin; a terminal query
	// from inside the program would race its input reader.
	m.lightBackground = !lipgloss.HasDarkBackground()

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run TUI: %w", err)
	}
	return nil
}
