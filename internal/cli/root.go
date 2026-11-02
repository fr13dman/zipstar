package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/fr13dman/zipstar-cli/internal/formats"
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
		Use:           "zipstar",
		Short:         "Archive and unarchive files (extensible formats; zip first)",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&opts.Format, "format", "", "archive format (inferred from extension when omitted)")
	root.PersistentFlags().BoolVarP(&opts.Verbose, "verbose", "v", false, "print each archived/extracted/listed path detail")
	root.PersistentFlags().BoolVar(&opts.Force, "force", false, "overwrite existing files")

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
