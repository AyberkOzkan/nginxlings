package main

import (
	"context"
	"fmt"
	"strings"
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

func marqueeTick() tea.Cmd {
	return tea.Tick(350*time.Millisecond, func(time.Time) tea.Msg {
		return marqueeTickMsg{}
	})
}

func (m model) selectedNameOverflows() bool {
	if len(m.exercises) == 0 || m.cursor < 0 || m.cursor >= len(m.exercises) {
		return false
	}
	layout := m.layout()
	nameWidth := maxInt(1, layout.listInnerWidth-8)
	name := m.exercises[m.cursor].DefaultName
	if m.i18n != nil {
		name = m.i18n.ExerciseName(m.currentLangCode(), m.exercises[m.cursor].ID, name)
	}
	return lipgloss.Width(name) > nameWidth
}

/* ---------- Bubble Tea Update ---------- */

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		layout := calculateLayout(m.width, m.height)
		m.listWidth = layout.listOuterWidth
		m.rightWidth = layout.rightOuterWidth
		m.marquee = 0
		m.readme.Width = layout.rightInnerWidth
		m.readme.Height = layout.readmeHeight

		m.loadReadmeForCurrent()
		if m.selectedNameOverflows() {
			return m, marqueeTick()
		}
		return m, nil

	case tea.KeyMsg:
		key := msg.String()

		// Reset onayı açıksa sadece y/n/esc handle et
		if m.confirmReset {
			switch key {
			case "y", "Y":
				// Tüm progress'i sıfırla
				for i := range m.exercises {
					m.exercises[i].Completed = false
				}
				if m.state != nil {
					m.state.Completed = make(map[string]bool)
					m.state.LastExerciseID = ""
					if err := m.state.Save(m.paths.StateFile); err != nil {
						m.status = fmt.Sprintf("Reset kaydedilemedi: %v", err)
					} else {
						m.status = "İlerleme sıfırlandı."
					}
				} else {
					m.status = "İlerleme sıfırlandı."
				}
				m.cursor = 0
				m.confirmReset = false
				return m, nil

			case "n", "N", "esc":
				m.status = "Reset iptal edildi."
				m.confirmReset = false
				return m, nil
			}
			// Farklı tuşsa görmezden gel
			return m, nil
		}

		switch key {

		// Global çıkış
		case "ctrl+c", "q":
			if m.loading && m.checkCancel != nil {
				cancel := m.checkCancel
				done := m.checkDone
				m.checkCancel = nil
				m.checkDone = nil
				return m, cancelCheckAndQuit(cancel, done)
			}
			return m, tea.Quit

		// SOL PANEL (egzersiz listesi) – W/S ile hareket
		case "w":
			if m.cursor > 0 {
				m.cursor--
				m.marquee = 0
				m.loadReadmeForCurrent()
				if m.selectedNameOverflows() {
					return m, marqueeTick()
				}
			}
		case "s":
			if m.cursor < len(m.exercises)-1 {
				m.cursor++
				m.marquee = 0
				m.loadReadmeForCurrent()
				if m.selectedNameOverflows() {
					return m, marqueeTick()
				}
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

		// CHECK çalıştır
		case "c", "enter":
			if m.loading {
				m.status = "Check zaten çalışıyor; mevcut sonuç bekleniyor."
				return m, nil
			}
			if len(m.exercises) == 0 {
				if m.i18n != nil {
					m.status = m.i18n.T(
						m.currentLangCode(),
						"status_no_exercises",
						"No exercises.",
					)
				} else {
					m.status = "No exercises."
				}
				return m, nil
			}

			idx := m.cursor
			ex := m.exercises[idx]
			m.loading = true
			ctx, cancel := context.WithTimeout(context.Background(), defaultCheckTimeout)
			done := make(chan struct{})
			m.checkCancel = cancel
			m.checkDone = done

			workingLabel := "[RUNNING]"
			if m.i18n != nil {
				workingLabel = m.i18n.T(
					m.currentLangCode(),
					"working_label",
					workingLabel,
				)
			}

			name := ex.DefaultName
			if m.i18n != nil {
				name = m.i18n.ExerciseName(m.currentLangCode(), ex.ID, ex.DefaultName)
			}

			m.status = fmt.Sprintf("%s \"%s\"...", workingLabel, name)
			return m, runCheck(ctx, idx, ex, done)

		// Hint
		case "h":
			if m.i18n != nil {
				m.status = m.i18n.T(
					m.currentLangCode(),
					"hint_default",
					"Hint: Read the README carefully; fix the TODOs in nginx.conf.",
				)
			} else {
				m.status = "Hint: Read the README carefully; fix the TODOs in nginx.conf."
			}

		// Dil değiştir
		case "l", "L":
			if m.i18n != nil && len(m.i18n.Langs()) > 0 {
				m.langIndex = m.i18n.NextLangIndex(m.langIndex)
				m.status = m.i18n.T(
					m.currentLangCode(),
					"status_lang_switched",
					"Language changed.",
				)
				m.loadReadmeForCurrent()

				// Dil tercihini state'e yaz
				if m.state != nil {
					m.state.LangCode = m.currentLangCode()
					_ = m.state.Save(m.paths.StateFile)
				}
			}

		// Progress reset
		case "r", "R":
			m.confirmReset = true
			m.status = "Tüm ilerlemeyi sıfırlamak istiyor musun? (y/n)"
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

		// Tüm çıktıyı trim’le
		out := strings.TrimSpace(msg.output)
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
				m.state.LastExerciseID = ex.ID
				if err := m.state.Save(m.paths.StateFile); err != nil {
					if out != "" {
						out = out + fmt.Sprintf("\n[state save error: %v]", err)
					} else {
						out = fmt.Sprintf("[state save error: %v]", err)
					}
				}
			}
		}

		// ✅ / ❌ her iki durumda da: check.sh çıktısını sağ panelde göster
		if out != "" {
			m.lastCheckOutput = out
			m.readme.SetContent(out)
		}

		// Status satırı için kısa özet
		if msg.err != nil {
			failLine := lastLine(out)
			if failLine == "" {
				failLine = "[FAIL] check.sh failed"
			}
			m.status = failLine
		} else {
			passLine := lastLine(out)
			if passLine == "" {
				passLine = "[PASS] check.sh OK"
			}
			m.status = passLine
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
		if !m.selectedNameOverflows() {
			m.marquee = 0
			return m, nil
		}
		m.marquee++
		return m, marqueeTick()
	}

	return m, nil
}
