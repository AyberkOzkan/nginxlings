package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadExercisesLoadsRepoExercises(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))

	exercises, err := LoadExercises(root)
	if err != nil {
		t.Fatalf("LoadExercises() error = %v", err)
	}

	if got, want := len(exercises), 18; got != want {
		t.Fatalf("len(exercises) = %d, want %d", got, want)
	}
	if got, want := exercises[0].ID, "01_static_server"; got != want {
		t.Fatalf("first exercise ID = %q, want %q", got, want)
	}
	if got, want := exercises[17].ID, "18_osi_layers_usecases"; got != want {
		t.Fatalf("last exercise ID = %q, want %q", got, want)
	}
	if got, want := exercises[1].DefaultName, "02 - Config Layout ve Default Paths"; got != want {
		t.Fatalf("second exercise DefaultName = %q, want %q", got, want)
	}
}

func TestLoadExercisesSortsByDirectoryName(t *testing.T) {
	root := newExerciseRoot(t)
	createExercise(t, root, "02_second")
	createExercise(t, root, "01_first")
	createExercise(t, root, "10_tenth")

	exercises, err := LoadExercises(root)
	if err != nil {
		t.Fatalf("LoadExercises() error = %v", err)
	}

	got := exerciseIDs(exercises)
	want := []string{"01_first", "02_second", "10_tenth"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("exercise IDs = %v, want %v", got, want)
	}
}

func TestLoadExercisesMissingReadmeEN(t *testing.T) {
	root := newExerciseRoot(t)
	createExercise(t, root, "01_static_server")
	removeExerciseFile(t, root, "01_static_server", "README.en.md")

	_, err := LoadExercises(root)
	if err == nil || !strings.Contains(err.Error(), "missing README.en.md") {
		t.Fatalf("LoadExercises() error = %v, want missing README.en.md", err)
	}
}

func TestLoadExercisesMissingReadmeTR(t *testing.T) {
	root := newExerciseRoot(t)
	createExercise(t, root, "01_static_server")
	removeExerciseFile(t, root, "01_static_server", "README.tr.md")

	_, err := LoadExercises(root)
	if err == nil || !strings.Contains(err.Error(), "missing README.tr.md") {
		t.Fatalf("LoadExercises() error = %v, want missing README.tr.md", err)
	}
}

func TestLoadExercisesMissingCheckScript(t *testing.T) {
	root := newExerciseRoot(t)
	createExercise(t, root, "01_static_server")
	removeExerciseFile(t, root, "01_static_server", "check.sh")

	_, err := LoadExercises(root)
	if err == nil || !strings.Contains(err.Error(), "missing check.sh") {
		t.Fatalf("LoadExercises() error = %v, want missing check.sh", err)
	}
}

func TestLoadExercisesInvalidExerciseDirectory(t *testing.T) {
	root := newExerciseRoot(t)
	createExercise(t, root, "01_static_server")
	if err := os.Mkdir(filepath.Join(root, "exercises", "bad-name"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := LoadExercises(root)
	if err == nil || !strings.Contains(err.Error(), "invalid exercise directory") {
		t.Fatalf("LoadExercises() error = %v, want invalid exercise directory", err)
	}
}

func TestLoadExercisesEmptyExercisesDirectory(t *testing.T) {
	root := newExerciseRoot(t)

	_, err := LoadExercises(root)
	if err == nil || !strings.Contains(err.Error(), "no exercise directories found") {
		t.Fatalf("LoadExercises() error = %v, want no exercise directories found", err)
	}
}

func TestLoadExercisesUsesManifestDefaultName(t *testing.T) {
	root := newExerciseRoot(t)
	createExercise(t, root, "01_static_server")
	writeManifest(t, root, `{
  "exercises": [
    {
      "id": "01_static_server",
      "default_name": "01 - Custom Title"
    }
  ]
}`)

	exercises, err := LoadExercises(root)
	if err != nil {
		t.Fatalf("LoadExercises() error = %v", err)
	}
	if got, want := exercises[0].DefaultName, "01 - Custom Title"; got != want {
		t.Fatalf("DefaultName = %q, want %q", got, want)
	}
}

func TestLoadExercisesManifestOmittingDirectoryUsesFallbackTitle(t *testing.T) {
	root := newExerciseRoot(t)
	createExercise(t, root, "01_static_server")
	createExercise(t, root, "02_config_layout")
	writeManifest(t, root, `{
  "exercises": [
    {
      "id": "01_static_server",
      "default_name": "01 - Custom Title"
    }
  ]
}`)

	exercises, err := LoadExercises(root)
	if err != nil {
		t.Fatalf("LoadExercises() error = %v", err)
	}
	if got, want := exercises[0].DefaultName, "01 - Custom Title"; got != want {
		t.Fatalf("first DefaultName = %q, want %q", got, want)
	}
	if got, want := exercises[1].DefaultName, "02 - Config Layout"; got != want {
		t.Fatalf("second DefaultName = %q, want fallback %q", got, want)
	}
}

func TestLoadExercisesBrokenManifestReturnsClearError(t *testing.T) {
	root := newExerciseRoot(t)
	createExercise(t, root, "01_static_server")
	writeManifest(t, root, `{bad json`)

	_, err := LoadExercises(root)
	if err == nil || !strings.Contains(err.Error(), "exercise manifest cannot be parsed") {
		t.Fatalf("LoadExercises() error = %v, want manifest parse error", err)
	}
}

func TestLoadExercisesManifestReferencesMissingDirectory(t *testing.T) {
	root := newExerciseRoot(t)
	createExercise(t, root, "01_static_server")
	writeManifest(t, root, `{
  "exercises": [
    {
      "id": "01_static_server",
      "default_name": "01 - Custom Title"
    },
    {
      "id": "02_missing",
      "default_name": "02 - Missing"
    }
  ]
}`)

	_, err := LoadExercises(root)
	if err == nil || !strings.Contains(err.Error(), `references missing directory "02_missing"`) {
		t.Fatalf("LoadExercises() error = %v, want missing manifest directory error", err)
	}
}

func TestLoadExercisesFallsBackToTitleFromDirectoryName(t *testing.T) {
	root := newExerciseRoot(t)
	createExercise(t, root, "01_static_server")

	exercises, err := LoadExercises(root)
	if err != nil {
		t.Fatalf("LoadExercises() error = %v", err)
	}
	if got, want := exercises[0].DefaultName, "01 - Static Server"; got != want {
		t.Fatalf("DefaultName = %q, want %q", got, want)
	}
}

func newExerciseRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "exercises"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func createExercise(t *testing.T, root, id string) {
	t.Helper()

	dir := filepath.Join(root, "exercises", id)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"README.en.md", "README.tr.md", "check.sh"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func removeExerciseFile(t *testing.T, root, id, name string) {
	t.Helper()

	if err := os.Remove(filepath.Join(root, "exercises", id, name)); err != nil {
		t.Fatal(err)
	}
}

func writeManifest(t *testing.T, root, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(root, "exercises", "manifest.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func exerciseIDs(exercises []Exercise) []string {
	ids := make([]string, len(exercises))
	for i, exercise := range exercises {
		ids[i] = exercise.ID
	}
	return ids
}
