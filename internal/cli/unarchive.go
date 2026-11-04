package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fr13dman/zipstar-cli/internal/archive"
	"github.com/fr13dman/zipstar-cli/internal/formats"
)

func newUnarchiveCmd(opts *Options) *cobra.Command {
	var (
		dir      string
		password string
	)

	cmd := &cobra.Command{
		Use:     "unarchive [flags] [<archive>]",
		Aliases: []string{"u"},
		Short:   "Extract an archive",
		Args:    cobra.MaximumNArgs(1),
		ValidArgsFunction: archivePathCompletion(opts.Registry),
		RunE: func(cmd *cobra.Command, args []string) error {
			archivePath, err := resolveArchiveArg(args, opts.Registry)
			if err != nil {
				return err
			}
			return runUnarchive(cmd.Context(), opts, unarchiveArgs{
				archive:  archivePath,
				dir:      dir,
				password: password,
				passSet:  cmd.Flags().Changed("password"),
			})
		},
	}

	cmd.Flags().StringVarP(&dir, "dir", "C", "", "destination directory (default: current directory)")
	cmd.Flags().StringVar(&password, "password", "", "archive password (or set ZIPSTAR_PASSWORD; prompted if needed)")

	return cmd
}

type unarchiveArgs struct {
	archive  string
	dir      string
	password string
	passSet  bool
}

func runUnarchive(ctx context.Context, opts *Options, a unarchiveArgs) error {
	format, err := archive.ResolveFormat(opts.Registry, opts.Format, a.archive)
	if err != nil {
		return Usagef("%v", err)
	}

	// Probe whether the archive needs a password by listing once when possible.
	// We always resolve a password if --password / env was provided; otherwise
	// we attempt extract and surface encrypted-entry errors clearly from the format layer.
	needPW := a.passSet || passwordEnvSet()
	pw, err := ResolvePassword(needPW, PasswordSources{
		Flag:    a.password,
		FlagSet: a.passSet,
		Prompt:  false,
	})
	if err != nil {
		return err
	}

	// If the archive is encrypted and no password source was given, prompt.
	if pw == "" {
		encrypted, err := archiveLooksEncrypted(ctx, format, a.archive)
		if err != nil {
			return Runtimef("%v", err)
		}
		if encrypted {
			pw, err = ResolvePassword(true, PasswordSources{Prompt: true})
			if err != nil {
				return err
			}
		}
	}

	dest := a.dir
	if dest == "" {
		dest = "."
	}

	count := 0
	onFile := func(name string) {
		count++
		if opts.Verbose {
			fmt.Println(name)
		}
	}

	if err := format.Unarchive(ctx, formats.UnarchiveOpts{
		Archive:  a.archive,
		DestDir:  dest,
		Password: pw,
		Force:    opts.Force,
		OnFile:   onFile,
	}); err != nil {
		return Runtimef("%v", err)
	}
	printSuccessHint("extracted %d entries to %s", count, dest)
	return nil
}

func passwordEnvSet() bool {
	// Avoid importing os at call sites repeatedly — thin wrapper.
	return envPassword() != ""
}
