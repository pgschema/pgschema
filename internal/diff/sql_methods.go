package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
	"github.com/pgschema/pgschema/internal/utils"
)

// SQL generation methods for DDLDiff that follow the SQL generator pattern

// generateDropTablesSQL generates DROP TABLE statements
func (d *DDLDiff) generateDropTablesSQL(w *SQLWriter, tables []*ir.Table, targetSchema string) {
	// Group tables by schema for topological sorting
	tablesBySchema := make(map[string][]*ir.Table)
	for _, table := range tables {
		tablesBySchema[table.Schema] = append(tablesBySchema[table.Schema], table)
	}

	// Process each schema using reverse topological sorting for drops
	for schemaName, schemaTables := range tablesBySchema {
		// Build a temporary schema with just these tables for topological sorting
		tempSchema := &ir.Schema{
			Name:   schemaName,
			Tables: make(map[string]*ir.Table),
		}
		for _, table := range schemaTables {
			tempSchema.Tables[table.Name] = table
		}

		// Get topologically sorted table names, then reverse for drop order
		sortedTableNames := tempSchema.GetTopologicallySortedTableNames()

		// Reverse the order for dropping (dependencies first)
		for i := len(sortedTableNames) - 1; i >= 0; i-- {
			tableName := sortedTableNames[i]
			table := tempSchema.Tables[tableName]
			w.WriteDDLSeparator()
			sql := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", table.Name)
			w.WriteStatementWithComment("TABLE", table.Name, table.Schema, "", sql, targetSchema)
		}
	}
}

// generateCreateTablesSQL generates CREATE TABLE statements with co-located indexes, constraints, triggers, and RLS
func (d *DDLDiff) generateCreateTablesSQL(w *SQLWriter, tables []*ir.Table, targetSchema string) {
	isDumpScenario := len(d.AddedTables) > 0 && len(d.DroppedTables) == 0 && len(d.ModifiedTables) == 0
	
	// Group tables by schema for topological sorting
	tablesBySchema := make(map[string][]*ir.Table)
	for _, table := range tables {
		tablesBySchema[table.Schema] = append(tablesBySchema[table.Schema], table)
	}

	// Process each schema using topological sorting
	for schemaName, schemaTables := range tablesBySchema {
		// Build a temporary schema with just these tables for topological sorting
		tempSchema := &ir.Schema{
			Name:   schemaName,
			Tables: make(map[string]*ir.Table),
		}
		for _, table := range schemaTables {
			tempSchema.Tables[table.Name] = table
		}

		// Get topologically sorted table names for dependency-aware output
		sortedTableNames := tempSchema.GetTopologicallySortedTableNames()

		// Process tables in topological order
		for _, tableName := range sortedTableNames {
			table := tempSchema.Tables[tableName]

			// Create the table
			w.WriteDDLSeparator()
			sql := d.generateTableSQL(table, targetSchema)
			w.WriteStatementWithComment("TABLE", table.Name, table.Schema, "", sql, targetSchema)

			// Co-locate table-related objects immediately after the table
			d.generateTableIndexes(w, table, targetSchema)
			d.generateTableConstraints(w, table, targetSchema)
			d.generateTableTriggers(w, table, targetSchema)
			generateTableRLS(w, table, targetSchema, d.AddedPolicies, isDumpScenario)
		}
	}
}

// generateModifyTablesSQL generates ALTER TABLE statements
func (d *DDLDiff) generateModifyTablesSQL(w *SQLWriter, diffs []*TableDiff, targetSchema string) {
	for _, diff := range diffs {
		statements := diff.GenerateMigrationSQL()
		for _, stmt := range statements {
			w.WriteDDLSeparator()
			w.WriteStatementWithComment("TABLE", diff.Table.Name, diff.Table.Schema, "", stmt, targetSchema)
		}
	}
}

// generateDropViewsSQL generates DROP VIEW statements
func (d *DDLDiff) generateDropViewsSQL(w *SQLWriter, views []*ir.View, targetSchema string) {
	// Group views by schema for topological sorting
	viewsBySchema := make(map[string][]*ir.View)
	for _, view := range views {
		viewsBySchema[view.Schema] = append(viewsBySchema[view.Schema], view)
	}

	// Process each schema using reverse topological sorting for drops
	for schemaName, schemaViews := range viewsBySchema {
		// Build a temporary schema with just these views for topological sorting
		tempSchema := &ir.Schema{
			Name:  schemaName,
			Views: make(map[string]*ir.View),
		}
		for _, view := range schemaViews {
			tempSchema.Views[view.Name] = view
		}

		// Get topologically sorted view names, then reverse for drop order
		sortedViewNames := tempSchema.GetTopologicallySortedViewNames()

		// Reverse the order for dropping (dependencies first)
		for i := len(sortedViewNames) - 1; i >= 0; i-- {
			viewName := sortedViewNames[i]
			view := tempSchema.Views[viewName]
			w.WriteDDLSeparator()
			sql := fmt.Sprintf("DROP VIEW IF EXISTS %s CASCADE;", view.Name)
			w.WriteStatementWithComment("VIEW", view.Name, view.Schema, "", sql, targetSchema)
		}
	}
}

