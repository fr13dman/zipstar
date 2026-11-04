package rar

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const unrarEnv = "ZIPSTAR_UNRAR"

// FindUnrar returns the path to an UnRAR binary, or ("", false) if none is available.
//
// Search order:
//  1. ZIPSTAR_UNRAR
//  2. unrar / unrar.exe / unrar-<goos>-<goarch>[.exe] next to this executable
//  3. third_party/unrar/<goos>_<goarch>/unrar[.exe] (walk up from cwd and executable)
//  4. unrar on PATH
func FindUnrar() (string, bool) {
	if p := os.Getenv(unrarEnv); p != "" {
		// ZIPSTAR_UNRAR=off|0 forces the pure-Go rardecode fallback (tests / debugging).
		if p == "0" || strings.EqualFold(p, "off") || strings.EqualFold(p, "false") {
			return "", false
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, true
		}
	}

	exe, err := os.Executable()
	if err == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		dir := filepath.Dir(exe)
		for _, name := range candidateNames() {
			p := filepath.Join(dir, name)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p, true
			}
		}
		if p := findInThirdParty(dir); p != "" {
			return p, true
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		if p := findInThirdParty(cwd); p != "" {
			return p, true
		}
	}

	if p, err := exec.LookPath(unrarBaseName()); err == nil {
		return p, true
	}
	return "", false
}

func candidateNames() []string {
	base := unrarBaseName()
	plat := fmt.Sprintf("unrar-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		return []string{base, plat + ".exe", plat}
	}
	return []string{base, plat}
}

func unrarBaseName() string {
	if runtime.GOOS == "windows" {
		return "unrar.exe"
	}
	return "unrar"
}

func platformDir() string {
	return runtime.GOOS + "_" + runtime.GOARCH
}

func findInThirdParty(start string) string {
	dir := start
	rel := filepath.Join("third_party", "unrar", platformDir(), unrarBaseName())
	for i := 0; i < 8; i++ {
		p := filepath.Join(dir, rel)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
