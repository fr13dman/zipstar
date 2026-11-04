package version

import (
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
)

func buildInfo(mainVersion string, settings ...string) *debug.BuildInfo {
	bi := &debug.BuildInfo{GoVersion: "go1.99.0", Main: debug.Module{Version: mainVersion}}
	for i := 0; i+1 < len(settings); i += 2 {
		bi.Settings = append(bi.Settings, debug.BuildSetting{Key: settings[i], Value: settings[i+1]})
	}
	return bi
}

func TestFromBuildInfo(t *testing.T) {
	vcs := []string{"vcs.revision", "5245830abcdef0123", "vcs.time", "2026-11-04T21:35:00Z", "vcs.modified", "true"}

	cases := []struct {
		name    string
		bi      *debug.BuildInfo
		stamped string
		want    Info
	}{
		{
			name: "local go build from a git checkout",
			bi:   buildInfo("(devel)", vcs...),
			want: Info{Version: "dev", Commit: "5245830", Modified: true, Time: "2026-11-04T21:35:00Z", Go: "go1.99.0"},
		},
		{
			name:    "installer stamps git describe",
			bi:      buildInfo("(devel)", vcs...),
			stamped: "v0.2.0",
			want:    Info{Version: "v0.2.0", Commit: "5245830", Modified: true, Time: "2026-11-04T21:35:00Z", Go: "go1.99.0"},
		},
		{
			name: "go install module@version",
			bi:   buildInfo("v0.3.1"),
			want: Info{Version: "v0.3.1", Go: "go1.99.0"},
		},
		{
			name: "no build info",
			bi:   nil,
			want: Info{Version: "dev", Go: runtime.Version()},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := fromBuildInfo(tc.bi, tc.stamped)
			tc.want.Platform = runtime.GOOS + "/" + runtime.GOARCH
			if got != tc.want {
				t.Fatalf("got %+v\nwant %+v", got, tc.want)
			}
		})
	}
}

func TestInfoString(t *testing.T) {
	full := Info{Version: "v0.2.0", Commit: "5245830", Modified: true, Time: "2026-11-04T21:35:00Z", Go: "go1.99.0", Platform: "darwin/arm64"}
	want := "zipstar v0.2.0\n" +
		"  commit: 5245830 (modified)\n" +
		"  built:  2026-11-04T21:35:00Z\n" +
		"  go:     go1.99.0 darwin/arm64\n"
	if got := full.String(); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}

	bare := Info{Version: "dev", Go: "go1.99.0", Platform: "linux/amd64"}
	if got := bare.String(); strings.Contains(got, "commit") || strings.Contains(got, "built") {
		t.Fatalf("unknown commit/time should be omitted:\n%s", got)
	}
}
