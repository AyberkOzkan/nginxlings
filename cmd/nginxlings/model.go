package main

import (
	"context"
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

type marqueeTickMsg struct {
	generation uint64
}

type model struct {
	exercises []Exercise
	cursor    int

	width  int
	height int
	ready  bool

	listWidth         int
	rightWidth        int
	marqueeGeneration uint64
	marquee           int

	readme viewport.Model

	status              string
	loading             bool
	checkCancel         context.CancelFunc
	checkDone           <-chan struct{}
	instructionsContent string
	// checkOutputs holds the sanitized output of the last check per exercise
	// ID for this session; Tab shows the selected exercise's entry.
	checkOutputs map[string]string
	// panel is what the right panel currently shows.
	panel panelMode

	i18n      *I18n
	langIndex int

	spinner spinner.Model
	styles  Styles

	// lightBackground selects the light markdown theme. It is detected once
	// before Bubble Tea starts reading the terminal.
	lightBackground bool

	// Kalıcı state için
	paths AppPaths
	state *AppState

	// Reset onayı için
	confirmReset bool
}

// panelMode selects the content of the right panel.
type panelMode int

const (
	panelInstructions panelMode = iota
	panelCheckOutput
	panelHint
)

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

// ui returns the UI string for key in the current language. fallback is the
// English text shown when translations are missing or failed to load.
func (m model) ui(key, fallback string) string {
	if m.i18n == nil {
		return fallback
	}
	return m.i18n.T(m.currentLangCode(), key, fallback)
}

// uif is ui with fmt.Sprintf arguments applied to the translated format.
func (m model) uif(key, fallback string, args ...any) string {
	return fmt.Sprintf(m.ui(key, fallback), args...)
}

// İlk açılışta status satırı
func (m *model) initialStatusText() string {
	return m.ui("status_initial", "Pick an exercise, edit its files, then press c to check your solution.")
}

/* ---------- Model başlangıcı ---------- */

// newTUIModel resolves paths, rejects an unknown --lang value, and builds the
// initial TUI model.
func newTUIModel(opts cliOptions) (model, error) {
	paths, err := ResolveAppPathsWithOptions(opts.root, opts.state, opts.workspace)
	if err != nil {
		return model{}, err
	}
	if err := validateLangCode(paths.AssetRoot, opts.lang); err != nil {
		return model{}, err
	}
	if err := syncWorkspace(paths); err != nil {
		return model{}, err
	}
	return initialModelWithOptions(paths, opts.lang), nil
}

func initialModelWithOptions(paths AppPaths, initialLang string) model {
	m := model{
		langIndex: 0,
		spinner:   newSpinner(),
		styles:    DefaultStyles(),
		paths:     paths,
	}

	// i18n ve state yükle
	i18n, i18nErr := LoadI18n(paths.AssetRoot)
	state, stateErr := LoadState(paths.StateFile)
	m.i18n = i18n
	m.state = state

	// Dil önceliği: --lang, sonra state, sonra English. Metinler dil
	// seçildikten sonra üretilir ki hata mesajları da doğru dilde olsun.
	m.langIndex = m.langIndexFor("en", 0)
	if m.state != nil && m.state.LangCode != "" {
		m.langIndex = m.langIndexFor(m.state.LangCode, m.langIndex)
	}
	if initialLang != "" {
		m.langIndex = m.langIndexFor(initialLang, m.langIndex)
	}

	vp := viewport.New(0, 0)
	vp.SetContent(m.ui("readme_initial", "Select an exercise and read the instructions here."))
	m.readme = vp

	if i18nErr != nil {
		m.status = fmt.Sprintf("Translations could not be loaded: %v", i18nErr)
	}
	if stateErr != nil && m.status == "" {
		m.status = m.uif("status_state_load_failed", "Progress could not be loaded: %v", stateErr)
	}

	exercises, exercisesErr := LoadExercisesFrom(paths.ExercisesDir())
	if exercisesErr != nil && m.status == "" {
		m.status = m.uif("status_exercises_load_failed", "Exercises could not be loaded: %v", exercisesErr)
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

	// Eğer status hâlâ boşsa, default initial text
	if m.status == "" {
		m.status = m.initialStatusText()
	}

	return m
}

// langIndexFor returns the index of code in the loaded languages, or current
// when the code is unknown or translations are unavailable.
func (m model) langIndexFor(code string, current int) int {
	for i, meta := range m.i18n.Langs() {
		if meta.Code == code {
			return i
		}
	}
	return current
}

/* ---------- README yükleme (dil koduna göre) ---------- */

func (m *model) loadReadmeForCurrent() {
	m.panel = panelInstructions
	if len(m.exercises) == 0 {
		content := m.ui("status_no_exercises", "No exercises found.")
		m.instructionsContent = content
		m.readme.SetContent(content)
		m.readme.GotoTop()
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
			m.instructionsContent = rendered
			m.readme.SetContent(rendered)
			m.readme.GotoTop()
			return
		}
	}

	content := m.uif("readme_not_found", "README not found (%s).", strings.Join(candidates, ", "))
	m.instructionsContent = content
	m.readme.SetContent(content)
	m.readme.GotoTop()
}
