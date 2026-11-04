# Feature: system-specific installer

## Goal

One command per platform that builds zipstar for the current machine, installs
it together with a matching UnRAR binary, and makes it runnable from any shell.

## Technologies considered

| Option | Notes | Decision |
|--------|-------|----------|
| Install to `/usr/sbin` | Requested originally. macOS SIP marks it `restricted` — writes fail even with `sudo` (verified: `Operation not permitted`). On Linux it is for sysadmin tools. | Rejected |
| `/usr/local/bin` (macOS/Linux) | Conventional for user-installed tools; on PATH by default; often user-writable on macOS | **Default** for `install.sh` |
| `~/.local/bin` | Never needs sudo, but not always on PATH | Available via `--prefix` |
| `%LOCALAPPDATA%\Programs\zipstar` (Windows) | Per-user, no admin; Program Files would need elevation | **Default** for `install.ps1` |
| Homebrew / Scoop / goreleaser packages | Better distribution story, but needs a tap/bucket and release pipeline | Deferred |
| `go install` | No UnRAR, no PATH help | Still works for Go users; not the installer |

## Flow (`scripts/install.sh`)

1. Parse options: `--prefix DIR` (default `/usr/local/bin`), `--unrar FILE`, `--no-unrar`, `--no-path`, `--uninstall`.
2. Detect the platform with `uname` → `darwin_arm64`, `linux_amd64`, …
3. `go build -trimpath -ldflags "-s -w"` into a temp dir.
4. UnRAR source, first match wins: `--unrar FILE` → `third_party/unrar/<plat>/unrar` → `fetch-unrar.sh <plat>`. If none works (offline, linux/arm64), warn and continue — zipstar falls back to its pure-Go decoder.
5. Use `sudo` only when the prefix (or its nearest existing parent) is not writable.
6. Install `zipstar` and `unrar-<os>-<arch>` into the prefix, and the UnRAR license to `<prefix>/../share/zipstar/UNRAR-LICENSE.txt`.
7. Smoke-test `zipstar --help`.
8. PATH: if the prefix is not on `$PATH`, append a marked block to the user's shell startup file, once:

```sh
# >>> zipstar PATH >>>
export PATH="/usr/local/bin:$PATH"
# <<< zipstar PATH <<<
```

| `$SHELL` | File |
|----------|------|
| zsh | `${ZDOTDIR:-$HOME}/.zshrc` |
| bash | `~/.bash_profile` (macOS) / `~/.bashrc` (Linux) |
| fish | `~/.config/fish/config.fish` (`fish_add_path`) |
| other | `~/.profile` |

`--uninstall` removes the binaries, the share directory, and the marked block from any of those files, leaving the rest of the file untouched.

If another `zipstar` appears earlier on PATH, the installer warns.

### Why `unrar-<os>-<arch>` and not `unrar`

`FindUnrar` looks for `unrar-<goos>-<goarch>` next to the zipstar executable,
so the bundled copy is found without PATH changes, and a Homebrew or system
`unrar` in `/usr/local/bin` is never overwritten.

## Flow (`scripts/install.ps1`)

Same shape, PowerShell-native:

- `-Prefix`, `-Unrar`, `-NoUnrar`, `-NoPath`, `-Uninstall`.
- RARLab's `unrarw64.exe` is a self-extracting RAR (GUI) containing `UnRAR.exe`. The installer saves it as `.rar` and unpacks it with the just-built `zipstar.exe` (`ZIPSTAR_UNRAR=off`, pure-Go decoder), so no extra tools are needed. `fetch-unrar.sh windows_amd64` uses the same trick (previously it saved the SFX itself as `unrar.exe`, which would have opened a GUI window instead of extracting).
- User PATH is read and written via the registry with `DoNotExpandEnvironmentNames` / `ExpandString`, so existing `%VAR%` entries are preserved; a user-variable change broadcasts `WM_SETTINGCHANGE` so new terminals see it.
- ARM64 Windows gets the x64 UnRAR (runs under emulation); RARLab publishes no ARM64 UnRAR CLI.

## Key snippets

Sudo only when needed:

```bash
needs_sudo() {
  local dir="$1"
  while [[ ! -e "$dir" ]]; do dir="$(dirname "$dir")"; done
  [[ ! -w "$dir" ]]
}
```

Idempotent PATH block removal:

```bash
awk -v b="$MARK_BEGIN" -v e="$MARK_END" '$0==b{skip=1; next} $0==e{skip=0; next} !skip' "$f"
```

## Tests

`scripts/install_test.go` runs `install.sh` in a sandbox (temp `HOME`, temp
prefix, minimal `PATH`, real Go caches so nothing is downloaded):

- Installs zipstar + UnRAR + license; adds the PATH block once across two runs; a fresh `zsh` sourcing the rc resolves the installed `zipstar`.
- The installed binary extracts through the UnRAR copied beside it (a fake UnRAR proves it is found next to the executable, not via PATH or the repo).
- `--uninstall` removes everything and restores the user's rc file byte-for-byte.
- Prefix already on PATH → rc untouched; `--no-unrar` → no UnRAR installed.
- Unknown options are rejected.
- `install.ps1` is parsed with `pwsh` when available (skipped otherwise). It has not been executed on Windows in this repo's test runs.

## Follow-ups

- Homebrew tap / Scoop bucket for binary distribution without a Go toolchain.
- Run `install.ps1` in Windows CI.
