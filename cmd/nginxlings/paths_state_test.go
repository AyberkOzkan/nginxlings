package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveStateFileUsesXDGStateHome(t *testing.T) {
	xdg := t.TempDir()
	getenv := mapEnv(map[string]string{
		envXDGStateHome: xdg,
		envHome:         t.TempDir(),
	})

	stateFile, stateRoot, err := resolveStateFile(getenv)
	if err != nil {
		t.Fatalf("resolveStateFile() error = %v", err)
	}

	wantRoot := filepath.Join(xdg, appDirName)
	if stateRoot != wantRoot {
		t.Fatalf("stateRoot = %q, want %q", stateRoot, wantRoot)
	}
	if stateFile != filepath.Join(wantRoot, stateFileName) {
		t.Fatalf("stateFile = %q, want %q", stateFile, filepath.Join(wantRoot, stateFileName))
	}
}

func TestResolveStateFileFallsBackToHome(t *testing.T) {
	home := t.TempDir()
	getenv := mapEnv(map[string]string{
		envHome: home,
	})

	stateFile, stateRoot, err := resolveStateFile(getenv)
	if err != nil {
		t.Fatalf("resolveStateFile() error = %v", err)
	}

	if strings.Contains(stateRoot, ".nginxlings_state.json") {
		t.Fatalf("stateRoot should not use legacy repo state file: %q", stateRoot)
	}
	if filepath.Dir(stateFile) != stateRoot {
		t.Fatalf("stateFile dir = %q, want stateRoot %q", filepath.Dir(stateFile), stateRoot)
	}
	if filepath.Base(stateFile) != stateFileName {
		t.Fatalf("stateFile base = %q, want %q", filepath.Base(stateFile), stateFileName)
	}
}

func TestResolveStateFileUsesCustomStateRoot(t *testing.T) {
	root := t.TempDir()
	getenv := mapEnv(map[string]string{
		envStateRoot: root,
		envHome:      t.TempDir(),
	})

	stateFile, stateRoot, err := resolveStateFile(getenv)
	if err != nil {
		t.Fatalf("resolveStateFile() error = %v", err)
	}

	if stateRoot != root {
		t.Fatalf("stateRoot = %q, want %q", stateRoot, root)
	}
	if stateFile != filepath.Join(root, stateFileName) {
		t.Fatalf("stateFile = %q, want %q", stateFile, filepath.Join(root, stateFileName))
	}
}

func TestResolveStateFileUsesCustomStateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom-state.json")
	getenv := mapEnv(map[string]string{
		envStateFile: path,
		envHome:      t.TempDir(),
	})

	stateFile, stateRoot, err := resolveStateFile(getenv)
	if err != nil {
		t.Fatalf("resolveStateFile() error = %v", err)
	}

	if stateFile != path {
		t.Fatalf("stateFile = %q, want %q", stateFile, path)
	}
	if stateRoot != filepath.Dir(path) {
		t.Fatalf("stateRoot = %q, want %q", stateRoot, filepath.Dir(path))
	}
}

func TestLoadStateMissingFileReturnsDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	state, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if state.Completed == nil || len(state.Completed) != 0 {
		t.Fatalf("Completed = %#v, want empty initialized map", state.Completed)
	}
}

func TestSaveAndLoadState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "state.json")
	state := &AppState{
		Completed:      map[string]bool{"01_static_server": true},
		LastExerciseID: "01_static_server",
		LangCode:       "tr",
	}

	if err := state.Save(path); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("state mode = %o, want 644", got)
	}
	tempFiles, err := filepath.Glob(filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(tempFiles) != 0 {
		t.Fatalf("temporary state files remain after save: %v", tempFiles)
	}

	loaded, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if !loaded.Completed["01_static_server"] {
		t.Fatalf("Completed[01_static_server] = false, want true")
	}
	if loaded.LastExerciseID != state.LastExerciseID {
		t.Fatalf("LastExerciseID = %q, want %q", loaded.LastExerciseID, state.LastExerciseID)
	}
	if loaded.LangCode != state.LangCode {
		t.Fatalf("LangCode = %q, want %q", loaded.LangCode, state.LangCode)
	}
}

func TestSaveStateRenameFailurePreservesLastValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	original := &AppState{
		Completed:      map[string]bool{"01_static_server": true},
		LastExerciseID: "01_static_server",
		LangCode:       "en",
	}
	if err := original.Save(path); err != nil {
		t.Fatalf("initial Save() error = %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	replacement := &AppState{
		Completed:      map[string]bool{"02_config_layout": true},
		LastExerciseID: "02_config_layout",
		LangCode:       "tr",
	}
	renameErr := errors.New("forced rename failure")
	err = replacement.save(path, func(string, string) error {
		return renameErr
	})
	if !errors.Is(err, renameErr) {
		t.Fatalf("save() error = %v, want forced rename failure", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("state changed after failed replacement\nbefore: %s\nafter: %s", before, after)
	}
	tempFiles, err := filepath.Glob(filepath.Join(dir, ".state.json.tmp-*"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(tempFiles) != 0 {
		t.Fatalf("temporary state files remain after failed save: %v", tempFiles)
	}
}

func TestLoadStateCorruptJSONReturnsDefaultAndError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}

	state, err := LoadState(path)
	if err == nil {
		t.Fatalf("LoadState() error = nil, want parse error")
	}
	if state == nil || state.Completed == nil {
		t.Fatalf("LoadState() state = %#v, want default initialized state", state)
	}
}

func TestLoadStatePartialJSONInitializesCompletedMap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(`{"last_exercise_id":"02_config_layout","lang_code":"tr"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	state, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if state.Completed == nil {
		t.Fatalf("Completed = nil, want initialized map")
	}
	if state.LastExerciseID != "02_config_layout" {
		t.Fatalf("LastExerciseID = %q, want 02_config_layout", state.LastExerciseID)
	}
	if state.LangCode != "tr" {
		t.Fatalf("LangCode = %q, want tr", state.LangCode)
	}
}

func TestResolveAppPathsFindsAssetRootFromRepoWorkDir(t *testing.T) {
	root := newAppAssetRoot(t)
	workDir := filepath.Join(root, "cmd", "nginxlings")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	getenv := mapEnv(map[string]string{
		envStateRoot: t.TempDir(),
		envHome:      t.TempDir(),
	})

	paths, err := resolveAppPaths(workDir, "", getenv)
	if err != nil {
		t.Fatalf("resolveAppPaths() error = %v", err)
	}
	if paths.AssetRoot != root {
		t.Fatalf("AssetRoot = %q, want %q", paths.AssetRoot, root)
	}
	if paths.AppRoot != root {
		t.Fatalf("AppRoot = %q, want %q", paths.AppRoot, root)
	}
}

func TestResolveAppPathsUsesAssetRootOverride(t *testing.T) {
	root := newAppAssetRoot(t)
	getenv := mapEnv(map[string]string{
		envAssetRoot: root,
		envStateRoot: t.TempDir(),
		envHome:      t.TempDir(),
	})

	paths, err := resolveAppPaths(t.TempDir(), "", getenv)
	if err != nil {
		t.Fatalf("resolveAppPaths() error = %v", err)
	}
	if paths.AssetRoot != root {
		t.Fatalf("AssetRoot = %q, want %q", paths.AssetRoot, root)
	}
}

func TestExistingLegacyStateFileDoesNotBlockNewStatePath(t *testing.T) {
	root := newAppAssetRoot(t)
	if err := os.WriteFile(filepath.Join(root, legacyStateName), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	stateRoot := t.TempDir()
	getenv := mapEnv(map[string]string{
		envAssetRoot: root,
		envStateRoot: stateRoot,
		envHome:      t.TempDir(),
	})

	paths, err := resolveAppPaths(root, "", getenv)
	if err != nil {
		t.Fatalf("resolveAppPaths() error = %v", err)
	}
	if paths.StateFile == filepath.Join(root, legacyStateName) {
		t.Fatalf("StateFile = legacy repo path %q", paths.StateFile)
	}
	if _, err := LoadState(paths.StateFile); err != nil {
		t.Fatalf("LoadState(new state file) error = %v", err)
	}
}

func newAppAssetRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	for _, name := range []string{"exercises", "i18n"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func mapEnv(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}
