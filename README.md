# zipstar

Archive / unarchive CLI with an extensible format plugin system. v1 ships **zip** (including AES-256 passwords) on macOS.

## Docs

- [SPEC](docs/SPEC.md) — commands, flags, exit codes, behavior
- [ARCHITECTURE](docs/ARCHITECTURE.md) — packages and how to add a format
- [FORMATS](docs/FORMATS.md) — zip contract and limitations
- [Implementation notes](docs/implementation/) — feature writeups

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

# Encrypt
./zipstar a --encrypt --password 'secret' -o secret.zip ./project

# Extract
./zipstar u secret.zip -C /tmp/out --password 'secret'

# List
./zipstar l -v out.zip
```

Aliases: `a` = `archive`, `u` = `unarchive`, `l` = `list`.

Password sources: `--password`, then `ZIPSTAR_PASSWORD`, then interactive prompt.

## Test

```bash
go test ./...
```

## Layout

```
cmd/zipstar/           # binary entrypoint
internal/cli/          # cobra commands
internal/archive/      # format + path orchestration
internal/formats/      # Format interface + zip backend
docs/                  # SPEC / ARCHITECTURE / FORMATS
```
