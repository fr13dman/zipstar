package archive

import (
	"path/filepath"
	"strings"
	"testing"
)

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
