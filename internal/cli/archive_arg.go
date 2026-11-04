package cli

import (
	"errors"
	"fmt"

	"github.com/fr13dman/zipstar-cli/internal/cli/picker"
	"github.com/fr13dman/zipstar-cli/internal/formats"
)

// ArchivePicker selects an archive path interactively.
type ArchivePicker interface {
	Pick(extensions []string) (string, error)
}

// defaultArchivePicker launches the Bubble Tea path picker.
type defaultArchivePicker struct{}

func (defaultArchivePicker) Pick(extensions []string) (string, error) {
	return picker.Run(picker.Options{Extensions: extensions})
}

// archivePicker is overridable in tests.
var archivePicker ArchivePicker = defaultArchivePicker{}

// SetArchivePickerForTest replaces the interactive picker (tests only).
func SetArchivePickerForTest(p ArchivePicker) (restore func()) {
	prev := archivePicker
	archivePicker = p
	return func() { archivePicker = prev }
}

// resolveArchiveArg returns args[0] when present. When missing, opens the
// interactive picker on a TTY; otherwise returns a usage error.
func resolveArchiveArg(args []string, reg *formats.Registry) (string, error) {
	if len(args) >= 1 && args[0] != "" {
		return args[0], nil
	}
	if !stdinIsTTY() {
		return "", Usagef("archive path required: pass <archive> (interactive picker requires a TTY)")
	}
	path, err := archivePicker.Pick(reg.Extensions())
	if err != nil {
		if errors.Is(err, picker.ErrCanceled) {
			return "", Usagef("%v", err)
		}
		return "", fmt.Errorf("%w", err)
	}
	if path == "" {
		return "", Usagef("archive path required")
	}
	return path, nil
}
