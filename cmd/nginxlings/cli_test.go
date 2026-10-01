package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCLIHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := runCLI([]string{"--help"}, &stdout, &stderr, nil, nil)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("stdout = %q, want usage", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunCLIVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := runCLI([]string{"--version"}, &stdout, &stderr, nil, nil)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "nginxlings ") {
		t.Fatalf("stdout = %q, want version", stdout.String())
	}
}

func TestRunCLIList(t *testing.T) {
	root := newCLIAppRoot(t)
	createExercise(t, root, "02_second")
	createExercise(t, root, "01_first")
	var stdout, stderr bytes.Buffer

	code := runCLI([]string{"--root", root, "--state", filepath.Join(t.TempDir(), "state.json"), "--list"}, &stdout, &stderr, nil, nil)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("list output lines = %v, want 2 lines", lines)
	}
	if !strings.HasPrefix(lines[0], "01_first\t") || !strings.HasPrefix(lines[1], "02_second\t") {
		t.Fatalf("list output = %q, want stable exercise order", stdout.String())
	}
}

func TestRunCLIListDoesNotRequireStateLocation(t *testing.T) {
	clearStateLocationEnv(t)
	root := newCLIAppRoot(t)
	createExercise(t, root, "01_static_server")
	var stdout, stderr bytes.Buffer

	code := runCLI([]string{"--root", root, "--list"}, &stdout, &stderr, nil, nil)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 without HOME/state; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "01_static_server") {
		t.Fatalf("stdout = %q, want discovered exercise", stdout.String())
	}
}

func TestRunCLICheckDoesNotRequireStateLocation(t *testing.T) {
	clearStateLocationEnv(t)
	root := newCLIAppRoot(t)
	createExercise(t, root, "01_static_server")
	var stdout, stderr bytes.Buffer
	runnerCalled := false

	code := runCLI(
		[]string{"--root", root, "--check", "01_static_server"},
		&stdout,
		&stderr,
		func(ex Exercise) (string, error) {
			runnerCalled = true
			return "[PASS] stateless\n", nil
		},
		nil,
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 without HOME/state; stderr=%q", code, stderr.String())
	}
	if !runnerCalled {
		t.Fatal("check runner was not called")
	}
	if !strings.Contains(stdout.String(), "[PASS] stateless") {
		t.Fatalf("stdout = %q, want runner output", stdout.String())
	}
}

func TestRunCLICheckValidExercise(t *testing.T) {
	root := newCLIAppRoot(t)
	createExercise(t, root, "01_static_server")
	var stdout, stderr bytes.Buffer
	var ran Exercise

	code := runCLI(
		[]string{"--root", root, "--state", filepath.Join(t.TempDir(), "state.json"), "--check", "01_static_server"},
		&stdout,
		&stderr,
		func(ex Exercise) (string, error) {
			ran = ex
			return "[PASS] ok\n", nil
		},
		nil,
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if ran.ID != "01_static_server" {
		t.Fatalf("ran.ID = %q, want 01_static_server", ran.ID)
	}
	if !strings.Contains(stdout.String(), "[PASS] ok") {
		t.Fatalf("stdout = %q, want runner output", stdout.String())
	}
}

func TestRunCLICheckMissingExercise(t *testing.T) {
	root := newCLIAppRoot(t)
	createExercise(t, root, "01_static_server")
	var stdout, stderr bytes.Buffer

	code := runCLI(
		[]string{"--root", root, "--state", filepath.Join(t.TempDir(), "state.json"), "--check", "99_missing"},
		&stdout,
		&stderr,
		func(ex Exercise) (string, error) {
			t.Fatalf("runner should not be called for missing exercise")
			return "", nil
		},
		nil,
	)

	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero")
	}
	if !strings.Contains(stderr.String(), `exercise "99_missing" not found`) {
		t.Fatalf("stderr = %q, want missing exercise error", stderr.String())
	}
}

func TestRunCLICheckFailureReturnsNonZero(t *testing.T) {
	root := newCLIAppRoot(t)
	createExercise(t, root, "01_static_server")
	var stdout, stderr bytes.Buffer

	code := runCLI(
		[]string{"--root", root, "--state", filepath.Join(t.TempDir(), "state.json"), "--check", "01_static_server"},
		&stdout,
		&stderr,
		func(ex Exercise) (string, error) {
			return "[FAIL] bad\n", errors.New("exit status 1")
		},
		nil,
	)

	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero")
	}
	if !strings.Contains(stdout.String(), "[FAIL] bad") {
		t.Fatalf("stdout = %q, want runner output", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Check failed") {
		t.Fatalf("stderr = %q, want check failure", stderr.String())
	}
}

func TestRunCLICheckTimeoutReturnsNonZero(t *testing.T) {
	root := newCLIAppRoot(t)
	createExercise(t, root, "01_static_server")
	var stdout, stderr bytes.Buffer

	code := runCLI(
		[]string{"--root", root, "--state", filepath.Join(t.TempDir(), "state.json"), "--check", "01_static_server"},
		&stdout,
		&stderr,
		func(Exercise) (string, error) {
			return "partial output\n", errors.New("check.sh timed out")
		},
		nil,
	)

	if code == 0 {
		t.Fatal("exit code = 0, want non-zero for timeout")
	}
	if !strings.Contains(stdout.String(), "partial output") {
		t.Fatalf("stdout = %q, want partial runner output", stdout.String())
	}
	if !strings.Contains(stderr.String(), "check.sh timed out") {
		t.Fatalf("stderr = %q, want timeout error", stderr.String())
	}
}

