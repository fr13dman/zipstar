// Package picker provides an interactive Bubble Tea archive path picker.
package picker

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// Options configures the path picker.
type Options struct {
	// Extensions are selectable archive suffixes including the dot (e.g. ".zip").
	Extensions []string
	// StartDir is the initial directory; empty means the process working directory.
	StartDir string
}

// Run launches the interactive picker and returns an absolute archive path.
// Returns ErrCanceled when the user quits without selecting.
func Run(opts Options) (string, error) {
	start := opts.StartDir
	if start == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		start = cwd
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("start directory is not a directory: %s", abs)
	}

	m := newModel(abs, opts.Extensions)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	out, ok := final.(model)
	if !ok {
		return "", fmt.Errorf("unexpected picker model type")
	}
	if out.canceled || out.selected == "" {
		return "", ErrCanceled
	}
	return out.selected, nil
}
