package tui

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"nginxlings/tests/testutil"
)

// fakeRoot builds an asset root with the repository's translations and two
// exercises whose checks answer at once: 01 passes and 02 fails. It lets the
// check flow run without Docker or NGINX.
func fakeRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	copy := exec.Command("cp", "-r", filepath.Join(testutil.RepoRoot(t), "i18n"), root)
	if out, err := copy.CombinedOutput(); err != nil {
		t.Fatalf("copy i18n: %v\n%s", err, out)
	}
	exercises := []struct{ id, name, script string }{
		{"01_pass", "01 - Passing Exercise", "echo '[1/1] Pretending to check...'\necho '[PASS] fake check passed'\n"},
		{"02_fail", "02 - Failing Exercise", "echo '[1/1] Pretending to check...'\necho '[FAIL] fake check failed'\nexit 1\n"},
	}
	type entry struct {
		ID          string `json:"id"`
		DefaultName string `json:"default_name"`
	}
	var manifest struct {
		Exercises []entry `json:"exercises"`
	}
	for _, ex := range exercises {
		dir := filepath.Join(root, "exercises", ex.id)
		title := "# " + strings.Replace(ex.name, " - ", " – ", 1) + "\n\nFake exercise for the TUI tests.\n"
		writeTestFile(t, filepath.Join(dir, "README.en.md"), title, 0o644)
		writeTestFile(t, filepath.Join(dir, "README.tr.md"), title, 0o644)
		writeTestFile(t, filepath.Join(dir, "check.sh"), "#!/usr/bin/env bash\n"+ex.script, 0o755)
		manifest.Exercises = append(manifest.Exercises, entry{ex.id, ex.name})
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "exercises", "manifest.json"), string(data), 0o644)
	return root
}

func writeTestFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

// readmeTitle is how the instructions panel shows an exercise's README title.
func readmeTitle(id string) string {
	return id[:2] + " – "
}

func TestTUINavigationCountsEveryKeyPress(t *testing.T) {
	s := start(t, options{cols: 100, rows: 30})
	s.waitForFrame(100, 30, textsFor(t, "en"))
	s.waitForText(readmeTitle("01"))

	s.press("s", "s", "s")
	s.waitForText(readmeTitle("04"))

	// Keys typed faster than a redraw reach the program in one read.
	s.write("sss")
	s.waitForText(readmeTitle("07"))
	s.write("ww")
	s.waitForText(readmeTitle("05"))
}

func TestTUIHintTogglesWithInstructions(t *testing.T) {
	en := textsFor(t, "en")
	s := start(t, options{cols: 100, rows: 30})
	s.waitForFrame(100, 30, en)

	s.press("h")
	s.waitForText(uiText(t, "en", "hint_title"))
	s.waitForText(uiText(t, "en", "status_showing_hint"))
	s.press("h")
	s.waitForText(uiText(t, "en", "status_showing_instructions"))
	s.waitForFrame(100, 30, en)
}

func TestTUILanguageSwitchTranslatesTheScreen(t *testing.T) {
	s := start(t, options{cols: 100, rows: 30, lang: "en"})
	s.waitForFrame(100, 30, textsFor(t, "en"))
	s.waitForText("01 – Basic Static NGINX Server")

	s.press("l")
	s.waitForFrame(100, 30, textsFor(t, "tr"))
	s.waitForText("01 – Basit Statik NGINX Sunucusu")
	s.waitForText(uiText(t, "tr", "exercises_title"))

	s.press("l")
	s.waitForFrame(100, 30, textsFor(t, "en"))
	s.waitForText("01 – Basic Static NGINX Server")
}

func TestTUIScrollMovesTheInstructions(t *testing.T) {
	s := start(t, options{cols: 100, rows: 30})
	s.waitForFrame(100, 30, textsFor(t, "en"))
	before := s.waitForText("## Goal")

	s.press("pgdown", "pgdown")
	s.waitFor("the instructions to scroll", func(rows []string) bool {
		return !strings.Contains(strings.Join(rows, "\n"), "## Goal") &&
			strings.Join(rows, "\n") != strings.Join(before, "\n")
	})
	s.press("pgup", "pgup")
	s.waitForText("## Goal")
}

func TestTUICheckFlowShowsResultAndProgress(t *testing.T) {
	en := textsFor(t, "en")
	root := fakeRoot(t)
	state := filepath.Join(t.TempDir(), "state.json")
	s := start(t, options{cols: 100, rows: 30, root: root, stateFile: state})
	s.waitForFrame(100, 30, en)
	s.waitForText(" 0/2 ")

	s.press("c")
	s.waitForText("[PASS] fake check passed")
	s.waitForText(uiText(t, "en", "check_output_title"))
	s.waitForText("✓ 01 - Passing Exercise")
	s.waitForText(" 1/2 ")

	s.press("tab")
	s.waitForText(uiText(t, "en", "instructions_title"))
	s.waitForText("01 – Passing Exercise")

	s.press("s", "c")
	s.waitForText("[FAIL] fake check failed")
	s.waitForText("x 02 - Failing Exercise")
	s.waitForText(" 1/2 ")
	s.waitForFrame(100, 30, en)

	s.press("q")
	if err := s.waitExit(); err != nil {
		t.Fatalf("nginxlings exited with %v\n%s", err, s.dump())
	}
	var saved struct {
		Completed      map[string]bool `json:"completed"`
		LastExerciseID string          `json:"last_exercise_id"`
	}
	data, err := os.ReadFile(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("state file: %v\n%s", err, data)
	}
	if !saved.Completed["01_pass"] || saved.Completed["02_fail"] || saved.LastExerciseID != "02_fail" {
		t.Fatalf("saved state = %+v, want 01_pass completed, 02_fail not, last 02_fail", saved)
	}
}

func TestTUIResetClearsProgress(t *testing.T) {
	en := textsFor(t, "en")
	s := start(t, options{cols: 100, rows: 30, root: fakeRoot(t)})
	s.waitForFrame(100, 30, en)
	s.press("c")
	s.waitForText(" 1/2 ")

	s.press("r")
	s.waitForText(uiText(t, "en", "status_reset_confirm"))
	s.press("n")
	s.waitForText(uiText(t, "en", "status_reset_canceled"))
	s.waitForText(" 1/2 ")

	s.press("r", "y")
	s.waitForText(uiText(t, "en", "status_reset_done"))
	s.waitForText(" 0/2 ")
	s.waitForText("x 01 - Passing Exercise")
}

func TestTUIRestartRestoresTheSelectedExercise(t *testing.T) {
	en := textsFor(t, "en")
	state := filepath.Join(t.TempDir(), "state.json")
	first := start(t, options{cols: 100, rows: 30, stateFile: state})
	first.waitForFrame(100, 30, en)
	first.press("s", "s")
	first.waitForText(readmeTitle("03"))
	first.press("q")
	if err := first.waitExit(); err != nil {
		t.Fatalf("nginxlings exited with %v", err)
	}

	second := start(t, options{cols: 100, rows: 30, stateFile: state})
	second.waitForFrame(100, 30, en)
	second.waitForText(readmeTitle("03"))
}
