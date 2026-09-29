package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
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

const (
	// panelFrameWidth is the border plus horizontal padding of Styles.Box;
	// inner widths are the text area left inside a panel.
	panelFrameWidth = 4
	// listRowPrefixWidth covers the " ➤ x " drawn before each exercise name.
	listRowPrefixWidth = 5
	// Side-by-side panels need at least 80 columns. The list takes about half
	// of the width but never starves the instructions panel.
	sideBySideMinWidth = 80
	listMinOuterWidth  = 36
	listMaxOuterWidth  = 60
	rightMinOuterWidth = 40
	// minStatusWidth is the smallest status message room worth keeping the
	// progress and language segments beside it.
	minStatusWidth = 10
)

func calculateLayout(width, height int) layoutInfo {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	var listOuter, rightOuter int
	if width < sideBySideMinWidth {
		listOuter = width
		rightOuter = width
	} else {
		listOuter = width * 48 / 100
		if listOuter < listMinOuterWidth {
			listOuter = listMinOuterWidth
		}
		if listOuter > listMaxOuterWidth {
			listOuter = listMaxOuterWidth
		}
		rightOuter = width - listOuter
		if rightOuter < rightMinOuterWidth {
			rightOuter = rightMinOuterWidth
			listOuter = width - rightOuter
		}
	}

	listInner := maxInt(1, listOuter-panelFrameWidth)
	rightInner := maxInt(1, rightOuter-panelFrameWidth)
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

// helpSeparator separates the entries of the translated key help text.
const helpSeparator = " · "

// truncateSegments keeps the whole helpSeparator-separated entries that fit in
// width, so the key help never ends in the middle of a key description.
func truncateSegments(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	parts := strings.Split(s, helpSeparator)
	out := parts[0]
	if lipgloss.Width(out) > width {
		return truncateCells(out, width)
	}
	for _, part := range parts[1:] {
		next := out + helpSeparator + part
		if lipgloss.Width(next) > width {
			break
		}
		out = next
	}
	return out
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
		layout.listInnerWidth = maxInt(1, m.listWidth-panelFrameWidth)
	}
	if m.rightWidth > 0 {
		layout.rightOuterWidth = m.rightWidth
		layout.rightInnerWidth = maxInt(1, m.rightWidth-panelFrameWidth)
	}
	return layout
}

// listNameWidth is the room left for an exercise name after the row prefix.
func (l layoutInfo) listNameWidth() int {
	return maxInt(1, l.listInnerWidth-listRowPrefixWidth)
}

// renderPanel draws content in a bordered box whose total size is exactly the
// outer size budgeted by calculateLayout. lipgloss Width/Height include the
// padding but not the border, so only the border is subtracted here.
func (m model) renderPanel(content string, outerWidth, outerHeight int) string {
	box := m.styles.Box
	return box.
		Width(maxInt(1, outerWidth-box.GetHorizontalBorderSize())).
		Height(maxInt(1, outerHeight-box.GetVerticalBorderSize())).
		Render(content)
}

var (
	mdMu            sync.Mutex
	mdRenderer      *glamour.TermRenderer
	mdRendererWidth int
	mdRendererLight bool
)

/* ---------- Markdown rendering ---------- */

// markdownStyle returns glamour's dark or light theme without the document
// margin and leading blank line: the panel border, padding, and title already
// frame the text, so they would only narrow the instructions.
func markdownStyle(light bool) ansi.StyleConfig {
	style := styles.DarkStyleConfig
	if light {
		style = styles.LightStyleConfig
	}
	var margin uint
	style.Document.Margin = &margin
	style.Document.BlockPrefix = ""
	return style
}

