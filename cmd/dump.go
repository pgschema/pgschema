package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/pgschema/pgschema/internal/ir"
	"github.com/pgschema/pgschema/internal/utils"
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
	Short: "Dump database schema for a specific schema",
	Long:  "Dump and output database schema information for a specific schema. Uses the --schema flag to target a particular schema (defaults to 'public').",
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

	// Build database connection
	config := &utils.ConnectionConfig{
		Host:     host,
		Port:     port,
		Database: db,
		User:     user,
		Password: finalPassword,
		SSLMode:  "prefer",
	}

	dbConn, err := utils.Connect(config)
	if err != nil {
		return err
	}
	defer dbConn.Close()

	ctx := context.Background()

	// Build IR using the IR system
	builder := ir.NewBuilder(dbConn)
	schemaIR, err := builder.BuildIR(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to build IR: %w", err)
	}

	sqlGenerator := ir.NewSQLGeneratorService(true) // Include comments for dump command
	// Generates SQL as if it were a diff from empty schema
	emptyIR := ir.NewIR()
	output := sqlGenerator.GenerateDiff(emptyIR, schemaIR, schema)

	fmt.Print(output)
	return nil
}
