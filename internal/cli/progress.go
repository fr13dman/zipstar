package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"unicode/utf8"
)

const progressNameWidth = 32

// ProgressBar is a simple stderr progress indicator for extract operations.
type ProgressBar struct {
	mu      sync.Mutex
	w       io.Writer
	label   string
	total   int
	current int
	name    string // member currently being extracted
	width   int
	enabled bool
	lastLen int // runes in the last drawn line, for clearing
}

// NewProgressBar creates a bar that writes to stderr when stderr is a TTY
// (or when force is true). total may be 0 for an indeterminate count.
func NewProgressBar(label string, total int, force bool) *ProgressBar {
	enabled := force
	if !enabled {
		if fi, err := os.Stderr.Stat(); err == nil {
			enabled = (fi.Mode() & os.ModeCharDevice) != 0
		}
	}
	return newProgressBar(os.Stderr, label, total, enabled)
}

// NewProgressBarTo is like NewProgressBar but always writes to w (for tests).
func NewProgressBarTo(w io.Writer, label string, total int) *ProgressBar {
	return newProgressBar(w, label, total, true)
}

func newProgressBar(w io.Writer, label string, total int, enabled bool) *ProgressBar {
	return &ProgressBar{
		w:       w,
		label:   label,
		total:   total,
		width:   28,
		enabled: enabled,
	}
}

// Add advances the bar by n completed items and redraws.
func (b *ProgressBar) Add(n int) {
	if b == nil || !b.enabled {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.current += n
	b.drawLocked()
}

// Advance counts one more member and shows name as the one in progress.
func (b *ProgressBar) Advance(name string) {
	if b == nil || !b.enabled {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.current++
	b.name = name
	b.drawLocked()
}

// SetTotal updates the expected total (e.g. after List).
func (b *ProgressBar) SetTotal(total int) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.total = total
	if b.enabled {
		b.drawLocked()
	}
}

// Println prints a status line above the bar, then redraws the bar.
// When the bar is disabled the line is written as-is.
func (b *ProgressBar) Println(msg string) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.enabled {
		fmt.Fprintln(b.w, msg)
		return
	}
	b.clearLocked()
	fmt.Fprintln(b.w, msg)
	b.drawLocked()
}

// Finish clears the bar line and prints a completion note.
func (b *ProgressBar) Finish(msg string) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.enabled {
		b.clearLocked()
	}
	if msg != "" {
		fmt.Fprintln(b.w, msg)
	}
}

func (b *ProgressBar) clearLocked() {
	if b.lastLen == 0 {
		return
	}
	fmt.Fprint(b.w, "\r"+strings.Repeat(" ", b.lastLen)+"\r")
	b.lastLen = 0
}

func (b *ProgressBar) drawLocked() {
	cur := b.current
	tot := b.total
	var filled int
	var frac string
	if tot > 0 {
		if cur > tot {
			cur = tot
		}
		filled = b.width * cur / tot
		frac = fmt.Sprintf("%d/%d", cur, tot)
	} else {
		filled = b.width / 4
		frac = fmt.Sprintf("%d", cur)
	}
	if filled > b.width {
		filled = b.width
	}
	bar := strings.Repeat("=", filled) + strings.Repeat(" ", b.width-filled)
	line := fmt.Sprintf("%s [%s] %s", b.label, bar, frac)
	if b.name != "" {
		line += " " + truncateLeft(b.name, progressNameWidth)
	}
	b.clearLocked()
	fmt.Fprint(b.w, "\r"+line)
	b.lastLen = utf8.RuneCountInString(line)
}

// truncateLeft keeps the end of s (the file name) when it exceeds max runes.
func truncateLeft(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return "…" + string(r[len(r)-max+1:])
}
