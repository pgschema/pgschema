package cmd

import (
	"fmt"
	"os"

	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/plan"
	"github.com/spf13/cobra"
)

var schema1File string
var schema2File string
var format string

var PlanCmd = &cobra.Command{
	Use:   "plan",
	Short: "Generate migration plan",
	Long:  "Generate and display a migration plan showing changes between two SQL schema files",
	RunE:  runPlan,
}

func init() {
	PlanCmd.Flags().StringVar(&schema1File, "schema1", "", "Path to first SQL schema file (required)")
	PlanCmd.Flags().StringVar(&schema2File, "schema2", "", "Path to second SQL schema file (required)")
	PlanCmd.Flags().StringVar(&format, "format", "text", "Output format: text, json, preview")
	PlanCmd.MarkFlagRequired("schema1")
	PlanCmd.MarkFlagRequired("schema2")
}

func runPlan(cmd *cobra.Command, args []string) error {
	// Read schema1 file
	schema1Data, err := os.ReadFile(schema1File)
	if err != nil {
		return fmt.Errorf("failed to read schema1 file: %w", err)
	}

	// Read schema2 file
	schema2Data, err := os.ReadFile(schema2File)
	if err != nil {
		return fmt.Errorf("failed to read schema2 file: %w", err)
	}

	// Generate diff
	ddlDiff, err := diff.Diff(string(schema1Data), string(schema2Data))
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
