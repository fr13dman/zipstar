package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/fr13dman/zipstar-cli/internal/archive"
	"github.com/fr13dman/zipstar-cli/internal/formats"
)

func newListCmd(opts *Options) *cobra.Command {
	var password string

	cmd := &cobra.Command{
		Use:     "list [flags] <archive>",
		Aliases: []string{"l"},
		Short:   "List archive contents",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), opts, listArgs{
				archive:  args[0],
				password: password,
				passSet:  cmd.Flags().Changed("password"),
			})
		},
	}

	cmd.Flags().StringVarP(&password, "password", "p", "", "archive password (or set ZIPSTAR_PASSWORD; prompted if needed)")
	return cmd
}

type listArgs struct {
	archive  string
	password string
	passSet  bool
}

func runList(ctx context.Context, opts *Options, a listArgs) error {
	format, err := archive.ResolveFormat(opts.Registry, opts.Format, a.archive)
	if err != nil {
		return Usagef("%v", err)
	}

	// Listing zip metadata does not require a password; names and the encrypted
	// flag come from headers. Only resolve a password when the user supplied one
	// (for backends that may need it later).
	needPW := a.passSet || envPassword() != ""
	pw, err := ResolvePassword(needPW, PasswordSources{
		Flag:    a.password,
		FlagSet: a.passSet,
		Prompt:  false,
	})
	if err != nil {
		return err
	}

	entries, err := format.List(ctx, formats.ListOpts{
		Archive:  a.archive,
		Password: pw,
	})
	if err != nil {
		return Runtimef("%v", err)
	}

	for _, e := range entries {
		if opts.Verbose {
			enc := "plain"
			if e.Encrypted {
				enc = "encrypted"
			}
			fmt.Fprintf(os.Stdout, "%s\t%d\t%v\t%s\t%s\n", e.Name, e.Size, e.Mode, e.Method, enc)
			continue
		}
		fmt.Fprintf(os.Stdout, "%s\t%d\t%v\n", e.Name, e.Size, e.Mode)
	}
	return nil
}

// archiveLooksEncrypted lists with no password and checks the Encrypted flag.
func archiveLooksEncrypted(ctx context.Context, format formats.Format, path string) (bool, error) {
	entries, err := format.List(ctx, formats.ListOpts{Archive: path})
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		if e.Encrypted {
			return true, nil
		}
	}
	return false, nil
}
