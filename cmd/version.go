package cmd

import (
	"fmt"

	"github.com/pgschema/pgschema/internal/version"
	"github.com/spf13/cobra"
)

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display the version number of pgschema",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("pgschema v%s@%s %s %s\n", version.Version(), version.GitCommit(), version.Platform(), version.GitDate())
	},
}
