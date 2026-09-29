package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type LangMeta struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type I18n struct {
	langs         []LangMeta
	uiStrings     map[string]map[string]string // langCode -> key -> value
	exerciseNames map[string]map[string]string // langCode -> exerciseID -> name
}

func LoadI18n(root string) (*I18n, error) {
	i18nDir := filepath.Join(root, "i18n")

	// 1) langs.json
	data, err := os.ReadFile(filepath.Join(i18nDir, "langs.json"))
	if err != nil {
		return nil, fmt.Errorf("langs.json cannot be read: %w", err)
	}

	var langs []LangMeta
	if err := json.Unmarshal(data, &langs); err != nil {
		return nil, fmt.Errorf("langs.json cannot be parsed: %w", err)
	}
	if len(langs) == 0 {
		return nil, fmt.Errorf("langs.json lists no languages")
	}

	uiDir := filepath.Join(i18nDir, "ui")
	exDir := filepath.Join(i18nDir, "exercises")

	ui := make(map[string]map[string]string)
	exNames := make(map[string]map[string]string)

	for _, l := range langs {
		code := l.Code

		// 2) UI strings: i18n/ui/<code>.json
		uiFile := filepath.Join(uiDir, fmt.Sprintf("%s.json", code))
		if b, err := os.ReadFile(uiFile); err == nil {
			m := map[string]string{}
			if err := json.Unmarshal(b, &m); err == nil {
				ui[code] = m
			}
		}

		// 3) Exercise names: i18n/exercises/<code>.json
		exFile := filepath.Join(exDir, fmt.Sprintf("%s.json", code))
		if b, err := os.ReadFile(exFile); err == nil {
			m := map[string]string{}
			if err := json.Unmarshal(b, &m); err == nil {
				exNames[code] = m
			}
		}
	}

	return &I18n{
		langs:         langs,
		uiStrings:     ui,
		exerciseNames: exNames,
	}, nil
}

// validateLangCode rejects an explicit language code that i18n/langs.json does
// not list. An empty code means "use the default". When translations cannot be
// loaded the code cannot be checked; the TUI then reports the load failure.
func validateLangCode(root, code string) error {
	if code == "" {
		return nil
	}
	i18n, err := LoadI18n(root)
	if err != nil {
		return nil
	}
	codes := make([]string, 0, len(i18n.Langs()))
	for _, meta := range i18n.Langs() {
		if meta.Code == code {
			return nil
		}
		codes = append(codes, meta.Code)
	}
	return fmt.Errorf("unknown language %q (available: %s)", code, strings.Join(codes, ", "))
}

func (i *I18n) Langs() []LangMeta {
	if i == nil {
		return nil
	}
	return i.langs
}

func (i *I18n) NextLangIndex(current int) int {
	if i == nil || len(i.langs) == 0 {
		return current
	}
	return (current + 1) % len(i.langs)
}

func (i *I18n) LangCode(idx int) string {
	if i == nil || len(i.langs) == 0 {
		return ""
	}
	if idx < 0 || idx >= len(i.langs) {
		return i.langs[0].Code
	}
	return i.langs[idx].Code
}

func (i *I18n) LangLabel(idx int) string {
	if i == nil || len(i.langs) == 0 {
		return ""
	}
	if idx < 0 || idx >= len(i.langs) {
		return i.langs[0].Label
	}
	return i.langs[idx].Label
}

// UI string çevirisi
func (i *I18n) T(langCode, key, fallback string) string {
	if i == nil {
		return fallback
	}
	if m, ok := i.uiStrings[langCode]; ok {
		if v, ok := m[key]; ok && v != "" {
			return v
		}
	}
	return fallback
}

// Egzersiz adı çevirisi
func (i *I18n) ExerciseName(langCode, exID, fallback string) string {
	if i == nil {
		return fallback
	}
	if m, ok := i.exerciseNames[langCode]; ok {
		if v, ok := m[exID]; ok && v != "" {
			return v
		}
	}
	return fallback
}