// generateCreateViewsSQL generates CREATE VIEW statements
func (d *DDLDiff) generateCreateViewsSQL(w *SQLWriter, views []*ir.View, targetSchema string) {
	// Group views by schema for topological sorting
	viewsBySchema := make(map[string][]*ir.View)
	for _, view := range views {
		viewsBySchema[view.Schema] = append(viewsBySchema[view.Schema], view)
	}

	// Process each schema using topological sorting
	for schemaName, schemaViews := range viewsBySchema {
		// Build a temporary schema with just these views for topological sorting
		tempSchema := &ir.Schema{
			Name:  schemaName,
			Views: make(map[string]*ir.View),
		}
		for _, view := range schemaViews {
			tempSchema.Views[view.Name] = view
		}

		// Get topologically sorted view names for dependency-aware output
		sortedViewNames := tempSchema.GetTopologicallySortedViewNames()

		// Process views in topological order
		for _, viewName := range sortedViewNames {
			view := tempSchema.Views[viewName]
			w.WriteDDLSeparator()
			sql := d.generateViewSQL(view, targetSchema)
			w.WriteStatementWithComment("VIEW", view.Name, view.Schema, "", sql, targetSchema)
		}
	}
}

// generateModifyViewsSQL generates ALTER VIEW statements
func (d *DDLDiff) generateModifyViewsSQL(w *SQLWriter, diffs []*ViewDiff, targetSchema string) {
	for _, diff := range diffs {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("CREATE OR REPLACE VIEW %s AS %s;", diff.New.Name, diff.New.Definition)
		w.WriteStatementWithComment("VIEW", diff.New.Name, diff.New.Schema, "", sql, targetSchema)
	}
}


// generateTableIndexes generates SQL for indexes belonging to a specific table
func (d *DDLDiff) generateTableIndexes(w *SQLWriter, table *ir.Table, targetSchema string) {
	// Get sorted index names for consistent output
	indexNames := make([]string, 0, len(table.Indexes))
	for indexName := range table.Indexes {
		indexNames = append(indexNames, indexName)
	}
	sort.Strings(indexNames)

	for _, indexName := range indexNames {
		index := table.Indexes[indexName]
		// Skip primary key indexes as they're handled with constraints
		if index.IsPrimary {
			continue
		}

		// Include all indexes for this table (for dump scenarios) or only added indexes (for diff scenarios)
		if d.isIndexInAddedList(index) {
			w.WriteDDLSeparator()
			sql := generateIndexSQL(index, targetSchema)
			w.WriteStatementWithComment("INDEX", indexName, table.Schema, "", sql, targetSchema)
		}
	}
}

// generateTableConstraints generates SQL for constraints belonging to a specific table
func (d *DDLDiff) generateTableConstraints(w *SQLWriter, table *ir.Table, targetSchema string) {
	// Get sorted constraint names for consistent output
	constraintNames := make([]string, 0, len(table.Constraints))
	for constraintName := range table.Constraints {
		constraintNames = append(constraintNames, constraintName)
	}
	sort.Strings(constraintNames)

	for _, constraintName := range constraintNames {
		constraint := table.Constraints[constraintName]
		// Skip PRIMARY KEY, UNIQUE, FOREIGN KEY, and CHECK constraints as they are now inline in CREATE TABLE
		if constraint.Type == ir.ConstraintTypePrimaryKey ||
			constraint.Type == ir.ConstraintTypeUnique ||
			constraint.Type == ir.ConstraintTypeForeignKey ||
			constraint.Type == ir.ConstraintTypeCheck {
			continue
		}

		// Only include constraints that would be in the added list
		w.WriteDDLSeparator()
		constraintSQL := d.generateConstraintSQL(constraint, targetSchema)
		w.WriteStatementWithComment("CONSTRAINT", constraintName, table.Schema, "", constraintSQL, targetSchema)
	}
}


// Helper methods to check if objects are in the added lists
func (d *DDLDiff) isIndexInAddedList(index *ir.Index) bool {
	for _, addedIndex := range d.AddedIndexes {
		if addedIndex.Name == index.Name && addedIndex.Schema == index.Schema && addedIndex.Table == index.Table {
			return true
		}
	}
	return false
}


