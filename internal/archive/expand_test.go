package archive

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandInputsGlob(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.rar", "b.rar", "c.zip"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, msgs, err := ExpandInputs([]string{filepath.Join(dir, "*.rar")})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	if len(msgs) != 1 {
		t.Fatalf("msgs %v", msgs)
	}
}

func TestExpandInputsNoMatch(t *testing.T) {
	_, _, err := ExpandInputs([]string{filepath.Join(t.TempDir(), "*.rar")})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExpandInputsPlain(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.rar")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, msgs, err := ExpandInputs([]string{p})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != filepath.Clean(p) {
		t.Fatalf("got %v", got)
	}
	if len(msgs) != 0 {
		t.Fatalf("unexpected msgs %v", msgs)
	}
}
