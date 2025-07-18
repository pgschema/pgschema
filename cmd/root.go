package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/pgschema/pgschema/cmd/apply"
	"github.com/pgschema/pgschema/cmd/dump"
	"github.com/pgschema/pgschema/cmd/plan"
	"github.com/pgschema/pgschema/internal/version"
	"github.com/spf13/cobra"
)

var Debug bool
var logger *slog.Logger

// Build-time variables set via ldflags
var (
	GitCommit = "unknown"
	BuildDate = "unknown"
)

var RootCmd = &cobra.Command{
	Use:   "pgschema",
	Short: "PostgreSQL schema dump and migration tool",
	Long: fmt.Sprintf(`pgschema is a CLI tool to dump and diff PostgreSQL schema.

Version: %s@%s %s %s

Commands:
  dump    Dump PostgreSQL schema
  plan    Generate migration plan
  apply   Apply schema migrations

Use "pgschema [command] --help" for more information about a command.`, 
		version.App(), GitCommit, platform(), BuildDate),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		setupLogger()
	},
}

func init() {
	RootCmd.PersistentFlags().BoolVar(&Debug, "debug", false, "Enable debug logging")
	RootCmd.AddCommand(dump.DumpCmd)
	RootCmd.AddCommand(plan.PlanCmd)
	RootCmd.AddCommand(apply.ApplyCmd)
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

// platform returns the OS/architecture combination
func platform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
