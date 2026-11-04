# internal/formats/rar

RAR backend for zipstar (extract / list only).

## Engines

1. **UnRAR CLI** (preferred) — official binary via release assets / `third_party/unrar` / `PATH`
2. **`github.com/nwaples/rardecode/v2`** — pure-Go fallback

`Archive` always returns `formats.ErrArchiveNotSupported`.

See:

- `docs/implementation/rar-archive-unarchive.md`
- `docs/implementation/rar-unrar-backend.md`
- `third_party/unrar/README.md`
