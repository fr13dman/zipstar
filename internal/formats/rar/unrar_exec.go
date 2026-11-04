package rar

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/fr13dman/zipstar-cli/internal/formats"
)

// listUnrar lists archive members using `unrar lb -v` (bare names, no decompress).
// -v walks every volume; without it UnRAR lists only members that start in the
// first volume. Members split across volumes appear once per volume, so names
// are de-duplicated in order.
func listUnrar(ctx context.Context, bin string, opts formats.ListOpts) ([]formats.Entry, error) {
	args := []string{"lb", "-v"}
	args = append(args, passwordArgs(opts.Password)...)
	args = append(args, opts.Archive)

	out, err := runUnrar(ctx, bin, args)
	if err != nil {
		return nil, mapUnrarError(opts.Archive, err, string(out))
	}

	var entries []formats.Entry
	seen := map[string]struct{}{}
	for _, line := range strings.Split(string(out), "\n") {
		name := strings.TrimRight(line, "\r")
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		entries = append(entries, formats.Entry{
			Name:   name,
			Method: "rar",
		})
	}

	// File-content encryption still allows lb without a password; probe so the
	// CLI password helper can prompt (Encrypted flag).
	if opts.Password == "" && len(entries) > 0 {
		if encrypted, probeErr := probeUnrarEncrypted(ctx, bin, opts.Archive); probeErr == nil && encrypted {
			for i := range entries {
				entries[i].Encrypted = true
			}
		}
	}
	return entries, nil
}

// extractUnrar extracts with UnRAR after validating member paths.
// When OnFile or OnVolume is set, UnRAR output is streamed and parsed so the
// callbacks fire as each member / volume starts (live progress).
func extractUnrar(ctx context.Context, bin string, opts formats.UnarchiveOpts) error {
	dest := opts.DestDir
	if dest == "" {
		dest = "."
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("cannot create destination %q: %w", dest, err)
	}
	// Absolute dest lets the parser tell UnRAR's "Extracting <dest>/<name>" lines
	// apart from its relative "... <name>" continuation lines.
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return fmt.Errorf("cannot resolve destination %q: %w", dest, err)
	}
	dest = absDest

	entries, err := listUnrar(ctx, bin, formats.ListOpts{
		Archive:  opts.Archive,
		Password: opts.Password,
	})
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := validateArchivePath(e.Name); err != nil {
			return err
		}
		target := filepath.Join(dest, filepath.Clean(e.Name))
		rel, relErr := filepath.Rel(dest, target)
		if relErr != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("refusing archive entry that escapes destination: %q", e.Name)
		}
		if _, err := os.Lstat(target); err == nil {
			if !opts.Force {
				return fmt.Errorf("refusing to overwrite existing path %q (pass --force to replace it)", target)
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("cannot stat %q: %w", target, err)
		}
	}

	destArg := dest
	if !strings.HasSuffix(destArg, string(filepath.Separator)) {
		destArg += string(filepath.Separator)
	}

	live := opts.OnFile != nil || opts.OnVolume != nil
	args := []string{"x", "-y"}
	if !live {
		args = append(args, "-idq")
	}
	if opts.Force {
		args = append(args, "-o+")
	} else {
		args = append(args, "-o-")
	}
	args = append(args, passwordArgs(opts.Password)...)
	args = append(args, opts.Archive, destArg)

	reported := map[string]struct{}{}
	var reportMu sync.Mutex
	onLine := func(line string) {
		kind, value := parseUnrarLine(line, dest)
		reportMu.Lock()
		defer reportMu.Unlock()
		switch kind {
		case unrarVolume:
			if opts.OnVolume != nil {
				opts.OnVolume(value)
			}
		case unrarFile:
			if opts.OnFile == nil {
				return
			}
			if _, ok := reported[value]; ok {
				return
			}
			reported[value] = struct{}{}
			opts.OnFile(value)
		}
	}

	out, err := runUnrarStreaming(ctx, bin, args, onLine)
	if err != nil {
		return mapUnrarError(opts.Archive, err, string(out))
	}

	// Report any listed member the parser did not see so counts stay complete.
	if opts.OnFile != nil {
		for _, e := range entries {
			reportMu.Lock()
			_, ok := reported[e.Name]
			reportMu.Unlock()
			if !ok {
				opts.OnFile(e.Name)
			}
		}
	}
	return nil
}

type unrarEvent int

const (
	unrarNone unrarEvent = iota
	unrarVolume
	unrarFile
)

// Trailing "  5% 16%" percentages and "OK" that UnRAR appends to member lines.
var unrarLineSuffix = regexp.MustCompile(`^(.*?)(?:\s+\d{1,3}%)*(?:\s+OK)?\s*$`)

