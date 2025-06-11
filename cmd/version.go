package cmd

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed VERSION
var versionFile string

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display the version number of pgschema",
	Run: func(cmd *cobra.Command, args []string) {
		version := strings.TrimSpace(versionFile)
		fmt.Printf("pgschema version %s\n", version)
	},
}