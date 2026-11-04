package archive

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fr13dman/zipstar-cli/internal/detect"
	"github.com/fr13dman/zipstar-cli/internal/formats"
)

type stubFormat struct {
	name string
	exts []string
}

func (s stubFormat) Name() string                         { return s.name }
func (s stubFormat) Extensions() []string                 { return s.exts }
func (s stubFormat) Archive(context.Context, formats.ArchiveOpts) error {
	return nil
}
func (s stubFormat) Unarchive(context.Context, formats.UnarchiveOpts) error {
	return nil
}
func (s stubFormat) List(context.Context, formats.ListOpts) ([]formats.Entry, error) {
	return nil, nil
}

func testRegistry(t *testing.T) *formats.Registry {
	t.Helper()
	reg := formats.NewRegistry()
	reg.Register(stubFormat{name: "zip", exts: []string{".zip"}})
	return reg
}

func TestResolveFormat_InferFromExtension(t *testing.T) {
	reg := testRegistry(t)
	f, err := ResolveFormat(reg, "", "out.zip")
	if err != nil {
		t.Fatal(err)
	}
	if f.Name() != "zip" {
		t.Fatalf("got %s", f.Name())
	}
}

func TestResolveFormat_Explicit(t *testing.T) {
	reg := testRegistry(t)
	f, err := ResolveFormat(reg, "zip", "out.zip")
	if err != nil {
		t.Fatal(err)
	}
	if f.Name() != "zip" {
		t.Fatalf("got %s", f.Name())
	}
}

func TestResolveFormat_MissingExtension(t *testing.T) {
	reg := testRegistry(t)
	_, err := ResolveFormat(reg, "", "out")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveFormat_UnknownExtension(t *testing.T) {
	reg := testRegistry(t)
	_, err := ResolveFormat(reg, "", "out.tar.gz")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveFormat_Mismatch(t *testing.T) {
	reg := testRegistry(t)
	reg.Register(stubFormat{name: "tar", exts: []string{".tar"}})
	_, err := ResolveFormat(reg, "zip", "out.tar")
	if err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestResolveFormat_UnknownFormat(t *testing.T) {
	reg := testRegistry(t)
	_, err := ResolveFormat(reg, "7z", "out.7z")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDeriveOutputName(t *testing.T) {
	f := stubFormat{name: "zip", exts: []string{".zip"}}
	got, err := DeriveOutputName("./project", f)
	if err != nil {
		t.Fatal(err)
	}
	if got != "project.zip" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveFormat_SniffNoExtension(t *testing.T) {
	reg := testRegistry(t)
	dir := t.TempDir()
	detect.SetCachePath(filepath.Join(dir, "sniff.json"))
	t.Cleanup(func() { detect.SetCachePath("") })

	path := filepath.Join(dir, "blob")
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

	got, err := ResolveFormat(reg, "", path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name() != "zip" {
		t.Fatalf("got %s", got.Name())
	}
}

func TestResolveFormat_MismatchStillErrors(t *testing.T) {
	reg := testRegistry(t)
	reg.Register(stubFormat{name: "tar", exts: []string{".tar"}})
	dir := t.TempDir()
	path := filepath.Join(dir, "out.tar")
	// Even if contents are zip, --format zip + .tar extension must mismatch.
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	_ = zw.Close()
	_ = f.Close()

	_, err = ResolveFormat(reg, "zip", path)
	if err == nil {
		t.Fatal("expected mismatch error")
	}
}
