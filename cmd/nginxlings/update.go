package main

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

/* ---------- check.sh  ---------- */

func runCheck(ctx context.Context, index int, ex Exercise, done chan<- struct{}) tea.Cmd {
	return func() tea.Msg {
		defer close(done)
		out, err := runExerciseCheckContext(ctx, ex)

		return checkResultMsg{
			output: out,
			err:    err,
			index:  index,
		}
	}
}

func cancelCheckAndQuit(cancel context.CancelFunc, done <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		cancel()
		if done != nil {
			<-done
		}
		return tea.Quit()
	}
}

const (
	marqueeTickInterval  = 350 * time.Millisecond
	marqueePauseInterval = 1050 * time.Millisecond
)

func marqueeTick(generation uint64, delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return marqueeTickMsg{generation: generation}
	})
}

func marqueeTickDelay(offset, cycleLength int) time.Duration {
	if offset == 0 || cycleLength > 0 && offset%cycleLength == 0 {
		return marqueePauseInterval
	}
	return marqueeTickInterval
}

func (m model) selectedExerciseName() string {
	if len(m.exercises) == 0 || m.cursor < 0 || m.cursor >= len(m.exercises) {
		return ""
	}

	ex := m.exercises[m.cursor]
	if m.i18n != nil {
		return m.i18n.ExerciseName(m.currentLangCode(), ex.ID, ex.DefaultName)
	}
	return ex.DefaultName
}

func (m model) selectedNameOverflows() bool {
	if len(m.exercises) == 0 || m.cursor < 0 || m.cursor >= len(m.exercises) {
		return false
	}
	layout := m.layout()
	nameWidth := layout.listNameWidth()
	if layout.compact {
		nameWidth = maxInt(1, m.width-4)
	}
	return lipgloss.Width(m.selectedExerciseName()) > nameWidth
}

func (m model) selectedMarqueeCycleLength() int {
	name := m.selectedExerciseName()
	if name == "" {
		return 0
	}
	return len([]rune(name + "   "))
}

func (m *model) resetMarquee() tea.Cmd {
	m.marquee = 0
	m.marqueeGeneration++
	if !m.selectedNameOverflows() {
		return nil
	}
	return marqueeTick(m.marqueeGeneration, marqueeTickDelay(0, m.selectedMarqueeCycleLength()))
}

func (m *model) persistSelectedExercise() {
	if m.state == nil || m.cursor < 0 || m.cursor >= len(m.exercises) {
		return
	}
	id := m.exercises[m.cursor].ID
	if m.state.LastExerciseID == id {
		return
	}
	m.state.LastExerciseID = id
	if err := m.state.Save(m.paths.StateFile); err != nil {
		m.status = m.uif("status_state_save_failed", "Progress could not be saved: %v", err)
	}
}

// quit stops the program, cancelling a running check first so its process
// group does not outlive the TUI.
func (m *model) quit() tea.Cmd {
	if m.loading && m.checkCancel != nil {
		cancel := m.checkCancel
		done := m.checkDone
		m.checkCancel = nil
		m.checkDone = nil
		return cancelCheckAndQuit(cancel, done)
	}
	return tea.Quit
}

// resetProgress clears completion, returns to the first exercise, and shows
// its instructions. Reset is refused while a check runs, so no late result
// can mark an exercise complete again.
func (m *model) resetProgress() tea.Cmd {
	for i := range m.exercises {
		m.exercises[i].Completed = false
	}
	m.cursor = 0
	m.checkOutputs = nil
	m.loadReadmeForCurrent()

	m.status = m.ui("status_reset_done", "Progress reset.")
	if m.state != nil {
		m.state.Completed = make(map[string]bool)
		m.state.LastExerciseID = ""
		if err := m.state.Save(m.paths.StateFile); err != nil {
			m.status = m.uif("status_state_save_failed", "Progress could not be saved: %v", err)
		}
	}
	return m.resetMarquee()
}

/* ---------- Bubble Tea Update ---------- */

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.update(msg)
	updated := next.(model)
	updated.fitReadme()
	return updated, cmd
}

// followsEnd reports whether the right panel shows the end of its text in a
// way worth keeping when the panel changes size: check output opened at its
// end, or any text the reader scrolled to the bottom.
func (m model) followsEnd() bool {
	return m.readme.AtBottom() && (m.readme.YOffset > 0 || m.panel == panelCheckOutput)
}

