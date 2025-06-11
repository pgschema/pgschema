package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "pgschema",
	Short: "Simple CLI tool",
	Long:  "A simple CLI tool with version information",
}

func init() {
	RootCmd.AddCommand(VersionCmd)
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
