package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var exerciseIDPattern = regexp.MustCompile(`^[0-9]{2}_[a-z0-9_]+$`)

// exerciseLibDir holds shell helpers shared by check scripts. It is a
// reserved name under exercises/ and is not an exercise.
const exerciseLibDir = "lib"

type exerciseManifest struct {
	Exercises []exerciseManifestEntry `json:"exercises"`
}

type exerciseManifestEntry struct {
	ID          string `json:"id"`
	DefaultName string `json:"default_name"`
}

// LoadExercises discovers exercises from the filesystem and validates the
// minimum files the TUI and check runner need.
func LoadExercises(baseDir string) ([]Exercise, error) {
	exRoot := filepath.Join(baseDir, "exercises")

	entries, err := os.ReadDir(exRoot)
	if err != nil {
		return nil, fmt.Errorf("exercises directory cannot be read: %w", err)
	}

	defaultNames, err := loadExerciseDefaultNames(exRoot)
	if err != nil {
		return nil, err
	}

	var dirs []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == exerciseLibDir {
			continue
		}
		if !exerciseIDPattern.MatchString(name) {
			return nil, fmt.Errorf("invalid exercise directory %q: expected name like 01_static_server", name)
		}
		dirs = append(dirs, entry)
	}

	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Name() < dirs[j].Name()
	})

	if len(dirs) == 0 {
		return nil, fmt.Errorf("no exercise directories found in %s", exRoot)
	}

	dirIDs := make(map[string]bool, len(dirs))
	for _, dir := range dirs {
		dirIDs[dir.Name()] = true
	}
	for id := range defaultNames {
		if !dirIDs[id] {
			return nil, fmt.Errorf("exercise manifest references missing directory %q", id)
		}
	}

	exercises := make([]Exercise, 0, len(dirs))
	for _, dir := range dirs {
		id := dir.Name()
		path := filepath.Join(exRoot, id)
		if err := validateExerciseDir(path); err != nil {
			return nil, fmt.Errorf("invalid exercise %q: %w", id, err)
		}

		defaultName := defaultNames[id]
		if defaultName == "" {
			defaultName = defaultExerciseName(id)
		}

		exercises = append(exercises, Exercise{
			ID:          id,
			DefaultName: defaultName,
			Path:        path,
		})
	}

	return exercises, nil
}

func loadExerciseDefaultNames(exRoot string) (map[string]string, error) {
	path := filepath.Join(exRoot, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("exercise manifest cannot be read: %w", err)
	}

	var manifest exerciseManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("exercise manifest cannot be parsed: %w", err)
	}

	names := make(map[string]string, len(manifest.Exercises))
	for _, entry := range manifest.Exercises {
		if entry.ID == "" {
			return nil, fmt.Errorf("exercise manifest contains an entry without id")
		}
		if !exerciseIDPattern.MatchString(entry.ID) {
			return nil, fmt.Errorf("exercise manifest contains invalid id %q", entry.ID)
		}
		if strings.TrimSpace(entry.DefaultName) == "" {
			return nil, fmt.Errorf("exercise manifest entry %q has empty default_name", entry.ID)
		}
		if _, exists := names[entry.ID]; exists {
			return nil, fmt.Errorf("exercise manifest contains duplicate id %q", entry.ID)
		}
		names[entry.ID] = entry.DefaultName
	}

	return names, nil
}

func validateExerciseDir(path string) error {
	requiredFiles := []string{"README.en.md", "README.tr.md", "check.sh"}
	for _, name := range requiredFiles {
		info, err := os.Stat(filepath.Join(path, name))
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("missing %s", name)
			}
			return fmt.Errorf("%s cannot be read: %w", name, err)
		}
		if info.IsDir() {
			return fmt.Errorf("%s is a directory", name)
		}
	}
	return nil
}

func defaultExerciseName(id string) string {
	parts := strings.SplitN(id, "_", 2)
	if len(parts) != 2 {
		return id
	}
	return fmt.Sprintf("%s - %s", parts[0], titleFromSlug(parts[1]))
}

func titleFromSlug(slug string) string {
	words := strings.Split(slug, "_")
	for i, word := range words {
		if word == "" {
			continue
		}
		runes := []rune(word)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}
