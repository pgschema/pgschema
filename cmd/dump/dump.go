package dump

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pgschema/pgschema/cmd/util"
	"github.com/pgschema/pgschema/internal/diff"
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
	config := &util.ConnectionConfig{
		Host:     host,
		Port:     port,
		Database: db,
		User:     user,
		Password: finalPassword,
		SSLMode:  "prefer",
	}

	dbConn, err := util.Connect(config)
	if err != nil {
		return err
	}
	defer dbConn.Close()

	ctx := context.Background()

	// Build IR using the IR system
	inspector := ir.NewInspector(dbConn)
	schemaIR, err := inspector.BuildIR(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to build IR: %w", err)
	}

	// Generate header with database metadata
	header := generateDumpHeader(schemaIR)

	// Generate dump SQL using the unified diff approach
	// This treats dump as a diff from empty schema to current schema
	output := diff.GenerateDumpSQL(schemaIR, true, schema)

	// Print header followed by the dump SQL
	fmt.Print(header)
	fmt.Print(output)
	return nil
}

// generateDumpHeader generates the header for database dumps with metadata
func generateDumpHeader(schemaIR *ir.IR) string {
	var header strings.Builder

	header.WriteString("--\n")
	header.WriteString("-- PostgreSQL database dump\n")
	header.WriteString("--\n")
	header.WriteString("\n")

	if schemaIR.Metadata.DatabaseVersion != "" {
		header.WriteString(fmt.Sprintf("-- Dumped from database version %s\n", schemaIR.Metadata.DatabaseVersion))
	}
	if schemaIR.Metadata.DumpVersion != "" {
		header.WriteString(fmt.Sprintf("-- Dumped by %s\n", schemaIR.Metadata.DumpVersion))
	}
	return header.String()
}
