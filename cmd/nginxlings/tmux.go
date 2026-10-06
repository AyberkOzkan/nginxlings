package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

// nginxlings runs in a tmux session of its own: the app in the top pane and
// a shell in the selected exercise's folder in the bottom pane, so a learner
// edits and runs commands without opening another terminal.

const (
	// tmuxSocket names a tmux server apart from any the learner runs, so
	// their own tmux configuration and sessions are not touched.
	tmuxSocket = "nginxlings"
	// envInTmux marks the session nginxlings started; inside it the app
	// opens the terminal pane and ends the session when it quits.
	envInTmux = "NGINXLINGS_TMUX"
	envNoTmux = "NGINXLINGS_NO_TMUX"
	// minSplitHeight is the smallest window that gets a terminal pane at
	// start, so the app keeps room for its panels.
	minSplitHeight = 20
)

// tmuxConfig configures nginxlings' own tmux server.
const tmuxConfig = `# Written by nginxlings on every start; changes here are overwritten.
set -g mouse on
set -g status off
set -sg escape-time 10
set -g default-terminal "screen-256color"
# Closing the terminal window detaches its client: end the session then.
set-hook -g client-detached kill-session
set -g history-limit 5000
# Double lines frame the terminal in the colour of nginxlings' panels. The
# line above it is the bottom line of nginxlings' pane and names the keys
# (nginxlings' pane title); the line below it shows the terminal's folder.
# Their text turns orange while the terminal has the keyboard.
set -g pane-border-lines double
# The lines appear with the terminal pane and go when it closes, so a lone
# nginxlings pane has no empty line below it.
set -g pane-border-status off
set-hook -g after-split-window 'set -w pane-border-status bottom'
set-hook -g pane-exited 'if -F "#{==:#{window_panes},1}" "set -w pane-border-status off"'
set-hook -g after-kill-pane 'if -F "#{==:#{window_panes},1}" "set -w pane-border-status off"'
set -g pane-border-style "fg=colour60"
set -g pane-active-border-style "fg=colour60"
set -g pane-border-format "#{?#{==:#{pane_index},0},#{?#{>:#{window_panes},1},#{?pane_active,#[fg=colour103],#[fg=colour214]#[bold]} #{pane_title} ,},#{?pane_active,#[fg=colour214]#[bold],#[fg=colour103]} #{s|^{{HOME}}|~|:pane_current_path} }"
# F2 moves between nginxlings and the terminal. Alt+Up and Alt+Down make the
# terminal taller or shorter; dragging the line above it works too.
bind -n F2 select-pane -t :.+
bind -n M-Up resize-pane -t "{bottom}" -U 2
bind -n M-Down resize-pane -t "{bottom}" -D 2
# The mouse wheel scrolls nginxlings' instructions, not tmux history.
bind -n WheelUpPane if -F -t = "#{alternate_on}" "send-keys -t = Up Up Up" "copy-mode -e -t ="
bind -n WheelDownPane if -F -t = "#{alternate_on}" "send-keys -t = Down Down Down" "send-keys -M"
`

// tmuxEnv is what the tmux launcher reads; tests replace it.
type tmuxEnv struct {
	getenv   func(string) string
	lookPath func(string) (string, error)
	// terminal reports whether stdin and stdout are a terminal; width and
	// height are its size.
	terminal      bool
	width, height int
}

func systemTmuxEnv() tmuxEnv {
	env := tmuxEnv{
		getenv:   os.Getenv,
		lookPath: exec.LookPath,
		terminal: term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())),
	}
	env.width, env.height, _ = term.GetSize(int(os.Stdout.Fd()))
	return env
}

// tmuxLaunch holds the two commands that start nginxlings' tmux session:
// create builds it in the background at the terminal's size, so a tmux that
// cannot start leaves nginxlings running on its own; attach then shows it.
type tmuxLaunch struct {
	create, attach []string
}

