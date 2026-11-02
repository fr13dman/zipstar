package formats_test

import (
	"context"
	"testing"

	"github.com/fr13dman/zipstar-cli/internal/formats"
)

type stub struct {
	name string
	exts []string
}

func (s stub) Name() string                         { return s.name }
func (s stub) Extensions() []string                 { return s.exts }
func (s stub) Archive(context.Context, formats.ArchiveOpts) error {
	return nil
}
func (s stub) Unarchive(context.Context, formats.UnarchiveOpts) error {
	return nil
}
func (s stub) List(context.Context, formats.ListOpts) ([]formats.Entry, error) {
	return nil, nil
}

func TestRegistryGetAndExt(t *testing.T) {
	reg := formats.NewRegistry()
	reg.Register(stub{name: "zip", exts: []string{".zip"}})

	f, ok := reg.Get("ZIP")
	if !ok || f.Name() != "zip" {
		t.Fatalf("Get failed: %v %v", f, ok)
	}
	f, ok = reg.ByExtension(".ZIP")
	if !ok || f.Name() != "zip" {
		t.Fatalf("ByExtension failed: %v %v", f, ok)
	}
}

func TestRegistryDuplicatePanics(t *testing.T) {
	reg := formats.NewRegistry()
	reg.Register(stub{name: "zip", exts: []string{".zip"}})
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	reg.Register(stub{name: "zip2", exts: []string{".zip"}})
}
