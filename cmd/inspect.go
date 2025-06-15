package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

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
	w.WriteDDLSeparator()

	var sectionsWritten int

	// Schemas (skip public schema)
	if hasSchemas(s) {
		if sectionsWritten > 0 {
			w.WriteDDLSeparator()
		}
		writeSchemas(w, s)
		sectionsWritten++
	}

	// Functions
	if hasFunctions(s) {
		if sectionsWritten > 0 {
			w.WriteDDLSeparator()
		}
		writeFunctions(w, s)
		sectionsWritten++
	}

	// Sequences
	if hasStandaloneSequences(s) {
		if sectionsWritten > 0 {
			w.WriteDDLSeparator()
		}
		writeStandaloneSequences(w, s)
		sectionsWritten++
	}

	// Tables and Views (dependency sorted)
	if hasTablesAndViews(s) {
		if sectionsWritten > 0 {
			w.WriteDDLSeparator()
		}
		writeTablesAndViews(w, s)
		sectionsWritten++
	}

	// Column defaults (for nextval sequences)
	if hasColumnDefaults(s) {
		if sectionsWritten > 0 {
			w.WriteDDLSeparator()
		}
		writeColumnDefaults(w, s)
		sectionsWritten++
	}

	// Key constraints (PRIMARY KEY, UNIQUE, CHECK)
	if hasConstraints(s) {
		if sectionsWritten > 0 {
			w.WriteDDLSeparator()
		}
		writeConstraints(w, s)
		sectionsWritten++
	}

	// Indexes
	if hasIndexes(s) {
		if sectionsWritten > 0 {
			w.WriteDDLSeparator()
		}
		writeIndexes(w, s)
		sectionsWritten++
	}

	// Triggers
	if hasTriggers(s) {
		if sectionsWritten > 0 {
			w.WriteDDLSeparator()
		}
		writeTriggers(w, s)
		sectionsWritten++
	}

	// Foreign Key constraints
	if hasForeignKeyConstraints(s) {
		if sectionsWritten > 0 {
			w.WriteDDLSeparator()
		}
		writeForeignKeyConstraints(w, s)
		sectionsWritten++
	}

	// RLS
	if hasRLS(s) {
		if sectionsWritten > 0 {
			w.WriteDDLSeparator()
		}
		writeRLS(w, s)
		sectionsWritten++
	}

	// Footer
	w.WriteDDLSeparator()
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
}

func writeSchemas(w *schema.SQLWriter, s *schema.Schema) {
	schemaNames := s.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]
		sql := dbSchema.GenerateSQL()
		if sql != "" {
			w.WriteString(sql)
		}
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
			sql := function.GenerateSQL()
			if sql != "" {
				w.WriteString(sql)
			}
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

		for i, sequenceName := range sequenceNames {
			sequence := dbSchema.Sequences[sequenceName]
			if i > 0 {
				w.WriteDDLSeparator()
			}
			w.WriteString(sequence.GenerateSQL())

			// Add sequence ownership if present
			ownershipSQL := sequence.GenerateOwnershipSQL()
			if ownershipSQL != "" {
				w.WriteDDLSeparator()
				w.WriteString(ownershipSQL)
			}
		}
	}
}

func writeTablesAndViews(w *schema.SQLWriter, s *schema.Schema) {
	// Get all objects and sort by dependencies
	objects := getDependencySortedObjects(s)

	for i, obj := range objects {
		if i > 0 {
			w.WriteDDLSeparator()
		}

		switch obj.Type {
		case "table":
			dbSchema := s.Schemas[obj.Schema]
			table := dbSchema.Tables[obj.Name]
			w.WriteString(table.GenerateSQL())

			// Write sequences owned by this table
			if hasSequencesForTable(s, obj.Schema, obj.Name) {
				w.WriteDDLSeparator()
				writeSequencesForTable(w, s, obj.Schema, obj.Name)
			}

		case "view":
			dbSchema := s.Schemas[obj.Schema]
			view := dbSchema.Views[obj.Name]
			w.WriteString(view.GenerateSQLWithSchemaContext(s))
		}
	}
}

func writeColumnDefaults(w *schema.SQLWriter, s *schema.Schema) {
	schemaNames := s.GetSortedSchemaNames()
	first := true

	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]

		// Sort table names for deterministic output
		tableNames := dbSchema.GetSortedTableNames()
		for _, tableName := range tableNames {
			table := dbSchema.Tables[tableName]
			// Only process base tables, not views
			if table.Type == schema.TableTypeBase {
				// Generate column defaults SQL with separators between all defaults
				columns := table.GetColumnsWithSequenceDefaults()
				for _, column := range columns {
					if !first {
						w.WriteDDLSeparator() // Add separator between all column defaults
					}
					w.WriteString(column.GenerateColumnDefaultSQL(table.Name, table.Schema))
					first = false
				}
			}
		}
	}
}

