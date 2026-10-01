package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// setupCheckTimeout bounds `docker info`, which hangs when the daemon does.
const setupCheckTimeout = 10 * time.Second

type problemKind int

const (
	problemMissingCommand problemKind = iota
	problemDockerGroup
	problemDockerRelogin
	problemDockerStopped
	problemDockerOther
	// notePodmanDNS is not a problem to fix: Podman's docker command runs
	// most exercises, but not those that use Docker's embedded DNS server.
	notePodmanDNS
	// noteNoTmux: without tmux nginxlings runs without its terminal pane.
	noteNoTmux
)

// dockerDNSAddress is Docker's embedded DNS server, which Podman lacks.
const dockerDNSAddress = "127.0.0.11"

// setupProblem is one thing that keeps some exercises from running.
type setupProblem struct {
	kind problemKind
	// command is the missing or failing command, for example "nginx".
	command string
	// exercises are the numbers of the exercises that need it, like "04".
	exercises []string
	// detail is Docker's own message for problemDockerOther.
	detail string
	// fix is the command that solves the problem, or nil when nginxlings
	// cannot tell which one does.
	fix []string
}

// setupReport lists the commands the exercises need, what is wrong, and
// notes about exercises that cannot run here although nothing is missing.
type setupReport struct {
	commands []string
	problems []setupProblem
	notes    []setupProblem
}

// setupEnv is what the setup check reads from the system; tests replace it.
type setupEnv struct {
	lookPath      func(string) (string, error)
	dockerInfo    func(context.Context) (string, error)
	dockerVersion func(context.Context) (string, error)
	// dockerGroup reports whether the user is in the docker group in the
	// group database (listed) and in this session (active).
	dockerGroup func() (listed, active bool)
	osRelease   map[string]string
	userName    string
}

func systemSetupEnv() setupEnv {
	name := os.Getenv("USER")
	if u, err := user.Current(); err == nil {
		name = u.Username
	}
	return setupEnv{
		lookPath:      lookPathWithSbin,
		dockerInfo:    runDockerInfo,
		dockerVersion: runDockerVersion,
		dockerGroup:   dockerGroupMembership,
		osRelease:     readOSRelease("/etc/os-release"),
		userName:      name,
	}
}

// requireCmdPattern finds the commands a check script asks for. Variables
// name the commands the shared library lets a learner override.
var requireCmdPattern = regexp.MustCompile(`(?m)^[ \t]*require_(cmd|docker)\b[ \t]*("?\$\{?([A-Z_]+)\}?"?|([a-z0-9_.-]+))?`)

var commandVariables = map[string]string{
	"NGINX_BIN":  "nginx",
	"PYTHON_BIN": "python3",
	"DOCKER_BIN": "docker",
	"CURL_BIN":   "curl",
}

// requiredCommands maps each command the exercise checks need to the
// exercises that need it. Every check runs in bash.
func requiredCommands(exercises []Exercise) map[string][]string {
	needs := make(map[string][]string)
	add := func(cmd, id string) {
		number := id[:2]
		for _, have := range needs[cmd] {
			if have == number {
				return
			}
		}
		needs[cmd] = append(needs[cmd], number)
	}
	for _, ex := range exercises {
		add("bash", ex.ID)
		data, err := os.ReadFile(filepath.Join(ex.Path, checkScriptName))
		if err != nil {
			continue
		}
		for _, match := range requireCmdPattern.FindAllStringSubmatch(string(data), -1) {
			switch {
			case match[1] == "docker":
				add("docker", ex.ID)
			case match[3] != "":
				if cmd, ok := commandVariables[match[3]]; ok {
					add(cmd, ex.ID)
				}
			case match[4] != "":
				add(match[4], ex.ID)
			}
		}
	}
	return needs
}

// runSetupCheck checks that every command the exercises need is installed
// and that Docker answers this user.
func runSetupCheck(ctx context.Context, exercises []Exercise, env setupEnv) setupReport {
	needs := requiredCommands(exercises)
	var report setupReport
	for cmd := range needs {
		report.commands = append(report.commands, cmd)
	}
	sort.Strings(report.commands)

	pm := detectPackageManager(env.osRelease)
	for _, cmd := range report.commands {
		if _, err := env.lookPath(cmd); err != nil {
			report.problems = append(report.problems, setupProblem{
				kind:      problemMissingCommand,
				command:   cmd,
				exercises: needs[cmd],
				fix:       pm.installCommand(cmd),
			})
			continue
		}
		if cmd == "docker" {
			if problem, ok := checkDocker(ctx, env); !ok {
				problem.exercises = needs[cmd]
				report.problems = append(report.problems, problem)
			}
			if isPodman(ctx, env) {
				if numbers := exercisesUsingDockerDNS(exercises); len(numbers) > 0 {
					report.notes = append(report.notes, setupProblem{kind: notePodmanDNS, command: "docker", exercises: numbers})
				}
			}
		}
	}
	if _, err := env.lookPath("tmux"); err != nil {
		report.notes = append(report.notes, setupProblem{kind: noteNoTmux, command: "tmux", fix: pm.installCommand("tmux")})
	}
	return report
}

