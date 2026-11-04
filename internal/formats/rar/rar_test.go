package rar_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fr13dman/zipstar-cli/internal/formats"
	rarfmt "github.com/fr13dman/zipstar-cli/internal/formats/rar"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join("testdata", name)
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	return p
}

// forceDecodeBackend disables UnRAR so tests exercise the rardecode path.
func forceDecodeBackend(t *testing.T) {
	t.Helper()
	t.Setenv("ZIPSTAR_UNRAR", "off")
}

func TestArchiveUnsupported(t *testing.T) {
	f := rarfmt.New()
	err := f.Archive(context.Background(), formats.ArchiveOpts{
		Output: filepath.Join(t.TempDir(), "out.rar"),
		Files:  []formats.FileEntry{{DiskPath: t.TempDir(), Name: "x"}},
	})
	if !errors.Is(err, formats.ErrArchiveNotSupported) {
		t.Fatalf("want ErrArchiveNotSupported, got %v", err)
	}
}

func TestListAndExtractPlain(t *testing.T) {
	forceDecodeBackend(t)
	cases := []string{"testfile.rar3.rar", "testfile.rar5.rar"}
	f := rarfmt.New()
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			path := fixture(t, name)
			entries, err := f.List(context.Background(), formats.ListOpts{Archive: path})
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || entries[0].Name != "testfile.txt" {
				t.Fatalf("entries: %+v", entries)
			}

			dest := t.TempDir()
			if err := f.Unarchive(context.Background(), formats.UnarchiveOpts{
				Archive: path,
				DestDir: dest,
			}); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(dest, "testfile.txt"))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != "Testing 123\n" {
				t.Fatalf("got %q", got)
			}
		})
	}
}

func TestExtractSolid(t *testing.T) {
	forceDecodeBackend(t)
	f := rarfmt.New()
	path := fixture(t, "solid.rar")
	dest := t.TempDir()
	if err := f.Unarchive(context.Background(), formats.UnarchiveOpts{
		Archive: path,
		DestDir: dest,
	}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dest, "exe", "test.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 45056 {
		t.Fatalf("size %d", info.Size())
	}
}

func TestExtractEncrypted(t *testing.T) {
	forceDecodeBackend(t)
	f := rarfmt.New()
	path := fixture(t, "encrypted_junrar.rar")

	if err := f.Unarchive(context.Background(), formats.UnarchiveOpts{
		Archive: path,
		DestDir: t.TempDir(),
	}); err == nil {
		t.Fatal("expected password required")
	}

	if err := f.Unarchive(context.Background(), formats.UnarchiveOpts{
		Archive:  path,
		DestDir:  t.TempDir(),
		Password: "wrong",
	}); err == nil {
		t.Fatal("expected wrong password error")
	}

	dest := t.TempDir()
	if err := f.Unarchive(context.Background(), formats.UnarchiveOpts{
		Archive:  path,
		DestDir:  dest,
		Password: "junrar",
	}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "file1.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "file1\n" {
		t.Fatalf("got %q", got)
	}
}

func TestListEncryptedFilesOnlyWithoutPassword(t *testing.T) {
	forceDecodeBackend(t)
	f := rarfmt.New()
	entries, err := f.List(context.Background(), formats.ListOpts{
		Archive: fixture(t, "encrypted_files_only.rar"),
	})
	if err != nil {
		t.Fatal(err)
	}
	var sawEncrypted bool
	for _, e := range entries {
		if e.Encrypted {
			sawEncrypted = true
			break
		}
	}
	if !sawEncrypted {
		t.Fatal("expected Encrypted entries without password")
	}
}

