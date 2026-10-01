package repo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nginxlings/tests/testutil"
)

func TestRepoExerciseReadmeLanguageParity(t *testing.T) {
	root := testutil.RepoRoot(t)
	exRoot := filepath.Join(root, "exercises")

	for _, id := range testutil.ExerciseIDs(t, exRoot) {
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
	root := testutil.RepoRoot(t)
	exRoot := filepath.Join(root, "exercises")

	for _, id := range testutil.ExerciseIDs(t, exRoot) {
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
	root := testutil.RepoRoot(t)
	exerciseRoot := filepath.Join(root, "exercises", "10_upstream_status_zone")

	checkPath := filepath.Join(exerciseRoot, testutil.CheckScriptName)
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
		testutil.AssertContains(t, checkPath, checkScript, required)
	}

	configPath := filepath.Join(exerciseRoot, "nginx.conf")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("%s: %v", configPath, err)
	}
	testutil.AssertContains(t, configPath, string(configData), "resolver 127.0.0.11")
	testutil.AssertContains(t, configPath, string(configData), "TODO: replace this placeholder with both Docker backend servers")

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
			testutil.AssertContains(t, path, content, required)
		}
		for _, loopbackBackend := range []string{"127.0.0.1:8081", "127.0.0.1:8082"} {
			if strings.Contains(content, loopbackBackend) {
				t.Fatalf("%s documents unreachable backend %q", path, loopbackBackend)
			}
		}
	}
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

// TestRepoReadmeTitleMatchesListName keeps the title at the top of the
// instructions panel identical to the exercise's name in the list, in every
// language. The list uses "NN - Name" and the README "# NN – Name".
func TestRepoReadmeTitleMatchesListName(t *testing.T) {
	root := testutil.RepoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "i18n", "langs.json"))
	if err != nil {
		t.Fatal(err)
	}
	var langs []langMeta
	if err := json.Unmarshal(data, &langs); err != nil {
		t.Fatal(err)
	}
	for _, lang := range langs {
		names := readRepoTranslationMap(t, filepath.Join(root, "i18n", "exercises", lang.Code+".json"))
		for _, id := range testutil.ExerciseIDs(t, filepath.Join(root, "exercises")) {
			path := filepath.Join(root, "exercises", id, "README."+lang.Code+".md")
			readme, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%s: %v", path, err)
			}
			title := ""
			for _, line := range strings.Split(string(readme), "\n") {
				if strings.HasPrefix(line, "# ") {
					title = strings.TrimPrefix(line, "# ")
					break
				}
			}
			if want := strings.Replace(names[id], " - ", " – ", 1); title != want {
				t.Errorf("%s title = %q, want the list name %q", path, title, want)
			}
		}
	}
}
