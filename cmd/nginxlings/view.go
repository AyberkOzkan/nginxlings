package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var (
	mdMu            sync.Mutex
	mdRenderer      *glamour.TermRenderer
	mdRendererWidth int
)

/* ---------- Markdown rendering ---------- */

func (m *model) renderMarkdown(raw string) string {
	width := m.readme.Width
	if width <= 0 {
		width = 80
	}

	mdMu.Lock()
	defer mdMu.Unlock()

	// renderer sadece width değiştiğinde yeniden oluşturulsun
	if mdRenderer == nil || mdRendererWidth != width {
		r, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(width),
		)
		if err == nil {
			mdRenderer = r
			mdRendererWidth = width
		} else {
			// renderer oluşturulamazsa raw döneriz
			return raw
		}
	}

	out, err := mdRenderer.Render(raw)
	if err != nil {
		return raw
	}
	return strings.TrimRight(out, "\n")
}

/* ---------- Bubble Tea View ---------- */

func (m model) View() string {
	if !m.ready {
		return "Nginxlings yükleniyor...\n"
	}

	langCode := m.currentLangCode()
	title := "Nginxlings"
	exTitle := "Exercises"
	instTitle := "Instructions"
	readyLabel := "[READY]"
	workingLabel := "[RUNNING]" // sadece çeviri için
	helpText := "(W/S) select exercise  (↑/↓, PgUp/PgDn) scroll instructions  (←/→) horiz scroll  (c/Enter) check  (h) hint  (l) lang  (r) reset  (q) quit"

	langLabel := "Lang"

	if m.i18n != nil {
		title = m.i18n.T(langCode, "title", title)
		exTitle = m.i18n.T(langCode, "exercises_title", exTitle)
		instTitle = m.i18n.T(langCode, "instructions_title", instTitle)
		readyLabel = m.i18n.T(langCode, "ready_label", readyLabel)
		workingLabel = m.i18n.T(langCode, "working_label", workingLabel)
		helpText = m.i18n.T(langCode, "help", helpText)
		langLabel = m.i18n.T(langCode, "lang_label", langLabel)
		_ = workingLabel
	}

	// Sol panel
	var listBuilder strings.Builder
	listBuilder.WriteString(m.styles.ListTitle.Render(exTitle) + "\n\n")

	if len(m.exercises) == 0 {
		if m.i18n != nil {
			listBuilder.WriteString("  (" + m.i18n.T(langCode, "status_no_exercises", "Hiç egzersiz yok.") + ")\n")
		} else {
			listBuilder.WriteString("  (Hiç egzersiz yok)\n")
		}
	} else {
		for i, ex := range m.exercises {
			arrow := " "
			if i == m.cursor {
				arrow = "➤"
			}

			icon := "x"
			if ex.Completed {
				icon = "✓"
			}

			displayName := ex.DefaultName
			if m.i18n != nil {
				displayName = m.i18n.ExerciseName(langCode, ex.ID, ex.DefaultName)
			}

			name := displayName
			if ex.Completed {
				name = m.styles.CompletedItem.Render(displayName)
			} else {
				name = m.styles.Item.Render(displayName)
			}

			if i == m.cursor {
				if ex.Completed {
					name = m.styles.SelectedCompletedItem.Render(displayName)
				} else {
					name = m.styles.SelectedItem.Render(displayName)
				}
			}

			line := fmt.Sprintf(" %s %s %s", arrow, icon, name)
			listBuilder.WriteString(line + "\n")
		}
	}

	listBox := m.styles.Box.Render(listBuilder.String())

	// Sağ panel
	var rightBuilder strings.Builder
	rightBuilder.WriteString(m.styles.Title.Render(instTitle) + "\n\n")
	rightBuilder.WriteString(m.readme.View())
	rightBox := m.styles.Box.Render(rightBuilder.String())

	// Responsive yerleşim
	var topRow string
	if m.width < 80 {
		// Dar ekran: liste üstte, instructions altta
		topRow = lipgloss.JoinVertical(
			lipgloss.Left,
			listBox,
			rightBox,
		)
	} else {
		// Geniş ekran: yan yana
		topRow = lipgloss.JoinHorizontal(
			lipgloss.Top,
			listBox,
			rightBox,
		)
	}

	// Progress info
	completed, total := m.progressCounts()
	progressLabel := fmt.Sprintf("%d/%d", completed, total)
	progressSegment := m.styles.Help.Render(" " + progressLabel + " ")

	// Status bar
	statusText := m.status
	if m.loading {
		statusText = fmt.Sprintf("%s %s %s", m.spinner.View(), workingLabel, m.status)
	} else {
		statusText = fmt.Sprintf("%s %s", readyLabel, m.status)
	}

	statusSegment := m.styles.StatusBar.Render(statusText)
	langInfo := m.styles.Help.Render(fmt.Sprintf(" %s: %s ", langLabel, m.currentLangLabel()))
	helpSegment := m.styles.Help.Render(helpText)

	statusLine := lipgloss.JoinHorizontal(
		lipgloss.Top,
		progressSegment,
		statusSegment,
		langInfo,
		helpSegment,
	)

	return m.styles.Title.Render(title) +
		"\n\n" +
		topRow +
		"\n\n" +
		statusLine +
		"\n"
}
