// Package checklib tests exercises/lib/check-lib.sh by running it with bash
// and fake binaries.
package checklib

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"nginxlings/tests/testutil"
)

// These tests run exercises/lib/check-lib.sh with bash and fake binaries.
// They never start real NGINX or Docker and never signal host processes
// other than the ones they start.

type libRun struct {
	output   string
	exitCode int
}

func (r libRun) lastLine() string {
	return testutil.LastLine(r.output)
}

func checkLibPath(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join(testutil.RepoRoot(t), "exercises", testutil.LibDir, "check-lib.sh"))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

// writeLibScript writes a check script that sources the shared library.
func writeLibScript(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	script := fmt.Sprintf(". %q\n%s\n", checkLibPath(t), body)
	path := filepath.Join(dir, "check.sh")
	if err := os.WriteFile(path, []byte("#!/usr/bin/env bash\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func runLibScript(t *testing.T, body string, env ...string) libRun {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", writeLibScript(t, body))
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	run := libRun{output: string(out)}
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		run.exitCode = exitErr.ExitCode()
	default:
		t.Fatalf("run script: %v\n%s", err, out)
	}
	return run
}

// writeFakeBin writes an executable into a new directory and returns a PATH
// value that puts it first.
func writeFakeBin(t *testing.T, name, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/usr/bin/env bash\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return "PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH")
}

func TestCheckLibRequireCmdNamesMissingPrerequisite(t *testing.T) {
	run := runLibScript(t, `require_cmd nginxlings-missing-cmd "needed for the test"`)

	if run.exitCode != 1 {
		t.Fatalf("exit code = %d, want 1\n%s", run.exitCode, run.output)
	}
	want := "[FAIL] Missing prerequisite: 'nginxlings-missing-cmd' is not installed (needed for the test)."
	if run.lastLine() != want {
		t.Fatalf("last line = %q, want %q", run.lastLine(), want)
	}
}

func TestCheckLibNginxRejectionNamesTheFirstError(t *testing.T) {
	tests := []struct {
		name, output, want string
	}{
		{
			name: "docker log format",
			output: `2026/09/29 20:22:34 [emerg] 1#1: no servers are inside upstream in /etc/nginx/nginx.conf:22
nginx: [emerg] no servers are inside upstream in /etc/nginx/nginx.conf:22
nginx: configuration file /etc/nginx/nginx.conf test failed`,
			want: "[FAIL] NGINX rejected the configuration: no servers are inside upstream in /etc/nginx/nginx.conf:22",
		},
		{
			name: "host nginx -t format",
			output: `nginx: [emerg] "listen" directive is not allowed here in /tmp/ex/nginx.conf:31
nginx: configuration file /tmp/ex/nginx.conf test failed`,
			want: `[FAIL] NGINX rejected the configuration: "listen" directive is not allowed here in /tmp/ex/nginx.conf:31`,
		},
		{
			name:   "no emerg line",
			output: "nginx: configuration file test failed",
			want:   "[FAIL] NGINX rejected the configuration: see the errors above.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := runLibScript(t, "fail_nginx_rejected - <<'EOF'\n"+tt.output+"\nEOF")
			if run.exitCode != 1 || run.lastLine() != tt.want {
				t.Fatalf("exit=%d last=%q, want %q", run.exitCode, run.lastLine(), tt.want)
			}
			if !strings.Contains(run.output, tt.output) {
				t.Fatalf("output does not repeat the NGINX output:\n%s", run.output)
			}
		})
	}
}

