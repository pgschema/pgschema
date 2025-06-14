package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pgschema/pgschema/internal/schema"
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
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	ctx := context.Background()
	
	// Build schema using the IR system
	builder := schema.NewBuilder(db)
	schemaIR, err := builder.BuildSchema(ctx)
	if err != nil {
		return fmt.Errorf("failed to build schema: %w", err)
	}

	// Generate SQL output using visitor pattern
	output := generateSQL(schemaIR)
	
	fmt.Print(output)
	return nil
}
// generateSQL generates complete SQL DDL from the schema IR using visitor pattern
func generateSQL(s *schema.Schema) string {
	w := schema.NewSQLWriter()
	
	// Header
	writeHeader(w, s)
	
	// Schemas (skip public schema)
	writeSchemas(w, s)
	
	// Functions
	writeFunctions(w, s)
	
	// Sequences
	writeStandaloneSequences(w, s)
	
	// Tables and Views (dependency sorted)
	writeTablesAndViews(w, s)
	
	// Indexes
	writeIndexes(w, s)
	
	// Triggers
	writeTriggers(w, s)
	
	// Foreign Key constraints
	writeForeignKeyConstraints(w, s)
	
	// RLS
	writeRLS(w, s)
	
	// Footer
	writeFooter(w, s)
	
	return w.String()
}

func writeHeader(w *schema.SQLWriter, s *schema.Schema) {
	w.WriteString("--\n")
	w.WriteString("-- PostgreSQL database dump\n")
	w.WriteString("--\n")
	w.WriteString("\n")
	w.WriteString(fmt.Sprintf("-- Dumped from database version %s\n", s.Metadata.DatabaseVersion))
	w.WriteString(fmt.Sprintf("-- Dumped by %s\n", s.Metadata.DumpVersion))
	w.WriteString("\n")
	w.WriteString("\n")
}

func writeSchemas(w *schema.SQLWriter, s *schema.Schema) {
	schemaNames := s.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]
		w.WriteString(dbSchema.GenerateSQL())
	}
}

func writeFunctions(w *schema.SQLWriter, s *schema.Schema) {
	schemaNames := s.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]
		
		// Sort function names for deterministic output
		var functionNames []string
		for name := range dbSchema.Functions {
			functionNames = append(functionNames, name)
		}
		sort.Strings(functionNames)
		
		for _, functionName := range functionNames {
			function := dbSchema.Functions[functionName]
			w.WriteString(function.GenerateSQL())
		}
	}
}

func writeStandaloneSequences(w *schema.SQLWriter, s *schema.Schema) {
	schemaNames := s.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]
		
		// Sort sequence names for deterministic output  
		var sequenceNames []string
		for name, sequence := range dbSchema.Sequences {
			// Only include sequences that are NOT owned by tables
			if sequence.OwnedByTable == "" {
				sequenceNames = append(sequenceNames, name)
			}
		}
		sort.Strings(sequenceNames)
		
		for _, sequenceName := range sequenceNames {
			sequence := dbSchema.Sequences[sequenceName]
			w.WriteString(sequence.GenerateSQL())
		}
	}
}

func writeTablesAndViews(w *schema.SQLWriter, s *schema.Schema) {
	// Get all objects and sort by dependencies
	objects := getDependencySortedObjects(s)
	
	for _, obj := range objects {
		switch obj.Type {
		case "table":
			dbSchema := s.Schemas[obj.Schema]
			table := dbSchema.Tables[obj.Name]
			w.WriteString(table.GenerateSQL())
			
			// Write sequences owned by this table
			writeSequencesForTable(w, s, obj.Schema, obj.Name)
			
			// Write column defaults
			w.WriteString(table.GenerateColumnDefaultsSQL())
			
			// Write table constraints
			w.WriteString(table.GenerateConstraintsSQL())
			
		case "view":
			dbSchema := s.Schemas[obj.Schema]
			view := dbSchema.Views[obj.Name]
			w.WriteString(view.GenerateSQL())
		}
	}
}

func writeSequencesForTable(w *schema.SQLWriter, s *schema.Schema, schemaName, tableName string) {
	dbSchema := s.Schemas[schemaName]
	
	var sequenceNames []string
	for name, sequence := range dbSchema.Sequences {
		if sequence.OwnedByTable == tableName {
			sequenceNames = append(sequenceNames, name)
		}
	}
	sort.Strings(sequenceNames)
	
	for _, sequenceName := range sequenceNames {
		sequence := dbSchema.Sequences[sequenceName]
		w.WriteString(sequence.GenerateSQL())
	}
}

