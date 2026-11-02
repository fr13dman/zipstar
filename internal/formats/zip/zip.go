// Package zip implements the zip archive Format.
//
// Non-encrypted archives use the standard library archive/zip so we can set a
// per-Writer deflate --level. Password-protected archives use github.com/yeka/zip
// (AES-256); that fork hard-codes deflate level 5, so --level 0 still means
// store and levels 1-9 use its built-in deflate.
package zip

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	stdzip "archive/zip"

	yezazip "github.com/yeka/zip"

	"github.com/fr13dman/zipstar-cli/internal/formats"
)

// Format is the zip backend.
type Format struct{}

// New returns a zip Format.
func New() *Format { return &Format{} }

func (f *Format) Name() string         { return "zip" }
func (f *Format) Extensions() []string { return []string{".zip"} }

func (f *Format) Archive(ctx context.Context, opts formats.ArchiveOpts) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if opts.Level < 0 || opts.Level > 9 {
		return fmt.Errorf("compression level must be 0-9, got %d", opts.Level)
	}

	if _, err := os.Lstat(opts.Output); err == nil {
		if !opts.Force {
			return fmt.Errorf("refusing to overwrite existing path %q (pass --force to replace it)", opts.Output)
		}
		if err := os.Remove(opts.Output); err != nil {
			return fmt.Errorf("cannot remove existing %q: %w", opts.Output, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("cannot stat %q: %w", opts.Output, err)
	}

	if dir := filepath.Dir(opts.Output); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("cannot create parent directory for %q: %w", opts.Output, err)
		}
	}

	if opts.Password != "" {
		return archiveEncrypted(ctx, opts)
	}
	return archivePlain(ctx, opts)
}

func archivePlain(ctx context.Context, opts formats.ArchiveOpts) error {
	out, err := os.Create(opts.Output)
	if err != nil {
		return fmt.Errorf("cannot create %q: %w", opts.Output, err)
	}
	defer out.Close()

	zw := stdzip.NewWriter(out)
	zw.RegisterCompressor(stdzip.Deflate, newStdFlateCompressor(opts.Level))
	defer func() { _ = zw.Close() }()

	for _, entry := range opts.Files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := writePlainEntry(zw, entry, opts.Level); err != nil {
			return err
		}
		if opts.OnFile != nil {
			opts.OnFile(entry.Name)
		}
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("finalizing zip %q: %w", opts.Output, err)
	}
	return out.Close()
}

func writePlainEntry(zw *stdzip.Writer, entry formats.FileEntry, level int) error {
	info, err := os.Lstat(entry.DiskPath)
	if err != nil {
		return fmt.Errorf("cannot stat %q: %w", entry.DiskPath, err)
	}

	name := filepath.ToSlash(entry.Name)
	header, err := stdzip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("header for %q: %w", entry.DiskPath, err)
	}
	header.Name = name
	header.Method = stdzip.Deflate
	if level == 0 {
		header.Method = stdzip.Store
	}
	header.Modified = info.ModTime()

	isSymlink := info.Mode()&os.ModeSymlink != 0
	isDir := info.IsDir()
	if isDir {
		if !strings.HasSuffix(header.Name, "/") {
			header.Name += "/"
		}
		header.Method = stdzip.Store
	}
	if isSymlink {
		header.Method = stdzip.Store
	}

	w, err := zw.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("creating zip entry %q: %w", name, err)
	}
	return copyEntryContent(w, entry.DiskPath, name, isDir, isSymlink)
}

func archiveEncrypted(ctx context.Context, opts formats.ArchiveOpts) error {
	out, err := os.Create(opts.Output)
	if err != nil {
		return fmt.Errorf("cannot create %q: %w", opts.Output, err)
	}
	defer out.Close()

	zw := yezazip.NewWriter(out)
	defer func() { _ = zw.Close() }()

	for _, entry := range opts.Files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := writeEncryptedEntry(zw, entry, opts.Password, opts.Level); err != nil {
			return err
		}
		if opts.OnFile != nil {
			opts.OnFile(entry.Name)
		}
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("finalizing zip %q: %w", opts.Output, err)
	}
	return out.Close()
}

func writeEncryptedEntry(zw *yezazip.Writer, entry formats.FileEntry, password string, level int) error {
	info, err := os.Lstat(entry.DiskPath)
	if err != nil {
		return fmt.Errorf("cannot stat %q: %w", entry.DiskPath, err)
	}

	name := filepath.ToSlash(entry.Name)
	header, err := yezazip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("header for %q: %w", entry.DiskPath, err)
	}
	header.Name = name
	header.Method = yezazip.Deflate
	if level == 0 {
		header.Method = yezazip.Store
	}
	header.SetModTime(info.ModTime())

	isSymlink := info.Mode()&os.ModeSymlink != 0
	isDir := info.IsDir()
	if isDir {
		if !strings.HasSuffix(header.Name, "/") {
			header.Name += "/"
		}
		header.Method = yezazip.Store
	}
	if isSymlink {
		header.Method = yezazip.Store
	}

	var w io.Writer
	if !isDir {
		header.SetPassword(password)
		header.SetEncryptionMethod(yezazip.AES256Encryption)
	}
	w, err = zw.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("creating zip entry %q: %w", name, err)
	}
	return copyEntryContent(w, entry.DiskPath, name, isDir, isSymlink)
}

