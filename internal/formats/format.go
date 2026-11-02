// Package formats defines the archive format plugin interface and registry.
//
// New formats implement Format and register themselves (see formats/zip).
// The CLI never talks to a concrete format type—only to this interface—so
// adding tar.gz later is a new package + Register call, not a CLI rewrite.
package formats

import (
	"context"
	"os"
	"time"
)

// FileEntry maps one on-disk path to its name inside the archive.
type FileEntry struct {
	DiskPath string // absolute or cleaned path on disk
	Name     string // path inside the archive (forward slashes, no leading /)
}

// ArchiveOpts controls creating an archive.
type ArchiveOpts struct {
	Output   string
	Files    []FileEntry
	Level    int    // compression level 0-9 (format may interpret as needed)
	Password string // non-empty enables encryption when the format supports it
	Force    bool
	OnFile   func(name string) // optional verbose callback per archived entry
}

// UnarchiveOpts controls extracting an archive.
type UnarchiveOpts struct {
	Archive  string
	DestDir  string
	Password string
	Force    bool
	OnFile   func(name string)
}

// ListOpts controls listing archive contents.
type ListOpts struct {
	Archive  string
	Password string
}

// Entry is one member of an archive, used by list.
type Entry struct {
	Name      string
	Size      int64
	Mode      os.FileMode
	ModTime   time.Time
	Method    string // e.g. "deflate", "store"
	Encrypted bool
}

// Format is the extensibility seam for archive backends (zip today, others later).
type Format interface {
	Name() string
	Extensions() []string // including the leading dot, e.g. ".zip"
	Archive(ctx context.Context, opts ArchiveOpts) error
	Unarchive(ctx context.Context, opts UnarchiveOpts) error
	List(ctx context.Context, opts ListOpts) ([]Entry, error)
}