func writeIndexes(w *schema.SQLWriter, s *schema.Schema) {
	schemaNames := s.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]
		
		var indexNames []string
		for name := range dbSchema.Indexes {
			indexNames = append(indexNames, name)
		}
		sort.Strings(indexNames)
		
		for _, indexName := range indexNames {
			index := dbSchema.Indexes[indexName]
			w.WriteString(index.GenerateSQL())
		}
	}
}

func writeTriggers(w *schema.SQLWriter, s *schema.Schema) {
	schemaNames := s.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]
		
		var triggerNames []string
		for name := range dbSchema.Triggers {
			triggerNames = append(triggerNames, name)
		}
		sort.Strings(triggerNames)
		
		for _, triggerName := range triggerNames {
			trigger := dbSchema.Triggers[triggerName]
			w.WriteString(trigger.GenerateSQL())
		}
	}
}

func writeForeignKeyConstraints(w *schema.SQLWriter, s *schema.Schema) {
	schemaNames := s.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]
		
		// Collect all foreign key constraints
		var foreignKeyConstraints []*schema.Constraint
		for _, table := range dbSchema.Tables {
			for _, constraint := range table.Constraints {
				if constraint.Type == schema.ConstraintTypeForeignKey {
					foreignKeyConstraints = append(foreignKeyConstraints, constraint)
				}
			}
		}
		
		// Sort by table name, then constraint name
		sort.Slice(foreignKeyConstraints, func(i, j int) bool {
			if foreignKeyConstraints[i].Table != foreignKeyConstraints[j].Table {
				return foreignKeyConstraints[i].Table < foreignKeyConstraints[j].Table
			}
			return foreignKeyConstraints[i].Name < foreignKeyConstraints[j].Name
		})
		
		for _, constraint := range foreignKeyConstraints {
			w.WriteString(constraint.GenerateSQL())
		}
	}
}

func writeRLS(w *schema.SQLWriter, s *schema.Schema) {
	// RLS enabled tables
	schemaNames := s.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]
		
		var rlsTables []string
		for tableName, table := range dbSchema.Tables {
			if table.RLSEnabled {
				rlsTables = append(rlsTables, tableName)
			}
		}
		sort.Strings(rlsTables)
		
		for _, tableName := range rlsTables {
			table := dbSchema.Tables[tableName]
			w.WriteString(table.GenerateRLSSQL())
		}
	}
	
	// RLS policies
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]
		
		var policyNames []string
		for name := range dbSchema.Policies {
			policyNames = append(policyNames, name)
		}
		sort.Strings(policyNames)
		
		for _, policyName := range policyNames {
			policy := dbSchema.Policies[policyName]
			w.WriteString(policy.GenerateSQL())
		}
	}
}

func writeFooter(w *schema.SQLWriter, s *schema.Schema) {
	w.WriteString("--\n")
	w.WriteString("-- PostgreSQL database dump complete\n")
	w.WriteString("--\n")
	w.WriteString("\n")
}

// Helper for dependency sorting
type dependencyObject struct {
	Schema string
	Name   string
	Type   string
}

func getDependencySortedObjects(s *schema.Schema) []dependencyObject {
	var objects []dependencyObject
	
	// Add all tables first (they have no dependencies)
	schemaNames := s.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]
		
		tableNames := dbSchema.GetSortedTableNames()
		for _, tableName := range tableNames {
			table := dbSchema.Tables[tableName]
			if table.Type == schema.TableTypeBase {
				objects = append(objects, dependencyObject{
					Schema: schemaName,
					Name:   tableName,
					Type:   "table",
				})
			}
		}
	}
	
	// Add views (TODO: implement proper dependency sorting)
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]
		
		var viewNames []string
		for name := range dbSchema.Views {
			viewNames = append(viewNames, name)
		}
		sort.Strings(viewNames)
		
		for _, viewName := range viewNames {
			objects = append(objects, dependencyObject{
				Schema: schemaName,
				Name:   viewName,
				Type:   "view",
			})
		}
	}
	
	return objects
}
