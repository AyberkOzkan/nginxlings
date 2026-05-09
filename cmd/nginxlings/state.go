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

func (s *AppState) Save(path string) error {
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

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("state dizini oluşturulamadı: %w", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("state yazılamadı: %w", err)
	}
	return nil
}
