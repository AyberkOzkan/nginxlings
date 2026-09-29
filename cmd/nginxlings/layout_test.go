package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestCalculateLayoutSmallWidth(t *testing.T) {
	layout := calculateLayout(48, 24)

	if layout.listOuterWidth != 48 {
		t.Fatalf("listOuterWidth = %d, want 48", layout.listOuterWidth)
	}
	if layout.rightOuterWidth != 48 {
		t.Fatalf("rightOuterWidth = %d, want 48", layout.rightOuterWidth)
	}
	if layout.listInnerWidth <= 0 || layout.rightInnerWidth <= 0 {
		t.Fatalf("inner widths must be positive: %+v", layout)
	}
}

func TestCalculateLayoutSmallHeight(t *testing.T) {
	layout := calculateLayout(100, 3)

	if layout.readmeHeight < 1 {
		t.Fatalf("readmeHeight = %d, want >= 1", layout.readmeHeight)
	}
	if layout.listOuterHeight < 1 || layout.rightOuterHeight < 1 || layout.listRows < 1 {
		t.Fatalf("vertical dimensions must be positive: %+v", layout)
	}
	if !layout.compact {
		t.Fatalf("compact = false, want true for a three-line terminal")
	}
}

func TestCalculateLayoutWidePanelWidths(t *testing.T) {
	layout := calculateLayout(120, 30)

	if got, want := layout.listOuterWidth+layout.rightOuterWidth, 120; got != want {
		t.Fatalf("total panel width = %d, want %d", got, want)
	}
	if layout.listOuterWidth < 40 {
		t.Fatalf("listOuterWidth = %d, want >= 40", layout.listOuterWidth)
	}
	if layout.rightOuterWidth < 34 {
		t.Fatalf("rightOuterWidth = %d, want >= 34", layout.rightOuterWidth)
	}
}

func TestViewNormalWideLayoutDoesNotTruncateExerciseName(t *testing.T) {
	name := "18 - OSI Layer 4/7 & Use-Case Mapping"
	m := layoutTestModel(100, 24)
	m.exercises = []Exercise{{
		ID:          "18_osi_layers_usecases",
		DefaultName: name,
	}}

	view := m.View()
	assertRenderedLinesWithin(t, view, 100)
	if !strings.Contains(view, name) {
		t.Fatalf("view should keep normal exercise name readable, got:\n%s", view)
	}
}

func TestViewVeryLongExerciseNameDoesNotWrapListLine(t *testing.T) {
	m := layoutTestModel(100, 24)
	m.exercises = []Exercise{{
		ID:          "01_static_server",
		DefaultName: "01 - This Exercise Name Is Intentionally Very Long And Should Not Push The Right Panel Away",
	}}

	view := m.View()
	assertRenderedLinesWithin(t, view, 100)
	if strings.Contains(view, "Push The Right Panel Away") {
		t.Fatalf("view should not allow extreme list name to overflow, got:\n%s", view)
	}
	if strings.Contains(view, "\n│ Intentionally") {
		t.Fatalf("view should not wrap clipped list name to a second line, got:\n%s", view)
	}
}

func TestMarqueeShortExerciseNameDoesNotAnimate(t *testing.T) {
	name := "01 - Static"

	if got := marqueeText(name, 30, 5); got != name {
		t.Fatalf("marqueeText() = %q, want %q", got, name)
	}
}

func TestMarqueeLongSelectedExerciseNameChangesAfterTick(t *testing.T) {
	name := "01 - This Exercise Name Is Long Enough To Scroll"
	m := layoutTestModel(50, 16)
	m.exercises = []Exercise{{ID: "01_long", DefaultName: name}}
	m.cursor = 0

	before := m.View()
	updated, _ := m.Update(marqueeTickMsg{})
	after := updated.(model).View()

	if before == after {
		t.Fatalf("selected long exercise view did not change after marquee tick\nbefore:\n%s\nafter:\n%s", before, after)
	}
	assertRenderedLinesWithin(t, after, 50)
}

func TestMarqueeOffsetResetsWhenSelectionChanges(t *testing.T) {
	m := layoutTestModel(50, 16)
	m.exercises = []Exercise{
		{ID: "01_long", DefaultName: "01 - This Exercise Name Is Long Enough To Scroll"},
		{ID: "02_other", DefaultName: "02 - Another Exercise Name Long Enough To Scroll"},
	}
	m.cursor = 0
	m.marquee = 7
	m.marqueeGeneration = 3

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	got := updated.(model)

	if got.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", got.cursor)
	}
	if got.marquee != 0 {
		t.Fatalf("marquee = %d, want reset to 0", got.marquee)
	}
	if got.marqueeGeneration != 4 {
		t.Fatalf("marqueeGeneration = %d, want 4", got.marqueeGeneration)
	}
}

