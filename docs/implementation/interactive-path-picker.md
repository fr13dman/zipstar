# Feature: Interactive path picker, sniff, and shell completion

## Summary

Makes `unarchive` and `list` conversational on a TTY when the archive argument is omitted: a Bubble Tea browser+typeahead picker selects the archive, content sniff recovers format when the extension is missing/unknown, thin shell completion filters by registered extensions, and a one-line stderr hint confirms success. Scripts that pass `<archive>` stay quiet and non-interactive (CI-first).

Guided by [Command Line Interface Guidelines](https://clig.dev/#philosophy): human-first gap-filling, conversation as the norm, saying just enough, TTY-only interactivity, and avoiding unexpected deep filesystem reads.

## Product decisions

| Topic | Choice |
|-------|--------|
| Interaction surface | In-process Bubble Tea picker (not hybrid-first) |
| Trigger | Missing arg + stdin TTY only |
| Commands | `unarchive` + `list` |
| Live listing | Current directory only (+ drill-in) |
| Detection | Extension filter while browsing; one-shot sniff (+ cache) on final path |
| Completion | Cobra `ValidArgsFunction` (extensions + dirs) |
| Confirm before extract | No (existing `--force` only) |
| Non-interactive kill switch | Non-TTY only (no `--no-input`) |
| Success output | One TTY hint on stderr |

## Technologies considered

| Option | Verdict |
|--------|---------|
| `charmbracelet/huh` | Good for forms; weak for directory browser + live filter |
| Custom `bubbletea` + `bubbles` + `lipgloss` | **Chosen** — full control for browser+typeahead |
| `golang.org/x/term` line editor only | Too weak for browse UX |
| Shell completion as primary | Rejected as primary; kept as thin sibling |
| Recursive background scan while typing | Rejected (CLIG surprise FS reads; latency) |

Password prompting remains outside Bubble Tea (`golang.org/x/term` raw mode): each typed character is masked as `*` with backspace support.

## Architecture

```
unarchive/list (no <archive>)
  → stdin TTY?  no → usage error
                 yes → picker.Run (Bubble Tea)
  → archive.ResolveFormat
       extension hit → format
       else → detect.SniffFormatCached → format or error
  → existing unarchive/list logic
  → TTY success hint on stderr
```

### Packages

- [`internal/cli/picker`](../../internal/cli/picker) — Bubble Tea model + `Run`
- [`internal/detect`](../../internal/detect) — zip magic sniff + `XDG_CACHE_HOME/zipstar/sniff.json` cache
- [`internal/cli/archive_arg.go`](../../internal/cli/archive_arg.go) — resolve arg or picker (`ArchivePicker` injectable for tests)
- [`internal/cli/complete.go`](../../internal/cli/complete.go) — shell path completion
- [`internal/archive/resolve.go`](../../internal/archive/resolve.go) — sniff fallback when extension missing/unknown

### Picker UX

- Start in cwd; `ReadDir` current directory only
- Enter on directory drills in; `..` goes up
- Filter box typeahead; absolute/relative/`~` path jumps on Enter
- Archives (registered extensions) highlighted and selectable
- Esc / Ctrl-C → `ErrCanceled` → exit code 1

### Sniff + cache

```go
name, ok, err := detect.SniffFormatCached(path)
```

- Zip signatures: local file `PK\x03\x04`, EOCD `PK\x05\x06`, etc.
- Cache key: absolute path + size + mtime
- Completers never sniff

### Shell completion

```bash
zipstar completion zsh > "${fpath[1]}/_zipstar"
```

`ValidArgsFunction` returns directories and `*.zip` (and future registered extensions).

## Future

- Shallow recursive picker listing (depth ≤ 2) under the current directory
- `archive` interactive multi-select
- Additional format sniffers as backends are added

## Tests

- `internal/cli/picker` — filter, drill, select, cancel, path jump, reject non-archive
- `internal/detect` — zip positive/negative, cache hit/stale
- `internal/archive` — sniff no-extension resolve; `--format` mismatch still errors
- `internal/cli` — missing arg non-TTY usage; picker stub + fake TTY extract; completion fixture

```bash
go test ./...
```
