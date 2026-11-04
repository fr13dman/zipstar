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

// Real-world names often contain glob characters; an existing path is literal.
func TestExpandInputsLiteralPathWithBrackets(t *testing.T) {
	p := filepath.Join(t.TempDir(), "29090D43[ddpanda].part1.rar")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, msgs, err := ExpandInputs([]string{p})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != p {
		t.Fatalf("got %v want [%s]", got, p)
	}
	if len(msgs) != 0 {
		t.Fatalf("literal path should not report a glob: %v", msgs)
	}
}

func TestExpandInputsBracketGlobStillWorks(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a1.rar", "a2.rar", "a3.rar"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, _, err := ExpandInputs([]string{filepath.Join(dir, "a[12].rar")})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("character-class glob: got %v", got)
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
