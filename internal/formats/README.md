# internal/formats

The **plugin seam** for archive backends.

- `format.go` — `Format` interface and option structs
- `errors.go` — shared sentinels (e.g. `ErrArchiveNotSupported`)
- `registry.go` — name + extension lookup
- `zip/` — zip create/extract/list
- `rar/` — rar extract/list (create unsupported)

## Adding a format

1. Implement `formats.Format` in a new subpackage.
2. `Register` it from `cmd/zipstar/main.go`.
3. Extend `docs/FORMATS.md` and add tests.

## Go tip

Keeping the interface in this package (not in `cli`) means backends never need to import cobra or password prompting code.
