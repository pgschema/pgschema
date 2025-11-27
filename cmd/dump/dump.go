package dump

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/pgschema/pgschema/cmd/util"
	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/dump"
	"github.com/pgschema/pgschema/ir"
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

// DumpConfig holds configuration for dump execution
type DumpConfig struct {
	Host      string
	Port      int
	DB        string
	User      string
	Password  string
	Schema    string
	MultiFile bool
	File      string
	QuoteAll  bool
}

var DumpCmd = &cobra.Command{
	Use:          "dump",
	Short:        "Dump database schema for a specific schema",
	Long:         "Dump and output database schema information for a specific schema. Uses the --schema flag to target a particular schema (defaults to 'public').",
	RunE:         runDump,
	SilenceUsage: true,
	PreRunE:      util.PreRunEWithEnvVarsAndConnection(&db, &user, &host, &port),
}

func init() {
	DumpCmd.Flags().StringVar(&host, "host", "localhost", "Database server host (env: PGHOST)")
	DumpCmd.Flags().IntVar(&port, "port", 5432, "Database server port (env: PGPORT)")
	DumpCmd.Flags().StringVar(&db, "db", "", "Database name (required) (env: PGDATABASE)")
	DumpCmd.Flags().StringVar(&user, "user", "", "Database user name (required) (env: PGUSER)")
	DumpCmd.Flags().StringVar(&password, "password", "", "Database password (optional, can also use PGPASSWORD env var)")
	DumpCmd.Flags().StringVar(&schema, "schema", "public", "Schema name to dump (default: public)")
	DumpCmd.Flags().BoolVar(&multiFile, "multi-file", false, "Output schema to multiple files organized by object type")
	DumpCmd.Flags().StringVar(&file, "file", "", "Output file path (required when --multi-file is used)")
}

// ExecuteDump executes the dump operation with the given configuration
func ExecuteDump(config *DumpConfig) (string, error) {
	// Validate flags
	if config.MultiFile && config.File == "" {
		// When --multi-file is used but no --file specified, emit warning and use single-file mode
		fmt.Fprintf(os.Stderr, "Warning: --multi-file flag requires --file to be specified. Fallback to single-file mode.\n")
		config.MultiFile = false
	}

	// Load ignore configuration
	ignoreConfig, err := util.LoadIgnoreFileWithStructure()
	if err != nil {
		return "", fmt.Errorf("failed to load .pgschemaignore: %w", err)
	}

	// Get IR from database using the shared utility
	schemaIR, err := util.GetIRFromDatabase(config.Host, config.Port, config.DB, config.User, config.Password, config.Schema, "pgschema", ignoreConfig)
	if err != nil {
		return "", fmt.Errorf("failed to get database schema: %w", err)
	}

	// Create an empty schema for comparison to generate a dump diff
	emptyIR := ir.NewIR()

	// Generate diff between empty schema and target schema (this represents a complete dump)
	diffs := diff.GenerateMigration(emptyIR, schemaIR, config.Schema, diff.QuoteAll(config.QuoteAll))

	// Create dump formatter
	formatter := dump.NewDumpFormatter(schemaIR.Metadata.DatabaseVersion, config.Schema)

	if config.MultiFile {
		// Multi-file mode - output to files
		err := formatter.FormatMultiFile(diffs, config.File)
		if err != nil {
			return "", fmt.Errorf("failed to create multi-file output: %w", err)
		}
		return "", nil
	} else {
		// Single file mode - return output as string
		output := formatter.FormatSingleFile(diffs)
		return output, nil
	}
}

func runDump(cmd *cobra.Command, args []string) error {
	// Derive final password: use flag if provided, otherwise check environment variable
	finalPassword := password
	if finalPassword == "" {
		if envPassword := os.Getenv("PGPASSWORD"); envPassword != "" {
			finalPassword = envPassword
		}
	}

	// Get quote-all flag from root command
	var quoteAll bool
	if cmd != nil {
		q, err := cmd.Root().PersistentFlags().GetBool("quote-all")
		if err == nil {
			quoteAll = q
		} else {
			slog.Warn("Failed to get quote-all flag", "error", err)
		}
	}

	// Create config from command-line flags
	config := &DumpConfig{
		Host:      host,
		Port:      port,
		DB:        db,
		User:      user,
		Password:  finalPassword,
		Schema:    schema,
		MultiFile: multiFile,
		File:      file,
		QuoteAll:  quoteAll,
	}

	// Execute dump
	output, err := ExecuteDump(config)
	if err != nil {
		return err
	}

	// Print output to stdout (only in single-file mode)
	if output != "" {
		fmt.Print(output)
	}

	return nil
}
