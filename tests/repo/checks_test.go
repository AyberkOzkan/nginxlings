// Package repo checks repository-wide contracts: manifest, translations,
// READMEs, check scripts, release and CI configuration. It reads files only,
// except for the third-party notices test, which asks the go command which
// modules a build links.
package repo

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"nginxlings/tests/testutil"
)

func TestRepoCheckScriptsAreExecutable(t *testing.T) {
	root := testutil.RepoRoot(t)
	exRoot := filepath.Join(root, "exercises")

	for _, id := range testutil.ExerciseIDs(t, exRoot) {
		path := filepath.Join(exRoot, id, testutil.CheckScriptName)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if info.IsDir() {
			t.Fatalf("%s is a directory", path)
		}
		if info.Mode()&0o111 == 0 {
			t.Fatalf("%s is not executable", path)
		}
	}
}

func TestRepoCheckScriptsUseSharedLibrary(t *testing.T) {
	root := testutil.RepoRoot(t)
	exRoot := filepath.Join(root, "exercises")

	// The library owns traps, temporary files, and error handling; checks wait
	// for readiness by polling instead of sleeping for whole seconds.
	forbidden := map[string]string{
		"DOCKER_BUILDKIT=0":        "use docker_build from the library",
		"mktemp":                   "use temp_file from the library",
		"trap ":                    "use on_exit from the library",
		"fail() {":                 "use fail from the library",
		`grep -v '^[[:space:]]*#'`: "use active_config from the library, which also splits one-line blocks",
	}
	wholeSecondSleep := regexp.MustCompile(`(?m)^[^#\n]*\bsleep[ \t]+[0-9]+([ \t;]|$)`)

	for _, id := range testutil.ExerciseIDs(t, exRoot) {
		checkPath := filepath.Join(exRoot, id, testutil.CheckScriptName)
		data, err := os.ReadFile(checkPath)
		if err != nil {
			t.Fatalf("%s: %v", checkPath, err)
		}
		script := string(data)

		testutil.AssertContains(t, checkPath, script, `. "${EX_DIR}/../lib/check-lib.sh"`)
		for text, reason := range forbidden {
			if strings.Contains(script, text) {
				t.Errorf("%s contains %q; %s", checkPath, text, reason)
			}
		}
		if match := wholeSecondSleep.FindString(script); match != "" {
			t.Errorf("%s sleeps for a fixed time (%q); poll for readiness instead", checkPath, strings.TrimSpace(match))
		}
	}
}

func TestRepoCheckRuntimeImagesArePinnedByDigest(t *testing.T) {
	root := testutil.RepoRoot(t)
	exRoot := filepath.Join(root, "exercises")

	// Images built from the exercise are named nginxlings/...; any other image a
	// check runs directly must carry a digest.
	imageDefault := regexp.MustCompile(`(?m)^([A-Z0-9_]*IMAGE[A-Z0-9_]*)="\$\{[A-Z0-9_]+:-([^}]+)\}"`)
	pinned := regexp.MustCompile(`^[a-z0-9./_-]+:[A-Za-z0-9._-]+@sha256:[0-9a-f]{64}$`)

	for _, id := range testutil.ExerciseIDs(t, exRoot) {
		checkPath := filepath.Join(exRoot, id, testutil.CheckScriptName)
		data, err := os.ReadFile(checkPath)
		if err != nil {
			t.Fatalf("%s: %v", checkPath, err)
		}
		for _, match := range imageDefault.FindAllStringSubmatch(string(data), -1) {
			name, image := match[1], match[2]
			if strings.HasPrefix(image, "nginxlings/") {
				continue
			}
			if !pinned.MatchString(image) {
				t.Errorf("%s: %s defaults to %q; pin it as <name>:<tag>@sha256:<digest>", checkPath, name, image)
			}
		}
	}
}

func TestRepoDockerExercisesUseScopedResources(t *testing.T) {
	root := testutil.RepoRoot(t)
	exRoot := filepath.Join(root, "exercises")

	libPath := filepath.Join(exRoot, testutil.LibDir, "check-lib.sh")
	lib, err := os.ReadFile(libPath)
	if err != nil {
		t.Fatal(err)
	}
	testutil.AssertContains(t, libPath, string(lib), `"${DOCKER_BIN}" build --label "${DOCKER_LABEL}"`)

	for _, id := range testutil.ExerciseIDs(t, exRoot) {
		dockerfilePath := filepath.Join(exRoot, id, "Dockerfile")
		if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
			continue
		} else if err != nil {
			t.Fatalf("%s: %v", dockerfilePath, err)
		}

		checkPath := filepath.Join(exRoot, id, testutil.CheckScriptName)
		data, err := os.ReadFile(checkPath)
		if err != nil {
			t.Fatalf("%s: %v", checkPath, err)
		}
		script := string(data)

		testutil.AssertContains(t, checkPath, script, `EXERCISE_ID="`+id+`"`)
		testutil.AssertContains(t, checkPath, script, `DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"`)
		testutil.AssertContains(t, checkPath, script, `nginxlings/${EXERCISE_ID}:check`)

		if strings.Contains(script, "builder prune") {
			t.Fatalf("%s must not prune global Docker builder cache", checkPath)
		}

		resourceCommands := strings.Count(script, `"${DOCKER_BIN}" build `) +
			strings.Count(script, `"${DOCKER_BIN}" run `) +
			strings.Count(script, `"${DOCKER_BIN}" network create `)
		labels := strings.Count(script, `--label "${DOCKER_LABEL}"`)
		if labels != resourceCommands {
			t.Fatalf("%s has %d Docker resource commands but %d labels", checkPath, resourceCommands, labels)
		}
	}
}

func TestRepoDockerfilesUsePinnedBaseImages(t *testing.T) {
	root := testutil.RepoRoot(t)
	exRoot := filepath.Join(root, "exercises")

	expected := map[string]string{
		"05_install_verify":      "FROM debian:12.15-slim",
		"18_osi_layers_usecases": "FROM alpine:3.20.10",
	}
	for _, id := range testutil.ExerciseIDs(t, exRoot) {
		dockerfilePath := filepath.Join(exRoot, id, "Dockerfile")
		data, err := os.ReadFile(dockerfilePath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatalf("%s: %v", dockerfilePath, err)
		}

		want := expected[id]
		if want == "" {
			want = "FROM nginx:1.28.3-alpine"
		}
		testutil.AssertContains(t, dockerfilePath, string(data), want)
	}
}

func TestRepoExercise04CheckMatchesDockerfilePermissions(t *testing.T) {
	root := testutil.RepoRoot(t)
	exerciseRoot := filepath.Join(root, "exercises", "04_worker_user")

	dockerfilePath := filepath.Join(exerciseRoot, "Dockerfile")
	dockerfile, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatalf("%s: %v", dockerfilePath, err)
	}
	for _, required := range []string{
		"chown -R webapp:webapp /srv/site",
		"chmod 0750 /srv/site",
		"chmod 0640 /srv/site/index.html",
	} {
		testutil.AssertContains(t, dockerfilePath, string(dockerfile), required)
	}

	checkPath := filepath.Join(exerciseRoot, testutil.CheckScriptName)
	check, err := os.ReadFile(checkPath)
	if err != nil {
		t.Fatalf("%s: %v", checkPath, err)
	}
	testutil.AssertContains(t, checkPath, string(check),
		`EXPECTED_PERMS=$'/srv/site webapp:webapp 750\n/srv/site/index.html webapp:webapp 640'`)
}
