package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr, runExerciseCheck, startBubbleTea))
}

func startBubbleTea(opts cliOptions) error {
	paths, err := ResolveAppPathsWithOptions(opts.root, opts.state)
	if err != nil {
		return err
	}

	p := tea.NewProgram(initialModelWithOptions(paths, opts.lang), tea.WithAltScreen())
	if err := p.Start(); err != nil {
		return fmt.Errorf("Hata: %w", err)
	}
	return nil
}
