package dump

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pgschema/pgschema/cmd/util"
	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/ir"
	"github.com/pgschema/pgschema/internal/version"
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

// generateDumpHeader generates the header for database dumps with metadata
func generateDumpHeader(schemaIR *ir.IR) string {
	var header strings.Builder

	header.WriteString("--\n")
	header.WriteString("-- pgschema database dump\n")
	header.WriteString("--\n")
	header.WriteString("\n")

	header.WriteString(fmt.Sprintf("-- Dumped from database version %s\n", schemaIR.Metadata.DatabaseVersion))
	header.WriteString(fmt.Sprintf("-- Dumped by pgschema version %s\n", version.App()))
	header.WriteString("\n")
	header.WriteString("\n")
	return header.String()
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

	if multiFile {
		// Multi-file mode - output to files
		multiWriter, err := diff.NewMultiFileWriter(file, true)
		if err != nil {
			return fmt.Errorf("failed to create multi-file writer: %w", err)
		}

		// Generate header with database metadata (same as single-file mode)
		header := generateDumpHeader(schemaIR)
		multiWriter.WriteHeader(header)

		// Generate dump SQL using multi-file writer
		result := diff.GenerateDumpSQL(schemaIR, schema, multiWriter, nil)

		// Print confirmation message (if any)
		if result != "" {
			fmt.Print(result)
		}
	} else {
		// Single file mode - output to stdout
		// Create SQLCollector to collect all SQL statements
		collector := diff.NewSQLCollector()

		// Generate dump SQL using collector (use dummy writer for compatibility)
		dummyWriter := diff.NewSingleFileWriter(false)
		diff.GenerateDumpSQL(schemaIR, schema, dummyWriter, collector)

		// Generate and print header
		header := generateDumpHeader(schemaIR)
		fmt.Print(header)

		// Print all SQL statements from collector with proper separators
		steps := collector.GetSteps()
		for i, step := range steps {
			// Add DDL separator with comment header
			fmt.Print("--\n")
			
			// Determine schema name for comment (use "-" for target schema)
			commentSchemaName := step.ObjectPath
			if strings.Contains(step.ObjectPath, ".") {
				parts := strings.Split(step.ObjectPath, ".")
				if len(parts) >= 2 && parts[0] == schema {
					commentSchemaName = "-"
				} else {
					commentSchemaName = parts[0]
				}
			}
			
			// Print object comment header
			objectName := step.ObjectPath
			if strings.Contains(step.ObjectPath, ".") {
				parts := strings.Split(step.ObjectPath, ".")
				if len(parts) >= 2 {
					objectName = parts[1]
				}
			}
			
			fmt.Printf("-- Name: %s; Type: %s; Schema: %s; Owner: -\n", objectName, strings.ToUpper(step.ObjectType), commentSchemaName)
			fmt.Print("--\n")
			fmt.Print("\n")
			
			// Print the SQL statement
			fmt.Print(step.SQL)
			
			// Add newline after SQL, and extra newline only if not last item
			if i < len(steps)-1 {
				fmt.Print("\n\n")
			}
		}
	}

	return nil
}

