package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdateDoesNotStartSecondCheckWhileLoading(t *testing.T) {
	m := model{
		exercises: []Exercise{{ID: "01_static_server", DefaultName: "01 - Static", Path: t.TempDir()}},
		loading:   true,
		readme:    viewport.New(80, 20),
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	got := updated.(model)

	if cmd != nil {
		t.Fatalf("cmd = %v, want nil while check is already loading", cmd)
	}
	if !got.loading {
		t.Fatalf("loading = false, want true")
	}
	if !strings.Contains(got.status, "zaten çalışıyor") {
		t.Fatalf("status = %q, want already running message", got.status)
	}
}

func TestUpdateCheckResultSuccessState(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	m := model{
		exercises: []Exercise{{ID: "01_static_server", DefaultName: "01 - Static"}},
		loading:   true,
		readme:    viewport.New(80, 20),
		state:     defaultState(),
		paths:     AppPaths{StateFile: statePath},
	}

	updated, cmd := m.Update(checkResultMsg{output: "[PASS] ok\n", index: 0})
	got := updated.(model)

	if cmd != nil {
		t.Fatalf("cmd = %v, want nil", cmd)
	}
	if got.loading {
		t.Fatalf("loading = true, want false")
	}
	if !got.exercises[0].Completed {
		t.Fatalf("exercise completed = false, want true")
	}
	if got.lastCheckOutput != "[PASS] ok" {
		t.Fatalf("lastCheckOutput = %q, want [PASS] ok", got.lastCheckOutput)
	}
	if !strings.Contains(got.readme.View(), "[PASS] ok") {
		t.Fatalf("readme view = %q, want check output", got.readme.View())
	}
	loaded, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if !loaded.Completed["01_static_server"] {
		t.Fatalf("saved completed state = false, want true")
	}
}

func TestUpdateCheckResultFailureState(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	m := model{
		exercises: []Exercise{{ID: "01_static_server", DefaultName: "01 - Static", Completed: true}},
		loading:   true,
		readme:    viewport.New(80, 20),
		state:     defaultState(),
		paths:     AppPaths{StateFile: statePath},
	}

	updated, _ := m.Update(checkResultMsg{output: "[FAIL] bad\n", err: errors.New("exit status 1"), index: 0})
	got := updated.(model)

	if got.loading {
		t.Fatalf("loading = true, want false")
	}
	if got.exercises[0].Completed {
		t.Fatalf("exercise completed = true, want false")
	}
	if got.lastCheckOutput != "[FAIL] bad" {
		t.Fatalf("lastCheckOutput = %q, want [FAIL] bad", got.lastCheckOutput)
	}
	if !strings.Contains(got.status, "[FAIL] bad") {
		t.Fatalf("status = %q, want fail line", got.status)
	}
}

func TestUpdateCheckResultEmptyFailureShowsRunnerError(t *testing.T) {
	m := model{
		exercises: []Exercise{{ID: "01_static_server", DefaultName: "01 - Static"}},
		loading:   true,
		readme:    viewport.New(80, 20),
	}

	updated, _ := m.Update(checkResultMsg{err: errors.New("check.sh not found for exercise 01_static_server"), index: 0})
	got := updated.(model)

	if !strings.Contains(got.lastCheckOutput, "check.sh not found") {
		t.Fatalf("lastCheckOutput = %q, want runner error", got.lastCheckOutput)
	}
	if !strings.Contains(got.status, "check.sh not found") {
		t.Fatalf("status = %q, want runner error", got.status)
	}
}

func TestUpdateSelectionLoadsSelectedExerciseInstructions(t *testing.T) {
	root := newCLIAppRoot(t)
	createExercise(t, root, "01_static_server")
	createExercise(t, root, "02_config_layout")
	writeFile(t, filepath.Join(root, "exercises", "01_static_server", "README.en.md"), "# First Exercise\n")
	writeFile(t, filepath.Join(root, "exercises", "02_config_layout", "README.en.md"), "# Second Exercise\n")

	m := newUpdateModelFromRoot(t, root, "en")
	m.ready = true
	m.width = 100
	m.height = 24

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	got := updated.(model)

	if got.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", got.cursor)
	}
	if !strings.Contains(got.instructionsContent, "Second Exercise") {
		t.Fatalf("instructionsContent = %q, want second exercise README", got.instructionsContent)
	}
}

func TestUpdateLanguageSwitchReloadsInstructions(t *testing.T) {
	root := newCLIAppRoot(t)
	createExercise(t, root, "01_static_server")
	writeFile(t, filepath.Join(root, "exercises", "01_static_server", "README.en.md"), "# English Instructions\n")
	writeFile(t, filepath.Join(root, "exercises", "01_static_server", "README.tr.md"), "# Turkce Talimatlar\n")

	m := newUpdateModelFromRoot(t, root, "en")
	m.ready = true
	m.width = 100
	m.height = 24

	if got := m.currentLangCode(); got != "en" {
		t.Fatalf("initial lang = %q, want en", got)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	got := updated.(model)

	if got.currentLangCode() != "tr" {
		t.Fatalf("lang after switch = %q, want tr", got.currentLangCode())
	}
	if !strings.Contains(got.instructionsContent, "Turkce Talimatlar") {
		t.Fatalf("instructionsContent = %q, want Turkish README", got.instructionsContent)
	}
}

func TestUpdateResetProgressClearsState(t *testing.T) {
	root := newCLIAppRoot(t)
	createExercise(t, root, "01_static_server")
	statePath := filepath.Join(t.TempDir(), "state.json")
	state := &AppState{
		Completed:      map[string]bool{"01_static_server": true},
		LastExerciseID: "01_static_server",
		LangCode:       "en",
	}
	if err := state.Save(statePath); err != nil {
		t.Fatal(err)
	}

	m := newUpdateModelFromRoot(t, root, "en")
	m.paths.StateFile = statePath
	m.state = state
	m.exercises[0].Completed = true
	m.cursor = 0
	m.confirmReset = true

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	got := updated.(model)

	if got.confirmReset {
		t.Fatalf("confirmReset = true, want false")
	}
	if got.exercises[0].Completed {
		t.Fatalf("exercise completed = true, want false")
	}
	if got.cursor != 0 {
		t.Fatalf("cursor = %d, want 0", got.cursor)
	}
	loaded, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if len(loaded.Completed) != 0 {
		t.Fatalf("Completed = %#v, want empty map", loaded.Completed)
	}
	if loaded.LastExerciseID != "" {
		t.Fatalf("LastExerciseID = %q, want empty", loaded.LastExerciseID)
	}
	if loaded.LangCode != "en" {
		t.Fatalf("LangCode = %q, want existing language preserved", loaded.LangCode)
	}
}

func newUpdateModelFromRoot(t *testing.T, root, lang string) model {
	t.Helper()

	statePath := filepath.Join(t.TempDir(), "state.json")
	paths := AppPaths{
		WorkDir:   root,
		AppRoot:   root,
		AssetRoot: root,
		StateRoot: filepath.Dir(statePath),
		StateFile: statePath,
	}
	m := initialModelWithOptions(paths, lang)
	if len(m.exercises) == 0 {
		t.Fatalf("test model has no exercises")
	}
	return m
}
