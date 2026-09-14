package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestUpdateStartsAndClearsCancelableCheck(t *testing.T) {
	dir := t.TempDir()
	writeCheckScript(t, dir, "#!/bin/sh\necho [PASS] done\n")
	m := model{
		exercises: []Exercise{{ID: "01_static_server", DefaultName: "01 - Static", Path: dir}},
		readme:    viewport.New(80, 20),
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	running := updated.(model)
	if cmd == nil {
		t.Fatal("check command is nil")
	}
	if !running.loading || running.checkCancel == nil || running.checkDone == nil {
		t.Fatal("active check lifecycle was not initialized")
	}

	result := cmd()
	updated, _ = running.Update(result)
	finished := updated.(model)
	if finished.loading || finished.checkCancel != nil || finished.checkDone != nil {
		t.Fatal("active check lifecycle was not cleared after completion")
	}
}

func TestUpdateQuitCancelsActiveCheckBeforeExit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	m := model{
		loading:     true,
		checkCancel: cancel,
		checkDone:   done,
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	got := updated.(model)
	if cmd == nil {
		t.Fatal("quit command is nil while a check is active")
	}
	if got.checkCancel != nil || got.checkDone != nil {
		t.Fatal("active check handles were not cleared")
	}

	canceled := make(chan struct{})
	releaseDone := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(canceled)
		<-releaseDone
		close(done)
	}()

	msgCh := make(chan tea.Msg, 1)
	go func() {
		msgCh <- cmd()
	}()
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("quit command did not cancel the active check")
	}
	select {
	case <-msgCh:
		t.Fatal("quit command returned before the active check finished")
	default:
	}

	close(releaseDone)
	select {
	case msg := <-msgCh:
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Fatalf("quit command message = %T, want tea.QuitMsg", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("quit command did not return after the active check finished")
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

func TestUpdateTogglesSuccessfulAndFailedCheckOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		err    error
	}{
		{name: "successful check", output: "[PASS] complete"},
		{name: "failed check", output: "[FAIL] incomplete", err: errors.New("exit status 1")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{
				exercises:           []Exercise{{ID: "01_static", DefaultName: "Static Server"}},
				readme:              viewport.New(80, 20),
				instructionsContent: "Exercise instructions",
			}
			m.readme.SetContent(m.instructionsContent)

			updated, _ := m.Update(checkResultMsg{output: tt.output, err: tt.err, index: 0})
			result := updated.(model)
			if !result.showingCheckOutput {
				t.Fatal("check result did not activate output view")
			}
			if !strings.Contains(result.readme.View(), tt.output) {
				t.Fatalf("readme = %q, want check output %q", result.readme.View(), tt.output)
			}

			updated, cmd := result.Update(tea.KeyMsg{Type: tea.KeyTab})
			instructions := updated.(model)
			if cmd != nil {
				t.Fatalf("toggle command = %v, want nil", cmd)
			}
			if instructions.showingCheckOutput {
				t.Fatal("Tab did not activate instructions view")
			}
			if !strings.Contains(instructions.readme.View(), "Exercise instructions") {
				t.Fatalf("readme = %q, want instructions", instructions.readme.View())
			}

			updated, _ = instructions.Update(tea.KeyMsg{Type: tea.KeyTab})
			output := updated.(model)
			if !output.showingCheckOutput {
				t.Fatal("second Tab did not reactivate check output view")
			}
			if !strings.Contains(output.readme.View(), tt.output) {
				t.Fatalf("readme = %q, want restored check output %q", output.readme.View(), tt.output)
			}
		})
	}
}

func TestUpdateOutputToggleUsesTranslatedLabelsAndHandlesMissingOutput(t *testing.T) {
	i18n := &I18n{
		langs: []LangMeta{{Code: "tr", Label: "Türkçe"}},
		uiStrings: map[string]map[string]string{
			"tr": {
				"instructions_title":          "Talimatlar",
				"check_output_title":          "Son Kontrol Çıktısı",
				"status_showing_instructions": "Egzersiz talimatları gösteriliyor.",
				"status_showing_check_output": "Son kontrol çıktısı gösteriliyor.",
				"status_no_check_output":      "Henüz kontrol çıktısı yok.",
			},
		},
	}
	m := layoutTestModel(100, 24)
	m.i18n = i18n
	m.exercises = []Exercise{{ID: "01_static", DefaultName: "Static Server"}}
	m.instructionsContent = "Talimat içeriği"
	m.readme.SetContent(m.instructionsContent)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	withoutOutput := updated.(model)
	if withoutOutput.showingCheckOutput {
		t.Fatal("empty output activated check output view")
	}
	if withoutOutput.status != "Henüz kontrol çıktısı yok." {
		t.Fatalf("status = %q, want translated empty-output status", withoutOutput.status)
	}

	withoutOutput.lastCheckOutput = "[PASS] tamamlandı"
	updated, _ = withoutOutput.Update(tea.KeyMsg{Type: tea.KeyTab})
	output := updated.(model)
	if output.status != "Son kontrol çıktısı gösteriliyor." {
		t.Fatalf("status = %q, want translated output status", output.status)
	}
	if view := output.View(); !strings.Contains(view, "Son Kontrol Çıktısı") {
		t.Fatalf("view does not contain translated output title:\n%s", view)
	}

	updated, _ = output.Update(tea.KeyMsg{Type: tea.KeyTab})
	instructions := updated.(model)
	if instructions.status != "Egzersiz talimatları gösteriliyor." {
		t.Fatalf("status = %q, want translated instructions status", instructions.status)
	}
	if view := instructions.View(); !strings.Contains(view, "Talimatlar") {
		t.Fatalf("view does not contain translated instructions title:\n%s", view)
	}
}

