// Command zipstar is the entrypoint for the zipstar archive CLI.
//
// Wiring only lives here: construct the format registry, register backends,
// and hand control to internal/cli. Keeping main thin makes formats swappable
// without touching command parsing.
package main

import (
	"os"

	"github.com/fr13dman/zipstar-cli/internal/cli"
	"github.com/fr13dman/zipstar-cli/internal/formats"
	zipfmt "github.com/fr13dman/zipstar-cli/internal/formats/zip"
)

func main() {
	reg := formats.NewRegistry()
	reg.Register(zipfmt.New())
	os.Exit(cli.Execute(reg))
}
