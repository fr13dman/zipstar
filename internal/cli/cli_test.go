package cli_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/fr13dman/zipstar-cli/internal/cli"
	"github.com/fr13dman/zipstar-cli/internal/formats"
	zipfmt "github.com/fr13dman/zipstar-cli/internal/formats/zip"
)

func TestArchiveUnarchiveCLI(t *testing.T) {
	reg := formats.NewRegistry()
	reg.Register(zipfmt.New())

	src := t.TempDir()
	file := filepath.Join(src, "hello.txt")
	if err := os.WriteFile(file, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()
	archivePath := filepath.Join(outDir, "hello.zip")
	extractDir := filepath.Join(outDir, "ex")

	root := cli.NewRoot(reg)
	root.SetArgs([]string{"a", "-o", archivePath, file})
	if err := root.Execute(); err != nil {
		t.Fatalf("archive: %v", err)
	}

	root = cli.NewRoot(reg)
	root.SetArgs([]string{"u", archivePath, "-C", extractDir})
	if err := root.Execute(); err != nil {
		t.Fatalf("unarchive: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(extractDir, "hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hi" {
		t.Fatalf("got %q", got)
	}
}

func TestEncryptArchiveCLI(t *testing.T) {
	reg := formats.NewRegistry()
	reg.Register(zipfmt.New())

	src := t.TempDir()
	file := filepath.Join(src, "secret.txt")
	if err := os.WriteFile(file, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "secret.zip")
	extractDir := t.TempDir()

	root := cli.NewRoot(reg)
	root.SetArgs([]string{"a", "--encrypt", "--password", "pw", "-o", archivePath, file})
	if err := root.Execute(); err != nil {
		t.Fatalf("archive: %v", err)
	}

	root = cli.NewRoot(reg)
	root.SetArgs([]string{"u", "--password", "pw", archivePath, "-C", extractDir})
	if err := root.Execute(); err != nil {
		t.Fatalf("unarchive: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(extractDir, "secret.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "secret" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatMismatchExit(t *testing.T) {
	reg := formats.NewRegistry()
	reg.Register(zipfmt.New())
	root := cli.NewRoot(reg)
	root.SetArgs([]string{"a", "--format", "zip", "-o", "out.tar", t.TempDir()})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	var ee *cli.ExitError
	if !errors.As(err, &ee) || ee.Code != cli.ExitUsage {
		t.Fatalf("want usage exit, got %v", err)
	}
}

func TestListCLI(t *testing.T) {
	reg := formats.NewRegistry()
	reg.Register(zipfmt.New())

	src := t.TempDir()
	file := filepath.Join(src, "a.txt")
	if err := os.WriteFile(file, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "a.zip")

	root := cli.NewRoot(reg)
	root.SetArgs([]string{"a", "-o", archivePath, file})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	root = cli.NewRoot(reg)
	root.SetArgs([]string{"l", "-v", archivePath})
	if err := root.Execute(); err != nil {
		t.Fatalf("list: %v", err)
	}
}
