package picker

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestApplyFilter(t *testing.T) {
	m := model{
		entries: []dirEntry{
			{Name: "..", IsDir: true},
			{Name: "a.zip", IsArchive: true},
			{Name: "notes.txt"},
			{Name: "b.zip", IsArchive: true},
			{Name: "src", IsDir: true},
		},
		exts: map[string]struct{}{".zip": {}},
	}
	m.filter.SetValue("zip")
	m.applyFilter()
	if len(m.filtered) != 2 {
		t.Fatalf("want 2 zip matches, got %d", len(m.filtered))
	}
	m.filter.SetValue("")
	m.applyFilter()
	if len(m.filtered) != 5 {
		t.Fatalf("want all entries, got %d", len(m.filtered))
	}
}

func TestReloadAndDrill(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(root, "pack.zip")
	if err := os.WriteFile(zipPath, []byte("PK\x03\x04"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "inner.zip"), []byte("PK\x03\x04"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModel(root, []string{".zip"})
	foundZip := false
	foundSub := false
	for _, e := range m.entries {
		if e.Name == "pack.zip" && e.IsArchive {
			foundZip = true
		}
		if e.Name == "sub" && e.IsDir {
			foundSub = true
		}
	}
	if !foundZip || !foundSub {
		t.Fatalf("listing missing entries: %#v", m.entries)
	}

	// Select sub directory via cursor + enter.
	for i, e := range m.filtered {
		if e.Name == "sub" {
			m.cursor = i
			break
		}
	}
	next, _ := m.handleEnter()
	m = next.(model)
	if m.dir != sub {
		t.Fatalf("dir = %q, want %q", m.dir, sub)
	}
	foundInner := false
	for _, e := range m.entries {
		if e.Name == "inner.zip" && e.IsArchive {
			foundInner = true
		}
	}
	if !foundInner {
		t.Fatalf("inner.zip not listed: %#v", m.entries)
	}
}

func TestSelectArchive(t *testing.T) {
	root := t.TempDir()
	zipPath := filepath.Join(root, "pack.zip")
	if err := os.WriteFile(zipPath, []byte("PK\x03\x04"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := newModel(root, []string{".zip"})
	for i, e := range m.filtered {
		if e.Name == "pack.zip" {
			m.cursor = i
			break
		}
	}
	next, cmd := m.handleEnter()
	m = next.(model)
	if m.selected == "" {
		t.Fatal("expected selection")
	}
	want, _ := filepath.Abs(zipPath)
	if m.selected != want {
		t.Fatalf("selected %q want %q", m.selected, want)
	}
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
}

func TestCancelKey(t *testing.T) {
	m := newModel(t.TempDir(), []string{".zip"})
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	out := next.(model)
	if !out.canceled {
		t.Fatal("expected canceled")
	}
	if cmd == nil {
		t.Fatal("expected quit")
	}
}

func TestRejectNonArchiveFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := newModel(root, []string{".zip"})
	for i, e := range m.filtered {
		if e.Name == "notes.txt" {
			m.cursor = i
			break
		}
	}
	next, _ := m.handleEnter()
	out := next.(model)
	if out.selected != "" {
		t.Fatal("should not select non-archive")
	}
	if out.statusErr == "" {
		t.Fatal("expected status error")
	}
}

func TestPathJump(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "d")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(sub, "x.zip")
	if err := os.WriteFile(zipPath, []byte("PK\x03\x04"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := newModel(root, []string{".zip"})
	m.filter.SetValue(zipPath)
	next, _ := m.handleEnter()
	out := next.(model)
	want, _ := filepath.Abs(zipPath)
	if out.selected != want {
		t.Fatalf("selected %q want %q", out.selected, want)
	}
}
