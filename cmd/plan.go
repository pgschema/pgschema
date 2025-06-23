package cmd

import (
	"fmt"
	"os"

	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/plan"
	"github.com/spf13/cobra"
)

var oldFile string
var newFile string
var format string

var PlanCmd = &cobra.Command{
	Use:   "plan",
	Short: "Generate migration plan",
	Long:  "Generate and display a migration plan showing changes between two SQL schema files",
	RunE:  runPlan,
}

func init() {
	PlanCmd.Flags().StringVar(&oldFile, "old", "", "Path to old SQL schema file (required)")
	PlanCmd.Flags().StringVar(&newFile, "new", "", "Path to new SQL schema file (required)")
	PlanCmd.Flags().StringVar(&format, "format", "text", "Output format: text, json, preview")
	PlanCmd.MarkFlagRequired("old")
	PlanCmd.MarkFlagRequired("new")
}

func runPlan(cmd *cobra.Command, args []string) error {
	// Read old schema file
	oldData, err := os.ReadFile(oldFile)
	if err != nil {
		return fmt.Errorf("failed to read old schema file: %w", err)
	}

	// Read new schema file
	newData, err := os.ReadFile(newFile)
	if err != nil {
		return fmt.Errorf("failed to read new schema file: %w", err)
	}

	// Generate diff
	ddlDiff, err := diff.Diff(string(oldData), string(newData))
	if err != nil {
		return fmt.Errorf("failed to generate diff: %w", err)
	}

	// Create plan from diff
	migrationPlan := plan.NewPlan(ddlDiff)

	// Output based on format
	switch format {
	case "json":
		jsonOutput, err := migrationPlan.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to generate JSON output: %w", err)
		}
		fmt.Print(jsonOutput)
	case "preview":
		fmt.Print(migrationPlan.Preview())
	case "text":
		fallthrough
	default:
		fmt.Print(migrationPlan.Summary())
	}

	return nil
}