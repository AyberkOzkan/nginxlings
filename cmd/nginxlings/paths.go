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
	StateRoot string
	StateFile string
}

func ResolveAppPaths() (AppPaths, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return AppPaths{}, fmt.Errorf("working directory cannot be read: %w", err)
	}
	executable, _ := os.Executable()
	return resolveAppPaths(workDir, executable, os.Getenv)
}

func resolveAppPaths(workDir, executable string, getenv func(string) string) (AppPaths, error) {
	assetRoot, err := resolveAssetRoot(workDir, executable, getenv)
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
		StateRoot: stateRoot,
		StateFile: stateFile,
	}, nil
}

func resolveAssetRoot(workDir, executable string, getenv func(string) string) (string, error) {
	if root := getenv(envAssetRoot); root != "" {
		abs, err := filepath.Abs(root)
		if err != nil {
			return "", fmt.Errorf("%s cannot be resolved: %w", envAssetRoot, err)
		}
		if !hasAppAssets(abs) {
			return "", fmt.Errorf("%s does not contain exercises/ and i18n/: %s", envAssetRoot, abs)
		}
		return abs, nil
	}

	if root, ok := findAssetRoot(workDir); ok {
		return root, nil
	}

	if executable != "" {
		if root, ok := findAssetRoot(filepath.Dir(executable)); ok {
			return root, nil
		}
	}

	return "", fmt.Errorf("asset root not found: run from the repository root or set %s", envAssetRoot)
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