func TestNonSelectedLongExerciseNameDoesNotAnimate(t *testing.T) {
	m := layoutTestModel(50, 16)
	m.exercises = []Exercise{
		{ID: "01_short", DefaultName: "01 - Short"},
		{ID: "02_long", DefaultName: "02 - This Non Selected Exercise Name Is Long Enough To Clip"},
	}
	m.cursor = 0
	m.marquee = 10

	before := m.View()
	m.marquee = 11
	after := m.View()

	if before != after {
		t.Fatalf("non-selected long exercise should not animate\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestSelectedMarqueeDoesNotWrapInNarrowLayout(t *testing.T) {
	m := layoutTestModel(42, 14)
	m.exercises = []Exercise{{
		ID:          "01_long",
		DefaultName: "01 - This Selected Exercise Name Is Long Enough To Need Marquee",
	}}
	m.cursor = 0
	m.marquee = 3

	view := m.View()
	assertRenderedLinesWithin(t, view, 42)
	if strings.Contains(view, "\n│ Selected") {
		t.Fatalf("selected marquee wrapped to a second list line, got:\n%s", view)
	}
}

func TestViewStatusHelpDoesNotExceedNarrowLayout(t *testing.T) {
	m := layoutTestModel(42, 12)
	m.status = "This status message is intentionally long and should be truncated safely"

	view := m.View()
	assertRenderedLinesWithin(t, view, 42)
}

func TestCompactSelectedExerciseUsesMarqueeWithoutEllipsis(t *testing.T) {
	m := layoutTestModel(24, 6)
	m.exercises = []Exercise{{
		ID:          "01_long",
		DefaultName: "01 - This Selected Exercise Name Must Scroll",
	}}

	before := strings.Split(strings.TrimRight(m.View(), "\n"), "\n")
	updated, _ := m.Update(marqueeTickMsg{})
	afterView := updated.(model).View()
	after := strings.Split(strings.TrimRight(afterView, "\n"), "\n")
	if len(before) < 2 || len(after) < 2 {
		t.Fatalf("compact render does not contain selected row\nbefore: %q\nafter: %q", before, after)
	}
	if strings.Contains(before[1], "...") || strings.Contains(after[1], "...") {
		t.Fatalf("compact selected row must not use ellipsis\nbefore: %q\nafter: %q", before[1], after[1])
	}
	if before[1] == after[1] {
		t.Fatalf("compact selected row did not move after marquee tick: %q", before[1])
	}
	assertRenderedLinesWithin(t, afterView, 24)
}

func TestMarqueeLanguageSwitchResetsAndReevaluatesOverflow(t *testing.T) {
	i18n := &I18n{
		langs: []LangMeta{
			{Code: "en", Label: "English"},
			{Code: "tr", Label: "Türkçe"},
		},
		uiStrings: map[string]map[string]string{
			"en": {"status_lang_switched": "Language switched."},
			"tr": {"status_lang_switched": "Dil değiştirildi."},
		},
		exerciseNames: map[string]map[string]string{
			"en": {"01_example": "Short title"},
			"tr": {"01_example": "Bu egzersiz başlığı seçili liste satırına kesinlikle sığmayacak kadar uzundur"},
		},
	}
	m := layoutTestModel(50, 16)
	m.i18n = i18n
	m.exercises = []Exercise{{ID: "01_example", DefaultName: "Fallback"}}
	m.marquee = 7
	m.marqueeGeneration = 4

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	longTitle := updated.(model)
	if longTitle.currentLangCode() != "tr" {
		t.Fatalf("language = %q, want tr", longTitle.currentLangCode())
	}
	if longTitle.marquee != 0 {
		t.Fatalf("marquee = %d, want reset to 0", longTitle.marquee)
	}
	if longTitle.marqueeGeneration != 5 {
		t.Fatalf("marqueeGeneration = %d, want 5", longTitle.marqueeGeneration)
	}
	if cmd == nil {
		t.Fatal("long translated title did not schedule marquee")
	}

	longTitle.marquee = 9
	updated, cmd = longTitle.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	shortTitle := updated.(model)
	if shortTitle.currentLangCode() != "en" {
		t.Fatalf("language = %q, want en", shortTitle.currentLangCode())
	}
	if shortTitle.marquee != 0 {
		t.Fatalf("marquee = %d, want reset to 0", shortTitle.marquee)
	}
	if shortTitle.marqueeGeneration != 6 {
		t.Fatalf("marqueeGeneration = %d, want 6", shortTitle.marqueeGeneration)
	}
	if cmd != nil {
		t.Fatal("short translated title scheduled an unnecessary marquee")
	}
}

func TestMarqueeIgnoresStaleTick(t *testing.T) {
	m := layoutTestModel(50, 16)
	m.exercises = []Exercise{{
		ID:          "01_long",
		DefaultName: "01 - This Exercise Name Is Long Enough To Scroll",
	}}
	m.marquee = 5
	m.marqueeGeneration = 8

	updated, cmd := m.Update(marqueeTickMsg{generation: 7})
	got := updated.(model)
	if got.marquee != 5 {
		t.Fatalf("marquee = %d, want stale tick to leave offset at 5", got.marquee)
	}
	if got.marqueeGeneration != 8 {
		t.Fatalf("marqueeGeneration = %d, want 8", got.marqueeGeneration)
	}
	if cmd != nil {
		t.Fatal("stale tick scheduled another marquee tick")
	}
}

func TestMarqueeTickDelayPausesAtReadableBoundaries(t *testing.T) {
	if got := marqueeTickDelay(0, 12); got != marqueePauseInterval {
		t.Fatalf("initial delay = %s, want %s", got, marqueePauseInterval)
	}
	if got := marqueeTickDelay(1, 12); got != marqueeTickInterval {
		t.Fatalf("scroll delay = %s, want %s", got, marqueeTickInterval)
	}
	if got := marqueeTickDelay(12, 12); got != marqueePauseInterval {
		t.Fatalf("full-cycle delay = %s, want %s", got, marqueePauseInterval)
	}
}

func TestVisibleExerciseRangeKeepsSelectionInBoundedWindow(t *testing.T) {
	tests := []struct {
		name      string
		cursor    int
		wantStart int
		wantEnd   int
	}{
		{name: "start", cursor: 0, wantStart: 0, wantEnd: 4},
		{name: "middle", cursor: 10, wantStart: 8, wantEnd: 12},
		{name: "end", cursor: 17, wantStart: 14, wantEnd: 18},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := visibleExerciseRange(18, tt.cursor, 4)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Fatalf("visibleExerciseRange() = (%d, %d), want (%d, %d)", start, end, tt.wantStart, tt.wantEnd)
			}
			if tt.cursor < start || tt.cursor >= end {
				t.Fatalf("selection %d is outside visible range [%d, %d)", tt.cursor, start, end)
			}
		})
	}
}

