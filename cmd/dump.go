package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pgschema/pgschema/internal/ir"
	"github.com/spf13/cobra"
)

var (
	host     string
	port     int
	db       string
	user     string
	password string
	schema   string
)

var DumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump database schema",
	Long:  "Dump and output database schema information including schemas and tables",
	RunE:  runDump,
}

func init() {
	DumpCmd.Flags().StringVar(&host, "host", "localhost", "Database server host")
	DumpCmd.Flags().IntVar(&port, "port", 5432, "Database server port")
	DumpCmd.Flags().StringVar(&db, "db", "", "Database name (required)")
	DumpCmd.Flags().StringVar(&user, "user", "", "Database user name (required)")
	DumpCmd.Flags().StringVar(&password, "password", "", "Database password (optional, can also use PGPASSWORD env var)")
	DumpCmd.Flags().StringVar(&schema, "schema", "public", "Schema name to dump (default: public)")
	DumpCmd.MarkFlagRequired("db")
	DumpCmd.MarkFlagRequired("user")
}

func runDump(cmd *cobra.Command, args []string) error {
	// Derive final password: use flag if provided, otherwise check environment variable
	finalPassword := password
	if finalPassword == "" {
		if envPassword := os.Getenv("PGPASSWORD"); envPassword != "" {
			finalPassword = envPassword
		}
	}

	// Build connection string from individual parameters
	dsn := buildDSN(host, port, db, user, finalPassword)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Build schema using the IR system
	builder := ir.NewBuilder(db)
	schemaIR, err := builder.BuildSchema(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to build schema: %w", err)
	}

	// Generate SQL output using unified SQL generator service
	sqlGenerator := ir.NewSQLGeneratorService(true) // Include comments for dump command
	output := sqlGenerator.GenerateSchemaSQL(schemaIR, schema)

	fmt.Print(output)
	return nil
}



// buildDSN constructs a PostgreSQL connection string from individual parameters
func buildDSN(host string, port int, db, user, password string) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("host=%s", host))
	parts = append(parts, fmt.Sprintf("port=%d", port))
	parts = append(parts, fmt.Sprintf("dbname=%s", db))
	parts = append(parts, fmt.Sprintf("user=%s", user))

	// Use password if provided
	if password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", password))
	}

	// Add default SSL mode
	parts = append(parts, "sslmode=prefer")

	return strings.Join(parts, " ")
}
