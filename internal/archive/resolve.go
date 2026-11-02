package archive

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fr13dman/zipstar-cli/internal/formats"
)

// ResolveFormat picks a Format from an explicit --format flag and/or a path's extension.
//
// Rules (locked in SPEC):
//   - If --format is set, that format must exist.
//   - If the path has an extension that belongs to a different format, error (mismatch).
//   - If --format is unset, the extension must identify a known format.
//   - If --format is unset and there is no usable extension, error (ambiguous).
func ResolveFormat(reg *formats.Registry, formatFlag, path string) (formats.Format, error) {
	ext := strings.ToLower(filepath.Ext(path))

	if formatFlag != "" {
		f, ok := reg.Get(formatFlag)
		if !ok {
			return nil, fmt.Errorf("unknown format %q (registered: %s)", formatFlag, strings.Join(reg.Names(), ", "))
		}
		if ext != "" {
			byExt, ok := reg.ByExtension(ext)
			if ok && !sameFormat(f, byExt) {
				return nil, fmt.Errorf("format mismatch: --format %s but path %q has extension %s (expected one of %s)",
					f.Name(), path, ext, strings.Join(f.Extensions(), ", "))
			}
			if !ok {
				// Extension is not registered to any format — still a conflict with an explicit format
				// only when it looks like an archive extension attempt. Treat unknown ext as mismatch
				// when --format is set so we never silently ignore a contradictory suffix.
				return nil, fmt.Errorf("format mismatch: --format %s but path %q has unknown extension %s (expected one of %s)",
					f.Name(), path, ext, strings.Join(f.Extensions(), ", "))
			}
		}
		return f, nil
	}

	if ext == "" {
		return nil, fmt.Errorf("cannot infer archive format from %q: add a known extension (e.g. .zip) or pass --format", path)
	}
	f, ok := reg.ByExtension(ext)
	if !ok {
		return nil, fmt.Errorf("unknown archive extension %q on %q: pass --format or use a supported extension", ext, path)
	}
	return f, nil
}

func sameFormat(a, b formats.Format) bool {
	return strings.EqualFold(a.Name(), b.Name())
}

// DeriveOutputName builds "<basename><ext>" from the first input and the format's primary extension.
func DeriveOutputName(firstInput string, f formats.Format) (string, error) {
	cleaned := filepath.Clean(firstInput)
	base := filepath.Base(cleaned)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "", fmt.Errorf("cannot derive output name from input %q: pass -o/--output", firstInput)
	}
	exts := f.Extensions()
	if len(exts) == 0 {
		return "", fmt.Errorf("format %s declares no extensions", f.Name())
	}
	return base + exts[0], nil
}