// parseUnrarLine classifies one line of `unrar x` output:
//
//	Extracting from /path/a.part2.rar          → volume event
//	Extracting  <dest>/dir/file   16%          → member start (dest stripped)
//	Creating    <dest>/dir         OK          → directory member
//	...         dir/file           30%  OK     → continuation in next volume
//
// UnRAR redraws percentages with backspaces, which are removed first.
func parseUnrarLine(line, absDest string) (unrarEvent, string) {
	line = strings.TrimSpace(strings.ReplaceAll(line, "\b", ""))
	if rest, ok := strings.CutPrefix(line, "Extracting from "); ok {
		return unrarVolume, strings.TrimSpace(rest)
	}
	var rest string
	switch {
	case strings.HasPrefix(line, "Extracting "):
		rest = strings.TrimPrefix(line, "Extracting ")
	case strings.HasPrefix(line, "Creating "):
		rest = strings.TrimPrefix(line, "Creating ")
	case strings.HasPrefix(line, "..."):
		rest = strings.TrimPrefix(line, "...")
	default:
		return unrarNone, ""
	}
	rest = strings.TrimSpace(rest)
	if m := unrarLineSuffix.FindStringSubmatch(rest); m != nil {
		rest = strings.TrimSpace(m[1])
	}
	if rest == "" {
		return unrarNone, ""
	}
	return unrarFile, memberName(rest, absDest)
}

// memberName converts an UnRAR-printed path back to the archive member name.
func memberName(printed, absDest string) string {
	clean := filepath.Clean(printed)
	if filepath.IsAbs(clean) {
		if rel, err := filepath.Rel(absDest, clean); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(clean)
}

func passwordArgs(password string) []string {
	if password == "" {
		return []string{"-p-"}
	}
	return []string{"-p" + password}
}

func probeUnrarEncrypted(ctx context.Context, bin, archive string) (bool, error) {
	args := []string{"t", "-idq", "-p-", archive}
	out, err := runUnrar(ctx, bin, args)
	if err == nil {
		return false, nil
	}
	msg := strings.ToLower(string(out) + " " + err.Error())
	if strings.Contains(msg, "password") || strings.Contains(msg, "encrypted") {
		return true, nil
	}
	return false, nil
}

func runUnrar(ctx context.Context, bin string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	combined := append(stdout.Bytes(), stderr.Bytes()...)
	if err != nil {
		return combined, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(combined)))
	}
	return stdout.Bytes(), nil
}

func runUnrarStreaming(ctx context.Context, bin string, args []string, onLine func(string)) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var combined bytes.Buffer
	var wg sync.WaitGroup
	scan := func(r io.Reader) {
		defer wg.Done()
		s := bufio.NewScanner(r)
		// UnRAR lines can be long with progress dots.
		s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for s.Scan() {
			line := s.Text()
			combined.WriteString(line)
			combined.WriteByte('\n')
			if onLine != nil {
				onLine(line)
			}
		}
	}
	wg.Add(2)
	go scan(stdout)
	go scan(stderr)
	wg.Wait()
	err = cmd.Wait()
	if err != nil {
		return combined.Bytes(), fmt.Errorf("%w: %s", err, strings.TrimSpace(combined.String()))
	}
	return combined.Bytes(), nil
}

func mapUnrarError(archive string, err error, out string) error {
	msg := strings.ToLower(out + " " + err.Error())
	switch {
	case strings.Contains(msg, "incorrect password"),
		strings.Contains(msg, "wrong password"):
		return fmt.Errorf("incorrect password for encrypted archive %q", archive)
	case strings.Contains(msg, "password"),
		strings.Contains(msg, "encrypted"),
		strings.Contains(msg, "header encrypted"):
		return fmt.Errorf("archive %q is encrypted: pass --password, set ZIPSTAR_PASSWORD, or run interactively to supply a password", archive)
	case strings.Contains(msg, "exit status 10"):
		return fmt.Errorf("refusing to overwrite existing paths under archive %q (pass --force to replace them)", archive)
	default:
		return fmt.Errorf("unrar %q: %v", archive, err)
	}
}

func validateArchivePath(name string) error {
	clean := filepath.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refusing unsafe archive entry %q", name)
	}
	if filepath.IsAbs(clean) || filepath.IsAbs(name) {
		return fmt.Errorf("refusing absolute archive entry %q", name)
	}
	if strings.HasPrefix(clean, "..") {
		return fmt.Errorf("refusing unsafe archive entry %q", name)
	}
	return nil
}
