package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

// stripSchemaPrefix removes the schema prefix from a type name if it matches the target schema
func stripSchemaPrefix(typeName, targetSchema string) string {
	if typeName == "" || targetSchema == "" {
		return typeName
	}

	// Check if the type has the target schema prefix
	prefix := targetSchema + "."
	if after, found := strings.CutPrefix(typeName, prefix); found {
		return after
	}

	return typeName
}

// sortConstraintColumnsByPosition sorts constraint columns by their position
func sortConstraintColumnsByPosition(columns []*ir.ConstraintColumn) []*ir.ConstraintColumn {
	sorted := make([]*ir.ConstraintColumn, len(columns))
	copy(sorted, columns)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Position < sorted[j].Position
	})
	return sorted
}

// diffTables compares two tables and returns the differences
func diffTables(oldTable, newTable *ir.Table) *TableDiff {
	diff := &TableDiff{
		Table:              newTable,
		AddedColumns:       []*ir.Column{},
		DroppedColumns:     []*ir.Column{},
		ModifiedColumns:    []*ColumnDiff{},
		AddedConstraints:   []*ir.Constraint{},
		DroppedConstraints: []*ir.Constraint{},
		AddedIndexes:       []*ir.Index{},
		DroppedIndexes:     []*ir.Index{},
		AddedTriggers:      []*ir.Trigger{},
		DroppedTriggers:    []*ir.Trigger{},
		ModifiedTriggers:   []*TriggerDiff{},
		AddedPolicies:      []*ir.RLSPolicy{},
		DroppedPolicies:    []*ir.RLSPolicy{},
		ModifiedPolicies:   []*PolicyDiff{},
		RLSChanges:         []*RLSChange{},
	}

	// Build maps for efficient lookup
	oldColumns := make(map[string]*ir.Column)
	newColumns := make(map[string]*ir.Column)

	for _, column := range oldTable.Columns {
		oldColumns[column.Name] = column
	}

	for _, column := range newTable.Columns {
		newColumns[column.Name] = column
	}

	// Find added columns
	for name, column := range newColumns {
		if _, exists := oldColumns[name]; !exists {
			diff.AddedColumns = append(diff.AddedColumns, column)
		}
	}

	// Find dropped columns
	for name, column := range oldColumns {
		if _, exists := newColumns[name]; !exists {
			diff.DroppedColumns = append(diff.DroppedColumns, column)
		}
	}

	// Find modified columns
	for name, newColumn := range newColumns {
		if oldColumn, exists := oldColumns[name]; exists {
			if !columnsEqual(oldColumn, newColumn) {
				diff.ModifiedColumns = append(diff.ModifiedColumns, &ColumnDiff{
					Old: oldColumn,
					New: newColumn,
				})
			}
		}
	}

	// Compare constraints
	oldConstraints := make(map[string]*ir.Constraint)
	newConstraints := make(map[string]*ir.Constraint)

	if oldTable.Constraints != nil {
		for name, constraint := range oldTable.Constraints {
			oldConstraints[name] = constraint
		}
	}

	if newTable.Constraints != nil {
		for name, constraint := range newTable.Constraints {
			newConstraints[name] = constraint
		}
	}

	// Find added constraints
	for name, constraint := range newConstraints {
		if _, exists := oldConstraints[name]; !exists {
			diff.AddedConstraints = append(diff.AddedConstraints, constraint)
		}
	}

	// Find dropped constraints
	for name, constraint := range oldConstraints {
		if _, exists := newConstraints[name]; !exists {
			diff.DroppedConstraints = append(diff.DroppedConstraints, constraint)
		}
	}

	// Compare indexes
	oldIndexes := make(map[string]*ir.Index)
	newIndexes := make(map[string]*ir.Index)

	for _, index := range oldTable.Indexes {
		oldIndexes[index.Name] = index
	}

	for _, index := range newTable.Indexes {
		newIndexes[index.Name] = index
	}

	// Find added indexes
	for name, index := range newIndexes {
		if _, exists := oldIndexes[name]; !exists {
			diff.AddedIndexes = append(diff.AddedIndexes, index)
		}
	}

	// Find dropped indexes
	for name, index := range oldIndexes {
		if _, exists := newIndexes[name]; !exists {
			diff.DroppedIndexes = append(diff.DroppedIndexes, index)
		}
	}

	// Compare triggers
	oldTriggers := make(map[string]*ir.Trigger)
	newTriggers := make(map[string]*ir.Trigger)

	if oldTable.Triggers != nil {
		for name, trigger := range oldTable.Triggers {
			oldTriggers[name] = trigger
		}
	}

	if newTable.Triggers != nil {
		for name, trigger := range newTable.Triggers {
			newTriggers[name] = trigger
		}
	}

	// Find added triggers
	for name, trigger := range newTriggers {
		if _, exists := oldTriggers[name]; !exists {
			diff.AddedTriggers = append(diff.AddedTriggers, trigger)
		}
	}

	// Find dropped triggers
	for name, trigger := range oldTriggers {
		if _, exists := newTriggers[name]; !exists {
			diff.DroppedTriggers = append(diff.DroppedTriggers, trigger)
		}
	}

	// Find modified triggers
	for name, newTrigger := range newTriggers {
		if oldTrigger, exists := oldTriggers[name]; exists {
			if !triggersEqual(oldTrigger, newTrigger) {
				diff.ModifiedTriggers = append(diff.ModifiedTriggers, &TriggerDiff{
					Old: oldTrigger,
					New: newTrigger,
				})
			}
		}
	}

	// Compare policies
	oldPolicies := make(map[string]*ir.RLSPolicy)
	newPolicies := make(map[string]*ir.RLSPolicy)

	if oldTable.Policies != nil {
		for name, policy := range oldTable.Policies {
			oldPolicies[name] = policy
		}
	}

	if newTable.Policies != nil {
		for name, policy := range newTable.Policies {
			newPolicies[name] = policy
		}
	}

	// Find added policies
	for name, policy := range newPolicies {
		if _, exists := oldPolicies[name]; !exists {
			diff.AddedPolicies = append(diff.AddedPolicies, policy)
		}
	}

	// Find dropped policies
	for name, policy := range oldPolicies {
		if _, exists := newPolicies[name]; !exists {
			diff.DroppedPolicies = append(diff.DroppedPolicies, policy)
		}
	}

	// Find modified policies
	for name, newPolicy := range newPolicies {
		if oldPolicy, exists := oldPolicies[name]; exists {
			if !policiesEqual(oldPolicy, newPolicy) {
				diff.ModifiedPolicies = append(diff.ModifiedPolicies, &PolicyDiff{
					Old: oldPolicy,
					New: newPolicy,
				})
			}
		}
	}

	// Check for RLS enable/disable changes
	if oldTable.RLSEnabled != newTable.RLSEnabled {
		diff.RLSChanges = append(diff.RLSChanges, &RLSChange{
			Table:   newTable,
			Enabled: newTable.RLSEnabled,
		})
	}

	// Return nil if no changes
	if len(diff.AddedColumns) == 0 && len(diff.DroppedColumns) == 0 &&
		len(diff.ModifiedColumns) == 0 && len(diff.AddedConstraints) == 0 &&
		len(diff.DroppedConstraints) == 0 && len(diff.AddedIndexes) == 0 &&
		len(diff.DroppedIndexes) == 0 && len(diff.AddedTriggers) == 0 &&
		len(diff.DroppedTriggers) == 0 && len(diff.ModifiedTriggers) == 0 &&
		len(diff.AddedPolicies) == 0 && len(diff.DroppedPolicies) == 0 &&
		len(diff.ModifiedPolicies) == 0 && len(diff.RLSChanges) == 0 {
		return nil
	}

	return diff
}

