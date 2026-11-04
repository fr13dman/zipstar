package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fr13dman/zipstar-cli/internal/formats"
)

// archivePathCompletion completes directories and files with registered archive extensions.
func archivePathCompletion(reg *formats.Registry) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		exts := make(map[string]struct{}, len(reg.Extensions()))
		for _, e := range reg.Extensions() {
			exts[strings.ToLower(e)] = struct{}{}
		}

		dir := "."
		prefix := toComplete
		if toComplete != "" {
			if strings.HasSuffix(toComplete, string(filepath.Separator)) {
				dir = toComplete
				prefix = ""
			} else {
				dir = filepath.Dir(toComplete)
				prefix = filepath.Base(toComplete)
				if dir == "." && !strings.Contains(toComplete, string(filepath.Separator)) {
					// keep dir as .
				}
			}
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, cobra.ShellCompDirectiveDefault
		}

		var out []string
		for _, e := range entries {
			name := e.Name()
			if prefix != "" && !strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
				continue
			}
			full := name
			if dir != "." {
				full = filepath.Join(dir, name)
			}
			if e.IsDir() {
				out = append(out, full+string(filepath.Separator))
				continue
			}
			ext := strings.ToLower(filepath.Ext(name))
			if _, ok := exts[ext]; ok {
				out = append(out, full)
			}
		}
		return out, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveDefault
	}
}
