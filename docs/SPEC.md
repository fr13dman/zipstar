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
| `--format` | inferred | Format name (e.g. `zip`) |
| `--level` | `5` | Compression level `0`–`9` |
| `--encrypt` | off | Encrypt entries (AES-256 for zip) |
| `--password` | unset | Password (see Password resolution) |
| `--force` | off | Overwrite existing output archive |
| `-v, --verbose` | off | Print each archived entry name |

### Unarchive

```
zipstar unarchive|u [flags] <archive>
```

| Flag | Default | Meaning |
|------|---------|---------|
| `-C, --dir` | `.` | Destination directory |
| `--format` | inferred | Format name |
| `--password` | unset | Password when archive is encrypted |
| `--force` | off | Overwrite existing extracted paths |
| `-v, --verbose` | off | Print each extracted entry name |

### List

```
zipstar list|l [flags] <archive>
```

| Flag | Default | Meaning |
|------|---------|---------|
| `--format` | inferred | Format name |
| `--password` | unset | Optional; not required for zip header listing |
| `-v, --verbose` | off | Add method and encrypted flag columns |

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

- Default: no progress output; errors go to stderr with enough context to act.
- `-v`: print per-entry names (and extra list columns).

## Out of scope (v1)

Exclude globs, xattrs / resource forks / Finder flags, `--json` list output, ZipCrypto, non-zip formats.
