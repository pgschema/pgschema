package plan

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pgschema/pgschema/cmd/util"
	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/fingerprint"
	"github.com/pgschema/pgschema/internal/include"
	"github.com/pgschema/pgschema/ir"
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
	outputHuman  string
	outputJSON   string
	outputSQL    string
	planNoColor  bool
)


var PlanCmd = &cobra.Command{
	Use:          "plan",
	Short:        "Generate migration plan for a specific schema",
	Long:         "Generate a migration plan to apply a desired schema state to a target database schema. Compares the desired state (from --file) with the current state of a specific schema (specified by --schema, defaults to 'public').",
	RunE:         runPlan,
	SilenceUsage: true,
	PreRunE: util.PreRunEWithEnvVars(&planDB, &planUser),
}

func init() {
	// Target database connection flags
	PlanCmd.Flags().StringVar(&planHost, "host", util.GetEnvWithDefault("PGHOST", "localhost"), "Database server host (env: PGHOST)")
	PlanCmd.Flags().IntVar(&planPort, "port", util.GetEnvIntWithDefault("PGPORT", 5432), "Database server port (env: PGPORT)")
	PlanCmd.Flags().StringVar(&planDB, "db", "", "Database name (required) (env: PGDATABASE)")
	PlanCmd.Flags().StringVar(&planUser, "user", "", "Database user name (required) (env: PGUSER)")
	PlanCmd.Flags().StringVar(&planPassword, "password", "", "Database password (optional, can also use PGPASSWORD env var)")
	PlanCmd.Flags().StringVar(&planSchema, "schema", "public", "Schema name")

	// Desired state schema file flag
	PlanCmd.Flags().StringVar(&planFile, "file", "", "Path to desired state SQL schema file (required)")

	// Output flags
	PlanCmd.Flags().StringVar(&outputHuman, "output-human", "", "Output human-readable format to stdout or file path")
	PlanCmd.Flags().StringVar(&outputJSON, "output-json", "", "Output JSON format to stdout or file path")
	PlanCmd.Flags().StringVar(&outputSQL, "output-sql", "", "Output SQL format to stdout or file path")
	PlanCmd.Flags().BoolVar(&planNoColor, "no-color", false, "Disable colored output")

	PlanCmd.MarkFlagRequired("file")
}

func runPlan(cmd *cobra.Command, args []string) error {
	// Derive final password: use provided password or check environment variable
	finalPassword := planPassword
	if finalPassword == "" {
		if envPassword := os.Getenv("PGPASSWORD"); envPassword != "" {
			finalPassword = envPassword
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
	}

	// Generate plan
	migrationPlan, err := GeneratePlan(config)
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
}

// GeneratePlan generates a migration plan from configuration
func GeneratePlan(config *PlanConfig) (*plan.Plan, error) {
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
	currentStateIR, err := util.GetIRFromDatabaseWithIgnoreConfig(config.Host, config.Port, config.DB, config.User, config.Password, config.Schema, config.ApplicationName, ignoreConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get current state from database: %w", err)
	}

	// Compute fingerprint of current database state
	sourceFingerprint, err := fingerprint.ComputeFingerprint(currentStateIR, config.Schema)
	if err != nil {
		return nil, fmt.Errorf("failed to compute source fingerprint: %w", err)
	}

	// Parse desired state to IR with target schema context
	desiredParser := ir.NewParser(config.Schema, ignoreConfig)
	desiredStateIR, err := desiredParser.ParseSQL(desiredState)
	if err != nil {
		return nil, fmt.Errorf("failed to parse desired state schema file: %w", err)
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
}
