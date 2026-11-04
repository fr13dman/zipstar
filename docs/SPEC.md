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

### Version

```
zipstar --version | -V
```

Prints the version and exits `0`, e.g.:

```
zipstar v0.2.0
  commit: 5245830 (modified)
  built:  2026-11-04T21:04:38Z
  go:     go1.27.3 darwin/arm64
```

The version comes from the build: the installers stamp `git describe --tags --always --dirty`; `go install …@vX.Y.Z` reports that module version; otherwise the Go toolchain's VCS pseudo-version (or `dev`). `commit`/`built` lines appear when the build recorded VCS information. `-v` remains `--verbose`.

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
zipstar unarchive|u [flags] [<archive|glob>...]
```

| Flag | Default | Meaning |
|------|---------|---------|
| `-C, --dir` | `.` | Destination directory |
| `-f, --format` | inferred | Format name |
| `-p, --password` | unset | Password when archive is encrypted |
| `-F, --force` | off | Overwrite existing extracted paths |
| `-P, --progress` | on | Progress bar on stderr; `-P=false` disables it |
| `-v, --verbose` | off | Print each extracted entry name on stdout |

Arguments may be concrete paths or shell-style globs (`*.rar`, `multi.part*.rar`). A path that exists is always taken literally, even if its name contains glob characters (e.g. `pack[v2].rar`). Passing any single volume of a multi-volume set (e.g. `x.part3.rar`) extracts the whole set, starting from the first volume. Status lines (glob matches, multi-volume coalescing, per-archive labels, completion) go to **stderr**. Multi-volume RAR sets matched by a glob are extracted once from the first volume; sibling parts are opened automatically beside it.

At least one argument is required when stdin is not a TTY. When no arguments are given and stdin is a TTY, zipstar opens an interactive path picker (see Interactive path picker) to choose a single archive. Canceling the picker (Esc) exits with code `1`.

After each archive, zipstar prints a summary line to stderr, e.g. `finished multi.part01.rar → <dir> (6 file(s) from 6 volumes)`, followed by `done: extracted N archive(s) into <dir>`. These lines print on TTY and non-TTY runs alike.

### List

```
zipstar list|l [flags] [<archive>]
```

| Flag | Default | Meaning |
|------|---------|---------|
| `-f, --format` | inferred | Format name |
| `-p, --password` | unset | Optional when headers are readable (zip; RAR without header encryption). Required when headers are encrypted (RAR `-hp`) |
| `-v, --verbose` | off | Add method and encrypted flag columns |

Every flag has a one-letter shortcut. Case matters: `-f` format / `-F` force, `-p` password / `-P` progress.

Default list line: `name<TAB>size<TAB>mode`  
Verbose list line: `name<TAB>size<TAB>mode<TAB>method<TAB>plain|encrypted`

`<archive>` follows the same optional/TTY picker rules as unarchive.

After a successful list on a TTY, zipstar prints one hint line to stderr, e.g. `listed N entries`.

## Interactive path picker

Used by `unarchive` and `list` when the archive argument is omitted and stdin is a TTY:

- Starts in the current working directory.
- Lists the **current directory only** (Enter on a directory drills in; `..` goes up). Live filtering does not recurse.
- The input box is a filter **or** a path. Plain text (e.g. `foo`) filters the current listing. Text that looks like a path (starts with `/`, `~`, `.`, or contains `/`) is browsed live: the part up to the last `/` selects the folder shown (relative paths resolve from the current folder), and the rest filters inside it — e.g. typing `~/Downloads/` shows Downloads, `~/Downloads/fo` filters it.
- **Tab** completes the highlighted entry into the input (folders get a trailing `/`, so their contents preview). **Enter** opens a typed folder (it becomes the current folder) or selects a typed file; on a partial path it opens/selects the highlighted match. A typed folder that does not exist shows `no such directory` until corrected.
- Files whose extension matches a registered format (today `.zip` and `.rar`) are highlighted as selectable archives.
- A multi-volume RAR set is listed **once**, as its first volume with a part count (e.g. `name.part1.rar (2 parts)`); later volumes (`.part2.rar`…, `.r00`…) are hidden. Selecting it extracts the whole set.
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

- `unarchive` prints short status messages to stderr (what will be extracted / multi-volume notes / per-archive summary / done).
- `unarchive` draws a progress bar on stderr by default; pass `-P=false` / `--progress=false` for log-friendly output (CI, piping), which keeps the plain status lines.
- `-v`: print per-entry names on stdout (and extra list columns for `list`).
- `list` prints a TTY-only success hint (`listed N entries`), independent of `-v`.
- Errors go to stderr with enough context to act.

## Format capabilities

| Format | Create (`archive`) | Extract (`unarchive`) | List |
|--------|--------------------|------------------------|------|
| `zip` | Yes | Yes | Yes |
| `rar` | **No** — exit `1` with a clear message | Yes | Yes |

For multi-volume RAR, pass the **first** volume path **or** a glob over the parts (zipstar coalesces to the first volume). Sibling parts must sit in the same directory.

RAR extract prefers a bundled/system **UnRAR** binary (per-platform release assets such as `unrar-darwin-arm64` beside the zipstar binary). If none is found, zipstar falls back to pure-Go decoding. Override with `ZIPSTAR_UNRAR=/path/to/unrar` or `ZIPSTAR_UNRAR=off`.

## Out of scope (v1)

Exclude xattrs / resource forks / Finder flags, `--json` list output, ZipCrypto, creating RAR archives, brace-style shell expansion, formats other than zip/rar (e.g. `tar*`), `--no-input` / `CI=1` kill switch, recursive live picker scans, `archive` interactive multi-select, extract confirmation prompts beyond `--force`.
