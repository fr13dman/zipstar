# zipstar SPEC (v1)

User-visible behavior for the `zipstar` CLI. Implementation details live in [ARCHITECTURE.md](ARCHITECTURE.md) and format contracts in [FORMATS.md](FORMATS.md).

## Audience

Developer / CI tooling: quiet by default, predictable exit codes, errors that are actionable without `-v`. Interactive gap-filling (path picker, password prompt, success hints) runs only when stdin is a TTY.

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
| `-e, --encrypt` | off | Encrypt entries (AES-256 for zip) |
| `-p, --password` | unset | Password (see Password resolution) |
| `--force` | off | Overwrite existing output archive |
| `-v, --verbose` | off | Print each archived entry name |

Root `zipstar --help` lists global flags only; use `zipstar archive --help` (or `a -h`) for the full archive flag set.

### Unarchive

```
zipstar unarchive|u [flags] [<archive>]
```

| Flag | Default | Meaning |
|------|---------|---------|
| `-C, --dir` | `.` | Destination directory |
| `--format` | inferred | Format name |
| `-p, --password` | unset | Password when archive is encrypted |
| `--force` | off | Overwrite existing extracted paths |
| `-v, --verbose` | off | Print each extracted entry name |

`<archive>` is required when stdin is not a TTY. When omitted and stdin is a TTY, zipstar opens an interactive path picker (see Interactive path picker). Canceling the picker (Esc) exits with code `1`.

After a successful extract on a TTY, zipstar prints one hint line to stderr, e.g. `extracted N entries to <dir>`. Non-TTY runs stay quiet (unless `-v`).

### List

```
zipstar list|l [flags] [<archive>]
```

| Flag | Default | Meaning |
|------|---------|---------|
| `--format` | inferred | Format name |
| `-p, --password` | unset | Optional; not required for zip header listing |
| `-v, --verbose` | off | Add method and encrypted flag columns |

Default list line: `name<TAB>size<TAB>mode`  
Verbose list line: `name<TAB>size<TAB>mode<TAB>method<TAB>plain|encrypted`

`<archive>` follows the same optional/TTY picker rules as unarchive.

After a successful list on a TTY, zipstar prints one hint line to stderr, e.g. `listed N entries`.

## Interactive path picker

Used by `unarchive` and `list` when the archive argument is omitted and stdin is a TTY:

- Starts in the current working directory.
- Lists the **current directory only** (Enter on a directory drills in; `..` goes up). Live filtering does not recurse.
- Typeahead/filter box filters the current listing; an absolute or relative path may be typed/pasted to jump.
- Files whose extension matches a registered format (today `.zip`) are highlighted as selectable archives.
- Esc / Ctrl-C cancels → exit `1`.

**Future (not this wave):** shallow recursive listing up to depth 2 under the current directory.

## Format selection

1. If `--format` is set, that format must be registered.
2. If the archive path has an extension belonging to a **different** registered format, **error** (mismatch).
3. If `--format` is unset and the path extension identifies a registered format, use that format.
4. If `--format` is unset and there is no usable extension (missing or unknown), **sniff** the file contents once (see Content sniff). On a hit, use the sniffed format. On a miss, **error** (ambiguous / unknown) as before.

When `-o` is omitted on `archive`, zipstar derives an output name using `--format` if set, otherwise `zip`, then applies the rules above to the derived path. Sniff applies to existing archive paths (`unarchive` / `list`), not to deriving `-o`.

## Content sniff

- One-shot magic-byte detection on the final selected/passed path (zip local-file / EOCD signatures today).
- Results cached under `$XDG_CACHE_HOME/zipstar/sniff.json` (fallback `~/.cache/zipstar/sniff.json`), keyed by absolute path + size + mtime. Stale or corrupt cache entries are ignored.
- Shell completion does **not** sniff (extension filter only).

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
- Password resolution order: `-p`/`--password` → env `ZIPSTAR_PASSWORD` → interactive prompt when a TTY is available (each character masked as `*`; backspace supported).
- Wrong or missing password → exit `2` with a clear message (no need for `-v`).

## Shell completion

Cobra’s `completion` subcommand is available:

```
zipstar completion zsh|bash|fish
```

`unarchive` and `list` complete directories and files matching registered archive extensions. Install example (zsh):

```
zipstar completion zsh > "${fpath[1]}/_zipstar"
```

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Usage / validation errors (including picker cancel) |
| `2` | Runtime errors (I/O, archive, password) |

## Verbosity and errors

- Default: no progress output; errors go to stderr with enough context to act.
- `-v`: print per-entry names (and extra list columns).
- TTY-only success hints (see Unarchive / List) are independent of `-v`.

## Out of scope (v1)

Exclude globs, xattrs / resource forks / Finder flags, `--json` list output, ZipCrypto, non-zip formats, `--no-input` / `CI=1` kill switch, recursive live picker scans, `archive` interactive multi-select, extract confirmation prompts beyond `--force`.
