package zip_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fr13dman/zipstar-cli/internal/formats"
	zipfmt "github.com/fr13dman/zipstar-cli/internal/formats/zip"
)

func TestArchiveUnarchiveRoundTrip(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "out.zip")

	file := filepath.Join(src, "hello.txt")
	content := []byte("hello zipstar")
	if err := os.WriteFile(file, content, 0o640); err != nil {
		t.Fatal(err)
	}
	mod := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(file, mod, mod); err != nil {
		t.Fatal(err)
	}

	f := zipfmt.New()
	err := f.Archive(context.Background(), formats.ArchiveOpts{
		Output: archivePath,
		Files: []formats.FileEntry{
			{DiskPath: file, Name: "hello.txt"},
		},
		Level: 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = f.Unarchive(context.Background(), formats.UnarchiveOpts{
		Archive: archivePath,
		DestDir: dst,
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dst, "hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("content mismatch: %q", got)
	}

	info, err := os.Stat(filepath.Join(dst, "hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v, want 0640", info.Mode().Perm())
	}
}

func TestSymlinkRoundTrip(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "link.zip")

	target := filepath.Join(src, "target.txt")
	if err := os.WriteFile(target, []byte("t"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(src, "link.txt")
	if err := os.Symlink("target.txt", link); err != nil {
		t.Fatal(err)
	}

	f := zipfmt.New()
	if err := f.Archive(context.Background(), formats.ArchiveOpts{
		Output: archivePath,
		Files: []formats.FileEntry{
			{DiskPath: link, Name: "link.txt"},
			{DiskPath: target, Name: "target.txt"},
		},
		Level: 5,
	}); err != nil {
		t.Fatal(err)
	}

	if err := f.Unarchive(context.Background(), formats.UnarchiveOpts{
		Archive: archivePath,
		DestDir: dst,
	}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Lstat(filepath.Join(dst, "link.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("expected symlink")
	}
	got, err := os.Readlink(filepath.Join(dst, "link.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "target.txt" {
		t.Fatalf("symlink target = %q", got)
	}
}

func TestOverwriteRefused(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "out.zip")
	if err := os.WriteFile(archivePath, []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(file, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := zipfmt.New()
	err := f.Archive(context.Background(), formats.ArchiveOpts{
		Output: archivePath,
		Files:  []formats.FileEntry{{DiskPath: file, Name: "a.txt"}},
		Level:  5,
		Force:  false,
	})
	if err == nil {
		t.Fatal("expected overwrite error")
	}

	if err := f.Archive(context.Background(), formats.ArchiveOpts{
		Output: archivePath,
		Files:  []formats.FileEntry{{DiskPath: file, Name: "a.txt"}},
		Level:  5,
		Force:  true,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestListAndEncrypt(t *testing.T) {
	src := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "secret.zip")
	file := filepath.Join(src, "secret.txt")
	if err := os.WriteFile(file, []byte("top secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	f := zipfmt.New()
	if err := f.Archive(context.Background(), formats.ArchiveOpts{
		Output:   archivePath,
		Files:    []formats.FileEntry{{DiskPath: file, Name: "secret.txt"}},
		Level:    6,
		Password: "s3cret",
	}); err != nil {
		t.Fatal(err)
	}

	entries, err := f.List(context.Background(), formats.ListOpts{Archive: archivePath})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !entries[0].Encrypted {
		t.Fatalf("expected encrypted entry, got %+v", entries)
	}

	dst := t.TempDir()
	err = f.Unarchive(context.Background(), formats.UnarchiveOpts{
		Archive:  archivePath,
		DestDir:  dst,
		Password: "wrong",
	})
	if err == nil {
		t.Fatal("expected wrong-password error")
	}

	if err := f.Unarchive(context.Background(), formats.UnarchiveOpts{
		Archive:  archivePath,
		DestDir:  dst,
		Password: "s3cret",
	}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dst, "secret.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "top secret" {
		t.Fatalf("got %q", got)
	}
}

func TestFolderArchive(t *testing.T) {
	src := t.TempDir()
	project := filepath.Join(src, "project")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	empty := filepath.Join(project, "empty")
	if err := os.Mkdir(empty, 0o755); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(t.TempDir(), "project.zip")
	f := zipfmt.New()
	files := []formats.FileEntry{
		{DiskPath: project, Name: "project"},
		{DiskPath: filepath.Join(project, "a.txt"), Name: "project/a.txt"},
		{DiskPath: empty, Name: "project/empty"},
	}
	if err := f.Archive(context.Background(), formats.ArchiveOpts{
		Output: archivePath,
		Files:  files,
		Level:  1,
	}); err != nil {
		t.Fatal(err)
	}

	dst := t.TempDir()
	if err := f.Unarchive(context.Background(), formats.UnarchiveOpts{
		Archive: archivePath,
		DestDir: dst,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, "project", "empty")); err != nil {
		t.Fatal(err)
	}
}

func TestLevelBounds(t *testing.T) {
	f := zipfmt.New()
	err := f.Archive(context.Background(), formats.ArchiveOpts{
		Output: filepath.Join(t.TempDir(), "x.zip"),
		Level:  10,
	})
	if err == nil {
		t.Fatal("expected level error")
	}
}
