package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type layoutInfo struct {
	listOuterWidth   int
	rightOuterWidth  int
	listInnerWidth   int
	rightInnerWidth  int
	listOuterHeight  int
	rightOuterHeight int
	listRows         int
	readmeHeight     int
	compact          bool
}

func calculateLayout(width, height int) layoutInfo {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	var listOuter, rightOuter int
	if width < 80 {
		listOuter = width
		rightOuter = width
	} else {
		listOuter = width * 50 / 100
		if listOuter < 48 {
			listOuter = 48
		}
		if listOuter > 60 {
			listOuter = 60
		}
		rightOuter = width - listOuter
		if rightOuter < 34 {
			rightOuter = 34
			listOuter = width - rightOuter
			if listOuter < 20 {
				listOuter = 20
			}
		}
	}

	listInner := maxInt(1, listOuter-4)
	rightInner := maxInt(1, rightOuter-4)
	compact := height < 9 || width < 80 && height < 14
	panelBudget := maxInt(1, height-4)
	listOuterHeight := panelBudget
	rightOuterHeight := panelBudget
	if width < 80 && !compact {
		listOuterHeight = maxInt(5, panelBudget*2/5)
		if listOuterHeight > panelBudget-5 {
			listOuterHeight = panelBudget - 5
		}
		rightOuterHeight = panelBudget - listOuterHeight
	}
	listRows := maxInt(1, listOuterHeight-4)
	readmeHeight := maxInt(1, rightOuterHeight-4)

	return layoutInfo{
		listOuterWidth:   listOuter,
		rightOuterWidth:  rightOuter,
		listInnerWidth:   listInner,
		rightInnerWidth:  rightInner,
		listOuterHeight:  listOuterHeight,
		rightOuterHeight: rightOuterHeight,
		listRows:         listRows,
		readmeHeight:     readmeHeight,
		compact:          compact,
	}
}

