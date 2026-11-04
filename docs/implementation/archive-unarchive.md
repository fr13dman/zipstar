# Feature: archive / unarchive

## Goal

Create and extract archives through an extensible `formats.Format` seam, with zip as the first backend.

## Technologies considered

| Option | Notes | Decision |
|--------|-------|----------|
| Shell out to `zip`/`unzip` | Fast to ship, poor portability/control | Rejected for core path |
| Stdlib `archive/zip` only | No AES encryption | Used for **unencrypted** write path |
| `github.com/yeka/zip` | AES-256 + read path for encrypted zips | Used for **encrypt** + all reads |
| `klauspost/compress` | Stronger compression codecs | Deferred |

## Flow

1. CLI validates flags (`--level`, `--encrypt` / password pairing).
2. `archive.ResolveFormat` picks the backend from `--format` / extension.
3. `archive.BuildEntries` maps inputs → `{DiskPath, Name}` with base-name roots.
4. `Format.Archive` / `Format.Unarchive` perform I/O.

### Path roots

```go
// ./project → project/...
// ./a.txt   → a.txt
entries, err := archive.BuildEntries(args)
```

### Overwrite

Checked in CLI via `archive.CheckOverwrite` and again inside the zip writer/extractor so direct Format use stays safe.

## Key snippets

Registering the format (extensibility):

```go
reg := formats.NewRegistry()
reg.Register(zipfmt.New())
reg.Register(rarfmt.New())
```

Creating a zip:

```go
format.Archive(ctx, formats.ArchiveOpts{
    Output: "project.zip",
    Files:  entries,
    Level:  5,
    Force:  false,
})
```

## Tests

- `internal/archive/paths_test.go` — roots, symlinks, `..`, duplicates, overwrite helper
- `internal/archive/resolve_test.go` — inference / mismatch
- `internal/formats/zip/zip_test.go` — round-trip, folder/empty dir, symlink, overwrite, levels

## Follow-ups

- Preserve xattrs on macOS
- Additional formats (`tar.gz`)

RAR extract/list is documented in [rar-archive-unarchive.md](rar-archive-unarchive.md).
Unarchive globs / multi-volume coalescing / progress: [unarchive-globs-progress.md](unarchive-globs-progress.md).
