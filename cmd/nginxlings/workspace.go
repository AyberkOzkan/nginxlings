package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// workspaceStateName is the file in a workspace that records the shipped
// content of every exercise file. An update compares against it to tell the
// files the learner changed from the ones it may replace.
const workspaceStateName = ".nginxlings-workspace.json"

type workspaceState struct {
	// Files maps a slash-separated path relative to the workspace to the
	// SHA-256 of the content that was shipped for it.
	Files map[string]string `json:"files"`
}

// isAppOwned reports whether a workspace file belongs to the app rather than
// the learner: check scripts, instructions, the shared check library and the
// manifest always match the installed version.
func isAppOwned(rel string) bool {
	base := path.Base(rel)
	return rel == "manifest.json" ||
		strings.HasPrefix(rel, exerciseLibDir+"/") ||
		base == checkScriptName ||
		strings.HasPrefix(base, "README") && strings.HasSuffix(base, ".md")
}

// SyncWorkspace brings the learner's workspace up to date with the shipped
// exercises in template. A missing workspace becomes a full copy. Later runs
// add new exercises and files, always replace app-owned files, and replace or
// remove other shipped files only while the learner has not changed them, so
// the learner's work is never overwritten.
func SyncWorkspace(template, workspace string) error {
	state, err := loadWorkspaceState(workspace)
	if err != nil {
		return err
	}

	shipped := make(map[string]string)
	err = filepath.WalkDir(template, func(src string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(template, src)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		data, mode, err := readShippedFile(src, d)
		if err != nil {
			return err
		}
		sum := sha256Hex(data)
		shipped[rel] = sum

		dst := filepath.Join(workspace, filepath.FromSlash(rel))
		current, err := os.ReadFile(dst)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			if _, tracked := state.Files[rel]; tracked && !isAppOwned(rel) {
				// The learner deleted a file they own; keep it deleted.
				return nil
			}
			return writeWorkspaceFile(dst, data, mode)
		case err != nil:
			return fmt.Errorf("exercise file cannot be read: %w", err)
		case bytes.Equal(current, data):
			return os.Chmod(dst, mode)
		case isAppOwned(rel) || sha256Hex(current) == state.Files[rel]:
			return writeWorkspaceFile(dst, data, mode)
		default:
			// The learner changed this file: keep their version.
			return nil
		}
	})
	if err != nil {
		return fmt.Errorf("exercise folder %s cannot be updated: %w", workspace, err)
	}

	for rel, sum := range state.Files {
		if _, still := shipped[rel]; still {
			continue
		}
		dst := filepath.Join(workspace, filepath.FromSlash(rel))
		if current, err := os.ReadFile(dst); err == nil && sha256Hex(current) == sum {
			if err := os.Remove(dst); err != nil {
				return fmt.Errorf("dropped exercise file cannot be removed: %w", err)
			}
		}
	}

	state.Files = shipped
	return saveWorkspaceState(workspace, state)
}

// syncWorkspace updates the learner's workspace from the installed
// exercises; it does nothing when the exercises are edited where they ship.
func syncWorkspace(paths AppPaths) error {
	if paths.Workspace == "" {
		return nil
	}
	return SyncWorkspace(filepath.Join(paths.AssetRoot, "exercises"), paths.Workspace)
}

// ResetExercise replaces the learner's copy of exercise id with the shipped
// starting files. Everything in the exercise directory is replaced, including
// files the learner added.
func ResetExercise(template, workspace, id string) error {
	if !exerciseIDPattern.MatchString(id) {
		return fmt.Errorf("invalid exercise id %q", id)
	}
	src := filepath.Join(template, id)
	if !dirExists(src) {
		return fmt.Errorf("exercise %q is not installed", id)
	}
	state, err := loadWorkspaceState(workspace)
	if err != nil {
		return err
	}
	dst := filepath.Join(workspace, id)
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("exercise folder %s cannot be removed: %w", dst, err)
	}
	for rel := range state.Files {
		if strings.HasPrefix(rel, id+"/") {
			delete(state.Files, rel)
		}
	}
	err = filepath.WalkDir(src, func(file string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(template, file)
		if err != nil {
			return err
		}
		data, mode, err := readShippedFile(file, d)
		if err != nil {
			return err
		}
		state.Files[filepath.ToSlash(rel)] = sha256Hex(data)
		return writeWorkspaceFile(filepath.Join(workspace, rel), data, mode)
	})
	if err != nil {
		return fmt.Errorf("exercise %s cannot be reset: %w", id, err)
	}
	return saveWorkspaceState(workspace, state)
}

// loadWorkspaceState reads the workspace's record of shipped files. A
// missing or empty directory is a new workspace; an existing directory
// without the record belongs to something else and is left alone.
func loadWorkspaceState(workspace string) (workspaceState, error) {
	state := workspaceState{Files: make(map[string]string)}
	data, err := os.ReadFile(filepath.Join(workspace, workspaceStateName))
	if errors.Is(err, fs.ErrNotExist) {
		entries, dirErr := os.ReadDir(workspace)
		if dirErr == nil && len(entries) > 0 {
			return state, fmt.Errorf("%s already exists and is not a nginxlings exercise folder; move it away or set %s to another folder", workspace, envWorkspace)
		}
		return state, nil
	}
	if err != nil {
		return state, fmt.Errorf("exercise folder record cannot be read: %w", err)
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, fmt.Errorf("exercise folder record %s cannot be parsed: %w", filepath.Join(workspace, workspaceStateName), err)
	}
	if state.Files == nil {
		state.Files = make(map[string]string)
	}
	return state, nil
}

func saveWorkspaceState(workspace string, state workspaceState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("exercise folder record cannot be encoded: %w", err)
	}
	return writeWorkspaceFile(filepath.Join(workspace, workspaceStateName), append(data, '\n'), 0o644)
}

// readShippedFile returns a template file's content and the mode its copy
// gets: executable files stay executable, everything else is 0644.
func readShippedFile(file string, d fs.DirEntry) ([]byte, os.FileMode, error) {
	info, err := d.Info()
	if err != nil {
		return nil, 0, err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, 0, err
	}
	mode := os.FileMode(0o644)
	if info.Mode().Perm()&0o111 != 0 {
		mode = 0o755
	}
	return data, mode, nil
}

// writeWorkspaceFile replaces dst through a temporary file, so an interrupted
// update never leaves half a file behind.
func writeWorkspaceFile(dst string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), "."+filepath.Base(dst)+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dst)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
