package plan

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pgschema/pgschema/cmd/util"
	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/fingerprint"
	"github.com/pgschema/pgschema/internal/include"
	"github.com/pgschema/pgschema/internal/plan"
	"github.com/pgschema/pgschema/internal/postgres"
	"github.com/pgschema/pgschema/ir"
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
	outputHuman  string
	outputJSON   string
	outputSQL    string
	planNoColor  bool

	// Plan database flags (optional - if not provided, uses embedded postgres)
	planDBHost     string
	planDBPort     int
	planDBDatabase string
	planDBUser     string
	planDBPassword string
)

var PlanCmd = &cobra.Command{
	Use:          "plan",
	Short:        "Generate migration plan for a specific schema",
	Long:         "Generate a migration plan to apply a desired schema state to a target database schema. Compares the desired state (from --file) with the current state of a specific schema (specified by --schema, defaults to 'public').",
	RunE:         runPlan,
	SilenceUsage: true,
	PreRunE:      util.PreRunEWithEnvVarsAndConnection(&planDB, &planUser, &planHost, &planPort),
}

func init() {
	// Target database connection flags
	PlanCmd.Flags().StringVar(&planHost, "host", "localhost", "Database server host (env: PGHOST)")
	PlanCmd.Flags().IntVar(&planPort, "port", 5432, "Database server port (env: PGPORT)")
	PlanCmd.Flags().StringVar(&planDB, "db", "", "Database name (required) (env: PGDATABASE)")
	PlanCmd.Flags().StringVar(&planUser, "user", "", "Database user name (required) (env: PGUSER)")
	PlanCmd.Flags().StringVar(&planPassword, "password", "", "Database password (optional, can also use PGPASSWORD env var)")
	PlanCmd.Flags().StringVar(&planSchema, "schema", "public", "Schema name")

	// Desired state schema file flag
	PlanCmd.Flags().StringVar(&planFile, "file", "", "Path to desired state SQL schema file (required)")

	// Plan database connection flags (optional - for using external database instead of embedded postgres)
	PlanCmd.Flags().StringVar(&planDBHost, "plan-host", "", "Plan database host (env: PGSCHEMA_PLAN_HOST). If provided, uses external database instead of embedded postgres")
	PlanCmd.Flags().IntVar(&planDBPort, "plan-port", 5432, "Plan database port (env: PGSCHEMA_PLAN_PORT)")
	PlanCmd.Flags().StringVar(&planDBDatabase, "plan-db", "", "Plan database name (env: PGSCHEMA_PLAN_DB)")
	PlanCmd.Flags().StringVar(&planDBUser, "plan-user", "", "Plan database user (env: PGSCHEMA_PLAN_USER)")
	PlanCmd.Flags().StringVar(&planDBPassword, "plan-password", "", "Plan database password (env: PGSCHEMA_PLAN_PASSWORD)")

	// Output flags
	PlanCmd.Flags().StringVar(&outputHuman, "output-human", "", "Output human-readable format to stdout or file path")
	PlanCmd.Flags().StringVar(&outputJSON, "output-json", "", "Output JSON format to stdout or file path")
	PlanCmd.Flags().StringVar(&outputSQL, "output-sql", "", "Output SQL format to stdout or file path")
	PlanCmd.Flags().BoolVar(&planNoColor, "no-color", false, "Disable colored output")

	PlanCmd.MarkFlagRequired("file")
}

