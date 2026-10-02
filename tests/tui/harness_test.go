// Package tui drives the built nginxlings binary in a pseudo-terminal and
// reads the screen through a terminal emulator, the way a user sees it.
package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/creack/pty"
	"github.com/hinshun/vt10x"

	"nginxlings/tests/testutil"
)

// binPath is the nginxlings binary built once for the whole package.
var binPath string

// toolsDir holds stand-ins for the commands the exercise checks need. It
// comes first on the program's PATH, so the setup check finds a ready
// machine wherever the tests run and the status row does not depend on what
// is installed.
var toolsDir string

// standIns are the commands the exercise checks ask for, except bash, which
// the checks themselves run in. docker answers the two calls the setup check
// makes; the others only have to exist.
var standIns = map[string]string{
	"docker": `case "$1" in
--version) echo "Docker version 29.1.3, build tui-test" ;;
info) echo "Server Version: 29.1.3" ;;
*) echo "tui tests: docker $* has no stand-in" >&2; exit 1 ;;
esac
`,
	"nginx":   standInOnly,
	"curl":    standInOnly,
	"python3": standInOnly,
}

const standInOnly = `echo "tui tests: $0 is a stand-in for the setup check" >&2
exit 1
`

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	root := testutil.RepoRootPath()
	dir, err := os.MkdirTemp("", "nginxlings-tui-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer os.RemoveAll(dir)
	binPath = filepath.Join(dir, "nginxlings")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/nginxlings")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build nginxlings: %v\n%s", err, out)
		return 1
	}
	toolsDir = filepath.Join(dir, "tools")
	if err := os.Mkdir(toolsDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	for name, body := range standIns {
		if err := os.WriteFile(filepath.Join(toolsDir, name), []byte("#!/bin/sh\n"+body), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	return m.Run()
}

// keyBytes maps key names to what a terminal sends for them.
var keyBytes = map[string]string{
	"tab":    "\t",
	"enter":  "\r",
	"esc":    "\x1b",
	"up":     "\x1b[A",
	"down":   "\x1b[B",
	"right":  "\x1b[C",
	"left":   "\x1b[D",
	"pgup":   "\x1b[5~",
	"pgdown": "\x1b[6~",
	"f2":     "\x1bOQ",
	"ctrl+c": "\x03",
}

// session is one running nginxlings process attached to a pseudo-terminal
// and a terminal emulator of the same size.
type session struct {
	t         *testing.T
	cmd       *exec.Cmd
	pty       *os.File
	vt        vt10x.Terminal
	done      chan struct{}
	exited    chan error
	stateFile string
	root      string
}

type options struct {
	cols, rows int
	lang       string
	// root is the asset root; empty means the repository.
	root string
	// stateFile is reused across sessions when set.
	stateFile string
	// workspace, when set, is the learner's exercise folder (--workspace).
	workspace string
	// tmux lets the program start its tmux session; tests run without it
	// unless they test it.
	tmux bool
	// env holds extra environment variables.
	env []string
}

// withoutEnv returns env without the named variables.
func withoutEnv(env []string, names ...string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		drop := false
		for _, n := range names {
			drop = drop || name == n
		}
		if !drop {
			out = append(out, kv)
		}
	}
	return out
}

func start(t *testing.T, opt options) *session {
	t.Helper()
	if opt.root == "" {
		opt.root = testutil.RepoRoot(t)
	}
	if opt.stateFile == "" {
		opt.stateFile = filepath.Join(t.TempDir(), "state.json")
	}
	if opt.lang == "" {
		opt.lang = "en"
	}
	args := []string{"--root", opt.root, "--state", opt.stateFile, "--lang", opt.lang}
	if opt.workspace != "" {
		args = append(args, "--workspace", opt.workspace)
	}
	if !opt.tmux {
		args = append(args, "--no-tmux")
	}
	cmd := exec.Command(binPath, args...)
	// A test run from inside tmux must not split the developer's window.
	cmd.Env = append(withoutEnv(os.Environ(), "TMUX", "TMUX_PANE", "PATH"),
		"TERM=xterm-256color", "PATH="+toolsDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	cmd.Env = append(cmd.Env, opt.env...)
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(opt.cols), Rows: uint16(opt.rows)})
	if err != nil {
		t.Fatalf("start nginxlings in a pty: %v", err)
	}
	s := &session{
		t:         t,
		cmd:       cmd,
		pty:       ptmx,
		vt:        vt10x.New(vt10x.WithSize(opt.cols, opt.rows), vt10x.WithWriter(ptmx)),
		done:      make(chan struct{}),
		exited:    make(chan error, 1),
		stateFile: opt.stateFile,
		root:      opt.root,
	}
	go s.pump()
	go func() { s.exited <- cmd.Wait() }()
	t.Cleanup(s.close)
	return s
}

