package cmd

import (
	"fmt"
	"runtime"

	"github.com/pgschema/pgschema/internal/version"
	"github.com/spf13/cobra"
)

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display the version number of pgschema",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("pgschema v%s@%s %s %s\n", version.App(), GitCommit, platform(), BuildDate)
	},
}

// Build-time variables set via ldflags
var (
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// platform returns the OS/architecture combination
func platform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}
