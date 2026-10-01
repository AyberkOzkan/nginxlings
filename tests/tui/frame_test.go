package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// frameTexts are the translated strings a frame check looks for.
type frameTexts struct {
	title, ready, working, helpStart string
}

func textsFor(t *testing.T, lang string) frameTexts {
	t.Helper()
	help := uiText(t, lang, "help")
	return frameTexts{
		title:     uiText(t, lang, "title"),
		ready:     uiText(t, lang, "ready_label"),
		working:   uiText(t, lang, "working_label"),
		helpStart: strings.SplitN(help, " ", 2)[0],
	}
}

// isCompact mirrors the layout rule in cmd/nginxlings/view.go: short
// terminals get a title, the selected exercise and a status line only.
func isCompact(cols, rows int) bool {
	return rows < 9 || cols < 80 && rows < 14
}

// checkFrame returns nil when screen is a complete, undamaged nginxlings frame
// for a cols x rows terminal, and otherwise says what is wrong.
func checkFrame(screen []string, cols, rows int, texts frameTexts) error {
	if len(screen) != rows {
		return fmt.Errorf("screen has %d rows, want %d", len(screen), rows)
	}
	for i, row := range screen {
		if n := len([]rune(row)); n > cols {
			return fmt.Errorf("row %d is %d cells wide, terminal has %d", i, n, cols)
		}
	}
	corners := strings.Count(strings.Join(screen, "\n"), "╭")

	if isCompact(cols, rows) {
		if corners != 0 {
			return fmt.Errorf("compact %dx%d layout draws panel borders", cols, rows)
		}
		used := 0
		for _, row := range screen {
			if strings.TrimSpace(row) != "" {
				used++
			}
		}
		if used > 3 {
			return fmt.Errorf("compact layout uses %d rows, want at most 3", used)
		}
		hasStatus := false
		for _, row := range screen {
			hasStatus = hasStatus || showsStatus(row, texts)
		}
		if !hasStatus {
			return fmt.Errorf("compact layout has no status line")
		}
		if rows >= 3 && !showsTitle(screen[0], texts.title, cols) {
			return fmt.Errorf("row 0 = %q, want the title", screen[0])
		}
		return nil
	}

	if !showsTitle(screen[0], texts.title, cols) {
		return fmt.Errorf("row 0 = %q, want the title %q", screen[0], texts.title)
	}
	if err := checkFooter(screen, texts); err != nil {
		return err
	}
	joined := strings.Join(screen, "\n")
	for _, corner := range []string{"╭", "╮", "╰", "╯"} {
		if n := strings.Count(joined, corner); n != 2 {
			return fmt.Errorf("found %d %q corners, want 2 (two closed panels)", n, corner)
		}
	}
	for i, row := range screen {
		if !strings.ContainsAny(row, "│╭╮╰╯") {
			continue
		}
		cells := []rune(row)
		if len(cells) != cols {
			return fmt.Errorf("row %d frame ends at column %d, want %d: %q", i, len(cells), cols, row)
		}
		if last := cells[len(cells)-1]; !strings.ContainsRune("│╮╯", last) {
			return fmt.Errorf("row %d ends with %q instead of a right border: %q", i, last, row)
		}
	}
	sideBySide := strings.Contains(joined, "╮╭")
	if cols >= 80 && !sideBySide {
		return fmt.Errorf("%d columns should show the panels side by side", cols)
	}
	if cols < 80 && sideBySide {
		return fmt.Errorf("%d columns should stack the panels", cols)
	}
	return nil
}

// showsTitle reports whether row starts with title, or with its beginning
// followed by "…" when the title is wider than the terminal.
func showsTitle(row, title string, cols int) bool {
	if strings.HasPrefix(row, title) {
		return true
	}
	r := []rune(title)
	if len(r) <= cols || cols < 1 {
		return false
	}
	return strings.HasPrefix(row, string(r[:cols-1])+"…")
}

func showsStatus(row string, texts frameTexts) bool {
	return strings.Contains(row, texts.ready) || strings.Contains(row, texts.working)
}

// footerRowsMax mirrors maxStatusRows + maxHelpRows in cmd/nginxlings/view.go.
const footerRowsMax = 4 + 3

