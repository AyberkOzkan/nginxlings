package main

import (
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

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	got := updated.(model)

	if got.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", got.cursor)
	}
	if got.marquee != 0 {
		t.Fatalf("marquee = %d, want reset to 0", got.marquee)
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
