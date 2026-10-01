package tui

import (
	"bytes"
	"testing"
)

func TestSplitAtRuneBoundary(t *testing.T) {
	border := []byte("╰──╯") // three bytes per rune
	for cut := 0; cut <= len(border); cut++ {
		complete, rest := splitAtRuneBoundary(border[:cut])
		if !bytes.Equal(append(append([]byte(nil), complete...), rest...), border[:cut]) {
			t.Fatalf("cut %d: parts do not add up", cut)
		}
		if len(rest) >= 3 || (len(rest) > 0 && cut%3 == 0) {
			t.Fatalf("cut %d: rest %q should hold only an incomplete rune", cut, rest)
		}
	}
}
