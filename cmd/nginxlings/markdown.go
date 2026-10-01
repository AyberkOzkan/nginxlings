package main

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// joinListLineBreaks returns the Markdown source with the line breaks inside
// list items replaced by spaces. Glamour reflows paragraphs to the panel
// width but keeps the line breaks of a list item, so an item wrapped in the
// source leaves short rows such as "Debian ve" in a narrow panel. In
// Markdown such a line break reads as a space, so joining the lines keeps
// the meaning. Lists inside block quotes are left alone, because their lines
// start with ">".
func joinListLineBreaks(src string) string {
	source := []byte(src)
	doc := goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser().Parse(text.NewReader(source))

	// breaks holds, in document order, the offsets where a line break inside
	// a list item starts.
	var breaks []int
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		t, ok := n.(*ast.Text)
		if !entering || !ok || !inListItem(n) {
			return ast.WalkContinue, nil
		}
		switch {
		case t.SoftLineBreak():
			breaks = append(breaks, t.Segment.Stop)
		case n.Parent().Kind() == ast.KindCodeSpan && bytes.HasSuffix(t.Segment.Value(source), []byte("\n")):
			// A code span's line break is part of its text segment.
			breaks = append(breaks, t.Segment.Stop-1)
		}
		return ast.WalkContinue, nil
	})

	var out strings.Builder
	last := 0
	for _, start := range breaks {
		end, ok := lineBreakEnd(source, start)
		if !ok || start < last {
			continue
		}
		out.Write(source[last:start])
		out.WriteByte(' ')
		last = end
	}
	out.Write(source[last:])
	return out.String()
}

// lineBreakEnd returns where the whitespace that starts at start ends, when
// it holds exactly one line break: spaces and tabs, the break, and the next
// line's indentation.
func lineBreakEnd(source []byte, start int) (int, bool) {
	i := start
	for i < len(source) && (source[i] == ' ' || source[i] == '\t') {
		i++
	}
	if i < len(source) && source[i] == '\r' {
		i++
	}
	if i >= len(source) || source[i] != '\n' {
		return 0, false
	}
	i++
	for i < len(source) && (source[i] == ' ' || source[i] == '\t') {
		i++
	}
	return i, true
}

// inListItem reports whether n is inside a list item and not inside a block
// quote.
func inListItem(n ast.Node) bool {
	found := false
	for p := n.Parent(); p != nil; p = p.Parent() {
		switch p.Kind() {
		case ast.KindBlockquote:
			return false
		case ast.KindListItem:
			found = true
		}
	}
	return found
}