// planTmux returns how to start nginxlings' tmux session, or false when
// nginxlings runs on its own: when told not to, inside tmux already, without
// a terminal, or without tmux.
func planTmux(opts cliOptions, env tmuxEnv, exe string, args []string, config string) (tmuxLaunch, bool) {
	if opts.noTmux || env.getenv(envNoTmux) != "" || env.getenv("TMUX") != "" ||
		!env.terminal || env.getenv("TERM") == "dumb" || env.width <= 0 || env.height <= 0 {
		return tmuxLaunch{}, false
	}
	tmux, err := env.lookPath("tmux")
	if err != nil {
		return tmuxLaunch{}, false
	}
	inner := envInTmux + "=1 exec " + shellQuote(exe)
	for _, arg := range args {
		inner += " " + shellQuote(arg)
	}
	session := fmt.Sprintf("nginxlings-%d", os.Getpid())
	// -u: nginxlings draws with UTF-8 even where the locale does not say
	// UTF-8; tmux would otherwise print its box lines and Turkish letters as _.
	return tmuxLaunch{
		create: []string{tmux, "-u", "-L", tmuxSocket, "-f", config, "new-session", "-d", "-s", session,
			"-x", fmt.Sprint(env.width), "-y", fmt.Sprint(env.height), inner},
		attach: []string{tmux, "-u", "-L", tmuxSocket, "attach-session", "-t", session},
	}, true
}

// plainHome matches a home directory that can go into the tmux format as it
// is; another one is shown in full instead of as ~.
var plainHome = regexp.MustCompile(`^/[A-Za-z0-9/_.-]+$`)

// renderTmuxConfig fills the learner's home directory into the
// configuration, so the terminal's folder reads ~/nginxlings/...
func renderTmuxConfig(home string) string {
	if !plainHome.MatchString(home) {
		home = "/nonexistent-home"
	}
	return strings.ReplaceAll(tmuxConfig, "{{HOME}}", strings.TrimRight(home, "/"))
}

// writeTmuxConfig writes the session configuration into the state directory.
func writeTmuxConfig(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	home, _ := os.UserHomeDir()
	path := filepath.Join(dir, "tmux.conf")
	return path, os.WriteFile(path, []byte(renderTmuxConfig(home)), 0o644)
}

// shellQuote quotes s for a POSIX shell.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// runTmux runs a tmux command against the server of the current session;
// tests replace it.
var runTmux = func(args ...string) (string, error) {
	out, err := exec.Command("tmux", args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// shells are the programs that may take a cd typed into the terminal pane.
var shells = map[string]bool{"bash": true, "sh": true, "dash": true, "ash": true, "zsh": true, "fish": true, "ksh": true, "mksh": true, "tcsh": true}

type terminalPaneMsg struct {
	pane string
	err  error
}

type shellDoneMsg struct{}

// openTerminalPane splits the window and starts a shell in dir below
// nginxlings. focus moves the cursor into it.
func openTerminalPane(dir string, height int, focus bool) tea.Cmd {
	return func() tea.Msg {
		rows := max(5, height*30/100)
		args := []string{"split-window", "-v", "-l", fmt.Sprint(rows), "-c", dir, "-P", "-F", "#{pane_id}"}
		if !focus {
			args = append(args, "-d")
		}
		pane, err := runTmux(args...)
		if err != nil {
			return terminalPaneMsg{err: fmt.Errorf("%v: %s", err, pane)}
		}
		return terminalPaneMsg{pane: pane}
	}
}

// moveTerminalTo types a cd to dir into the terminal pane and moves there.
// A program other than a shell keeps its input: nano would otherwise get the
// command as text.
func moveTerminalTo(pane, dir string) (busy string, err error) {
	command, err := runTmux("display-message", "-p", "-t", pane, "#{pane_current_command}")
	if err != nil {
		return "", err
	}
	if shells[filepath.Base(command)] {
		if _, err := runTmux("send-keys", "-t", pane, "C-u"); err != nil {
			return "", err
		}
		if _, err := runTmux("send-keys", "-t", pane, "-l", "cd -- "+shellQuote(dir)); err != nil {
			return "", err
		}
		if _, err := runTmux("send-keys", "-t", pane, "Enter"); err != nil {
			return "", err
		}
	} else {
		busy = command
	}
	_, err = runTmux("select-pane", "-t", pane)
	return busy, err
}

// runShell suspends nginxlings and runs the learner's shell in dir, for a
// terminal without tmux. Typing exit returns to nginxlings. Tests replace it.
var runShell = func(dir, hint string) tea.Cmd {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.Command("/bin/sh", "-c", `printf '%s\n' "$1"; exec "$0"`, shell, hint)
	cmd.Dir = dir
	return tea.ExecProcess(cmd, func(error) tea.Msg { return shellDoneMsg{} })
}
