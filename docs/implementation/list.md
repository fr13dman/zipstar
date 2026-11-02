# Feature: list

## Goal

Inspect archive members without extracting. Forces every `Format` to implement a read/metadata path, which keeps future backends honest.

## CLI

```
zipstar list|l [-v] [--format zip] [--password ...] archive.zip
```

- Default: `name`, `size`, `mode` (tab-separated)
- `-v`: also `method`, `plain|encrypted`

## Implementation

`internal/cli/list.go` resolves format, optionally resolves a user-supplied password, then calls `Format.List`.

Zip backend uses `yeka/zip.OpenReader` so encrypted flags and names are visible **without** a password (no interactive prompt on `list`).

## Why not `--json` in v1

Kept line-oriented for CI `cut`/`awk` simplicity. JSON can be added later without changing the Format interface (CLI formatting only).

## Tests

Covered via `TestListAndEncrypt` in `internal/formats/zip/zip_test.go` (encrypted flag) and CLI password helpers.