func writeConstraints(w *schema.SQLWriter, s *schema.Schema) {
	schemaNames := s.GetSortedSchemaNames()
	isFirst := true

	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]

		// Sort table names for deterministic output
		tableNames := dbSchema.GetSortedTableNames()
		for _, tableName := range tableNames {
			table := dbSchema.Tables[tableName]
			// Only process base tables, not views
			if table.Type == schema.TableTypeBase {
				// Generate constraints SQL for PRIMARY KEY and UNIQUE constraints only (CHECK constraints are inline)
				constraintNames := table.GetSortedConstraintNames()
				for _, constraintName := range constraintNames {
					constraint := table.Constraints[constraintName]
					if constraint.Type == schema.ConstraintTypePrimaryKey || constraint.Type == schema.ConstraintTypeUnique {
						if !isFirst {
							w.WriteDDLSeparator() // Add separator between all constraints
						}
						w.WriteString(constraint.GenerateSQL())
						isFirst = false
					}
				}
			}
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

		// Add sequence ownership if present
		ownershipSQL := sequence.GenerateOwnershipSQL()
		if ownershipSQL != "" {
			w.WriteDDLSeparator()
			w.WriteString(ownershipSQL)
		}
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

		for i, indexName := range indexNames {
			if i > 0 {
				w.WriteDDLSeparator()
			}
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

		for i, triggerName := range triggerNames {
			if i > 0 {
				w.WriteDDLSeparator()
			}
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

		for i, constraint := range foreignKeyConstraints {
			if i > 0 {
				w.WriteDDLSeparator()
			}
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
			sql := table.GenerateRLSSQL()
			if sql != "" {
				w.WriteString(sql)
			}
		}
	}

	// RLS policies
	var hasRLSTables bool
	for _, dbSchema := range s.Schemas {
		for _, table := range dbSchema.Tables {
			if table.RLSEnabled {
				hasRLSTables = true
				break
			}
		}
		if hasRLSTables {
			break
		}
	}

	if hasRLSTables {
		w.WriteDDLSeparator()
	}

	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]

		var policyNames []string
		for name := range dbSchema.Policies {
			policyNames = append(policyNames, name)
		}
		sort.Strings(policyNames)

		for i, policyName := range policyNames {
			if i > 0 {
				w.WriteDDLSeparator()
			}
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

// Helper functions to check if sections have content

func hasSchemas(s *schema.Schema) bool {
	for schemaName := range s.Schemas {
		if schemaName != "public" {
			return true
		}
	}
	return false
}

func hasFunctions(s *schema.Schema) bool {
	for _, dbSchema := range s.Schemas {
		if len(dbSchema.Functions) > 0 {
			return true
		}
	}
	return false
}

func hasStandaloneSequences(s *schema.Schema) bool {
	for _, dbSchema := range s.Schemas {
		for _, sequence := range dbSchema.Sequences {
			if sequence.OwnedByTable == "" {
				return true
			}
		}
	}
	return false
}

func hasTablesAndViews(s *schema.Schema) bool {
	for _, dbSchema := range s.Schemas {
		if len(dbSchema.Tables) > 0 || len(dbSchema.Views) > 0 {
			return true
		}
	}
	return false
}

func hasColumnDefaults(s *schema.Schema) bool {
	for _, dbSchema := range s.Schemas {
		for _, table := range dbSchema.Tables {
			if table.Type == schema.TableTypeBase {
				if len(table.GetColumnsWithSequenceDefaults()) > 0 {
					return true
				}
			}
		}
	}
	return false
}

func hasConstraints(s *schema.Schema) bool {
	for _, dbSchema := range s.Schemas {
		for _, table := range dbSchema.Tables {
			if table.Type == schema.TableTypeBase {
				for _, constraint := range table.Constraints {
					if constraint.Type == schema.ConstraintTypePrimaryKey || constraint.Type == schema.ConstraintTypeUnique {
						return true
					}
				}
			}
		}
	}
	return false
}

func hasIndexes(s *schema.Schema) bool {
	for _, dbSchema := range s.Schemas {
		if len(dbSchema.Indexes) > 0 {
			return true
		}
	}
	return false
}

func hasTriggers(s *schema.Schema) bool {
	for _, dbSchema := range s.Schemas {
		if len(dbSchema.Triggers) > 0 {
			return true
		}
	}
	return false
}

func hasForeignKeyConstraints(s *schema.Schema) bool {
	for _, dbSchema := range s.Schemas {
		for _, table := range dbSchema.Tables {
			for _, constraint := range table.Constraints {
				if constraint.Type == schema.ConstraintTypeForeignKey {
					return true
				}
			}
		}
	}
	return false
}

func hasRLS(s *schema.Schema) bool {
	for _, dbSchema := range s.Schemas {
		for _, table := range dbSchema.Tables {
			if table.RLSEnabled {
				return true
			}
		}
		if len(dbSchema.Policies) > 0 {
			return true
		}
	}
	return false
}

func hasSequencesForTable(s *schema.Schema, schemaName, tableName string) bool {
	dbSchema := s.Schemas[schemaName]
	for _, sequence := range dbSchema.Sequences {
		if sequence.OwnedByTable == tableName {
			return true
		}
	}
	return false
}

// Helper for dependency sorting
type dependencyObject struct {
	Schema string
	Name   string
	Type   string
}

func getDependencySortedObjects(s *schema.Schema) []dependencyObject {
	var objects []dependencyObject

	schemaNames := s.GetSortedSchemaNames()

	// Build dependency-aware ordering: tables first, then views that depend on them
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]

		// Add tables in dependency-aware order (not just alphabetical)
		tableNames := getTableNamesInDependencyOrder(dbSchema)
		for _, tableName := range tableNames {
			table := dbSchema.Tables[tableName]
			if table.Type == schema.TableTypeBase {
				objects = append(objects, dependencyObject{
					Schema: schemaName,
					Name:   tableName,
					Type:   "table",
				})

				// Add views that depend on this table immediately after
				objects = append(objects, getViewsDependingOnTable(dbSchema, tableName, schemaName)...)
			}
		}
	}

	return objects
}

