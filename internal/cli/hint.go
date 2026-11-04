package cli

import (
	"fmt"
	"os"
)

// printSuccessHint writes a one-line human hint to stderr when stdin is a TTY.
func printSuccessHint(format string, args ...any) {
	if !stdinIsTTY() {
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
