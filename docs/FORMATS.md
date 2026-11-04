# zipstar FORMATS (v1)

## zip

| Property | Value |
|----------|-------|
| Name | `zip` |
| Extensions | `.zip` |
| Compression | Deflate; `--level 0` stores uncompressed; `1`–`9` deflate |
| Encryption | AES-256 (WinZip-compatible) when `--encrypt` is set |
| Symlinks | Stored as symlinks (link target as entry payload) |
| Modes / times | Unix permission bits and modification time preserved when the format allows |
| xattrs / resource forks | **Not** preserved in v1 |

### Compression level details

- **Unencrypted** archives use Go’s `archive/zip` with a per-Writer flate compressor, so `--level` is honored (`0`–`9`).
- **Encrypted** archives use `github.com/yeka/zip`. That library’s built-in deflate is fixed at level 5; zipstar still accepts `--level`, maps `0` → store, and `1`–`9` → deflate (library default).

### Encryption details

- Algorithm: AES-256 via `yeka/zip`.
- Directory entries are not encrypted as payloads; file and symlink entries are.
- Some older `unzip` builds cannot read AES zips; modern tools (7-Zip, recent Info-ZIP, many GUI tools) can.
- ZipCrypto is intentionally unsupported.

### Known limitations

- No AppleDouble / resource-fork handling.
- No extended attributes.
- Encrypted deflate level is not finely tunable (see above).
- Listing encrypted member *contents* requires a password; the encrypted flag and names are available from headers.

## rar

| Property | Value |
|----------|-------|
| Name | `rar` |
| Extensions | `.rar` |
| Create | **Not supported** (`Archive` → `formats.ErrArchiveNotSupported`, CLI exit `1`) |
| Extract / list | Prefer bundled/system **UnRAR** CLI; fall back to `github.com/nwaples/rardecode/v2` |
| Encryption | Password via existing CLI resolution; supports per-file and header encryption |
| Solid | Supported (sequential `Next` + read) |
| Multi-volume | Supported when volumes are adjacent; pass the first volume path or a glob over parts (CLI coalesces) |
| Symlinks | Restored when headers encode Unix/Windows symlink link types |
| Modes / times | Best-effort from RAR host attributes; owner read ensured on extract |
| xattrs / resource forks | **Not** preserved |

### Known limitations

- No RAR writer (patent / no open encoder).
- `--level` and `--encrypt` on `archive` do not apply (create unsupported).
- Windows / MS-DOS attribute bits map imperfectly to Unix modes (rardecode path).
- Listing without a password works when headers are readable; header-encrypted (`-hp`) archives require a password for list and extract.
- `.cbr` and other non-`.rar` extensions are not registered.
- UnRAR is shipped as **per-platform release assets** (`unrar-darwin-arm64`, …). Unsupported arches (e.g. linux/arm64 today) use rardecode unless `ZIPSTAR_UNRAR` is set. Set `ZIPSTAR_UNRAR=off` to force rardecode.

See `docs/implementation/rar-archive-unarchive.md` and `docs/implementation/rar-unrar-backend.md`.

## Future formats

New formats register through `formats.Format` with their own extension list. Reserved ideas: `tar`, `tar.gz`, `tar.zst`. Each must document path rules, compression, and encryption (if any) in this file.
