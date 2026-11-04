# zipstar

Archive / unarchive CLI with an extensible format plugin system. v1 ships **zip** (including AES-256 passwords) on macOS.

## Docs

- [SPEC](docs/SPEC.md) — commands, flags, exit codes, behavior
- [ARCHITECTURE](docs/ARCHITECTURE.md) — packages, request flow, interactive paths
- [FORMATS](docs/FORMATS.md) — zip contract and limitations
- [Implementation notes](docs/implementation/) — feature writeups (picker, AES passwords, …)

## Build

Requires Go 1.22+ (developed with Go 1.27).

```bash
go build -o zipstar ./cmd/zipstar
```

## Usage

```bash
# Archive a folder (writes project.zip)
./zipstar a ./project

# Archive with explicit output and compression level
./zipstar archive -o out.zip --level 9 ./project ./notes.txt

# Encrypt (-e); omit --password on a TTY to get a masked prompt: Password: ******
./zipstar a -e -o secret.zip ./project

# Encrypt with explicit password (visible on the command line — prefer prompt/env)
./zipstar a -e --password 'secret' -o secret.zip ./project

# Extract to another directory
./zipstar u secret.zip -C /tmp/out

# Extract encrypted archive (TTY prompts Password: ****** if flag/env unset)
./zipstar u secret.zip -C /tmp/out

# Interactive extract/list (TTY): omit <archive> to open the path picker
./zipstar u -C /tmp/out
./zipstar l

# List
./zipstar l -v out.zip

# Shell completion (zsh example) — unarchive/list complete dirs + *.zip
./zipstar completion zsh > "${fpath[1]}/_zipstar"
```

Aliases: `a` = `archive`, `u` = `unarchive`, `l` = `list`.

| Shortcut | Long form | Notes |
|----------|-----------|--------|
| `-e` | `--encrypt` | Archive only; creates AES-256 zip |
| `-o` | `--output` | Archive output path |
| `-C` | `--dir` | Unarchive destination (default `.`) |
| `-v` | `--verbose` | Per-entry / extra list columns |

Password sources: `--password` → `ZIPSTAR_PASSWORD` → interactive masked prompt (`*`) on a TTY.

Interactive path picker and success hints run only when stdin is a TTY. Scripts should pass `<archive>` and passwords via flag/env. Details: [interactive-path-picker](docs/implementation/interactive-path-picker.md), [zip-aes-passwords](docs/implementation/zip-aes-passwords.md).

## Test

```bash
go test ./...
```

## Layout

```
cmd/zipstar/           # binary entrypoint
internal/cli/          # cobra commands, masked password, completion, TTY hints
internal/cli/picker/   # Bubble Tea archive path picker
internal/archive/      # format resolve (ext + sniff) + path orchestration
internal/detect/       # content sniff + cache
internal/formats/      # Format interface + zip backend
docs/                  # SPEC / ARCHITECTURE / FORMATS / implementation
```
