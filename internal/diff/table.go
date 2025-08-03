package diff

import (
	"fmt"
	"regexp"
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
func diffTables(oldTable, newTable *ir.Table) *tableDiff {
	diff := &tableDiff{
		Table:              newTable,
		AddedColumns:       []*ir.Column{},
		DroppedColumns:     []*ir.Column{},
		ModifiedColumns:    []*columnDiff{},
		AddedConstraints:   []*ir.Constraint{},
		DroppedConstraints: []*ir.Constraint{},
		AddedIndexes:       []*ir.Index{},
		DroppedIndexes:     []*ir.Index{},
		ModifiedIndexes:    []*indexDiff{},
		AddedTriggers:      []*ir.Trigger{},
		DroppedTriggers:    []*ir.Trigger{},
		ModifiedTriggers:   []*triggerDiff{},
		AddedPolicies:      []*ir.RLSPolicy{},
		DroppedPolicies:    []*ir.RLSPolicy{},
		ModifiedPolicies:   []*policyDiff{},
		RLSChanges:         []*rlsChange{},
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
				diff.ModifiedColumns = append(diff.ModifiedColumns, &columnDiff{
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

	// Find modified indexes (currently just comment changes)
	for name, newIndex := range newIndexes {
		if oldIndex, exists := oldIndexes[name]; exists {
			if oldIndex.Comment != newIndex.Comment {
				diff.ModifiedIndexes = append(diff.ModifiedIndexes, &indexDiff{
					Old: oldIndex,
					New: newIndex,
				})
			}
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
				diff.ModifiedTriggers = append(diff.ModifiedTriggers, &triggerDiff{
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
				diff.ModifiedPolicies = append(diff.ModifiedPolicies, &policyDiff{
					Old: oldPolicy,
					New: newPolicy,
				})
			}
		}
	}

	// Check for RLS enable/disable changes
	if oldTable.RLSEnabled != newTable.RLSEnabled {
		diff.RLSChanges = append(diff.RLSChanges, &rlsChange{
			Table:   newTable,
			Enabled: newTable.RLSEnabled,
		})
	}

	// Check for table comment changes
	if oldTable.Comment != newTable.Comment {
		diff.CommentChanged = true
		diff.OldComment = oldTable.Comment
		diff.NewComment = newTable.Comment
	}

	// Return nil if no changes
	if len(diff.AddedColumns) == 0 && len(diff.DroppedColumns) == 0 &&
		len(diff.ModifiedColumns) == 0 && len(diff.AddedConstraints) == 0 &&
		len(diff.DroppedConstraints) == 0 && len(diff.AddedIndexes) == 0 &&
		len(diff.DroppedIndexes) == 0 && len(diff.ModifiedIndexes) == 0 &&
		len(diff.AddedTriggers) == 0 && len(diff.DroppedTriggers) == 0 &&
		len(diff.ModifiedTriggers) == 0 && len(diff.AddedPolicies) == 0 &&
		len(diff.DroppedPolicies) == 0 && len(diff.ModifiedPolicies) == 0 &&
		len(diff.RLSChanges) == 0 && !diff.CommentChanged {
		return nil
	}

	return diff
}

// generateCreateTablesSQL generates CREATE TABLE statements with co-located indexes, constraints, triggers, and RLS
// Tables are assumed to be pre-sorted in topological order for dependency-aware creation
func generateCreateTablesSQL(tables []*ir.Table, targetSchema string, collector *diffCollector) {
	// Process tables in the provided order (already topologically sorted)
	for _, table := range tables {
		// Create the table
		sql := generateTableSQL(table, targetSchema)

		// Create context for this statement
		context := &diffContext{
			Type:                "table",
			Operation:           "create",
			Path:                fmt.Sprintf("%s.%s", table.Schema, table.Name),
			Source:              table,
			CanRunInTransaction: true, // CREATE TABLE can run in a transaction
		}

		collector.collect(context, sql)

		// Add table comment
		if table.Comment != "" {
			tableName := qualifyEntityName(table.Schema, table.Name, targetSchema)
			sql := fmt.Sprintf("COMMENT ON TABLE %s IS %s;", tableName, quoteString(table.Comment))

			// Create context for this statement
			context := &diffContext{
				Type:                "table.comment",
				Operation:           "create",
				Path:                fmt.Sprintf("%s.%s", table.Schema, table.Name),
				Source:              table,
				CanRunInTransaction: true,
			}

			collector.collect(context, sql)
		}

		// Add column comments
		for _, column := range table.Columns {
			if column.Comment != "" {
				tableName := qualifyEntityName(table.Schema, table.Name, targetSchema)
				sql := fmt.Sprintf("COMMENT ON COLUMN %s.%s IS %s;", tableName, column.Name, quoteString(column.Comment))

				// Create context for this statement
				context := &diffContext{
					Type:                "table.column.comment",
					Operation:           "create",
					Path:                fmt.Sprintf("%s.%s.%s", table.Schema, table.Name, column.Name),
					Source:              table,
					CanRunInTransaction: true,
				}

				collector.collect(context, sql)
			}
		}

		// Convert map to slice for indexes
		indexes := make([]*ir.Index, 0, len(table.Indexes))
		for _, index := range table.Indexes {
			indexes = append(indexes, index)
		}
		generateCreateIndexesSQL(indexes, targetSchema, collector)

		// Handle RLS enable changes (before creating policies) - only for diff scenarios
		if table.RLSEnabled {
			rlsChanges := []*rlsChange{{Table: table, Enabled: true}}
			generateRLSChangesSQL(rlsChanges, targetSchema, collector)
		}

		// Create policies - only for diff scenarios
		policies := make([]*ir.RLSPolicy, 0, len(table.Policies))
		for _, policy := range table.Policies {
			policies = append(policies, policy)
		}
		generateCreatePoliciesSQL(policies, targetSchema, collector)
	}
}

// generateModifyTablesSQL generates ALTER TABLE statements
func generateModifyTablesSQL(diffs []*tableDiff, targetSchema string, collector *diffCollector) {
	// Diffs are already sorted by the Diff operation
	for _, diff := range diffs {
		// Pass collector to generateAlterTableStatements to collect with proper context
		diff.generateAlterTableStatements(targetSchema, collector)

		// Handle indexes separately to properly track transaction support
		// Drop indexes
		for _, index := range diff.DroppedIndexes {
			sql := fmt.Sprintf("DROP INDEX IF EXISTS %s;", qualifyEntityName(index.Schema, index.Name, targetSchema))
			context := &diffContext{
				Type:                "table.index",
				Operation:           "drop",
				Path:                fmt.Sprintf("%s.%s.%s", index.Schema, index.Table, index.Name),
				Source:              index,
				CanRunInTransaction: true,
			}
			collector.collect(context, sql)
		}

		// Add indexes
		for _, index := range diff.AddedIndexes {
			sql := generateIndexSQL(index, targetSchema)
			context := &diffContext{
				Type:                "table.index",
				Operation:           "create",
				Path:                fmt.Sprintf("%s.%s.%s", index.Schema, index.Table, index.Name),
				Source:              index,
				CanRunInTransaction: !index.IsConcurrent, // CREATE INDEX CONCURRENTLY cannot run in a transaction
			}
			collector.collect(context, sql)
		}
	}
}

// generateDropTablesSQL generates DROP TABLE statements
// Tables are assumed to be pre-sorted in reverse topological order for dependency-aware dropping
func generateDropTablesSQL(tables []*ir.Table, targetSchema string, collector *diffCollector) {
	// Process tables in the provided order (already reverse topologically sorted)
	for _, table := range tables {
		tableName := qualifyEntityName(table.Schema, table.Name, targetSchema)
		sql := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", tableName)

		// Create context for this statement
		context := &diffContext{
			Type:                "table",
			Operation:           "drop",
			Path:                fmt.Sprintf("%s.%s", table.Schema, table.Name),
			Source:              table,
			CanRunInTransaction: true,
		}

		collector.collect(context, sql)
	}
}

// generateTableSQL generates CREATE TABLE statement
func generateTableSQL(table *ir.Table, targetSchema string) string {
	// Only include table name without schema if it's in the target schema
	tableName := qualifyEntityName(table.Schema, table.Name, targetSchema)

	var parts []string
	parts = append(parts, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (", tableName))

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
func (td *tableDiff) generateAlterTableStatements(targetSchema string, collector *diffCollector) {
	// Drop constraints first (before dropping columns) - already sorted by the Diff operation
	for _, constraint := range td.DroppedConstraints {
		tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, targetSchema)
		sql := fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;", tableName, constraint.Name)
		
		context := &diffContext{
			Type:                "table.constraint",
			Operation:           "drop",
			Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, constraint.Name),
			Source:              constraint,
			CanRunInTransaction: true,
		}
		collector.collect(context, sql)
	}

	// Drop columns - already sorted by the Diff operation
	for _, column := range td.DroppedColumns {
		tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, targetSchema)
		sql := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", tableName, column.Name)
		
		context := &diffContext{
			Type:                "table.column",
			Operation:           "drop",
			Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, column.Name),
			Source:              column,
			CanRunInTransaction: true,
		}
		collector.collect(context, sql)
	}

	// Add new columns - already sorted by the Diff operation
	// Track which constraints are handled inline with column additions
	handledFKConstraints := make(map[string]bool)
	handledPKConstraints := make(map[string]bool)
	handledUKConstraints := make(map[string]bool)

	for _, column := range td.AddedColumns {
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

		// Check if this column has an associated primary key constraint in AddedConstraints
		var pkConstraint *ir.Constraint
		for _, constraint := range td.AddedConstraints {
			if constraint.Type == ir.ConstraintTypePrimaryKey &&
				len(constraint.Columns) == 1 &&
				constraint.Columns[0].Name == column.Name {
				pkConstraint = constraint
				handledPKConstraints[constraint.Name] = true
				break
			}
		}

		// Check if this column has an associated unique constraint in AddedConstraints
		var ukConstraint *ir.Constraint
		for _, constraint := range td.AddedConstraints {
			if constraint.Type == ir.ConstraintTypeUnique &&
				len(constraint.Columns) == 1 &&
				constraint.Columns[0].Name == column.Name {
				ukConstraint = constraint
				handledUKConstraints[constraint.Name] = true
				break
			}
		}

		// Use line break format for complex statements (with foreign keys, primary keys, or unique keys)
		var stmt string
		columnType := formatColumnDataType(column)
		tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, targetSchema)
		if fkConstraint != nil || pkConstraint != nil || ukConstraint != nil {
			// Use multi-line format for complex statements with constraints
			stmt = fmt.Sprintf("ALTER TABLE %s\nADD COLUMN %s %s",
				tableName, column.Name, columnType)
		} else {
			// Use single-line format for simple column additions
			stmt = fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
				tableName, column.Name, columnType)
		}

		// Add foreign key reference inline if present
		if fkConstraint != nil {
			referencedTableName := getTableNameWithSchema(fkConstraint.ReferencedSchema, fkConstraint.ReferencedTable, targetSchema)
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
			switch column.Identity.Generation {
			case "ALWAYS":
				stmt += " GENERATED ALWAYS AS IDENTITY"
			case "BY DEFAULT":
				stmt += " GENERATED BY DEFAULT AS IDENTITY"
			}
		}

		// Don't add DEFAULT for SERIAL columns or if identity is present
		if column.DefaultValue != nil && column.Identity == nil && !isSerialColumn(column) {
			stmt += fmt.Sprintf(" DEFAULT %s", *column.DefaultValue)
		}

		// Don't add NOT NULL for identity columns or SERIAL columns as they are implicitly NOT NULL
		// Also skip NOT NULL if we're adding PRIMARY KEY inline (PRIMARY KEY implies NOT NULL)
		if !column.IsNullable && column.Identity == nil && !isSerialColumn(column) && pkConstraint == nil {
			stmt += " NOT NULL"
		}

		// Add PRIMARY KEY inline if present
		if pkConstraint != nil {
			stmt += " PRIMARY KEY"
		}

		// Add UNIQUE inline if present (and no PRIMARY KEY, since PRIMARY KEY implies UNIQUE)
		if ukConstraint != nil && pkConstraint == nil {
			stmt += " UNIQUE"
		}

		context := &diffContext{
			Type:                "table.column",
			Operation:           "create",
			Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, column.Name),
			Source:              column,
			CanRunInTransaction: true,
		}
		collector.collect(context, stmt+";")
	}

	// Modify existing columns - already sorted by the Diff operation
	for _, columnDiff := range td.ModifiedColumns {
		// Generate column modification statements and collect each with proper context
		columnStatements := columnDiff.generateColumnSQL(td.Table.Schema, td.Table.Name, targetSchema)
		for _, stmt := range columnStatements {
			context := &diffContext{
				Type:                "table.column",
				Operation:           "alter",
				Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, columnDiff.New.Name),
				Source:              columnDiff,
				CanRunInTransaction: true,
			}
			collector.collect(context, stmt)
		}
	}

	// Add new constraints - already sorted by the Diff operation
	for _, constraint := range td.AddedConstraints {
		// Skip FK constraints that were already handled inline with column additions
		if constraint.Type == ir.ConstraintTypeForeignKey && handledFKConstraints[constraint.Name] {
			continue
		}
		// Skip PK constraints that were already handled inline with column additions
		if constraint.Type == ir.ConstraintTypePrimaryKey && handledPKConstraints[constraint.Name] {
			continue
		}
		// Skip UK constraints that were already handled inline with column additions
		if constraint.Type == ir.ConstraintTypeUnique && handledUKConstraints[constraint.Name] {
			continue
		}
		switch constraint.Type {
		case ir.ConstraintTypeUnique:
			// Sort columns by position
			columns := sortConstraintColumnsByPosition(constraint.Columns)
			var columnNames []string
			for _, col := range columns {
				columnNames = append(columnNames, col.Name)
			}
			tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, targetSchema)
			sql := fmt.Sprintf("ALTER TABLE %s\nADD CONSTRAINT %s UNIQUE (%s);",
				tableName, constraint.Name, strings.Join(columnNames, ", "))
			
			context := &diffContext{
				Type:                "table.constraint",
				Operation:           "create",
				Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, constraint.Name),
				Source:              constraint,
				CanRunInTransaction: true,
			}
			collector.collect(context, sql)

		case ir.ConstraintTypeCheck:
			// CheckClause already contains "CHECK (...)" from the constraint definition
			tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, targetSchema)
			sql := fmt.Sprintf("ALTER TABLE %s\nADD CONSTRAINT %s %s;",
				tableName, constraint.Name, constraint.CheckClause)
			
			context := &diffContext{
				Type:                "table.constraint",
				Operation:           "create",
				Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, constraint.Name),
				Source:              constraint,
				CanRunInTransaction: true,
			}
			collector.collect(context, sql)

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

			tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, targetSchema)
			referencedTableName := getTableNameWithSchema(constraint.ReferencedSchema, constraint.ReferencedTable, targetSchema)
			sql := fmt.Sprintf("ALTER TABLE %s\nADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s)",
				tableName, constraint.Name,
				strings.Join(columnNames, ", "),
				referencedTableName,
				strings.Join(refColumnNames, ", "))

			// Add referential actions
			if constraint.UpdateRule != "" && constraint.UpdateRule != "NO ACTION" {
				sql += fmt.Sprintf(" ON UPDATE %s", constraint.UpdateRule)
			}
			if constraint.DeleteRule != "" && constraint.DeleteRule != "NO ACTION" {
				sql += fmt.Sprintf(" ON DELETE %s", constraint.DeleteRule)
			}

			// Add deferrable clause
			if constraint.Deferrable {
				if constraint.InitiallyDeferred {
					sql += " DEFERRABLE INITIALLY DEFERRED"
				} else {
					sql += " DEFERRABLE"
				}
			}

			sql += ";"
			
			context := &diffContext{
				Type:                "table.constraint",
				Operation:           "create",
				Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, constraint.Name),
				Source:              constraint,
				CanRunInTransaction: true,
			}
			collector.collect(context, sql)

		case ir.ConstraintTypePrimaryKey:
			// Sort columns by position
			columns := sortConstraintColumnsByPosition(constraint.Columns)
			var columnNames []string
			for _, col := range columns {
				columnNames = append(columnNames, col.Name)
			}
			tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, targetSchema)
			sql := fmt.Sprintf("ALTER TABLE %s\nADD CONSTRAINT %s PRIMARY KEY (%s);",
				tableName, constraint.Name, strings.Join(columnNames, ", "))
			
			context := &diffContext{
				Type:                "table.constraint",
				Operation:           "create",
				Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, constraint.Name),
				Source:              constraint,
				CanRunInTransaction: true,
			}
			collector.collect(context, sql)
		}
	}

	// Handle RLS changes
	for _, rlsChange := range td.RLSChanges {
		tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, targetSchema)
		var sql string
		if rlsChange.Enabled {
			sql = fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY;", tableName)
		} else {
			sql = fmt.Sprintf("ALTER TABLE %s DISABLE ROW LEVEL SECURITY;", tableName)
		}
		
		context := &diffContext{
			Type:                "table.rls",
			Operation:           "alter",
			Path:                fmt.Sprintf("%s.%s", td.Table.Schema, td.Table.Name),
			Source:              rlsChange,
			CanRunInTransaction: true,
		}
		collector.collect(context, sql)
	}

	// Drop policies - already sorted by the Diff operation
	for _, policy := range td.DroppedPolicies {
		tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, targetSchema)
		sql := fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s;", policy.Name, tableName)
		
		context := &diffContext{
			Type:                "table.policy",
			Operation:           "drop",
			Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, policy.Name),
			Source:              policy,
			CanRunInTransaction: true,
		}
		collector.collect(context, sql)
	}

	// Drop triggers - already sorted by the Diff operation
	for _, trigger := range td.DroppedTriggers {
		tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, targetSchema)
		sql := fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s;", trigger.Name, tableName)
		
		context := &diffContext{
			Type:                "table.trigger",
			Operation:           "drop",
			Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, trigger.Name),
			Source:              trigger,
			CanRunInTransaction: true,
		}
		collector.collect(context, sql)
	}

	// Note: Indexes are handled separately in generateModifyTablesSQL to properly track transaction support

	// Add triggers - already sorted by the Diff operation
	for _, trigger := range td.AddedTriggers {
		sql := generateTriggerSQLWithMode(trigger, targetSchema)
		
		context := &diffContext{
			Type:                "table.trigger",
			Operation:           "create",
			Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, trigger.Name),
			Source:              trigger,
			CanRunInTransaction: true,
		}
		collector.collect(context, sql)
	}

	// Add policies - already sorted by the Diff operation
	for _, policy := range td.AddedPolicies {
		sql := generatePolicySQL(policy, targetSchema)
		
		context := &diffContext{
			Type:                "table.policy",
			Operation:           "create",
			Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, policy.Name),
			Source:              policy,
			CanRunInTransaction: true,
		}
		collector.collect(context, sql)
	}

	// Modify triggers - already sorted by the Diff operation
	for _, triggerDiff := range td.ModifiedTriggers {
		// Use CREATE OR REPLACE for modified triggers
		sql := generateTriggerSQLWithMode(triggerDiff.New, targetSchema)
		
		context := &diffContext{
			Type:                "table.trigger",
			Operation:           "alter",
			Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, triggerDiff.New.Name),
			Source:              triggerDiff,
			CanRunInTransaction: true,
		}
		collector.collect(context, sql)
	}

	// Modify policies - already sorted by the Diff operation
	for _, policyDiff := range td.ModifiedPolicies {
		// Check if this policy needs to be recreated (DROP + CREATE)
		if needsRecreate(policyDiff.Old, policyDiff.New) {
			tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, targetSchema)
			// Drop and recreate policy for modification
			sql := fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s;", policyDiff.Old.Name, tableName)
			
			context := &diffContext{
				Type:                "table.policy",
				Operation:           "drop",
				Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, policyDiff.Old.Name),
				Source:              policyDiff,
				CanRunInTransaction: true,
			}
			collector.collect(context, sql)
			
			sql = generatePolicySQL(policyDiff.New, targetSchema)
			context = &diffContext{
				Type:                "table.policy",
				Operation:           "create",
				Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, policyDiff.New.Name),
				Source:              policyDiff,
				CanRunInTransaction: true,
			}
			collector.collect(context, sql)
		} else {
			// Use ALTER POLICY for simple changes
			sql := generateAlterPolicySQL(policyDiff.Old, policyDiff.New, targetSchema)
			
			context := &diffContext{
				Type:                "table.policy",
				Operation:           "alter",
				Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, policyDiff.New.Name),
				Source:              policyDiff,
				CanRunInTransaction: true,
			}
			collector.collect(context, sql)
		}
	}

	// Handle table comment changes
	if td.CommentChanged {
		tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, targetSchema)
		var sql string
		if td.NewComment == "" {
			sql = fmt.Sprintf("COMMENT ON TABLE %s IS NULL;", tableName)
		} else {
			sql = fmt.Sprintf("COMMENT ON TABLE %s IS %s;", tableName, quoteString(td.NewComment))
		}
		
		context := &diffContext{
			Type:                "table.comment",
			Operation:           "alter",
			Path:                fmt.Sprintf("%s.%s", td.Table.Schema, td.Table.Name),
			Source:              td,
			CanRunInTransaction: true,
		}
		collector.collect(context, sql)
	}

	// Handle column comment changes
	for _, colDiff := range td.ModifiedColumns {
		if colDiff.Old.Comment != colDiff.New.Comment {
			tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, targetSchema)
			var sql string
			if colDiff.New.Comment == "" {
				sql = fmt.Sprintf("COMMENT ON COLUMN %s.%s IS NULL;", tableName, colDiff.New.Name)
			} else {
				sql = fmt.Sprintf("COMMENT ON COLUMN %s.%s IS %s;", tableName, colDiff.New.Name, quoteString(colDiff.New.Comment))
			}
			
			context := &diffContext{
				Type:                "table.column.comment",
				Operation:           "alter",
				Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, colDiff.New.Name),
				Source:              colDiff,
				CanRunInTransaction: true,
			}
			collector.collect(context, sql)
		}
	}

	// Handle index comment changes
	for _, indexDiff := range td.ModifiedIndexes {
		if indexDiff.Old.Comment != indexDiff.New.Comment {
			indexName := qualifyEntityName(indexDiff.New.Schema, indexDiff.New.Name, targetSchema)
			var sql string
			if indexDiff.New.Comment == "" {
				sql = fmt.Sprintf("COMMENT ON INDEX %s IS NULL;", indexName)
			} else {
				sql = fmt.Sprintf("COMMENT ON INDEX %s IS %s;", indexName, quoteString(indexDiff.New.Comment))
			}
			
			context := &diffContext{
				Type:                "table.index.comment",
				Operation:           "alter",
				Path:                fmt.Sprintf("%s.%s.%s", indexDiff.New.Schema, indexDiff.New.Table, indexDiff.New.Name),
				Source:              indexDiff,
				CanRunInTransaction: true,
			}
			collector.collect(context, sql)
		}
	}
}

