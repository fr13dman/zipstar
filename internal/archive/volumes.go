package archive

import (
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

// CoalesceArchives collapses multi-volume RAR sets so only the first volume is
// extracted. Non-RAR paths (and standalone .rar files) pass through unchanged.
func CoalesceArchives(paths []string) []ExtractTarget {
	type setInfo struct {
		key      string
		first    string
		members  []string
		firstNum int
	}

	byKey := map[string]*setInfo{}
	var passthrough []string

	for _, p := range paths {
		base := filepath.Base(p)
		dir := filepath.Dir(p)

		if m := partRAR.FindStringSubmatch(base); m != nil {
			n, _ := strconv.Atoi(m[2])
			key := filepath.Join(dir, strings.ToLower(m[1])+".part.rar")
			si := byKey[key]
			if si == nil {
				si = &setInfo{key: key, firstNum: -1}
				byKey[key] = si
			}
			si.members = append(si.members, p)
			if si.firstNum < 0 || n < si.firstNum {
				si.firstNum = n
				si.first = p
			}
			continue
		}

		if m := oldContRAR.FindStringSubmatch(base); m != nil {
			key := filepath.Join(dir, strings.ToLower(m[1])+".oldrar")
			si := byKey[key]
			if si == nil {
				si = &setInfo{key: key, firstNum: 1 << 30}
				byKey[key] = si
			}
			si.members = append(si.members, p)
			continue
		}

		if strings.EqualFold(filepath.Ext(base), ".rar") {
			stem := strings.TrimSuffix(base, filepath.Ext(base))
			key := filepath.Join(dir, strings.ToLower(stem)+".oldrar")
			if si, ok := byKey[key]; ok {
				si.members = append(si.members, p)
				si.first = p
				si.firstNum = 0
				continue
			}
			si := &setInfo{key: key, first: p, firstNum: 0, members: []string{p}}
			byKey[key] = si
			continue
		}

		passthrough = append(passthrough, p)
	}

	var keys []string
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var out []ExtractTarget
	for _, k := range keys {
		si := byKey[k]
		sort.Strings(si.members)
		if strings.HasSuffix(k, ".oldrar") {
			if len(si.members) == 1 && si.first != "" {
				out = append(out, ExtractTarget{
					Path:        si.first,
					Volumes:     []string{si.first},
					VolumeCount: 1,
				})
				continue
			}
			if si.first == "" {
				sort.Slice(si.members, func(i, j int) bool {
					return filepath.Base(si.members[i]) < filepath.Base(si.members[j])
				})
				si.first = si.members[0]
			}
		}
		if si.first == "" {
			si.first = si.members[0]
		}
		vols := append([]string(nil), si.members...)
		out = append(out, ExtractTarget{
			Path:        si.first,
			Volumes:     vols,
			VolumeCount: len(vols),
		})
	}

	for _, p := range passthrough {
		out = append(out, ExtractTarget{
			Path:        p,
			Volumes:     []string{p},
			VolumeCount: 1,
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