// isPodman reports whether the docker command is Podman's, which RHEL-like
// systems install instead of Docker.
func isPodman(ctx context.Context, env setupEnv) bool {
	ctx, cancel := context.WithTimeout(ctx, setupCheckTimeout)
	defer cancel()
	out, _ := env.dockerVersion(ctx)
	return strings.Contains(strings.ToLower(out), "podman")
}

func runDockerVersion(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "docker", "--version").CombinedOutput()
	return string(out), err
}

// exercisesUsingDockerDNS returns the exercises whose files mention Docker's
// embedded DNS server, so an exercise added later is covered too.
func exercisesUsingDockerDNS(exercises []Exercise) []string {
	var numbers []string
	for _, ex := range exercises {
		_ = filepath.WalkDir(ex.Path, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			data, err := os.ReadFile(path)
			if err == nil && strings.Contains(string(data), dockerDNSAddress) {
				numbers = append(numbers, ex.ID[:2])
				return filepath.SkipAll
			}
			return nil
		})
	}
	return numbers
}

// checkDocker asks the Docker daemon for its version and explains a failure.
func checkDocker(ctx context.Context, env setupEnv) (setupProblem, bool) {
	ctx, cancel := context.WithTimeout(ctx, setupCheckTimeout)
	defer cancel()
	out, err := env.dockerInfo(ctx)
	if err == nil {
		return setupProblem{}, true
	}
	problem := setupProblem{command: "docker"}
	lower := strings.ToLower(out)
	switch {
	case strings.Contains(lower, "permission denied"):
		listed, active := env.dockerGroup()
		if listed && !active {
			problem.kind = problemDockerRelogin
		} else {
			problem.kind = problemDockerGroup
			problem.fix = []string{"sudo", "usermod", "-aG", "docker", env.userName}
		}
	// Docker 28 and older say "Cannot connect to the Docker daemon ... Is the
	// docker daemon running?"; Docker 29 says "failed to connect to the docker
	// API at ...; check if the path is correct and if the daemon is running".
	case strings.Contains(lower, "cannot connect to the docker daemon"),
		strings.Contains(lower, "failed to connect to the docker api"),
		strings.Contains(lower, "daemon is running"), strings.Contains(lower, "docker daemon running"):
		problem.kind = problemDockerStopped
		problem.fix = startDockerCommand(env.lookPath)
	default:
		problem.kind = problemDockerOther
		problem.detail = errorLine(out)
		if problem.detail == "" {
			problem.detail = err.Error()
		}
	}
	return problem, false
}

// runDockerInfo runs `docker info` like the checks do. Podman's docker
// command answers it too, but has no ServerVersion field, so no --format is
// used. Older Docker clients print the daemon error on stdout and only
// "errors pretty printing info" on stderr, so both are returned.
func runDockerInfo(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "docker", "info").CombinedOutput()
	if ctx.Err() != nil {
		return "docker info did not answer in time", ctx.Err()
	}
	return string(out), err
}

// dockerGroupMembership compares the group database with this session's
// groups: a user added to the docker group must log in again to use it.
func dockerGroupMembership() (listed, active bool) {
	group, err := user.LookupGroup("docker")
	if err != nil {
		return false, false
	}
	gid, err := strconv.Atoi(group.Gid)
	if err != nil {
		return false, false
	}
	if u, err := user.Current(); err == nil {
		if ids, err := u.GroupIds(); err == nil {
			for _, id := range ids {
				listed = listed || id == group.Gid
			}
		}
	}
	if groups, err := os.Getgroups(); err == nil {
		for _, g := range groups {
			active = active || g == gid
		}
	}
	return listed, active
}

