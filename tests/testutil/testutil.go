// Package testutil holds helpers shared by the test suites under tests/.
// The suites treat the repository and the built program as a black box, so
// these helpers read files and never import package main.
package testutil

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

// CheckScriptName is the check script every exercise directory contains.
const CheckScriptName = "check.sh"

// LibDir is the reserved directory under exercises/ that holds the shared
// check library; it is not an exercise.
const LibDir = "lib"

// ExerciseIDPattern matches exercise directory names such as 01_static_server.
var ExerciseIDPattern = regexp.MustCompile(`^[0-9]{2}_[a-z0-9_]+$`)

// RepoRoot returns the absolute path of the repository root, found from this
// file's location so it works from any test package directory.
func RepoRoot(t testing.TB) string {
	t.Helper()
	root := RepoRootPath()
	if root == "" {
		t.Fatal("cannot locate testutil source file")
	}
	return root
}

// RepoRootPath is RepoRoot for callers without a testing.TB, such as
// TestMain. It returns "" when the source location is unknown.
func RepoRootPath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// ExerciseIDs returns the exercise directory names in exercisesDir in sorted
// order. It skips hidden directories and the shared library directory, and
// fails on any other name that is not a valid exercise ID.
func ExerciseIDs(t testing.TB, exercisesDir string) []string {
	t.Helper()
	entries, err := os.ReadDir(exercisesDir)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || entry.Name() == LibDir {
			continue
		}
		if !ExerciseIDPattern.MatchString(entry.Name()) {
			t.Fatalf("invalid exercise directory name %q", entry.Name())
		}
		ids = append(ids, entry.Name())
	}
	sort.Strings(ids)
	return ids
}

// AssertContains fails the test when content does not contain want.
func AssertContains(t testing.TB, path, content, want string) {
	t.Helper()
	if !strings.Contains(content, want) {
		t.Fatalf("%s does not contain %q", path, want)
	}
}

// LastLine returns the last non-empty line of s, trimmed.
func LastLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// WaitForFile waits up to three seconds for path to exist.
func WaitForFile(t testing.TB, path string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		} else if !os.IsNotExist(err) {
			t.Fatalf("inspect %s: %v", path, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}