// fitReadme gives the right panel the height the status and key help leave
// for it; a long status message takes rows from the instructions.
func (m *model) fitReadme() {
	if !m.ready {
		return
	}
	height := m.layout().readmeHeight
	if m.readme.Height == height {
		return
	}
	atEnd := m.followsEnd()
	m.readme.Height = height
	if atEnd {
		m.readme.GotoBottom()
	} else {
		m.readme.SetYOffset(m.readme.YOffset)
	}
}

func (m model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		panel := m.panel
		// Re-rendering at the new width changes the line count, so remember
		// where the reader is and return there afterwards.
		atEnd := m.followsEnd()
		offset, lines := m.readme.YOffset, m.readme.TotalLineCount()
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		layout := calculateLayout(m.width, m.height)
		m.listWidth = layout.listOuterWidth
		m.rightWidth = layout.rightOuterWidth
		m.readme.Width = layout.rightInnerWidth
		m.readme.Height = m.layout().readmeHeight

		m.loadReadmeForCurrent()
		switch {
		case panel == panelCheckOutput && m.selectedCheckOutput() != "":
			m.showCheckOutput()
		case panel == panelHint:
			m.showHint()
		}
		switch {
		case atEnd:
			m.readme.GotoBottom()
		case lines > 0:
			m.readme.SetYOffset(offset * m.readme.TotalLineCount() / lines)
		}
		return m, m.resetMarquee()

	case tea.KeyMsg:
		// Runes that arrive in one read come as one message, for example
		// "sss" when a key is held while the view redraws. Handle them as
		// separate presses; a paste stays one message.
		if msg.Type == tea.KeyRunes && len(msg.Runes) > 1 && !msg.Paste {
			return m.updateEachRune(msg)
		}
		key := msg.String()

		// ctrl+c her durumda çıkar, reset onayı beklenirken bile
		if key == "ctrl+c" {
			return m, m.quit()
		}

		// Reset onayı açıksa sadece y/n/esc handle et
		if m.confirmReset {
			switch key {
			case "y", "Y":
				m.confirmReset = false
				return m, m.resetProgress()

			case "n", "N", "esc":
				m.status = m.ui("status_reset_canceled", "Reset canceled.")
				m.confirmReset = false
				return m, nil
			}
			// Farklı tuşsa görmezden gel
			return m, nil
		}

		switch key {

		// Global çıkış
		case "q":
			return m, m.quit()

		// SOL PANEL (egzersiz listesi) – W/S ile hareket
		case "w":
			if m.cursor > 0 {
				m.cursor--
				m.loadReadmeForCurrent()
				m.persistSelectedExercise()
				return m, m.resetMarquee()
			}
		case "s":
			if m.cursor < len(m.exercises)-1 {
				m.cursor++
				m.loadReadmeForCurrent()
				m.persistSelectedExercise()
				return m, m.resetMarquee()
			}

		// SAĞ PANEL (instructions) – ok tuşları ve PgUp/PgDn ile scroll
		case "up":
			m.readme.LineUp(1)
		case "down":
			m.readme.LineDown(1)

		case "pgup":
			m.readme.LineUp(5)
		case "pgdown":
			m.readme.LineDown(5)

		// YATAY scroll: sağ/sol ok tuşları
		case "right":
			m.readme.ScrollRight(4)
		case "left":
			m.readme.ScrollLeft(4)

		case "tab":
			if m.panel != panelInstructions {
				m.panel = panelInstructions
				m.readme.SetContent(m.instructionsContent)
				m.readme.GotoTop()
				m.status = m.ui("status_showing_instructions", "Showing exercise instructions.")
			} else if m.selectedCheckOutput() != "" {
				m.showCheckOutput()
				m.status = m.ui("status_showing_check_output", "Showing last check output.")
			} else {
				m.status = m.ui("status_no_check_output", "No check output yet.")
			}

		// CHECK çalıştır
		case "c", "enter":
			if m.loading {
				m.status = m.ui("status_check_already_running", "A check is already running; waiting for its result.")
				return m, nil
			}
			if len(m.exercises) == 0 {
				m.status = m.ui("status_no_exercises", "No exercises found.")
				return m, nil
			}

			idx := m.cursor
			ex := m.exercises[idx]
			m.loading = true
			ctx, cancel := context.WithTimeout(context.Background(), defaultCheckTimeout)
			done := make(chan struct{})
			m.checkCancel = cancel
			m.checkDone = done

			name := ex.DefaultName
			if m.i18n != nil {
				name = m.i18n.ExerciseName(m.currentLangCode(), ex.ID, ex.DefaultName)
			}

			m.status = fmt.Sprintf("\"%s\"...", name)
			return m, runCheck(ctx, idx, ex, done)

		// Hint: sağ panelde gösterilir, çünkü durum satırına sığmaz
		case "h":
			if m.panel == panelHint {
				m.panel = panelInstructions
				m.readme.SetContent(m.instructionsContent)
				m.readme.GotoTop()
				m.status = m.ui("status_showing_instructions", "Showing exercise instructions.")
			} else {
				m.showHint()
				m.status = m.ui("status_showing_hint", "Showing the hint. Press h or Tab to return to the instructions.")
			}

		// Dil değiştir
		case "l", "L":
			if m.i18n != nil && len(m.i18n.Langs()) > 0 {
				m.langIndex = m.i18n.NextLangIndex(m.langIndex)
				m.status = m.ui("status_lang_switched", "Language switched.")
				m.loadReadmeForCurrent()

				// Dil tercihini state'e yaz
				if m.state != nil {
					m.state.LangCode = m.currentLangCode()
					if err := m.state.Save(m.paths.StateFile); err != nil {
						m.status = m.uif("status_state_save_failed", "Progress could not be saved: %v", err)
					}
				}
				return m, m.resetMarquee()
			}

		// Progress reset
		case "r", "R":
			if m.loading {
				m.status = m.ui("status_reset_blocked", "Wait for the running check to finish before resetting progress.")
				return m, nil
			}
			m.confirmReset = true
			m.status = m.ui("status_reset_confirm", "Reset all progress? (y/n)")
		}

		// KeyMsg case’inin sonunda mutlaka return et
		return m, nil

	case checkResultMsg:
		m.loading = false
		if m.checkCancel != nil {
			m.checkCancel()
		}
		m.checkCancel = nil
		m.checkDone = nil

		// Kaçış dizilerini ve kontrol karakterlerini temizle
		out := sanitizeCheckOutput(msg.output)
		if msg.err != nil && out == "" {
			out = "[FAIL] " + msg.err.Error()
		}

		// Egzersiz state’ini güncelle
		if msg.index >= 0 && msg.index < len(m.exercises) {
			ex := m.exercises[msg.index]

			success := (msg.err == nil)
			m.exercises[msg.index].Completed = success

			if m.state != nil {
				if m.state.Completed == nil {
					m.state.Completed = make(map[string]bool)
				}
				m.state.Completed[ex.ID] = success
				if err := m.state.Save(m.paths.StateFile); err != nil {
					if out != "" {
						out = out + fmt.Sprintf("\n[state save error: %v]", err)
					} else {
						out = fmt.Sprintf("[state save error: %v]", err)
					}
				}
			}
		}

		// Status satırı için kısa özet
		summary := lastLine(out)
		if summary == "" && msg.err != nil {
			summary = "[FAIL] check.sh failed"
		} else if summary == "" {
			summary = "[PASS] check.sh OK"
		}
		m.status = summary

		// Çıktıyı egzersize bağla; panel sadece sonuç hâlâ seçili egzersize
		// aitse değişir, yoksa kullanıcının baktığı içerik yerinde kalır.
		if msg.index >= 0 && msg.index < len(m.exercises) {
			id := m.exercises[msg.index].ID
			if out != "" {
				if m.checkOutputs == nil {
					m.checkOutputs = make(map[string]string)
				}
				m.checkOutputs[id] = out
			}
			if msg.index == m.cursor {
				if out != "" {
					m.showCheckOutput()
				}
			} else {
				m.status = id + ": " + summary
			}
		}

		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case marqueeTickMsg:
		if msg.generation != m.marqueeGeneration {
			return m, nil
		}
		if !m.selectedNameOverflows() {
			m.marquee = 0
			m.marqueeGeneration++
			return m, nil
		}
		m.marquee++
		return m, marqueeTick(
			m.marqueeGeneration,
			marqueeTickDelay(m.marquee, m.selectedMarqueeCycleLength()),
		)
	}

	return m, nil
}

// updateEachRune applies a multi-rune key message one rune at a time and
// batches the commands the single presses return.
func (m model) updateEachRune(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var current tea.Model = m
	cmds := make([]tea.Cmd, 0, len(msg.Runes))
	for _, r := range msg.Runes {
		var cmd tea.Cmd
		current, cmd = current.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}, Alt: msg.Alt})
		cmds = append(cmds, cmd)
	}
	return current, tea.Batch(cmds...)
}
