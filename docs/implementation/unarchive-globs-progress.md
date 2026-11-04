# Feature: unarchive globs + progress

## Goal

Let `unarchive` accept shell-style globs / multiple archives, surface clear status
messages, and show a **live** progress bar while members are extracted — without
double-extracting multi-volume RAR sets when a glob matches every part.

## Why not “pass every part”?

Multi-volume RAR still only needs the **first** volume path. UnRAR / `rardecode`
open sibling volumes by name. Globs like `multi.part*.rar` match *all* parts, so
zipstar **coalesces** the set and extracts once from the first volume.

**All members from every volume unpack into the same `-C` destination.** The
`.partNN.rar` files are inputs, not separate extract targets.

## Technologies considered

| Option | Notes | Decision |
|--------|-------|----------|
| Shell expands globs before argv | Works interactively; fails under Go tests / when quoted | Still works; we also expand in-process |
| `filepath.Glob` in zipstar | Portable, matches quoted patterns | **Chosen** for expansion |
| Progress library (`schollz/progressbar`) | Feature-rich | Deferred — small custom stderr bar |
| Progress only on a TTY | Quiet in CI, but users saw no bar in some terminals | Replaced — bar is **on by default**; `--progress=false` opts out |
| Quiet UnRAR then batch `OnFile` | No live ticks | Rejected — stream UnRAR lines and call `OnFile` during extract |

## Flow

1. `ExpandInputs` — expand each arg with `filepath.Glob` (error if a glob matches nothing).
2. `CoalesceArchives` → `[]ExtractTarget` with `Path` + `VolumeCount` / `Volumes`.
3. Status: multi-volume note (`N volumes … all members unpack into <dest>`), then `extracting K archive(s) into <dest>`.
4. Per target: header-only `List` for the bar total (skipped with `--progress=false`) → `Unarchive` with:
   - `OnFile(name)` — advances the bar and shows the member currently extracting
   - `OnVolume(path)` — prints `reading volume i/N: <part>` above the bar as each part is opened
5. Summary: `finished <first part> → <dest> (F file(s) from V volumes)`.

```bash
./zipstar u 'big.part*.rar' -C /tmp/out
# glob "big.part*.rar" matched 4 file(s)
# multi-volume set (4 volumes: big.part1.rar, …); opening big.part1.rar — all members unpack into /tmp/out
# extracting 1 archive(s) into /tmp/out
# [1/1] big.part1.rar (4 volumes → /tmp/out)
#   reading volume 1/4: big.part1.rar
#   reading volume 2/4: big.part2.rar
#   reading volume 3/4: big.part3.rar
#   reading volume 4/4: big.part4.rar
#   extract [=====================       ] 3/4 src/file3.bin     (default; redraws in place; hidden with --progress=false)
#   finished big.part1.rar → /tmp/out (4 file(s) from 4 volumes)
# done: extracted 1 archive(s) into /tmp/out
```

### UnRAR output parsing

`unrar x` streams one line per member and per volume; zipstar parses them live
([`parseUnrarLine`](../../internal/formats/rar/unrar_exec.go)):

| UnRAR line | Event |
|------------|-------|
| `Extracting from /p/big.part2.rar` | volume |
| `Extracting  <dest>/src/file2.bin   \b\b\b\b 16%` | member start (dest stripped, backspaces removed) |
| `...         src/file2.bin   30%  OK` | continuation in next volume (de-duplicated) |
| `Creating    <dest>/src   OK` | directory member |

Listing uses `unrar lb -v`: plain `lb` on the first part only lists members that
*start* in that part, which under-counted the bar and skipped path-safety checks
for later volumes.

## Key code

- [`internal/archive/expand.go`](../../internal/archive/expand.go)
- [`internal/archive/volumes.go`](../../internal/archive/volumes.go) — `ExtractTarget`
- [`internal/cli/progress.go`](../../internal/cli/progress.go)
- [`internal/cli/unarchive.go`](../../internal/cli/unarchive.go)
- [`internal/formats/rar/unrar_exec.go`](../../internal/formats/rar/unrar_exec.go) — streamed `OnFile`

## Tests

- `expand_test.go` / `volumes_test.go` — coalesce volume counts
- `TestRARGlobMultiVolumeUnarchiveCLI` — UnRAR and rardecode × default / `--progress=false`: all members in one dest, `reading volume i/6` lines, `6 file(s) from 6 volumes`; default draws a bar reaching 6/6, opt-out draws none
- `TestMultiVolumeAllVolumesSameDest` — `List` sees all 6 members; `OnFile` ×6 and `OnVolume` ×6 on both engines
- `TestFakeUnrarCallbacksDuringExtract` — callbacks fire before `Unarchive` returns; continuation de-duped
- `TestParseUnrarLine` — real UnRAR line formats (backspaces, `...`, `Creating`)
- `progress_test.go` — `Advance` shows current file, `Println` above the bar, disabled bar prints plain lines

## Limits

- Globs are `filepath` patterns (`*`, `?`, `[…]`), not full shell brace expansion.
- Coalescing is filename-based (`.partN` / `.rNN`); unusual naming may need an explicit first-volume path.
- `-v` still prints each member name on stdout; the bar stays on stderr.
