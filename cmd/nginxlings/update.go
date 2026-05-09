package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

/* ---------- check.sh  ---------- */

func runCheck(index int, ex Exercise) tea.Cmd {
	return func() tea.Msg {
		out, err := runExerciseCheck(ex)

		return checkResultMsg{
			output: out,
			err:    err,
			index:  index,
		}
	}
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

		var listWidth, rightWidth int

		if m.width < 80 {
			// Küçük terminal: iki panel de tam genişlikte
			listWidth = m.width - 4
			rightWidth = m.width - 4
		} else {
			// Normal/geniş terminal: solda ~%30 liste, sağda instructions
			listWidth = int(float32(m.width) * 0.30)
			if listWidth < 24 {
				listWidth = 24
			}
			rightWidth = m.width - listWidth - 4
			if rightWidth < 30 {
				rightWidth = 30
			}
		}

		contentHeight := m.height - 5
		if contentHeight < 8 {
			contentHeight = 8
		}

		m.readme.Width = rightWidth - 4
		m.readme.Height = contentHeight - 2

		m.loadReadmeForCurrent()
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
			return m, tea.Quit

		// SOL PANEL (egzersiz listesi) – W/S ile hareket
		case "w":
			if m.cursor > 0 {
				m.cursor--
				m.loadReadmeForCurrent()
			}
		case "s":
			if m.cursor < len(m.exercises)-1 {
				m.cursor++
				m.loadReadmeForCurrent()
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
			return m, runCheck(idx, ex)

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
	}

	return m, nil
}
