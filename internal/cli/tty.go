package cli

import (
	"os"

	"golang.org/x/term"
)

// stdinTTYCheck is overridable in tests.
var stdinTTYCheck = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// stdinIsTTY reports whether stdin is an interactive terminal.
func stdinIsTTY() bool {
	return stdinTTYCheck()
}

// SetStdinTTYForTest overrides TTY detection (tests only).
func SetStdinTTYForTest(isTTY bool) (restore func()) {
	prev := stdinTTYCheck
	stdinTTYCheck = func() bool { return isTTY }
	return func() { stdinTTYCheck = prev }
}
