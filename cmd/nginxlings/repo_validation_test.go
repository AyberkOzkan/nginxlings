package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
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

func TestRepoDockerExercisesUseScopedResources(t *testing.T) {
	root := repoRoot(t)
	exRoot := filepath.Join(root, "exercises")

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
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
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
