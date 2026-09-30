package repo

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

	"nginxlings/tests/testutil"
)

func TestRepoManifestMatchesExerciseDirectories(t *testing.T) {
	root := testutil.RepoRoot(t)
	exRoot := filepath.Join(root, "exercises")

	manifestIDs := readRepoManifestIDs(t, exRoot)
	dirIDs := testutil.ExerciseIDs(t, exRoot)

	if strings.Join(manifestIDs, ",") != strings.Join(dirIDs, ",") {
		t.Fatalf("manifest IDs = %v, want exercise directories %v", manifestIDs, dirIDs)
	}
}

// TestRepoManifestFallbackNamesAreEnglish keeps manifest default names equal
// to the English titles, because they are shown when translations fail to
// load and English is the default UI language.
func TestRepoManifestFallbackNamesAreEnglish(t *testing.T) {
	root := testutil.RepoRoot(t)
	english := readRepoTranslationMap(t, filepath.Join(root, "i18n", "exercises", "en.json"))
	for _, entry := range readRepoManifest(t, filepath.Join(root, "exercises")).Exercises {
		if entry.DefaultName != english[entry.ID] {
			t.Errorf("%s default_name = %q, want English title %q", entry.ID, entry.DefaultName, english[entry.ID])
		}
	}
}

func TestRepoTranslationFilesCoverManifest(t *testing.T) {
	root := testutil.RepoRoot(t)
	manifestIDs := readRepoManifestIDs(t, filepath.Join(root, "exercises"))

	langsData, err := os.ReadFile(filepath.Join(root, "i18n", "langs.json"))
	if err != nil {
		t.Fatal(err)
	}
	var langs []langMeta
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

// TestRepoUIFormatVerbsMatchAcrossLanguages keeps fmt verbs aligned, so a
// translation cannot drop or add the %v that uif fills in.
func TestRepoUIFormatVerbsMatchAcrossLanguages(t *testing.T) {
	root := testutil.RepoRoot(t)
	langsData, err := os.ReadFile(filepath.Join(root, "i18n", "langs.json"))
	if err != nil {
		t.Fatal(err)
	}
	var langs []langMeta
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
	dir := filepath.Join(testutil.RepoRoot(t), "cmd", "nginxlings")
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

func readRepoManifestIDs(t *testing.T, exRoot string) []string {
	t.Helper()

	manifest := readRepoManifest(t, exRoot)

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

// langMeta mirrors one entry of i18n/langs.json.
type langMeta struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

// exerciseManifest mirrors exercises/manifest.json.
type exerciseManifest struct {
	Exercises []struct {
		ID          string `json:"id"`
		DefaultName string `json:"default_name"`
	} `json:"exercises"`
}

func readRepoManifest(t *testing.T, exRoot string) exerciseManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(exRoot, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest exerciseManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}
