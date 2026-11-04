package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fr13dman/zipstar-cli/internal/archive"
	"github.com/fr13dman/zipstar-cli/internal/formats"
)

func newUnarchiveCmd(opts *Options) *cobra.Command {
	var (
		dir      string
		password string
		progress bool
	)

	cmd := &cobra.Command{
		Use:     "unarchive [flags] [<archive|glob>...]",
		Aliases: []string{"u"},
		Short:   "Extract one or more archives (globs supported)",
		Long: `Extract one or more archives. Arguments may be paths or shell-style globs
(e.g. '*.rar', 'multi.part*.rar'). With no arguments on a terminal, an
interactive picker opens to choose an archive.

Multi-volume RAR sets matched by a glob are coalesced: only the first volume is
opened; sibling volumes are read automatically. All members unpack into the
single destination directory (-C).`,
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: archivePathCompletion(opts.Registry),
		RunE: func(cmd *cobra.Command, args []string) error {
			inputs := args
			if len(inputs) == 0 {
				archivePath, err := resolveArchiveArg(args, opts.Registry)
				if err != nil {
					return err
				}
				inputs = []string{archivePath}
			}
			return runUnarchive(cmd.Context(), opts, unarchiveArgs{
				inputs:   inputs,
				dir:      dir,
				password: password,
				passSet:  cmd.Flags().Changed("password"),
				progress: progress,
				status:   cmd.ErrOrStderr(),
			})
		},
	}

	cmd.Flags().StringVarP(&dir, "dir", "C", "", "destination directory (default: current directory)")
	cmd.Flags().StringVarP(&password, "password", "p", "", "archive password (or set ZIPSTAR_PASSWORD; prompted if needed)")
	cmd.Flags().BoolVarP(&progress, "progress", "P", true, "show a progress bar on stderr (use -P=false / --progress=false to disable)")

	return cmd
}

type unarchiveArgs struct {
	inputs   []string
	dir      string
	password string
	passSet  bool
	progress bool
	status   io.Writer
}

func runUnarchive(ctx context.Context, opts *Options, a unarchiveArgs) error {
	status := a.status
	if status == nil {
		status = os.Stderr
	}

	expanded, expandMsgs, err := archive.ExpandInputs(a.inputs)
	if err != nil {
		return Usagef("%v", err)
	}
	for _, m := range expandMsgs {
		fmt.Fprintln(status, m)
	}

	targets := archive.DiscoverVolumes(archive.CoalesceArchives(expanded))
	if len(targets) == 0 {
		return Usagef("no archives to extract after expanding inputs")
	}

	dest := a.dir
	if dest == "" {
		dest = "."
	}

	for _, t := range targets {
		if t.VolumeCount > 1 {
			bases := make([]string, len(t.Volumes))
			for i, v := range t.Volumes {
				bases[i] = filepath.Base(v)
			}
			fmt.Fprintf(status, "multi-volume set (%d volumes: %s); opening %s — all members unpack into %s\n",
				t.VolumeCount, summarizeBases(bases), filepath.Base(t.Path), dest)
		}
	}

	fmt.Fprintf(status, "extracting %d archive(s) into %s\n", len(targets), dest)

	for i, t := range targets {
		if err := ctx.Err(); err != nil {
			return err
		}
		if t.VolumeCount > 1 {
			fmt.Fprintf(status, "[%d/%d] %s (%d volumes → %s)\n", i+1, len(targets), filepath.Base(t.Path), t.VolumeCount, dest)
		} else {
			fmt.Fprintf(status, "[%d/%d] %s\n", i+1, len(targets), filepath.Base(t.Path))
		}

		if err := unarchiveOne(ctx, opts, a, t, dest, status); err != nil {
			return err
		}
	}

	fmt.Fprintf(status, "done: extracted %d archive(s) into %s\n", len(targets), dest)
	return nil
}

func summarizeBases(bases []string) string {
	if len(bases) <= 4 {
		return strings.Join(bases, ", ")
	}
	return fmt.Sprintf("%s … %s", bases[0], bases[len(bases)-1])
}

func unarchiveOne(ctx context.Context, opts *Options, a unarchiveArgs, target archive.ExtractTarget, dest string, status io.Writer) error {
	archivePath := target.Path
	format, err := archive.ResolveFormat(opts.Registry, opts.Format, archivePath)
	if err != nil {
		return Usagef("%v", err)
	}

	needPW := a.passSet || passwordEnvSet()
	pw, err := ResolvePassword(needPW, PasswordSources{
		Flag:    a.password,
		FlagSet: a.passSet,
		Prompt:  false,
	})
	if err != nil {
		return err
	}

	if pw == "" {
		encrypted, err := archiveLooksEncrypted(ctx, format, archivePath)
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

	wantProgress := a.progress
	bar := newProgressBar(status, "  extract", 0, wantProgress)
	if wantProgress {
		// Header-only list for every backend; sets N/M before extract starts.
		if entries, listErr := format.List(ctx, formats.ListOpts{Archive: archivePath, Password: pw}); listErr == nil {
			bar.SetTotal(len(entries))
		}
	}

	files := 0
	onFile := func(name string) {
		files++
		bar.Advance(name)
		if opts.Verbose {
			fmt.Fprintln(os.Stdout, name)
		}
	}

	// Single-volume archives also report their one volume; only print volume
	// lines once a set is known (glob) or a second volume actually appears.
	volumes := 0
	var firstVolume string
	volumeLine := func(i int, path string) string {
		if target.VolumeCount > 1 {
			return fmt.Sprintf("  reading volume %d/%d: %s", i, target.VolumeCount, filepath.Base(path))
		}
		return fmt.Sprintf("  reading volume %d: %s", i, filepath.Base(path))
	}
	onVolume := func(path string) {
		volumes++
		switch {
		case volumes == 1:
			firstVolume = path
			if target.VolumeCount > 1 {
				bar.Println(volumeLine(1, path))
			}
		case volumes == 2 && target.VolumeCount <= 1:
			bar.Println(volumeLine(1, firstVolume))
			bar.Println(volumeLine(2, path))
		default:
			bar.Println(volumeLine(volumes, path))
		}
	}

	if err := format.Unarchive(ctx, formats.UnarchiveOpts{
		Archive:  archivePath,
		DestDir:  dest,
		Password: pw,
		Force:    opts.Force,
		OnFile:   onFile,
		OnVolume: onVolume,
	}); err != nil {
		bar.Finish("")
		return Runtimef("%v", err)
	}

	summary := fmt.Sprintf("%d file(s)", files)
	if volumes > 1 {
		summary += fmt.Sprintf(" from %d volumes", volumes)
	}
	bar.Finish(fmt.Sprintf("  finished %s → %s (%s)", filepath.Base(archivePath), dest, summary))
	return nil
}

func passwordEnvSet() bool {
	return envPassword() != ""
}
