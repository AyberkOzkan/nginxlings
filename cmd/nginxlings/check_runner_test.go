package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
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

func TestRunExerciseCheckContextTimesOut(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("process-group timeout behavior is Linux-specific")
	}
	dir := t.TempDir()
	writeCheckScript(t, dir, "#!/bin/sh\necho started\nsleep 30\n")
	ex := Exercise{ID: "01_timeout", Path: dir}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	started := time.Now()
	out, err := runExerciseCheckContext(ctx, ex)
	elapsed := time.Since(started)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("runExerciseCheckContext() error = %v, want deadline exceeded", err)
	}
	if elapsed > checkTerminationGrace+2*time.Second {
		t.Fatalf("timeout returned after %s, want at most %s", elapsed, checkTerminationGrace+2*time.Second)
	}
	if !strings.Contains(out, "[FAIL] check.sh timed out") {
		t.Fatalf("output = %q, want understandable timeout failure", out)
	}
}

func TestRunExerciseCheckContextCancellationStopsChildProcess(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("process-group cancellation behavior is Linux-specific")
	}
	dir := t.TempDir()
	writeCheckScript(t, dir, `#!/bin/sh
child=""
cleanup() {
  kill "$child" 2>/dev/null || true
  wait "$child" 2>/dev/null || true
  echo parent-stopped > parent.stopped
  exit 0
}
trap cleanup TERM
(
  trap "echo child-stopped > child.stopped; exit 0" TERM
  while :; do sleep 1; done
) &
child="$!"
echo "$child" > child.pid
wait "$child"
`)
	ex := Exercise{ID: "01_cancel", Path: dir}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type result struct {
		output string
		err    error
	}
	resultCh := make(chan result, 1)
	go func() {
		output, err := runExerciseCheckContext(ctx, ex)
		resultCh <- result{output: output, err: err}
	}()

	childPID := waitForPIDFile(t, filepath.Join(dir, "child.pid"))
	cancel()

	var got result
	select {
	case got = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("canceled check did not return")
	}
	if !errors.Is(got.err, context.Canceled) {
		t.Fatalf("runExerciseCheckContext() error = %v, want canceled", got.err)
	}
	if !strings.Contains(got.output, "[FAIL] check.sh canceled") {
		t.Fatalf("output = %q, want understandable cancellation failure", got.output)
	}
	waitForFile(t, filepath.Join(dir, "child.stopped"))
	waitForFile(t, filepath.Join(dir, "parent.stopped"))
	if err := syscall.Kill(childPID, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("child process %d still exists after cancellation: %v", childPID, err)
	}
}

func waitForPIDFile(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			value := strings.TrimSpace(string(data))
			if value == "" {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			pid, err := strconv.Atoi(value)
			if err != nil {
				t.Fatalf("parse child PID %q: %v", data, err)
			}
			return pid
		}
		if !os.IsNotExist(err) {
			t.Fatalf("read %s: %v", path, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
	return 0
}

func waitForFile(t *testing.T, path string) {
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

func writeCheckScript(t *testing.T, dir, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, checkScriptName), []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}
