# Feature: `--version` / `-V`

## Goal

Let users (and bug reports) see exactly which zipstar build is running.

```
$ zipstar -V
zipstar 5245830-dirty
  commit: 5245830 (modified)
  built:  2026-11-04T21:04:38Z
  go:     go1.27.3 darwin/arm64
```

## Technologies considered

| Option | Notes | Decision |
|--------|-------|----------|
| `-v` shorthand | Already `--verbose` on every command; reassigning breaks existing usage | Rejected — use `-V` (same case-split convention as `-p`/`-P`, `-f`/`-F`; matches `python -V`, `ssh -V`) |
| `zipstar version` subcommand | Extra command for one line of output | Not added; flag only |
| Hard-coded constant | Drifts from tags | Rejected |
| `-ldflags -X …/internal/version.Version=…` | Exact release string at build time | **Used by installers** (`git describe --tags --always --dirty`) |
| `runtime/debug.ReadBuildInfo` | Module version for `go install …@vX`, plus `vcs.revision` / `vcs.time` / `vcs.modified` stamped by `go build` | **Fallback + commit/time details** |

## How it works

[`internal/version`](../../internal/version/version.go) resolves, in order:

1. `Version` set via `-ldflags -X` (installers).
2. `debug.BuildInfo.Main.Version` when it is a real version (not `(devel)`). Since Go 1.24 a plain `go build` in a git checkout records a pseudo-version such as `v0.0.0-20261104210438-52458301902e+dirty`.
3. `dev`.

Commit (short), modified flag and commit time come from the build's VCS settings and are omitted when absent.

Cobra wiring in [`internal/cli/root.go`](../../internal/cli/root.go):

```go
root := &cobra.Command{ /* … */ Version: version.String() }
// -v is --verbose, so --version takes -V. Cobra reuses this flag instead
// of adding its own and prints the version when it is set.
root.Flags().BoolP("version", "V", false, "print the zipstar version and exit")
root.SetVersionTemplate("{{.Version}}")
```

Installers stamp the version:

```bash
VERSION="$(git -C "$ROOT" describe --tags --always --dirty 2>/dev/null || echo dev)"
go build -ldflags "-s -w -X github.com/fr13dman/zipstar-cli/internal/version.Version=$VERSION" ./cmd/zipstar
```

## Tests

- `internal/version` — resolution order (stamped, `(devel)` + VCS, `go install` module version, no build info) and output formatting.
- `TestVersionFlag` — `--version` and `-V` print `version.String()`; `-v` does not.
- `TestAllFlagsHaveShortcuts` — now covers the root command (`-V`).
- `scripts/install_test.go` — the installed binary's `-V` matches `git describe`, and the installer reports it.

## Follow-ups

- Tag a first release (e.g. `v0.1.0`) so stamped versions read as releases rather than commit hashes.
