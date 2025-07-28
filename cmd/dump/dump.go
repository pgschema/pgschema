package dump

import (
	"context"
	"fmt"
	"os"

	"github.com/pgschema/pgschema/cmd/util"
	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/ir"
	"github.com/spf13/cobra"
)

var (
	host      string
	port      int
	db        string
	user      string
	password  string
	schema    string
	multiFile bool
	file      string
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
	DumpCmd.Flags().BoolVar(&multiFile, "multi-file", false, "Output schema to multiple files organized by object type")
	DumpCmd.Flags().StringVar(&file, "file", "", "Output file path (required when --multi-file is used)")
	DumpCmd.MarkFlagRequired("db")
	DumpCmd.MarkFlagRequired("user")
}

func runDump(cmd *cobra.Command, args []string) error {
	// Validate flags
	if multiFile && file == "" {
		// When --multi-file is used but no --file specified, emit warning and use single-file mode
		fmt.Fprintf(os.Stderr, "Warning: --multi-file flag requires --file to be specified. Fallback to single-file mode.\n")
		multiFile = false
	}

	// Derive final password: use flag if provided, otherwise check environment variable
	finalPassword := password
	if finalPassword == "" {
		if envPassword := os.Getenv("PGPASSWORD"); envPassword != "" {
			finalPassword = envPassword
		}
	}

	// Build database connection
	config := &util.ConnectionConfig{
		Host:            host,
		Port:            port,
		Database:        db,
		User:            user,
		Password:        finalPassword,
		SSLMode:         "prefer",
		ApplicationName: "pgschema",
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

	// Choose writer based on multi-file flag
	var writer diff.Writer
	if multiFile {
		// Multi-file mode - output to files
		multiWriter, err := diff.NewMultiFileWriter(file, true)
		if err != nil {
			return fmt.Errorf("failed to create multi-file writer: %w", err)
		}

		// Generate header with database metadata (same as single-file mode)
		header := diff.GenerateDumpHeader(schemaIR)
		multiWriter.WriteHeader(header)

		// Generate dump SQL using multi-file writer
		result := diff.GenerateDumpSQL(schemaIR, schema, multiWriter)

		// Print confirmation message (if any)
		if result != "" {
			fmt.Print(result)
		}
	} else {
		// Single file mode - output to stdout
		writer = diff.NewSingleFileWriter(true)

		// Generate header with database metadata
		header := diff.GenerateDumpHeader(schemaIR)

		// Generate dump SQL using the unified diff approach
		output := diff.GenerateDumpSQL(schemaIR, schema, writer)

		// Print header followed by the dump SQL
		fmt.Print(header)
		fmt.Print(output)
	}

	return nil
}

