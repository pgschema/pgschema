package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pgschema/pgschema/internal/ir"
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
	builder := ir.NewBuilder(db)
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
func generateSQL(s *ir.Schema) string {
	w := ir.NewSQLWriter()

	// Header
	writeHeader(w, s)
	w.WriteDDLSeparator()

	var sectionsWritten int

	// Extensions
	if hasExtensions(s) {
		writeExtensions(w, s)
		w.WriteDDLSeparator()
		sectionsWritten++
	}

	// Types
	if hasTypes(s) {
		writeTypes(w, s)
		w.WriteDDLSeparator()
		sectionsWritten++
	}

	// Schemas (skip public schema)
	if hasSchemas(s) {
		writeSchemas(w, s)
		w.WriteDDLSeparator()
		sectionsWritten++
	}

	// Functions
	if hasFunctions(s) {
		writeFunctions(w, s)
		w.WriteDDLSeparator()
		sectionsWritten++
	}

	// Procedures
	if hasProcedures(s) {
		writeProcedures(w, s)
		w.WriteDDLSeparator()
		sectionsWritten++
	}

	// Aggregates
	if hasAggregates(s) {
		writeAggregates(w, s)
		w.WriteDDLSeparator()
		sectionsWritten++
	}

	// Sequences
	if hasStandaloneSequences(s) {
		writeStandaloneSequences(w, s)
		w.WriteDDLSeparator()
		sectionsWritten++
	}

	// Tables and Views (dependency sorted)
	if hasTablesAndViews(s) {
		writeTablesAndViews(w, s)
		w.WriteDDLSeparator()
		sectionsWritten++
	}

	// Partition Attachments
	if hasPartitionAttachments(s) {
		writePartitionAttachments(w, s)
		w.WriteDDLSeparator()
		sectionsWritten++
	}

	// Column defaults are now handled inline in table creation

	// Key constraints (PRIMARY KEY, UNIQUE, CHECK)
	if hasConstraints(s) {
		writeConstraints(w, s)
		w.WriteDDLSeparator()
		sectionsWritten++
	}

	// Indexes
	if hasIndexes(s) {
		writeIndexes(w, s)
		w.WriteDDLSeparator()
		sectionsWritten++
	}

	// Index Attachments
	if hasIndexAttachments(s) {
		writeIndexAttachments(w, s)
		w.WriteDDLSeparator()
		sectionsWritten++
	}

	// Triggers
	if hasTriggers(s) {
		writeTriggers(w, s)
		w.WriteDDLSeparator()
		sectionsWritten++
	}

	// Foreign Key constraints
	if hasForeignKeyConstraints(s) {
		writeForeignKeyConstraints(w, s)
		w.WriteDDLSeparator()
		sectionsWritten++
	}

	// RLS
	if hasRLS(s) {
		writeRLS(w, s)
		w.WriteDDLSeparator()
		sectionsWritten++
	}

	// Footer
	writeFooter(w, s)

	return w.String()
}

func writeHeader(w *ir.SQLWriter, s *ir.Schema) {
	w.WriteString("--\n")
	w.WriteString("-- PostgreSQL database dump\n")
	w.WriteString("--\n")
	w.WriteString("\n")
	w.WriteString(fmt.Sprintf("-- Dumped from database version %s\n", s.Metadata.DatabaseVersion))
	w.WriteString(fmt.Sprintf("-- Dumped by %s\n", s.Metadata.DumpVersion))
}

func writeExtensions(w *ir.SQLWriter, s *ir.Schema) {
	// Get sorted extension names for consistent output
	var extensionNames []string
	for name := range s.Extensions {
		extensionNames = append(extensionNames, name)
	}

	// Sort extension names alphabetically
	for i := 0; i < len(extensionNames); i++ {
		for j := i + 1; j < len(extensionNames); j++ {
			if extensionNames[i] > extensionNames[j] {
				extensionNames[i], extensionNames[j] = extensionNames[j], extensionNames[i]
			}
		}
	}

	for i, extensionName := range extensionNames {
		extension := s.Extensions[extensionName]
		sql := extension.GenerateSQL()
		if sql != "" {
			w.WriteString(sql)
			// Add DDL separator between extensions (but not after the last one)
			if i < len(extensionNames)-1 {
				w.WriteDDLSeparator()
			}
		}
	}
}

