package cli_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/fr13dman/zipstar-cli/internal/cli"
	"github.com/fr13dman/zipstar-cli/internal/formats"
	rarfmt "github.com/fr13dman/zipstar-cli/internal/formats/rar"
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

// Every flag on every command must have a one-letter shortcut, shown in --help.
func TestAllFlagsHaveShortcuts(t *testing.T) {
	want := map[string]map[string]string{
		"archive":   {"output": "o", "level": "l", "encrypt": "e", "password": "p", "format": "f", "force": "F", "verbose": "v"},
		"unarchive": {"dir": "C", "password": "p", "progress": "P", "format": "f", "force": "F", "verbose": "v"},
		"list":      {"password": "p", "format": "f", "force": "F", "verbose": "v"},
	}
	reg := formats.NewRegistry()
	reg.Register(zipfmt.New())
	root := cli.NewRoot(reg)

	for name, flags := range want {
		cmd, _, err := root.Find([]string{name})
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		if err := cmd.Help(); err != nil {
			t.Fatal(err)
		}
		help := buf.String()

		seen := map[string]bool{}
		check := func(f *pflag.Flag) {
			if f.Name == "help" {
				return
			}
			seen[f.Name] = true
			short, ok := flags[f.Name]
			if !ok {
				t.Errorf("%s: flag --%s missing from expected shortcut table", name, f.Name)
				return
			}
			if f.Shorthand != short {
				t.Errorf("%s: --%s shorthand %q want %q", name, f.Name, f.Shorthand, short)
			}
			if entry := "-" + short + ", --" + f.Name; !strings.Contains(help, entry) {
				t.Errorf("%s help missing %q", name, entry)
			}
		}
		cmd.LocalFlags().VisitAll(check)
		cmd.InheritedFlags().VisitAll(check)
		for flag := range flags {
			if !seen[flag] {
				t.Errorf("%s: expected flag --%s not registered", name, flag)
			}
		}
	}
}

