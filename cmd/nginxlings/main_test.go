package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHomeCursorOutputMovesHomeAfterEachFrame(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	out := homeCursorOutput{f}

	frame := "\x1b[Htitle\r\nbody\x1b[2;0H"
	if n, err := out.Write([]byte(frame)); err != nil || n != len(frame) {
		t.Fatalf("Write(frame) = %d, %v; want %d, nil", n, err, len(frame))
	}
	// Leaving the alternate screen must keep the cursor where the terminal
	// restores it.
	leave := "\x1b[?1049l"
	if _, err := out.Write([]byte(leave)); err != nil {
		t.Fatal(err)
	}
	f.Close()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := frame + "\x1b[H" + leave; string(got) != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
