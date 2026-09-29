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
	paths, err := ResolveAppPathsWithOptions(opts.root, opts.state)
	if err != nil {
		return err
	}

	m := initialModelWithOptions(paths, opts.lang)
	// Query the background before Bubble Tea owns stdin; a terminal query
	// from inside the program would race its input reader.
	m.lightBackground = !lipgloss.HasDarkBackground()

	p := tea.NewProgram(m, tea.WithAltScreen())
	if err := p.Start(); err != nil {
		return fmt.Errorf("Hata: %w", err)
	}
	return nil
}
