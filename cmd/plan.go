package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/ir"
	"github.com/pgschema/pgschema/internal/plan"
	"github.com/spf13/cobra"
)

var (
	// Source 1 (database connection)
	host1     string
	port1     int
	dbname1   string
	username1 string
	schema1   string

	// Source 1 (schema file)
	file1 string

	// Source 2 (database connection)
	host2     string
	port2     int
	dbname2   string
	username2 string
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
	PlanCmd.Flags().StringVar(&dbname1, "dbname1", "", "Database name for source 1")
	PlanCmd.Flags().StringVar(&username1, "username1", "", "Database user name for source 1")
	PlanCmd.Flags().StringVar(&schema1, "schema1", "public", "Schema name for source 1")

	// Source 1 schema file flag
	PlanCmd.Flags().StringVar(&file1, "file1", "", "Path to first SQL schema file")

	// Source 2 database connection flags
	PlanCmd.Flags().StringVar(&host2, "host2", "localhost", "Database server host for source 2")
	PlanCmd.Flags().IntVar(&port2, "port2", 5432, "Database server port for source 2")
	PlanCmd.Flags().StringVar(&dbname2, "dbname2", "", "Database name for source 2")
	PlanCmd.Flags().StringVar(&username2, "username2", "", "Database user name for source 2")
	PlanCmd.Flags().StringVar(&schema2, "schema2", "public", "Schema name for source 2")

	// Source 2 schema file flag
	PlanCmd.Flags().StringVar(&file2, "file2", "", "Path to second SQL schema file")

	// Output format
	PlanCmd.Flags().StringVar(&format, "format", "text", "Output format: text, json, preview")
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
	case "preview":
		fmt.Print(migrationPlan.Preview())
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
	source1DB := (dbname1 != "" || username1 != "")
	source1File := (file1 != "")

	if source1DB && source1File {
		return fmt.Errorf("source 1: cannot specify both database connection and schema file")
	}
	if !source1DB && !source1File {
		return fmt.Errorf("source 1: must specify either database connection (--dbname1, --username1) or schema file (--file1)")
	}

	// Check source 2
	source2DB := (dbname2 != "" || username2 != "")
	source2File := (file2 != "")

	if source2DB && source2File {
		return fmt.Errorf("source 2: cannot specify both database connection and schema file")
	}
	if !source2DB && !source2File {
		return fmt.Errorf("source 2: must specify either database connection (--dbname2, --username2) or schema file (--file2)")
	}

	// Additional validation for database connections
	if source1DB && (dbname1 == "" || username1 == "") {
		return fmt.Errorf("source 1: both --dbname1 and --username1 are required for database connection")
	}
	if source2DB && (dbname2 == "" || username2 == "") {
		return fmt.Errorf("source 2: both --dbname2 and --username2 are required for database connection")
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
			return getSchemaFromDatabase(host1, port1, dbname1, username1, schema1)
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
			return getSchemaFromDatabase(host2, port2, dbname2, username2, schema2)
		}
	}
}

// getSchemaFromDatabase connects to a database and extracts schema using the IR system
func getSchemaFromDatabase(host string, port int, dbname, username, schemaName string) (string, error) {
	// Build connection string
	dsn := buildDSN(host, port, dbname, username)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return "", fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Build schema using the IR system
	builder := ir.NewBuilder(db)
	schemaIR, err := builder.BuildSchema(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to build schema: %w", err)
	}

	// If a specific schema is requested, filter the results
	if schemaName != "" {
		schemaIR = filterSchemaByName(schemaIR, schemaName)
	}

	// Generate SQL output using the same logic as inspect command
	return generateSQL(schemaIR), nil
}

// filterSchemaByName filters the schema IR to only include the specified schema
func filterSchemaByName(s *ir.Schema, targetSchema string) *ir.Schema {
	filtered := &ir.Schema{
		Metadata:             s.Metadata,
		Extensions:           make(map[string]*ir.Extension),
		Schemas:              make(map[string]*ir.DBSchema),
		PartitionAttachments: []*ir.PartitionAttachment{},
		IndexAttachments:     []*ir.IndexAttachment{},
	}

	// Only include the target schema if it exists
	if dbSchema, exists := s.Schemas[targetSchema]; exists {
		filtered.Schemas[targetSchema] = dbSchema
	}

	// Extensions are global, so include them
	for name, ext := range s.Extensions {
		filtered.Extensions[name] = ext
	}

	return filtered
}