func TestViewFitsTerminalHeightAndKeepsStatusVisible(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{name: "wide", width: 120, height: 30},
		{name: "standard 80x24", width: 80, height: 24},
		{name: "narrow stacked", width: 60, height: 24},
		{name: "very short compact", width: 42, height: 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := layoutTestModel(tt.width, tt.height)
			m.cursor = 10
			m.exercises = make([]Exercise, 18)
			for i := range m.exercises {
				m.exercises[i] = Exercise{
					ID:          fmt.Sprintf("%02d_exercise", i+1),
					DefaultName: fmt.Sprintf("%02d - Exercise", i+1),
				}
			}

			rendered := m.View()
			assertRenderedLinesWithin(t, rendered, tt.width)
			if got := renderedRows(rendered); got > tt.height {
				t.Fatalf("rendered rows = %d, want <= %d\n%s", got, tt.height, rendered)
			}
			if !strings.Contains(rendered, "Nginxlings") {
				t.Fatalf("title is not visible in %dx%d render:\n%s", tt.width, tt.height, rendered)
			}
			if !strings.Contains(rendered, "Ready") {
				t.Fatalf("status is not visible in %dx%d render:\n%s", tt.width, tt.height, rendered)
			}
			if !strings.Contains(rendered, "11 - Exercise") {
				t.Fatalf("selected exercise is not visible in %dx%d render:\n%s", tt.width, tt.height, rendered)
			}
		})
	}
}

func TestViewKeepsViewportWideLinesIntact(t *testing.T) {
	for _, size := range []struct{ width, height int }{{80, 24}, {100, 30}, {60, 24}} {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			m := layoutTestModel(size.width, size.height)
			fullLine := strings.Repeat("x", m.readme.Width)
			m.readme.SetContent(fullLine + "\n[FAIL] nginx -t failed for 01_static_server because this line is long")

			rendered := m.View()
			assertRenderedLinesWithin(t, rendered, size.width)
			if got := renderedRows(rendered); got > size.height {
				t.Fatalf("rendered rows = %d, want <= %d\n%s", got, size.height, rendered)
			}
			if !strings.Contains(rendered, fullLine) {
				t.Fatalf("viewport-wide line was wrapped by the panel:\n%s", rendered)
			}
			if !strings.HasPrefix(rendered, "Nginxlings") {
				t.Fatalf("title row was pushed out of the frame:\n%s", rendered)
			}
		})
	}
}

