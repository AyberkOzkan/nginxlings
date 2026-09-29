package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
