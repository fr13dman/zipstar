package cli_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	root.SetArgs([]string{"a", "-e", "-p", "pw", "-o", archivePath, file})
	if err := root.Execute(); err != nil {
		t.Fatalf("archive: %v", err)
	}

	root = cli.NewRoot(reg)
	root.SetArgs([]string{"u", "-p", "pw", archivePath, "-C", extractDir})
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

func TestArchiveHelpShowsEncryptAndPasswordShorts(t *testing.T) {
	reg := formats.NewRegistry()
	reg.Register(zipfmt.New())
	root := cli.NewRoot(reg)

	archiveCmd, _, err := root.Find([]string{"archive"})
	if err != nil {
		t.Fatal(err)
	}
	enc := archiveCmd.Flags().Lookup("encrypt")
	if enc == nil || enc.Shorthand != "e" {
		t.Fatalf("encrypt shorthand: %#v", enc)
	}
	pw := archiveCmd.Flags().Lookup("password")
	if pw == nil || pw.Shorthand != "p" {
		t.Fatalf("password shorthand: %#v", pw)
	}

	var buf bytes.Buffer
	archiveCmd.SetOut(&buf)
	archiveCmd.SetErr(&buf)
	if err := archiveCmd.Help(); err != nil {
		t.Fatal(err)
	}
	help := buf.String()
	for _, want := range []string{"-e, --encrypt", "-p, --password"} {
		if !strings.Contains(help, want) {
			t.Fatalf("archive help missing %q:\n%s", want, help)
		}
	}
}

func TestRootHelpPointsToSubcommandHelp(t *testing.T) {
	reg := formats.NewRegistry()
	reg.Register(zipfmt.New())
	root := cli.NewRoot(reg)

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	if err := root.Help(); err != nil {
		t.Fatal(err)
	}
	help := buf.String()
	for _, want := range []string{"zipstar archive --help", "-e/--encrypt", "-p/--password"} {
		if !strings.Contains(help, want) {
			t.Fatalf("root help missing %q:\n%s", want, help)
		}
	}
}
