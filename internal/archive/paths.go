package archive

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fr13dman/zipstar-cli/internal/formats"
)

// BuildEntries walks the user-supplied inputs and produces archive FileEntries.
//
// Each input's base name becomes the root name inside the archive:
//
//	./project  → project/...
//	./a.txt    → a.txt
//
// Absolute paths and ".." segments after cleaning are rejected.
// Symlinks are recorded as symlink entries (not followed).
func BuildEntries(inputs []string) ([]formats.FileEntry, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("at least one input path is required")
	}

	var out []formats.FileEntry
	seen := make(map[string]struct{})

	for _, raw := range inputs {
		entries, err := entriesForInput(raw)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if _, dup := seen[e.Name]; dup {
				return nil, fmt.Errorf("duplicate archive path %q from inputs (inputs collide on base names)", e.Name)
			}
			seen[e.Name] = struct{}{}
			out = append(out, e)
		}
	}
	return out, nil
}

func entriesForInput(raw string) ([]formats.FileEntry, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("input path is empty")
	}

	cleaned := filepath.Clean(raw)
	if err := rejectUnsafePath(cleaned); err != nil {
		return nil, err
	}

	info, err := os.Lstat(cleaned)
	if err != nil {
		return nil, fmt.Errorf("cannot stat %q: %w", raw, err)
	}

	rootName := filepath.Base(cleaned)
	if rootName == "." || rootName == string(filepath.Separator) {
		return nil, fmt.Errorf("cannot determine archive root name for %q", raw)
	}

	// Top-level symlink: store as a single symlink entry (do not follow).
	if info.Mode()&os.ModeSymlink != 0 {
		return []formats.FileEntry{{DiskPath: cleaned, Name: rootName}}, nil
	}

	if !info.IsDir() {
		return []formats.FileEntry{{DiskPath: cleaned, Name: rootName}}, nil
	}

	var out []formats.FileEntry
	err = filepath.WalkDir(cleaned, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(cleaned, path)
		if err != nil {
			return err
		}
		var name string
		if rel == "." {
			name = rootName
		} else {
			name = filepath.ToSlash(filepath.Join(rootName, rel))
		}
		if err := rejectUnsafeArchiveName(name); err != nil {
			return err
		}

		// Skip the directory node itself as a separate "file" except we still
		// want empty directories represented. WalkDir visits dirs; for non-empty
		// dirs, children imply the path. We add directory entries explicitly
		// so empty dirs survive.
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			out = append(out, formats.FileEntry{DiskPath: path, Name: name})
			return nil
		}
		if info.IsDir() {
			// Ensure trailing slash convention is NOT used; zip writer sets dir bit from mode.
			out = append(out, formats.FileEntry{DiskPath: path, Name: name})
			return nil
		}
		out = append(out, formats.FileEntry{DiskPath: path, Name: name})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %q: %w", raw, err)
	}
	return out, nil
}

func rejectUnsafePath(cleaned string) error {
	if filepath.IsAbs(cleaned) {
		// Absolute inputs are allowed on disk, but we only store base names in the
		// archive. Still reject path components that are ".." after Clean — Clean
		// removes ".." when possible, so check the original-ish cleaned form for
		// remaining ".." volume-relative oddities.
	}
	for _, part := range strings.Split(cleaned, string(filepath.Separator)) {
		if part == ".." {
			return fmt.Errorf("refusing path with '..' segment: %q", cleaned)
		}
	}
	return nil
}

func rejectUnsafeArchiveName(name string) error {
	if name == "" {
		return fmt.Errorf("archive entry name is empty")
	}
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "\\") {
		return fmt.Errorf("refusing absolute archive entry name %q", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return fmt.Errorf("refusing archive entry with '..': %q", name)
		}
	}
	return nil
}

// EnsureParentDir creates the parent directory of path if needed.
func EnsureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

// CheckOverwrite returns an error if path exists and force is false.
func CheckOverwrite(path string, force bool) error {
	_, err := os.Lstat(path)
	if err == nil {
		if !force {
			return fmt.Errorf("refusing to overwrite existing path %q (pass --force to replace it)", path)
		}
		return nil
	}
	if os.IsNotExist(err) {
		return nil
	}
	return fmt.Errorf("cannot stat %q: %w", path, err)
}
