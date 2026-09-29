package main

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

// sanitizeCheckOutput turns raw check.sh output into plain text that is safe
// to draw inside the TUI. Escape sequences are removed because a child tool
// (docker, curl, a package manager) could otherwise move the cursor or change
// terminal state. A line rewritten with "\r" keeps its last non-empty text,
// tabs become spaces, and other control characters are dropped.
func sanitizeCheckOutput(raw string) string {
	raw = ansi.Strip(raw)
	raw = strings.ReplaceAll(raw, "\r\n", "\n")

	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		lines[i] = sanitizeCheckOutputLine(line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func sanitizeCheckOutputLine(line string) string {
	segments := strings.Split(line, "\r")
	line = ""
	for j := len(segments) - 1; j >= 0; j-- {
		if strings.TrimSpace(segments[j]) != "" {
			line = segments[j]
			break
		}
	}

	line = strings.ReplaceAll(line, "\t", "    ")
	line = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, line)
	return strings.TrimRight(line, " ")
}

// wrapCheckOutput soft-wraps sanitized output to the viewport width so long
// lines stay readable instead of being cut at the panel edge.
func wrapCheckOutput(output string, width int) string {
	if width <= 0 {
		return output
	}
	return ansi.Wrap(output, width, "")
}

// selectedCheckOutput is the output of the last check run in this session for
// the selected exercise, or "" when it has not been checked yet.
func (m model) selectedCheckOutput() string {
	if m.cursor < 0 || m.cursor >= len(m.exercises) {
		return ""
	}
	return m.checkOutputs[m.exercises[m.cursor].ID]
}

// showCheckOutput switches the right panel to the selected exercise's output.
func (m *model) showCheckOutput() {
	m.panel = panelCheckOutput
	m.readme.SetContent(wrapCheckOutput(m.selectedCheckOutput(), m.readme.Width))
	m.readme.GotoTop()
}