// generateCreateTablesSQL generates CREATE TABLE statements with co-located indexes, constraints, triggers, and RLS
// Tables are assumed to be pre-sorted in topological order for dependency-aware creation
func generateCreateTablesSQL(w *SQLWriter, tables []*ir.Table, targetSchema string, compare bool) {
	// Process tables in the provided order (already topologically sorted)
	for _, table := range tables {
		// Create the table
		w.WriteDDLSeparator()
		sql := generateTableSQL(table, targetSchema)
		w.WriteStatementWithComment("TABLE", table.Name, table.Schema, "", sql, targetSchema)

		// Convert map to slice for indexes
		indexes := make([]*ir.Index, 0, len(table.Indexes))
		for _, index := range table.Indexes {
			indexes = append(indexes, index)
		}
		generateCreateIndexesSQL(w, indexes, targetSchema)

		// Handle RLS enable changes (before creating policies) - only for diff scenarios
		if table.RLSEnabled {
			rlsChanges := []*RLSChange{{Table: table, Enabled: true}}
			generateRLSChangesSQL(w, rlsChanges, targetSchema)
		}

		// Create policies - only for diff scenarios
		policies := make([]*ir.RLSPolicy, 0, len(table.Policies))
		for _, policy := range table.Policies {
			policies = append(policies, policy)
		}
		generateCreatePoliciesSQL(w, policies, targetSchema)

		// Create triggers - only for diff scenarios
		triggers := make([]*ir.Trigger, 0, len(table.Triggers))
		for _, trigger := range table.Triggers {
			triggers = append(triggers, trigger)
		}
		generateCreateTriggersSQL(w, triggers, targetSchema, compare)
	}
}