func TestViewPanelsUseFullTerminalWidth(t *testing.T) {
	for _, width := range []int{80, 100, 120, 160} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			m := layoutTestModel(width, 24)
			lines := strings.Split(m.View(), "\n")
			// Row 0 is the title and row 1 the gap; row 2 is the panel top border.
			if got := lipgloss.Width(lines[2]); got != width {
				t.Fatalf("panel row width = %d, want %d\n%s", got, width, lines[2])
			}
		})
	}
}

func TestViewKeepsKeyHelpAndLanguageVisibleWithLongStatus(t *testing.T) {
	for _, size := range []struct{ width, height int }{{80, 24}, {60, 24}, {120, 30}} {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			m := layoutTestModel(size.width, size.height)
			m.status = "[FAIL] This status message is long enough to fill the whole status row on its own"

			rendered := m.View()
			lines := strings.Split(rendered, "\n")
			if got := len(lines); got > size.height {
				t.Fatalf("rendered rows = %d, want <= %d\n%s", got, size.height, rendered)
			}
			statusRow, helpRow := lines[len(lines)-2], lines[len(lines)-1]
			if !strings.Contains(statusRow, "Lang:") || !strings.Contains(statusRow, "[FAIL]") {
				t.Fatalf("status row = %q, want language and status", statusRow)
			}
			if !strings.HasPrefix(strings.TrimSpace(helpRow), "W/S select · c/Enter check") {
				t.Fatalf("help row = %q, want key help", helpRow)
			}
			assertRenderedLinesWithin(t, rendered, size.width)
		})
	}
}

func TestTruncateSegmentsDropsWholeEntries(t *testing.T) {
	help := "W/S seç · c/Enter kontrol · Tab görünüm · h ipucu · l dil · r sıfırla · q çık · ↑/↓ PgUp/PgDn ←/→ kaydır"

	if got := truncateSegments(help, 78); got != "W/S seç · c/Enter kontrol · Tab görünüm · h ipucu · l dil · r sıfırla · q çık" {
		t.Fatalf("truncateSegments(78) = %q", got)
	}
	if got := truncateSegments(help, 20); got != "W/S seç" {
		t.Fatalf("truncateSegments(20) = %q", got)
	}
	if got := truncateSegments(help, 5); lipgloss.Width(got) > 5 {
		t.Fatalf("truncateSegments(5) = %q, wider than 5", got)
	}
	if got := truncateSegments(help, 200); got != help {
		t.Fatalf("truncateSegments(200) = %q, want unchanged", got)
	}
}

func TestCalculateLayoutGivesInstructionsRoomAt80Columns(t *testing.T) {
	layout := calculateLayout(80, 24)

	if layout.rightOuterWidth < rightMinOuterWidth {
		t.Fatalf("rightOuterWidth = %d, want >= %d", layout.rightOuterWidth, rightMinOuterWidth)
	}
	if got := layout.listOuterWidth + layout.rightOuterWidth; got != 80 {
		t.Fatalf("total panel width = %d, want 80", got)
	}
}

func TestMarkdownStyleDropsDocumentMargin(t *testing.T) {
	for _, light := range []bool{false, true} {
		style := markdownStyle(light)
		if style.Document.Margin == nil || *style.Document.Margin != 0 {
			t.Fatalf("light=%v document margin = %v, want 0", light, style.Document.Margin)
		}
		if style.Document.BlockPrefix != "" {
			t.Fatalf("light=%v document block prefix = %q, want empty", light, style.Document.BlockPrefix)
		}
	}
	if *markdownStyle(false).Document.Margin == *markdownStyle(false).CodeBlock.Margin {
		t.Fatal("code block margin should be kept to set code apart from prose")
	}
}

// renderedRows counts rows the way Bubble Tea's renderer does: it splits the
// frame on "\n", so a trailing newline adds an extra row.
func renderedRows(view string) int {
	return len(strings.Split(view, "\n"))
}

func layoutTestModel(width, height int) model {
	layout := calculateLayout(width, height)
	vp := viewport.New(layout.rightInnerWidth, layout.readmeHeight)
	vp.SetContent("Instructions")

	return model{
		ready:      true,
		width:      width,
		height:     height,
		listWidth:  layout.listOuterWidth,
		rightWidth: layout.rightOuterWidth,
		readme:     vp,
		status:     "Ready",
		styles:     DefaultStyles(),
	}
}

func assertRenderedLinesWithin(t *testing.T, rendered string, width int) {
	t.Helper()

	for _, line := range strings.Split(strings.TrimRight(rendered, "\n"), "\n") {
		if got := lipgloss.Width(line); got > width {
			t.Fatalf("line width = %d, want <= %d\nline: %q\nrendered:\n%s", got, width, line, rendered)
		}
	}
}
