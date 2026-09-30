// Package exercises runs every exercise's real check.sh end to end: the
// starting files must fail, the solution must pass, and the solution written
// on one line must pass too.
//
// Solutions are not part of this repository. Point NGINXLINGS_SOLUTIONS at a
// directory that holds one subdirectory per exercise ID with the solved files,
// for example:
//
//	NGINXLINGS_SOLUTIONS=~/projects/nginxlings-solutions/solutions \
//	  go test ./tests/exercises -v -timeout 30m
//
// Without the variable the tests are skipped. Checks need Docker or a host
// NGINX; host-NGINX checks run in a private network namespace when unshare
// allows it, so services on the machine's ports 8080/8081 do not interfere.
package exercises

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"nginxlings/tests/testutil"
)

const solutionsEnv = "NGINXLINGS_SOLUTIONS"

// checkTimeout bounds one check.sh run; the slowest checks take seconds.
const checkTimeout = 5 * time.Minute

func TestExercisesFailAtStartAndPassWhenSolved(t *testing.T) {
	solutions := os.Getenv(solutionsEnv)
	if solutions == "" {
		t.Skipf("%s is not set; see the package comment or tests/TESTING.md", solutionsEnv)
	}
	if info, err := os.Stat(solutions); err != nil || !info.IsDir() {
		t.Fatalf("%s=%q is not a directory", solutionsEnv, solutions)
	}
	root := testutil.RepoRoot(t)
	exercisesDir := filepath.Join(root, "exercises")
	isolate := canIsolateNetwork()

	for _, id := range testutil.ExerciseIDs(t, exercisesDir) {
		id := id
		t.Run(id, func(t *testing.T) {
			host := usesHostNginx(t, filepath.Join(exercisesDir, id, testutil.CheckScriptName))
			if err := requirePrerequisite(host); err != nil {
				t.Skip(err)
			}
			solution := filepath.Join(solutions, id)
			if _, err := os.Stat(solution); err != nil {
				t.Fatalf("no solution for %s in %s: %v", id, solutions, err)
			}

			t.Run("starting_files_fail", func(t *testing.T) {
				dir := copyExercise(t, exercisesDir, id)
				expectResult(t, runCheck(t, dir, host && isolate), false)
			})
			t.Run("solution_passes", func(t *testing.T) {
				dir := copyExercise(t, exercisesDir, id)
				overlay(t, solution, dir)
				expectResult(t, runCheck(t, dir, host && isolate), true)
			})
			t.Run("one_line_solution_passes", func(t *testing.T) {
				dir := copyExercise(t, exercisesDir, id)
				overlay(t, solution, dir)
				if !onelineConfigs(t, dir) {
					t.Skip("the exercise has no NGINX configuration to rewrite")
				}
				expectResult(t, runCheck(t, dir, host && isolate), true)
			})
		})
	}
}

type checkRun struct {
	output   string
	exitCode int
}

// runCheck runs ./check.sh in dir, inside a private network namespace when
// isolated is true.
func runCheck(t *testing.T, dir string, isolated bool) checkRun {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()
	var cmd *exec.Cmd
	if isolated {
		cmd = exec.CommandContext(ctx, "unshare", isolatedArgs("./check.sh")...)
	} else {
		cmd = exec.CommandContext(ctx, "./check.sh")
	}
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	run := checkRun{output: out.String()}
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		run.exitCode = exitErr.ExitCode()
	default:
		t.Fatalf("run check.sh: %v\n%s", err, out.String())
	}
	if ctx.Err() != nil {
		t.Fatalf("check.sh did not finish within %s\n%s", checkTimeout, tail(run.output, 40))
	}
	return run
}

// isolatedArgs builds an unshare command line that gives the check its own
// network namespace with a loopback interface, then maps the user back to
// the caller so NGINX does not believe it runs as root.
func isolatedArgs(command string) []string {
	uid, gid := os.Getuid(), os.Getgid()
	script := fmt.Sprintf("ip link set lo up && exec unshare -U --map-user=%d --map-group=%d %s", uid, gid, command)
	return []string{"-rn", "bash", "-c", script}
}

func canIsolateNetwork() bool {
	if _, err := exec.LookPath("unshare"); err != nil {
		return false
	}
	return exec.Command("unshare", isolatedArgs("true")...).Run() == nil
}

func requirePrerequisite(host bool) error {
	if host {
		if _, err := exec.LookPath("nginx"); err != nil {
			return errors.New("this check needs NGINX on the host")
		}
		return nil
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		return errors.New("this check needs a running Docker daemon")
	}
	return nil
}

func usesHostNginx(t *testing.T, checkScript string) bool {
	t.Helper()
	data, err := os.ReadFile(checkScript)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.Contains(data, []byte("start_host_nginx")) || bytes.Contains(data, []byte("nginx_test "))
}

var resultLine = regexp.MustCompile(`^\[(PASS|FAIL)\]`)

func expectResult(t *testing.T, run checkRun, pass bool) {
	t.Helper()
	last := testutil.LastLine(run.output)
	want := "[FAIL]"
	if pass {
		want = "[PASS]"
	}
	switch {
	case !resultLine.MatchString(last):
		t.Fatalf("last line %q is not a [PASS]/[FAIL] summary\n%s", last, tail(run.output, 40))
	case !strings.HasPrefix(last, want):
		t.Fatalf("got %q, want %s\n%s", last, want, tail(run.output, 40))
	case pass && run.exitCode != 0, !pass && run.exitCode == 0:
		t.Fatalf("exit code %d does not match %q\n%s", run.exitCode, last, tail(run.output, 40))
	}
	t.Log(last)
}

// copyExercise copies the exercise and the shared library into a temporary
// exercises/ directory, because check.sh sources ../lib/check-lib.sh.
func copyExercise(t *testing.T, exercisesDir, id string) string {
	t.Helper()
	dest := filepath.Join(t.TempDir(), "exercises")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{testutil.LibDir, id} {
		cp := exec.Command("cp", "-a", filepath.Join(exercisesDir, name), dest)
		if out, err := cp.CombinedOutput(); err != nil {
			t.Fatalf("copy %s: %v\n%s", name, err, out)
		}
	}
	dir := filepath.Join(dest, id)
	// Runtime files left by earlier manual runs are not part of the exercise.
	_ = os.RemoveAll(filepath.Join(dir, "logs"))
	return dir
}

// overlay copies the solution's files over the exercise copy.
func overlay(t *testing.T, solution, dir string) {
	t.Helper()
	cp := exec.Command("cp", "-a", solution+"/.", dir)
	if out, err := cp.CombinedOutput(); err != nil {
		t.Fatalf("apply solution: %v\n%s", err, out)
	}
}

// onelineConfigs rewrites nginx.conf and the included conf.d/ and server.d/
// files onto a single line with comment lines dropped. NGINX accepts that
// form, so every check must accept it too. It reports whether it found any
// configuration to rewrite.
func onelineConfigs(t *testing.T, dir string) bool {
	t.Helper()
	var files []string
	for _, pattern := range []string{"nginx.conf", "conf.d/*.conf", "server.d/*.conf"} {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, matches...)
	}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var kept []string
		for _, line := range strings.Split(string(data), "\n") {
			if trimmed := strings.TrimSpace(line); trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				kept = append(kept, trimmed)
			}
		}
		if err := os.WriteFile(path, []byte(strings.Join(kept, " ")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return len(files) > 0
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