// checkFooter checks the rows under the panels: the status starts right under
// the bottom border, the key help follows it and ends on the last row, and no
// footer row is empty. Both may wrap onto several rows.
func checkFooter(screen []string, texts frameTexts) error {
	bottom := -1
	for i := len(screen) - 1; i >= 0; i-- {
		if strings.Contains(screen[i], "╰") {
			bottom = i
			break
		}
	}
	if bottom < 0 {
		return fmt.Errorf("no panel bottom border")
	}
	footer := screen[bottom+1:]
	if len(footer) < 2 || len(footer) > footerRowsMax {
		return fmt.Errorf("%d rows under the panels, want 2 to %d", len(footer), footerRowsMax)
	}
	if !showsStatus(footer[0], texts) {
		return fmt.Errorf("row %d = %q, want the status under the panels", bottom+1, footer[0])
	}
	help := -1
	for i, row := range footer {
		if strings.TrimSpace(row) == "" {
			return fmt.Errorf("row %d under the panels is empty", bottom+1+i)
		}
		if help < 0 && i > 0 && strings.HasPrefix(strings.TrimSpace(row), texts.helpStart) {
			help = i
		}
	}
	if help < 0 {
		return fmt.Errorf("no key help after the status rows %q", footer)
	}
	return nil
}

// waitForFrame waits until the screen is a valid frame for cols x rows.
func (s *session) waitForFrame(cols, rows int, texts frameTexts) []string {
	s.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		screen := s.screen()
		err := checkFrame(screen, cols, rows, texts)
		if err == nil {
			return screen
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("no valid %dx%d frame: %v\n%s", cols, rows, err, s.dump())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

var frameSizes = []struct{ cols, rows int }{
	{30, 6}, {40, 8}, {40, 12}, {40, 16}, {60, 13}, {60, 14}, {60, 20}, {79, 24},
	{80, 9}, {80, 12}, {80, 24}, {100, 30}, {120, 40}, {160, 50}, {220, 60},
}

// TestTUIFrameAtEachSize starts the TUI at many terminal sizes and checks
// that every first frame fits and is intact.
func TestTUIFrameAtEachSize(t *testing.T) {
	for _, lang := range []string{"en", "tr"} {
		texts := textsFor(t, lang)
		for _, size := range frameSizes {
			size := size
			t.Run(fmt.Sprintf("%s_%dx%d", lang, size.cols, size.rows), func(t *testing.T) {
				t.Parallel()
				s := start(t, options{cols: size.cols, rows: size.rows, lang: lang})
				s.waitForFrame(size.cols, size.rows, texts)
			})
		}
	}
}

// TestTUIResizeKeepsFrameIntact resizes one running TUI through growing and
// shrinking sizes; after each resize the frame must settle and stay intact.
func TestTUIResizeKeepsFrameIntact(t *testing.T) {
	texts := textsFor(t, "en")
	s := start(t, options{cols: 100, rows: 30, lang: "en"})
	s.waitForFrame(100, 30, texts)
	sequence := []struct{ cols, rows int }{
		{80, 24}, {60, 20}, {40, 12}, {30, 6}, {120, 40}, {79, 24}, {80, 24}, {220, 60}, {60, 14}, {100, 30},
	}
	for _, size := range sequence {
		s.resize(size.cols, size.rows)
		s.waitForFrame(size.cols, size.rows, texts)
	}
}

// TestFrameCheckRejectsDamagedScreens proves the frame check is not a no-op:
// it takes a real screen and damages it the ways a broken layout would.
func TestFrameCheckRejectsDamagedScreens(t *testing.T) {
	texts := textsFor(t, "en")
	s := start(t, options{cols: 100, rows: 30})
	good := s.waitForFrame(100, 30, texts)

	damage := map[string]func([]string) []string{
		"border one cell short": func(rows []string) []string {
			r := []rune(rows[5])
			rows[5] = string(r[:len(r)-1])
			return rows
		},
		"row wider than the terminal": func(rows []string) []string {
			rows[5] += "│"
			return rows
		},
		"missing status line": func(rows []string) []string {
			for i, row := range rows {
				if strings.Contains(row, texts.ready) {
					rows[i] = ""
				}
			}
			return rows
		},
		"missing key help": func(rows []string) []string {
			rows[len(rows)-1] = ""
			return rows
		},
		"title scrolled away": func(rows []string) []string {
			return append(rows[1:], "")
		},
		"panel not closed": func(rows []string) []string {
			for i := len(rows) - 1; i >= 0; i-- {
				if strings.Contains(rows[i], "╯") {
					rows[i] = strings.Replace(rows[i], "╯", "│", 1)
					break
				}
			}
			return rows
		},
		"extra row": func(rows []string) []string {
			return append(rows, "")
		},
	}
	for name, apply := range damage {
		screen := apply(append([]string(nil), good...))
		if err := checkFrame(screen, 100, 30, texts); err == nil {
			t.Errorf("%s: checkFrame accepted the damaged screen", name)
		}
	}
}