// pump copies program output into the emulator until the pty closes.
func (s *session) pump() {
	defer close(s.done)
	buf := make([]byte, 32*1024)
	var pending []byte
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			data := append(pending, buf[:n]...)
			complete, rest := splitAtRuneBoundary(data)
			_, _ = s.vt.Write(complete)
			pending = append([]byte(nil), rest...)
		}
		if err != nil {
			return
		}
	}
}

// splitAtRuneBoundary returns data up to the last complete UTF-8 rune and the
// incomplete tail. vt10x decodes each Write on its own and loses a rune that a
// read splits in two, which shows up as a missing border cell in wide frames.
func splitAtRuneBoundary(data []byte) (complete, rest []byte) {
	for back := 1; back <= utf8.UTFMax && back <= len(data); back++ {
		i := len(data) - back
		if !utf8.RuneStart(data[i]) {
			continue
		}
		if utf8.FullRune(data[i:]) {
			return data, nil
		}
		return data[:i], data[i:]
	}
	return data, nil
}

func (s *session) close() {
	if s.cmd.ProcessState == nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	select {
	case <-s.exited:
	case <-time.After(5 * time.Second):
	}
	_ = s.pty.Close()
	<-s.done
}

// screen returns the emulator's rows with trailing spaces removed.
func (s *session) screen() []string {
	// String takes the emulator lock itself; locking here would deadlock.
	text := s.vt.String()
	_, height := s.vt.Size()
	rows := strings.Split(text, "\n")
	if len(rows) > height {
		rows = rows[:height]
	}
	for i, row := range rows {
		rows[i] = strings.TrimRight(row, " ")
	}
	for len(rows) < height {
		rows = append(rows, "")
	}
	return rows
}

func (s *session) dump() string {
	cols, rows := s.vt.Size()
	return fmt.Sprintf("screen %dx%d:\n%s", cols, rows, strings.Join(s.screen(), "\n"))
}

// waitFor polls the screen until cond holds, and fails with the screen when
// it does not within timeout.
func (s *session) waitFor(what string, cond func(rows []string) bool) []string {
	s.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		rows := s.screen()
		if cond(rows) {
			return rows
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("timed out waiting for %s\n%s", what, s.dump())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (s *session) waitForText(text string) []string {
	s.t.Helper()
	return s.waitFor(fmt.Sprintf("%q on screen", text), func(rows []string) bool {
		return strings.Contains(strings.Join(rows, "\n"), text)
	})
}

// press sends keys one at a time, the way a person types them.
func (s *session) press(keys ...string) {
	s.t.Helper()
	for _, key := range keys {
		data, ok := keyBytes[key]
		if !ok {
			data = key
		}
		s.write(data)
		time.Sleep(60 * time.Millisecond)
	}
}

// write sends raw bytes in a single write.
func (s *session) write(data string) {
	s.t.Helper()
	if _, err := io.WriteString(s.pty, data); err != nil {
		s.t.Fatalf("write %q to pty: %v", data, err)
	}
}

// resize changes the emulator and the pty size; the program receives
// SIGWINCH and redraws.
func (s *session) resize(cols, rows int) {
	s.t.Helper()
	s.vt.Resize(cols, rows)
	if err := pty.Setsize(s.pty, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)}); err != nil {
		s.t.Fatalf("resize pty: %v", err)
	}
}

// waitExit waits for the program to exit and returns its error.
func (s *session) waitExit() error {
	s.t.Helper()
	select {
	case err := <-s.exited:
		s.exited <- err
		return err
	case <-time.After(5 * time.Second):
		s.t.Fatalf("nginxlings did not exit\n%s", s.dump())
		return errors.New("timeout")
	}
}

// uiText returns one string from i18n/ui/<lang>.json.
func uiText(t *testing.T, lang, key string) string {
	t.Helper()
	path := filepath.Join(testutil.RepoRoot(t), "i18n", "ui", lang+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var texts map[string]string
	if err := json.Unmarshal(data, &texts); err != nil {
		t.Fatal(err)
	}
	text, ok := texts[key]
	if !ok {
		t.Fatalf("%s has no %q", path, key)
	}
	return text
}
