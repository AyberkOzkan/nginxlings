package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRequiredCommandsOfTheShippedExercises(t *testing.T) {
	exercises, err := LoadExercises(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	needs := requiredCommands(exercises)
	want := map[string]string{
		"bash":    "01–18",
		"curl":    "01–07, 09–18",
		"nginx":   "01–03, 06",
		"python3": "06, 08, 18",
		"docker":  "04–05, 07–18",
	}
	for cmd, ranges := range want {
		if got := exerciseRanges(needs[cmd]); got != ranges {
			t.Errorf("%s needed by %q, want %q", cmd, got, ranges)
		}
	}
	if len(needs) != len(want) {
		t.Errorf("commands = %v, want only %v", needs, want)
	}
}

func TestExerciseRanges(t *testing.T) {
	if got := exerciseRanges([]string{"07", "04", "05", "09", "08", "12"}); got != "04–05, 07–09, 12" {
		t.Fatalf("exerciseRanges() = %q", got)
	}
}

func TestDetectPackageManager(t *testing.T) {
	tests := []struct {
		name    string
		release map[string]string
		docker  string
	}{
		{"ubuntu", map[string]string{"ID": "ubuntu", "ID_LIKE": "debian"}, "sudo apt install -y docker.io"},
		{"mint", map[string]string{"ID": "linuxmint", "ID_LIKE": "ubuntu debian"}, "sudo apt install -y docker.io"},
		{"fedora", map[string]string{"ID": "fedora"}, "sudo dnf install -y moby-engine"},
		{"rocky", map[string]string{"ID": "rocky", "ID_LIKE": "rhel centos fedora"}, "sudo dnf install -y podman-docker"},
		{"tumbleweed", map[string]string{"ID": "opensuse-tumbleweed", "ID_LIKE": "opensuse suse"}, "sudo zypper --non-interactive install docker"},
		{"manjaro", map[string]string{"ID": "manjaro", "ID_LIKE": "arch"}, "sudo pacman -S --needed --noconfirm docker"},
		{"alpine", map[string]string{"ID": "alpine"}, "sudo apk add docker"},
		{"unknown", map[string]string{"ID": "plan9"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shellCommand(detectPackageManager(tt.release).installCommand("docker")); got != tt.docker {
				t.Fatalf("docker install = %q, want %q", got, tt.docker)
			}
		})
	}
	if got := shellCommand(detectPackageManager(map[string]string{"ID": "arch"}).installCommand("python3")); got != "sudo pacman -S --needed --noconfirm python" {
		t.Fatalf("arch python3 = %q", got)
	}
}

func TestReadOSRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "os-release")
	writeFile(t, path, "NAME=\"Fedora Linux\"\nID=fedora\n# comment\nID_LIKE='rhel'\n")
	got := readOSRelease(path)
	if got["ID"] != "fedora" || got["NAME"] != "Fedora Linux" || got["ID_LIKE"] != "rhel" {
		t.Fatalf("readOSRelease() = %v", got)
	}
	if len(readOSRelease(filepath.Join(t.TempDir(), "missing"))) != 0 {
		t.Fatal("a missing file should give an empty map")
	}
}

// fakeSetupEnv has every command except missing; Docker answers with
// dockerOut and dockerErr.
func fakeSetupEnv(missing string, dockerOut string, dockerErr error, listed, active bool) setupEnv {
	return setupEnv{
		lookPath: func(cmd string) (string, error) {
			if cmd == missing {
				return "", errors.New("not found")
			}
			return "/usr/bin/" + cmd, nil
		},
		dockerInfo:    func(context.Context) (string, error) { return dockerOut, dockerErr },
		dockerVersion: func(context.Context) (string, error) { return "Docker version 29.0.0, build 1", nil },
		dockerGroup:   func() (bool, bool) { return listed, active },
		osRelease:     map[string]string{"ID": "ubuntu", "ID_LIKE": "debian"},
		userName:      "learner",
	}
}

// setupExercises are two exercises: 01 runs NGINX on the host, 02 needs Docker.
func setupExercises(t *testing.T) []Exercise {
	t.Helper()
	dir := t.TempDir()
	scripts := map[string]string{
		"01_host":   "require_cmd \"${NGINX_BIN}\" \"install NGINX\"\nrequire_cmd curl\n",
		"02_docker": "  require_docker\nrequire_cmd \"${PYTHON_BIN}\"\n# require_cmd ignored-in-a-comment\n",
	}
	var exercises []Exercise
	for id, script := range scripts {
		path := filepath.Join(dir, id)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(path, checkScriptName), script)
		exercises = append(exercises, Exercise{ID: id, Path: path})
	}
	return exercises
}

