package apply

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pgschema/pgschema/cmd/util"
	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/ir"
	"github.com/pgschema/pgschema/internal/plan"
	"github.com/spf13/cobra"
)

var (
	applyHost        string
	applyPort        int
	applyDB          string
	applyUser        string
	applyPassword    string
	applySchema      string
	applyFile        string
	applyAutoApprove bool
	applyNoColor     bool
	applyDryRun      bool
)

var ApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply migration plan to update a database schema",
	Long:  "Apply a desired schema state to a target database schema. Compares the desired state (from --file) with the current state of a specific schema (specified by --schema, defaults to 'public') and applies the necessary changes.",
	RunE:  runApply,
}

func init() {
	// Target database connection flags
	ApplyCmd.Flags().StringVar(&applyHost, "host", "localhost", "Database server host")
	ApplyCmd.Flags().IntVar(&applyPort, "port", 5432, "Database server port")
	ApplyCmd.Flags().StringVar(&applyDB, "db", "", "Database name (required)")
	ApplyCmd.Flags().StringVar(&applyUser, "user", "", "Database user name (required)")
	ApplyCmd.Flags().StringVar(&applyPassword, "password", "", "Database password (optional)")
	ApplyCmd.Flags().StringVar(&applySchema, "schema", "public", "Schema name")

	// Desired state schema file flag
	ApplyCmd.Flags().StringVar(&applyFile, "file", "", "Path to desired state SQL schema file (required)")

	// Auto-approve flag
	ApplyCmd.Flags().BoolVar(&applyAutoApprove, "auto-approve", false, "Apply changes without prompting for approval")
	ApplyCmd.Flags().BoolVar(&applyNoColor, "no-color", false, "Disable colored output")
	ApplyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Show plan without applying changes")

	// Mark required flags
	ApplyCmd.MarkFlagRequired("db")
	ApplyCmd.MarkFlagRequired("user")
	ApplyCmd.MarkFlagRequired("file")
}

func runApply(cmd *cobra.Command, args []string) error {
	// Derive final password: use flag if provided, otherwise check environment variable
	finalPassword := applyPassword
	if finalPassword == "" {
		if envPassword := os.Getenv("PGPASSWORD"); envPassword != "" {
			finalPassword = envPassword
		}
	}

	// Validate desired state file before connecting to the database
	desiredStateData, err := os.ReadFile(applyFile)
	if err != nil {
		return fmt.Errorf("failed to read desired state schema file: %w", err)
	}
	desiredState := string(desiredStateData)

	// Get current state from target database
	currentStateIR, err := getIRFromDatabase(applyHost, applyPort, applyDB, applyUser, finalPassword, applySchema)
	if err != nil {
		return fmt.Errorf("failed to get current state from database: %w", err)
	}

	// Parse desired state to IR
	desiredParser := ir.NewParser()
	desiredStateIR, err := desiredParser.ParseSQL(desiredState)
	if err != nil {
		return fmt.Errorf("failed to parse desired state schema file: %w", err)
	}

	// Generate diff (current -> desired) using IR directly
	ddlDiff := diff.Diff(currentStateIR, desiredStateIR)

	// Create plan from diff
	migrationPlan := plan.NewPlan(ddlDiff, applySchema)

	// Check if there are any changes to apply by examining the diff
	hasChanges := hasAnyChanges(ddlDiff)
	if !hasChanges {
		fmt.Println("No changes to apply. Database schema is already up to date.")
		return nil
	}

	// Display the plan
	fmt.Print(migrationPlan.HumanColored(!applyNoColor))

	// If dry-run, just print the plan and return
	if applyDryRun {
		return nil
	}

	// Prompt for approval if not auto-approved
	if !applyAutoApprove {
		fmt.Print("\nDo you want to apply these changes? (yes/no): ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read user input: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "yes" && response != "y" {
			fmt.Println("Apply cancelled.")
			return nil
		}
	}

	// Apply the changes
	fmt.Println("\nApplying changes...")

	// Build database connection for applying changes
	config := &util.ConnectionConfig{
		Host:     applyHost,
		Port:     applyPort,
		Database: applyDB,
		User:     applyUser,
		Password: finalPassword,
		SSLMode:  "prefer",
	}

	conn, err := util.Connect(config)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close()

	ctx := context.Background()

	// Generate SQL statements from the plan
	sqlStatements := migrationPlan.ToSQL()

	// Skip execution if no changes
	if strings.TrimSpace(sqlStatements) == "-- No changes detected" || strings.TrimSpace(sqlStatements) == "-- No DDL statements generated" {
		fmt.Println("No SQL statements to execute.")
		return nil
	}

	// Execute the SQL statements
	_, err = conn.ExecContext(ctx, sqlStatements)
	if err != nil {
		return fmt.Errorf("failed to apply changes: %w", err)
	}

	fmt.Println("Changes applied successfully!")
	return nil
}

// getIRFromDatabase connects to a database and extracts schema using the IR system
func getIRFromDatabase(host string, port int, db, user, password, schemaName string) (*ir.IR, error) {
	// Build database connection
	config := &util.ConnectionConfig{
		Host:     host,
		Port:     port,
		Database: db,
		User:     user,
		Password: password,
		SSLMode:  "prefer",
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

// hasAnyChanges checks if the DDLDiff contains any changes
func hasAnyChanges(ddlDiff *diff.DDLDiff) bool {
	return len(ddlDiff.AddedSchemas) > 0 ||
		len(ddlDiff.DroppedSchemas) > 0 ||
		len(ddlDiff.ModifiedSchemas) > 0 ||
		len(ddlDiff.AddedTables) > 0 ||
		len(ddlDiff.DroppedTables) > 0 ||
		len(ddlDiff.ModifiedTables) > 0 ||
		len(ddlDiff.AddedViews) > 0 ||
		len(ddlDiff.DroppedViews) > 0 ||
		len(ddlDiff.ModifiedViews) > 0 ||
		len(ddlDiff.AddedExtensions) > 0 ||
		len(ddlDiff.DroppedExtensions) > 0 ||
		len(ddlDiff.AddedFunctions) > 0 ||
		len(ddlDiff.DroppedFunctions) > 0 ||
		len(ddlDiff.ModifiedFunctions) > 0 ||
		len(ddlDiff.AddedProcedures) > 0 ||
		len(ddlDiff.DroppedProcedures) > 0 ||
		len(ddlDiff.ModifiedProcedures) > 0 ||
		len(ddlDiff.AddedTypes) > 0 ||
		len(ddlDiff.DroppedTypes) > 0 ||
		len(ddlDiff.ModifiedTypes) > 0
}