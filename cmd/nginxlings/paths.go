package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	appDirName      = "nginxlings"
	stateFileName   = "state.json"
	legacyStateName = ".nginxlings_state.json"
	envAssetRoot    = "NGINXLINGS_ASSET_ROOT"
	envWorkspace    = "NGINXLINGS_WORKSPACE"
	envStateRoot    = "NGINXLINGS_STATE_ROOT"
	envStateFile    = "NGINXLINGS_STATE_FILE"
	envXDGStateHome = "XDG_STATE_HOME"
	envHome         = "HOME"
	envUserProfile  = "USERPROFILE"
)

type AppPaths struct {
	WorkDir   string
	AppRoot   string
	AssetRoot string
	// Workspace is the learner's own copy of the exercises. It is empty when
	// the exercises are edited where they ship, as in the repository or an
	// extracted release archive.
	Workspace string
	StateRoot string
	StateFile string
}

// ExercisesDir is where the learner edits and checks the exercises.
func (p AppPaths) ExercisesDir() string {
	if p.Workspace != "" {
		return p.Workspace
	}
	return filepath.Join(p.AssetRoot, "exercises")
}

func ResolveAppPaths() (AppPaths, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return AppPaths{}, fmt.Errorf("working directory cannot be read: %w", err)
	}
	executable, _ := os.Executable()
	return resolveAppPaths(workDir, executable, os.Getenv)
}

func ResolveAppPathsWithOptions(assetRoot, stateFile, workspace string) (AppPaths, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return AppPaths{}, fmt.Errorf("working directory cannot be read: %w", err)
	}
	executable, _ := os.Executable()
	return resolveAppPaths(workDir, executable, envOverlay(assetRoot, stateFile, workspace))
}

// ResolveExercisePathsWithOptions resolves the asset root and the workspace
// for commands that need no state file.
func ResolveExercisePathsWithOptions(assetRoot, workspace string) (AppPaths, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return AppPaths{}, fmt.Errorf("working directory cannot be read: %w", err)
	}
	executable, _ := os.Executable()
	getenv := envOverlay(assetRoot, "", workspace)
	root, installed, err := resolveAssetRoot(workDir, executable, getenv)
	if err != nil {
		return AppPaths{}, err
	}
	ws, err := resolveWorkspace(installed, getenv)
	if err != nil {
		return AppPaths{}, err
	}
	return AppPaths{WorkDir: workDir, AppRoot: root, AssetRoot: root, Workspace: ws}, nil
}

func ResolveAssetRootWithOptions(assetRoot string) (string, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("working directory cannot be read: %w", err)
	}
	executable, _ := os.Executable()
	root, _, err := resolveAssetRoot(workDir, executable, envOverlay(assetRoot, "", ""))
	return root, err
}

func resolveAppPaths(workDir, executable string, getenv func(string) string) (AppPaths, error) {
	assetRoot, installed, err := resolveAssetRoot(workDir, executable, getenv)
	if err != nil {
		return AppPaths{}, err
	}
	workspace, err := resolveWorkspace(installed, getenv)
	if err != nil {
		return AppPaths{}, err
	}

	stateFile, stateRoot, err := resolveStateFile(getenv)
	if err != nil {
		return AppPaths{}, err
	}

	return AppPaths{
		WorkDir:   workDir,
		AppRoot:   assetRoot,
		AssetRoot: assetRoot,
		Workspace: workspace,
		StateRoot: stateRoot,
		StateFile: stateFile,
	}, nil
}

// resolveAssetRoot finds the directory with exercises/ and i18n/. installed
// reports that it is the share directory of an installed package, whose files
// the learner cannot edit.
func resolveAssetRoot(workDir, executable string, getenv func(string) string) (root string, installed bool, err error) {
	if root := getenv(envAssetRoot); root != "" {
		abs, err := filepath.Abs(root)
		if err != nil {
			return "", false, fmt.Errorf("%s cannot be resolved: %w", envAssetRoot, err)
		}
		if !hasAppAssets(abs) {
			return "", false, fmt.Errorf("%s does not contain exercises/ and i18n/: %s", envAssetRoot, abs)
		}
		return abs, false, nil
	}

	if root, ok := findAssetRoot(workDir); ok {
		return root, false, nil
	}

	if executable != "" {
		if root, ok := findAssetRoot(filepath.Dir(executable)); ok {
			return root, false, nil
		}
		// A package installs <prefix>/bin/nginxlings and its assets in
		// <prefix>/share/nginxlings.
		share := filepath.Join(filepath.Dir(filepath.Dir(executable)), "share", appDirName)
		if hasAppAssets(share) {
			return share, true, nil
		}
	}

	return "", false, fmt.Errorf("asset root not found: run from the repository root or set %s", envAssetRoot)
}

// resolveWorkspace returns the learner's exercise directory: the
// NGINXLINGS_WORKSPACE value, ~/nginxlings for an installed package, or ""
// to edit the exercises where they ship.
func resolveWorkspace(installed bool, getenv func(string) string) (string, error) {
	if dir := getenv(envWorkspace); dir != "" {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return "", fmt.Errorf("%s cannot be resolved: %w", envWorkspace, err)
		}
		return abs, nil
	}
	if !installed {
		return "", nil
	}
	home := getenv(envHome)
	if home == "" {
		return "", fmt.Errorf("the exercise folder cannot be placed: set %s or %s", envHome, envWorkspace)
	}
	return filepath.Join(home, appDirName), nil
}

func findAssetRoot(start string) (string, bool) {
	if start == "" {
		return "", false
	}
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}

	for {
		if hasAppAssets(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func hasAppAssets(root string) bool {
	return dirExists(filepath.Join(root, "exercises")) && dirExists(filepath.Join(root, "i18n"))
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func resolveStateFile(getenv func(string) string) (stateFile, stateRoot string, err error) {
	if path := getenv(envStateFile); path != "" {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", "", fmt.Errorf("%s cannot be resolved: %w", envStateFile, err)
		}
		return abs, filepath.Dir(abs), nil
	}

	root, err := resolveStateRoot(getenv)
	if err != nil {
		return "", "", err
	}
	return filepath.Join(root, stateFileName), root, nil
}

func resolveStateRoot(getenv func(string) string) (string, error) {
	if root := getenv(envStateRoot); root != "" {
		abs, err := filepath.Abs(root)
		if err != nil {
			return "", fmt.Errorf("%s cannot be resolved: %w", envStateRoot, err)
		}
		return abs, nil
	}

	if root := getenv(envXDGStateHome); root != "" {
		return filepath.Join(root, appDirName), nil
	}

	home := getenv(envHome)
	if home == "" && runtime.GOOS == "windows" {
		home = getenv(envUserProfile)
	}
	if home == "" {
		return "", fmt.Errorf("state root cannot be resolved: set %s or %s", envStateRoot, envXDGStateHome)
	}

	if runtime.GOOS == "windows" {
		return filepath.Join(home, "AppData", "Local", appDirName, "State"), nil
	}
	return filepath.Join(home, ".local", "state", appDirName), nil
}