func writeTypes(w *ir.SQLWriter, s *ir.Schema) {
	schemaNames := s.GetSortedSchemaNames()
	var allTypes []*ir.Type

	// Collect all types across all schemas
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]

		// Get sorted type names for consistent output
		var typeNames []string
		for name := range dbSchema.Types {
			typeNames = append(typeNames, name)
		}

		// Sort type names alphabetically
		for i := 0; i < len(typeNames); i++ {
			for j := i + 1; j < len(typeNames); j++ {
				if typeNames[i] > typeNames[j] {
					typeNames[i], typeNames[j] = typeNames[j], typeNames[i]
				}
			}
		}

		for _, typeName := range typeNames {
			allTypes = append(allTypes, dbSchema.Types[typeName])
		}
	}

	// Write types with DDL separators
	for i, customType := range allTypes {
		sql := customType.GenerateSQL()
		if sql != "" {
			w.WriteString(sql)
			// Add DDL separator between types (but not after the last one)
			if i < len(allTypes)-1 {
				w.WriteDDLSeparator()
			}
		}
	}
}

func writeSchemas(w *ir.SQLWriter, s *ir.Schema) {
	schemaNames := s.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]
		sql := dbSchema.GenerateSQL()
		if sql != "" {
			w.WriteString(sql)
		}
	}
}

func writeFunctions(w *ir.SQLWriter, s *ir.Schema) {
	schemaNames := s.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]

		// Sort function names for deterministic output
		var functionNames []string
		for name := range dbSchema.Functions {
			functionNames = append(functionNames, name)
		}
		sort.Strings(functionNames)

		for i, functionName := range functionNames {
			function := dbSchema.Functions[functionName]
			if i > 0 {
				w.WriteDDLSeparator()
			}
			sql := function.GenerateSQL()
			if sql != "" {
				w.WriteString(sql)
			}
		}
	}
}

func writeProcedures(w *ir.SQLWriter, s *ir.Schema) {
	schemaNames := s.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]

		// Sort procedure names for deterministic output
		var procedureNames []string
		for name := range dbSchema.Procedures {
			procedureNames = append(procedureNames, name)
		}
		sort.Strings(procedureNames)

		for i, procedureName := range procedureNames {
			procedure := dbSchema.Procedures[procedureName]
			if i > 0 {
				w.WriteDDLSeparator()
			}
			sql := procedure.GenerateSQL()
			if sql != "" {
				w.WriteString(sql)
			}
		}
	}
}

func writeAggregates(w *ir.SQLWriter, s *ir.Schema) {
	schemaNames := s.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]

		// Sort aggregate names for deterministic output
		var aggregateNames []string
		for name := range dbSchema.Aggregates {
			aggregateNames = append(aggregateNames, name)
		}
		sort.Strings(aggregateNames)

		for i, aggregateName := range aggregateNames {
			aggregate := dbSchema.Aggregates[aggregateName]
			if i > 0 {
				w.WriteDDLSeparator()
			}
			sql := aggregate.GenerateSQL()
			if sql != "" {
				w.WriteString(sql)
			}
		}
	}
}

