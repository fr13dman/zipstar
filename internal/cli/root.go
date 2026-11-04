package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/fr13dman/zipstar-cli/internal/formats"
	"github.com/fr13dman/zipstar-cli/internal/version"
)

// Options shared across commands.
type Options struct {
	Registry *formats.Registry
	Format   string
	Verbose  bool
	Force    bool
}

// NewRoot builds the zipstar command tree.
func NewRoot(reg *formats.Registry) *cobra.Command {
	opts := &Options{Registry: reg}

	root := &cobra.Command{
		Use:   "zipstar",
		Short: "Archive and unarchive files (extensible formats; zip first)",
		Long: `Archive and unarchive files (extensible formats; zip first).

Every flag has a one-letter shortcut. Global: -f/--format, -F/--force,
-v/--verbose; -V/--version prints the version. Command-specific flags (such as
-e/--encrypt, -p/--password, -l/--level, -P/--progress) appear on the
subcommand help pages:

  zipstar archive --help
  zipstar unarchive --help
  zipstar list --help`,
		Example: `  zipstar archive --help
  zipstar a -e -p secret -l 9 -o out.zip ./folder
  zipstar u -p secret -C /tmp/out out.zip
  zipstar u -F -P=false -C /tmp/out 'multi.part*.rar'
  zipstar l -v -f rar archive.rar`,
		Version:       version.String(),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	// -v is --verbose, so --version takes -V. Cobra reuses this flag instead
	// of adding its own and prints the version when it is set.
	root.Flags().BoolP("version", "V", false, "print the zipstar version and exit")
	root.SetVersionTemplate("{{.Version}}")

	root.PersistentFlags().StringVarP(&opts.Format, "format", "f", "", "archive format (inferred from extension when omitted)")
	root.PersistentFlags().BoolVarP(&opts.Verbose, "verbose", "v", false, "print each archived/extracted/listed path detail")
	root.PersistentFlags().BoolVarP(&opts.Force, "force", "F", false, "overwrite existing files")

	root.AddCommand(newArchiveCmd(opts))
	root.AddCommand(newUnarchiveCmd(opts))
	root.AddCommand(newListCmd(opts))

	return root
}

// Execute runs the root command and maps errors to process exit codes.
func Execute(reg *formats.Registry) int {
	root := NewRoot(reg)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err.Error())
		var ee *ExitError
		if errors.As(err, &ee) {
			return ee.Code
		}
		return ExitRuntime
	}
	return ExitOK
}
