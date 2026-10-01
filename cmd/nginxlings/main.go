package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
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

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithOutput(homeCursorOutput{os.Stdout}))
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

// homeCursorOutput is the program's output. It moves the cursor back to the
// top-left corner after every frame Bubble Tea draws in the alternate screen.
// Bubble Tea leaves the cursor on the last row, and tmux shrinks a pane whose
// cursor is on its last row by deleting rows from the top: making the
// terminal pane taller hid the title and made the screen jump until the next
// frame. With the cursor at the top, tmux deletes rows from the bottom, which
// the next frame redraws. Every alternate-screen frame starts with a move to
// the top-left corner, so the extra move changes nothing else. Embedding the
// *os.File keeps Bubble Tea treating the output as a terminal.
type homeCursorOutput struct{ *os.File }

func (o homeCursorOutput) Write(p []byte) (int, error) {
	if !bytes.HasPrefix(p, []byte(xansi.CursorHomePosition)) {
		return o.File.Write(p)
	}
	n, err := o.File.Write(append(p[:len(p):len(p)], xansi.CursorHomePosition...))
	return min(n, len(p)), err
}