// getTableNamesInDependencyOrder returns table names using topological sort
func getTableNamesInDependencyOrder(dbSchema *schema.DBSchema) []string {
	// Get all table names
	var allTables []string
	for tableName := range dbSchema.Tables {
		if dbSchema.Tables[tableName].Type == schema.TableTypeBase {
			allTables = append(allTables, tableName)
		}
	}

	// For now, use simple alphabetical sorting since we don't have table dependency info
	// TODO: Implement proper topological sorting when foreign key dependencies are parsed
	sort.Strings(allTables)

	return allTables
}

// getViewsDependingOnTable returns views that should be placed after a specific table
func getViewsDependingOnTable(dbSchema *schema.DBSchema, tableName, schemaName string) []dependencyObject {
	var viewObjects []dependencyObject

	// Get views that should be placed after this table, in dependency order
	viewsForTable := getViewsForTable(dbSchema, tableName)
	for _, viewName := range viewsForTable {
		viewObjects = append(viewObjects, dependencyObject{
			Schema: schemaName,
			Name:   viewName,
			Type:   "view",
		})
	}

	return viewObjects
}

// getViewsForTable returns views that should be placed after a table, using topological sort
func getViewsForTable(dbSchema *schema.DBSchema, tableName string) []string {
	// Get all views that depend on this table
	var dependentViews []string

	for viewName, view := range dbSchema.Views {
		if viewDependsOnTable(view, tableName) {
			dependentViews = append(dependentViews, viewName)
		}
	}

	if len(dependentViews) == 0 {
		return []string{}
	}

	// Perform topological sort on the dependent views
	return topologicalSortViews(dbSchema, dependentViews)
}

// viewDependsOnTable checks if a view depends on a specific table
func viewDependsOnTable(view *schema.View, tableName string) bool {
	// Simple heuristic: check if table name appears in view definition
	// This can be enhanced with proper dependency parsing later
	return strings.Contains(strings.ToLower(view.Definition), strings.ToLower(tableName))
}

// topologicalSortViews performs topological sorting on views based on dependencies
func topologicalSortViews(dbSchema *schema.DBSchema, viewNames []string) []string {
	// Build dependency graph
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	// Initialize
	for _, viewName := range viewNames {
		inDegree[viewName] = 0
		adjList[viewName] = []string{}
	}

	// Build edges: if viewA depends on viewB, add edge viewB -> viewA
	for _, viewA := range viewNames {
		viewAObj := dbSchema.Views[viewA]
		for _, viewB := range viewNames {
			if viewA != viewB && viewDependsOnView(viewAObj, viewB) {
				adjList[viewB] = append(adjList[viewB], viewA)
				inDegree[viewA]++
			}
		}
	}

	// Kahn's algorithm for topological sorting
	var queue []string
	var result []string

	// Find all nodes with no incoming edges
	for viewName, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, viewName)
		}
	}

	// Sort initial queue alphabetically for deterministic output
	sort.Strings(queue)

	for len(queue) > 0 {
		// Remove node from queue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// For each neighbor, reduce in-degree
		neighbors := adjList[current]
		sort.Strings(neighbors) // For deterministic output

		for _, neighbor := range neighbors {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
				sort.Strings(queue) // Keep queue sorted for deterministic output
			}
		}
	}

	// Check for cycles (shouldn't happen with proper views)
	if len(result) != len(viewNames) {
		// Fallback to alphabetical sorting if cycle detected
		sort.Strings(viewNames)
		return viewNames
	}

	return result
}

// viewDependsOnView checks if viewA depends on viewB
func viewDependsOnView(viewA *schema.View, viewBName string) bool {
	// Simple heuristic: check if viewB name appears in viewA definition
	// This can be enhanced with proper SQL parsing later
	return strings.Contains(strings.ToLower(viewA.Definition), strings.ToLower(viewBName))
}