// generateModifyTablesSQL generates ALTER TABLE statements
func generateModifyTablesSQL(w *SQLWriter, diffs []*TableDiff, targetSchema string) {
	for _, diff := range diffs {
		statements := diff.generateAlterTableStatements()
		for _, stmt := range statements {
			w.WriteDDLSeparator()
			w.WriteStatementWithComment("TABLE", diff.Table.Name, diff.Table.Schema, "", stmt, targetSchema)
		}
	}
}

// generateDropTablesSQL generates DROP TABLE statements
// Tables are assumed to be pre-sorted in reverse topological order for dependency-aware dropping
func generateDropTablesSQL(w *SQLWriter, tables []*ir.Table, targetSchema string) {
	// Process tables in the provided order (already reverse topologically sorted)
	for _, table := range tables {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", table.Name)
		w.WriteStatementWithComment("TABLE", table.Name, table.Schema, "", sql, targetSchema)
	}
}

// generateTableSQL generates CREATE TABLE statement
func generateTableSQL(table *ir.Table, targetSchema string) string {
	// Only include table name without schema if it's in the target schema
	tableName := qualifyEntityName(table.Schema, table.Name, targetSchema)

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
		constraintDef := generateConstraintSQL(constraint, targetSchema)
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

// generateAlterTableStatements generates SQL statements for table modifications
func (td *TableDiff) generateAlterTableStatements() []string {
	var statements []string

	// Drop constraints first (before dropping columns)
	for _, constraint := range td.DroppedConstraints {
		tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, td.Table.Schema)
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;",
			tableName, constraint.Name))
	}

	// Drop columns (sort by position for consistent ordering)
	sortedDroppedColumns := make([]*ir.Column, len(td.DroppedColumns))
	copy(sortedDroppedColumns, td.DroppedColumns)
	sort.Slice(sortedDroppedColumns, func(i, j int) bool {
		return sortedDroppedColumns[i].Position < sortedDroppedColumns[j].Position
	})
	for _, column := range sortedDroppedColumns {
		tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, td.Table.Schema)
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;",
			tableName, column.Name))
	}

	// Add new columns (sort by position for consistent ordering)
	sortedAddedColumns := make([]*ir.Column, len(td.AddedColumns))
	copy(sortedAddedColumns, td.AddedColumns)
	sort.Slice(sortedAddedColumns, func(i, j int) bool {
		return sortedAddedColumns[i].Position < sortedAddedColumns[j].Position
	})

	// Track which FK constraints are handled inline with column additions
	handledFKConstraints := make(map[string]bool)

	for _, column := range sortedAddedColumns {
		// Check if this column has an associated foreign key constraint
		var fkConstraint *ir.Constraint
		for _, constraint := range td.Table.Constraints {
			if constraint.Type == ir.ConstraintTypeForeignKey &&
				len(constraint.Columns) == 1 &&
				constraint.Columns[0].Name == column.Name {
				fkConstraint = constraint
				handledFKConstraints[constraint.Name] = true
				break
			}
		}

		// Use line break format for complex statements (with foreign keys)
		var stmt string
		tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, td.Table.Schema)
		if fkConstraint != nil {
			stmt = fmt.Sprintf("ALTER TABLE %s\nADD COLUMN %s %s",
				tableName, column.Name, column.DataType)
		} else {
			stmt = fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
				tableName, column.Name, column.DataType)
		}

		// Add foreign key reference inline if present
		if fkConstraint != nil {
			referencedTableName := getTableNameWithSchema(fkConstraint.ReferencedSchema, fkConstraint.ReferencedTable, td.Table.Schema)
			stmt += fmt.Sprintf(" REFERENCES %s", referencedTableName)

			if len(fkConstraint.ReferencedColumns) > 0 {
				var refCols []string
				for _, refCol := range fkConstraint.ReferencedColumns {
					refCols = append(refCols, refCol.Name)
				}
				stmt += fmt.Sprintf("(%s)", strings.Join(refCols, ", "))
			}

			// Add referential actions
			if fkConstraint.UpdateRule != "" && fkConstraint.UpdateRule != "NO ACTION" {
				stmt += fmt.Sprintf(" ON UPDATE %s", fkConstraint.UpdateRule)
			}
			if fkConstraint.DeleteRule != "" && fkConstraint.DeleteRule != "NO ACTION" {
				stmt += fmt.Sprintf(" ON DELETE %s", fkConstraint.DeleteRule)
			}

			// Add deferrable clause
			if fkConstraint.Deferrable {
				if fkConstraint.InitiallyDeferred {
					stmt += " DEFERRABLE INITIALLY DEFERRED"
				} else {
					stmt += " DEFERRABLE"
				}
			}
		}

		// Add identity column syntax
		if column.Identity != nil {
			if column.Identity.Generation == "ALWAYS" {
				stmt += " GENERATED ALWAYS AS IDENTITY"
			} else if column.Identity.Generation == "BY DEFAULT" {
				stmt += " GENERATED BY DEFAULT AS IDENTITY"
			}
		}

		if column.DefaultValue != nil && column.Identity == nil {
			stmt += fmt.Sprintf(" DEFAULT %s", *column.DefaultValue)
		}

		if !column.IsNullable {
			stmt += " NOT NULL"
		}

		statements = append(statements, stmt+";")
	}

	// Modify existing columns (sort by position to maintain column order)
	sortedModifiedColumns := make([]*ColumnDiff, len(td.ModifiedColumns))
	copy(sortedModifiedColumns, td.ModifiedColumns)
	sort.Slice(sortedModifiedColumns, func(i, j int) bool {
		return sortedModifiedColumns[i].New.Position < sortedModifiedColumns[j].New.Position
	})
	for _, columnDiff := range sortedModifiedColumns {
		statements = append(statements, columnDiff.generateColumnSQL(td.Table.Schema, td.Table.Name)...)
	}

	// Add new constraints (sort by name for consistent ordering)
	sortedConstraints := make([]*ir.Constraint, 0)
	for _, constraint := range td.AddedConstraints {
		// Skip FK constraints that were already handled inline with column additions
		if constraint.Type == ir.ConstraintTypeForeignKey && handledFKConstraints[constraint.Name] {
			continue
		}
		sortedConstraints = append(sortedConstraints, constraint)
	}
	sort.Slice(sortedConstraints, func(i, j int) bool {
		return sortedConstraints[i].Name < sortedConstraints[j].Name
	})
	for _, constraint := range sortedConstraints {
		switch constraint.Type {
		case ir.ConstraintTypeUnique:
			// Sort columns by position
			columns := sortConstraintColumnsByPosition(constraint.Columns)
			var columnNames []string
			for _, col := range columns {
				columnNames = append(columnNames, col.Name)
			}
			tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, td.Table.Schema)
			stmt := fmt.Sprintf("ALTER TABLE %s \nADD CONSTRAINT %s UNIQUE (%s);",
				tableName, constraint.Name, strings.Join(columnNames, ", "))
			statements = append(statements, stmt)

		case ir.ConstraintTypeCheck:
			// CheckClause already contains "CHECK (...)" from the constraint definition
			tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, td.Table.Schema)
			stmt := fmt.Sprintf("ALTER TABLE %s \nADD CONSTRAINT %s %s;",
				tableName, constraint.Name, constraint.CheckClause)
			statements = append(statements, stmt)

		case ir.ConstraintTypeForeignKey:
			// Sort columns by position
			columns := sortConstraintColumnsByPosition(constraint.Columns)
			var columnNames []string
			for _, col := range columns {
				columnNames = append(columnNames, col.Name)
			}

			// Sort referenced columns by position
			var refColumnNames []string
			if len(constraint.ReferencedColumns) > 0 {
				refColumns := sortConstraintColumnsByPosition(constraint.ReferencedColumns)
				for _, col := range refColumns {
					refColumnNames = append(refColumnNames, col.Name)
				}
			}

			tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, td.Table.Schema)
			referencedTableName := getTableNameWithSchema(constraint.ReferencedSchema, constraint.ReferencedTable, td.Table.Schema)
			stmt := fmt.Sprintf("ALTER TABLE %s \nADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s)",
				tableName, constraint.Name,
				strings.Join(columnNames, ", "),
				referencedTableName,
				strings.Join(refColumnNames, ", "))

			// Add referential actions
			if constraint.UpdateRule != "" && constraint.UpdateRule != "NO ACTION" {
				stmt += fmt.Sprintf(" ON UPDATE %s", constraint.UpdateRule)
			}
			if constraint.DeleteRule != "" && constraint.DeleteRule != "NO ACTION" {
				stmt += fmt.Sprintf(" ON DELETE %s", constraint.DeleteRule)
			}

			// Add deferrable clause
			if constraint.Deferrable {
				if constraint.InitiallyDeferred {
					stmt += " DEFERRABLE INITIALLY DEFERRED"
				} else {
					stmt += " DEFERRABLE"
				}
			}

			statements = append(statements, stmt+";")

		case ir.ConstraintTypePrimaryKey:
			// Sort columns by position
			columns := sortConstraintColumnsByPosition(constraint.Columns)
			var columnNames []string
			for _, col := range columns {
				columnNames = append(columnNames, col.Name)
			}
			stmt := fmt.Sprintf("ALTER TABLE %s.%s \nADD CONSTRAINT %s PRIMARY KEY (%s);",
				td.Table.Schema, td.Table.Name, constraint.Name, strings.Join(columnNames, ", "))
			statements = append(statements, stmt)
		}
	}

	// Handle RLS changes
	for _, rlsChange := range td.RLSChanges {
		tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, td.Table.Schema)
		if rlsChange.Enabled {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY;", tableName))
		} else {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s DISABLE ROW LEVEL SECURITY;", tableName))
		}
	}

	// Drop policies
	for _, policy := range td.DroppedPolicies {
		tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, td.Table.Schema)
		statements = append(statements, fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s;", policy.Name, tableName))
	}

	// Drop triggers
	for _, trigger := range td.DroppedTriggers {
		tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, td.Table.Schema)
		statements = append(statements, fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s;", trigger.Name, tableName))
	}

	// Drop indexes
	for _, index := range td.DroppedIndexes {
		statements = append(statements, fmt.Sprintf("DROP INDEX IF EXISTS %s;", qualifyEntityName(index.Schema, index.Name, td.Table.Schema)))
	}

	// Add indexes
	for _, index := range td.AddedIndexes {
		statements = append(statements, generateIndexSQL(index, td.Table.Schema))
	}

	// Add triggers
	for _, trigger := range td.AddedTriggers {
		statements = append(statements, generateTriggerSQLWithMode(trigger, td.Table.Schema, true))
	}

	// Add policies
	for _, policy := range td.AddedPolicies {
		statements = append(statements, generatePolicySQL(policy, td.Table.Schema))
	}

	// Modify triggers
	for _, triggerDiff := range td.ModifiedTriggers {
		// Use CREATE OR REPLACE for modified triggers
		statements = append(statements, generateTriggerSQLWithMode(triggerDiff.New, td.Table.Schema, true))
	}

	// Modify policies
	for _, policyDiff := range td.ModifiedPolicies {
		// Check if this policy needs to be recreated (DROP + CREATE)
		if needsRecreate(policyDiff.Old, policyDiff.New) {
			tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, td.Table.Schema)
			// Drop and recreate policy for modification
			statements = append(statements, fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s;", policyDiff.Old.Name, tableName))
			statements = append(statements, generatePolicySQL(policyDiff.New, td.Table.Schema))
		} else {
			// Use ALTER POLICY for simple changes
			statements = append(statements, generateAlterPolicySQL(policyDiff.Old, policyDiff.New, td.Table.Schema))
		}
	}

	return statements
}

