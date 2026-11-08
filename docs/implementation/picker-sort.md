# Feature: Picker sort by name or date

## Goal

Let users reorder the interactive archive picker (`unarchive` / `list` with no `<archive>` on a TTY) so the archive they want is near the top. The most common need is "the thing I just downloaded", so the default is newest first.

```
Select archive
/Users/me/Downloads
sort: date (newest first)

> filter or path…

> ..
  report-final.zip
  photos.part1.rar (3 parts)
  old-backup.zip
  …

type to filter or a path (/, ~/, ../) to browse · tab complete · ↑↓ move · enter open/select · ctrl+s sort name/date · ctrl+r reverse · esc cancel
```

## Behavior

| Key | Effect |
|-----|--------|
| (default) | Date (modification time), newest first |
| Ctrl+S | Switch sort between date and name. Name starts A-Z; date starts newest first |
| Ctrl+R | Reverse the current sort (A-Z and Z-A, newest and oldest first) |

- `..` is always the first row.
- Folders and files are sorted together (no folders-first grouping), matching the previous mixed listing.
- Name sort is case-insensitive (`alpha`, `Bravo`, `charlie`), with byte order breaking ties so the result is deterministic. Date sort breaks equal-time ties by name.
- A multi-volume RAR set (listed once as its first volume) sorts by the **first volume's** modification time, because that's the file the row stands for.
- The highlighted entry stays highlighted after a re-sort; the cursor follows it by name.
- The sort applies to the filtered results, carries over when you move between folders (Enter, `..`, typed paths), and resets when the picker runs again.

## Product decisions

| Topic | Choice |
|-------|--------|
| Default | Date, newest first (most recent download or build first) |
| Controls | Two keys: Ctrl+S for the sort field, Ctrl+R for direction |
| `..` | Pinned on top regardless of sort |
| Grouping | None; folders and files mixed |
| CLI flag (`--sort`) | Not added; the setting only lives inside the picker |
| Persistence | Per picker run only |

## Technologies considered

| Option | Notes | Decision |
|--------|-------|----------|
| Letter keys (`s`, `r`) | Every printable key goes to the filter/path textinput, so letters would be typed into the filter | Rejected |
| Ctrl+S / Ctrl+R | Not used by the textinput. Bubble Tea puts the terminal in raw mode, which turns off XON/XOFF flow control, so Ctrl+S reaches the program instead of pausing output | **Chosen** |
| Single key cycling four orders | Fewer keys, but more presses to reach a given order and harder to predict | Rejected |
| `sort.SliceStable` | Stable, works on the `..`-trimmed sub-slice in place | **Used** |
| `slices.SortStableFunc` | Equivalent (Go 1.21+); either works | Not needed |
| `DirEntry.Info()` for mtime | Reuses the `ReadDir` result (lstat semantics, so a symlink sorts by the link's own time) | **Used** |
| Extra `os.Stat` per entry | Follows symlinks, but adds a syscall per entry | Rejected |

## How it works

All changes are in [`internal/cli/picker/model.go`](../../internal/cli/picker/model.go).

State:

```go
type sortField int

const (
	sortByDate sortField = iota
	sortByName
)

type dirEntry struct {
	// …
	ModTime time.Time
}

type model struct {
	// …
	sortBy   sortField
	sortDesc bool // date: newest first; name: Z-A
}
```

`newModel` starts with `sortBy: sortByDate, sortDesc: true`. `reload()` records each entry's modification time from `e.Info()`. If that fails, the time stays zero and the entry sorts as the oldest, so the listing still loads. `reload()` then calls `sortEntries()` instead of `applyFilter()`:

```go
func (m *model) sortEntries() {
	rest := m.entries
	if len(rest) > 0 && rest[0].Name == ".." {
		rest = rest[1:]
	}
	sort.SliceStable(rest, func(i, j int) bool {
		return m.entryLess(rest[i], rest[j])
	})
	m.applyFilter()
}

func (m *model) entryLess(a, b dirEntry) bool {
	if m.sortBy == sortByDate && !a.ModTime.Equal(b.ModTime) {
		if m.sortDesc {
			return a.ModTime.After(b.ModTime)
		}
		return a.ModTime.Before(b.ModTime)
	}
	if m.sortBy == sortByName && m.sortDesc {
		return nameLess(b.Name, a.Name)
	}
	return nameLess(a.Name, b.Name)
}
```

`applyFilter` copies matches from `m.entries` in order, so filtered results come out already sorted, and moving between folders re-sorts through `reload()`.

Key handling happens in `Update` before the textinput sees the key, the same way Tab and Enter are handled:

```go
case tea.KeyCtrlS:
	if m.sortBy == sortByDate {
		m.setSort(sortByName, false)
	} else {
		m.setSort(sortByDate, true)
	}
	return m, nil
case tea.KeyCtrlR:
	m.setSort(m.sortBy, !m.sortDesc)
	return m, nil
```

`setSort` remembers the highlighted name, re-sorts, and moves the cursor back to that name. `View` shows `sortLabel()` (for example `sort: name (A-Z)`) under the folder path, and the help line lists both keys.

## Tests

In [`internal/cli/picker/model_test.go`](../../internal/cli/picker/model_test.go), fixtures set distinct modification times with `os.Chtimes` (applied after all paths exist, since creating a child changes the parent folder's time). The names are chosen so that case-insensitive order differs from byte order.

- `TestDefaultSortNewestFirst`: `..` first, then newest to oldest.
- `TestCtrlSTogglesNameSort`: Ctrl+S gives case-insensitive A-Z without typing into the filter; pressing it again goes back to date, newest first.
- `TestCtrlRReverses`: date oldest first, then name A-Z, then Z-A; `..` stays on top.
- `TestSortKeepsHighlightedEntry`: the cursor stays on the same entry across Ctrl+S, Ctrl+R, Ctrl+S.
- `TestSortPersistsAcrossFilterAndDrill`: filtered results are sorted, and name A-Z carries over into a subfolder.
- `TestMultiPartSetSortsByFirstVolume`: a set with an old `part1` and a new `part2` sorts by `part1`.
- `TestViewShowsSortIndicator`: all four sort labels and the new help text appear in `View()`.

```bash
go test ./internal/cli/picker/
go test ./...
```

## Follow-ups

- A `--sort name|date` flag, or a remembered preference, if users want a different default.
- Optional folders-first grouping.
