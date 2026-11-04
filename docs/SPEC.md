# zipstar SPEC (v1)

User-visible behavior for the `zipstar` CLI. Implementation details live in [ARCHITECTURE.md](ARCHITECTURE.md) and format contracts in [FORMATS.md](FORMATS.md).

## Audience

Developer / CI tooling: quiet by default, predictable exit codes, errors that are actionable without `-v`.

## Platform

macOS is the supported v1 target. Code should remain portable; other OSes are best-effort.

## Commands

| Command | Alias | Purpose |
|---------|-------|---------|
| `archive` | `a` | Create an archive from files and/or folders |
| `unarchive` | `u` | Extract an archive |
| `list` | `l` | List archive members |

### Archive

```
zipstar archive|a [flags] <paths...>
```

| Flag | Default | Meaning |
|------|---------|---------|
| `-o, --output` | derived | Output archive path. If omitted: `<basename-of-first-input>.<ext>` |
| `-f, --format` | inferred | Format name (e.g. `zip`) |
| `-l, --level` | `5` | Compression level `0`–`9` |
| `-e, --encrypt` | off | Encrypt entries (AES-256 for zip) |
| `-p, --password` | unset | Password (see Password resolution) |
| `-F, --force` | off | Overwrite existing output archive |
| `-v, --verbose` | off | Print each archived entry name |

Root `zipstar --help` lists global flags only; use `zipstar archive --help` (or `a -h`) for the full archive flag set.

### Unarchive

```
zipstar unarchive|u [flags] <archive|glob>...
```

| Flag | Default | Meaning |
|------|---------|---------|
| `-C, --dir` | `.` | Destination directory |
| `-f, --format` | inferred | Format name |
| `-p, --password` | unset | Password when archive is encrypted |
| `-F, --force` | off | Overwrite existing extracted paths |
| `-P, --progress` | on | Progress bar on stderr; `-P=false` disables it |
| `-v, --verbose` | off | Print each extracted entry name on stdout |

Arguments may be concrete paths or shell-style globs (`*.rar`, `multi.part*.rar`). Status lines (glob matches, multi-volume coalescing, per-archive labels, completion) go to **stderr**. Multi-volume RAR sets matched by a glob are extracted once from the first volume; sibling parts are opened automatically beside it.

### List

```
zipstar list|l [flags] <archive>
```

| Flag | Default | Meaning |
|------|---------|---------|
| `-f, --format` | inferred | Format name |
| `-p, --password` | unset | Optional when headers are readable (zip; RAR without header encryption). Required when headers are encrypted (RAR `-hp`) |
| `-v, --verbose` | off | Add method and encrypted flag columns |

Every flag has a one-letter shortcut. Case matters: `-f` format / `-F` force, `-p` password / `-P` progress.

Default list line: `name<TAB>size<TAB>mode`  
Verbose list line: `name<TAB>size<TAB>mode<TAB>method<TAB>plain|encrypted`

## Format selection

1. If `--format` is set, that format must be registered.
2. If the archive path has an extension belonging to a **different** registered format, **error** (mismatch).
3. If `--format` is unset, the path extension must identify a registered format.
4. If `--format` is unset and there is no usable extension, **error** (ambiguous).

When `-o` is omitted on `archive`, zipstar derives an output name using `--format` if set, otherwise `zip`, then applies the rules above to the derived path.

## Paths inside the archive

- Each input’s **base name** is the root entry name (`./project` → `project/...`, `./a.txt` → `a.txt`).
- Absolute paths are not stored; `..` segments are rejected.
- Duplicate archive entry names from colliding base names are an error.
- Symlinks are stored/restored as symlinks (not followed).
- Empty directories are preserved.

## Overwrite

Refuse to overwrite an existing archive or extracted path unless `--force` is set.

## Passwords

- Creating an encrypted archive requires `--encrypt`.
- Providing `--password` without `--encrypt` on `archive` is an error.
- Password resolution order: `--password` → env `ZIPSTAR_PASSWORD` → interactive prompt (no echo) when a TTY is available.
- Wrong or missing password → exit `2` with a clear message (no need for `-v`).

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Usage / validation errors |
| `2` | Runtime errors (I/O, archive, password) |

## Verbosity and errors

- `unarchive` prints short status messages to stderr (what will be extracted / multi-volume notes / done).
- `unarchive` draws a progress bar on stderr by default; pass `-P=false` / `--progress=false` for log-friendly output (CI, piping), which keeps the plain status lines.
- `-v`: print per-entry names on stdout (and extra list columns for `list`).
- Errors go to stderr with enough context to act.

## Format capabilities

| Format | Create (`archive`) | Extract (`unarchive`) | List |
|--------|--------------------|------------------------|------|
| `zip` | Yes | Yes | Yes |
| `rar` | **No** — exit `1` with a clear message | Yes | Yes |

For multi-volume RAR, pass the **first** volume path **or** a glob over the parts (zipstar coalesces to the first volume). Sibling parts must sit in the same directory.

RAR extract prefers a bundled/system **UnRAR** binary (per-platform release assets such as `unrar-darwin-arm64` beside the zipstar binary). If none is found, zipstar falls back to pure-Go decoding. Override with `ZIPSTAR_UNRAR=/path/to/unrar` or `ZIPSTAR_UNRAR=off`.

## Out of scope (v1)

Exclude xattrs / resource forks / Finder flags, `--json` list output, ZipCrypto, creating RAR archives, brace-style shell expansion, and formats other than zip/rar (e.g. `tar*`).
