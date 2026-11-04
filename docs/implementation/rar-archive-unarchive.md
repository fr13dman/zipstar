# Feature: RAR extract / list

## Goal

Add RAR as a second `formats.Format` backend: **unarchive** and **list** only. Creating `.rar` archives is explicitly unsupported (no open-source Go encoder; RAR creation is patent-encumbered).

## Technologies considered

| Option | Notes | Decision |
|--------|-------|----------|
| Bundled **UnRAR** CLI + rardecode fallback | Fast native extract; portable fallback | **Chosen** (see [rar-unrar-backend.md](rar-unrar-backend.md)) |
| `github.com/nwaples/rardecode/v2` only | Pure Go; slower on large/solid | Fallback when UnRAR missing |
| `mholt/archives` | Thin wrapper over rardecode; heavier dependency | Rejected — zipstar already owns the Format seam |
| `gen2brain/go-unarr` | CGO / libunarr | Rejected — CGO tax + weaker coverage |
| RAR create (any) | No viable open encoder | Out of scope |

## Flow

1. CLI resolves format from `.rar` / `--format rar` (`archive.ResolveFormat`).
2. Password: `--password` → `ZIPSTAR_PASSWORD` → prompt when encrypted (including header-encrypted archives that fail `List` without a password).
3. `Format.Unarchive` / `Format.List` call `rardecode.OpenReader` and walk members with sequential `Next()` (required for solid archives).
4. `Format.Archive` returns `formats.ErrArchiveNotSupported` → CLI exit `1`.

```go
reg.Register(rarfmt.New())

err := format.Unarchive(ctx, formats.UnarchiveOpts{
    Archive:  "archive.rar",
    DestDir:  "/tmp/out",
    Password: pw,
    Force:    false,
})
```

### Multi-volume

Pass the **first** volume path (e.g. `multi.part01.rar`). Sibling volumes must sit beside it so `OpenReader` can open them. Missing volume → exit `2`.

### Path safety

Same as zip: reject `..` / absolute names; refuse escape outside dest; refuse overwrite unless `--force`.

## Fixtures

Committed under `internal/formats/rar/testdata/` (see that README). Sources: ssokolow/rar-test-files, sharpcompress TestArchives, junrar samples. Hermetic CI — no `rar` binary required.

## Tests

- `internal/formats/rar/rar_test.go` — plain RAR4/5, solid, encrypted, multi-volume, zip-slip, force, Archive unsupported
- `internal/cli/cli_test.go` — `u`/`l` on fixture; `a -o out.rar` → usage exit
- `internal/archive/resolve_test.go` — unknown format uses `7z` (rar is registered)

## Limits

- No create / `--encrypt` on archive for RAR
- `--level` ignored (N/A for extract)
- Windows / MS-DOS attribute → Unix mode is best-effort; owner read is ensured on extract
- Header-encrypted archives require a password for list and extract
