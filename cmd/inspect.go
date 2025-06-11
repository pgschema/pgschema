package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

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

	// Print pg_dump style header
	version := "0.0.1" // default
	if versionBytes, err := os.ReadFile("VERSION"); err == nil {
		version = strings.TrimSpace(string(versionBytes))
	}
	
	fmt.Println("--")
	fmt.Println("-- PostgreSQL database dump")
	fmt.Println("--")
	fmt.Println("")
	fmt.Println("-- Dumped from database version 15.0")
	fmt.Printf("-- Dumped by pgschema version %s\n", version)
	fmt.Println("")

	// Get schemas and create them
	schemas, err := q.GetSchemas(ctx)
	if err != nil {
		return fmt.Errorf("failed to get schemas: %w", err)
	}

	for _, schema := range schemas {
		schemaName := fmt.Sprintf("%s", schema)
		if schemaName != "public" {
			fmt.Printf("CREATE SCHEMA %s;\n\n", schemaName)
		}
	}

	// Get sequences and create them
	sequences, err := q.GetSequences(ctx)
	if err != nil {
		return fmt.Errorf("failed to get sequences: %w", err)
	}

	for _, seq := range sequences {
		schemaName := fmt.Sprintf("%s", seq.SequenceSchema)
		seqName := fmt.Sprintf("%s", seq.SequenceName)
		dataType := fmt.Sprintf("%s", seq.DataType)
		startValue := fmt.Sprintf("%s", seq.StartValue)
		minValue := fmt.Sprintf("%s", seq.MinimumValue)
		maxValue := fmt.Sprintf("%s", seq.MaximumValue)
		increment := fmt.Sprintf("%s", seq.Increment)
		
		fmt.Printf("CREATE SEQUENCE %s.%s\n", schemaName, seqName)
		fmt.Printf("    AS %s\n", dataType)
		fmt.Printf("    START WITH %s\n", startValue)
		fmt.Printf("    INCREMENT BY %s\n", increment)
		fmt.Printf("    MINVALUE %s\n", minValue)
		fmt.Printf("    MAXVALUE %s\n", maxValue)
		fmt.Printf("    CACHE 1;\n\n")
	}

	// Get tables and columns to build CREATE TABLE statements
	tables, err := q.GetTables(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tables: %w", err)
	}

	columns, err := q.GetColumns(ctx)
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	constraints, err := q.GetConstraints(ctx)
	if err != nil {
		return fmt.Errorf("failed to get constraints: %w", err)
	}

	// Group columns by table
	tableColumns := make(map[string][]any)
	for _, col := range columns {
		tableKey := fmt.Sprintf("%s.%s", col.TableSchema, col.TableName)
		tableColumns[tableKey] = append(tableColumns[tableKey], col)
	}

	// Group constraints by table
	tableConstraints := make(map[string][]any)
	for _, constraint := range constraints {
		tableKey := fmt.Sprintf("%s.%s", constraint.TableSchema, constraint.TableName)
		tableConstraints[tableKey] = append(tableConstraints[tableKey], constraint)
	}

	// Generate CREATE TABLE statements
	for _, table := range tables {
		schemaName := fmt.Sprintf("%s", table.TableSchema)
		tableName := fmt.Sprintf("%s", table.TableName)
		tableType := fmt.Sprintf("%s", table.TableType)
		
		if tableType == "VIEW" {
			continue // Skip views for now
		}

		tableKey := fmt.Sprintf("%s.%s", schemaName, tableName)
		
		fmt.Printf("CREATE TABLE %s.%s (\n", schemaName, tableName)
		
		// Add columns
		tableCols := tableColumns[tableKey]
		for i, col := range tableCols {
			colRow := col.(queries.GetColumnsRow)
			if i > 0 {
				fmt.Printf(",\n")
			}
			
			colName := fmt.Sprintf("%s", colRow.ColumnName)
			dataType := fmt.Sprintf("%s", colRow.DataType)
			isNullable := fmt.Sprintf("%s", colRow.IsNullable)
			
			// Build column definition
			fmt.Printf("    %s %s", colName, dataType)
			
			// Add length/precision for specific types
			if colRow.CharacterMaximumLength != nil {
				maxLen := fmt.Sprintf("%s", colRow.CharacterMaximumLength)
				if maxLen != "<nil>" && maxLen != "" {
					fmt.Printf("(%s)", maxLen)
				}
			} else if colRow.NumericPrecision != nil && colRow.NumericScale != nil {
				precision := fmt.Sprintf("%s", colRow.NumericPrecision)
				scale := fmt.Sprintf("%s", colRow.NumericScale)
				if precision != "<nil>" && scale != "<nil>" {
					fmt.Printf("(%s,%s)", precision, scale)
				}
			}
			
			// Add NOT NULL constraint
			if isNullable == "NO" {
				fmt.Printf(" NOT NULL")
			}
			
			// Add default value
			if colRow.ColumnDefault != nil {
				defaultVal := fmt.Sprintf("%s", colRow.ColumnDefault)
				if defaultVal != "<nil>" && defaultVal != "" {
					fmt.Printf(" DEFAULT %s", defaultVal)
				}
			}
		}
		
		fmt.Printf("\n);\n\n")
		
		// Add constraints for this table
		tableConsts := tableConstraints[tableKey]
		for _, constraint := range tableConsts {
			constRow := constraint.(queries.GetConstraintsRow)
			constraintType := fmt.Sprintf("%s", constRow.ConstraintType)
			constraintName := fmt.Sprintf("%s", constRow.ConstraintName)
			
			switch constraintType {
			case "PRIMARY KEY":
				columnName := fmt.Sprintf("%s", constRow.ColumnName)
				fmt.Printf("ALTER TABLE ONLY %s.%s\n", schemaName, tableName)
				fmt.Printf("    ADD CONSTRAINT %s PRIMARY KEY (%s);\n\n", constraintName, columnName)
			case "FOREIGN KEY":
				columnName := fmt.Sprintf("%s", constRow.ColumnName)
				foreignTable := fmt.Sprintf("%s", constRow.ForeignTableName)
				foreignColumn := fmt.Sprintf("%s", constRow.ForeignColumnName)
				foreignSchema := fmt.Sprintf("%s", constRow.ForeignTableSchema)
				if foreignTable != "<nil>" && foreignColumn != "<nil>" {
					fmt.Printf("ALTER TABLE ONLY %s.%s\n", schemaName, tableName)
					fmt.Printf("    ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s.%s(%s);\n\n", 
						constraintName, columnName, foreignSchema, foreignTable, foreignColumn)
				}
			case "CHECK":
				checkClause := fmt.Sprintf("%s", constRow.CheckClause)
				if checkClause != "<nil>" && checkClause != "" {
					fmt.Printf("ALTER TABLE ONLY %s.%s\n", schemaName, tableName)
					fmt.Printf("    ADD CONSTRAINT %s CHECK (%s);\n\n", constraintName, checkClause)
				}
			}
		}
	}

	return nil
}