func TestCheckLibActiveConfigPutsEachDirectiveOnItsOwnLine(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{
			name:  "one-line blocks",
			input: "http { server { listen 8080; server_name a.local b.local; root /srv; } }",
			want:  "http {\n server {\n listen 8080;\n server_name a.local b.local;\n root /srv;\n }\n }\n",
		},
		{
			name:  "comments",
			input: "# configuration file /etc/nginx/nginx.conf:\nuser nginx; # workers\n    #user root;\nevents {}",
			want:  "user nginx;\nevents {\n}\n",
		},
		{
			name:  "quoted strings keep separators and comment marks",
			input: `log_format main '$remote_addr "$request" #1 {x};'; return 200 "a;b{c}\"d\";e";`,
			want:  "log_format main '$remote_addr \"$request\" #1 {x};';\n return 200 \"a;b{c}\\\"d\\\";e\";\n",
		},
		{
			name:  "variables in braces",
			input: `set $x ${uri}-a; add_header X-A "${host}";`,
			want:  "set $x ${uri}-a;\n add_header X-A \"${host}\";\n",
		},
		{
			name:  "status lines pass through",
			input: "nginx: the configuration file /etc/nginx/nginx.conf syntax is ok\nstream { }",
			want:  "nginx: the configuration file /etc/nginx/nginx.conf syntax is ok\nstream {\n }\n",
		},
		{
			name:  "quoted string over two lines",
			input: "log_format main '$a\n    $b';",
			want:  "log_format main '$a\n    $b';\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := runLibScript(t, "active_config <<'EOF'\n"+tt.input+"\nEOF")
			if run.exitCode != 0 || run.output != tt.want {
				t.Fatalf("exit=%d output:\n%q\nwant:\n%q", run.exitCode, run.output, tt.want)
			}
		})
	}

	t.Run("file argument", func(t *testing.T) {
		run := runLibScript(t, `active_config <(printf 'a 1; b 2;\n')`)
		if want := "a 1;\n b 2;\n"; run.exitCode != 0 || run.output != want {
			t.Fatalf("exit=%d output=%q, want %q", run.exitCode, run.output, want)
		}
	})
}

func TestCheckLibDockerBuildNamesTheFailingStep(t *testing.T) {
	tests := []struct {
		name, log, want string
	}{
		{
			name: "failing RUN step",
			log: `#5 [2/2] RUN apt-get install -y nginx
#5 0.170 Reading package lists...
#5 0.181 E: Unable to locate package nginx
#5 ERROR: process "/bin/sh -c apt-get install -y nginx" did not complete successfully: exit code: 100
ERROR: failed to build: failed to solve: process "/bin/sh -c apt-get install -y nginx" did not complete successfully: exit code: 100`,
			want: "[FAIL] docker build failed for img: E: Unable to locate package nginx",
		},
		{
			name: "Dockerfile parse error",
			log: `#1 [internal] load build definition from Dockerfile
#1 transferring dockerfile: 120B done
ERROR: failed to build: failed to solve: dockerfile parse error on line 3: unknown instruction: RUNN`,
			want: "[FAIL] docker build failed for img: dockerfile parse error on line 3: unknown instruction: RUNN",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logPath := filepath.Join(t.TempDir(), "build.log")
			if err := os.WriteFile(logPath, []byte(tt.log+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			fakePath := writeFakeBin(t, "docker", fmt.Sprintf("cat %q\nexit 1\n", logPath))
			run := runLibScript(t, `DOCKER_LABEL=test; docker_build img .`, fakePath, "DOCKER_BIN=docker")
			if run.exitCode != 1 || run.lastLine() != tt.want {
				t.Fatalf("exit=%d last=%q, want %q", run.exitCode, run.lastLine(), tt.want)
			}
		})
	}
}

func TestCheckLibUnexpectedErrorEndsWithOneFailLine(t *testing.T) {
	run := runLibScript(t, "echo '[1/2] Starting...'\nvalue=\"$(false)\"\necho unreachable")

	if run.exitCode == 0 {
		t.Fatalf("exit code = 0, want failure\n%s", run.output)
	}
	if !strings.HasPrefix(run.lastLine(), "[FAIL] Unexpected error (exit 1) at check.sh line") {
		t.Fatalf("last line = %q, want unexpected-error summary", run.lastLine())
	}
	if got := strings.Count(run.output, "[FAIL]"); got != 1 {
		t.Fatalf("[FAIL] lines = %d, want 1\n%s", got, run.output)
	}
	if strings.Contains(run.output, "unreachable") {
		t.Fatalf("script continued after the error\n%s", run.output)
	}
}

func TestCheckLibCleansUpOnFailAndOnSignal(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "hook-ran")
	tmpRecord := filepath.Join(t.TempDir(), "tmp-path")

	run := runLibScript(t, fmt.Sprintf(`
temp_file scratch
printf '%%s' "${scratch}" > %q
on_exit "touch %q"
fail "stopping"`, tmpRecord, marker))
	if run.exitCode != 1 || run.lastLine() != "[FAIL] stopping" {
		t.Fatalf("exit=%d last=%q", run.exitCode, run.lastLine())
	}
	assertExists(t, marker)
	tmpPath, err := os.ReadFile(tmpRecord)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(string(tmpPath)); !os.IsNotExist(err) {
		t.Fatalf("temp file %s survived exit: %v", tmpPath, err)
	}

	signalMarker := filepath.Join(t.TempDir(), "signal-hook-ran")
	ready := filepath.Join(t.TempDir(), "ready")
	cmd := exec.Command("bash", writeLibScript(t, fmt.Sprintf(`
on_exit "touch %q"
touch %q
while :; do sleep 0.05; done`, signalMarker, ready)))
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	testutil.WaitForFile(t, ready)
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	err = cmd.Wait()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 143 {
		t.Fatalf("wait error = %v, want exit status 143", err)
	}
	assertExists(t, signalMarker)
}

