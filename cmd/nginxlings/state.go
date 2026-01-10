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

func stateFile(baseDir string) string {
	// Proje kökünde gizli bir JSON dosyası:
	//   /home/.../nginxlings/.nginxlings_state.json
	return filepath.Join(baseDir, ".nginxlings_state.json")
}

func LoadState(baseDir string) (*AppState, error) {
	path := stateFile(baseDir)

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// İlk çalıştırma: state yoksa default
			return defaultState(), nil
		}
		return defaultState(), fmt.Errorf("state okunamadı: %w", err)
	}

	st := defaultState()
	if err := json.Unmarshal(data, st); err != nil {
		return defaultState(), fmt.Errorf("state parse edilemedi: %w", err)
	}

	if st.Completed == nil {
		st.Completed = make(map[string]bool)
	}

	return st, nil
}

func (s *AppState) Save(baseDir string) error {
	if s == nil {
		return nil
	}
	if s.Completed == nil {
		s.Completed = make(map[string]bool)
	}

	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("state marshal hatası: %w", err)
	}

	path := stateFile(baseDir)
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("state yazılamadı: %w", err)
	}
	return nil
}