func TestShortFlagsEndToEnd(t *testing.T) {
	reg := formats.NewRegistry()
	reg.Register(zipfmt.New())

	src := t.TempDir()
	file := filepath.Join(src, "a.txt")
	if err := os.WriteFile(file, []byte("short"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "a.zip")
	dest := t.TempDir()

	// -f is honoured: naming the wrong format for a .zip path is a usage error.
	mismatchReg := formats.NewRegistry()
	mismatchReg.Register(zipfmt.New())
	mismatchReg.Register(rarfmt.New())
	root := cli.NewRoot(mismatchReg)
	root.SetArgs([]string{"a", "-f", "rar", "-o", out, file})
	var ee *cli.ExitError
	if err := root.Execute(); !errors.As(err, &ee) || ee.Code != cli.ExitUsage {
		t.Fatalf("-f rar with .zip output: want usage error, got %v", err)
	}

	for _, args := range [][]string{
		{"a", "-f", "zip", "-l", "9", "-o", out, file},
		{"a", "-f", "zip", "-F", "-e", "-p", "pw", "-o", out, file}, // -F overwrites
		{"u", "-f", "zip", "-p", "pw", "-P=false", "-C", dest, out},
		{"u", "-f", "zip", "-p", "pw", "-F", "-P=false", "-C", dest, out}, // -F overwrites extract
		{"l", "-f", "zip", "-v", out},
	} {
		root := cli.NewRoot(reg)
		var errBuf bytes.Buffer
		root.SetErr(&errBuf)
		root.SetArgs(args)
		if err := root.Execute(); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, errBuf.String())
		}
		if args[0] == "u" && strings.Contains(errBuf.String(), "extract [") {
			t.Fatalf("-P=false should disable the bar: %q", errBuf.String())
		}
	}
	got, err := os.ReadFile(filepath.Join(dest, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "short" {
		t.Fatalf("got %q", got)
	}
}

func TestRARUnarchiveAndListCLI(t *testing.T) {
	reg := formats.NewRegistry()
	reg.Register(rarfmt.New())

	archivePath := filepath.Join("..", "formats", "rar", "testdata", "testfile.rar5.rar")
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	extractDir := t.TempDir()

	root := cli.NewRoot(reg)
	root.SetArgs([]string{"u", archivePath, "-C", extractDir})
	if err := root.Execute(); err != nil {
		t.Fatalf("unarchive: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(extractDir, "testfile.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "Testing 123\n" {
		t.Fatalf("got %q", got)
	}

	root = cli.NewRoot(reg)
	root.SetArgs([]string{"l", archivePath})
	if err := root.Execute(); err != nil {
		t.Fatalf("list: %v", err)
	}
}

func TestRARGlobMultiVolumeUnarchiveCLI(t *testing.T) {
	pattern := filepath.Join("..", "formats", "rar", "testdata", "multivol", "multi.part*.rar")

	for _, engine := range []string{"unrar", "rardecode"} {
		for _, progress := range []bool{true, false} {
			name := engine + "/progress=" + map[bool]string{true: "on", false: "off"}[progress]
			t.Run(name, func(t *testing.T) {
				if engine == "unrar" {
					t.Setenv("ZIPSTAR_UNRAR", "")
					if _, ok := rarfmt.FindUnrar(); !ok {
						t.Skip("UnRAR not available; run ./scripts/fetch-unrar.sh")
					}
				} else {
					t.Setenv("ZIPSTAR_UNRAR", "off")
				}

				reg := formats.NewRegistry()
				reg.Register(rarfmt.New())
				extractDir := t.TempDir()

				// Progress is on by default; the off case must opt out explicitly.
				args := []string{"u", pattern, "-C", extractDir}
				if !progress {
					args = append(args, "--progress=false")
				}
				root := cli.NewRoot(reg)
				var errBuf bytes.Buffer
				root.SetErr(&errBuf)
				root.SetArgs(args)
				if err := root.Execute(); err != nil {
					t.Fatalf("unarchive: %v\nstderr=%s", err, errBuf.String())
				}

				for member, size := range map[string]int64{
					"exe/test.exe": 45056,
					"jpg/test.jpg": 40372,
					"тест.txt":     15498,
				} {
					info, err := os.Stat(filepath.Join(extractDir, member))
					if err != nil {
						t.Fatalf("%s missing — all volumes should unpack into %s: %v", member, extractDir, err)
					}
					if info.Size() != size {
						t.Fatalf("%s size %d want %d", member, info.Size(), size)
					}
				}

				stderr := errBuf.String()
				for _, want := range []string{
					"extracting 1 archive",
					"all members unpack into " + extractDir,
					"reading volume 1/6: multi.part01.rar",
					"reading volume 6/6: multi.part06.rar",
					"(6 file(s) from 6 volumes)",
				} {
					if !strings.Contains(stderr, want) {
						t.Fatalf("stderr missing %q:\n%s", want, stderr)
					}
				}
				hasBar := strings.Contains(stderr, "extract [")
				if progress && (!hasBar || !strings.Contains(stderr, "6/6")) {
					t.Fatalf("default run should draw a bar reaching 6/6:\n%s", stderr)
				}
				if !progress && hasBar {
					t.Fatalf("--progress=false should not draw a bar:\n%s", stderr)
				}
			})
		}
	}
}

func TestRARArchiveUnsupportedCLI(t *testing.T) {
	reg := formats.NewRegistry()
	reg.Register(rarfmt.New())

	src := t.TempDir()
	file := filepath.Join(src, "a.txt")
	if err := os.WriteFile(file, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "out.rar")

	root := cli.NewRoot(reg)
	root.SetArgs([]string{"a", "-o", out, file})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	var ee *cli.ExitError
	if !errors.As(err, &ee) || ee.Code != cli.ExitUsage {
		t.Fatalf("want usage exit, got %v", err)
	}
	if !strings.Contains(ee.Message, "not supported") {
		t.Fatalf("message: %q", ee.Message)
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
	for _, want := range []string{
		"zipstar archive --help", "-e/--encrypt", "-p/--password",
		"-f/--format", "-F/--force", "-l/--level", "-P/--progress",
		"-f, --format", "-F, --force", "-v, --verbose",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("root help missing %q:\n%s", want, help)
		}
	}
}
