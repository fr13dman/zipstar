package detect

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func writeZip(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("hi")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSniffFormat_Zip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "blob")
	writeZip(t, path)
	name, ok, err := SniffFormat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || name != "zip" {
		t.Fatalf("got %q ok=%v", name, ok)
	}
}

func TestSniffFormat_Negative(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "blob")
	if err := os.WriteFile(path, []byte("not an archive"), 0o644); err != nil {
		t.Fatal(err)
	}
	name, ok, err := SniffFormat(path)
	if err != nil {
		t.Fatal(err)
	}
	if ok || name != "" {
		t.Fatalf("expected miss, got %q ok=%v", name, ok)
	}
}

func TestSniffFormatCached(t *testing.T) {
	dir := t.TempDir()
	SetCachePath(filepath.Join(dir, "sniff.json"))
	t.Cleanup(func() { SetCachePath("") })

	path := filepath.Join(dir, "blob")
	writeZip(t, path)

	name, ok, err := SniffFormatCached(path)
	if err != nil || !ok || name != "zip" {
		t.Fatalf("first sniff: name=%q ok=%v err=%v", name, ok, err)
	}

	// Corrupt the file but restore mtime/size impossible easily — change content
	// with same size is hard; instead verify cache hit by checking LookupCache.
	id, err := IdentityOf(path)
	if err != nil {
		t.Fatal(err)
	}
	got, hit := LookupCache(id)
	if !hit || got != "zip" {
		t.Fatalf("cache miss: hit=%v got=%q", hit, got)
	}

	// Stale: rewrite with different size.
	if err := os.WriteFile(path, []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	id2, err := IdentityOf(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, hit := LookupCache(id2); hit {
		t.Fatal("expected stale cache miss")
	}
}