func TestRunSetupCheckExplainsEachProblem(t *testing.T) {
	denied := errors.New("exit status 1")
	tests := []struct {
		name     string
		env      setupEnv
		kind     problemKind
		fix      string
		commands string
	}{
		{"missing nginx", fakeSetupEnv("nginx", "", nil, false, false), problemMissingCommand, "sudo apt install -y nginx", "01"},
		{"not in the docker group", fakeSetupEnv("", "permission denied while trying to connect to the Docker daemon socket", denied, false, false), problemDockerGroup, "sudo usermod -aG docker learner", "02"},
		{"added but not logged in again", fakeSetupEnv("", "Got permission denied", denied, true, false), problemDockerRelogin, "", "02"},
		{"daemon stopped", fakeSetupEnv("", "Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?", denied, true, true), problemDockerStopped, "sudo systemctl enable --now docker", "02"},
		{"daemon stopped (Docker 29)", fakeSetupEnv("", "failed to connect to the docker API at unix:///var/run/docker.sock; check if the path is correct and if the daemon is running: dial unix /var/run/docker.sock: connect: no such file or directory", denied, true, true), problemDockerStopped, "sudo systemctl enable --now docker", "02"},
		{"daemon stopped (old Debian client)", fakeSetupEnv("", "Client:\n Version: 20.10.24+dfsg1\n\nServer:\nERROR: Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?\nerrors pretty printing info\n", denied, true, true), problemDockerStopped, "sudo systemctl enable --now docker", "02"},
		{"other", fakeSetupEnv("", "Emulate Docker CLI using podman. Create /etc/containers/nodocker to quiet msg.\nClient: x\nerror during connect: boom\nmore", denied, true, true), problemDockerOther, "", "02"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := runSetupCheck(context.Background(), setupExercises(t), tt.env)
			if len(report.problems) != 1 {
				t.Fatalf("problems = %+v, want one", report.problems)
			}
			p := report.problems[0]
			if p.kind != tt.kind || shellCommand(p.fix) != tt.fix || exerciseRanges(p.exercises) != tt.commands {
				t.Fatalf("problem = %+v (fix %q), want kind %v fix %q for %s", p, shellCommand(p.fix), tt.kind, tt.fix, tt.commands)
			}
			if tt.kind == problemDockerOther && p.detail != "error during connect: boom" {
				t.Fatalf("detail = %q, want the first line of Docker's message", p.detail)
			}
		})
	}

	report := runSetupCheck(context.Background(), setupExercises(t), fakeSetupEnv("", "27.0", nil, true, true))
	if len(report.problems) != 0 || strings.Join(report.commands, ",") != "bash,curl,docker,nginx,python3" {
		t.Fatalf("healthy report = %+v", report)
	}
	lines := setupReportLines(report, func(_, fallback string) string { return fallback })
	if len(lines) != 1 || !lines[0].ok || !strings.Contains(lines[0].text, "bash, curl, docker, nginx, python3") {
		t.Fatalf("healthy lines = %+v", lines)
	}
}

func TestSetupMarkdownOffersTheFix(t *testing.T) {
	report := runSetupCheck(context.Background(), setupExercises(t), fakeSetupEnv("nginx", "", nil, false, false))
	md := setupMarkdown(setupReportLines(report, func(_, fallback string) string { return fallback }), func(_, fallback string) string { return fallback })
	for _, want := range []string{"✗ nginx is not installed.", "Needed by exercises 01.", "```sh\nsudo apt install -y nginx\n```", "Press f"} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown lacks %q:\n%s", want, md)
		}
	}
}