// generateTableSQL generates CREATE TABLE statement
func (d *DDLDiff) generateTableSQL(table *ir.Table, targetSchema string) string {
	// Only include table name without schema if it's in the target schema
	tableName := utils.QualifyEntityName(table.Schema, table.Name, targetSchema)

	var parts []string
	parts = append(parts, fmt.Sprintf("CREATE TABLE %s (", tableName))

	// Add columns
	var columnParts []string
	for _, column := range table.Columns {
		// Build column definition with SERIAL detection
		var builder strings.Builder
		writeColumnDefinitionToBuilder(&builder, table, column, targetSchema)
		columnParts = append(columnParts, fmt.Sprintf("    %s", builder.String()))
	}

	// Add constraints inline in the correct order (PRIMARY KEY, UNIQUE, FOREIGN KEY)
	inlineConstraints := getInlineConstraintsForTable(table)
	for _, constraint := range inlineConstraints {
		constraintDef := d.generateConstraintSQL(constraint, targetSchema)
		if constraintDef != "" {
			columnParts = append(columnParts, fmt.Sprintf("    %s", constraintDef))
		}
	}

	parts = append(parts, strings.Join(columnParts, ",\n"))

	// Add partition clause for partitioned tables
	if table.IsPartitioned && table.PartitionStrategy != "" && table.PartitionKey != "" {
		parts = append(parts, fmt.Sprintf(")\nPARTITION BY %s (%s);", table.PartitionStrategy, table.PartitionKey))
	} else {
		parts = append(parts, ");")
	}

	return strings.Join(parts, "\n")
}

// generateViewSQL generates CREATE VIEW statement
func (d *DDLDiff) generateViewSQL(view *ir.View, targetSchema string) string {
	// Only include view name without schema if it's in the target schema
	viewName := utils.QualifyEntityName(view.Schema, view.Name, targetSchema)
	return fmt.Sprintf("CREATE VIEW %s AS\n%s", viewName, view.Definition)
}

// generateSequenceSQL was removed as it's unused
// (sequences are not tracked separately in DDL diffs)

// getSortedConstraintNames returns constraint names sorted alphabetically
func getSortedConstraintNames(constraints map[string]*ir.Constraint) []string {
	return utils.SortedKeys(constraints)
}

// getInlineConstraintsForTable returns constraints in the correct order: PRIMARY KEY, UNIQUE, FOREIGN KEY
func getInlineConstraintsForTable(table *ir.Table) []*ir.Constraint {
	var inlineConstraints []*ir.Constraint

	// Get constraint names sorted for consistent output
	constraintNames := getSortedConstraintNames(table.Constraints)

	// Separate constraints by type for proper ordering
	var primaryKeys []*ir.Constraint
	var uniques []*ir.Constraint
	var foreignKeys []*ir.Constraint
	var checkConstraints []*ir.Constraint

	for _, constraintName := range constraintNames {
		constraint := table.Constraints[constraintName]

		// Categorize constraints by type
		switch constraint.Type {
		case ir.ConstraintTypePrimaryKey:
			primaryKeys = append(primaryKeys, constraint)
		case ir.ConstraintTypeUnique:
			uniques = append(uniques, constraint)
		case ir.ConstraintTypeForeignKey:
			foreignKeys = append(foreignKeys, constraint)
		case ir.ConstraintTypeCheck:
			// Only include table-level CHECK constraints (not column-level ones)
			// Column-level CHECK constraints are handled inline with the column definition
			if len(constraint.Columns) != 1 {
				checkConstraints = append(checkConstraints, constraint)
			}
		}
	}

	// Add constraints in order: PRIMARY KEY, UNIQUE, FOREIGN KEY, CHECK
	inlineConstraints = append(inlineConstraints, primaryKeys...)
	inlineConstraints = append(inlineConstraints, uniques...)
	inlineConstraints = append(inlineConstraints, foreignKeys...)
	inlineConstraints = append(inlineConstraints, checkConstraints...)

	return inlineConstraints
}

// generateConstraintSQL generates constraint definition for inline table constraints
func (d *DDLDiff) generateConstraintSQL(constraint *ir.Constraint, targetSchema string) string {
	// Helper function to get column names from ConstraintColumn array
	getColumnNames := func(columns []*ir.ConstraintColumn) []string {
		var names []string
		for _, col := range columns {
			names = append(names, col.Name)
		}
		return names
	}

	switch constraint.Type {
	case ir.ConstraintTypePrimaryKey:
		return fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(getColumnNames(constraint.Columns), ", "))
	case ir.ConstraintTypeUnique:
		return fmt.Sprintf("UNIQUE (%s)", strings.Join(getColumnNames(constraint.Columns), ", "))
	case ir.ConstraintTypeForeignKey:
		stmt := fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s (%s)",
			strings.Join(getColumnNames(constraint.Columns), ", "),
			constraint.ReferencedTable, strings.Join(getColumnNames(constraint.ReferencedColumns), ", "))
		// Only add ON DELETE/UPDATE if they are not the default "NO ACTION"
		if constraint.DeleteRule != "" && constraint.DeleteRule != "NO ACTION" {
			stmt += fmt.Sprintf(" ON DELETE %s", constraint.DeleteRule)
		}
		if constraint.UpdateRule != "" && constraint.UpdateRule != "NO ACTION" {
			stmt += fmt.Sprintf(" ON UPDATE %s", constraint.UpdateRule)
		}
		return stmt
	case ir.ConstraintTypeCheck:
		return constraint.CheckClause
	default:
		return ""
	}
}
