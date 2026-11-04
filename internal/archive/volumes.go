package archive

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// new-style: archive.part1.rar, archive.part01.rar, archive.part001.rar
var partRAR = regexp.MustCompile(`(?i)^(.+)\.part(\d+)\.rar$`)

// old-style continuation: archive.r00, archive.r01, … (not the lead .rar)
var oldContRAR = regexp.MustCompile(`(?i)^(.+)\.r(\d+)$`)

// ExtractTarget is one archive to open after glob expansion / multi-volume coalescing.
type ExtractTarget struct {
	Path        string   // first volume / archive path to open
	Volumes     []string // all matched volume paths (len >= 1); Path is the first volume
	VolumeCount int      // len(Volumes)
}

// VolumeOf reports whether base (a file name, not a path) is named like one
// volume of a multi-volume RAR set. key identifies the set (case-insensitive
// stem plus naming style); num orders volumes within it, lowest first.
//
// A plain "x.rar" is reported as the potential lead (num 0) of an old-style
// "x.rar, x.r00, x.r01, …" set; it is only a set when siblings exist.
func VolumeOf(base string) (key string, num int, ok bool) {
	if m := partRAR.FindStringSubmatch(base); m != nil {
		n, _ := strconv.Atoi(m[2])
		return strings.ToLower(m[1]) + "\x00part", n, true
	}
	if m := oldContRAR.FindStringSubmatch(base); m != nil {
		n, _ := strconv.Atoi(m[2])
		return strings.ToLower(m[1]) + "\x00old", n + 1, true
	}
	if strings.EqualFold(filepath.Ext(base), ".rar") {
		return strings.ToLower(strings.TrimSuffix(base, filepath.Ext(base))) + "\x00old", 0, true
	}
	return "", 0, false
}

type volume struct {
	path string
	num  int
}

// orderedPaths sorts vols first-volume-first and returns their paths.
func orderedPaths(vols []volume) []string {
	sort.Slice(vols, func(i, j int) bool {
		if vols[i].num != vols[j].num {
			return vols[i].num < vols[j].num
		}
		return vols[i].path < vols[j].path
	})
	out := make([]string, len(vols))
	for i, v := range vols {
		out[i] = v.path
	}
	return out
}

func targetOf(paths []string) ExtractTarget {
	return ExtractTarget{Path: paths[0], Volumes: paths, VolumeCount: len(paths)}
}

// CoalesceArchives collapses multi-volume RAR sets so only the first volume is
// extracted. Non-RAR paths (and standalone .rar files) pass through unchanged.
func CoalesceArchives(paths []string) []ExtractTarget {
	groups := map[string][]volume{}
	var out []ExtractTarget
	for _, p := range paths {
		key, num, ok := VolumeOf(filepath.Base(p))
		if !ok {
			out = append(out, targetOf([]string{p}))
			continue
		}
		k := filepath.Dir(p) + "\x00" + key
		groups[k] = append(groups[k], volume{path: p, num: num})
	}
	for _, vols := range groups {
		out = append(out, targetOf(orderedPaths(vols)))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// SiblingVolumes returns every volume of path's multi-volume set found in its
// directory, first volume first, or nil when path is not part of such a set.
func SiblingVolumes(path string) []string {
	key, _, ok := VolumeOf(filepath.Base(path))
	if !ok {
		return nil
	}
	dir := filepath.Dir(path)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var vols []volume
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		if k, n, ok := VolumeOf(e.Name()); ok && k == key {
			vols = append(vols, volume{path: filepath.Join(dir, e.Name()), num: n})
		}
	}
	if len(vols) < 2 {
		return nil
	}
	return orderedPaths(vols)
}

// DiscoverVolumes completes targets that name a single volume of a larger set
// (e.g. a picker selection or one explicit "x.part2.rar"): the target gains the
// set's other volumes and Path moves to the first volume, which is where
// extraction must start. Targets that end up identical are de-duplicated.
func DiscoverVolumes(targets []ExtractTarget) []ExtractTarget {
	seen := map[string]bool{}
	out := make([]ExtractTarget, 0, len(targets))
	for _, t := range targets {
		if t.VolumeCount <= 1 {
			if vols := SiblingVolumes(t.Path); vols != nil {
				t = targetOf(vols)
			}
		}
		if seen[t.Path] {
			continue
		}
		seen[t.Path] = true
		out = append(out, t)
	}
	return out
}