func visibleExerciseRange(total, cursor, limit int) (int, int) {
	if total <= 0 || limit <= 0 {
		return 0, 0
	}
	if limit >= total {
		return 0, total
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= total {
		cursor = total - 1
	}

	start := cursor - limit/2
	if start < 0 {
		start = 0
	}
	if start+limit > total {
		start = total - limit
	}
	return start, start + limit
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func truncateCells(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}

	var b strings.Builder
	for _, r := range s {
		next := b.String() + string(r)
		if lipgloss.Width(next)+3 > width {
			break
		}
		b.WriteRune(r)
	}
	return b.String() + "..."
}

func clipCells(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}

	var b strings.Builder
	for _, r := range s {
		next := b.String() + string(r)
		if lipgloss.Width(next) > width {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}

func marqueeText(s string, width, offset int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}

	runes := []rune(s + "   ")
	if len(runes) == 0 {
		return ""
	}
	offset = offset % len(runes)
	scrolled := string(runes[offset:]) + string(runes[:offset])
	return clipCells(scrolled, width)
}

func (m model) layout() layoutInfo {
	layout := calculateLayout(m.width, m.height)
	if m.listWidth > 0 {
		layout.listOuterWidth = m.listWidth
		layout.listInnerWidth = maxInt(1, m.listWidth-4)
	}
	if m.rightWidth > 0 {
		layout.rightOuterWidth = m.rightWidth
		layout.rightInnerWidth = maxInt(1, m.rightWidth-4)
	}
	return layout
}

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

	layout := m.layout()
	langCode := m.currentLangCode()
	title := "Nginxlings"
	exTitle := "Exercises"
	instTitle := "Instructions"
	checkOutputTitle := "Last Check Output"
	readyLabel := "[READY]"
	workingLabel := "[RUNNING]" // sadece çeviri için
	helpText := "(W/S) select exercise  (↑/↓, PgUp/PgDn) scroll instructions  (←/→) horiz scroll  (c/Enter) check  (Tab) view  (h) hint  (l) lang  (r) reset  (q) quit"

	langLabel := "Lang"

	if m.i18n != nil {
		title = m.i18n.T(langCode, "title", title)
		exTitle = m.i18n.T(langCode, "exercises_title", exTitle)
		instTitle = m.i18n.T(langCode, "instructions_title", instTitle)
		checkOutputTitle = m.i18n.T(langCode, "check_output_title", checkOutputTitle)
		readyLabel = m.i18n.T(langCode, "ready_label", readyLabel)
		workingLabel = m.i18n.T(langCode, "working_label", workingLabel)
		helpText = m.i18n.T(langCode, "help", helpText)
		langLabel = m.i18n.T(langCode, "lang_label", langLabel)
		_ = workingLabel
	}
	if m.showingCheckOutput {
		instTitle = checkOutputTitle
	}

	if layout.compact {
		lines := make([]string, 0, 3)
		if m.height >= 3 {
			lines = append(lines, m.styles.Title.Render(truncateCells(title, m.width)))
		}
		if m.height >= 2 {
			selected := clipCells(exTitle, m.width)
			selectedStyle := m.styles.SelectedItem
			if len(m.exercises) > 0 && m.cursor >= 0 && m.cursor < len(m.exercises) {
				ex := m.exercises[m.cursor]
				name := ex.DefaultName
				if m.i18n != nil {
					name = m.i18n.ExerciseName(langCode, ex.ID, ex.DefaultName)
				}
				icon := "x"
				if ex.Completed {
					icon = "✓"
					selectedStyle = m.styles.SelectedCompletedItem
				}
				prefix := fmt.Sprintf("➤ %s ", icon)
				selected = prefix + marqueeText(name, maxInt(1, m.width-lipgloss.Width(prefix)), m.marquee)
			}
			lines = append(lines, selectedStyle.Render(selected))
		}

		statusText := m.status
		if m.loading {
			statusText = fmt.Sprintf("%s %s %s", m.spinner.View(), workingLabel, m.status)
		} else {
			statusText = fmt.Sprintf("%s %s", readyLabel, m.status)
		}
		completed, total := m.progressCounts()
		compactStatus := fmt.Sprintf("%d/%d %s %s: %s", completed, total, statusText, langLabel, m.currentLangLabel())
		lines = append(lines, m.styles.Help.Padding(0).Render(truncateCells(compactStatus, m.width)))
		return strings.Join(lines, "\n") + "\n"
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
		start, end := visibleExerciseRange(len(m.exercises), m.cursor, layout.listRows)
		for i := start; i < end; i++ {
			ex := m.exercises[i]
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
			nameWidth := maxInt(1, layout.listInnerWidth-8)
			if i == m.cursor {
				displayName = marqueeText(displayName, nameWidth, m.marquee)
			} else {
				displayName = clipCells(displayName, nameWidth)
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

	listBox := m.styles.Box.Width(layout.listInnerWidth).Height(maxInt(1, layout.listOuterHeight-2)).Render(strings.TrimRight(listBuilder.String(), "\n"))

	// Sağ panel
	var rightBuilder strings.Builder
	rightBuilder.WriteString(m.styles.Title.Width(layout.rightInnerWidth).Render(truncateCells(instTitle, layout.rightInnerWidth)) + "\n\n")
	rightBuilder.WriteString(m.readme.View())
	rightBox := m.styles.Box.Width(layout.rightInnerWidth).Height(maxInt(1, layout.rightOuterHeight-2)).Render(strings.TrimRight(rightBuilder.String(), "\n"))

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

	statusSegment := m.styles.StatusBar.Render(" " + truncateCells(statusText, maxInt(1, m.width-lipgloss.Width(progressSegment)-2)) + " ")
	langInfoText := fmt.Sprintf("%s: %s", langLabel, m.currentLangLabel())
	langInfo := m.styles.Help.Render(" " + truncateCells(langInfoText, maxInt(1, m.width-lipgloss.Width(progressSegment)-lipgloss.Width(statusSegment)-2)) + " ")
	usedWidth := lipgloss.Width(progressSegment) + lipgloss.Width(statusSegment) + lipgloss.Width(langInfo)
	helpSegment := ""
	if remaining := m.width - usedWidth; remaining > 4 {
		helpSegment = m.styles.Help.Render(truncateCells(helpText, remaining))
	}

	statusLine := lipgloss.JoinHorizontal(
		lipgloss.Top,
		progressSegment,
		statusSegment,
		langInfo,
		helpSegment,
	)
	if lipgloss.Width(statusLine) > m.width {
		plainStatus := strings.TrimSpace(fmt.Sprintf("%s %s %s %s", progressLabel, statusText, langInfoText, helpText))
		statusLine = m.styles.Help.Render(truncateCells(plainStatus, maxInt(1, m.width-2)))
	}

	return m.styles.Title.Render(title) +
		"\n\n" +
		topRow +
		"\n\n" +
		statusLine +
		"\n"
}
