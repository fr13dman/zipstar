package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fr13dman/zipstar-cli/internal/archive"
	"github.com/fr13dman/zipstar-cli/internal/formats"
)

func newListCmd(opts *Options) *cobra.Command {
	var password string

	cmd := &cobra.Command{
		Use:               "list [flags] [<archive>]",
		Aliases:           []string{"l"},
		Short:             "List archive contents",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: archivePathCompletion(opts.Registry),
		RunE: func(cmd *cobra.Command, args []string) error {
			archivePath, err := resolveArchiveArg(args, opts.Registry)
			if err != nil {
				return err
			}
			return runList(cmd.Context(), opts, listArgs{
				archive:  archivePath,
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
	printSuccessHint("listed %d entries", len(entries))
	return nil
}

// archiveLooksEncrypted lists with no password and checks the Encrypted flag.
// Formats that encrypt headers (e.g. RAR -hp) may fail List without a password;
// those errors are treated as "encrypted" so the CLI can prompt.
func archiveLooksEncrypted(ctx context.Context, format formats.Format, path string) (bool, error) {
	entries, err := format.List(ctx, formats.ListOpts{Archive: path})
	if err != nil {
		if looksLikePasswordRequired(err) {
			return true, nil
		}
		return false, err
	}
	for _, e := range entries {
		if e.Encrypted {
			return true, nil
		}
	}
	return false, nil
}

func looksLikePasswordRequired(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "password") || strings.Contains(msg, "encrypted")
}
