package picker

import (
	"os"
	"path/filepath"
	"strings"
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

// pickerTree builds root/{pack.zip, sub/{inner.zip, notes.txt, deep/}}.
func pickerTree(t *testing.T) (root, sub string) {
	t.Helper()
	root = t.TempDir()
	sub = filepath.Join(root, "sub")
	for _, d := range []string{sub, filepath.Join(sub, "deep")} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{filepath.Join(root, "pack.zip"), filepath.Join(sub, "inner.zip"), filepath.Join(sub, "notes.txt")} {
		if err := os.WriteFile(f, []byte("PK\x03\x04"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root, sub
}

// typeKeys sends one key message per rune, like a user typing.
func typeKeys(m model, s string) model {
	for _, r := range s {
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = next.(model)
	}
	return m
}

func press(m model, k tea.KeyType) (model, tea.Cmd) {
	next, cmd := m.Update(tea.KeyMsg{Type: k})
	return next.(model), cmd
}

func names(es []dirEntry) []string {
	out := make([]string, 0, len(es))
	for _, e := range es {
		out = append(out, e.Name)
	}
	return out
}

func hasName(es []dirEntry, name string) bool {
	for _, e := range es {
		if e.Name == name {
			return true
		}
	}
	return false
}

func TestTypingFolderPathPreviewsIt(t *testing.T) {
	root, sub := pickerTree(t)
	m := newModel(root, []string{".zip"})

	m = typeKeys(m, "sub/")
	if m.viewDir != sub {
		t.Fatalf("viewDir %q want %q", m.viewDir, sub)
	}
	if !hasName(m.filtered, "inner.zip") || hasName(m.filtered, "pack.zip") {
		t.Fatalf("listing should show sub's contents, got %v", names(m.filtered))
	}
	if m.dir != root {
		t.Fatalf("typing must not commit the directory before Enter: dir %q", m.dir)
	}
}

func TestTypingPartialPathFiltersInsideFolder(t *testing.T) {
	root, sub := pickerTree(t)
	m := newModel(root, []string{".zip"})

	m = typeKeys(m, sub+"/inn")
	if m.viewDir != sub {
		t.Fatalf("viewDir %q want %q", m.viewDir, sub)
	}
	if got := names(m.filtered); len(got) != 1 || got[0] != "inner.zip" {
		t.Fatalf("filtered %v want [inner.zip]", got)
	}

	// Enter on a partial path picks the highlighted match.
	m, cmd := press(m, tea.KeyEnter)
	want := filepath.Join(sub, "inner.zip")
	if m.selected != want {
		t.Fatalf("selected %q want %q (status %q)", m.selected, want, m.statusErr)
	}
	if cmd == nil {
		t.Fatal("expected quit")
	}
}

func TestEnterOnTypedFolderSwitchesToIt(t *testing.T) {
	root, sub := pickerTree(t)
	m := newModel(root, []string{".zip"})

	m = typeKeys(m, "sub")
	// "sub" without a separator is a plain filter: still listing root.
	if m.viewDir != root {
		t.Fatalf("viewDir %q want %q", m.viewDir, root)
	}
	m = typeKeys(m, "/")
	m, _ = press(m, tea.KeyEnter)
	if m.dir != sub || m.viewDir != sub {
		t.Fatalf("dir %q viewDir %q want %q", m.dir, m.viewDir, sub)
	}
	if m.filter.Value() != "" {
		t.Fatalf("input should clear after switching, got %q", m.filter.Value())
	}

	// Relative paths now resolve from the new folder.
	m = typeKeys(m, "../")
	if m.viewDir != root {
		t.Fatalf("../ from sub: viewDir %q want %q", m.viewDir, root)
	}
}

func TestPastedPathPreviews(t *testing.T) {
	root, sub := pickerTree(t)
	m := newModel(root, []string{".zip"})
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(sub + "/"), Paste: true})
	m = next.(model)
	if m.viewDir != sub || !hasName(m.filtered, "inner.zip") {
		t.Fatalf("paste: viewDir %q listing %v", m.viewDir, names(m.filtered))
	}
}

func TestTildeAndDotDotPaths(t *testing.T) {
	root, sub := pickerTree(t)
	t.Setenv("HOME", root)

	m := newModel(sub, []string{".zip"})
	m = typeKeys(m, "~/")
	if m.viewDir != root {
		t.Fatalf("~/: viewDir %q want %q", m.viewDir, root)
	}

	m = newModel(sub, []string{".zip"})
	m = typeKeys(m, "..")
	if m.viewDir != root {
		t.Fatalf("..: viewDir %q want %q", m.viewDir, root)
	}
}

func TestTabCompletesHighlighted(t *testing.T) {
	root, sub := pickerTree(t)
	m := newModel(root, []string{".zip"})

	m = typeKeys(m, "su")
	m, _ = press(m, tea.KeyTab)
	if got, want := m.filter.Value(), "sub"+string(filepath.Separator); got != want {
		t.Fatalf("tab on dir: input %q want %q", got, want)
	}
	if m.viewDir != sub {
		t.Fatalf("tab on dir should preview it: viewDir %q", m.viewDir)
	}

	m = typeKeys(m, "inn")
	m, _ = press(m, tea.KeyTab)
	if got, want := m.filter.Value(), filepath.Join("sub", "inner.zip"); got != want {
		t.Fatalf("tab on file: input %q want %q", got, want)
	}
	m, _ = press(m, tea.KeyEnter)
	if want := filepath.Join(sub, "inner.zip"); m.selected != want {
		t.Fatalf("selected %q want %q", m.selected, want)
	}
}

func TestTypingMissingFolderShowsError(t *testing.T) {
	root, _ := pickerTree(t)
	m := newModel(root, []string{".zip"})
	m = typeKeys(m, "nope/")
	if !strings.Contains(m.statusErr, "no such directory") {
		t.Fatalf("statusErr %q", m.statusErr)
	}
	if len(m.filtered) != 0 {
		t.Fatalf("filtered should be empty, got %v", names(m.filtered))
	}
	// Backspacing out of the bad path restores the listing and clears the error.
	for range "nope/" {
		m, _ = press(m, tea.KeyBackspace)
	}
	if m.statusErr != "" || m.viewDir != root || !hasName(m.filtered, "pack.zip") {
		t.Fatalf("after backspace: status %q viewDir %q listing %v", m.statusErr, m.viewDir, names(m.filtered))
	}
}

func TestMultiPartSetShownOnceAndSelectable(t *testing.T) {
	root := t.TempDir()
	for _, n := range []string{
		"29090D43[ddpanda].part1.rar", "29090D43[ddpanda].part2.rar",
		"old.rar", "old.r00", "old.r01",
		"single.rar",
	} {
		if err := os.WriteFile(filepath.Join(root, n), []byte("Rar!"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := newModel(root, []string{".zip", ".rar"})

	parts := map[string]int{}
	for _, e := range m.filtered {
		parts[e.Name] = e.Parts
		if strings.Contains(e.Name, "part2") || strings.HasSuffix(e.Name, ".r00") || strings.HasSuffix(e.Name, ".r01") {
			t.Fatalf("later volumes should be hidden, listed %q", e.Name)
		}
	}
	for name, want := range map[string]int{"29090D43[ddpanda].part1.rar": 2, "old.rar": 3, "single.rar": 0} {
		got, listed := parts[name]
		if !listed || got != want {
			t.Fatalf("%s: listed=%v parts=%d want %d (listing %v)", name, listed, got, want, names(m.filtered))
		}
	}
	if !strings.Contains(m.View(), "(2 parts)") {
		t.Fatalf("view should label the set:\n%s", m.View())
	}

	// Filtering then Enter selects the set's first volume.
	m = typeKeys(m, "ddpanda")
	m, cmd := press(m, tea.KeyEnter)
	if want := filepath.Join(root, "29090D43[ddpanda].part1.rar"); m.selected != want {
		t.Fatalf("selected %q want %q", m.selected, want)
	}
	if cmd == nil {
		t.Fatal("expected quit")
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
