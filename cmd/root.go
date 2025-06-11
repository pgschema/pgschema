package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "pgschema",
	Short: "PostgreSQL schema diff tool",
	Long:  "A CLI tool to compare PostgreSQL schemas from directories or databases",
}

func init() {
	RootCmd.AddCommand(DiffCmd)
	RootCmd.AddCommand(VersionCmd)
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}