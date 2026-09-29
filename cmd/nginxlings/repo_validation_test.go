package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestRepoManifestMatchesExerciseDirectories(t *testing.T) {
	root := repoRoot(t)
	exRoot := filepath.Join(root, "exercises")

	manifestIDs := readRepoManifestIDs(t, exRoot)
	dirIDs := readRepoExerciseDirIDs(t, exRoot)

	if strings.Join(manifestIDs, ",") != strings.Join(dirIDs, ",") {
		t.Fatalf("manifest IDs = %v, want exercise directories %v", manifestIDs, dirIDs)
	}
}

func TestRepoLegacyWrapperIsAbsent(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "bin", "nginxlings.sh")
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("legacy wrapper still exists: %s", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("%s: %v", path, err)
	}
}

func TestRepoReleaseTargetsLinuxWithoutNPMOrHomebrew(t *testing.T) {
	root := repoRoot(t)

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

	if _, err := os.Stat(filepath.Join(root, "packaging")); err == nil {
		t.Fatal("legacy npm/Homebrew packaging directory still exists")
	} else if !os.IsNotExist(err) {
		t.Fatalf("inspect packaging directory: %v", err)
	}
}

// TestRepoDocsMatchSupportedToolchainAndTargets keeps contributor and release
// docs aligned with go.mod and the Linux-only GitHub Release pipeline.
func TestRepoDocsMatchSupportedToolchainAndTargets(t *testing.T) {
	root := repoRoot(t)

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
		for _, stale := range []string{"node --test", "packaging/", "dist/npm", "dist/homebrew", "four archives"} {
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
	root := repoRoot(t)
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

// TestRepoManifestFallbackNamesAreEnglish keeps manifest default names equal
// to the English titles, because they are shown when translations fail to
// load and English is the default UI language.
func TestRepoManifestFallbackNamesAreEnglish(t *testing.T) {
	root := repoRoot(t)
	exercises, err := LoadExercises(root)
	if err != nil {
		t.Fatal(err)
	}
	english := readRepoTranslationMap(t, filepath.Join(root, "i18n", "exercises", "en.json"))
	for _, ex := range exercises {
		if ex.DefaultName != english[ex.ID] {
			t.Errorf("%s default_name = %q, want English title %q", ex.ID, ex.DefaultName, english[ex.ID])
		}
	}
}

func TestRepoCIRunsPinnedGovulncheck(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, ".github", "workflows", "ci.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}

	workflow := string(data)
	assertContains(t, path, workflow, "go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...")
	if strings.Contains(workflow, "govulncheck@latest") {
		t.Fatal("CI must pin govulncheck instead of using @latest")
	}
}

func TestRepoExerciseReadmeLanguageParity(t *testing.T) {
	root := repoRoot(t)
	exRoot := filepath.Join(root, "exercises")

	for _, id := range readRepoExerciseDirIDs(t, exRoot) {
		for _, name := range []string{"README.en.md", "README.tr.md"} {
			path := filepath.Join(exRoot, id, name)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%s: %v", path, err)
			}
			if strings.TrimSpace(string(data)) == "" {
				t.Fatalf("%s is empty", path)
			}
		}
	}
}

func TestRepoExerciseReadmeHeadingParity(t *testing.T) {
	root := repoRoot(t)
	exRoot := filepath.Join(root, "exercises")

	for _, id := range readRepoExerciseDirIDs(t, exRoot) {
		enPath := filepath.Join(exRoot, id, "README.en.md")
		trPath := filepath.Join(exRoot, id, "README.tr.md")
		enData, err := os.ReadFile(enPath)
		if err != nil {
			t.Fatalf("%s: %v", enPath, err)
		}
		trData, err := os.ReadFile(trPath)
		if err != nil {
			t.Fatalf("%s: %v", trPath, err)
		}

		enHeadings := markdownHeadingLevels(string(enData))
		trHeadings := markdownHeadingLevels(string(trData))
		if strings.Join(enHeadings, ",") != strings.Join(trHeadings, ",") {
			t.Fatalf("%s heading levels = %v, want README.tr.md levels %v", id, enHeadings, trHeadings)
		}
	}
}

func TestRepoExercise10DocsMatchDockerTopology(t *testing.T) {
	root := repoRoot(t)
	exerciseRoot := filepath.Join(root, "exercises", "10_upstream_status_zone")

	checkPath := filepath.Join(exerciseRoot, checkScriptName)
	checkData, err := os.ReadFile(checkPath)
	if err != nil {
		t.Fatalf("%s: %v", checkPath, err)
	}
	checkScript := string(checkData)
	for _, required := range []string{
		`B1_NAME="ex10_backend1"`,
		`B2_NAME="ex10_backend2"`,
		`--network "${NET_NAME}"`,
	} {
		assertContains(t, checkPath, checkScript, required)
	}

	configPath := filepath.Join(exerciseRoot, "nginx.conf")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("%s: %v", configPath, err)
	}
	assertContains(t, configPath, string(configData), "resolver 127.0.0.11")
	assertContains(t, configPath, string(configData), "TODO: replace this placeholder with both Docker backend servers")

	for _, name := range []string{"README.en.md", "README.tr.md"} {
		path := filepath.Join(exerciseRoot, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		content := string(data)
		for _, required := range []string{
			"ex10_backend1:8081",
			"ex10_backend2:8082",
			"127.0.0.11",
			"resolve",
		} {
			assertContains(t, path, content, required)
		}
		for _, loopbackBackend := range []string{"127.0.0.1:8081", "127.0.0.1:8082"} {
			if strings.Contains(content, loopbackBackend) {
				t.Fatalf("%s documents unreachable backend %q", path, loopbackBackend)
			}
		}
	}
}

func TestRepoTranslationFilesCoverManifest(t *testing.T) {
	root := repoRoot(t)
	manifestIDs := readRepoManifestIDs(t, filepath.Join(root, "exercises"))

	langsData, err := os.ReadFile(filepath.Join(root, "i18n", "langs.json"))
	if err != nil {
		t.Fatal(err)
	}
	var langs []LangMeta
	if err := json.Unmarshal(langsData, &langs); err != nil {
		t.Fatal(err)
	}
	if len(langs) == 0 {
		t.Fatal("i18n/langs.json has no languages")
	}

	var expectedUIKeys []string
	for index, lang := range langs {
		if strings.TrimSpace(lang.Code) == "" || strings.TrimSpace(lang.Label) == "" {
			t.Fatalf("invalid language metadata: %+v", lang)
		}

		exerciseNames := readRepoTranslationMap(t, filepath.Join(root, "i18n", "exercises", lang.Code+".json"))
		exerciseIDs := nonEmptyTranslationKeys(t, lang.Code+" exercise", exerciseNames)
		if strings.Join(exerciseIDs, ",") != strings.Join(manifestIDs, ",") {
			t.Fatalf("%s exercise translation IDs = %v, want manifest IDs %v", lang.Code, exerciseIDs, manifestIDs)
		}

		hints := readRepoTranslationMap(t, filepath.Join(root, "i18n", "hints", lang.Code+".json"))
		hintIDs := nonEmptyTranslationKeys(t, lang.Code+" hint", hints)
		if strings.Join(hintIDs, ",") != strings.Join(manifestIDs, ",") {
			t.Fatalf("%s hint IDs = %v, want manifest IDs %v", lang.Code, hintIDs, manifestIDs)
		}

		uiStrings := readRepoTranslationMap(t, filepath.Join(root, "i18n", "ui", lang.Code+".json"))
		uiKeys := nonEmptyTranslationKeys(t, lang.Code+" UI", uiStrings)
		if index == 0 {
			expectedUIKeys = uiKeys
			continue
		}
		if strings.Join(uiKeys, ",") != strings.Join(expectedUIKeys, ",") {
			t.Fatalf("%s UI translation keys = %v, want %v", lang.Code, uiKeys, expectedUIKeys)
		}
	}
}

func TestRepoCheckScriptsAreExecutable(t *testing.T) {
	root := repoRoot(t)
	exRoot := filepath.Join(root, "exercises")

	for _, id := range readRepoExerciseDirIDs(t, exRoot) {
		path := filepath.Join(exRoot, id, checkScriptName)
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
	root := repoRoot(t)
	exRoot := filepath.Join(root, "exercises")

	// The library owns traps, temporary files, and error handling; checks wait
	// for readiness by polling instead of sleeping for whole seconds.
	forbidden := map[string]string{
		"DOCKER_BUILDKIT=0": "use docker_build from the library",
		"mktemp":            "use temp_file from the library",
		"trap ":             "use on_exit from the library",
		"fail() {":          "use fail from the library",
	}
	wholeSecondSleep := regexp.MustCompile(`(?m)^[^#\n]*\bsleep[ \t]+[0-9]+([ \t;]|$)`)

	for _, id := range readRepoExerciseDirIDs(t, exRoot) {
		checkPath := filepath.Join(exRoot, id, checkScriptName)
		data, err := os.ReadFile(checkPath)
		if err != nil {
			t.Fatalf("%s: %v", checkPath, err)
		}
		script := string(data)

		assertContains(t, checkPath, script, `. "${EX_DIR}/../lib/check-lib.sh"`)
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
	root := repoRoot(t)
	exRoot := filepath.Join(root, "exercises")

	// Images built from the exercise are named nginxlings/...; any other image a
	// check runs directly must carry a digest.
	imageDefault := regexp.MustCompile(`(?m)^([A-Z0-9_]*IMAGE[A-Z0-9_]*)="\$\{[A-Z0-9_]+:-([^}]+)\}"`)
	pinned := regexp.MustCompile(`^[a-z0-9./_-]+:[A-Za-z0-9._-]+@sha256:[0-9a-f]{64}$`)

	for _, id := range readRepoExerciseDirIDs(t, exRoot) {
		checkPath := filepath.Join(exRoot, id, checkScriptName)
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
	root := repoRoot(t)
	exRoot := filepath.Join(root, "exercises")

	libPath := filepath.Join(exRoot, exerciseLibDir, "check-lib.sh")
	lib, err := os.ReadFile(libPath)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, libPath, string(lib), `"${DOCKER_BIN}" build --label "${DOCKER_LABEL}"`)

	for _, id := range readRepoExerciseDirIDs(t, exRoot) {
		dockerfilePath := filepath.Join(exRoot, id, "Dockerfile")
		if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
			continue
		} else if err != nil {
			t.Fatalf("%s: %v", dockerfilePath, err)
		}

		checkPath := filepath.Join(exRoot, id, checkScriptName)
		data, err := os.ReadFile(checkPath)
		if err != nil {
			t.Fatalf("%s: %v", checkPath, err)
		}
		script := string(data)

		assertContains(t, checkPath, script, `EXERCISE_ID="`+id+`"`)
		assertContains(t, checkPath, script, `DOCKER_LABEL="com.nginxlings.exercise=${EXERCISE_ID}"`)
		assertContains(t, checkPath, script, `nginxlings/${EXERCISE_ID}:check`)

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
	root := repoRoot(t)
	exRoot := filepath.Join(root, "exercises")

	expected := map[string]string{
		"05_install_verify":      "FROM debian:12.15-slim",
		"18_osi_layers_usecases": "FROM alpine:3.20.10",
	}
	for _, id := range readRepoExerciseDirIDs(t, exRoot) {
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
		assertContains(t, dockerfilePath, string(data), want)
	}
}

func TestRepoExercisesStartUnsolved(t *testing.T) {
	root := repoRoot(t)

	tests := []struct {
		name      string
		path      string
		required  []string
		forbidden []string
	}{
		{
			name: "static server config",
			path: filepath.Join(root, "exercises", "01_static_server", "nginx.conf"),
			required: []string{
				"TODO: point the document root",
				"root missing;",
				"index missing.html;",
			},
			forbidden: []string{"root .;", "index index.html;"},
		},
		{
			name:      "static server content",
			path:      filepath.Join(root, "exercises", "01_static_server", "index.html"),
			required:  []string{"TODO: replace this placeholder"},
			forbidden: []string{"Merhaba Nginxlings"},
		},
		{
			name: "shared zone sizing config",
			path: filepath.Join(root, "exercises", "16_shared_zone_sizing", "nginx.conf"),
			required: []string{
				"zone=req_zone:64k rate=1r/m;",
				"try_files /index.html =404;",
			},
			// burst=0 is rejected by nginx -t, and return would bypass limit_req.
			forbidden: []string{"burst=0", "return 200"},
		},
		{
			name:      "install and verify Dockerfile",
			path:      filepath.Join(root, "exercises", "05_install_verify", "Dockerfile"),
			required:  []string{"TODO: install NGINX"},
			forbidden: []string{"apt-get install"},
		},
		{
			name: "HTTP load balancer config",
			path: filepath.Join(root, "exercises", "07_http_load_balancer", "nginx.conf"),
			required: []string{
				"TODO: replace this placeholder with both local backend servers",
				"TODO: proxy requests to the backend upstream",
			},
			forbidden: []string{
				"server 127.0.0.1:8081;",
				"server 127.0.0.1:8082;",
				"proxy_pass http://backend;",
			},
		},
		{
			name: "upstream status zone config",
			path: filepath.Join(root, "exercises", "10_upstream_status_zone", "nginx.conf"),
			required: []string{
				"TODO: enable the required shared memory zone",
				"TODO: replace this placeholder with both Docker backend servers",
				"TODO: expose the selected upstream",
				"TODO: proxy requests to the backend upstream",
			},
			forbidden: []string{
				"zone up_mem 64k;",
				"server ex10_backend1:8081",
				"server ex10_backend2:8082",
				"add_header X-Upstream $upstream_addr",
				"proxy_pass http://backend;",
			},
		},
		{
			name:      "NGINX worker user config",
			path:      filepath.Join(root, "exercises", "17_verify_nginx_user", "nginx.conf"),
			required:  []string{"TODO: configure NGINX workers to run as the nginx user"},
			forbidden: []string{"user nginx;"},
		},
		{
			name: "OSI layer proxy config",
			path: filepath.Join(root, "exercises", "18_osi_layers_usecases", "nginx.conf"),
			required: []string{
				"TODO: load the NGINX stream module",
				"TODO: define an HTTP upstream and proxy this location to it",
				"TODO: expose the selected L7 upstream",
				"TODO: add a stream context that proxies port 9000",
			},
			forbidden: []string{
				"load_module /usr/lib/nginx/modules/ngx_stream_module.so;",
				"upstream http_pool",
				"proxy_pass http://http_pool;",
				"add_header X-L7-Upstream",
				"stream {",
				"upstream tcp_pool",
				"proxy_pass tcp_pool;",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatalf("%s: %v", tt.path, err)
			}
			content := string(data)
			for _, required := range tt.required {
				assertContains(t, tt.path, content, required)
			}
			for _, forbidden := range tt.forbidden {
				if strings.Contains(content, forbidden) {
					t.Fatalf("%s unexpectedly contains completed solution %q", tt.path, forbidden)
				}
			}
		})
	}

	legacyConfig := filepath.Join(root, "exercises", "07_http_load_balancer", "nginx.conf.default")
	if _, err := os.Stat(legacyConfig); err == nil {
		t.Fatalf("unused duplicate config still exists: %s", legacyConfig)
	} else if !os.IsNotExist(err) {
		t.Fatalf("inspect %s: %v", legacyConfig, err)
	}
}

func readRepoTranslationMap(t *testing.T, path string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	translations := map[string]string{}
	if err := json.Unmarshal(data, &translations); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return translations
}

func nonEmptyTranslationKeys(t *testing.T, label string, translations map[string]string) []string {
	t.Helper()
	keys := make([]string, 0, len(translations))
	for key, value := range translations {
		if strings.TrimSpace(value) == "" {
			t.Fatalf("%s translation %q is empty", label, key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func markdownHeadingLevels(content string) []string {
	var levels []string
	inFence := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		line = strings.TrimLeft(line, " \t")
		level := 0
		for level < len(line) && level < 6 && line[level] == '#' {
			level++
		}
		if level > 0 && len(line) > level && line[level] == ' ' {
			levels = append(levels, strings.Repeat("#", level))
		}
	}
	return levels
}

func assertContains(t *testing.T, path, content, want string) {
	t.Helper()
	if !strings.Contains(content, want) {
		t.Fatalf("%s does not contain %q", path, want)
	}
}

// TestRepoUIFormatVerbsMatchAcrossLanguages keeps fmt verbs aligned, so a
// translation cannot drop or add the %v that uif fills in.
func TestRepoUIFormatVerbsMatchAcrossLanguages(t *testing.T) {
	root := repoRoot(t)
	langsData, err := os.ReadFile(filepath.Join(root, "i18n", "langs.json"))
	if err != nil {
		t.Fatal(err)
	}
	var langs []LangMeta
	if err := json.Unmarshal(langsData, &langs); err != nil {
		t.Fatal(err)
	}

	verbs := regexp.MustCompile(`%[-+# 0]*[0-9]*(?:\.[0-9]+)?[a-zA-Z%]`)
	reference := readRepoTranslationMap(t, filepath.Join(root, "i18n", "ui", langs[0].Code+".json"))
	for _, lang := range langs[1:] {
		translated := readRepoTranslationMap(t, filepath.Join(root, "i18n", "ui", lang.Code+".json"))
		for key, want := range reference {
			wantVerbs := strings.Join(verbs.FindAllString(want, -1), " ")
			gotVerbs := strings.Join(verbs.FindAllString(translated[key], -1), " ")
			if gotVerbs != wantVerbs {
				t.Errorf("%s %s format verbs = %q, want %q like %s", lang.Code, key, gotVerbs, wantVerbs, langs[0].Code)
			}
		}
	}
}

// TestRepoGoStringLiteralsHaveNoTurkishText keeps user-facing text in the
// i18n files: Go literals are English fallbacks or developer-facing errors.
func TestRepoGoStringLiteralsHaveNoTurkishText(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "cmd", "nginxlings")
	fset := token.NewFileSet()
	packages, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}

	for _, pkg := range packages {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(node ast.Node) bool {
				lit, ok := node.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				value, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("%s: cannot unquote %s: %v", fset.Position(lit.Pos()), lit.Value, err)
				}
				if strings.ContainsAny(value, "çğıİöşüÇĞÖŞÜ") {
					t.Errorf("%s: Turkish text in Go literal %q; move it to i18n/ui", fset.Position(lit.Pos()), value)
				}
				return true
			})
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	return filepath.Clean(filepath.Join("..", ".."))
}

func readRepoManifestIDs(t *testing.T, exRoot string) []string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(exRoot, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}

	var manifest exerciseManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}

	ids := make([]string, 0, len(manifest.Exercises))
	seen := make(map[string]bool, len(manifest.Exercises))
	for _, entry := range manifest.Exercises {
		if seen[entry.ID] {
			t.Fatalf("duplicate manifest exercise ID %q", entry.ID)
		}
		seen[entry.ID] = true
		ids = append(ids, entry.ID)
	}
	sort.Strings(ids)
	return ids
}

func readRepoExerciseDirIDs(t *testing.T, exRoot string) []string {
	t.Helper()

	entries, err := os.ReadDir(exRoot)
	if err != nil {
		t.Fatal(err)
	}

	var ids []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || entry.Name() == exerciseLibDir {
			continue
		}
		if !exerciseIDPattern.MatchString(entry.Name()) {
			t.Fatalf("invalid exercise directory name %q", entry.Name())
		}
		ids = append(ids, entry.Name())
	}
	sort.Strings(ids)
	return ids
}
