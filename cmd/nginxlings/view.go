package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
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
	// The status message and the key help wrap onto extra rows when one row
	// is too narrow for them. The extra rows come out of the instructions,
	// which keep at least minReadmeRows rows.
	maxStatusRows = 4
	maxHelpRows   = 3
	minReadmeRows = 6
)

// isCompactSize reports whether a terminal is too small for the panels; it
// then shows the title, the selected exercise and one status row.
func isCompactSize(width, height int) bool {
	return height < 9 || width < 80 && height < 14
}

// calculateLayout is the layout with one status row and one key help row.
func calculateLayout(width, height int) layoutInfo {
	return calculateLayoutWithFooter(width, height, 2)
}

// calculateLayoutWithFooter sizes the panels for a terminal whose bottom
// footerRows rows hold the status and the key help. Rows beyond one status
// row and one key help row come out of the instructions panel only.
func calculateLayoutWithFooter(width, height, footerRows int) layoutInfo {
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
	compact := isCompactSize(width, height)
	// The title row and the gap under it come before the panels.
	baseBudget := maxInt(1, height-4)
	panelBudget := maxInt(1, height-2-footerRows)
	listOuterHeight := panelBudget
	rightOuterHeight := panelBudget
	if width < 80 && !compact {
		listOuterHeight = maxInt(5, baseBudget*2/5)
		if listOuterHeight > baseBudget-5 {
			listOuterHeight = baseBudget - 5
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

// listWindow returns the exercises [start, end) drawn in a list of rows rows.
// When exercises are hidden, markers say how many: the one for hidden rows
// above uses the gap under the list title, the one for hidden rows below
// takes the last row. The selected exercise always stays visible.
func listWindow(total, cursor, rows int) (start, end int) {
	if total <= rows || rows < 2 {
		return visibleExerciseRange(total, cursor, rows)
	}
	start, end = visibleExerciseRange(total, cursor, rows-1)
	if end == total {
		// Nothing is hidden below, so the last row can show an exercise.
		return maxInt(0, total-rows), total
	}
	return start, end
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ellipsis marks text cut to fit; it takes one cell.
const ellipsis = "…"

func truncateCells(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	return strings.TrimRight(clipCells(s, width-1), " ") + ellipsis
}

// wrapText word-wraps s into at most maxRows rows no wider than width, with
// the first row indent cells narrower. Text that still does not fit ends in
// an ellipsis on the last row.
func wrapText(s string, width, indent, maxRows int) []string {
	if width <= 0 || maxRows <= 0 {
		return nil
	}
	indent = min(maxInt(0, indent), width-1)
	s = strings.ReplaceAll(s, "\n", " ")
	// A placeholder word in front makes the first row wrap indent cells early.
	placeholder := strings.Repeat("x", indent)
	if indent > 0 {
		s = placeholder + " " + s
	}
	rows := strings.Split(xansi.Wrap(s, width, ""), "\n")
	rows[0] = strings.TrimPrefix(rows[0], placeholder)
	for i := range rows {
		rows[i] = strings.TrimSpace(rows[i])
	}
	if len(rows) > maxRows {
		lastWidth := width
		if maxRows == 1 {
			lastWidth = width - indent
		}
		rest := strings.Join(rows[maxRows-1:], " ")
		rows = append(rows[:maxRows-1], truncateCells(rest, lastWidth))
	}
	return rows
}

// helpSeparator separates the entries of the translated key help text.
const helpSeparator = " · "

// wrapSegments packs the helpSeparator-separated entries of s into at most
// maxRows rows no wider than width. An entry is never split across rows, so
// the key help never ends in the middle of a key description; entries that
// do not fit in maxRows rows are left out.
func wrapSegments(s string, width, maxRows int) []string {
	maxRows = maxInt(1, maxRows)
	rows := make([]string, 0, maxRows)
	current := ""
	for _, part := range strings.Split(s, helpSeparator) {
		if current != "" && lipgloss.Width(current+helpSeparator+part) <= width {
			current += helpSeparator + part
			continue
		}
		if current != "" {
			rows = append(rows, truncateCells(current, width))
			if len(rows) == maxRows {
				return rows
			}
		}
		current = part
	}
	return append(rows, truncateCells(current, width))
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

// layout sizes the panels around the rows the current status and key help
// need, so View and the scroll keys agree on the instructions height.
func (m model) layout() layoutInfo {
	status, help := m.footer()
	return m.layoutWithFooter(len(status) + len(help))
}

func (m model) layoutWithFooter(footerRows int) layoutInfo {
	layout := calculateLayoutWithFooter(m.width, m.height, footerRows)
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

/* ---------- Status and key help ---------- */

func (m model) helpText() string {
	return m.ui("help", "W/S select · c/Enter check · Tab view · h hint · l language · r reset · q quit · ↑/↓ PgUp/PgDn ←/→ scroll")
}

// statusText is the status message behind the ready or running label.
func (m model) statusText() string {
	if m.loading {
		return fmt.Sprintf("%s %s %s", m.spinner.View(), m.ui("working_label", "[RUNNING]"), m.status)
	}
	return fmt.Sprintf("%s %s", m.ui("ready_label", "[READY]"), m.status)
}

// footer returns the rendered status rows and key help rows drawn under the
// panels. Each wraps onto extra rows instead of being cut, as long as the
// instructions keep minReadmeRows rows. When rows are short the status
// message gets them first: it carries the check result, while the key help
// stays the same all session. Compact terminals have no footer.
func (m model) footer() (status, help []string) {
	if isCompactSize(m.width, m.height) {
		return nil, nil
	}
	extra := maxInt(0, calculateLayout(m.width, m.height).readmeHeight-minReadmeRows)
	status = m.statusRows(1 + min(maxStatusRows-1, extra))
	extra -= len(status) - 1

	helpWidth := maxInt(1, m.width-m.styles.Help.GetHorizontalPadding())
	for _, row := range wrapSegments(m.helpText(), helpWidth, 1+min(maxHelpRows-1, extra)) {
		help = append(help, m.styles.Help.Render(row))
	}
	return status, help
}

// statusRows renders the status in at most maxRows rows. Progress and
// language have fixed widths at the start of the first row; the status
// message fills the rest of it and wraps onto rows of the full width.
func (m model) statusRows(maxRows int) []string {
	completed, total := m.progressCounts()
	progressLabel := fmt.Sprintf("%d/%d", completed, total)
	progressSegment := m.styles.Help.Render(progressLabel)
	langSegment := m.styles.Help.Render(fmt.Sprintf("%s: %s", m.ui("lang_label", "Lang"), m.currentLangLabel()))
	statusText := m.statusText()

	var rows []string
	segmentsWidth := lipgloss.Width(progressSegment) + lipgloss.Width(langSegment)
	textWidth := maxInt(1, m.width-m.styles.StatusBar.GetHorizontalPadding())
	if textWidth-segmentsWidth < minStatusWidth {
		plainStatus := fmt.Sprintf("%s %s", progressLabel, statusText)
		for _, row := range wrapText(plainStatus, textWidth, 0, maxRows) {
			rows = append(rows, m.styles.Help.Render(row))
		}
		return rows
	}
	for i, row := range wrapText(statusText, textWidth, segmentsWidth, maxRows) {
		segment := m.styles.StatusBar.Render(row)
		if i == 0 {
			segment = lipgloss.JoinHorizontal(lipgloss.Top, progressSegment, langSegment, segment)
		}
		rows = append(rows, segment)
	}
	return rows
}

/* ---------- Bubble Tea View ---------- */

func (m model) View() string {
	if !m.ready {
		return m.ui("status_loading", "Loading Nginxlings...")
	}

	statusRows, helpRows := m.footer()
	layout := m.layoutWithFooter(len(statusRows) + len(helpRows))
	langCode := m.currentLangCode()
	title := m.ui("title", "Nginxlings")
	exTitle := m.ui("exercises_title", "Exercises")
	instTitle := m.ui("instructions_title", "Instructions")
	switch m.panel {
	case panelCheckOutput:
		instTitle = m.ui("check_output_title", "Last Check Output")
	case panelHint:
		instTitle = m.ui("hint_title", "Hint")
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

		completed, total := m.progressCounts()
		compactStatus := fmt.Sprintf("%d/%d %s %s: %s", completed, total, m.statusText(), m.ui("lang_label", "Lang"), m.currentLangLabel())
		lines = append(lines, m.styles.Help.Padding(0).Render(truncateCells(compactStatus, m.width)))
		// No trailing newline: the renderer splits on "\n" and would drop the
		// first row to fit the extra empty one.
		return strings.Join(lines, "\n")
	}

	// Sol panel
	var listBuilder strings.Builder
	listTitleWidth := maxInt(1, layout.listInnerWidth-m.styles.ListTitle.GetHorizontalPadding())
	listBuilder.WriteString(m.styles.ListTitle.Render(truncateCells(exTitle, listTitleWidth)) + "\n")

	if len(m.exercises) == 0 {
		listBuilder.WriteString("\n  (" + m.ui("status_no_exercises", "No exercises found.") + ")\n")
	} else {
		start, end := listWindow(len(m.exercises), m.cursor, layout.listRows)
		// The gap under the title counts the exercises hidden above. Those
		// hidden below get the last row, which listWindow leaves free when
		// the list has two rows or more; otherwise the gap counts them too.
		above, below := start, len(m.exercises)-end
		belowInLastRow := below > 0 && end-start < layout.listRows
		if belowInLastRow {
			listBuilder.WriteString(m.listMarker(above, 0, layout.listInnerWidth) + "\n")
		} else {
			listBuilder.WriteString(m.listMarker(above, below, layout.listInnerWidth) + "\n")
		}
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
				displayName = truncateCells(displayName, nameWidth)
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
		if belowInLastRow {
			listBuilder.WriteString(m.listMarker(0, below, layout.listInnerWidth))
		}
	}

	listBox := m.renderPanel(strings.TrimRight(listBuilder.String(), "\n"), layout.listOuterWidth, layout.listOuterHeight)

	// Sağ panel
	var rightBuilder strings.Builder
	rightBuilder.WriteString(m.styles.Title.Width(layout.rightInnerWidth).Render(truncateCells(instTitle, layout.rightInnerWidth)) + "\n\n")
	// The footer can change height without an Update, for example in tests
	// that render a model directly, so the viewport takes the height here.
	readme := m.readme
	readme.Height = layout.readmeHeight
	rightBuilder.WriteString(readme.View())
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

	// No trailing newline: the renderer splits on "\n" and would drop the
	// title row to fit the extra empty one.
	return m.styles.Title.Render(truncateCells(title, m.width)) +
		"\n\n" +
		topRow +
		"\n" +
		strings.Join(append(statusRows, helpRows...), "\n")
}

// listMarker is the faint list row that counts the exercises hidden above
// and below the visible ones; it is empty when none are hidden.
func (m model) listMarker(above, below, width int) string {
	var parts []string
	if above > 0 {
		parts = append(parts, "↑ "+m.uif("list_more", "%d more", above))
	}
	if below > 0 {
		parts = append(parts, "↓ "+m.uif("list_more", "%d more", below))
	}
	if len(parts) == 0 {
		return ""
	}
	text := "   " + strings.Join(parts, helpSeparator)
	return m.styles.Help.Padding(0).Render(truncateCells(text, width))
}
