package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/pgschema/pgschema/cmd/dump"
	"github.com/pgschema/pgschema/cmd/plan"
	"github.com/spf13/cobra"
)

var Debug bool
var logger *slog.Logger

var RootCmd = &cobra.Command{
	Use:   "pgschema",
	Short: "Simple CLI tool",
	Long:  "A simple CLI tool with version information",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		setupLogger()
	},
}

func init() {
	RootCmd.PersistentFlags().BoolVar(&Debug, "debug", false, "Enable debug logging")
	RootCmd.AddCommand(VersionCmd)
	RootCmd.AddCommand(dump.DumpCmd)
	RootCmd.AddCommand(plan.PlanCmd)
}

func setupLogger() {
	level := slog.LevelInfo
	if Debug {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(os.Stderr, opts)
	logger = slog.New(handler)
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
