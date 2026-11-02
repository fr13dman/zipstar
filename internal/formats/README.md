# internal/formats

The **plugin seam** for archive backends.

- `format.go` — `Format` interface and option structs
- `registry.go` — name + extension lookup
- `zip/` — first concrete backend

## Adding a format

1. Implement `formats.Format` in a new subpackage.
2. `Register` it from `cmd/zipstar/main.go`.
3. Extend `docs/FORMATS.md` and add tests.

## Go tip

Keeping the interface in this package (not in `cli`) means backends never need to import cobra or password prompting code.
