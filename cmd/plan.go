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
	// Source 1 (database connection)
	host1     string
	port1     int
	db1       string
	user1     string
	password1 string
	schema1   string

	// Source 1 (schema file)
	file1 string

	// Source 2 (database connection)
	host2     string
	port2     int
	db2       string
	user2     string
	password2 string
	schema2   string

	// Source 2 (schema file)
	file2 string

	// Output format
	format string
)

var PlanCmd = &cobra.Command{
	Use:   "plan",
	Short: "Generate migration plan",
	Long:  "Generate and display a migration plan showing changes between two database schemas or schema files",
	RunE:  runPlan,
}

func init() {
	// Source 1 database connection flags
	PlanCmd.Flags().StringVar(&host1, "host1", "localhost", "Database server host for source 1")
	PlanCmd.Flags().IntVar(&port1, "port1", 5432, "Database server port for source 1")
	PlanCmd.Flags().StringVar(&db1, "db1", "", "Database name for source 1")
	PlanCmd.Flags().StringVar(&user1, "user1", "", "Database user name for source 1")
	PlanCmd.Flags().StringVar(&password1, "password1", "", "Database password for source 1 (optional)")
	PlanCmd.Flags().StringVar(&schema1, "schema1", "public", "Schema name for source 1")

	// Source 1 schema file flag
	PlanCmd.Flags().StringVar(&file1, "file1", "", "Path to first SQL schema file")

	// Source 2 database connection flags
	PlanCmd.Flags().StringVar(&host2, "host2", "localhost", "Database server host for source 2")
	PlanCmd.Flags().IntVar(&port2, "port2", 5432, "Database server port for source 2")
	PlanCmd.Flags().StringVar(&db2, "db2", "", "Database name for source 2")
	PlanCmd.Flags().StringVar(&user2, "user2", "", "Database user name for source 2")
	PlanCmd.Flags().StringVar(&password2, "password2", "", "Database password for source 2 (optional)")
	PlanCmd.Flags().StringVar(&schema2, "schema2", "public", "Schema name for source 2")

	// Source 2 schema file flag
	PlanCmd.Flags().StringVar(&file2, "file2", "", "Path to second SQL schema file")

	// Output format
	PlanCmd.Flags().StringVar(&format, "format", "text", "Output format: text, json")
}

func runPlan(cmd *cobra.Command, args []string) error {
	// Validate that each source has exactly one input method
	if err := validateSourceInputs(); err != nil {
		return err
	}

	// Get schema data from source 1
	schema1Data, err := getSchemaData(1)
	if err != nil {
		return fmt.Errorf("failed to get schema data from source 1: %w", err)
	}

	// Get schema data from source 2
	schema2Data, err := getSchemaData(2)
	if err != nil {
		return fmt.Errorf("failed to get schema data from source 2: %w", err)
	}

	// Generate diff
	ddlDiff, err := diff.Diff(schema1Data, schema2Data)
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
	case "text":
		fallthrough
	default:
		fmt.Print(migrationPlan.Summary())
	}

	return nil
}

// validateSourceInputs ensures that each source has exactly one input method specified
func validateSourceInputs() error {
	// Check source 1
	source1DB := (db1 != "" || user1 != "")
	source1File := (file1 != "")

	if source1DB && source1File {
		return fmt.Errorf("source 1: cannot specify both database connection and schema file")
	}
	if !source1DB && !source1File {
		return fmt.Errorf("source 1: must specify either database connection (--db1, --user1) or schema file (--file1)")
	}

	// Check source 2
	source2DB := (db2 != "" || user2 != "")
	source2File := (file2 != "")

	if source2DB && source2File {
		return fmt.Errorf("source 2: cannot specify both database connection and schema file")
	}
	if !source2DB && !source2File {
		return fmt.Errorf("source 2: must specify either database connection (--db2, --user2) or schema file (--file2)")
	}

	// Additional validation for database connections
	if source1DB && (db1 == "" || user1 == "") {
		return fmt.Errorf("source 1: both --db1 and --user1 are required for database connection")
	}
	if source2DB && (db2 == "" || user2 == "") {
		return fmt.Errorf("source 2: both --db2 and --user2 are required for database connection")
	}

	return nil
}

// getSchemaData retrieves schema data from either database or file based on source number
func getSchemaData(sourceNum int) (string, error) {
	if sourceNum == 1 {
		if file1 != "" {
			// Read from file
			data, err := os.ReadFile(file1)
			if err != nil {
				return "", fmt.Errorf("failed to read schema file: %w", err)
			}
			return string(data), nil
		} else {
			// Connect to database and extract schema
			return getSchemaFromDatabase(host1, port1, db1, user1, password1, schema1)
		}
	} else {
		if file2 != "" {
			// Read from file
			data, err := os.ReadFile(file2)
			if err != nil {
				return "", fmt.Errorf("failed to read schema file: %w", err)
			}
			return string(data), nil
		} else {
			// Connect to database and extract schema
			return getSchemaFromDatabase(host2, port2, db2, user2, password2, schema2)
		}
	}
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