func startDockerCommand(lookPath func(string) (string, error)) []string {
	if _, err := lookPath("systemctl"); err == nil {
		return []string{"sudo", "systemctl", "enable", "--now", "docker"}
	}
	if _, err := lookPath("rc-service"); err == nil {
		return []string{"sudo", "sh", "-c", "rc-update add docker default && rc-service docker start"}
	}
	return nil
}

// packageManager installs the packages that provide the commands the
// exercises need on one family of distributions.
type packageManager struct {
	install []string
	// packages maps a command to its package where the names differ.
	packages map[string]string
}

func (pm packageManager) installCommand(cmd string) []string {
	if pm.install == nil {
		return nil
	}
	pkg := cmd
	if name, ok := pm.packages[cmd]; ok {
		pkg = name
	}
	return append(append([]string(nil), pm.install...), pkg)
}

// detectPackageManager picks the package manager from /etc/os-release.
func detectPackageManager(osRelease map[string]string) packageManager {
	ids := strings.Fields(osRelease["ID"] + " " + osRelease["ID_LIKE"])
	has := func(names ...string) bool {
		for _, id := range ids {
			for _, name := range names {
				if id == name {
					return true
				}
			}
		}
		return false
	}
	switch {
	case has("debian", "ubuntu"):
		return packageManager{install: []string{"sudo", "apt", "install", "-y"}, packages: map[string]string{"docker": "docker.io"}}
	case osRelease["ID"] == "fedora":
		return packageManager{install: []string{"sudo", "dnf", "install", "-y"}, packages: map[string]string{"docker": "moby-engine"}}
	case has("rhel", "centos", "fedora"):
		return packageManager{install: []string{"sudo", "dnf", "install", "-y"}, packages: map[string]string{"docker": "podman-docker"}}
	case has("suse", "opensuse"):
		return packageManager{install: []string{"sudo", "zypper", "--non-interactive", "install"}}
	case has("arch"):
		return packageManager{install: []string{"sudo", "pacman", "-S", "--needed", "--noconfirm"}, packages: map[string]string{"python3": "python"}}
	case has("alpine"):
		return packageManager{install: []string{"sudo", "apk", "add"}}
	}
	return packageManager{}
}

// readOSRelease parses KEY=value lines; a missing file gives an empty map.
func readOSRelease(path string) map[string]string {
	values := make(map[string]string)
	file, err := os.Open(path)
	if err != nil {
		return values
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if !ok || strings.HasPrefix(key, "#") {
			continue
		}
		if unquoted, err := strconv.Unquote(value); err == nil {
			value = unquoted
		}
		values[strings.TrimSpace(key)] = strings.Trim(value, `'`)
	}
	return values
}

// errorLine returns the line of Docker's output that names the error, or
// else its first line, skipping the note Podman's docker command prints on
// every run.
func errorLine(s string) string {
	first := ""
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Emulate Docker CLI using podman") {
			continue
		}
		if first == "" {
			first = line
		}
		if strings.Contains(strings.ToLower(line), "error") {
			return line
		}
	}
	return first
}

// exerciseRanges writes exercise numbers compactly: 04, 05, 07, 08, 09
// becomes "04–05, 07–09".
func exerciseRanges(numbers []string) string {
	sorted := append([]string(nil), numbers...)
	sort.Strings(sorted)
	var parts []string
	for i := 0; i < len(sorted); {
		j := i
		for j+1 < len(sorted) && atoiOr(sorted[j+1], -1) == atoiOr(sorted[j], -2)+1 {
			j++
		}
		if j == i {
			parts = append(parts, sorted[i])
		} else {
			parts = append(parts, sorted[i]+"–"+sorted[j])
		}
		i = j + 1
	}
	return strings.Join(parts, ", ")
}

func atoiOr(s string, fallback int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}

// shellCommand writes a fix command the way a learner would type it.
func shellCommand(argv []string) string {
	quoted := make([]string, len(argv))
	for i, arg := range argv {
		if arg == "" || strings.ContainsAny(arg, " &|;'\"$") {
			arg = strconv.Quote(arg)
		}
		quoted[i] = arg
	}
	return strings.Join(quoted, " ")
}

// errNoFix means no problem in the report has a command that fixes it.
var errNoFix = errors.New("no fix to run")

// firstFix returns the first fix command, problems before notes.
func (r setupReport) firstFix() ([]string, error) {
	for _, problem := range append(append([]setupProblem(nil), r.problems...), r.notes...) {
		if problem.fix != nil {
			return problem.fix, nil
		}
	}
	return nil, errNoFix
}

