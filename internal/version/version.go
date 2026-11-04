// Package version reports which zipstar build is running.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

// Version is stamped at build time:
//
//	go build -ldflags "-X github.com/fr13dman/zipstar-cli/internal/version.Version=v1.2.3"
//
// The installers set it from `git describe --tags --always --dirty`.
var Version = ""

// Info describes the running build.
type Info struct {
	Version  string // release version, or "dev" for an unversioned build
	Commit   string // short VCS revision, if known
	Modified bool   // built from a working tree with uncommitted changes
	Time     string // commit time (RFC 3339), if known
	Go       string // Go toolchain version
	Platform string // GOOS/GOARCH
}

// Get returns information about the running binary.
func Get() Info {
	bi, _ := debug.ReadBuildInfo()
	return fromBuildInfo(bi, Version)
}

// String formats Get() for `zipstar --version`.
func String() string { return Get().String() }

func fromBuildInfo(bi *debug.BuildInfo, stamped string) Info {
	info := Info{
		Version:  "dev",
		Go:       runtime.Version(),
		Platform: runtime.GOOS + "/" + runtime.GOARCH,
	}
	if bi != nil {
		if bi.GoVersion != "" {
			info.Go = bi.GoVersion
		}
		// `go install module@v1.2.3` records the module version.
		if v := bi.Main.Version; v != "" && v != "(devel)" {
			info.Version = v
		}
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				info.Commit = s.Value
				if len(info.Commit) > 7 {
					info.Commit = info.Commit[:7]
				}
			case "vcs.time":
				info.Time = s.Value
			case "vcs.modified":
				info.Modified = s.Value == "true"
			}
		}
	}
	if stamped != "" {
		info.Version = stamped
	}
	return info
}

func (i Info) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "zipstar %s\n", i.Version)
	if i.Commit != "" {
		commit := i.Commit
		if i.Modified {
			commit += " (modified)"
		}
		fmt.Fprintf(&b, "  commit: %s\n", commit)
	}
	if i.Time != "" {
		fmt.Fprintf(&b, "  built:  %s\n", i.Time)
	}
	fmt.Fprintf(&b, "  go:     %s %s\n", i.Go, i.Platform)
	return b.String()
}