func runPlan(cmd *cobra.Command, args []string) error {
	// Apply environment variables to plan database flags
	util.ApplyPlanDBEnvVars(cmd, &planDBHost, &planDBDatabase, &planDBUser, &planDBPassword, &planDBPort)

	// Validate plan database flags if plan-host is provided
	if err := util.ValidatePlanDBFlags(planDBHost, planDBDatabase, planDBUser); err != nil {
		return err
	}

	// Derive final password: use provided password or check environment variable
	finalPassword := planPassword
	if finalPassword == "" {
		if envPassword := os.Getenv("PGPASSWORD"); envPassword != "" {
			finalPassword = envPassword
		}
	}

	// Derive final plan database password
	finalPlanPassword := planDBPassword
	if finalPlanPassword == "" {
		if envPassword := os.Getenv("PGSCHEMA_PLAN_PASSWORD"); envPassword != "" {
			finalPlanPassword = envPassword
		}
	}

	// Create plan configuration
	config := &PlanConfig{
		Host:            planHost,
		Port:            planPort,
		DB:              planDB,
		User:            planUser,
		Password:        finalPassword,
		Schema:          planSchema,
		File:            planFile,
		ApplicationName: "pgschema",
		// Plan database configuration
		PlanDBHost:     planDBHost,
		PlanDBPort:     planDBPort,
		PlanDBDatabase: planDBDatabase,
		PlanDBUser:     planDBUser,
		PlanDBPassword: finalPlanPassword,
	}

	// Create desired state provider (embedded postgres or external database)
	provider, err := CreateDesiredStateProvider(config)
	if err != nil {
		return err
	}
	defer provider.Stop()

	// Generate plan
	migrationPlan, err := GeneratePlan(config, provider)
	if err != nil {
		return err
	}

	// Determine which outputs to generate
	outputs, err := determineOutputs()
	if err != nil {
		return err
	}

	// Process each output
	for _, output := range outputs {
		if err := processOutput(migrationPlan, output, cmd); err != nil {
			return err
		}
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
	// Plan database configuration (optional - for external database)
	PlanDBHost     string
	PlanDBPort     int
	PlanDBDatabase string
	PlanDBUser     string
	PlanDBPassword string
}

// CreateDesiredStateProvider creates either an embedded PostgreSQL instance or connects to an external database
// for validating the desired state schema. The caller is responsible for calling Stop() on the returned provider.
func CreateDesiredStateProvider(config *PlanConfig) (postgres.DesiredStateProvider, error) {
	// Detect target database PostgreSQL version (needed for both embedded and external)
	pgVersion, err := postgres.DetectPostgresVersionFromDB(
		config.Host,
		config.Port,
		config.DB,
		config.User,
		config.Password,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to detect PostgreSQL version: %w", err)
	}

	// Extract major version from the target database's version string (e.g., "16.9.0" -> 16).
	// The version string format is "XX.Y.Z" where XX is the major version.
	var targetMajorVersion int
	_, err = fmt.Sscanf(string(pgVersion), "%d.", &targetMajorVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PostgreSQL version %s: %w", pgVersion, err)
	}

	// If plan-host is provided, use external database
	if config.PlanDBHost != "" {
		externalConfig := &postgres.ExternalDatabaseConfig{
			Host:               config.PlanDBHost,
			Port:               config.PlanDBPort,
			Database:           config.PlanDBDatabase,
			Username:           config.PlanDBUser,
			Password:           config.PlanDBPassword,
			TargetMajorVersion: targetMajorVersion,
		}
		return postgres.NewExternalDatabase(externalConfig)
	}

	// Otherwise, use embedded PostgreSQL
	return CreateEmbeddedPostgresForPlan(config, pgVersion)
}

// CreateEmbeddedPostgresForPlan creates a temporary embedded PostgreSQL instance
// for validating the desired state schema. The instance should be stopped by the caller.
func CreateEmbeddedPostgresForPlan(config *PlanConfig, pgVersion postgres.PostgresVersion) (*postgres.EmbeddedPostgres, error) {
	// Start embedded PostgreSQL with matching version
	embeddedConfig := &postgres.EmbeddedPostgresConfig{
		Version:  pgVersion,
		Database: "pgschema_temp",
		Username: "pgschema",
		Password: "pgschema",
	}
	embeddedPG, err := postgres.StartEmbeddedPostgres(embeddedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to start embedded PostgreSQL: %w", err)
	}

	return embeddedPG, nil
}

// GeneratePlan generates a migration plan from configuration.
// The caller must provide a non-nil provider instance for validating the desired state schema.
// The caller is responsible for managing the provider lifecycle (creation and cleanup).
func GeneratePlan(config *PlanConfig, provider postgres.DesiredStateProvider) (*plan.Plan, error) {
	// Load ignore configuration
	ignoreConfig, err := util.LoadIgnoreFileWithStructure()
	if err != nil {
		return nil, fmt.Errorf("failed to load .pgschemaignore: %w", err)
	}

	// Process desired state file with include directives
	processor := include.NewProcessor(filepath.Dir(config.File))
	desiredState, err := processor.ProcessFile(config.File)
	if err != nil {
		return nil, fmt.Errorf("failed to process desired state schema file: %w", err)
	}

	// Get current state from target database
	currentStateIR, err := util.GetIRFromDatabase(config.Host, config.Port, config.DB, config.User, config.Password, config.Schema, config.ApplicationName, ignoreConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get current state from database: %w", err)
	}

	// Compute fingerprint of current database state
	sourceFingerprint, err := fingerprint.ComputeFingerprint(currentStateIR, config.Schema)
	if err != nil {
		return nil, fmt.Errorf("failed to compute source fingerprint: %w", err)
	}

	ctx := context.Background()

	// Apply desired state SQL to the provider (embedded postgres or external database)
	if err := provider.ApplySchema(ctx, config.Schema, desiredState); err != nil {
		return nil, fmt.Errorf("failed to apply desired state: %w", err)
	}

	// Inspect the provider database to get desired state IR
	providerHost, providerPort, providerDB, providerUsername, providerPassword := provider.GetConnectionDetails()

	// Determine which schema to inspect
	// For external database: use the temporary schema name
	// For embedded postgres: use the config.Schema (GetSchemaName returns empty string)
	schemaToInspect := provider.GetSchemaName()
	if schemaToInspect == "" {
		schemaToInspect = config.Schema
	}

	desiredStateIR, err := util.GetIRFromDatabase(providerHost, providerPort, providerDB, providerUsername, providerPassword, schemaToInspect, config.ApplicationName, ignoreConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get desired state: %w", err)
	}

	// If using external database with temporary schema, normalize the schema name in the desired state IR
	// to match the target schema. This ensures that generated DDL uses the target schema name, not the
	// temporary schema name from the external database.
	if schemaToInspect != config.Schema {
		normalizeSchemaNames(desiredStateIR, schemaToInspect, config.Schema)
	}

	// Generate diff (current -> desired) using IR directly
	diffs := diff.GenerateMigration(currentStateIR, desiredStateIR, config.Schema)

	// Create plan from diffs with fingerprint
	migrationPlan := plan.NewPlanWithFingerprint(diffs, sourceFingerprint)

	return migrationPlan, nil
}

// outputSpec represents a single output specification
type outputSpec struct {
	format string // "human", "json", or "sql"
	target string // "stdout" or file path
}

// determineOutputs parses the output flags and returns the list of outputs to generate
func determineOutputs() ([]outputSpec, error) {
	var outputs []outputSpec
	stdoutCount := 0

	// Check each output flag
	if outputHuman != "" {
		if outputHuman == "stdout" {
			stdoutCount++
		}
		outputs = append(outputs, outputSpec{format: "human", target: outputHuman})
	}

	if outputJSON != "" {
		if outputJSON == "stdout" {
			stdoutCount++
		}
		outputs = append(outputs, outputSpec{format: "json", target: outputJSON})
	}

	if outputSQL != "" {
		if outputSQL == "stdout" {
			stdoutCount++
		}
		outputs = append(outputs, outputSpec{format: "sql", target: outputSQL})
	}

	// Validate only one stdout
	if stdoutCount > 1 {
		return nil, fmt.Errorf("only one output format can use stdout")
	}

	// Default behavior: if no outputs specified, output human to stdout
	if len(outputs) == 0 {
		outputs = append(outputs, outputSpec{format: "human", target: "stdout"})
	}

	return outputs, nil
}

// processOutput writes the plan in the specified format to the target destination
func processOutput(migrationPlan *plan.Plan, output outputSpec, cmd *cobra.Command) error {
	var content string
	var err error

	// Generate content based on format
	switch output.format {
	case "human":
		// For human format, use colored output when writing to stdout, unless explicitly disabled
		useColor := output.target == "stdout" && !planNoColor
		content = migrationPlan.HumanColored(useColor)
	case "json":
		// Check if debug flag is set on the root command
		debug, _ := cmd.Root().PersistentFlags().GetBool("debug")
		content, err = migrationPlan.ToJSONWithDebug(debug)
		if err != nil {
			return fmt.Errorf("failed to generate JSON output: %w", err)
		}
		content += "\n"
	case "sql":
		content = migrationPlan.ToSQL(plan.SQLFormatRaw)
	default:
		return fmt.Errorf("unknown output format: %s", output.format)
	}

	// Write to target
	if output.target == "stdout" {
		fmt.Print(content)
	} else {
		// Write to file
		if err := os.WriteFile(output.target, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s output to %s: %w", output.format, output.target, err)
		}
	}

	return nil
}

// normalizeSchemaNames replaces all occurrences of fromSchema with toSchema in the IR.
// This is used when inspecting an external database with a temporary schema name - we need
// to normalize the schema names in the IR to match the target schema for proper DDL generation.
func normalizeSchemaNames(irData *ir.IR, fromSchema, toSchema string) {
	// Normalize schema names in Schemas map
	if schema, exists := irData.Schemas[fromSchema]; exists {
		delete(irData.Schemas, fromSchema)
		schema.Name = toSchema
		irData.Schemas[toSchema] = schema

		// Normalize schema names in all objects within this schema
		// Tables
		for _, table := range schema.Tables {
			table.Schema = toSchema
		}

		// Views
		for _, view := range schema.Views {
			view.Schema = toSchema
		}

		// Functions
		for _, fn := range schema.Functions {
			fn.Schema = toSchema
		}

		// Procedures
		for _, proc := range schema.Procedures {
			proc.Schema = toSchema
		}

		// Types
		for _, typ := range schema.Types {
			typ.Schema = toSchema
		}

		// Sequences
		for _, seq := range schema.Sequences {
			seq.Schema = toSchema
		}

		// Aggregates
		for _, agg := range schema.Aggregates {
			agg.Schema = toSchema
		}
	}
}

// ResetFlags resets all global flag variables to their default values for testing
func ResetFlags() {
	planHost = "localhost"
	planPort = 5432
	planDB = ""
	planUser = ""
	planPassword = ""
	planSchema = "public"
	planFile = ""
	outputHuman = ""
	outputJSON = ""
	outputSQL = ""
	planNoColor = false
	planDBHost = ""
	planDBPort = 5432
	planDBDatabase = ""
	planDBUser = ""
	planDBPassword = ""
}
