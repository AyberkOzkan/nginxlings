package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/x/ansi"
)

func TestJoinListLineBreaks(t *testing.T) {
	tests := []struct {
		name, src, want string
	}{
		{
			name: "bullet item",
			src:  "- www-data is the web server account on Debian and\n  Ubuntu.\n- Second\n",
			want: "- www-data is the web server account on Debian and Ubuntu.\n- Second\n",
		},
		{
			name: "numbered item with a lazy line and trailing space",
			src:  "1. Start the \nbackend.\n",
			want: "1. Start the backend.\n",
		},
		{
			name: "nested list stays nested",
			src:  "- outer item\n  wrapped\n  - inner item\n    wrapped\n",
			want: "- outer item wrapped\n  - inner item wrapped\n",
		},
		{
			name: "code span wrapped inside an item",
			src:  "- run `nginx\n  -t` first\n",
			want: "- run `nginx -t` first\n",
		},
		{
			name: "CRLF line endings",
			src:  "- one\r\n  two\r\n",
			want: "- one two\r\n",
		},
		{
			name: "paragraph outside a list",
			src:  "A paragraph\nwrapped in the source.\n",
			want: "A paragraph\nwrapped in the source.\n",
		},
		{
			name: "list inside a block quote",
			src:  "> - quoted item\n>   wrapped\n",
			want: "> - quoted item\n>   wrapped\n",
		},
		{
			name: "code block inside an item",
			src:  "- item:\n\n  ```nginx\n  listen 8080;\n  root /srv;\n  ```\n",
			want: "- item:\n\n  ```nginx\n  listen 8080;\n  root /srv;\n  ```\n",
		},
		{
			name: "hard line break",
			src:  "- first line  \n  second line\n",
			want: "- first line  \n  second line\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := joinListLineBreaks(tt.src); got != tt.want {
				t.Fatalf("joinListLineBreaks(%q) = %q, want %q", tt.src, got, tt.want)
			}
		})
	}
}

// TestRenderedListItemReflowsInANarrowPanel is DOC-003: a list item wrapped
// in the source must not leave a short row where the source broke.
func TestRenderedListItemReflowsInANarrowPanel(t *testing.T) {
	src := "- `getpwnam(\"www-data\") failed`: www-data is the web server account on Debian and\n  Ubuntu. Which accounts does this Alpine image have?\n"
	r, err := glamour.NewTermRenderer(glamour.WithStyles(markdownStyle(false)), glamour.WithWordWrap(40))
	if err != nil {
		t.Fatal(err)
	}
	out, err := r.Render(joinListLineBreaks(src))
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range strings.Split(ansi.Strip(out), "\n") {
		if strings.TrimSpace(row) == "and" || strings.HasSuffix(strings.TrimSpace(row), "Debian and") && !strings.Contains(row, "web") {
			t.Fatalf("a source line break survived as a short row %q:\n%s", row, ansi.Strip(out))
		}
	}
	if !strings.Contains(strings.Join(strings.Fields(ansi.Strip(out)), " "), "Debian and Ubuntu.") {
		t.Fatalf("rendered item lost text:\n%s", ansi.Strip(out))
	}
}
