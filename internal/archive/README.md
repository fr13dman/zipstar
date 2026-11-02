# internal/archive

Orchestration helpers shared by all formats: **which format?** and **which paths go in the archive under which names?**

## Files

| File | Role |
|------|------|
| `resolve.go` | `--format` + extension inference / mismatch errors; derive `-o` name |
| `paths.go` | `BuildEntries`, overwrite check, unsafe path rejection |

## Go tip

This package does not import `internal/formats/zip`. It only depends on the `formats` interface/registry, so new backends do not create import cycles.
