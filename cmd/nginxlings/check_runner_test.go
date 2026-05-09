package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidateCheckScriptMissing(t *testing.T) {
	ex := Exercise{ID: "01_missing", Path: t.TempDir()}

	err := validateCheckScript(ex, "linux")
	if err == nil || !strings.Contains(err.Error(), "check.sh not found") {
		t.Fatalf("validateCheckScript() error = %v, want missing check.sh", err)
	}
}

func TestValidateCheckScriptNotExecutable(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, checkScriptName), []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ex := Exercise{ID: "01_not_exec", Path: dir}

	err := validateCheckScript(ex, "linux")
	if err == nil || !strings.Contains(err.Error(), "not executable") {
		t.Fatalf("validateCheckScript() error = %v, want not executable", err)
	}
}

func TestValidateCheckScriptUnsupportedWindows(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, checkScriptName), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ex := Exercise{ID: "01_windows", Path: dir}

	err := validateCheckScript(ex, "windows")
	if err == nil || !strings.Contains(err.Error(), "Native Windows") && !strings.Contains(err.Error(), "native Windows") {
		t.Fatalf("validateCheckScript() error = %v, want Windows unsupported", err)
	}
}

func TestRunExerciseCheckSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("check.sh execution is intentionally unsupported on native Windows")
	}
	dir := t.TempDir()
	writeCheckScript(t, dir, "#!/bin/sh\necho '[PASS] fake ok'\n")
	ex := Exercise{ID: "01_success", Path: dir}

	out, err := runExerciseCheck(ex)
	if err != nil {
		t.Fatalf("runExerciseCheck() error = %v; out=%q", err, out)
	}
	if !strings.Contains(out, "[PASS] fake ok") {
		t.Fatalf("output = %q, want pass marker", out)
	}
}

func TestRunExerciseCheckFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("check.sh execution is intentionally unsupported on native Windows")
	}
	dir := t.TempDir()
	writeCheckScript(t, dir, "#!/bin/sh\necho '[FAIL] fake bad'\nexit 7\n")
	ex := Exercise{ID: "01_failure", Path: dir}

	out, err := runExerciseCheck(ex)
	if err == nil {
		t.Fatalf("runExerciseCheck() error = nil, want failure")
	}
	if !strings.Contains(out, "[FAIL] fake bad") {
		t.Fatalf("output = %q, want fail marker", out)
	}
}

func writeCheckScript(t *testing.T, dir, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, checkScriptName), []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}
