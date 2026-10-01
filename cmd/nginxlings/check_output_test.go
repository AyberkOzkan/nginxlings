package main

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestSanitizeCheckOutputRemovesTerminalControl(t *testing.T) {
	raw := "\x1b]0;docker build\x07\x1b[1;32m[1/3] Building\x1b[0m\r\n" +
		"progress 10%\rprogress 55%\rprogress 100%\r\n" +
		"step\tone\x1b[2K\x1b[1A\n" +
		"bell\x07 and backspace\b\n" +
		"[FAIL] done\r\n"

	got := sanitizeCheckOutput(raw)

	want := "[1/3] Building\nprogress 100%\nstep    one\nbell and backspace\n[FAIL] done"
	if got != want {
		t.Fatalf("sanitizeCheckOutput() = %q, want %q", got, want)
	}
}

func TestSanitizeCheckOutputKeepsTextBeforeTrailingCarriageReturn(t *testing.T) {
	if got := sanitizeCheckOutput("downloading 100%\r"); got != "downloading 100%" {
		t.Fatalf("sanitizeCheckOutput() = %q, want the last visible text", got)
	}
}

func TestWrapCheckOutputKeepsEveryCharacter(t *testing.T) {
	line := "[FAIL] nginx -t failed for 01_static_server because port 8080 is already in use"

	wrapped := wrapCheckOutput(line, 20)

	for _, row := range strings.Split(wrapped, "\n") {
		if w := lipgloss.Width(row); w > 20 {
			t.Fatalf("row width = %d, want <= 20: %q", w, row)
		}
	}
	if strings.ReplaceAll(strings.ReplaceAll(wrapped, "\n", ""), " ", "") != strings.ReplaceAll(line, " ", "") {
		t.Fatalf("wrapping lost text:\n%s", wrapped)
	}
}

func TestLongCheckOutputWrapsInsideThe80ColumnPanel(t *testing.T) {
	m := layoutTestModel(80, 24)
	m.exercises = []Exercise{{ID: "01_static_server", DefaultName: "01 - Static"}}
	line := "[FAIL] nginx -t failed for 01_static_server because port 8080 is already in use"

	updated, _ := m.Update(checkResultMsg{output: line, err: errors.New("exit status 1"), index: 0})
	rendered := updated.(model).View()

	assertRenderedLinesWithin(t, rendered, 80)
	if got := renderedRows(rendered); got > 24 {
		t.Fatalf("rendered rows = %d, want <= 24\n%s", got, rendered)
	}
	if !strings.Contains(rendered, "already in use") {
		t.Fatalf("end of the long output line is not visible:\n%s", rendered)
	}
}

func TestCheckResultForAnotherExerciseKeepsCurrentPanel(t *testing.T) {
	m := layoutTestModel(100, 24)
	m.exercises = []Exercise{
		{ID: "01_static_server", DefaultName: "01 - Static"},
		{ID: "02_config_layout", DefaultName: "02 - Layout"},
	}
	m.cursor = 1
	m.instructionsContent = "Second instructions"
	m.readme.SetContent(m.instructionsContent)
	m.loading = true

	updated, _ := m.Update(checkResultMsg{output: "[PASS] first exercise", index: 0})
	got := updated.(model)

	if got.panel == panelCheckOutput {
		t.Fatal("result for another exercise replaced the selected exercise's panel")
	}
	if !strings.Contains(got.readme.View(), "Second instructions") {
		t.Fatalf("readme = %q, want the selected exercise's instructions", got.readme.View())
	}
	if got.status != "01_static_server: [PASS] first exercise" {
		t.Fatalf("status = %q, want the result prefixed with its exercise", got.status)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyTab})
	got = updated.(model)
	if got.panel == panelCheckOutput || got.status != "No check output yet." {
		t.Fatalf("Tab on an unchecked exercise showed output: status=%q", got.status)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("w")})
	updated, _ = updated.(model).Update(tea.KeyMsg{Type: tea.KeyTab})
	got = updated.(model)
	if got.panel != panelCheckOutput || !strings.Contains(got.readme.View(), "[PASS] first exercise") {
		t.Fatalf("Tab on the checked exercise did not show its output: %q", got.readme.View())
	}
}
