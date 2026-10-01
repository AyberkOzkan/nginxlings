package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

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
	// With the options checked, hand over to nginxlings' own tmux session,
	// which runs this program again in its top pane. If tmux cannot start,
	// run without it.
	if root, err := resolveStateRoot(os.Getenv); err == nil {
		if config, err := writeTmuxConfig(root); err == nil {
			exe, _ := os.Executable()
			if launch, ok := planTmux(opts, systemTmuxEnv(), exe, os.Args[1:], config); ok {
				if exec.Command(launch.create[0], launch.create[1:]...).Run() == nil {
					_ = syscall.Exec(launch.attach[0], launch.attach, os.Environ())
				}
			}
		}
	}
	// Query the background before Bubble Tea owns stdin; a terminal query
	// from inside the program would race its input reader.
	m.lightBackground = !lipgloss.HasDarkBackground()

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	// Quitting ends nginxlings' own tmux session, terminal pane included,
	// and returns to where the learner started.
	if m.ownTmux {
		_ = exec.Command("tmux", "kill-session").Run()
	}
	if err != nil {
		return fmt.Errorf("run TUI: %w", err)
	}
	return nil
}