// reportLine is one translated line of a setup report: what is wrong, and
// the command that fixes it when there is one. A note says what cannot run
// here although nothing is missing.
type reportLine struct {
	ok       bool
	note     bool
	text     string
	fixIntro string
	fix      string
}

// setupReportLines turns a report into translated lines. t looks up a UI
// string by key with an English fallback.
func setupReportLines(report setupReport, t func(key, fallback string) string) []reportLine {
	var lines []reportLine
	if len(report.problems) == 0 {
		lines = append(lines, reportLine{
			ok:   true,
			text: fmt.Sprintf(t("doctor_ok", "Everything the exercises need is ready: %s."), strings.Join(report.commands, ", ")),
		})
	}
	for _, p := range report.problems {
		line := reportLine{}
		if p.fix != nil {
			line.fix = shellCommand(p.fix)
		}
		switch p.kind {
		case problemMissingCommand:
			line.text = fmt.Sprintf(t("doctor_missing_cmd", "%s is not installed."), p.command)
			line.fixIntro = t("doctor_install_fix", "Install it with:")
			if p.fix == nil {
				line.fixIntro = t("doctor_install_manual", "Install it with your distribution's package manager.")
			}
		case problemDockerGroup:
			line.text = t("doctor_docker_group", "Your user cannot use Docker yet because it is not in the docker group.")
			line.fixIntro = t("doctor_docker_group_fix", "Add yourself to the group, then log out and back in:")
		case problemDockerRelogin:
			line.text = t("doctor_docker_relogin", "You are in the docker group, but this session started before you were added. Log out and back in, or restart the computer.")
		case problemDockerStopped:
			line.text = t("doctor_docker_stopped", "Docker is installed but not running.")
			line.fixIntro = t("doctor_docker_start_fix", "Start it, and keep it running after a restart:")
			if p.fix == nil {
				line.fixIntro = t("doctor_docker_start_manual", "Start the Docker service.")
			}
		case problemDockerOther:
			line.text = fmt.Sprintf(t("doctor_docker_other", "Docker does not answer: %s"), p.detail)
		}
		line.text += " " + fmt.Sprintf(t("doctor_needed_by", "Needed by exercises %s."), exerciseRanges(p.exercises))
		lines = append(lines, line)
	}
	for _, n := range report.notes {
		line := reportLine{note: true}
		switch n.kind {
		case notePodmanDNS:
			line.text = fmt.Sprintf(t("doctor_podman_dns", "The docker command here is Podman's. Exercises %s use Docker's DNS server on 127.0.0.11, which Podman does not have, so they cannot pass here. Install Docker CE from Docker's repository for them: https://docs.docker.com/engine/install/"), exerciseRanges(n.exercises))
		case noteNoTmux:
			line.text = t("doctor_no_tmux", "tmux is not installed, so nginxlings runs without the terminal pane below it.")
			if n.fix != nil {
				line.fixIntro = t("doctor_install_fix", "Install it with:")
				line.fix = shellCommand(n.fix)
			}
		}
		lines = append(lines, line)
	}
	return lines
}

// setupMarkdown renders report lines for the setup panel. Each fix gets a
// code block of its own, so the command is never broken across lines.
func setupMarkdown(lines []reportLine, t func(key, fallback string) string) string {
	var b strings.Builder
	hasFix := false
	for _, line := range lines {
		mark := "✗"
		switch {
		case line.ok:
			mark = "✓"
		case line.note:
			mark = "!"
		}
		fmt.Fprintf(&b, "%s %s\n\n", mark, line.text)
		if line.fixIntro != "" {
			b.WriteString(line.fixIntro + "\n\n")
		}
		if line.fix != "" {
			fmt.Fprintf(&b, "```sh\n%s\n```\n\n", line.fix)
			hasFix = true
		}
	}
	if hasFix {
		b.WriteString(t("doctor_fix_hint", "Press f to run the first command above. It asks for your password in this terminal.") + "\n")
	}
	return b.String()
}

// setupText renders report lines for the terminal.
func setupText(lines []reportLine) string {
	var b strings.Builder
	for _, line := range lines {
		mark := "[FAIL]"
		switch {
		case line.ok:
			mark = "[OK]"
		case line.note:
			mark = "[NOTE]"
		}
		fmt.Fprintf(&b, "%s %s\n", mark, line.text)
		if line.fixIntro != "" {
			fmt.Fprintf(&b, "       %s\n", line.fixIntro)
		}
		if line.fix != "" {
			fmt.Fprintf(&b, "       %s\n", line.fix)
		}
	}
	return b.String()
}