func TestWindowResizePreservesActiveCheckOutputView(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "README.md"), "# Instructions\n")
	m := layoutTestModel(100, 24)
	m.exercises = []Exercise{{ID: "01_static", DefaultName: "Static Server", Path: dir}}
	m.instructionsContent = "Instructions"
	m.lastCheckOutput = "[PASS] complete"
	m.showingCheckOutput = true
	m.readme.SetContent(m.lastCheckOutput)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 20})
	got := updated.(model)
	if !got.showingCheckOutput {
		t.Fatal("window resize switched away from active check output")
	}
	if !strings.Contains(got.readme.View(), "[PASS] complete") {
		t.Fatalf("readme = %q, want preserved check output", got.readme.View())
	}
	if !strings.Contains(got.instructionsContent, "Instructions") {
		t.Fatalf("instructionsContent = %q, want re-rendered instructions", got.instructionsContent)
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
	m.readme.Width = 48
	m.readme.Height = 16
	m.lastCheckOutput = "[PASS] previous"
	m.showingCheckOutput = true
	m.readme.SetContent(m.lastCheckOutput)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	got := updated.(model)

	if got.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", got.cursor)
	}
	if !strings.Contains(got.instructionsContent, "Second Exercise") {
		t.Fatalf("instructionsContent = %q, want second exercise README", got.instructionsContent)
	}
	if got.showingCheckOutput {
		t.Fatal("selection change did not return to instructions view")
	}
	if !strings.Contains(got.readme.View(), "Second Exercise") {
		t.Fatalf("readme = %q, want selected exercise instructions", got.readme.View())
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
	m.readme.Width = 48
	m.readme.Height = 16
	m.lastCheckOutput = "[PASS] previous"
	m.showingCheckOutput = true
	m.readme.SetContent(m.lastCheckOutput)

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
	if got.showingCheckOutput {
		t.Fatal("language switch did not return to instructions view")
	}
	if !strings.Contains(got.readme.View(), "Turkce Talimatlar") {
		t.Fatalf("readme = %q, want translated instructions", got.readme.View())
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

func TestUpdateRunningStatusRendersOneTranslatedLabel(t *testing.T) {
	tests := []struct {
		name      string
		langIndex int
		width     int
		height    int
		label     string
	}{
		{name: "English", langIndex: 0, width: 80, height: 24, label: "[RUNNING]"},
		{name: "Turkish narrow", langIndex: 1, width: 42, height: 12, label: "[ÇALIŞIYOR]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i18n := &I18n{
				langs: []LangMeta{
					{Code: "en", Label: "English"},
					{Code: "tr", Label: "Türkçe"},
				},
				uiStrings: map[string]map[string]string{
					"en": {"working_label": "[RUNNING]"},
					"tr": {"working_label": "[ÇALIŞIYOR]"},
				},
				exerciseNames: map[string]map[string]string{
					"en": {"01_static": "Static Server"},
					"tr": {"01_static": "Statik Sunucu"},
				},
			}
			m := layoutTestModel(tt.width, tt.height)
			m.i18n = i18n
			m.langIndex = tt.langIndex
			m.exercises = []Exercise{{ID: "01_static", DefaultName: "Static Server"}}

			updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
			running := updated.(model)
			if cmd == nil {
				t.Fatal("check command is nil")
			}
			if running.checkCancel == nil {
				t.Fatal("check cancel function is nil")
			}
			running.checkCancel()
			if strings.Contains(running.status, tt.label) {
				t.Fatalf("semantic status %q contains rendered label %q", running.status, tt.label)
			}

			view := running.View()
			if got := strings.Count(view, tt.label); got != 1 {
				t.Fatalf("running label count = %d, want 1\nview:\n%s", got, view)
			}
			assertRenderedLinesWithin(t, view, tt.width)
		})
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
