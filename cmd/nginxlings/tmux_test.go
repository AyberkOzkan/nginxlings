package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func fakeTmuxEnv(vars map[string]string, haveTmux bool) tmuxEnv {
	return tmuxEnv{
		getenv: func(key string) string { return vars[key] },
		lookPath: func(name string) (string, error) {
			if name == "tmux" && haveTmux {
				return "/usr/bin/tmux", nil
			}
			return "", errors.New("not found")
		},
		terminal: true,
		width:    100,
		height:   30,
	}
}

func TestPlanTmuxStartsItsOwnSessionOnlyWhenItCan(t *testing.T) {
	launch, ok := planTmux(cliOptions{}, fakeTmuxEnv(nil, true), "/usr/bin/nginxlings", []string{"--lang", "tr"}, "/state/tmux.conf")
	if !ok {
		t.Fatal("planTmux() refused a terminal with tmux")
	}
	session := fmt.Sprintf("nginxlings-%d", os.Getpid())
	wantCreate := []string{"/usr/bin/tmux", "-u", "-L", "nginxlings", "-f", "/state/tmux.conf", "new-session", "-d", "-s", session,
		"-x", "100", "-y", "30", "NGINXLINGS_TMUX=1 exec '/usr/bin/nginxlings' '--lang' 'tr'"}
	if strings.Join(launch.create, "|") != strings.Join(wantCreate, "|") {
		t.Fatalf("create = %q\nwant     %q", launch.create, wantCreate)
	}
	if want := []string{"/usr/bin/tmux", "-u", "-L", "nginxlings", "attach-session", "-t", session}; strings.Join(launch.attach, "|") != strings.Join(want, "|") {
		t.Fatalf("attach = %q, want %q", launch.attach, want)
	}

	refusals := map[string]func() (cliOptions, tmuxEnv){
		"--no-tmux": func() (cliOptions, tmuxEnv) { return cliOptions{noTmux: true}, fakeTmuxEnv(nil, true) },
		"NGINXLINGS_NO_TMUX": func() (cliOptions, tmuxEnv) {
			return cliOptions{}, fakeTmuxEnv(map[string]string{envNoTmux: "1"}, true)
		},
		"inside tmux": func() (cliOptions, tmuxEnv) {
			return cliOptions{}, fakeTmuxEnv(map[string]string{"TMUX": "/tmp/x,1,0"}, true)
		},
		"TERM=dumb": func() (cliOptions, tmuxEnv) {
			return cliOptions{}, fakeTmuxEnv(map[string]string{"TERM": "dumb"}, true)
		},
		"no tmux": func() (cliOptions, tmuxEnv) { return cliOptions{}, fakeTmuxEnv(nil, false) },
		"no terminal": func() (cliOptions, tmuxEnv) {
			env := fakeTmuxEnv(nil, true)
			env.terminal = false
			return cliOptions{}, env
		},
		"unknown size": func() (cliOptions, tmuxEnv) {
			env := fakeTmuxEnv(nil, true)
			env.width = 0
			return cliOptions{}, env
		},
	}
	for name, setup := range refusals {
		opts, env := setup()
		if _, ok := planTmux(opts, env, "/usr/bin/nginxlings", nil, "/state/tmux.conf"); ok {
			t.Errorf("%s: planTmux() started tmux", name)
		}
	}
}

func TestRenderTmuxConfigShortensTheHomeDirectory(t *testing.T) {
	if got := renderTmuxConfig("/home/learner/"); !strings.Contains(got, "#{s|^/home/learner|~|:pane_current_path}") || strings.Contains(got, "{{HOME}}") {
		t.Fatalf("config does not shorten /home/learner:\n%s", got)
	}
	// A home the format cannot hold as it is stays in full.
	if got := renderTmuxConfig("/home/a|b"); strings.Contains(got, "a|b") || !strings.Contains(got, "^/nonexistent-home|") {
		t.Fatal("an unusual home directory went into the format")
	}
}

func TestShellQuote(t *testing.T) {
	if got := shellQuote("/home/it's mine"); got != `'/home/it'\''s mine'` {
		t.Fatalf("shellQuote() = %s", got)
	}
}

