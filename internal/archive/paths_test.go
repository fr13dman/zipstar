package archive

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildEntries_FileAndDir(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(file, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(dir, "project")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(subdir, "b.txt")
	if err := os.WriteFile(nested, []byte("yo"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := BuildEntries([]string{file, subdir})
	if err != nil {
		t.Fatal(err)
	}

	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name] = true
	}
	for _, want := range []string{"a.txt", "project", "project/b.txt"} {
		if !names[want] {
			t.Fatalf("missing entry %q in %#v", want, names)
		}
	}
}

func TestBuildEntries_RejectDotDot(t *testing.T) {
	_, err := BuildEntries([]string{"../outside"})
	if err == nil {
		t.Fatal("expected error for .. path")
	}
}

func TestBuildEntries_SymlinkFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink("target.txt", link); err != nil {
		t.Fatal(err)
	}

	entries, err := BuildEntries([]string{link})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "link.txt" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}

func TestBuildEntries_DuplicateBaseNames(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "x")
	b := filepath.Join(dir, "nested")
	if err := os.MkdirAll(b, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(a), []byte("1"), 0o644); err != nil {
		// a is a file named x
	}
	// recreate: a as file
	_ = os.RemoveAll(a)
	if err := os.WriteFile(a, []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(b, "x")
	if err := os.WriteFile(other, []byte("2"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := BuildEntries([]string{a, other})
	if err == nil {
		t.Fatal("expected duplicate archive path error")
	}
}

func TestCheckOverwrite(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "out.zip")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckOverwrite(p, false); err == nil {
		t.Fatal("expected refuse without force")
	}
	if err := CheckOverwrite(p, true); err != nil {
		t.Fatal(err)
	}
}
