# Feature: UnRAR fast path + multi-platform release assets

## Goal

Speed up RAR extract/list by preferring the official **UnRAR** CLI when a
platform binary is available, while keeping pure-Go `rardecode` as a fallback
(dev machines, unsupported arches, `ZIPSTAR_UNRAR=off`).

## Technologies considered

| Option | Notes | Decision |
|--------|-------|----------|
| Keep rardecode only | Portable, slow on large/solid RAR | Fallback only |
| CGO `go-unarr` | Faster sometimes; weaker RAR5/multi-vol; CGO tax | Rejected |
| System `unrar` on PATH only | Fast, but requires user install | Supported as last discovery step |
| **Bundled UnRAR release assets** | Fast + no separate install for supported platforms | **Chosen** |

## Bundling model

Official UnRAR binaries are redistributable inside other packages (see
`third_party/unrar/LICENSE.txt`). We do **not** commit binaries to git.

```bash
./scripts/fetch-unrar.sh                 # detect host OS/arch
./scripts/fetch-unrar.sh --all           # all supported release platforms
./scripts/fetch-unrar.sh darwin_arm64    # one platform
```

Produces:

| Path | Role |
|------|------|
| `third_party/unrar/<goos>_<goarch>/unrar[.exe]` | Local / CI discovery |
| `dist/unrar/unrar-<goos>-<goarch>[.exe]` | GitHub release asset names |

Supported fetch targets today: `darwin_arm64`, `darwin_amd64`, `linux_amd64`,
`windows_amd64`. Linux arm64 has no matching RARLab package in the current set →
rardecode fallback unless `ZIPSTAR_UNRAR` points at a binary.

## Runtime discovery (`FindUnrar`)

1. `ZIPSTAR_UNRAR` (path, or `off`/`0`/`false` to force rardecode)
2. `unrar` / `unrar-<goos>-<goarch>` next to the zipstar executable (release layout)
3. Walk up from cwd/exe for `third_party/unrar/<goos>_<goarch>/`
4. `PATH`
5. Else rardecode

## Extract / list behavior

- **List:** `unrar lb -v` (no decompress; `-v` walks every volume, names de-duplicated).
  File-encrypted archives still list names; a cheap `unrar t -p-` probe sets
  `Encrypted` so the CLI can prompt.
- **Extract:** validate every member path across all volumes (zip-slip) → overwrite
  checks → `unrar x -o+/-` with output streamed and parsed for live `OnFile` /
  `OnVolume` callbacks.
- **Progress:** header-only list sets the bar total; the bar advances as each
  member starts and volume lines print as each part opens. The rardecode
  fallback reports the same events (`ReadCloser.Volumes()`).

## Key code

- [`internal/formats/rar/locate.go`](../../internal/formats/rar/locate.go)
- [`internal/formats/rar/unrar_exec.go`](../../internal/formats/rar/unrar_exec.go)
- [`internal/formats/rar/rar.go`](../../internal/formats/rar/rar.go) — dispatcher
- [`scripts/fetch-unrar.sh`](../../scripts/fetch-unrar.sh)

## Tests

- Existing suite forces `ZIPSTAR_UNRAR=off` for rardecode coverage.
- `TestUnrar*` runs when a binary is discoverable (`./scripts/fetch-unrar.sh`).

## Release packaging note

Attach both `zipstar-<platform>` and `unrar-<platform>` (or place `unrar` beside
the zipstar binary in the same archive). Users on another OS/arch download that
platform’s pair — no separate UnRAR install required.