// TestTmuxConfigLoads loads the configuration into a real tmux server. A
// parse error makes tmux apply none of it: {bottom} without quotes did that.
func TestTmuxConfigLoads(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux is not installed")
	}
	dir, err := os.MkdirTemp("", "nlx") // short: tmux socket paths are limited
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	config, err := writeTmuxConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	tmux := func(args ...string) string {
		cmd := exec.Command("tmux", append([]string{"-L", "configtest"}, args...)...)
		cmd.Env = append(os.Environ(), "TMUX_TMPDIR="+dir, "TMUX=")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("tmux %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	tmux("-f", config, "new-session", "-d", "-s", "probe", "-x", "80", "-y", "24", "sleep 30")
	defer exec.Command("tmux", "-L", "configtest", "kill-server").Run()

	if got := tmux("show-options", "-g", "status"); strings.TrimSpace(got) != "status off" {
		t.Fatalf("status option = %q: the configuration was not applied", got)
	}
	if got := tmux("show-options", "-g", "pane-border-lines"); strings.TrimSpace(got) != "pane-border-lines double" {
		t.Fatalf("pane-border-lines = %q, want double", got)
	}
	if got := tmux("show-hooks", "-g"); !strings.Contains(got, "kill-session") {
		t.Fatalf("hooks = %q, want the client-detached kill-session hook", got)
	}
	// The framing lines come with the terminal pane and go with it.
	tmux("split-window", "-v", "-l", "6", "sleep 1")
	if got := tmux("show-options", "-wv", "-t", "probe", "pane-border-status"); strings.TrimSpace(got) != "bottom" {
		t.Fatalf("pane-border-status with a terminal = %q, want bottom", got)
	}
	time.Sleep(1500 * time.Millisecond)
	if got := tmux("show-options", "-wv", "-t", "probe", "pane-border-status"); strings.TrimSpace(got) != "off" {
		t.Fatalf("pane-border-status after the terminal exited = %q, want off", got)
	}
	// A session created in the background must survive until a client
	// attaches; destroy-unattached would remove it at once.
	tmux("has-session", "-t", "probe")
	if got := tmux("list-keys", "-T", "root"); !strings.Contains(got, "F2") || !strings.Contains(got, "M-Up") {
		t.Fatalf("root keys lack F2 or M-Up:\n%s", got)
	}
}

// fakeTmux records tmux commands and answers display-message with command.
func fakeTmux(t *testing.T, command string, alive bool) *[]string {
	t.Helper()
	var calls []string
	orig := runTmux
	t.Cleanup(func() { runTmux = orig })
	runTmux = func(args ...string) (string, error) {
		calls = append(calls, strings.Join(args, " "))
		switch args[0] {
		case "display-message":
			if !alive {
				return "can't find pane", errors.New("exit status 1")
			}
			return command, nil
		case "split-window":
			return "%9", nil
		}
		return "", nil
	}
	return &calls
}

func runCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("no command")
	}
	return cmd()
}

func TestTerminalToSelectedTypesCdIntoAShell(t *testing.T) {
	m, workspace := workspaceModel(t)
	m.inTmux, m.terminalPane = true, "%3"
	calls := fakeTmux(t, "/bin/bash", true)

	msg := runCmd(t, m.terminalToSelected())
	dir := filepath.Join(workspace, "01_static")
	if moved, ok := msg.(terminalMovedMsg); !ok || moved.busy != "" || moved.dir != dir {
		t.Fatalf("msg = %#v, want a move to %s", msg, dir)
	}
	want := []string{
		"display-message -p -t %3 #{pane_current_command}",
		"send-keys -t %3 C-u",
		"send-keys -t %3 -l cd -- '" + dir + "'",
		"send-keys -t %3 Enter",
		"select-pane -t %3",
	}
	if strings.Join(*calls, "\n") != strings.Join(want, "\n") {
		t.Fatalf("tmux calls:\n%s\nwant:\n%s", strings.Join(*calls, "\n"), strings.Join(want, "\n"))
	}
	m = updateWith(m, msg)
	if !strings.Contains(m.status, "Terminal moved to") {
		t.Fatalf("status = %q", m.status)
	}
}

func TestTerminalToSelectedLeavesABusyProgramAlone(t *testing.T) {
	m, _ := workspaceModel(t)
	m.inTmux, m.terminalPane = true, "%3"
	calls := fakeTmux(t, "nano", true)

	msg := runCmd(t, m.terminalToSelected())
	if moved, ok := msg.(terminalMovedMsg); !ok || moved.busy != "nano" {
		t.Fatalf("msg = %#v, want busy nano", msg)
	}
	for _, call := range *calls {
		if strings.HasPrefix(call, "send-keys") {
			t.Fatalf("typed into nano: %q", call)
		}
	}
	if m = updateWith(m, msg); !strings.Contains(m.status, "running nano") {
		t.Fatalf("status = %q", m.status)
	}
}

func TestTerminalToSelectedReopensAClosedPane(t *testing.T) {
	m, workspace := workspaceModel(t)
	m.inTmux, m.terminalPane = true, "%3"
	calls := fakeTmux(t, "", false)

	msg := runCmd(t, m.terminalToSelected())
	if pane, ok := msg.(terminalPaneMsg); !ok || pane.pane != "%9" {
		t.Fatalf("msg = %#v, want the new pane %%9", msg)
	}
	var split string
	for _, call := range *calls {
		if strings.HasPrefix(call, "split-window") {
			split = call
		}
	}
	if !strings.Contains(split, "-c "+filepath.Join(workspace, "01_static")) || strings.Contains(split, " -d") {
		t.Fatalf("split = %q, want a focused pane in the exercise folder", split)
	}
	if m = updateWith(m, msg); m.terminalPane != "%9" {
		t.Fatalf("terminalPane = %q", m.terminalPane)
	}
}

func TestTerminalWithoutTmuxRunsAShell(t *testing.T) {
	m, workspace := workspaceModel(t)
	var ranIn string
	orig := runShell
	t.Cleanup(func() { runShell = orig })
	runShell = func(dir, hint string) tea.Cmd {
		ranIn = dir
		return func() tea.Msg { return shellDoneMsg{} }
	}
	m = updateWith(m, runCmd(t, m.terminalToSelected()))
	if ranIn != filepath.Join(workspace, "01_static") || m.status != "Back from the shell." {
		t.Fatalf("shell ran in %q, status %q", ranIn, m.status)
	}
}

func TestOwnSessionOpensTheTerminalOnceWhenTallEnough(t *testing.T) {
	m, _ := workspaceModel(t)
	m.ownTmux, m.inTmux, m.terminalTried = true, true, false
	m = updateWith(m, tea.WindowSizeMsg{Width: 100, Height: minSplitHeight - 1})
	if m.terminalTried {
		t.Fatal("a short window opened the terminal")
	}
	m = updateWith(m, tea.WindowSizeMsg{Width: 100, Height: 30})
	if !m.terminalTried {
		t.Fatal("the terminal was not opened at start")
	}
}