func TestUpdateSetupPanelRunsTheFirstFix(t *testing.T) {
	m, _ := workspaceModel(t)
	defer func(orig func() setupEnv) { newSetupEnv = orig }(newSetupEnv)
	newSetupEnv = func() setupEnv { return fakeSetupEnv("", "permission denied", errors.New("exit 1"), false, false) }
	var ran []string
	defer func(orig func([]string) tea.Cmd) { runFix = orig }(runFix)
	runFix = func(argv []string) tea.Cmd {
		ran = argv
		return func() tea.Msg { return setupFixDoneMsg{} }
	}

	// The startup check reports the problem in the status and the status row.
	m.status = m.initialStatusText()
	m = updateWith(m, m.setupCheckCmd()())
	if !strings.Contains(m.status, "Press d") || !strings.Contains(m.View(), "! Setup: 1") {
		t.Fatalf("status = %q; the status row should warn:\n%s", m.status, m.View())
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = updated.(model)
	if m.panel != panelSetup || cmd == nil || !strings.Contains(m.readme.View(), "docker group") {
		t.Fatalf("panel = %v, cmd = %v, content:\n%s", m.panel, cmd, m.readme.View())
	}
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	m = updated.(model)
	if shellCommand(ran) != "sudo usermod -aG docker learner" {
		t.Fatalf("ran %q", ran)
	}
	m = updateWith(m, cmd())
	if !strings.Contains(m.status, "Checking the setup again") {
		t.Fatalf("status after the fix = %q", m.status)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if updated.(model).panel != panelInstructions {
		t.Fatal("d did not return to the instructions")
	}
}

func TestUpdateKeysPanelToggles(t *testing.T) {
	m, _ := workspaceModel(t)
	m = pressKeys(m, "?")
	if m.panel != panelKeys || !strings.Contains(m.readme.View(), "quit") {
		t.Fatalf("panel = %v, content:\n%s", m.panel, m.readme.View())
	}
	m = pressKeys(m, "?")
	if m.panel != panelInstructions {
		t.Fatal("? did not return to the instructions")
	}
}

// updateWith applies msg and returns the model.
func updateWith(m model, msg tea.Msg) model {
	updated, _ := m.Update(msg)
	return updated.(model)
}

func TestRunCLIDoctor(t *testing.T) {
	root := newCLIAppRoot(t)
	createExercise(t, root, "01_first")
	defer func(orig func() setupEnv) { newSetupEnv = orig }(newSetupEnv)

	newSetupEnv = func() setupEnv { return fakeSetupEnv("", "27.0", nil, true, true) }
	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"--root", root, "--doctor"}, &stdout, &stderr, nil, nil); code != 0 {
		t.Fatalf("healthy --doctor exit = %d, stderr %s", code, stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "[OK]") {
		t.Fatalf("stdout = %q", stdout.String())
	}

	newSetupEnv = func() setupEnv { return fakeSetupEnv("bash", "27.0", nil, true, true) }
	stdout.Reset()
	if code := runCLI([]string{"--root", root, "--doctor"}, &stdout, &stderr, nil, nil); code != 1 {
		t.Fatalf("--doctor with bash missing exit = %d", code)
	}
	if !strings.Contains(stdout.String(), "[FAIL] bash is not installed.") || !strings.Contains(stdout.String(), "sudo apt install -y bash") {
		t.Fatalf("stdout = %q", stdout.String())
	}

	if code := runCLI([]string{"--root", root, "--doctor", "--list"}, &stdout, &stderr, nil, nil); code != 2 {
		t.Fatalf("--doctor --list exit = %d, want a usage error", code)
	}
}

func TestExercisesUsingDockerDNSOfTheShippedExercises(t *testing.T) {
	exercises, err := LoadExercises(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if got := exerciseRanges(exercisesUsingDockerDNS(exercises)); got != "10" {
		t.Fatalf("exercises using Docker's DNS = %q, want 10", got)
	}
}

func TestRunSetupCheckNotesPodmanWithoutCountingAProblem(t *testing.T) {
	exercises := setupExercises(t)
	writeFile(t, filepath.Join(exercises[0].Path, "nginx.conf"), "resolver 127.0.0.11;\n")
	env := fakeSetupEnv("", "", nil, true, true)
	env.dockerVersion = func(context.Context) (string, error) { return "podman version 5.8.2\n", nil }

	report := runSetupCheck(context.Background(), exercises, env)
	if len(report.problems) != 0 || len(report.notes) != 1 || exerciseRanges(report.notes[0].exercises) != exercises[0].ID[:2] {
		t.Fatalf("report = %+v, want one Podman note and no problems", report)
	}
	text := setupText(setupReportLines(report, func(_, fallback string) string { return fallback }))
	if !strings.HasPrefix(text, "[OK]") || !strings.Contains(text, "[NOTE] The docker command here is Podman's.") || !strings.Contains(text, "127.0.0.11") {
		t.Fatalf("text = %q", text)
	}

	// With Docker itself there is no note.
	env.dockerVersion = func(context.Context) (string, error) { return "Docker version 29.0.0", nil }
	if report := runSetupCheck(context.Background(), exercises, env); len(report.notes) != 0 {
		t.Fatalf("notes with Docker = %+v", report.notes)
	}
}

func TestRunSetupCheckNotesAMissingTmuxWithItsFix(t *testing.T) {
	report := runSetupCheck(context.Background(), setupExercises(t), fakeSetupEnv("tmux", "27.0", nil, true, true))
	if len(report.problems) != 0 || len(report.notes) != 1 || report.notes[0].kind != noteNoTmux {
		t.Fatalf("report = %+v, want one tmux note and no problems", report)
	}
	text := setupText(setupReportLines(report, func(_, fallback string) string { return fallback }))
	if !strings.Contains(text, "[NOTE] tmux is not installed") || !strings.Contains(text, "sudo apt install -y tmux") {
		t.Fatalf("text = %q", text)
	}
	if fix, err := report.firstFix(); err != nil || shellCommand(fix) != "sudo apt install -y tmux" {
		t.Fatalf("firstFix() = %q, %v", fix, err)
	}
}

func TestStatusRowWarnsOnlyAboutProblems(t *testing.T) {
	m, _ := workspaceModel(t)
	m.setup = &setupReport{notes: []setupProblem{{kind: notePodmanDNS, exercises: []string{"10"}}}}
	if strings.Contains(m.View(), "! Setup") {
		t.Fatal("a Podman note alone must not raise the setup warning")
	}
}
