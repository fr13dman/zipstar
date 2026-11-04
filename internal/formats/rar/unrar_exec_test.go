package rar

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/fr13dman/zipstar-cli/internal/formats"
)

func TestParseUnrarLine(t *testing.T) {
	dest := "/tmp/out"
	cases := []struct {
		line string
		kind unrarEvent
		want string
	}{
		{"Extracting from /data/big.part2.rar", unrarVolume, "/data/big.part2.rar"},
		{"Extracting  /tmp/out/exe/test.exe          \b\b\b\b  5%\b\b\b\b 16%", unrarFile, "exe/test.exe"},
		{"...         exe/test.exe                   \b\b\b\b 30%\b\b\b\b\b  OK ", unrarFile, "exe/test.exe"},
		{"Creating    /tmp/out/jpg                              OK", unrarFile, "jpg"},
		{"Extracting  /tmp/out/my file.txt   OK", unrarFile, "my file.txt"},
		{"UNRAR 7.23 freeware      Copyright (c) 1993-2026 Alexander Roshal", unrarNone, ""},
		{"All OK", unrarNone, ""},
		{"", unrarNone, ""},
	}
	for _, tc := range cases {
		kind, got := parseUnrarLine(tc.line, dest)
		if kind != tc.kind || got != tc.want {
			t.Errorf("line %q: got (%v, %q) want (%v, %q)", tc.line, kind, got, tc.kind, tc.want)
		}
	}
}

func TestFakeUnrarCallbacksDuringExtract(t *testing.T) {
	fake, err := filepath.Abs(filepath.Join("testdata", "fake_unrar.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(fake, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZIPSTAR_UNRAR", fake)

	archive := filepath.Join(t.TempDir(), "dummy.rar")
	if err := os.WriteFile(archive, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()

	var mu sync.Mutex
	var files, volumes []string
	firstCB := make(chan struct{})
	var once sync.Once
	done := make(chan error, 1)
	go func() {
		done <- New().Unarchive(context.Background(), formats.UnarchiveOpts{
			Archive: archive,
			DestDir: dest,
			Force:   true,
			OnFile: func(name string) {
				mu.Lock()
				files = append(files, name)
				mu.Unlock()
				once.Do(func() { close(firstCB) })
			},
			OnVolume: func(path string) {
				mu.Lock()
				volumes = append(volumes, filepath.Base(path))
				mu.Unlock()
			},
		})
	}()

	select {
	case <-firstCB:
	case err := <-done:
		t.Fatalf("Unarchive returned before any OnFile: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for OnFile during extract")
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	// Dest prefix stripped and the "..." continuation de-duplicated.
	if want := []string{"a.txt", "b.txt"}; !reflect.DeepEqual(files, want) {
		t.Fatalf("files %#v want %#v", files, want)
	}
	if want := []string{"dummy.rar", "dummy.part2.rar"}; !reflect.DeepEqual(volumes, want) {
		t.Fatalf("volumes %#v want %#v", volumes, want)
	}
}

func TestFakeUnrarListDedupesAcrossVolumes(t *testing.T) {
	fake, err := filepath.Abs(filepath.Join("testdata", "fake_unrar.sh"))
	if err != nil {
		t.Fatal(err)
	}
	entries, err := listUnrar(context.Background(), fake, formats.ListOpts{Archive: "dummy.rar"})
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name)
	}
	if want := []string{"a.txt", "b.txt"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("names %#v want %#v", names, want)
	}
}

// Multi-volume fixture, both engines: every member from every volume lands in
// the one destination, and each volume is reported.
func TestMultiVolumeAllVolumesSameDest(t *testing.T) {
	for _, engine := range []string{"unrar", "rardecode"} {
		t.Run(engine, func(t *testing.T) {
			if engine == "unrar" {
				t.Setenv("ZIPSTAR_UNRAR", "")
				if _, ok := FindUnrar(); !ok {
					t.Skip("UnRAR not available; run ./scripts/fetch-unrar.sh")
				}
			} else {
				t.Setenv("ZIPSTAR_UNRAR", "off")
			}

			archive := filepath.Join("testdata", "multivol", "multi.part01.rar")
			f := New()

			entries, err := f.List(context.Background(), formats.ListOpts{Archive: archive})
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 6 {
				t.Fatalf("List across volumes: got %d entries, want 6", len(entries))
			}

			dest := t.TempDir()
			var files, volumes []string
			err = f.Unarchive(context.Background(), formats.UnarchiveOpts{
				Archive:  archive,
				DestDir:  dest,
				OnFile:   func(name string) { files = append(files, name) },
				OnVolume: func(path string) { volumes = append(volumes, filepath.Base(path)) },
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(files) != 6 {
				t.Fatalf("OnFile calls %d want 6: %#v", len(files), files)
			}
			if len(volumes) != 6 {
				t.Fatalf("OnVolume calls %d want 6: %#v", len(volumes), volumes)
			}
			for name, size := range map[string]int64{
				"exe/test.exe": 45056,
				"jpg/test.jpg": 40372,
				"тест.txt":     15498,
			} {
				info, err := os.Stat(filepath.Join(dest, name))
				if err != nil {
					t.Fatalf("%s missing from shared dest: %v", name, err)
				}
				if info.Size() != size {
					t.Fatalf("%s size %d want %d", name, info.Size(), size)
				}
			}
		})
	}
}
