package archive

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestVolumeOf(t *testing.T) {
	cases := []struct {
		name    string
		ok      bool
		num     int
		sameSet string // another name that must share the key
	}{
		{"29090D43[ddpanda].part1.rar", true, 1, "29090D43[ddpanda].part2.rar"},
		{"Movie.PART02.RAR", true, 2, "movie.part10.rar"},
		{"pack.rar", true, 0, "pack.r00"},
		{"pack.r00", true, 1, "pack.r01"},
		{"notes.txt", false, 0, ""},
		{"archive.zip", false, 0, ""},
	}
	for _, tc := range cases {
		key, num, ok := VolumeOf(tc.name)
		if ok != tc.ok || (ok && num != tc.num) {
			t.Errorf("%s: got (num %d, ok %v) want (num %d, ok %v)", tc.name, num, ok, tc.num, tc.ok)
		}
		if tc.sameSet != "" {
			if other, _, _ := VolumeOf(tc.sameSet); other != key {
				t.Errorf("%s and %s should be one set", tc.name, tc.sameSet)
			}
		}
	}
	a, _, _ := VolumeOf("x.part1.rar")
	b, _, _ := VolumeOf("x.rar")
	if a == b {
		t.Error("new-style x.partN.rar and old-style x.rar sets must not merge")
	}
}

func writeFiles(t *testing.T, dir string, names ...string) {
	t.Helper()
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// One volume given (picker selection or explicit arg) → whole set, from part 1.
func TestDiscoverVolumesFromAnyPart(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir,
		"29090D43[ddpanda].part1.rar", "29090D43[ddpanda].part2.rar", "29090D43[ddpanda].part3.rar",
		"solo.rar", "other.part1.rar")
	part := func(n string) string { return filepath.Join(dir, "29090D43[ddpanda].part"+n+".rar") }

	for _, given := range []string{part("1"), part("2"), part("3")} {
		got := DiscoverVolumes(CoalesceArchives([]string{given}))
		if len(got) != 1 {
			t.Fatalf("%s: got %d targets", given, len(got))
		}
		if got[0].Path != part("1") || got[0].VolumeCount != 3 {
			t.Fatalf("%s: target %+v want first volume %s with 3 volumes", given, got[0], part("1"))
		}
		if want := []string{part("1"), part("2"), part("3")}; !reflect.DeepEqual(got[0].Volumes, want) {
			t.Fatalf("volumes %v want %v", got[0].Volumes, want)
		}
	}

	// A standalone archive and a one-part "set" stay single.
	for _, name := range []string{"solo.rar", "other.part1.rar"} {
		got := DiscoverVolumes(CoalesceArchives([]string{filepath.Join(dir, name)}))
		if len(got) != 1 || got[0].VolumeCount != 1 {
			t.Fatalf("%s: got %+v", name, got)
		}
	}
}

func TestDiscoverVolumesDedupes(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, "x.part1.rar", "x.part2.rar")
	targets := []ExtractTarget{
		targetOf([]string{filepath.Join(dir, "x.part2.rar")}),
		targetOf([]string{filepath.Join(dir, "x.part1.rar")}),
	}
	if got := DiscoverVolumes(targets); len(got) != 1 {
		t.Fatalf("same set given twice should extract once, got %+v", got)
	}
}

func TestCoalesceArchivesPartSet(t *testing.T) {
	dir := "/tmp/vols"
	inputs := []string{
		filepath.Join(dir, "multi.part03.rar"),
		filepath.Join(dir, "multi.part01.rar"),
		filepath.Join(dir, "multi.part02.rar"),
		filepath.Join(dir, "other.zip"),
	}
	got := CoalesceArchives(inputs)
	if len(got) != 2 {
		t.Fatalf("got %#v (want first volume + zip)", got)
	}
	var sawFirst, sawZip bool
	for _, g := range got {
		if strings.HasSuffix(g.Path, "multi.part01.rar") {
			sawFirst = true
			if g.VolumeCount != 3 {
				t.Fatalf("volume count %d", g.VolumeCount)
			}
		}
		if strings.HasSuffix(g.Path, "other.zip") {
			sawZip = true
			if g.VolumeCount != 1 {
				t.Fatalf("zip volume count %d", g.VolumeCount)
			}
		}
		if strings.Contains(g.Path, "part02") || strings.Contains(g.Path, "part03") {
			t.Fatalf("should not extract continuation %s", g.Path)
		}
	}
	if !sawFirst || !sawZip {
		t.Fatalf("got %#v", got)
	}
}

func TestCoalesceArchivesStandaloneRAR(t *testing.T) {
	got := CoalesceArchives([]string{"/data/solo.rar"})
	if len(got) != 1 || got[0].Path != "/data/solo.rar" || got[0].VolumeCount != 1 {
		t.Fatalf("got %#v", got)
	}
}

func TestCoalesceArchivesOldStyle(t *testing.T) {
	got := CoalesceArchives([]string{
		"/data/pack.r00",
		"/data/pack.rar",
		"/data/pack.r01",
	})
	if len(got) != 1 || !strings.HasSuffix(got[0].Path, "pack.rar") {
		t.Fatalf("got %#v", got)
	}
	if got[0].VolumeCount != 3 {
		t.Fatalf("volume count %d", got[0].VolumeCount)
	}
}
