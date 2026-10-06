package repo

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"nginxlings/tests/testutil"
)

func TestRepoLegacyWrapperIsAbsent(t *testing.T) {
	root := testutil.RepoRoot(t)
	path := filepath.Join(root, "bin", "nginxlings.sh")
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("legacy wrapper still exists: %s", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("%s: %v", path, err)
	}
}

func TestRepoReleaseTargetsLinuxWithoutNPMOrHomebrew(t *testing.T) {
	root := testutil.RepoRoot(t)

	configPath := filepath.Join(root, ".goreleaser.yml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("%s: %v", configPath, err)
	}
	config := string(configData)
	if !strings.Contains(config, "      - linux\n") {
		t.Fatal("GoReleaser config does not include Linux")
	}
	for _, unsupported := range []string{"      - darwin\n", "      - windows\n"} {
		if strings.Contains(config, unsupported) {
			t.Fatalf("GoReleaser config includes unsupported target %q", strings.TrimSpace(unsupported))
		}
	}

	workflowPath := filepath.Join(root, ".github", "workflows", "release.yml")
	workflowData, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("%s: %v", workflowPath, err)
	}
	workflow := strings.ToLower(string(workflowData))
	for _, forbidden := range []string{"npm", "homebrew", "setup-node"} {
		if strings.Contains(workflow, forbidden) {
			t.Fatalf("release workflow unexpectedly references %q", forbidden)
		}
	}

	// packaging/ holds the Linux package files; npm and Homebrew stay out.
	entries, err := os.ReadDir(filepath.Join(root, "packaging"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("inspect packaging directory: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == "package.json" || name == "npm" || name == "homebrew" || strings.HasSuffix(name, ".rb") {
			t.Fatalf("packaging/%s is npm or Homebrew packaging", name)
		}
	}
}

// TestRepoLinuxPackagesShipTheAppAndItsDependencies checks the nfpm section:
// the package formats, the files an installed nginxlings needs, and that every
// format depends on the commands the exercises run.
func TestRepoLinuxPackagesShipTheAppAndItsDependencies(t *testing.T) {
	root := testutil.RepoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, ".goreleaser.yml"))
	if err != nil {
		t.Fatal(err)
	}
	config := string(data)
	nfpms, _, found := strings.Cut(config[strings.Index(config, "nfpms:"):], "\nchecksum:")
	if !found || !strings.HasPrefix(nfpms, "nfpms:") {
		t.Fatal(".goreleaser.yml has no nfpms section before checksum")
	}
	for _, want := range []string{
		"      - deb\n", "      - rpm\n", "      - archlinux\n",
		"    bindir: /usr/bin\n",
		"      - src: exercises/\n        dst: /usr/share/nginxlings/exercises\n        type: tree\n",
		"      - src: i18n/\n        dst: /usr/share/nginxlings/i18n\n        type: tree\n",
		"dst: /usr/share/applications/nginxlings.desktop\n",
		"dst: /usr/share/icons/hicolor/scalable/apps/nginxlings.svg\n",
		"      postinstall: packaging/postinstall.sh\n",
	} {
		if !strings.Contains(nfpms, want) {
			t.Errorf("nfpms section lacks %q", strings.TrimSpace(want))
		}
	}
	// Each format names NGINX, curl, bash, Python and a Docker provider.
	for format, docker := range map[string]string{"deb": "docker.io", "rpm": "/usr/bin/dockerd", "archlinux": "docker"} {
		block, _, _ := strings.Cut(nfpms[strings.Index(nfpms, "      "+format+":\n"):], "\n      recommends:")
		for _, dep := range []string{"bash", "curl", "nginx", "python", "tmux", docker} {
			if !strings.Contains(block, dep) {
				t.Errorf("%s dependencies lack %s", format, dep)
			}
		}
	}
	for _, src := range []string{"packaging/nginxlings.desktop", "packaging/nginxlings.svg", "packaging/postinstall.sh", "install.sh"} {
		if _, err := os.Stat(filepath.Join(root, src)); err != nil {
			t.Errorf("%s: %v", src, err)
		}
	}
}

// TestRepoDesktopEntryOpensTheTUIInATerminal keeps the menu entry usable by
// someone who never typed a command: it runs nginxlings in a terminal.
func TestRepoDesktopEntryOpensTheTUIInATerminal(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(testutil.RepoRoot(t), "packaging", "nginxlings.desktop"))
	if err != nil {
		t.Fatal(err)
	}
	entry := string(data)
	if !strings.HasPrefix(entry, "[Desktop Entry]\n") {
		t.Fatal("the desktop file does not start with [Desktop Entry]")
	}
	for _, line := range []string{"Type=Application", "Name=Nginxlings", "Exec=nginxlings", "Icon=nginxlings", "Terminal=true", "Comment[tr]="} {
		if !strings.Contains(entry, "\n"+line) {
			t.Errorf("the desktop file lacks %q", line)
		}
	}
}

