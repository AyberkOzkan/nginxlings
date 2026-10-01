package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

const checkScriptName = "check.sh"

// sbinDirs hold programs such as nginx that Debian keeps out of a regular
// user's PATH.
var sbinDirs = []string{"/usr/local/sbin", "/usr/sbin", "/sbin"}

// withSbinPath returns env with the sbin directories appended to PATH, so a
// check finds nginx wherever the distribution installed it.
func withSbinPath(env []string) []string {
	out := make([]string, 0, len(env)+1)
	path := ""
	for _, kv := range env {
		if value, ok := strings.CutPrefix(kv, "PATH="); ok {
			path = value
			continue
		}
		out = append(out, kv)
	}
	have := make(map[string]bool)
	for _, dir := range filepath.SplitList(path) {
		have[dir] = true
	}
	for _, dir := range sbinDirs {
		if !have[dir] {
			if path != "" {
				path += string(os.PathListSeparator)
			}
			path += dir
		}
	}
	return append(out, "PATH="+path)
}

// lookPathWithSbin finds cmd on PATH or in the sbin directories.
func lookPathWithSbin(cmd string) (string, error) {
	if path, err := exec.LookPath(cmd); err == nil {
		return path, nil
	}
	for _, dir := range sbinDirs {
		path := filepath.Join(dir, cmd)
		if info, err := os.Stat(path); err == nil && !info.IsDir() && info.Mode().Perm()&0o111 != 0 {
			return path, nil
		}
	}
	return exec.LookPath(cmd)
}

const defaultCheckTimeout = 10 * time.Minute
const checkTerminationGrace = 2 * time.Second

func runExerciseCheck(ex Exercise) (string, error) {
	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	ctx, cancel := context.WithTimeout(signalCtx, defaultCheckTimeout)
	defer cancel()
	return runExerciseCheckContext(ctx, ex)
}

func runExerciseCheckContext(ctx context.Context, ex Exercise) (string, error) {
	if err := validateCheckScript(ex, runtime.GOOS); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", checkContextError(err)
	}

	cmd := exec.Command("./" + checkScriptName)
	cmd.Dir = ex.Path
	cmd.Env = withSbinPath(os.Environ())
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Start(); err != nil {
		return buf.String(), fmt.Errorf("start %s for exercise %s: %w", checkScriptName, ex.ID, err)
	}

	wait := make(chan error, 1)
	go func() {
		wait <- cmd.Wait()
	}()

	select {
	case err := <-wait:
		return buf.String(), err
	case <-ctx.Done():
		return stopCheckProcessGroup(ctx.Err(), cmd.Process, wait, &buf)
	}
}

func stopCheckProcessGroup(contextErr error, process *os.Process, wait <-chan error, buf *bytes.Buffer) (string, error) {
	terminateErr := signalCheckProcessGroup(process.Pid, syscall.SIGTERM)
	timer := time.NewTimer(checkTerminationGrace)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer timer.Stop()
	defer ticker.Stop()

	leaderDone := false
	for processGroupExists(process.Pid) {
		select {
		case <-wait:
			leaderDone = true
		case <-ticker.C:
		case <-timer.C:
			killErr := signalCheckProcessGroup(process.Pid, syscall.SIGKILL)
			if terminateErr == nil {
				terminateErr = killErr
			}
			if !leaderDone {
				<-wait
			}
			goto stopped
		}
	}
	if !leaderDone {
		<-wait
	}

stopped:
	err := checkContextError(contextErr)
	if terminateErr != nil {
		err = fmt.Errorf("%w (failed to terminate process group: %v)", err, terminateErr)
	}
	output := buf.String()
	if output != "" && !strings.HasSuffix(output, "\n") {
		output += "\n"
	}
	output += "[FAIL] " + err.Error() + "\n"
	return output, err
}

func processGroupExists(pid int) bool {
	err := syscall.Kill(-pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func signalCheckProcessGroup(pid int, signal syscall.Signal) error {
	err := syscall.Kill(-pid, signal)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}

func checkContextError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%s timed out: %w", checkScriptName, context.DeadlineExceeded)
	}
	return fmt.Errorf("%s canceled: %w", checkScriptName, context.Canceled)
}

func validateCheckScript(ex Exercise, goos string) error {
	if goos == "windows" {
		return fmt.Errorf("native Windows is not supported for %s; run Nginxlings in WSL2 or another POSIX shell environment", checkScriptName)
	}

	path := filepath.Join(ex.Path, checkScriptName)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s not found for exercise %s", checkScriptName, ex.ID)
		}
		return fmt.Errorf("%s cannot be read for exercise %s: %w", checkScriptName, ex.ID, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory for exercise %s", checkScriptName, ex.ID)
	}
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("%s is not executable for exercise %s; run chmod +x %s", checkScriptName, ex.ID, path)
	}
	return nil
}
