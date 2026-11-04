package cli_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/fr13dman/zipstar-cli/internal/cli"
	"github.com/fr13dman/zipstar-cli/internal/formats"
	zipfmt "github.com/fr13dman/zipstar-cli/internal/formats/zip"
)

type stubPicker struct {
	path string
	err  error
}

func (s stubPicker) Pick([]string) (string, error) {
	return s.path, s.err
}

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

func TestUnarchiveMissingArgNonTTY(t *testing.T) {
	reg := formats.NewRegistry()
	reg.Register(zipfmt.New())
	root := cli.NewRoot(reg)
	root.SetArgs([]string{"u"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when archive arg missing")
	}
	var ee *cli.ExitError
	if !errors.As(err, &ee) || ee.Code != cli.ExitUsage {
		t.Fatalf("want usage exit, got %v", err)
	}
	if !strings.Contains(ee.Message, "archive path required") {
		t.Fatalf("unexpected message: %s", ee.Message)
	}
}

func TestUnarchiveMissingArgWithPickerStub(t *testing.T) {
	reg := formats.NewRegistry()
	reg.Register(zipfmt.New())

	src := t.TempDir()
	file := filepath.Join(src, "hello.txt")
	if err := os.WriteFile(file, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "hello.zip")
	extractDir := t.TempDir()

	root := cli.NewRoot(reg)
	root.SetArgs([]string{"a", "-o", archivePath, file})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	restoreTTY := cli.SetStdinTTYForTest(true)
	t.Cleanup(restoreTTY)
	restorePicker := cli.SetArchivePickerForTest(stubPicker{path: archivePath})
	t.Cleanup(restorePicker)

	root = cli.NewRoot(reg)
	root.SetArgs([]string{"u", "-C", extractDir})
	if err := root.Execute(); err != nil {
		t.Fatalf("unarchive via picker: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(extractDir, "hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hi" {
		t.Fatalf("got %q", got)
	}
}

func TestArchivePathCompletion(t *testing.T) {
	reg := formats.NewRegistry()
	reg.Register(zipfmt.New())

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.zip"), []byte("PK"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	root := cli.NewRoot(reg)
	var unarchive *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "unarchive" {
			unarchive = c
			break
		}
	}
	if unarchive == nil || unarchive.ValidArgsFunction == nil {
		t.Fatal("unarchive ValidArgsFunction missing")
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	comps, _ := unarchive.ValidArgsFunction(unarchive, nil, "")
	joined := strings.Join(comps, "\n")
	if !strings.Contains(joined, "a.zip") {
		t.Fatalf("expected a.zip in %v", comps)
	}
	if strings.Contains(joined, "b.txt") {
		t.Fatalf("did not expect b.txt in %v", comps)
	}
	foundDir := false
	for _, c := range comps {
		if strings.HasPrefix(c, "sub") && strings.HasSuffix(c, string(filepath.Separator)) {
			foundDir = true
		}
	}
	if !foundDir {
		t.Fatalf("expected sub/ in %v", comps)
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
