package rar

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nwaples/rardecode/v2"

	"github.com/fr13dman/zipstar-cli/internal/formats"
)

func unarchiveDecode(ctx context.Context, opts formats.UnarchiveOpts) error {
	rc, err := openRAR(opts.Archive, opts.Password)
	if err != nil {
		return mapOpenError(opts.Archive, err)
	}
	defer rc.Close()

	dest := opts.DestDir
	if dest == "" {
		dest = "."
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("cannot create destination %q: %w", dest, err)
	}

	// rardecode opens sibling volumes lazily; report each one as it appears.
	reportedVols := 0
	reportVolumes := func() {
		if opts.OnVolume == nil {
			return
		}
		vols := rc.Volumes()
		for ; reportedVols < len(vols); reportedVols++ {
			opts.OnVolume(vols[reportedVols])
		}
	}
	reportVolumes()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		hdr, err := rc.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return mapReadError(opts.Archive, err)
		}
		reportVolumes()
		if opts.OnFile != nil {
			opts.OnFile(hdr.Name)
		}
		if err := extractEntry(rc, hdr, dest, opts.Force); err != nil {
			return err
		}
		reportVolumes()
	}
	return nil
}

// listDecode reads file headers only (no decompression), across all volumes.
func listDecode(ctx context.Context, opts formats.ListOpts) ([]formats.Entry, error) {
	var ropts []rardecode.Option
	if opts.Password != "" {
		ropts = append(ropts, rardecode.Password(opts.Password))
	}
	files, err := rardecode.List(opts.Archive, ropts...)
	if err != nil {
		return nil, mapOpenError(opts.Archive, err)
	}

	out := make([]formats.Entry, 0, len(files))
	for _, f := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		hdr := &f.FileHeader
		out = append(out, formats.Entry{
			Name:      hdr.Name,
			Size:      hdr.UnPackedSize,
			Mode:      os.FileMode(hdr.Mode()),
			ModTime:   hdr.ModificationTime,
			Method:    methodName(hdr),
			Encrypted: hdr.Encrypted || hdr.HeaderEncrypted,
		})
	}
	return out, nil
}

func openRAR(path, password string) (*rardecode.ReadCloser, error) {
	var opts []rardecode.Option
	if password != "" {
		opts = append(opts, rardecode.Password(password))
	}
	return rardecode.OpenReader(path, opts...)
}

func extractEntry(r io.Reader, hdr *rardecode.FileHeader, dest string, force bool) error {
	name := filepath.Clean(hdr.Name)
	if name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refusing unsafe archive entry %q", hdr.Name)
	}
	if filepath.IsAbs(name) || filepath.IsAbs(hdr.Name) {
		return fmt.Errorf("refusing absolute archive entry %q", hdr.Name)
	}
	if strings.HasPrefix(name, "..") {
		return fmt.Errorf("refusing unsafe archive entry %q", hdr.Name)
	}

	target := filepath.Join(dest, name)
	rel, err := filepath.Rel(dest, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("refusing archive entry that escapes destination: %q", hdr.Name)
	}

	mode := os.FileMode(hdr.Mode())

	if hdr.IsDir {
		if err := os.MkdirAll(target, dirPerm(mode)); err != nil {
			return fmt.Errorf("mkdir %q: %w", target, err)
		}
		return applyTimes(target, hdr.ModificationTime)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("mkdir parent of %q: %w", target, err)
	}

	if _, err := os.Lstat(target); err == nil {
		if !force {
			return fmt.Errorf("refusing to overwrite existing path %q (pass --force to replace it)", target)
		}
		if err := os.RemoveAll(target); err != nil {
			return fmt.Errorf("cannot remove %q: %w", target, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("cannot stat %q: %w", target, err)
	}

	if isSymlink(hdr, mode) {
		targetLink := hdr.LinkTarget
		if targetLink == "" {
			data, err := io.ReadAll(r)
			if err != nil {
				return mapReadError(hdr.Name, err)
			}
			targetLink = string(data)
		} else {
			if _, err := io.Copy(io.Discard, r); err != nil {
				return mapReadError(hdr.Name, err)
			}
		}
		if err := os.Symlink(targetLink, target); err != nil {
			return fmt.Errorf("create symlink %q: %w", target, err)
		}
		return nil
	}

	wantPerm := mode.Perm()
	if wantPerm == 0 {
		wantPerm = 0o644
	}
	wantPerm |= 0o400
	createPerm := wantPerm | 0o200
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, createPerm)
	if err != nil {
		return fmt.Errorf("create %q: %w", target, err)
	}
	if _, err := io.Copy(out, r); err != nil {
		out.Close()
		return mapReadError(hdr.Name, err)
	}
	if err := out.Close(); err != nil {
		return err
	}
	if err := os.Chmod(target, wantPerm); err != nil {
		return fmt.Errorf("chmod %q: %w", target, err)
	}
	return applyTimes(target, hdr.ModificationTime)
}

func isSymlink(hdr *rardecode.FileHeader, mode fs.FileMode) bool {
	if mode&os.ModeSymlink != 0 {
		return true
	}
	switch hdr.LinkType {
	case rardecode.LinkTypeUnixSymlink, rardecode.LinkTypeWindowsSymlink:
		return true
	default:
		return false
	}
}

func dirPerm(mode os.FileMode) os.FileMode {
	p := mode.Perm()
	if p == 0 {
		return 0o755
	}
	return p
}

func applyTimes(path string, mod time.Time) error {
	if mod.IsZero() {
		return nil
	}
	return os.Chtimes(path, mod, mod)
}

func methodName(hdr *rardecode.FileHeader) string {
	if hdr.Solid {
		return "rar-solid"
	}
	return "rar"
}

func mapOpenError(archive string, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, rardecode.ErrArchiveEncrypted),
		errors.Is(err, rardecode.ErrArchivedFileEncrypted):
		return fmt.Errorf("archive %q is encrypted: pass --password, set ZIPSTAR_PASSWORD, or run interactively to supply a password", archive)
	case errors.Is(err, rardecode.ErrBadPassword):
		return fmt.Errorf("incorrect password for encrypted archive %q", archive)
	default:
		return fmt.Errorf("cannot open archive %q: %w", archive, err)
	}
}

func mapReadError(name string, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, rardecode.ErrArchivedFileEncrypted),
		errors.Is(err, rardecode.ErrArchiveEncrypted):
		return fmt.Errorf("archive entry %q is encrypted: pass --password, set ZIPSTAR_PASSWORD, or run interactively to supply a password", name)
	case errors.Is(err, rardecode.ErrBadPassword):
		return fmt.Errorf("incorrect password for encrypted archive entry %q", name)
	default:
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "password") {
			return fmt.Errorf("incorrect password for encrypted archive entry %q", name)
		}
		return fmt.Errorf("extract %q: %w", name, err)
	}
}

// NeedsPassword reports whether err indicates a password is required (or wrong).
func NeedsPassword(err error) bool {
	return isPasswordErr(err)
}

func isPasswordErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, rardecode.ErrArchiveEncrypted) ||
		errors.Is(err, rardecode.ErrArchivedFileEncrypted) ||
		errors.Is(err, rardecode.ErrBadPassword) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "password") || strings.Contains(msg, "encrypted")
}
