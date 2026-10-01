package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveAppPathsUsesTheShareDirOfAnInstalledPackage(t *testing.T) {
	prefix := t.TempDir()
	share := filepath.Join(prefix, "share", "nginxlings")
	for _, dir := range []string{"exercises", "i18n"} {
		if err := os.MkdirAll(filepath.Join(share, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	home := t.TempDir()
	getenv := mapEnv(map[string]string{envHome: home, envStateRoot: t.TempDir()})

	paths, err := resolveAppPaths(t.TempDir(), filepath.Join(prefix, "bin", "nginxlings"), getenv)
	if err != nil {
		t.Fatalf("resolveAppPaths() error = %v", err)
	}
	if paths.AssetRoot != share {
		t.Fatalf("AssetRoot = %q, want %q", paths.AssetRoot, share)
	}
	if want := filepath.Join(home, "nginxlings"); paths.Workspace != want || paths.ExercisesDir() != want {
		t.Fatalf("Workspace = %q, ExercisesDir = %q, want %q", paths.Workspace, paths.ExercisesDir(), want)
	}
}

func TestResolveAppPathsEditsInPlaceOutsideAPackage(t *testing.T) {
	root := newAppAssetRoot(t)
	getenv := mapEnv(map[string]string{envAssetRoot: root, envHome: t.TempDir(), envStateRoot: t.TempDir()})

	paths, err := resolveAppPaths(t.TempDir(), "", getenv)
	if err != nil {
		t.Fatalf("resolveAppPaths() error = %v", err)
	}
	if paths.Workspace != "" || paths.ExercisesDir() != filepath.Join(root, "exercises") {
		t.Fatalf("Workspace = %q, ExercisesDir = %q, want in-place exercises", paths.Workspace, paths.ExercisesDir())
	}

	workspace := t.TempDir()
	getenv = mapEnv(map[string]string{envAssetRoot: root, envWorkspace: workspace, envStateRoot: t.TempDir()})
	paths, err = resolveAppPaths(t.TempDir(), "", getenv)
	if err != nil {
		t.Fatalf("resolveAppPaths() error = %v", err)
	}
	if paths.ExercisesDir() != workspace {
		t.Fatalf("ExercisesDir = %q, want NGINXLINGS_WORKSPACE %q", paths.ExercisesDir(), workspace)
	}
}

// newTemplate writes a shipped exercises/ tree and returns its path.
func newTemplate(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "exercises")
	for rel, content := range files {
		mode := os.FileMode(0o644)
		if strings.HasSuffix(rel, ".sh") {
			mode = 0o755
		}
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func readWorkspaceFile(t *testing.T, workspace, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(workspace, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

func writeWorkspaceTestFile(t *testing.T, workspace, rel, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(workspace, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

var shippedV1 = map[string]string{
	"manifest.json":                 `{"exercises":[]}`,
	"lib/check-lib.sh":              "lib v1",
	"01_static/check.sh":            "check v1",
	"01_static/README.en.md":        "readme v1",
	"01_static/nginx.conf":          "conf v1",
	"01_static/index.html":          "page v1",
	"01_static/helper.py":           "helper v1",
	"02_proxy/check.sh":             "check v1",
	"02_proxy/nginx.conf":           "proxy v1",
	"02_proxy/conf.d/default.conf":  "default v1",
	"02_proxy/conf.d/obsolete.conf": "obsolete v1",
}

func TestSyncWorkspaceCopiesEverythingTheFirstTime(t *testing.T) {
	template := newTemplate(t, shippedV1)
	workspace := filepath.Join(t.TempDir(), "nginxlings")

	if err := SyncWorkspace(template, workspace); err != nil {
		t.Fatalf("SyncWorkspace() error = %v", err)
	}
	for rel, content := range shippedV1 {
		if got := readWorkspaceFile(t, workspace, rel); got != content {
			t.Fatalf("%s = %q, want %q", rel, got, content)
		}
	}
	info, err := os.Stat(filepath.Join(workspace, "01_static", "check.sh"))
	if err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("check.sh mode = %v (%v), want 0755", info.Mode().Perm(), err)
	}
	if _, err := LoadExercisesFrom(workspace); err == nil || !strings.Contains(err.Error(), "README.tr.md") {
		// The fixture has no Turkish README; this only proves the loader reads the workspace.
		t.Fatalf("LoadExercisesFrom(workspace) error = %v, want the missing README.tr.md", err)
	}
}

func TestSyncWorkspaceUpdateKeepsTheLearnersChanges(t *testing.T) {
	template := newTemplate(t, shippedV1)
	workspace := filepath.Join(t.TempDir(), "nginxlings")
	if err := SyncWorkspace(template, workspace); err != nil {
		t.Fatal(err)
	}

	// The learner edits their config, deletes a page, and tampers with the check.
	writeWorkspaceTestFile(t, workspace, "01_static/nginx.conf", "my solution")
	if err := os.Remove(filepath.Join(workspace, "01_static", "index.html")); err != nil {
		t.Fatal(err)
	}
	writeWorkspaceTestFile(t, workspace, "01_static/check.sh", "exit 0")

	// A new version changes every shipped file, drops one and adds an exercise.
	v2 := map[string]string{}
	for rel, content := range shippedV1 {
		v2[rel] = strings.Replace(content, "v1", "v2", 1)
	}
	delete(v2, "02_proxy/conf.d/obsolete.conf")
	v2["03_new/check.sh"] = "check v2"
	v2["03_new/nginx.conf"] = "new v2"
	if err := SyncWorkspace(newTemplate(t, v2), workspace); err != nil {
		t.Fatalf("SyncWorkspace() update error = %v", err)
	}

	want := map[string]string{
		"01_static/nginx.conf":         "my solution", // learner's work is kept
		"01_static/check.sh":           "check v2",    // app-owned files are replaced
		"01_static/README.en.md":       "readme v2",
		"lib/check-lib.sh":             "lib v2",
		"01_static/helper.py":          "helper v2", // untouched shipped files follow the update
		"02_proxy/conf.d/default.conf": "default v2",
		"03_new/nginx.conf":            "new v2", // new exercises arrive
	}
	for rel, content := range want {
		if got := readWorkspaceFile(t, workspace, rel); got != content {
			t.Errorf("%s = %q, want %q", rel, got, content)
		}
	}
	for _, rel := range []string{"01_static/index.html", "02_proxy/conf.d/obsolete.conf"} {
		if _, err := os.Stat(filepath.Join(workspace, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Errorf("%s exists (%v), want it gone", rel, err)
		}
	}
}

func TestSyncWorkspaceKeepsAChangedFileTheNewVersionDrops(t *testing.T) {
	template := newTemplate(t, shippedV1)
	workspace := filepath.Join(t.TempDir(), "nginxlings")
	if err := SyncWorkspace(template, workspace); err != nil {
		t.Fatal(err)
	}
	writeWorkspaceTestFile(t, workspace, "02_proxy/conf.d/obsolete.conf", "learner notes")

	v2 := map[string]string{}
	for rel, content := range shippedV1 {
		v2[rel] = content
	}
	delete(v2, "02_proxy/conf.d/obsolete.conf")
	if err := SyncWorkspace(newTemplate(t, v2), workspace); err != nil {
		t.Fatal(err)
	}
	if got := readWorkspaceFile(t, workspace, "02_proxy/conf.d/obsolete.conf"); got != "learner notes" {
		t.Fatalf("changed dropped file = %q, want the learner's content kept", got)
	}
}

func TestSyncWorkspaceRefusesAFolderItDidNotCreate(t *testing.T) {
	template := newTemplate(t, shippedV1)
	workspace := t.TempDir()
	writeWorkspaceTestFile(t, workspace, "thesis.txt", "do not touch")

	err := SyncWorkspace(template, workspace)
	if err == nil || !strings.Contains(err.Error(), "not a nginxlings exercise folder") {
		t.Fatalf("SyncWorkspace() error = %v, want a refusal", err)
	}
	if entries, _ := os.ReadDir(workspace); len(entries) != 1 {
		t.Fatalf("workspace has %d entries, want only the existing file", len(entries))
	}
}

func TestResetExerciseRestoresTheStartingFiles(t *testing.T) {
	template := newTemplate(t, shippedV1)
	workspace := filepath.Join(t.TempDir(), "nginxlings")
	if err := SyncWorkspace(template, workspace); err != nil {
		t.Fatal(err)
	}
	writeWorkspaceTestFile(t, workspace, "01_static/nginx.conf", "broken attempt")
	writeWorkspaceTestFile(t, workspace, "01_static/scratch.txt", "learner file")
	writeWorkspaceTestFile(t, workspace, "02_proxy/nginx.conf", "other work")

	if err := ResetExercise(template, workspace, "01_static"); err != nil {
		t.Fatalf("ResetExercise() error = %v", err)
	}
	if got := readWorkspaceFile(t, workspace, "01_static/nginx.conf"); got != "conf v1" {
		t.Fatalf("nginx.conf = %q, want the starting file", got)
	}
	if _, err := os.Stat(filepath.Join(workspace, "01_static", "scratch.txt")); !os.IsNotExist(err) {
		t.Fatalf("scratch.txt survived the reset (%v)", err)
	}
	if got := readWorkspaceFile(t, workspace, "02_proxy/nginx.conf"); got != "other work" {
		t.Fatalf("another exercise changed: %q", got)
	}

	// The reset file counts as unchanged, so the next update may replace it.
	v2 := map[string]string{}
	for rel, content := range shippedV1 {
		v2[rel] = content
	}
	v2["01_static/nginx.conf"] = "conf v2"
	if err := SyncWorkspace(newTemplate(t, v2), workspace); err != nil {
		t.Fatal(err)
	}
	if got := readWorkspaceFile(t, workspace, "01_static/nginx.conf"); got != "conf v2" {
		t.Fatalf("nginx.conf after update = %q, want conf v2", got)
	}

	if err := ResetExercise(template, workspace, "../etc"); err == nil {
		t.Fatal("ResetExercise accepted an invalid id")
	}
}