func (m *model) renderMarkdown(raw string) string {
	width := m.readme.Width
	if width <= 0 {
		width = 80
	}

	mdMu.Lock()
	defer mdMu.Unlock()

	// renderer sadece width veya tema değiştiğinde yeniden oluşturulsun
	if mdRenderer == nil || mdRendererWidth != width || mdRendererLight != m.lightBackground {
		r, err := glamour.NewTermRenderer(
			glamour.WithStyles(markdownStyle(m.lightBackground)),
			glamour.WithWordWrap(width),
		)
		if err == nil {
			mdRenderer = r
			mdRendererWidth = width
			mdRendererLight = m.lightBackground
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
		return m.ui("status_loading", "Loading Nginxlings...")
	}

	layout := m.layout()
	langCode := m.currentLangCode()
	title := m.ui("title", "Nginxlings")
	exTitle := m.ui("exercises_title", "Exercises")
	instTitle := m.ui("instructions_title", "Instructions")
	if m.panel == panelCheckOutput {
		instTitle = m.ui("check_output_title", "Last Check Output")
	}
	readyLabel := m.ui("ready_label", "[READY]")
	workingLabel := m.ui("working_label", "[RUNNING]")
	helpText := m.ui("help", "W/S select · c/Enter check · Tab view · h hint · l language · r reset · q quit · ↑/↓ PgUp/PgDn ←/→ scroll")
	langLabel := m.ui("lang_label", "Lang")

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
		// No trailing newline: the renderer splits on "\n" and would drop the
		// first row to fit the extra empty one.
		return strings.Join(lines, "\n")
	}

	// Sol panel
	var listBuilder strings.Builder
	listTitleWidth := maxInt(1, layout.listInnerWidth-m.styles.ListTitle.GetHorizontalPadding())
	listBuilder.WriteString(m.styles.ListTitle.Render(truncateCells(exTitle, listTitleWidth)) + "\n\n")

	if len(m.exercises) == 0 {
		listBuilder.WriteString("  (" + m.ui("status_no_exercises", "No exercises found.") + ")\n")
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
			nameWidth := layout.listNameWidth()
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

	listBox := m.renderPanel(strings.TrimRight(listBuilder.String(), "\n"), layout.listOuterWidth, layout.listOuterHeight)

	// Sağ panel
	var rightBuilder strings.Builder
	rightBuilder.WriteString(m.styles.Title.Width(layout.rightInnerWidth).Render(truncateCells(instTitle, layout.rightInnerWidth)) + "\n\n")
	rightBuilder.WriteString(m.readme.View())
	rightBox := m.renderPanel(strings.TrimRight(rightBuilder.String(), "\n"), layout.rightOuterWidth, layout.rightOuterHeight)

	// Responsive yerleşim
	var topRow string
	if m.width < sideBySideMinWidth {
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

	// Status line: progress and language have fixed widths, the status
	// message takes the rest of the row.
	completed, total := m.progressCounts()
	progressLabel := fmt.Sprintf("%d/%d", completed, total)
	progressSegment := m.styles.Help.Render(progressLabel)
	langSegment := m.styles.Help.Render(fmt.Sprintf("%s: %s", langLabel, m.currentLangLabel()))

	statusText := fmt.Sprintf("%s %s", readyLabel, m.status)
	if m.loading {
		statusText = fmt.Sprintf("%s %s %s", m.spinner.View(), workingLabel, m.status)
	}

	var statusLine string
	statusRoom := m.width - lipgloss.Width(progressSegment) - lipgloss.Width(langSegment) - m.styles.StatusBar.GetHorizontalPadding()
	if statusRoom >= minStatusWidth {
		statusSegment := m.styles.StatusBar.Render(truncateCells(statusText, statusRoom))
		statusLine = lipgloss.JoinHorizontal(lipgloss.Top, progressSegment, langSegment, statusSegment)
	} else {
		plainStatus := fmt.Sprintf("%s %s", progressLabel, statusText)
		statusLine = m.styles.Help.Render(truncateCells(plainStatus, maxInt(1, m.width-m.styles.Help.GetHorizontalPadding())))
	}

	// Key help gets its own row so the status message never hides it.
	helpLine := m.styles.Help.Render(truncateSegments(helpText, maxInt(1, m.width-m.styles.Help.GetHorizontalPadding())))

	// No trailing newline: the renderer splits on "\n" and would drop the
	// title row to fit the extra empty one.
	return m.styles.Title.Render(truncateCells(title, m.width)) +
		"\n\n" +
		topRow +
		"\n" +
		statusLine +
		"\n" +
		helpLine
}
