package cmd

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pgschema/pgschema/internal/queries"
	"github.com/spf13/cobra"
)

var dsn string

var InspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect database schema",
	Long:  "Inspect and output database schema information including schemas and tables",
	RunE:  runInspect,
}

func init() {
	InspectCmd.Flags().StringVar(&dsn, "dsn", "", "Database connection string (required)")
	InspectCmd.MarkFlagRequired("dsn")
}

func runInspect(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	q := queries.New(db)

	// Get schemas
	fmt.Println("## Schemas")
	schemas, err := q.GetSchemas(ctx)
	if err != nil {
		return fmt.Errorf("failed to get schemas: %w", err)
	}

	for _, schema := range schemas {
		fmt.Printf("- %s\n", schema)
	}

	// Get tables
	fmt.Println("\n## Tables")
	tables, err := q.GetTables(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tables: %w", err)
	}

	currentSchema := ""
	for _, table := range tables {
		schemaName := fmt.Sprintf("%s", table.TableSchema)
		if schemaName != currentSchema {
			currentSchema = schemaName
			fmt.Printf("\n### Schema: %s\n", currentSchema)
		}
		fmt.Printf("- %s (%s)\n", table.TableName, table.TableType)
	}

	return nil
}