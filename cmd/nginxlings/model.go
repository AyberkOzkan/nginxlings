package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

/* ---------- Veri tipleri ---------- */

type Exercise struct {
	ID          string
	DefaultName string
	Path        string
	Completed   bool
}

type checkResultMsg struct {
	output string
	err    error
	index  int
}

type model struct {
	exercises []Exercise
	cursor    int

	width  int
	height int
	ready  bool

	readme viewport.Model

	status  string
	loading bool

	i18n      *I18n
	langIndex int

	spinner spinner.Model
	styles  Styles

	// Kalıcı state için
	paths AppPaths
	state *AppState

	// Reset onayı için
	confirmReset bool
}

/* ---------- Yardımcı fonksiyonlar ---------- */

func newSpinner() spinner.Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	return sp
}

func lastLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

func (m model) currentLangCode() string {
	if m.i18n == nil {
		return ""
	}
	return m.i18n.LangCode(m.langIndex)
}

func (m model) currentLangLabel() string {
	if m.i18n == nil {
		return ""
	}
	return m.i18n.LangLabel(m.langIndex)
}

// Belirli bir ID'ye sahip egzersizin index'ini bul (yoksa -1)
func indexOfExercise(exs []Exercise, id string) int {
	for i, e := range exs {
		if e.ID == id {
			return i
		}
	}
	return -1
}

func (m *model) progressCounts() (completed, total int) {
	total = len(m.exercises)
	for _, ex := range m.exercises {
		if ex.Completed {
			completed++
		}
	}
	return
}

// İlk açılışta status satırı
func (m *model) initialStatusText() string {
	base := "↑/↓ ile egzersiz seç, c ile check, h ile hint, l ile dil değiştir, r ile reset, q ile çık."
	if m.i18n != nil {
		base = m.i18n.T(m.currentLangCode(), "status_initial", base)
	}
	return base
}

/* ---------- Model başlangıcı ---------- */

func initialModel() model {
	paths, pathsErr := ResolveAppPaths()

	m := model{
		langIndex: 0,
		spinner:   newSpinner(),
		styles:    DefaultStyles(),
		paths:     paths,
	}

	// README viewport
	vp := viewport.New(0, 0)
	defaultReadme := "Select an exercise and read the instructions here."
	vp.SetContent(defaultReadme)
	m.readme = vp

	if pathsErr != nil {
		m.status = fmt.Sprintf("Uygulama yolları çözülemedi: %v", pathsErr)
		return m
	}

	// i18n yükle
	i18n, i18nErr := LoadI18n(paths.AssetRoot)

	// state yükle
	state, stateErr := LoadState(paths.StateFile)

	m.i18n = i18n
	m.state = state

	m.langIndex = 0
	if m.i18n != nil {
		for i, meta := range m.i18n.Langs() {
			if meta.Code == "en" {
				m.langIndex = i
				break
			}
		}
	}

	if i18n != nil {
		defaultReadme = i18n.T(m.currentLangCode(), "readme_initial", defaultReadme)
	}
	vp.SetContent(defaultReadme)
	m.readme = vp

	if i18nErr != nil {
		m.status = fmt.Sprintf("i18n yüklenemedi: %v", i18nErr)
	}
	if stateErr != nil && m.status == "" {
		m.status = fmt.Sprintf("State yüklenemedi: %v", stateErr)
	}

	exercises, exercisesErr := LoadExercises(paths.AssetRoot)
	if exercisesErr != nil && m.status == "" {
		m.status = fmt.Sprintf("Egzersizler yüklenemedi: %v", exercisesErr)
	}

	// State'ten tamamlananları işaretle
	if m.state != nil && m.state.Completed != nil {
		for i := range exercises {
			if m.state.Completed[exercises[i].ID] {
				exercises[i].Completed = true
			}
		}
	}

	m.exercises = exercises

	// Varsayılan cursor 0
	m.cursor = 0

	// State'ten en son egzersizi yükle
	if m.state != nil && m.state.LastExerciseID != "" {
		if idx := indexOfExercise(m.exercises, m.state.LastExerciseID); idx >= 0 {
			m.cursor = idx
		}
	}

	// State'ten dil tercihini yükle
	if m.i18n != nil && m.state != nil && m.state.LangCode != "" {
		for i, meta := range m.i18n.Langs() {
			if meta.Code == m.state.LangCode {
				m.langIndex = i
				break
			}
		}
	}

	// Eğer status hâlâ boşsa, default initial text
	if m.status == "" {
		m.status = m.initialStatusText()
	}

	return m
}

/* ---------- README yükleme (dil koduna göre) ---------- */

func (m *model) loadReadmeForCurrent() {
	if len(m.exercises) == 0 {
		if m.i18n != nil {
			m.readme.SetContent(
				m.i18n.T(m.currentLangCode(), "status_no_exercises", "Hiç egzersiz yok."),
			)
		} else {
			m.readme.SetContent("Hiç egzersiz yok.")
		}
		return
	}

	ex := m.exercises[m.cursor]
	lang := m.currentLangCode()

	var candidates []string
	if lang != "" {
		candidates = []string{
			fmt.Sprintf("README.%s.md", lang),
			fmt.Sprintf("README.%s", lang),
			"README.md",
		}
	} else {
		candidates = []string{"README.md"}
	}

	for _, name := range candidates {
		path := filepath.Join(ex.Path, name)
		if data, err := os.ReadFile(path); err == nil {
			rendered := m.renderMarkdown(string(data))
			m.readme.SetContent(rendered)
			return
		}
	}

	m.readme.SetContent(fmt.Sprintf("README bulunamadı (%s).", strings.Join(candidates, ", ")))
}
