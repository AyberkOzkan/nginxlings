package main

import (
	"fmt"
	"regexp"
	"strconv"
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
	if strings.Contains(before[1], ellipsis) || strings.Contains(after[1], ellipsis) {
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

func TestListWindowLeavesTheLastRowForTheHiddenBelowMarker(t *testing.T) {
	tests := []struct {
		name                string
		total, cursor, rows int
		wantStart, wantEnd  int
	}{
		{name: "all fit", total: 5, cursor: 4, rows: 5, wantStart: 0, wantEnd: 5},
		{name: "top", total: 18, cursor: 0, rows: 16, wantStart: 0, wantEnd: 15},
		{name: "middle", total: 18, cursor: 9, rows: 16, wantStart: 2, wantEnd: 17},
		{name: "bottom", total: 18, cursor: 17, rows: 16, wantStart: 2, wantEnd: 18},
		{name: "one row", total: 18, cursor: 5, rows: 1, wantStart: 5, wantEnd: 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := listWindow(tt.total, tt.cursor, tt.rows)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Fatalf("listWindow() = (%d, %d), want (%d, %d)", start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

func TestViewListMarksHiddenExercises(t *testing.T) {
	marker := regexp.MustCompile(`([↑↓]) (\d+) more`)
	for _, cursor := range []int{0, 9, 17} {
		t.Run(fmt.Sprint(cursor), func(t *testing.T) {
			m := layoutTestModel(80, 24)
			m.cursor = cursor
			for i := 0; i < 18; i++ {
				m.exercises = append(m.exercises, Exercise{
					ID:          fmt.Sprintf("%02d_exercise", i+1),
					DefaultName: fmt.Sprintf("%02d - Exercise", i+1),
				})
			}

			rendered := m.View()
			if got := renderedRows(rendered); got != 24 {
				t.Fatalf("rendered rows = %d, want 24\n%s", got, rendered)
			}
			shown := strings.Count(rendered, " - Exercise")
			hidden := map[string]int{}
			for _, match := range marker.FindAllStringSubmatch(rendered, -1) {
				n, _ := strconv.Atoi(match[2])
				hidden[match[1]] = n
			}
			if shown+hidden["↑"]+hidden["↓"] != 18 {
				t.Fatalf("%d shown + %v hidden != 18 exercises\n%s", shown, hidden, rendered)
			}
			if cursor == 0 && (hidden["↑"] != 0 || hidden["↓"] == 0) || cursor == 17 && (hidden["↑"] == 0 || hidden["↓"] != 0) {
				t.Fatalf("cursor %d: hidden = %v\n%s", cursor, hidden, rendered)
			}
			if !strings.Contains(rendered, fmt.Sprintf("%02d - Exercise", cursor+1)) {
				t.Fatalf("selected exercise %d is not visible\n%s", cursor+1, rendered)
			}
		})
	}
}

// TestViewDimsOnlyExercisesWithoutACheck: a tried exercise and the selected
// one keep their look; only an exercise no check has run on is dimmed.
func TestViewDimsOnlyExercisesWithoutACheck(t *testing.T) {
	m := layoutTestModel(100, 24)
	// Tests render without colour, so the untried style marks its text here.
	m.styles.UntriedItem = lipgloss.NewStyle().Transform(func(s string) string { return "<" + s + ">" })
	m.exercises = []Exercise{
		{ID: "01_done", DefaultName: "01 - Done", Completed: true, Attempted: true},
		{ID: "02_failed", DefaultName: "02 - Failed", Attempted: true},
		{ID: "03_untried", DefaultName: "03 - Untried"},
		{ID: "04_selected", DefaultName: "04 - Selected"},
	}
	m.cursor = 3

	rendered := m.View()
	for _, want := range []string{"✓ 01 - Done", "x 02 - Failed", "<x> <03 - Untried>", "➤ x 04 - Selected"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("list lacks %q:\n%s", want, rendered)
		}
	}
}

func TestViewOneRowListCountsBothDirectionsInTheGap(t *testing.T) {
	m := layoutTestModel(40, 16)
	m.cursor = 3
	for i := 0; i < 18; i++ {
		m.exercises = append(m.exercises, Exercise{ID: fmt.Sprintf("%02d_x", i+1), DefaultName: fmt.Sprintf("%02d - Exercise", i+1)})
	}
	if rows := m.layout().listRows; rows != 1 {
		t.Fatalf("list rows = %d, want 1 for this test", rows)
	}

	rendered := m.View()
	if !strings.Contains(rendered, "↑ 3 more · ↓ 14 more") {
		t.Fatalf("gap row does not count both directions:\n%s", rendered)
	}
	if got := renderedRows(rendered); got != 16 {
		t.Fatalf("rendered rows = %d, want 16\n%s", got, rendered)
	}
}

func TestViewCutExerciseNameEndsInEllipsis(t *testing.T) {
	m := layoutTestModel(60, 24)
	m.exercises = []Exercise{
		{ID: "01_short", DefaultName: "01 - Short"},
		{ID: "02_long", DefaultName: "02 - This Exercise Name Is Far Too Long For The Narrow List Panel"},
	}

	rendered := m.View()
	assertRenderedLinesWithin(t, rendered, 60)
	if !strings.Contains(rendered, "02 - This Exercise Name") || !strings.Contains(rendered, ellipsis+" │") {
		t.Fatalf("cut name does not end in an ellipsis:\n%s", rendered)
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

// footerOf returns the rows under the panels' bottom border: the status rows
// followed by the key help rows.
func footerOf(t *testing.T, rendered string) []string {
	t.Helper()
	lines := strings.Split(rendered, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], "╰") {
			return lines[i+1:]
		}
	}
	t.Fatalf("no panel bottom border in:\n%s", rendered)
	return nil
}

// squeeze drops all whitespace, so wrapped text compares equal to the original.
func squeeze(s string) string {
	return strings.Join(strings.Fields(s), "")
}

func TestViewWrapsLongStatusAndKeyHelp(t *testing.T) {
	const message = "[FAIL] NGINX rejected the configuration: getpwnam(\"www-data\") failed in /etc/nginx/nginx.conf:5"
	for _, size := range []struct{ width, height int }{{80, 24}, {120, 30}, {60, 30}, {40, 30}} {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			m := layoutTestModel(size.width, size.height)
			m.status = message

			rendered := m.View()
			if got := renderedRows(rendered); got != size.height {
				t.Fatalf("rendered rows = %d, want %d\n%s", got, size.height, rendered)
			}
			assertRenderedLinesWithin(t, rendered, size.width)

			footer := footerOf(t, rendered)
			helpStart := -1
			for i, row := range footer {
				if strings.HasPrefix(strings.TrimSpace(row), "W/S select") {
					helpStart = i
					break
				}
			}
			if helpStart < 1 {
				t.Fatalf("footer = %q, want status rows followed by the key help", footer)
			}
			if !strings.Contains(footer[0], "Lang:") && size.width >= 60 {
				t.Fatalf("first status row = %q, want the language", footer[0])
			}
			status := strings.Join(footer[:helpStart], " ")
			if !strings.Contains(squeeze(status), squeeze(message)) {
				t.Fatalf("status rows = %q, want the whole message %q", footer[:helpStart], message)
			}
			help := strings.Join(footer[helpStart:], " ")
			for _, entry := range []string{"? all keys", "q quit"} {
				if !strings.Contains(help, entry) {
					t.Fatalf("key help rows = %q, want %q", footer[helpStart:], entry)
				}
			}
		})
	}
}

func TestViewCutsStatusAndHelpWhenPanelsNeedTheRoom(t *testing.T) {
	const message = "[FAIL] NGINX rejected the configuration: getpwnam(\"www-data\") failed in /etc/nginx/nginx.conf:5"
	tests := []struct {
		width, height, statusRows int
		cut                       bool
	}{
		{80, 14, 1, true},
		{60, 14, 1, true},
		{40, 16, 1, true},
		// Two spare rows: the status takes both, the key help keeps one.
		{60, 24, 3, false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%dx%d", tt.width, tt.height), func(t *testing.T) {
			m := layoutTestModel(tt.width, tt.height)
			m.status = message

			rendered := m.View()
			if got := renderedRows(rendered); got != tt.height {
				t.Fatalf("rendered rows = %d, want %d\n%s", got, tt.height, rendered)
			}
			assertRenderedLinesWithin(t, rendered, tt.width)
			footer := footerOf(t, rendered)
			if len(footer) != tt.statusRows+1 {
				t.Fatalf("footer = %q, want %d status rows and one key help row", footer, tt.statusRows)
			}
			if cut := strings.HasSuffix(strings.TrimSpace(footer[tt.statusRows-1]), ellipsis); cut != tt.cut {
				t.Fatalf("last status row = %q, cut = %v, want %v", footer[tt.statusRows-1], cut, tt.cut)
			}
			if !strings.HasPrefix(strings.TrimSpace(footer[tt.statusRows]), "W/S select") {
				t.Fatalf("help row = %q, want key help", footer[tt.statusRows])
			}
			if got := m.layoutWithFooter(len(footer)).readmeHeight; got < minReadmeRows && len(footer) > 2 {
				t.Fatalf("instructions height = %d, want at least %d", got, minReadmeRows)
			}
		})
	}
}

func TestFooterRowsLeaveTheListHeightAlone(t *testing.T) {
	for _, size := range []struct{ width, height int }{{60, 24}, {40, 30}, {80, 24}} {
		base := calculateLayout(size.width, size.height)
		tall := calculateLayoutWithFooter(size.width, size.height, 5)
		if tall.listOuterHeight != base.listOuterHeight && size.width < sideBySideMinWidth {
			t.Fatalf("%dx%d list height = %d with a tall footer, want %d", size.width, size.height, tall.listOuterHeight, base.listOuterHeight)
		}
		if tall.readmeHeight != base.readmeHeight-3 {
			t.Fatalf("%dx%d instructions height = %d with 3 extra footer rows, want %d", size.width, size.height, tall.readmeHeight, base.readmeHeight-3)
		}
	}
}

func TestWrapSegmentsKeepsWholeEntries(t *testing.T) {
	help := "W/S seç · c/Enter kontrol · Tab görünüm · h ipucu · l dil · r sıfırla · q çık · ↑/↓ PgUp/PgDn ←/→ kaydır"

	tests := []struct {
		width, maxRows int
		want           []string
	}{
		{200, 3, []string{help}},
		{78, 1, []string{"W/S seç · c/Enter kontrol · Tab görünüm · h ipucu · l dil · r sıfırla · q çık"}},
		{78, 3, []string{
			"W/S seç · c/Enter kontrol · Tab görünüm · h ipucu · l dil · r sıfırla · q çık",
			"↑/↓ PgUp/PgDn ←/→ kaydır",
		}},
		{58, 3, []string{
			"W/S seç · c/Enter kontrol · Tab görünüm · h ipucu · l dil",
			"r sıfırla · q çık · ↑/↓ PgUp/PgDn ←/→ kaydır",
		}},
		{20, 1, []string{"W/S seç"}},
	}
	for _, tt := range tests {
		got := wrapSegments(help, tt.width, tt.maxRows)
		if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
			t.Errorf("wrapSegments(%d, %d) = %q, want %q", tt.width, tt.maxRows, got, tt.want)
		}
	}
	for _, row := range wrapSegments(help, 5, 3) {
		if lipgloss.Width(row) > 5 {
			t.Fatalf("wrapSegments(5) row %q is wider than 5", row)
		}
	}
}

func TestWrapTextEndsInEllipsisWhenRowsRunOut(t *testing.T) {
	text := "[FAIL] NGINX rejected the configuration: getpwnam(\"www-data\") failed in /etc/nginx/nginx.conf:5"

	if got := wrapText(text, 200, 20, 4); len(got) != 1 || got[0] != text {
		t.Fatalf("wrapText(200) = %q, want the text on one row", got)
	}
	rows := wrapText(text, 40, 0, 4)
	if len(rows) != 3 || squeeze(strings.Join(rows, "")) != squeeze(text) {
		t.Fatalf("wrapText(40, 0, 4) = %q, want the whole text on 3 rows", rows)
	}
	indented := wrapText(text, 40, 20, 4)
	if lipgloss.Width(indented[0]) > 20 || squeeze(strings.Join(indented, "")) != squeeze(text) {
		t.Fatalf("wrapText(40, 20, 4) = %q, want a first row of at most 20 cells and the whole text", indented)
	}
	rows = wrapText(text, 40, 0, 2)
	if len(rows) != 2 || !strings.HasSuffix(rows[1], ellipsis) {
		t.Fatalf("wrapText(40, 0, 2) = %q, want 2 rows ending in an ellipsis", rows)
	}
	if one := wrapText(text, 40, 20, 1); len(one) != 1 || lipgloss.Width(one[0]) > 20 || !strings.HasSuffix(one[0], ellipsis) {
		t.Fatalf("wrapText(40, 20, 1) = %q, want one cut row of at most 20 cells", one)
	}
	for _, row := range wrapText(text, 7, 3, 4) {
		if lipgloss.Width(row) > 7 {
			t.Fatalf("wrapText(7) row %q is wider than 7", row)
		}
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