func TestListHeaderEncryptedRequiresPassword(t *testing.T) {
	forceDecodeBackend(t)
	f := rarfmt.New()
	_, err := f.List(context.Background(), formats.ListOpts{
		Archive: fixture(t, "encrypted_files_and_header.rar"),
	})
	if err == nil {
		t.Fatal("expected error without password")
	}
	if !rarfmt.NeedsPassword(err) && !strings.Contains(strings.ToLower(err.Error()), "encrypted") {
		t.Fatalf("want password/encrypted signal, got %v", err)
	}

	entries, err := f.List(context.Background(), formats.ListOpts{
		Archive:  fixture(t, "encrypted_files_and_header.rar"),
		Password: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected entries with password")
	}
}

func TestExtractRAR5Encrypted(t *testing.T) {
	forceDecodeBackend(t)
	f := rarfmt.New()
	dest := t.TempDir()
	if err := f.Unarchive(context.Background(), formats.UnarchiveOpts{
		Archive:  fixture(t, "rar5_encrypted_files_only.rar"),
		DestDir:  dest,
		Password: "test",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "jpg", "test.jpg")); err != nil {
		t.Fatal(err)
	}
}

func TestMultiVolume(t *testing.T) {
	forceDecodeBackend(t)
	f := rarfmt.New()
	dest := t.TempDir()
	if err := f.Unarchive(context.Background(), formats.UnarchiveOpts{
		Archive: fixture(t, filepath.Join("multivol", "multi.part01.rar")),
		DestDir: dest,
	}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dest, "exe", "test.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 45056 {
		t.Fatalf("size %d", info.Size())
	}
}

func TestRefuseZipSlip(t *testing.T) {
	forceDecodeBackend(t)
	f := rarfmt.New()
	err := f.Unarchive(context.Background(), formats.UnarchiveOpts{
		Archive: fixture(t, "parent_dir.rar"),
		DestDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected zip-slip refusal")
	}
	if !strings.Contains(err.Error(), "unsafe") && !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOverwriteRequiresForce(t *testing.T) {
	forceDecodeBackend(t)
	f := rarfmt.New()
	path := fixture(t, "testfile.rar5.rar")
	dest := t.TempDir()
	if err := f.Unarchive(context.Background(), formats.UnarchiveOpts{
		Archive: path,
		DestDir: dest,
	}); err != nil {
		t.Fatal(err)
	}
	err := f.Unarchive(context.Background(), formats.UnarchiveOpts{
		Archive: path,
		DestDir: dest,
		Force:   false,
	})
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("want force error, got %v", err)
	}
	if err := f.Unarchive(context.Background(), formats.UnarchiveOpts{
		Archive: path,
		DestDir: dest,
		Force:   true,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestNameAndExtensions(t *testing.T) {
	f := rarfmt.New()
	if f.Name() != "rar" {
		t.Fatalf("name %q", f.Name())
	}
	if len(f.Extensions()) != 1 || f.Extensions()[0] != ".rar" {
		t.Fatalf("exts %#v", f.Extensions())
	}
}

func requireUnrar(t *testing.T) string {
	t.Helper()
	// Clear override so third_party / PATH discovery works.
	t.Setenv("ZIPSTAR_UNRAR", "")
	bin, ok := rarfmt.FindUnrar()
	if !ok {
		t.Skip("UnRAR binary not available; run ./scripts/fetch-unrar.sh")
	}
	return bin
}

func TestUnrarExtractPlain(t *testing.T) {
	_ = requireUnrar(t)
	f := rarfmt.New()
	dest := t.TempDir()
	if err := f.Unarchive(context.Background(), formats.UnarchiveOpts{
		Archive: fixture(t, "testfile.rar5.rar"),
		DestDir: dest,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "testfile.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "Testing 123\n" {
		t.Fatalf("got %q", got)
	}
}

func TestUnrarRefuseZipSlip(t *testing.T) {
	_ = requireUnrar(t)
	f := rarfmt.New()
	err := f.Unarchive(context.Background(), formats.UnarchiveOpts{
		Archive: fixture(t, "parent_dir.rar"),
		DestDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected zip-slip refusal")
	}
	if !strings.Contains(err.Error(), "unsafe") && !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnrarEncryptedAndOverwrite(t *testing.T) {
	_ = requireUnrar(t)
	f := rarfmt.New()
	path := fixture(t, "encrypted_junrar.rar")
	dest := t.TempDir()
	if err := f.Unarchive(context.Background(), formats.UnarchiveOpts{
		Archive:  path,
		DestDir:  dest,
		Password: "junrar",
	}); err != nil {
		t.Fatal(err)
	}
	err := f.Unarchive(context.Background(), formats.UnarchiveOpts{
		Archive:  path,
		DestDir:  dest,
		Password: "junrar",
		Force:    false,
	})
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("want force error, got %v", err)
	}
}

func TestFindUnrarOff(t *testing.T) {
	t.Setenv("ZIPSTAR_UNRAR", "off")
	if _, ok := rarfmt.FindUnrar(); ok {
		t.Fatal("expected FindUnrar disabled")
	}
}
