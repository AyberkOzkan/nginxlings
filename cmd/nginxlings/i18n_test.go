package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestLoadI18nLoadsUIAndExerciseTranslations(t *testing.T) {
	root := newAppAssetRoot(t)
	writeI18n(t, root)
	writeFile(t, filepath.Join(root, "i18n", "ui", "en.json"), `{"title":"Custom Title"}`)
	writeFile(t, filepath.Join(root, "i18n", "exercises", "en.json"), `{"01_static_server":"Translated Exercise"}`)

	i18n, err := LoadI18n(root)
	if err != nil {
		t.Fatalf("LoadI18n() error = %v", err)
	}

	if got := i18n.T("en", "title", "fallback"); got != "Custom Title" {
		t.Fatalf("T() = %q, want Custom Title", got)
	}
	if got := i18n.ExerciseName("en", "01_static_server", "fallback"); got != "Translated Exercise" {
		t.Fatalf("ExerciseName() = %q, want Translated Exercise", got)
	}
}

func TestI18nMissingKeysUseFallbacks(t *testing.T) {
	root := newAppAssetRoot(t)
	writeI18n(t, root)

	i18n, err := LoadI18n(root)
	if err != nil {
		t.Fatalf("LoadI18n() error = %v", err)
	}

	if got := i18n.T("en", "missing_key", "fallback ui"); got != "fallback ui" {
		t.Fatalf("T() = %q, want fallback ui", got)
	}
	if got := i18n.ExerciseName("en", "99_missing", "DefaultName"); got != "DefaultName" {
		t.Fatalf("ExerciseName() = %q, want DefaultName", got)
	}
}

func TestLoadI18nIgnoresBrokenTranslationJSONAndFallsBack(t *testing.T) {
	root := newAppAssetRoot(t)
	writeI18n(t, root)
	writeFile(t, filepath.Join(root, "i18n", "ui", "en.json"), `{bad json`)
	writeFile(t, filepath.Join(root, "i18n", "exercises", "en.json"), `{bad json`)

	i18n, err := LoadI18n(root)
	if err != nil {
		t.Fatalf("LoadI18n() error = %v", err)
	}

	if got := i18n.T("en", "title", "fallback title"); got != "fallback title" {
		t.Fatalf("T() = %q, want fallback title", got)
	}
	if got := i18n.ExerciseName("en", "01_static_server", "DefaultName"); got != "DefaultName" {
		t.Fatalf("ExerciseName() = %q, want DefaultName", got)
	}
}

func TestLoadI18nMissingTranslationFilesDoesNotPanic(t *testing.T) {
	root := newAppAssetRoot(t)
	writeFile(t, filepath.Join(root, "i18n", "langs.json"), `[
  { "code": "en", "label": "English" }
]`)

	i18n, err := LoadI18n(root)
	if err != nil {
		t.Fatalf("LoadI18n() error = %v", err)
	}
	if got := i18n.T("en", "title", "fallback title"); got != "fallback title" {
		t.Fatalf("T() = %q, want fallback title", got)
	}
}

func TestLoadI18nBrokenLangsJSONReturnsError(t *testing.T) {
	root := newAppAssetRoot(t)
	writeFile(t, filepath.Join(root, "i18n", "langs.json"), `{bad json`)

	_, err := LoadI18n(root)
	if err == nil || !strings.Contains(err.Error(), "langs.json cannot be parsed") {
		t.Fatalf("LoadI18n() error = %v, want langs.json parse error", err)
	}
}

func TestInitialModelWithMissingI18nDoesNotPanic(t *testing.T) {
	root := newAppAssetRoot(t)
	createExercise(t, root, "01_static_server")
	paths := AppPaths{
		WorkDir:   root,
		AppRoot:   root,
		AssetRoot: root,
		StateRoot: t.TempDir(),
		StateFile: filepath.Join(t.TempDir(), "state.json"),
	}

	model := initialModelWithOptions(paths, "en")

	if len(model.exercises) != 1 {
		t.Fatalf("len(exercises) = %d, want 1", len(model.exercises))
	}
	if !strings.Contains(model.status, "Translations could not be loaded") {
		t.Fatalf("status = %q, want i18n error", model.status)
	}
}

func TestLoadI18nLoadsExerciseHints(t *testing.T) {
	root := newAppAssetRoot(t)
	writeI18n(t, root)
	if err := os.MkdirAll(filepath.Join(root, "i18n", "hints"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "i18n", "hints", "tr.json"), `{"01_static_server":"root dizinine bak"}`)
	writeFile(t, filepath.Join(root, "i18n", "hints", "en.json"), `{broken`)

	i18n, err := LoadI18n(root)
	if err != nil {
		t.Fatalf("LoadI18n() error = %v", err)
	}
	if got := i18n.Hint("tr", "01_static_server"); got != "root dizinine bak" {
		t.Fatalf("Hint(tr) = %q, want translated hint", got)
	}
	if got := i18n.Hint("en", "01_static_server"); got != "" {
		t.Fatalf("Hint(en) = %q, want empty for a broken hints file", got)
	}
	var missing *I18n
	if got := missing.Hint("en", "01_static_server"); got != "" {
		t.Fatalf("nil Hint() = %q, want empty", got)
	}
}