func TestRunCLILangFlagStartsTUIWithLang(t *testing.T) {
	var stdout, stderr bytes.Buffer
	var got cliOptions

	code := runCLI([]string{"--lang", "tr"}, &stdout, &stderr, nil, func(opts cliOptions) error {
		got = opts
		return nil
	})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if got.lang != "tr" {
		t.Fatalf("lang = %q, want tr", got.lang)
	}
}

func TestRunCLIRootFlag(t *testing.T) {
	root := newCLIAppRoot(t)
	createExercise(t, root, "01_static_server")
	var stdout, stderr bytes.Buffer

	code := runCLI([]string{"--root", root, "--state", filepath.Join(t.TempDir(), "state.json"), "--list"}, &stdout, &stderr, nil, nil)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "01_static_server") {
		t.Fatalf("stdout = %q, want root exercise", stdout.String())
	}
}

func TestRunCLIStateFlagStartsTUIWithState(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	var stdout, stderr bytes.Buffer
	var got cliOptions

	code := runCLI([]string{"--state", statePath}, &stdout, &stderr, nil, func(opts cliOptions) error {
		got = opts
		return nil
	})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if got.state != statePath {
		t.Fatalf("state = %q, want %q", got.state, statePath)
	}
}

func TestRunCLINoFlagsStartsTUIMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	started := false

	code := runCLI(nil, &stdout, &stderr, nil, func(opts cliOptions) error {
		started = true
		if opts.mode != modeTUI {
			t.Fatalf("mode = %v, want modeTUI", opts.mode)
		}
		return nil
	})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if !started {
		t.Fatalf("TUI starter was not called")
	}
}

func TestInitialModelWithOptionsUsesLangFlag(t *testing.T) {
	root := newCLIAppRoot(t)
	createExercise(t, root, "01_static_server")
	paths := AppPaths{
		WorkDir:   root,
		AppRoot:   root,
		AssetRoot: root,
		StateRoot: t.TempDir(),
		StateFile: filepath.Join(t.TempDir(), "state.json"),
	}

	m := initialModelWithOptions(paths, "tr")

	if got := m.currentLangCode(); got != "tr" {
		t.Fatalf("currentLangCode() = %q, want tr", got)
	}
}

func TestNewTUIModelRejectsUnknownLanguage(t *testing.T) {
	root := newCLIAppRoot(t)
	createExercise(t, root, "01_static_server")

	_, err := newTUIModel(cliOptions{root: root, state: filepath.Join(t.TempDir(), "state.json"), lang: "xx"})

	if err == nil {
		t.Fatal("newTUIModel() error = nil, want unknown language error")
	}
	for _, want := range []string{`unknown language "xx"`, "available: tr, en"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want %q", err, want)
		}
	}
}

func TestNewTUIModelAcceptsKnownLanguage(t *testing.T) {
	root := newCLIAppRoot(t)
	createExercise(t, root, "01_static_server")

	m, err := newTUIModel(cliOptions{root: root, state: filepath.Join(t.TempDir(), "state.json"), lang: "tr"})

	if err != nil {
		t.Fatalf("newTUIModel() error = %v", err)
	}
	if got := m.currentLangCode(); got != "tr" {
		t.Fatalf("currentLangCode() = %q, want tr", got)
	}
}

func TestRunCLIReportsUnknownLanguage(t *testing.T) {
	root := newCLIAppRoot(t)
	createExercise(t, root, "01_static_server")
	var stdout, stderr bytes.Buffer
	startTUI := func(opts cliOptions) error {
		_, err := newTUIModel(opts)
		return err
	}

	code := runCLI(
		[]string{"--root", root, "--state", filepath.Join(t.TempDir(), "state.json"), "--lang", "xx"},
		&stdout, &stderr, nil, startTUI,
	)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), `unknown language "xx"`) {
		t.Fatalf("stderr = %q, want unknown language error", stderr.String())
	}
}

func clearStateLocationEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		envStateFile,
		envStateRoot,
		envXDGStateHome,
		envHome,
		envUserProfile,
	} {
		t.Setenv(key, "")
	}
}

func newCLIAppRoot(t *testing.T) string {
	t.Helper()

	root := newAppAssetRoot(t)
	writeI18n(t, root)
	return root
}

func writeI18n(t *testing.T, root string) {
	t.Helper()

	writeFile(t, filepath.Join(root, "i18n", "langs.json"), `[
  { "code": "tr", "label": "Türkçe" },
  { "code": "en", "label": "English" }
]`)
	if err := os.MkdirAll(filepath.Join(root, "i18n", "ui"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "i18n", "exercises"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "i18n", "ui", "tr.json"), `{"readme_initial":"TR initial"}`)
	writeFile(t, filepath.Join(root, "i18n", "ui", "en.json"), `{"readme_initial":"EN initial"}`)
	writeFile(t, filepath.Join(root, "i18n", "exercises", "tr.json"), `{}`)
	writeFile(t, filepath.Join(root, "i18n", "exercises", "en.json"), `{}`)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