// writeColumnDefinitionToBuilder builds column definitions with SERIAL detection and proper formatting
// This is moved from ir/table.go to consolidate SQL generation in the diff module
func writeColumnDefinitionToBuilder(builder *strings.Builder, table *ir.Table, column *ir.Column, targetSchema string) {
	builder.WriteString(column.Name)
	builder.WriteString(" ")

	// Data type - handle array types and precision/scale for appropriate types
	dataType := column.DataType

	// Strip schema prefix if it matches the target schema
	dataType = stripSchemaPrefix(dataType, targetSchema)

	// Check if this is a SERIAL column (integer with nextval default)
	isSerial := isSerialColumn(column)
	if isSerial {
		// Use SERIAL, SMALLSERIAL, or BIGSERIAL based on the data type
		switch dataType {
		case "smallint":
			dataType = "SMALLSERIAL"
		case "bigint":
			dataType = "BIGSERIAL"
		default: // integer
			dataType = "SERIAL"
		}
	} else {
		// Array types are already normalized during IR construction (e.g., text[], integer[])
		if column.MaxLength != nil && (dataType == "character varying" || dataType == "varchar") {
			dataType = fmt.Sprintf("character varying(%d)", *column.MaxLength)
		} else if column.MaxLength != nil && dataType == "character" {
			dataType = fmt.Sprintf("character(%d)", *column.MaxLength)
		} else if column.Precision != nil && column.Scale != nil && (dataType == "numeric" || dataType == "decimal") {
			dataType = fmt.Sprintf("%s(%d,%d)", dataType, *column.Precision, *column.Scale)
		} else if column.Precision != nil && (dataType == "numeric" || dataType == "decimal") {
			dataType = fmt.Sprintf("%s(%d)", dataType, *column.Precision)
		}
		// For integer types like "integer", "bigint", "smallint", do not add precision/scale
	}

	builder.WriteString(dataType)

	// Identity columns
	if column.Identity != nil {
		if column.Identity.Generation == "ALWAYS" {
			builder.WriteString(" GENERATED ALWAYS AS IDENTITY")
		} else if column.Identity.Generation == "BY DEFAULT" {
			builder.WriteString(" GENERATED BY DEFAULT AS IDENTITY")
		}
	}

	// Default (include all defaults inline, but skip for SERIAL columns)
	if column.DefaultValue != nil && column.Identity == nil && !isSerial {
		defaultValue := *column.DefaultValue
		// Handle schema-agnostic sequence references in defaults
		if strings.Contains(defaultValue, "nextval") {
			// Remove schema qualifiers from sequence references in the target schema
			// Use targetSchema if provided, otherwise fall back to the table's schema
			schemaToRemove := targetSchema
			if schemaToRemove == "" {
				schemaToRemove = table.Schema
			}
			schemaPrefix := schemaToRemove + "."
			defaultValue = strings.ReplaceAll(defaultValue, schemaPrefix, "")
		}

		// Strip type qualifiers from default values
		defaultValue = stripTypeQualifiers(defaultValue)

		builder.WriteString(fmt.Sprintf(" DEFAULT %s", defaultValue))
	}

	// Nullability
	if !column.IsNullable {
		builder.WriteString(" NOT NULL")
	}

	// Add inline CHECK constraints for this column
	for _, constraint := range table.Constraints {
		if constraint.Type == ir.ConstraintTypeCheck &&
			len(constraint.Columns) == 1 &&
			constraint.Columns[0].Name == column.Name {
			// Use simpler format for inline CHECK constraints
			checkClause := constraint.CheckClause
			// Remove the "CHECK " prefix if present to get just the condition
			if after, found := strings.CutPrefix(checkClause, "CHECK "); found {
				checkClause = after
			}
			// Simplify verbose PostgreSQL CHECK expressions to developer-friendly format
			checkClause = simplifyCheckClause(checkClause)
			builder.WriteString(fmt.Sprintf(" CHECK (%s)", checkClause))
		}
	}
}

