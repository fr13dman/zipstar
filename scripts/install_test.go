package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type sandbox struct {
	home   string
	prefix string
	env    []string
}

// newSandbox isolates install.sh: HOME, SHELL and PATH point at temp/system
// locations, while Go's caches are kept so the build needs no network.
func newSandbox(t *testing.T, prefixOnPath bool) sandbox {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("install.sh targets macOS and Linux; Windows uses install.ps1")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go not on PATH")
	}
	tmp := t.TempDir()
	s := sandbox{home: filepath.Join(tmp, "home"), prefix: filepath.Join(tmp, "bin")}
	if err := os.MkdirAll(s.home, 0o755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Dir(goBin) + ":/usr/bin:/bin:/usr/sbin:/sbin"
	if prefixOnPath {
		path = s.prefix + ":" + path
	}
	s.env = []string{"HOME=" + s.home, "SHELL=/bin/zsh", "PATH=" + path}
	for _, k := range []string{"GOCACHE", "GOMODCACHE", "GOPATH", "GOTOOLCHAIN"} {
		out, err := exec.Command("go", "env", k).Output()
		if err != nil {
			t.Fatal(err)
		}
		s.env = append(s.env, k+"="+strings.TrimSpace(string(out)))
	}
	return s
}

func (s sandbox) run(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("bash", append([]string{"install.sh"}, args...)...)
	cmd.Env = s.env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install.sh %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func unrarName() string { return "unrar-" + runtime.GOOS + "-" + runtime.GOARCH }

func fakeUnrar(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "internal", "formats", "rar", "testdata", "fake_unrar.sh"))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestInstallAddsToPathAndUninstallReverts(t *testing.T) {
	s := newSandbox(t, false)
	rc := filepath.Join(s.home, ".zshrc")
	if err := os.WriteFile(rc, []byte("# user config\nalias ll='ls -l'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := s.run(t, "--prefix", s.prefix, "--unrar", fakeUnrar(t))
	if !strings.Contains(out, "added "+s.prefix+" to PATH in "+rc) {
		t.Fatalf("expected PATH edit message:\n%s", out)
	}
	zipstar := filepath.Join(s.prefix, "zipstar")
	if info, err := os.Stat(zipstar); err != nil || info.Mode()&0o111 == 0 {
		t.Fatalf("zipstar not installed executable: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.prefix, unrarName())); err != nil {
		t.Fatalf("UnRAR not installed beside zipstar: %v", err)
	}
	license := filepath.Join(filepath.Dir(s.prefix), "share", "zipstar", "UNRAR-LICENSE.txt")
	if _, err := os.Stat(license); err != nil {
		t.Fatalf("UnRAR license not installed: %v", err)
	}

	// The installer stamps `git describe` into the binary.
	described, err := exec.Command("git", "-C", "..", "describe", "--tags", "--always", "--dirty").Output()
	if err != nil {
		t.Fatalf("git describe: %v", err)
	}
	wantVersion := "zipstar " + strings.TrimSpace(string(described))
	got, err := exec.Command(zipstar, "-V").Output()
	if err != nil {
		t.Fatalf("zipstar -V: %v", err)
	}
	if first := strings.SplitN(string(got), "\n", 2)[0]; first != wantVersion {
		t.Fatalf("installed version %q want %q", first, wantVersion)
	}
	if !strings.Contains(out, "installed zipstar "+strings.TrimPrefix(wantVersion, "zipstar ")+" -> ") {
		t.Fatalf("installer should report the version:\n%s", out)
	}

	// Re-running must not duplicate the PATH block.
	s.run(t, "--prefix", s.prefix, "--unrar", fakeUnrar(t))
	rcBody, _ := os.ReadFile(rc)
	if n := strings.Count(string(rcBody), "# >>> zipstar PATH >>>"); n != 1 {
		t.Fatalf("PATH block present %d times:\n%s", n, rcBody)
	}
	if !strings.Contains(string(rcBody), `export PATH="`+s.prefix+`:$PATH"`) {
		t.Fatalf("rc missing export line:\n%s", rcBody)
	}

	// The block works: a fresh zsh with the rc finds the installed zipstar.
	if zsh, err := exec.LookPath("zsh"); err == nil {
		cmd := exec.Command(zsh, "-c", "source "+rc+" && command -v zipstar")
		cmd.Env = s.env
		got, err := cmd.Output()
		if err != nil || strings.TrimSpace(string(got)) != zipstar {
			t.Fatalf("zipstar not resolved via rc PATH: %q %v", got, err)
		}
	}

	// The installed binary uses the UnRAR copied beside it (found next to the
	// executable, not via PATH or the repo), so extraction goes through it.
	work := t.TempDir()
	archive := filepath.Join(work, "dummy.rar")
	if err := os.WriteFile(archive, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(zipstar, "u", "-P=false", "-C", filepath.Join(work, "out"), archive)
	cmd.Dir = work
	cmd.Env = s.env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("installed zipstar extract: %v\n%s", err, out)
	}
	for _, name := range []string{"a.txt", "b.txt"} {
		if _, err := os.Stat(filepath.Join(work, "out", name)); err != nil {
			t.Fatalf("bundled UnRAR not used (missing %s): %v", name, err)
		}
	}

	s.run(t, "--prefix", s.prefix, "--uninstall")
	for _, p := range []string{zipstar, filepath.Join(s.prefix, unrarName()), filepath.Dir(license)} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("uninstall left %s behind", p)
		}
	}
	rcBody, _ = os.ReadFile(rc)
	if want := "# user config\nalias ll='ls -l'\n"; string(rcBody) != want+"\n" && string(rcBody) != want {
		t.Fatalf("uninstall should restore user rc, got:\n%q", rcBody)
	}
}

func TestInstallLeavesRcAloneWhenAlreadyOnPath(t *testing.T) {
	s := newSandbox(t, true)
	out := s.run(t, "--prefix", s.prefix, "--no-unrar")
	if !strings.Contains(out, s.prefix+" is already on PATH") {
		t.Fatalf("expected already-on-PATH message:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(s.home, ".zshrc")); !os.IsNotExist(err) {
		t.Fatal("rc file should not be created when the prefix is already on PATH")
	}
	if _, err := os.Stat(filepath.Join(s.prefix, unrarName())); !os.IsNotExist(err) {
		t.Fatal("--no-unrar should not install UnRAR")
	}
}

func TestInstallRejectsUnknownOption(t *testing.T) {
	s := newSandbox(t, false)
	cmd := exec.Command("bash", "install.sh", "--bogus")
	cmd.Env = s.env
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "unknown option: --bogus") {
		t.Fatalf("want unknown-option error, got %v\n%s", err, out)
	}
}

// install.ps1 can only run on Windows; where pwsh exists, at least make sure
// it parses.
func TestInstallPs1Parses(t *testing.T) {
	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		t.Skip("pwsh not installed")
	}
	script := `$errs = $null; [void][System.Management.Automation.Language.Parser]::ParseFile((Resolve-Path 'install.ps1'), [ref]$null, [ref]$errs); if ($errs) { $errs | ForEach-Object { $_.Message }; exit 1 }`
	if out, err := exec.Command(pwsh, "-NoProfile", "-Command", script).CombinedOutput(); err != nil {
		t.Fatalf("install.ps1 parse errors: %v\n%s", err, out)
	}
}
