package plan

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pgschema/pgschema/cmd/util"
	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/include"
	"github.com/pgschema/pgschema/internal/ir"
	"github.com/pgschema/pgschema/internal/plan"
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
	planNoColor  bool
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
	PlanCmd.Flags().StringVar(&planFormat, "format", "human", "Output format: human, json, sql")
	PlanCmd.Flags().BoolVar(&planNoColor, "no-color", false, "Disable colored output")

	// Mark required flags
	PlanCmd.MarkFlagRequired("db")
	PlanCmd.MarkFlagRequired("user")
	PlanCmd.MarkFlagRequired("file")
}

func runPlan(cmd *cobra.Command, args []string) error {
	// Create plan configuration
	config := &PlanConfig{
		Host:            planHost,
		Port:            planPort,
		DB:              planDB,
		User:            planUser,
		Password:        planPassword,
		Schema:          planSchema,
		File:            planFile,
		ApplicationName: "pgschema",
	}

	// Generate plan
	migrationPlan, err := GeneratePlan(config)
	if err != nil {
		return err
	}

	// Output based on format
	switch planFormat {
	case "json":
		jsonOutput, err := migrationPlan.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to generate JSON output: %w", err)
		}
		fmt.Print(jsonOutput)
	case "sql":
		sqlOutput := migrationPlan.ToSQL(plan.SQLFormatRaw)
		fmt.Print(sqlOutput)
	case "human":
		fallthrough
	default:
		// Use colored output unless explicitly disabled
		fmt.Print(migrationPlan.HumanColored(!planNoColor))
	}

	return nil
}

// PlanConfig holds configuration for plan generation
type PlanConfig struct {
	Host            string
	Port            int
	DB              string
	User            string
	Password        string
	Schema          string
	File            string
	ApplicationName string
}

// GeneratePlan generates a migration plan from configuration
func GeneratePlan(config *PlanConfig) (*plan.Plan, error) {
	// Derive final password: use provided password or check environment variable
	finalPassword := config.Password
	if finalPassword == "" {
		if envPassword := os.Getenv("PGPASSWORD"); envPassword != "" {
			finalPassword = envPassword
		}
	}

	// Process desired state file with include directives
	processor := include.NewProcessor(filepath.Dir(config.File))
	desiredState, err := processor.ProcessFile(config.File)
	if err != nil {
		return nil, fmt.Errorf("failed to process desired state schema file: %w", err)
	}

	// Get current state from target database
	currentStateIR, err := getIRFromDatabase(config.Host, config.Port, config.DB, config.User, finalPassword, config.Schema, config.ApplicationName)
	if err != nil {
		return nil, fmt.Errorf("failed to get current state from database: %w", err)
	}

	// Parse desired state to IR
	desiredParser := ir.NewParser()
	desiredStateIR, err := desiredParser.ParseSQL(desiredState)
	if err != nil {
		return nil, fmt.Errorf("failed to parse desired state schema file: %w", err)
	}

	// Generate diff (current -> desired) using IR directly
	diffs := diff.GenerateMigration(currentStateIR, desiredStateIR, config.Schema)

	// Create plan from diffs
	migrationPlan := plan.NewPlan(diffs, config.Schema)

	return migrationPlan, nil
}

// getIRFromDatabase connects to a database and extracts schema using the IR system
func getIRFromDatabase(host string, port int, db, user, password, schemaName, applicationName string) (*ir.IR, error) {
	// Build database connection
	config := &util.ConnectionConfig{
		Host:            host,
		Port:            port,
		Database:        db,
		User:            user,
		Password:        password,
		SSLMode:         "prefer",
		ApplicationName: applicationName,
	}

	conn, err := util.Connect(config)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	ctx := context.Background()

	// Build IR using the IR system
	inspector := ir.NewInspector(conn)

	// Default to public schema if none specified
	targetSchema := schemaName
	if targetSchema == "" {
		targetSchema = "public"
	}

	schemaIR, err := inspector.BuildIR(ctx, targetSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to build IR: %w", err)
	}

	return schemaIR, nil
}

