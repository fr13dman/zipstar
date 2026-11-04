package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ExpandInputs expands shell-style globs in the given patterns.
// A path that exists is always taken literally, even when its name contains
// glob characters (e.g. "pack[v2].rar"). A glob that matches nothing is an error.
// Results are de-duplicated and sorted for stable messaging.
func ExpandInputs(patterns []string) ([]string, []string, error) {
	if len(patterns) == 0 {
		return nil, nil, fmt.Errorf("at least one archive path or glob is required")
	}

	var (
		out      []string
		messages []string
		seen     = make(map[string]struct{})
	)

	add := func(path string) {
		clean := filepath.Clean(path)
		if _, ok := seen[clean]; ok {
			return
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}

	for _, pat := range patterns {
		if !hasGlobMeta(pat) {
			add(pat)
			continue
		}
		if _, err := os.Lstat(pat); err == nil {
			add(pat)
			continue
		}
		matches, err := filepath.Glob(pat)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid glob %q: %w", pat, err)
		}
		if len(matches) == 0 {
			return nil, nil, fmt.Errorf("glob %q matched no files", pat)
		}
		sort.Strings(matches)
		messages = append(messages, fmt.Sprintf("glob %q matched %d file(s)", pat, len(matches)))
		for _, m := range matches {
			add(m)
		}
	}

	sort.Strings(out)
	return out, messages, nil
}

func hasGlobMeta(s string) bool {
	return strings.ContainsAny(s, "*?[")
}