// writeColumnDefinitionToBuilder builds column definitions with SERIAL detection and proper formatting
// This is moved from ir/table.go to consolidate SQL generation in the diff module
func writeColumnDefinitionToBuilder(builder *strings.Builder, table *ir.Table, column *ir.Column, targetSchema string) {
	builder.WriteString(column.Name)
	builder.WriteString(" ")

	// Data type - handle array types and precision/scale for appropriate types
	dataType := formatColumnDataTypeForCreate(column)

	// Strip schema prefix if it matches the target schema
	dataType = stripSchemaPrefix(dataType, targetSchema)

	builder.WriteString(dataType)

	// Check if this column is part of a single-column primary key (for inlining PRIMARY KEY)
	var isSingleColumnPrimaryKey bool
	// Check if this column is part of any primary key (for skipping NOT NULL)
	var isPartOfPrimaryKey bool

	for _, constraint := range table.Constraints {
		if constraint.Type == ir.ConstraintTypePrimaryKey {
			// Check if this column is in this primary key constraint
			for _, col := range constraint.Columns {
				if col.Name == column.Name {
					isPartOfPrimaryKey = true
					// Also check if it's a single-column primary key
					if len(constraint.Columns) == 1 {
						isSingleColumnPrimaryKey = true
					}
					break
				}
			}
		}
		if isPartOfPrimaryKey {
			break
		}
	}

	// Add PRIMARY KEY inline for single-column primary keys
	if isSingleColumnPrimaryKey {
		builder.WriteString(" PRIMARY KEY")
	}

	// Identity columns
	if column.Identity != nil {
		switch column.Identity.Generation {
		case "ALWAYS":
			builder.WriteString(" GENERATED ALWAYS AS IDENTITY")
		case "BY DEFAULT":
			builder.WriteString(" GENERATED BY DEFAULT AS IDENTITY")
		}
	}

	// Default (include all defaults inline, but skip for SERIAL columns)
	if column.DefaultValue != nil && column.Identity == nil && !isSerialColumn(column) {
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

	// Nullability - skip NOT NULL for columns that are part of any primary key since PRIMARY KEY implies NOT NULL
	if !column.IsNullable && !isPartOfPrimaryKey {
		builder.WriteString(" NOT NULL")
	}

	// Add inline UNIQUE constraint for single-column unique constraints
	for _, constraint := range table.Constraints {
		if constraint.Type == ir.ConstraintTypeUnique &&
			len(constraint.Columns) == 1 &&
			constraint.Columns[0].Name == column.Name {
			builder.WriteString(" UNIQUE")
			break
		}
	}

	// Add inline FOREIGN KEY (REFERENCES) for single-column foreign keys
	for _, constraint := range table.Constraints {
		if constraint.Type == ir.ConstraintTypeForeignKey &&
			len(constraint.Columns) == 1 &&
			constraint.Columns[0].Name == column.Name {
			referencedTableName := getTableNameWithSchema(constraint.ReferencedSchema, constraint.ReferencedTable, targetSchema)
			builder.WriteString(fmt.Sprintf(" REFERENCES %s", referencedTableName))

			if len(constraint.ReferencedColumns) > 0 {
				builder.WriteString(fmt.Sprintf("(%s)", constraint.ReferencedColumns[0].Name))
			}

			// Add ON DELETE/UPDATE actions if specified
			if constraint.DeleteRule != "" && constraint.DeleteRule != "NO ACTION" {
				builder.WriteString(fmt.Sprintf(" ON DELETE %s", constraint.DeleteRule))
			}
			if constraint.UpdateRule != "" && constraint.UpdateRule != "NO ACTION" {
				builder.WriteString(fmt.Sprintf(" ON UPDATE %s", constraint.UpdateRule))
			}

			// Add deferrable options
			if constraint.Deferrable {
				builder.WriteString(" DEFERRABLE")
				if constraint.InitiallyDeferred {
					builder.WriteString(" INITIALLY DEFERRED")
				}
			}
			break
		}
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

// formatColumnDataType formats a column's data type with appropriate modifiers for ALTER TABLE statements
func formatColumnDataType(column *ir.Column) string {
	dataType := column.DataType

	// Handle SERIAL types
	if isSerialColumn(column) {
		switch column.DataType {
		case "smallint", "int2":
			return "smallserial"
		case "bigint", "int8":
			return "bigserial"
		default:
			return "serial"
		}
	}

	// Keep terse forms like timestamptz as preferred

	// Add precision/scale/length modifiers
	if column.MaxLength != nil && (dataType == "varchar" || dataType == "character varying") {
		return fmt.Sprintf("varchar(%d)", *column.MaxLength)
	} else if column.MaxLength != nil && dataType == "character" {
		return fmt.Sprintf("character(%d)", *column.MaxLength)
	} else if column.Precision != nil && column.Scale != nil && (dataType == "numeric" || dataType == "decimal") {
		return fmt.Sprintf("%s(%d,%d)", dataType, *column.Precision, *column.Scale)
	} else if column.Precision != nil && (dataType == "numeric" || dataType == "decimal") {
		return fmt.Sprintf("%s(%d)", dataType, *column.Precision)
	}

	return dataType
}

// formatColumnDataTypeForCreate formats a column's data type with appropriate modifiers for CREATE TABLE statements
func formatColumnDataTypeForCreate(column *ir.Column) string {
	dataType := column.DataType

	// Handle SERIAL types (uppercase for CREATE TABLE)
	if isSerialColumn(column) {
		switch column.DataType {
		case "smallint", "int2":
			return "SMALLSERIAL"
		case "bigint", "int8":
			return "BIGSERIAL"
		default:
			return "SERIAL"
		}
	}

	// Keep timestamptz as-is for CREATE TABLE (don't convert to verbose form)

	// Add precision/scale/length modifiers
	if column.MaxLength != nil && (dataType == "varchar" || dataType == "character varying") {
		return fmt.Sprintf("varchar(%d)", *column.MaxLength)
	} else if column.MaxLength != nil && dataType == "character" {
		return fmt.Sprintf("character(%d)", *column.MaxLength)
	} else if column.Precision != nil && column.Scale != nil && (dataType == "numeric" || dataType == "decimal") {
		return fmt.Sprintf("%s(%d,%d)", dataType, *column.Precision, *column.Scale)
	} else if column.Precision != nil && (dataType == "numeric" || dataType == "decimal") {
		return fmt.Sprintf("%s(%d)", dataType, *column.Precision)
	}

	return dataType
}

// stripTypeQualifiers removes PostgreSQL type qualifiers from default values
func stripTypeQualifiers(defaultValue string) string {
	// Use regex to match any type qualifier pattern (::typename)
	// This handles both built-in types and user-defined types like enums
	re := regexp.MustCompile(`(.*)::[a-zA-Z_][a-zA-Z0-9_\s]*(\[\])?$`)
	matches := re.FindStringSubmatch(defaultValue)
	if len(matches) > 1 {
		return matches[1]
	}
	return defaultValue
}

// simplifyCheckClause converts verbose PostgreSQL CHECK expressions to developer-friendly format
func simplifyCheckClause(checkClause string) string {
	// Remove outer parentheses if present (may be multiple layers)
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
			valuesPart = strings.TrimSuffix(valuesPart, "])")
			valuesPart = strings.TrimSuffix(valuesPart, "])) ")
			valuesPart = strings.TrimSuffix(valuesPart, "]))")
			valuesPart = strings.TrimSuffix(valuesPart, "])")

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
