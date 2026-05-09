package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const checkScriptName = "check.sh"

func runExerciseCheck(ex Exercise) (string, error) {
	if err := validateCheckScript(ex, runtime.GOOS); err != nil {
		return "", err
	}

	cmd := exec.Command("./" + checkScriptName)
	cmd.Dir = ex.Path

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	return buf.String(), err
}

func validateCheckScript(ex Exercise, goos string) error {
	if goos == "windows" {
		return fmt.Errorf("native Windows is not supported for %s; run Nginxlings in WSL2 or another POSIX shell environment", checkScriptName)
	}

	path := filepath.Join(ex.Path, checkScriptName)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s not found for exercise %s", checkScriptName, ex.ID)
		}
		return fmt.Errorf("%s cannot be read for exercise %s: %w", checkScriptName, ex.ID, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory for exercise %s", checkScriptName, ex.ID)
	}
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("%s is not executable for exercise %s; run chmod +x %s", checkScriptName, ex.ID, path)
	}
	return nil
}