func TestCheckLibRequireFreePortReportsBusyPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)

	busy := runLibScript(t, `require_free_port `+port+` "the test server"; pass "free"`)
	if busy.exitCode != 1 || !strings.Contains(busy.lastLine(), "Port "+port+" (needed by the test server) is already in use") {
		t.Fatalf("busy port: exit=%d last=%q", busy.exitCode, busy.lastLine())
	}

	listener.Close()
	free := runLibScript(t, `require_free_port `+port+`; pass "free"`)
	if free.exitCode != 0 || free.lastLine() != "[PASS] free" {
		t.Fatalf("free port: exit=%d output=%q", free.exitCode, free.output)
	}
}

func TestCheckLibWaitHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	up := runLibScript(t, `wait_http `+server.URL+` 5 && pass "up"`)
	if up.exitCode != 0 || up.lastLine() != "[PASS] up" {
		t.Fatalf("reachable server: exit=%d output=%q", up.exitCode, up.output)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closedURL := "http://" + listener.Addr().String() + "/"
	listener.Close()
	down := runLibScript(t, `wait_http `+closedURL+` 1 || fail "down"`)
	if down.exitCode != 1 || down.lastLine() != "[FAIL] down" {
		t.Fatalf("closed port: exit=%d output=%q", down.exitCode, down.output)
	}
}

func TestCheckLibStopsTheHostNginxItStarted(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "fake-nginx.pid")
	argsFile := filepath.Join(t.TempDir(), "fake-nginx.args")
	path := writeFakeBin(t, "nginx", fmt.Sprintf(`
echo "$$" > %q
echo "$*" > %q
trap 'exit 0' TERM
while :; do sleep 0.05; done
`, pidFile, argsFile))

	run := runLibScript(t, fmt.Sprintf(`
start_host_nginx /tmp/nginx.conf /tmp/prefix
for i in $(seq 1 100); do [[ -s %q ]] && break; sleep 0.05; done
fail "checked"`, pidFile), path)

	if run.exitCode != 1 || run.lastLine() != "[FAIL] checked" {
		t.Fatalf("exit=%d output=%q", run.exitCode, run.output)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(args)); got != "-c /tmp/nginx.conf -p /tmp/prefix -g daemon off;" {
		t.Fatalf("fake nginx args = %q, want foreground start", got)
	}
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Kill(pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("fake nginx %d still running after the check exited (kill -0 = %v)", pid, err)
	}
}

func TestCheckLibReportsHostNginxStartupFailure(t *testing.T) {
	path := writeFakeBin(t, "nginx", `
echo 'nginx: [emerg] bind() to 0.0.0.0:8080 failed (98: Address already in use)' >&2
exit 1
`)

	run := runLibScript(t, `start_host_nginx /tmp/nginx.conf /tmp/prefix
wait_host_nginx http://127.0.0.1:1/ 5`, path)

	if run.exitCode != 1 {
		t.Fatalf("exit code = %d, want 1\n%s", run.exitCode, run.output)
	}
	if !strings.Contains(run.output, "Address already in use") {
		t.Fatalf("output does not show NGINX stderr:\n%s", run.output)
	}
	if run.lastLine() != "[FAIL] NGINX exited during startup; see the errors above." {
		t.Fatalf("last line = %q", run.lastLine())
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}
