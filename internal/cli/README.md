# internal/cli

Cobra commands and user-facing concerns: flags, password resolution, exit codes, interactive picker wiring, shell completion, TTY success hints.

## Files

| File | Role |
|------|------|
| `root.go` | Root command, persistent flags (`--format`, `-v`, `--force`), `Execute` → exit code |
| `archive.go` | `archive` / `a` (`-e/--encrypt`, `-o`, password) |
| `unarchive.go` | `unarchive` / `u` (optional archive arg + picker) |
| `list.go` | `list` / `l` (optional archive arg + picker) |
| `archive_arg.go` | Resolve `<archive>` or launch `ArchivePicker` on TTY |
| `complete.go` | `ValidArgsFunction` for archive paths |
| `hint.go` | TTY-only success one-liners on stderr |
| `tty.go` | stdin TTY check (overridable in tests) |
| `password.go` | Flag → env → prompt |
| `exit.go` | `ExitError` with codes `1` (usage) and `2` (runtime) |
| `picker/` | Bubble Tea browser + typeahead path picker |

## Go tip

`RunE` returning an error lets cobra skip printing usage when `SilenceUsage: true`. We wrap validation failures as `Usagef(...)` so `main` can exit `1` without treating them like I/O failures.
