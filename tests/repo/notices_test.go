package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"nginxlings/tests/testutil"
)

const (
	noticesFile = "THIRD_PARTY_NOTICES.md"
	// Set to any value to rewrite the notices file instead of comparing it.
	updateNoticesEnv = "NGINXLINGS_UPDATE_NOTICES"
)

var licenseFilePattern = regexp.MustCompile(`(?i)^(licen[sc]e|copying)(\.(md|txt))?$`)

// component is one piece of third-party code in a release binary.
type component struct {
	name    string
	version string
	license string
	text    string
}

// TestRepoThirdPartyNoticesMatchTheBinary keeps THIRD_PARTY_NOTICES.md equal
// to the licenses of Go and of every module compiled into a release binary.
// Those licenses ask for their text in every copy, and the releases ship this
// file for that. Unlike the rest of this package the test runs the go
// command, which knows what a build links.
func TestRepoThirdPartyNoticesMatchTheBinary(t *testing.T) {
	root := testutil.RepoRoot(t)
	// The go command reruns a cached test when a file it read has changed;
	// reading go.mod ties this test to the dependency versions.
	if _, err := os.ReadFile(filepath.Join(root, "go.mod")); err != nil {
		t.Fatal(err)
	}

	want := renderNotices(releaseComponents(t, root))
	path := filepath.Join(root, noticesFile)
	if os.Getenv(updateNoticesEnv) != "" {
		if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s does not match the modules in the binary; run `%s=1 go test ./tests/repo -run ThirdPartyNotices` and review the diff",
			noticesFile, updateNoticesEnv)
	}
}

// TestRepoReleasesShipThirdPartyNotices checks that the notices leave the
// repository with the binary: in the archives and in the Linux packages.
func TestRepoReleasesShipThirdPartyNotices(t *testing.T) {
	root := testutil.RepoRoot(t)
	shipped := false
	for _, glob := range releaseArchiveGlobs(t, filepath.Join(root, ".goreleaser.yml")) {
		shipped = shipped || glob == noticesFile
	}
	if !shipped {
		t.Errorf("the archives in .goreleaser.yml do not include %s", noticesFile)
	}

	config, err := os.ReadFile(filepath.Join(root, ".goreleaser.yml"))
	if err != nil {
		t.Fatal(err)
	}
	want := "      - src: " + noticesFile + "\n        dst: /usr/share/licenses/nginxlings/" + noticesFile + "\n"
	if !strings.Contains(string(config), want) {
		t.Errorf("the packages in .goreleaser.yml do not install %s", noticesFile)
	}
}

// releaseComponents returns Go itself and the modules a release build links,
// sorted by module path. It fails on a module whose license it cannot name.
func releaseComponents(t *testing.T, root string) []component {
	t.Helper()

	// The release targets in .goreleaser.yml.
	dirs := map[string]string{}
	versions := map[string]string{}
	for _, arch := range []string{"amd64", "arm64"} {
		out := goOutput(t, root, []string{"GOOS=linux", "GOARCH=" + arch, "CGO_ENABLED=0"},
			"list", "-deps", "-f", "{{with .Module}}{{if not .Main}}{{.Path}}\t{{.Version}}\t{{.Dir}}{{end}}{{end}}", "./cmd/nginxlings")
		for _, line := range strings.Split(out, "\n") {
			fields := strings.Split(line, "\t")
			if len(fields) != 3 {
				continue
			}
			versions[fields[0]], dirs[fields[0]] = fields[1], fields[2]
		}
	}
	if len(dirs) == 0 {
		t.Fatal("go list reported no modules for ./cmd/nginxlings")
	}
	paths := make([]string, 0, len(dirs))
	for path := range dirs {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	goRoot := strings.TrimSpace(goOutput(t, root, nil, "env", "GOROOT"))
	components := []component{readComponent(t, "Go standard library and runtime", "", goRoot)}
	for _, path := range paths {
		components = append(components, readComponent(t, path, versions[path], dirs[path]))
	}
	return components
}

func goOutput(t *testing.T, dir string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if exit, ok := err.(*exec.ExitError); ok {
			stderr = string(exit.Stderr)
		}
		t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, stderr)
	}
	return string(out)
}

// readComponent reads the one license file at the top of dir.
func readComponent(t *testing.T, name, version, dir string) component {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && licenseFilePattern.MatchString(entry.Name()) {
			files = append(files, entry.Name())
		}
	}
	if len(files) != 1 {
		t.Fatalf("%s: want one license file in %s, found %v", name, dir, files)
	}
	data, err := os.ReadFile(filepath.Join(dir, files[0]))
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}

	text := normalizeLicenseText(string(data))
	license := licenseName(text)
	if license == "" {
		t.Fatalf("%s: %s is not a license this test knows (MIT, BSD). Check that it is compatible with AGPL-3.0-only and what it asks of a binary release, then teach licenseName about it.",
			name, files[0])
	}
	return component{name: name, version: version, license: license, text: text}
}

// normalizeLicenseText drops carriage returns, trailing spaces and the blank
// lines around the text, so the notices file stays free of whitespace noise.
func normalizeLicenseText(text string) string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t\r")
	}
	return strings.Trim(strings.Join(lines, "\n"), "\n")
}

// licenseName recognizes the permissive licenses the dependencies use by
// their operative sentences. It returns "" for anything else.
func licenseName(text string) string {
	flat := strings.Join(strings.Fields(text), " ")
	switch {
	case strings.Contains(flat, "Permission is hereby granted, free of charge, to any person obtaining a copy"):
		return "MIT"
	case strings.Contains(flat, "Redistribution and use in source and binary forms"):
		if strings.Contains(flat, "endorse or promote products") {
			return "BSD-3-Clause"
		}
		return "BSD-2-Clause"
	}
	return ""
}

func renderNotices(components []component) string {
	var b strings.Builder
	b.WriteString(`# Third-Party Notices

Nginxlings is licensed under AGPL-3.0-only; see ` + "`LICENSE`" + `. Its release binary
also contains the Go standard library and runtime and the Go modules listed
below. Each of them has its own license, which asks that its text accompany
every copy, so the texts are reproduced here.

This file is generated; "Third-Party Notices" in ` + "`CONTRIBUTING.md`" + ` says how
to update it.

| Component | Version | License |
| --- | --- | --- |
`)
	for _, c := range components {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", c.name, c.version, c.license)
	}
	for _, c := range components {
		fence := "```"
		for strings.Contains(c.text, fence) {
			fence += "`"
		}
		title := c.name
		if c.version != "" {
			title += " " + c.version
		}
		fmt.Fprintf(&b, "\n## %s\n\nLicense: %s\n\n%stext\n%s\n%s\n", title, c.license, fence, c.text, fence)
	}
	return b.String()
}