func TestHintKeyTogglesExerciseHintPanel(t *testing.T) {
	m := layoutTestModel(100, 24)
	m.i18n = &I18n{
		langs: []LangMeta{{Code: "tr", Label: "Türkçe"}},
		uiStrings: map[string]map[string]string{"tr": {
			"hint_title":          "İpucu",
			"status_showing_hint": "İpucu gösteriliyor.",
		}},
		hints: map[string]map[string]string{"tr": {"01_static_server": "Önce `root` direktifine bak."}},
	}
	m.exercises = []Exercise{
		{ID: "01_static_server", DefaultName: "01 - Static"},
		{ID: "02_config_layout", DefaultName: "02 - Layout"},
	}
	m.instructionsContent = "Talimat içeriği"
	m.readme.SetContent(m.instructionsContent)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	hint := updated.(model)
	if hint.panel != panelHint {
		t.Fatalf("panel = %v, want hint panel", hint.panel)
	}
	if view := strings.Join(strings.Fields(ansi.Strip(hint.readme.View())), " "); !strings.Contains(view, "Önce root direktifine bak.") {
		t.Fatalf("readme = %q, want exercise hint", view)
	}
	if view := hint.View(); !strings.Contains(view, "İpucu") || hint.status != "İpucu gösteriliyor." {
		t.Fatalf("hint title/status missing: status=%q\n%s", hint.status, view)
	}

	updated, _ = hint.Update(tea.WindowSizeMsg{Width: 90, Height: 20})
	if resized := updated.(model); resized.panel != panelHint {
		t.Fatalf("resize left the hint panel: %v", resized.panel)
	}

	updated, _ = hint.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	if back := updated.(model); back.panel != panelInstructions || !strings.Contains(back.readme.View(), "Talimat içeriği") {
		t.Fatalf("second h did not return to instructions: panel=%v", back.panel)
	}

	updated, _ = hint.Update(tea.KeyMsg{Type: tea.KeyTab})
	if back := updated.(model); back.panel != panelInstructions {
		t.Fatalf("Tab from hint did not return to instructions: panel=%v", back.panel)
	}

	updated, _ = hint.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if moved := updated.(model); moved.panel != panelInstructions {
		t.Fatalf("selection change kept the hint panel: panel=%v", moved.panel)
	}
}

func TestHintFallsBackToGenericHint(t *testing.T) {
	m := layoutTestModel(100, 24)
	m.exercises = []Exercise{{ID: "01_static_server", DefaultName: "01 - Static"}}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	got := updated.(model)

	if view := ansi.Strip(got.readme.View()); !strings.Contains(view, "Read the README carefully") {
		t.Fatalf("readme = %q, want generic hint", view)
	}
}

func TestStatusMessagesFallBackToEnglishWithoutTranslations(t *testing.T) {
	m := model{
		exercises: []Exercise{{ID: "01_static_server", DefaultName: "01 - Static"}},
		readme:    viewport.New(40, 10),
	}

	for _, step := range []struct{ key, want string }{
		{"r", "Reset all progress? (y/n)"},
		{"n", "Reset canceled."},
		{"r", "Reset all progress? (y/n)"},
		{"y", "Progress reset."},
	} {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(step.key)})
		m = updated.(model)
		if m.status != step.want {
			t.Fatalf("after %q status = %q, want %q", step.key, m.status, step.want)
		}
	}
}

func TestStatusMessagesUseCurrentLanguage(t *testing.T) {
	m := model{
		i18n: &I18n{
			langs: []LangMeta{{Code: "tr", Label: "Türkçe"}},
			uiStrings: map[string]map[string]string{"tr": {
				"status_check_already_running": "Kontrol zaten çalışıyor.",
				"status_reset_confirm":         "Sıfırlansın mı? (y/n)",
				"status_state_save_failed":     "Kaydedilemedi: %v",
			}},
		},
		exercises: []Exercise{{ID: "01_static_server", DefaultName: "01 - Static"}},
		readme:    viewport.New(40, 10),
		loading:   true,
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = updated.(model)
	if m.status != "Kontrol zaten çalışıyor." {
		t.Fatalf("status = %q, want translated running message", m.status)
	}

	m.loading = false
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = updated.(model)
	if m.status != "Sıfırlansın mı? (y/n)" {
		t.Fatalf("status = %q, want translated reset prompt", m.status)
	}

	if got := m.uif("status_state_save_failed", "Progress could not be saved: %v", "disk full"); got != "Kaydedilemedi: disk full" {
		t.Fatalf("uif() = %q, want translated format with argument", got)
	}
}

func TestInitialModelStatusUsesSavedLanguage(t *testing.T) {
	root := newCLIAppRoot(t)
	writeFile(t, filepath.Join(root, "i18n", "ui", "tr.json"), `{"status_exercises_load_failed":"Egzersizler yüklenemedi: %v"}`)
	statePath := filepath.Join(t.TempDir(), "state.json")
	if err := (&AppState{LangCode: "tr"}).Save(statePath); err != nil {
		t.Fatal(err)
	}

	m := initialModelWithOptions(AppPaths{AssetRoot: root, StateFile: statePath}, "")

	if got := m.currentLangCode(); got != "tr" {
		t.Fatalf("language = %q, want saved tr", got)
	}
	if !strings.HasPrefix(m.status, "Egzersizler yüklenemedi:") {
		t.Fatalf("status = %q, want exercise load error in the saved language", m.status)
	}
}
