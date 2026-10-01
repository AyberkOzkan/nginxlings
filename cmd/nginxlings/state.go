package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// AppState: Kullanıcının kalıcı durumu
// - Hangi egzersizler tamamlandı?
// - En son hangi egzersizdeydi?
// - Hangi dili seçmişti?
type AppState struct {
	Completed      map[string]bool `json:"completed"`
	LastExerciseID string          `json:"last_exercise_id"`
	LangCode       string          `json:"lang_code"`
}

func defaultState() *AppState {
	return &AppState{
		Completed: make(map[string]bool),
	}
}

func LoadState(path string) (*AppState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// İlk çalıştırma: state yoksa default
			return defaultState(), nil
		}
		return defaultState(), fmt.Errorf("state cannot be read: %w", err)
	}

	st := defaultState()
	if err := json.Unmarshal(data, st); err != nil {
		return defaultState(), fmt.Errorf("state cannot be parsed: %w", err)
	}

	if st.Completed == nil {
		st.Completed = make(map[string]bool)
	}

	return st, nil
}

func (s *AppState) Save(path string) error {
	return s.save(path, os.Rename)
}

func (s *AppState) save(path string, renameFile func(string, string) error) error {
	if s == nil {
		return nil
	}
	if s.Completed == nil {
		s.Completed = make(map[string]bool)
	}

	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("state cannot be encoded: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("state directory cannot be created: %w", err)
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("state temporary file cannot be created: %w", err)
	}
	tmpPath := tmp.Name()
	closed := false
	defer func() {
		if !closed {
			_ = tmp.Close()
		}
		_ = os.Remove(tmpPath)
	}()

	if err := tmp.Chmod(0o644); err != nil {
		return fmt.Errorf("state temporary file mode cannot be set: %w", err)
	}
	if _, err := tmp.Write(b); err != nil {
		return fmt.Errorf("state temporary file cannot be written: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("state temporary file cannot be synced: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("state temporary file cannot be closed: %w", err)
	}
	closed = true

	if err := renameFile(tmpPath, path); err != nil {
		return fmt.Errorf("state file cannot be replaced atomically: %w", err)
	}

	dirHandle, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("state directory cannot be opened: %w", err)
	}
	defer dirHandle.Close()
	if err := dirHandle.Sync(); err != nil {
		return fmt.Errorf("state directory cannot be synced: %w", err)
	}
	return nil
}
