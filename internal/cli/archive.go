package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fr13dman/zipstar-cli/internal/archive"
	"github.com/fr13dman/zipstar-cli/internal/formats"
)

func newArchiveCmd(opts *Options) *cobra.Command {
	var (
		output   string
		level    int
		encrypt  bool
		password string
	)

	cmd := &cobra.Command{
		Use:     "archive [flags] <paths...>",
		Aliases: []string{"a"},
		Short:   "Create an archive from files and/or folders",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runArchive(cmd.Context(), opts, archiveArgs{
				inputs:   args,
				output:   output,
				level:    level,
				encrypt:  encrypt,
				password: password,
				passSet:  cmd.Flags().Changed("password"),
			})
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "output archive path (derived from first input when omitted)")
	cmd.Flags().IntVarP(&level, "level", "l", 5, "compression level 0-9 (0=fastest, 9=smallest)")
	cmd.Flags().BoolVarP(&encrypt, "encrypt", "e", false, "encrypt archive entries with AES-256")
	cmd.Flags().StringVarP(&password, "password", "p", "", "archive password (or set ZIPSTAR_PASSWORD; prompted if needed)")

	return cmd
}

type archiveArgs struct {
	inputs   []string
	output   string
	level    int
	encrypt  bool
	password string
	passSet  bool
}

func runArchive(ctx context.Context, opts *Options, a archiveArgs) error {
	if a.level < 0 || a.level > 9 {
		return Usagef("--level must be between 0 and 9 (got %d)", a.level)
	}

	outPath := a.output
	if outPath == "" {
		// When -o is omitted, pick a format to derive "<basename>.<ext>".
		// Prefer --format; otherwise default to zip so the derived name carries .zip.
		deriveFlag := opts.Format
		if deriveFlag == "" {
			deriveFlag = "zip"
		}
		deriveFmt, err := archive.ResolveFormat(opts.Registry, deriveFlag, "")
		if err != nil {
			return Usagef("%v", err)
		}
		outPath, err = archive.DeriveOutputName(a.inputs[0], deriveFmt)
		if err != nil {
			return Usagef("%v", err)
		}
	}

	format, err := archive.ResolveFormat(opts.Registry, opts.Format, outPath)
	if err != nil {
		return Usagef("%v", err)
	}

	pw, err := ResolvePassword(a.encrypt, PasswordSources{
		Flag:    a.password,
		FlagSet: a.passSet,
		Prompt:  true,
	})
	if err != nil {
		return err
	}
	if a.encrypt && pw == "" {
		return Usagef("--encrypt requires a password via --password, %s, or an interactive prompt", passwordEnv)
	}
	if !a.encrypt && a.passSet {
		return Usagef("--password was provided without --encrypt; pass --encrypt to create an encrypted archive")
	}

	entries, err := archive.BuildEntries(a.inputs)
	if err != nil {
		return Usagef("%v", err)
	}

	if err := archive.CheckOverwrite(outPath, opts.Force); err != nil {
		return Runtimef("%v", err)
	}

	var onFile func(string)
	if opts.Verbose {
		onFile = func(name string) { fmt.Println(name) }
	}

	if err := format.Archive(ctx, formats.ArchiveOpts{
		Output:   outPath,
		Files:    entries,
		Level:    a.level,
		Password: pw,
		Force:    opts.Force,
		OnFile:   onFile,
	}); err != nil {
		if errors.Is(err, formats.ErrArchiveNotSupported) {
			return Usagef("%v", err)
		}
		return Runtimef("%v", err)
	}
	return nil
}