// isSerialColumn checks if a column is a SERIAL column (integer type with nextval default)
func isSerialColumn(column *ir.Column) bool {
	// Check if column has nextval default
	if column.DefaultValue == nil || !strings.Contains(*column.DefaultValue, "nextval") {
		return false
	}

	// Check if column is an integer type
	switch column.DataType {
	case "integer", "int4", "smallint", "int2", "bigint", "int8":
		return true
	default:
		return false
	}
}

// stripTypeQualifiers removes PostgreSQL type qualifiers from default values
func stripTypeQualifiers(defaultValue string) string {
	// Common PostgreSQL type qualifiers that should be stripped from default values
	typeQualifiers := []string{
		"::text",
		"::jsonb",
		"::json",
		"::numeric",
		"::decimal",
		"::integer",
		"::int",
		"::bigint",
		"::smallint",
		"::boolean",
		"::bool",
		"::character varying",
		"::varchar",
		"::character",
		"::bpchar",
		"::timestamp",
		"::timestamptz",
		"::time",
		"::timetz",
		"::date",
		"::real",
		"::double precision",
		"::bytea",
		"::uuid",
		"::inet",
		"::cidr",
		"::macaddr",
		"::xml",
	}

	for _, qualifier := range typeQualifiers {
		if strings.HasSuffix(defaultValue, qualifier) {
			return strings.TrimSuffix(defaultValue, qualifier)
		}
	}
	return defaultValue
}

