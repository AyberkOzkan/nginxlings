package main

import (
	"path/filepath"
	"strings"
	"testing"
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
	if err == nil || !strings.Contains(err.Error(), "langs.json parse") {
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
	if !strings.Contains(model.status, "i18n") {
		t.Fatalf("status = %q, want i18n error", model.status)
	}
}
