package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/stripe/pg-schema-diff/pkg/diff"
	"github.com/stripe/pg-schema-diff/pkg/tempdb"
	_ "github.com/lib/pq"
)

var (
	sourceDir   string
	sourceDSN   string
	targetDir   string
	targetDSN   string
	tempDbDSN   string
)

var DiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare two PostgreSQL schemas",
	Long:  "Compare schemas from directories or databases and show the differences",
	RunE:  runDiff,
}

func init() {
	DiffCmd.Flags().StringVar(&sourceDir, "source-dir", "", "Source schema directory")
	DiffCmd.Flags().StringVar(&sourceDSN, "source-dsn", "", "Source database connection string")
	DiffCmd.Flags().StringVar(&targetDir, "target-dir", "", "Target schema directory")
	DiffCmd.Flags().StringVar(&targetDSN, "target-dsn", "", "Target database connection string")
	DiffCmd.Flags().StringVar(&tempDbDSN, "temp-db-dsn", "", "Temporary database connection string (required for directory-based schemas)")
}

func runDiff(cmd *cobra.Command, args []string) error {
	if (sourceDir == "" && sourceDSN == "") || (targetDir == "" && targetDSN == "") {
		return fmt.Errorf("must specify both source and target (either --source-dir/--source-dsn and --target-dir/--target-dsn)")
	}
	
	if (sourceDir != "" && sourceDSN != "") || (targetDir != "" && targetDSN != "") {
		return fmt.Errorf("cannot specify both directory and DSN for the same schema")
	}

	// Check if temp db is required for directory schemas
	if (sourceDir != "" || targetDir != "") && tempDbDSN == "" {
		return fmt.Errorf("--temp-db-dsn is required when using directory-based schemas")
	}

	ctx := context.Background()
	var sourceSchema, targetSchema diff.SchemaSource
	var err error
	var planOpts []diff.PlanOpt

	// Set up temp db factory if needed
	if tempDbDSN != "" {
		tempDb, err := sql.Open("postgres", tempDbDSN)
		if err != nil {
			return fmt.Errorf("failed to connect to temp database: %w", err)
		}
		defer tempDb.Close()

		if err := tempDb.Ping(); err != nil {
			return fmt.Errorf("failed to ping temp database: %w", err)
		}

		factory, err := tempdb.NewOnInstanceFactory(ctx, func(ctx context.Context, dbName string) (*sql.DB, error) {
			// Parse the original DSN and replace the database name
			dsnWithNewDB, err := replaceDatabaseInDSN(tempDbDSN, dbName)
			if err != nil {
				return nil, fmt.Errorf("failed to modify DSN for database %s: %w", dbName, err)
			}
			return sql.Open("postgres", dsnWithNewDB)
		})
		if err != nil {
			return fmt.Errorf("failed to create temp db factory: %w", err)
		}
		defer factory.Close()

		planOpts = append(planOpts, diff.WithTempDbFactory(factory))
	}

	if sourceDir != "" {
		sourceSchema, err = loadSchemaFromDirectory(sourceDir)
		if err != nil {
			return fmt.Errorf("failed to load source schema from directory: %w", err)
		}
	} else {
		sourceSchema, err = loadSchemaFromDatabase(sourceDSN)
		if err != nil {
			return fmt.Errorf("failed to load source schema from database: %w", err)
		}
	}

	if targetDir != "" {
		targetSchema, err = loadSchemaFromDirectory(targetDir)
		if err != nil {
			return fmt.Errorf("failed to load target schema from directory: %w", err)
		}
	} else {
		targetSchema, err = loadSchemaFromDatabase(targetDSN)
		if err != nil {
			return fmt.Errorf("failed to load target schema from database: %w", err)
		}
	}

	plan, err := diff.Generate(ctx, sourceSchema, targetSchema, planOpts...)
	if err != nil {
		return fmt.Errorf("failed to generate diff: %w", err)
	}

	if len(plan.Statements) == 0 {
		fmt.Println("No differences found between schemas")
		return nil
	}

	fmt.Printf("Found %d differences:\n\n", len(plan.Statements))
	for _, stmt := range plan.Statements {
		fmt.Println(stmt.DDL)
	}

	return nil
}

func loadSchemaFromDirectory(dir string) (diff.SchemaSource, error) {
	return diff.DirSchemaSource([]string{dir})
}

func loadSchemaFromDatabase(dsn string) (diff.SchemaSource, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return diff.DBSchemaSource(db), nil
}

func replaceDatabaseInDSN(dsn, newDatabase string) (string, error) {
	// Parse the DSN URL
	parsedURL, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("failed to parse DSN: %w", err)
	}

	// Replace the database name (path)
	parsedURL.Path = "/" + newDatabase

	return parsedURL.String(), nil
}