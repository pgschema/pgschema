package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/ir"
	"github.com/pgschema/pgschema/internal/plan"
	"github.com/pgschema/pgschema/internal/utils"
	"github.com/spf13/cobra"
)

var (
	planHost     string
	planPort     int
	planDB       string
	planUser     string
	planPassword string
	planSchema   string
	planFile     string
	planFormat   string
)

var PlanCmd = &cobra.Command{
	Use:   "plan",
	Short: "Generate migration plan for a specific schema",
	Long:  "Generate a migration plan to apply a desired schema state to a target database schema. Compares the desired state (from --file) with the current state of a specific schema (specified by --schema, defaults to 'public').",
	RunE:  runPlan,
}

func init() {
	// Target database connection flags
	PlanCmd.Flags().StringVar(&planHost, "host", "localhost", "Database server host")
	PlanCmd.Flags().IntVar(&planPort, "port", 5432, "Database server port")
	PlanCmd.Flags().StringVar(&planDB, "db", "", "Database name (required)")
	PlanCmd.Flags().StringVar(&planUser, "user", "", "Database user name (required)")
	PlanCmd.Flags().StringVar(&planPassword, "password", "", "Database password (optional)")
	PlanCmd.Flags().StringVar(&planSchema, "schema", "public", "Schema name")

	// Desired state schema file flag
	PlanCmd.Flags().StringVar(&planFile, "file", "", "Path to desired state SQL schema file (required)")

	// Output format
	PlanCmd.Flags().StringVar(&planFormat, "format", "text", "Output format: text, json")

	// Mark required flags
	PlanCmd.MarkFlagRequired("db")
	PlanCmd.MarkFlagRequired("user")
	PlanCmd.MarkFlagRequired("file")
}

func runPlan(cmd *cobra.Command, args []string) error {
	// Derive final password: use flag if provided, otherwise check environment variable
	finalPassword := planPassword
	if finalPassword == "" {
		if envPassword := os.Getenv("PGPASSWORD"); envPassword != "" {
			finalPassword = envPassword
		}
	}

	// Get current state from target database
	currentState, err := getSchemaFromDatabase(planHost, planPort, planDB, planUser, finalPassword, planSchema)
	if err != nil {
		return fmt.Errorf("failed to get current state from database: %w", err)
	}

	// Get desired state from schema file
	desiredStateData, err := os.ReadFile(planFile)
	if err != nil {
		return fmt.Errorf("failed to read desired state schema file: %w", err)
	}
	desiredState := string(desiredStateData)

	// Generate diff (current -> desired)
	ddlDiff, err := diff.Diff(currentState, desiredState)
	if err != nil {
		return fmt.Errorf("failed to generate diff: %w", err)
	}

	// Create plan from diff
	migrationPlan := plan.NewPlan(ddlDiff)

	// Output based on format
	switch planFormat {
	case "json":
		jsonOutput, err := migrationPlan.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to generate JSON output: %w", err)
		}
		fmt.Print(jsonOutput)
	case "text":
		fallthrough
	default:
		fmt.Print(migrationPlan.Summary())
	}

	return nil
}



// getSchemaFromDatabase connects to a database and extracts schema using the IR system
func getSchemaFromDatabase(host string, port int, db, user, password, schemaName string) (string, error) {
	// Build database connection
	config := &utils.ConnectionConfig{
		Host:     host,
		Port:     port,
		Database: db,
		User:     user,
		Password: password,
		SSLMode:  "prefer",
	}

	conn, err := utils.Connect(config)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	ctx := context.Background()

	// Build schema using the IR system
	builder := ir.NewBuilder(conn)

	// Default to public schema if none specified
	targetSchema := schemaName
	if targetSchema == "" {
		targetSchema = "public"
	}

	schemaIR, err := builder.BuildSchema(ctx, targetSchema)
	if err != nil {
		return "", fmt.Errorf("failed to build schema: %w", err)
	}

	// Generate SQL output using unified SQL generator service
	sqlGenerator := ir.NewSQLGeneratorService(false) // Don't include comments for plan command
	return sqlGenerator.GenerateSchemaSQL(schemaIR, ""), nil
}