func writeStandaloneSequences(w *ir.SQLWriter, s *ir.Schema) {
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

func writeTablesAndViews(w *ir.SQLWriter, s *ir.Schema) {
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

func writeConstraints(w *ir.SQLWriter, s *ir.Schema) {
	schemaNames := s.GetSortedSchemaNames()
	isFirst := true

	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]

		// Sort table names for deterministic output
		tableNames := dbSchema.GetSortedTableNames()
		for _, tableName := range tableNames {
			table := dbSchema.Tables[tableName]
			// Only process base tables, not views
			if table.Type == ir.TableTypeBase {
				// Generate constraints SQL for PRIMARY KEY and UNIQUE constraints only (CHECK constraints are inline)
				constraintNames := table.GetSortedConstraintNames()
				for _, constraintName := range constraintNames {
					constraint := table.Constraints[constraintName]
					if constraint.Type == ir.ConstraintTypePrimaryKey || constraint.Type == ir.ConstraintTypeUnique {
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

func writeSequencesForTable(w *ir.SQLWriter, s *ir.Schema, schemaName, tableName string) {
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

func writeIndexes(w *ir.SQLWriter, s *ir.Schema) {
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

func writeTriggers(w *ir.SQLWriter, s *ir.Schema) {
	schemaNames := s.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]

		var triggerNames []string
		for name := range dbSchema.Triggers {
			triggerNames = append(triggerNames, name)
		}
		// Sort triggers by table.trigger format for deterministic order
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

func writeForeignKeyConstraints(w *ir.SQLWriter, s *ir.Schema) {
	schemaNames := s.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]

		// Collect all foreign key constraints
		var foreignKeyConstraints []*ir.Constraint
		for _, table := range dbSchema.Tables {
			for _, constraint := range table.Constraints {
				if constraint.Type == ir.ConstraintTypeForeignKey {
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

func writeRLS(w *ir.SQLWriter, s *ir.Schema) {
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

func writeFooter(w *ir.SQLWriter, s *ir.Schema) {
	w.WriteString("--\n")
	w.WriteString("-- PostgreSQL database dump complete\n")
	w.WriteString("--\n")
	w.WriteString("\n")
}

// Helper functions to check if sections have content

func hasExtensions(s *ir.Schema) bool {
	return len(s.Extensions) > 0
}

func hasTypes(s *ir.Schema) bool {
	for _, dbSchema := range s.Schemas {
		if len(dbSchema.Types) > 0 {
			return true
		}
	}
	return false
}

func hasSchemas(s *ir.Schema) bool {
	for schemaName := range s.Schemas {
		if schemaName != "public" {
			return true
		}
	}
	return false
}

func hasFunctions(s *ir.Schema) bool {
	for _, dbSchema := range s.Schemas {
		if len(dbSchema.Functions) > 0 {
			return true
		}
	}
	return false
}

func hasProcedures(s *ir.Schema) bool {
	for _, dbSchema := range s.Schemas {
		if len(dbSchema.Procedures) > 0 {
			return true
		}
	}
	return false
}

func hasAggregates(s *ir.Schema) bool {
	for _, dbSchema := range s.Schemas {
		if len(dbSchema.Aggregates) > 0 {
			return true
		}
	}
	return false
}

func hasStandaloneSequences(s *ir.Schema) bool {
	for _, dbSchema := range s.Schemas {
		for _, sequence := range dbSchema.Sequences {
			if sequence.OwnedByTable == "" {
				return true
			}
		}
	}
	return false
}

func hasTablesAndViews(s *ir.Schema) bool {
	for _, dbSchema := range s.Schemas {
		if len(dbSchema.Tables) > 0 || len(dbSchema.Views) > 0 {
			return true
		}
	}
	return false
}

func hasConstraints(s *ir.Schema) bool {
	for _, dbSchema := range s.Schemas {
		for _, table := range dbSchema.Tables {
			if table.Type == ir.TableTypeBase {
				for _, constraint := range table.Constraints {
					if constraint.Type == ir.ConstraintTypePrimaryKey || constraint.Type == ir.ConstraintTypeUnique {
						return true
					}
				}
			}
		}
	}
	return false
}

func hasIndexes(s *ir.Schema) bool {
	for _, dbSchema := range s.Schemas {
		if len(dbSchema.Indexes) > 0 {
			return true
		}
	}
	return false
}

func hasTriggers(s *ir.Schema) bool {
	for _, dbSchema := range s.Schemas {
		if len(dbSchema.Triggers) > 0 {
			return true
		}
	}
	return false
}

func hasForeignKeyConstraints(s *ir.Schema) bool {
	for _, dbSchema := range s.Schemas {
		for _, table := range dbSchema.Tables {
			for _, constraint := range table.Constraints {
				if constraint.Type == ir.ConstraintTypeForeignKey {
					return true
				}
			}
		}
	}
	return false
}

func hasRLS(s *ir.Schema) bool {
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

func hasSequencesForTable(s *ir.Schema, schemaName, tableName string) bool {
	dbSchema := s.Schemas[schemaName]
	for _, sequence := range dbSchema.Sequences {
		if sequence.OwnedByTable == tableName {
			return true
		}
	}
	return false
}

func hasPartitionAttachments(s *ir.Schema) bool {
	return len(s.PartitionAttachments) > 0
}

func hasIndexAttachments(s *ir.Schema) bool {
	return len(s.IndexAttachments) > 0
}

func writePartitionAttachments(w *ir.SQLWriter, s *ir.Schema) {
	for i, attachment := range s.PartitionAttachments {
		if i > 0 {
			w.WriteDDLSeparator()
		}
		stmt := fmt.Sprintf("ALTER TABLE ONLY %s.%s ATTACH PARTITION %s.%s %s;",
			attachment.ParentSchema, attachment.ParentTable,
			attachment.ChildSchema, attachment.ChildTable,
			attachment.PartitionBound)
		w.WriteStatementWithComment("TABLE ATTACH", attachment.ChildTable, attachment.ChildSchema, "", stmt)
	}
}

func writeIndexAttachments(w *ir.SQLWriter, s *ir.Schema) {
	for i, attachment := range s.IndexAttachments {
		if i > 0 {
			w.WriteDDLSeparator()
		}
		stmt := fmt.Sprintf("ALTER INDEX %s.%s ATTACH PARTITION %s.%s;",
			attachment.ParentSchema, attachment.ParentIndex,
			attachment.ChildSchema, attachment.ChildIndex)
		w.WriteStatementWithComment("INDEX ATTACH", attachment.ChildIndex, attachment.ChildSchema, "", stmt)
	}
}

// Helper for dependency sorting
type dependencyObject struct {
	Schema string
	Name   string
	Type   string
}

func getDependencySortedObjects(s *ir.Schema) []dependencyObject {
	var objects []dependencyObject

	schemaNames := s.GetSortedSchemaNames()

	// Build dependency-aware ordering: tables first, then views that depend on them
	for _, schemaName := range schemaNames {
		dbSchema := s.Schemas[schemaName]

		// Add tables in dependency-aware order (not just alphabetical)
		tableNames := getTableNamesInDependencyOrder(dbSchema)
		for _, tableName := range tableNames {
			table := dbSchema.Tables[tableName]
			if table.Type == ir.TableTypeBase {
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
func getTableNamesInDependencyOrder(dbSchema *ir.DBSchema) []string {
	// Get all table names
	var allTables []string
	for tableName := range dbSchema.Tables {
		if dbSchema.Tables[tableName].Type == ir.TableTypeBase {
			allTables = append(allTables, tableName)
		}
	}

	// For now, use simple alphabetical sorting since we don't have table dependency info
	// TODO: Implement proper topological sorting when foreign key dependencies are parsed
	sort.Strings(allTables)

	return allTables
}

// getViewsDependingOnTable returns views that should be placed after a specific table
func getViewsDependingOnTable(dbSchema *ir.DBSchema, tableName, schemaName string) []dependencyObject {
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
func getViewsForTable(dbSchema *ir.DBSchema, tableName string) []string {
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
func viewDependsOnTable(view *ir.View, tableName string) bool {
	// Simple heuristic: check if table name appears in view definition
	// This can be enhanced with proper dependency parsing later
	return strings.Contains(strings.ToLower(view.Definition), strings.ToLower(tableName))
}

// topologicalSortViews performs topological sorting on views based on dependencies
func topologicalSortViews(dbSchema *ir.DBSchema, viewNames []string) []string {
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
func viewDependsOnView(viewA *ir.View, viewBName string) bool {
	// Simple heuristic: check if viewB name appears in viewA definition
	// This can be enhanced with proper SQL parsing later
	return strings.Contains(strings.ToLower(viewA.Definition), strings.ToLower(viewBName))
}
