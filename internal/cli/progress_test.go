package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestProgressBarSetTotalAndAdvance(t *testing.T) {
	var buf bytes.Buffer
	bar := NewProgressBarTo(&buf, "  extract", 0)
	bar.SetTotal(4)
	bar.Advance("exe/test.exe")
	bar.Advance("jpg/test.jpg")
	out := buf.String()
	if !strings.Contains(out, "2/4") {
		t.Fatalf("expected 2/4 in output, got %q", out)
	}
	if !strings.Contains(out, "jpg/test.jpg") {
		t.Fatalf("expected current file name in bar, got %q", out)
	}
	bar.Finish("done")
	if !strings.HasSuffix(buf.String(), "done\n") {
		t.Fatalf("missing finish: %q", buf.String())
	}
}

func TestProgressBarPrintlnKeepsLineAboveBar(t *testing.T) {
	var buf bytes.Buffer
	bar := NewProgressBarTo(&buf, "  extract", 2)
	bar.Advance("a.txt")
	bar.Println("  reading volume 2/2: x.part2.rar")
	out := buf.String()
	i := strings.Index(out, "reading volume 2/2")
	if i < 0 {
		t.Fatalf("volume line missing: %q", out)
	}
	// The bar is redrawn after the status line.
	if !strings.Contains(out[i:], "1/2") {
		t.Fatalf("bar not redrawn after status line: %q", out)
	}
}

func TestProgressBarDisabledPrintsPlainLines(t *testing.T) {
	var buf bytes.Buffer
	bar := newProgressBar(&buf, "  extract", 0, false)
	bar.Advance("a.txt")
	bar.Println("  reading volume 1/2: x.part1.rar")
	bar.Finish("  finished")
	if got, want := buf.String(), "  reading volume 1/2: x.part1.rar\n  finished\n"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestTruncateLeft(t *testing.T) {
	if got := truncateLeft("short", 10); got != "short" {
		t.Fatalf("got %q", got)
	}
	if got := truncateLeft("very/long/path/to/file.txt", 10); got != "…/file.txt" {
		t.Fatalf("got %q", got)
	}
}
