package apply

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	planCmd "github.com/pgschema/pgschema/cmd/plan"
	"github.com/pgschema/pgschema/cmd/util"
	"github.com/spf13/cobra"
)

var (
	applyHost            string
	applyPort            int
	applyDB              string
	applyUser            string
	applyPassword        string
	applySchema          string
	applyFile            string
	applyAutoApprove     bool
	applyNoColor         bool
	applyDryRun          bool
	applyLockTimeout     string
	applyApplicationName string
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

	// Apply behavior flags
	ApplyCmd.Flags().BoolVar(&applyAutoApprove, "auto-approve", false, "Apply changes without prompting for approval")
	ApplyCmd.Flags().BoolVar(&applyNoColor, "no-color", false, "Disable colored output")
	ApplyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Show plan without applying changes")
	ApplyCmd.Flags().StringVar(&applyLockTimeout, "lock-timeout", "", "Maximum time to wait for database locks (e.g., 30s, 5m, 1h)")
	ApplyCmd.Flags().StringVar(&applyApplicationName, "application-name", "pgschema", "Application name for database connection (visible in pg_stat_activity)")

	// Mark required flags
	ApplyCmd.MarkFlagRequired("db")
	ApplyCmd.MarkFlagRequired("user")
	ApplyCmd.MarkFlagRequired("file")
}

func runApply(cmd *cobra.Command, args []string) error {
	// Create plan configuration
	config := &planCmd.PlanConfig{
		Host:            applyHost,
		Port:            applyPort,
		DB:              applyDB,
		User:            applyUser,
		Password:        applyPassword,
		Schema:          applySchema,
		File:            applyFile,
		ApplicationName: applyApplicationName,
	}

	// Generate plan using shared logic
	migrationPlan, err := planCmd.GeneratePlan(config)
	if err != nil {
		return err
	}

	// Check if there are any changes to apply by examining the diff
	hasChanges := planCmd.HasAnyChanges(migrationPlan.Diff)
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
	connConfig := &util.ConnectionConfig{
		Host:            applyHost,
		Port:            applyPort,
		Database:        applyDB,
		User:            applyUser,
		Password:        config.Password,
		SSLMode:         "prefer",
		ApplicationName: applyApplicationName,
	}

	conn, err := util.Connect(connConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close()

	ctx := context.Background()

	// Set lock timeout before executing changes
	if applyLockTimeout != "" {
		_, err = conn.ExecContext(ctx, fmt.Sprintf("SET lock_timeout = '%s'", applyLockTimeout))
		if err != nil {
			return fmt.Errorf("failed to set lock timeout: %w", err)
		}
	}

	// Generate SQL statements from the plan
	sqlStatements := migrationPlan.ToSQL()

	// Skip execution if no changes
	if strings.TrimSpace(sqlStatements) == "-- No changes detected" || strings.TrimSpace(sqlStatements) == "-- No DDL statements generated" {
		fmt.Println("No SQL statements to execute.")
		return nil
	}

	// Execute the SQL statements based on transaction requirements
	if migrationPlan.EnableTransaction {
		// Default behavior - execute in transaction
		_, err = conn.ExecContext(ctx, sqlStatements)
		if err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}
	} else {
		// Execute each statement individually to avoid implicit transaction
		// Note: This means no rollback protection if something fails

		// Use pg_query_go to properly split SQL statements
		statements, err := pg_query.SplitWithParser(sqlStatements, true) // trimSpace = true
		if err != nil {
			return fmt.Errorf("failed to split SQL statements: %w", err)
		}

		for _, stmt := range statements {
			if strings.TrimSpace(stmt) == "" {
				continue // Skip empty statements
			}
			fmt.Printf("Executing (non-transactional): %s\n", strings.TrimSpace(stmt))
			_, err = conn.ExecContext(ctx, stmt)
			if err != nil {
				return fmt.Errorf("failed to apply statement '%s': %w", strings.TrimSpace(stmt), err)
			}
		}
	}

	fmt.Println("Changes applied successfully!")
	return nil
}