// simplifyCheckClause converts verbose PostgreSQL CHECK expressions to developer-friendly format
func simplifyCheckClause(checkClause string) string {
	// Remove outer parentheses if present (may be multiple layers)
	checkClause = strings.TrimSpace(checkClause)
	for strings.HasPrefix(checkClause, "(") && strings.HasSuffix(checkClause, ")") {
		checkClause = strings.TrimSpace(checkClause[1 : len(checkClause)-1])
	}

	// Convert "column = ANY (ARRAY['val1'::text, 'val2'::text])" to "column IN('val1', 'val2')"
	if strings.Contains(checkClause, "= ANY (ARRAY[") {
		// Extract the column name and values
		parts := strings.Split(checkClause, " = ANY (ARRAY[")
		if len(parts) == 2 {
			columnName := strings.TrimSpace(parts[0])

			// Remove the closing ])))
			valuesPart := parts[1]
			if strings.HasSuffix(valuesPart, "])") {
				valuesPart = valuesPart[:len(valuesPart)-2]
			}
			if strings.HasSuffix(valuesPart, "])) ") {
				valuesPart = valuesPart[:len(valuesPart)-4]
			}
			if strings.HasSuffix(valuesPart, "]))") {
				valuesPart = valuesPart[:len(valuesPart)-3]
			}
			if strings.HasSuffix(valuesPart, "])") {
				valuesPart = valuesPart[:len(valuesPart)-2]
			}

			// Split the values and clean them up
			values := strings.Split(valuesPart, ", ")
			var cleanValues []string
			for _, val := range values {
				val = strings.TrimSpace(val)
				// Remove type casts like ::text
				if idx := strings.Index(val, "::"); idx != -1 {
					val = val[:idx]
				}
				cleanValues = append(cleanValues, val)
			}

			return fmt.Sprintf("%s IN(%s)", columnName, strings.Join(cleanValues, ", "))
		}
	}

	// Convert "column ~~ 'pattern'::text" to "column LIKE 'pattern'"
	if strings.Contains(checkClause, " ~~ ") {
		parts := strings.Split(checkClause, " ~~ ")
		if len(parts) == 2 {
			columnName := strings.TrimSpace(parts[0])
			pattern := strings.TrimSpace(parts[1])
			// Remove type cast
			if idx := strings.Index(pattern, "::"); idx != -1 {
				pattern = pattern[:idx]
			}
			return fmt.Sprintf("%s LIKE %s", columnName, pattern)
		}
	}

	// If no simplification matched, return the clause as-is
	return checkClause
}

