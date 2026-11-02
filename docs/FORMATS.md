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

## Future formats

New formats register through `formats.Format` with their own extension list. Reserved ideas: `tar`, `tar.gz`, `tar.zst`. Each must document path rules, compression, and encryption (if any) in this file.
