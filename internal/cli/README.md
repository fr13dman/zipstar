# internal/cli

Cobra commands and user-facing concerns: flags, password resolution, exit codes.

## Files

| File | Role |
|------|------|
| `root.go` | Root command, persistent flags (`--format`, `-v`, `--force`), `Execute` → exit code |
| `archive.go` | `archive` / `a` |
| `unarchive.go` | `unarchive` / `u` |
| `list.go` | `list` / `l` |
| `password.go` | Flag → env → prompt |
| `exit.go` | `ExitError` with codes `1` (usage) and `2` (runtime) |

## Go tip

`RunE` returning an error lets cobra skip printing usage when `SilenceUsage: true`. We wrap validation failures as `Usagef(...)` so `main` can exit `1` without treating them like I/O failures.