func copyEntryContent(w io.Writer, diskPath, name string, isDir, isSymlink bool) error {
	switch {
	case isDir:
		return nil
	case isSymlink:
		target, err := os.Readlink(diskPath)
		if err != nil {
			return fmt.Errorf("readlink %q: %w", diskPath, err)
		}
		if _, err := io.WriteString(w, target); err != nil {
			return fmt.Errorf("writing symlink %q: %w", name, err)
		}
		return nil
	default:
		f, err := os.Open(diskPath)
		if err != nil {
			return fmt.Errorf("open %q: %w", diskPath, err)
		}
		defer f.Close()
		if _, err := io.Copy(w, f); err != nil {
			return fmt.Errorf("writing %q: %w", name, err)
		}
		return nil
	}
}

func (f *Format) Unarchive(ctx context.Context, opts formats.UnarchiveOpts) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r, err := yezazip.OpenReader(opts.Archive)
	if err != nil {
		return fmt.Errorf("cannot open archive %q: %w", opts.Archive, err)
	}
	defer r.Close()

	dest := opts.DestDir
	if dest == "" {
		dest = "."
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("cannot create destination %q: %w", dest, err)
	}

	for _, zf := range r.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := extractFile(zf, dest, opts.Password, opts.Force); err != nil {
			return err
		}
		if opts.OnFile != nil {
			opts.OnFile(zf.Name)
		}
	}
	return nil
}

func extractFile(zf *yezazip.File, dest, password string, force bool) error {
	name := filepath.Clean(zf.Name)
	if name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refusing unsafe archive entry %q", zf.Name)
	}
	if filepath.IsAbs(name) {
		return fmt.Errorf("refusing absolute archive entry %q", zf.Name)
	}

	target := filepath.Join(dest, name)
	rel, err := filepath.Rel(dest, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("refusing archive entry that escapes destination: %q", zf.Name)
	}

	mode := zf.Mode()

	if zf.FileInfo().IsDir() || strings.HasSuffix(zf.Name, "/") {
		if err := os.MkdirAll(target, mode.Perm()); err != nil {
			return fmt.Errorf("mkdir %q: %w", target, err)
		}
		return applyTimes(target, zf.ModTime())
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

	if zf.IsEncrypted() {
		if password == "" {
			return fmt.Errorf("archive entry %q is encrypted: pass --password, set ZIPSTAR_PASSWORD, or run interactively to supply a password", zf.Name)
		}
		zf.SetPassword(password)
	}

	rc, err := zf.Open()
	if err != nil {
		if isBadPassword(err) {
			return fmt.Errorf("incorrect password for encrypted archive entry %q", zf.Name)
		}
		return fmt.Errorf("open entry %q: %w", zf.Name, err)
	}
	defer rc.Close()

	if mode&os.ModeSymlink != 0 {
		data, err := io.ReadAll(rc)
		if err != nil {
			return fmt.Errorf("read symlink %q: %w", zf.Name, err)
		}
		if err := os.Symlink(string(data), target); err != nil {
			return fmt.Errorf("create symlink %q: %w", target, err)
		}
		return nil
	}

	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return fmt.Errorf("create %q: %w", target, err)
	}
	if _, err := io.Copy(out, rc); err != nil {
		out.Close()
		if isBadPassword(err) {
			return fmt.Errorf("incorrect password for encrypted archive entry %q", zf.Name)
		}
		return fmt.Errorf("extract %q: %w", zf.Name, err)
	}
	if err := out.Close(); err != nil {
		return err
	}
	return applyTimes(target, zf.ModTime())
}

func applyTimes(path string, mod time.Time) error {
	if mod.IsZero() {
		return nil
	}
	return os.Chtimes(path, mod, mod)
}

func isBadPassword(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "password") || strings.Contains(msg, "authentication") || strings.Contains(msg, "decrypt")
}

func (f *Format) List(ctx context.Context, opts formats.ListOpts) ([]formats.Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r, err := yezazip.OpenReader(opts.Archive)
	if err != nil {
		return nil, fmt.Errorf("cannot open archive %q: %w", opts.Archive, err)
	}
	defer r.Close()

	out := make([]formats.Entry, 0, len(r.File))
	for _, zf := range r.File {
		if zf.IsEncrypted() && opts.Password != "" {
			zf.SetPassword(opts.Password)
		}
		out = append(out, formats.Entry{
			Name:      zf.Name,
			Size:      int64(zf.UncompressedSize64),
			Mode:      zf.Mode(),
			ModTime:   zf.ModTime(),
			Method:    methodName(zf.Method),
			Encrypted: zf.IsEncrypted(),
		})
	}
	return out, nil
}

func methodName(method uint16) string {
	switch method {
	case yezazip.Deflate:
		return "deflate"
	case yezazip.Store:
		return "store"
	case 99:
		return "aes"
	default:
		return fmt.Sprintf("method-%d", method)
	}
}