// TestRepoReleaseArchiveIncludesEveryAsset checks that the GoReleaser archive
// globs cover every exercise and translation file, so a new asset type cannot
// be left out of a release silently, and that no glob has gone stale.
func TestRepoReleaseArchiveIncludesEveryAsset(t *testing.T) {
	root := testutil.RepoRoot(t)
	globs := releaseArchiveGlobs(t, filepath.Join(root, ".goreleaser.yml"))
	patterns := make([]*regexp.Regexp, len(globs))
	for i, glob := range globs {
		patterns[i] = releaseGlobRegexp(glob)
	}

	// install.sh installs from the archive and needs the menu entry and icon.
	assets := []string{"LICENSE", "README.md", "install.sh", "packaging/nginxlings.desktop", "packaging/nginxlings.svg"}
	for _, dir := range []string{"exercises", "i18n"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// Skip the runtime artifacts listed in .gitignore.
			if entry.IsDir() {
				if entry.Name() == "logs" {
					return filepath.SkipDir
				}
				return nil
			}
			name := entry.Name()
			if strings.HasPrefix(name, ".check_body") || name == "response.html" ||
				name == "nginx_test.log" || strings.HasSuffix(name, ".pid") {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			assets = append(assets, filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}

	used := make([]bool, len(globs))
	for _, asset := range assets {
		included := false
		for i, pattern := range patterns {
			if pattern.MatchString(asset) {
				used[i] = true
				included = true
			}
		}
		if !included {
			t.Errorf("%s is not included by any archive glob in .goreleaser.yml", asset)
		}
	}
	for i, glob := range globs {
		if !used[i] {
			t.Errorf("archive glob %q in .goreleaser.yml matches no file", glob)
		}
	}
}

// releaseArchiveGlobs returns the "files:" entries of the archives section.
func releaseArchiveGlobs(t *testing.T, path string) []string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	var globs []string
	inArchives, inFiles := false, false
	for _, line := range strings.Split(string(data), "\n") {
		switch {
		case line == "archives:":
			inArchives = true
		case inArchives && line != "" && !strings.HasPrefix(line, " "):
			inArchives, inFiles = false, false
		case inArchives && line == "    files:":
			inFiles = true
		case inFiles && strings.HasPrefix(line, "      - "):
			globs = append(globs, strings.TrimSpace(strings.TrimPrefix(line, "      - ")))
		case inFiles:
			inFiles = false
		}
	}
	if len(globs) == 0 {
		t.Fatalf("%s: no archive files found", path)
	}
	return globs
}

// releaseGlobRegexp translates a GoReleaser file glob into a regexp over
// slash-separated repository paths; "**/" also matches no directory.
func releaseGlobRegexp(glob string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(glob); i++ {
		switch {
		case strings.HasPrefix(glob[i:], "**/"):
			b.WriteString("(?:.*/)?")
			i += 2
		case strings.HasPrefix(glob[i:], "**"):
			b.WriteString(".*")
			i++
		case glob[i] == '*':
			b.WriteString("[^/]*")
		case glob[i] == '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(glob[i : i+1]))
		}
	}
	b.WriteString("$")
	return regexp.MustCompile(b.String())
}

// TestRepoDocsMatchSupportedToolchainAndTargets keeps contributor and release
// docs aligned with go.mod and the Linux-only GitHub Release pipeline.
func TestRepoDocsMatchSupportedToolchainAndTargets(t *testing.T) {
	root := testutil.RepoRoot(t)

	goMod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	goVersion := regexp.MustCompile(`(?m)^go ([0-9]+\.[0-9]+)`).FindSubmatch(goMod)
	if goVersion == nil {
		t.Fatal("go.mod has no go directive")
	}
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "Go " + string(goVersion[1]) + " or newer"; !strings.Contains(string(readme), want) {
		t.Fatalf("README.md does not state %q from go.mod", want)
	}

	for _, doc := range []string{"CONTRIBUTING.md", filepath.Join("docs", "RELEASING.md")} {
		data, err := os.ReadFile(filepath.Join(root, doc))
		if err != nil {
			t.Fatal(err)
		}
		for _, stale := range []string{"node --test", "packaging/npm", "packaging/homebrew", "dist/npm", "dist/homebrew", "four archives"} {
			if strings.Contains(string(data), stale) {
				t.Fatalf("%s still references removed release tooling %q", doc, stale)
			}
		}
	}
}

// TestRepoWorkflowsPinActionsAndLimitPermissions requires every third-party
// action to be pinned to a reviewed commit SHA with a version comment, and
// the CI workflow to run with read-only repository permissions.
func TestRepoWorkflowsPinActionsAndLimitPermissions(t *testing.T) {
	root := testutil.RepoRoot(t)
	pinned := regexp.MustCompile(`^\s*(?:-\s*)?uses:\s*[\w.-]+/[\w./-]+@[0-9a-f]{40} # v[0-9]+\.[0-9]+\.[0-9]+\s*$`)

	for _, name := range []string{"ci.yml", "release.yml"} {
		path := filepath.Join(root, ".github", "workflows", name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, "uses:") && !pinned.MatchString(line) {
				t.Errorf("%s:%d: action is not pinned to a commit SHA with a version comment: %s", path, i+1, strings.TrimSpace(line))
			}
		}
	}

	ci, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ci), "\npermissions:\n  contents: read\n") {
		t.Fatal("ci.yml must declare top-level permissions: contents: read")
	}
}

func TestRepoCIRunsPinnedGovulncheck(t *testing.T) {
	root := testutil.RepoRoot(t)
	path := filepath.Join(root, ".github", "workflows", "ci.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}

	workflow := string(data)
	testutil.AssertContains(t, path, workflow, "go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...")
	if strings.Contains(workflow, "govulncheck@latest") {
		t.Fatal("CI must pin govulncheck instead of using @latest")
	}
}
