package archive

import (
	"context"
	"testing"

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
