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
		Table:               newTable,
		AddedColumns:        []*ir.Column{},
		DroppedColumns:      []*ir.Column{},
		ModifiedColumns:     []*columnDiff{},
		AddedConstraints:    []*ir.Constraint{},
		DroppedConstraints:  []*ir.Constraint{},
		ModifiedConstraints: []*constraintDiff{},
		AddedIndexes:        []*ir.Index{},
		DroppedIndexes:      []*ir.Index{},
		ModifiedIndexes:     []*indexDiff{},
		AddedTriggers:       []*ir.Trigger{},
		DroppedTriggers:     []*ir.Trigger{},
		ModifiedTriggers:    []*triggerDiff{},
		AddedPolicies:       []*ir.RLSPolicy{},
		DroppedPolicies:     []*ir.RLSPolicy{},
		ModifiedPolicies:    []*policyDiff{},
		RLSChanges:          []*rlsChange{},
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

	// Find modified constraints
	for name, newConstraint := range newConstraints {
		if oldConstraint, exists := oldConstraints[name]; exists {
			if !constraintsEqual(oldConstraint, newConstraint) {
				diff.ModifiedConstraints = append(diff.ModifiedConstraints, &constraintDiff{
					Old: oldConstraint,
					New: newConstraint,
				})
			}
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

	// Find modified indexes (comment changes and structural changes)
	for name, newIndex := range newIndexes {
		if oldIndex, exists := oldIndexes[name]; exists {
			structurallyEqual := indexesStructurallyEqual(oldIndex, newIndex)
			commentChanged := oldIndex.Comment != newIndex.Comment

			// If only comments changed, treat as modification
			if structurallyEqual && commentChanged {
				diff.ModifiedIndexes = append(diff.ModifiedIndexes, &indexDiff{
					Old: oldIndex,
					New: newIndex,
				})
			} else if !structurallyEqual {
				// If structure changed, treat as drop + add for proper online handling
				diff.DroppedIndexes = append(diff.DroppedIndexes, oldIndex)
				diff.AddedIndexes = append(diff.AddedIndexes, newIndex)
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
		len(diff.DroppedConstraints) == 0 && len(diff.ModifiedConstraints) == 0 &&
		len(diff.AddedIndexes) == 0 && len(diff.DroppedIndexes) == 0 &&
		len(diff.ModifiedIndexes) == 0 && len(diff.AddedTriggers) == 0 &&
		len(diff.DroppedTriggers) == 0 && len(diff.ModifiedTriggers) == 0 &&
		len(diff.AddedPolicies) == 0 && len(diff.DroppedPolicies) == 0 &&
		len(diff.ModifiedPolicies) == 0 && len(diff.RLSChanges) == 0 &&
		!diff.CommentChanged {
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
			Type:                DiffTypeTable,
			Operation:           DiffOperationCreate,
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
				Type:                DiffTypeTableComment,
				Operation:           DiffOperationCreate,
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
					Type:                DiffTypeTableColumnComment,
					Operation:           DiffOperationCreate,
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
			Type:                DiffTypeTable,
			Operation:           DiffOperationDrop,
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
			Type:                DiffTypeTableConstraint,
			Operation:           DiffOperationDrop,
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
			Type:                DiffTypeTableColumn,
			Operation:           DiffOperationDrop,
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
			Type:                DiffTypeTableColumn,
			Operation:           DiffOperationCreate,
			Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, column.Name),
			Source:              column,
			CanRunInTransaction: true,
		}
		collector.collect(context, stmt+";")
	}

	// Modify existing columns - already sorted by the Diff operation
	for _, columnDiff := range td.ModifiedColumns {
		// Generate column modification statements and collect as a single step
		columnResults := columnDiff.generateColumnSQL(td.Table.Schema, td.Table.Name, targetSchema)
		// Emit separate diffs for each column operation
		for _, result := range columnResults {
			context := &diffContext{
				Type:                DiffTypeTableColumn,
				Operation:           DiffOperationAlter,
				Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, columnDiff.New.Name),
				Source:              columnDiff,
				CanRunInTransaction: true,
			}

			// Handle each statement in the result
			for _, stmt := range result.Statements {
				if result.Rewrite != nil {
					// Use collectWithRewrite for statements that have online variants
					collector.collectWithRewrite(context, stmt, result.Rewrite)
				} else {
					// Use regular collect for statements without rewrites
					collector.collect(context, stmt)
				}
			}
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
				Type:                DiffTypeTableConstraint,
				Operation:           DiffOperationCreate,
				Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, constraint.Name),
				Source:              constraint,
				CanRunInTransaction: true,
			}
			collector.collect(context, sql)

		case ir.ConstraintTypeCheck:
			// CheckClause already contains "CHECK (...)" from the constraint definition
			tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, targetSchema)
			canonicalSQL := fmt.Sprintf("ALTER TABLE %s\nADD CONSTRAINT %s %s;",
				tableName, constraint.Name, constraint.CheckClause)

			// Generate rewrite for online operations
			notValidSQL := fmt.Sprintf("ALTER TABLE %s\nADD CONSTRAINT %s %s NOT VALID;",
				tableName, constraint.Name, constraint.CheckClause)
			validateSQL := fmt.Sprintf("ALTER TABLE %s VALIDATE CONSTRAINT %s;",
				tableName, constraint.Name)

			rewrite := &DiffRewrite{
				Statements: []RewriteStatement{
					{
						SQL:                 notValidSQL,
						CanRunInTransaction: true,
					},
					{
						SQL:                 validateSQL,
						CanRunInTransaction: true,
					},
				},
			}

			context := &diffContext{
				Type:                DiffTypeTableConstraint,
				Operation:           DiffOperationCreate,
				Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, constraint.Name),
				Source:              constraint,
				CanRunInTransaction: true,
			}
			collector.collectWithRewrite(context, canonicalSQL, rewrite)

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
			canonicalSQL := fmt.Sprintf("ALTER TABLE %s\nADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s)",
				tableName, constraint.Name,
				strings.Join(columnNames, ", "),
				referencedTableName,
				strings.Join(refColumnNames, ", "))

			// Add referential actions
			if constraint.UpdateRule != "" && constraint.UpdateRule != "NO ACTION" {
				canonicalSQL += fmt.Sprintf(" ON UPDATE %s", constraint.UpdateRule)
			}
			if constraint.DeleteRule != "" && constraint.DeleteRule != "NO ACTION" {
				canonicalSQL += fmt.Sprintf(" ON DELETE %s", constraint.DeleteRule)
			}

			// Add deferrable clause
			if constraint.Deferrable {
				if constraint.InitiallyDeferred {
					canonicalSQL += " DEFERRABLE INITIALLY DEFERRED"
				} else {
					canonicalSQL += " DEFERRABLE"
				}
			}

			canonicalSQL += ";"

			// Generate rewrite for online operations
			notValidSQL := canonicalSQL[:len(canonicalSQL)-1] + " NOT VALID;"
			validateSQL := fmt.Sprintf("ALTER TABLE %s VALIDATE CONSTRAINT %s;",
				tableName, constraint.Name)

			rewrite := &DiffRewrite{
				Statements: []RewriteStatement{
					{
						SQL:                 notValidSQL,
						CanRunInTransaction: true,
					},
					{
						SQL:                 validateSQL,
						CanRunInTransaction: true,
					},
				},
			}

			context := &diffContext{
				Type:                DiffTypeTableConstraint,
				Operation:           DiffOperationCreate,
				Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, constraint.Name),
				Source:              constraint,
				CanRunInTransaction: true,
			}
			collector.collectWithRewrite(context, canonicalSQL, rewrite)

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
				Type:                DiffTypeTableConstraint,
				Operation:           DiffOperationCreate,
				Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, constraint.Name),
				Source:              constraint,
				CanRunInTransaction: true,
			}
			collector.collect(context, sql)
		}
	}

	// Handle modified constraints - drop and recreate them as separate operations
	for _, constraintDiff := range td.ModifiedConstraints {
		tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, targetSchema)
		constraint := constraintDiff.New

		// Step 1: Drop the old constraint
		dropSQL := fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;", tableName, constraintDiff.Old.Name)
		dropContext := &diffContext{
			Type:                DiffTypeTableConstraint,
			Operation:           DiffOperationDrop,
			Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, constraintDiff.Old.Name),
			Source:              constraintDiff.Old,
			CanRunInTransaction: true,
		}
		collector.collect(dropContext, dropSQL)

		// Step 2: Add new constraint
		var addSQL string
		switch constraint.Type {
		case ir.ConstraintTypeUnique:
			// Sort columns by position
			columns := sortConstraintColumnsByPosition(constraint.Columns)
			var columnNames []string
			for _, col := range columns {
				columnNames = append(columnNames, col.Name)
			}
			addSQL = fmt.Sprintf("ALTER TABLE %s\nADD CONSTRAINT %s UNIQUE (%s);",
				tableName, constraint.Name, strings.Join(columnNames, ", "))

		case ir.ConstraintTypeCheck:
			// Add CHECK constraint
			addSQL = fmt.Sprintf("ALTER TABLE %s\nADD CONSTRAINT %s %s;",
				tableName, constraint.Name, constraint.CheckClause)

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

			referencedTableName := getTableNameWithSchema(constraint.ReferencedSchema, constraint.ReferencedTable, targetSchema)
			addSQL = fmt.Sprintf("ALTER TABLE %s\nADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s)",
				tableName, constraint.Name,
				strings.Join(columnNames, ", "),
				referencedTableName,
				strings.Join(refColumnNames, ", "))

			// Add referential actions
			if constraint.UpdateRule != "" && constraint.UpdateRule != "NO ACTION" {
				addSQL += fmt.Sprintf(" ON UPDATE %s", constraint.UpdateRule)
			}
			if constraint.DeleteRule != "" && constraint.DeleteRule != "NO ACTION" {
				addSQL += fmt.Sprintf(" ON DELETE %s", constraint.DeleteRule)
			}

			// Add deferrable clause
			if constraint.Deferrable {
				if constraint.InitiallyDeferred {
					addSQL += " DEFERRABLE INITIALLY DEFERRED"
				} else {
					addSQL += " DEFERRABLE"
				}
			}

			addSQL += ";"

		case ir.ConstraintTypePrimaryKey:
			// Sort columns by position
			columns := sortConstraintColumnsByPosition(constraint.Columns)
			var columnNames []string
			for _, col := range columns {
				columnNames = append(columnNames, col.Name)
			}
			addSQL = fmt.Sprintf("ALTER TABLE %s\nADD CONSTRAINT %s PRIMARY KEY (%s);",
				tableName, constraint.Name, strings.Join(columnNames, ", "))
		}

		addContext := &diffContext{
			Type:                DiffTypeTableConstraint,
			Operation:           DiffOperationCreate,
			Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, constraint.Name),
			Source:              constraint,
			CanRunInTransaction: true,
		}

		// Generate rewrite for CHECK and FOREIGN KEY constraints
		var rewrite *DiffRewrite
		if constraint.Type == ir.ConstraintTypeCheck || constraint.Type == ir.ConstraintTypeForeignKey {
			var notValidSQL, validateSQL string

			switch constraint.Type {
			case ir.ConstraintTypeCheck:
				notValidSQL = fmt.Sprintf("ALTER TABLE %s\nADD CONSTRAINT %s %s NOT VALID;",
					tableName, constraint.Name, constraint.CheckClause)
				validateSQL = fmt.Sprintf("ALTER TABLE %s VALIDATE CONSTRAINT %s;",
					tableName, constraint.Name)

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

				referencedTableName := getTableNameWithSchema(constraint.ReferencedSchema, constraint.ReferencedTable, targetSchema)
				notValidSQL = fmt.Sprintf("ALTER TABLE %s\nADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s)",
					tableName, constraint.Name,
					strings.Join(columnNames, ", "),
					referencedTableName,
					strings.Join(refColumnNames, ", "))

				// Add referential actions
				if constraint.UpdateRule != "" && constraint.UpdateRule != "NO ACTION" {
					notValidSQL += fmt.Sprintf(" ON UPDATE %s", constraint.UpdateRule)
				}
				if constraint.DeleteRule != "" && constraint.DeleteRule != "NO ACTION" {
					notValidSQL += fmt.Sprintf(" ON DELETE %s", constraint.DeleteRule)
				}

				// Add deferrable clause
				if constraint.Deferrable {
					if constraint.InitiallyDeferred {
						notValidSQL += " DEFERRABLE INITIALLY DEFERRED"
					} else {
						notValidSQL += " DEFERRABLE"
					}
				}

				notValidSQL += " NOT VALID;"
				validateSQL = fmt.Sprintf("ALTER TABLE %s VALIDATE CONSTRAINT %s;",
					tableName, constraint.Name)
			}

			rewrite = &DiffRewrite{
				Statements: []RewriteStatement{
					{
						SQL:                 notValidSQL,
						CanRunInTransaction: true,
					},
					{
						SQL:                 validateSQL,
						CanRunInTransaction: true,
					},
				},
			}
		}

		if rewrite != nil {
			collector.collectWithRewrite(addContext, addSQL, rewrite)
		} else {
			collector.collect(addContext, addSQL)
		}
	}

	// Handle RLS changes
	for _, rlsChange := range td.RLSChanges {
		tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, targetSchema)
		var sql string
		var operation DiffOperation
		if rlsChange.Enabled {
			sql = fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY;", tableName)
			operation = DiffOperationCreate
		} else {
			sql = fmt.Sprintf("ALTER TABLE %s DISABLE ROW LEVEL SECURITY;", tableName)
			operation = DiffOperationDrop
		}

		context := &diffContext{
			Type:                DiffTypeTableRLS,
			Operation:           operation,
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
			Type:                DiffTypeTablePolicy,
			Operation:           DiffOperationDrop,
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
			Type:                DiffTypeTableTrigger,
			Operation:           DiffOperationDrop,
			Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, trigger.Name),
			Source:              trigger,
			CanRunInTransaction: true,
		}
		collector.collect(context, sql)
	}

	// Handle online index replacement for CONCURRENT indexes
	// First, identify indexes that need online replacement (dropped and added with same name and CONCURRENT)
	onlineReplacements := make(map[string]*ir.Index)
	regularDrops := []*ir.Index{}

	for _, droppedIndex := range td.DroppedIndexes {
		foundReplacement := false
		for _, addedIndex := range td.AddedIndexes {
			if droppedIndex.Name == addedIndex.Name {
				onlineReplacements[droppedIndex.Name] = addedIndex
				foundReplacement = true
				break
			}
		}
		if !foundReplacement {
			regularDrops = append(regularDrops, droppedIndex)
		}
	}

	// Regular index drops (not part of online replacement)
	for _, index := range regularDrops {
		sql := fmt.Sprintf("DROP INDEX IF EXISTS %s;", qualifyEntityName(index.Schema, index.Name, targetSchema))
		context := &diffContext{
			Type:                DiffTypeTableIndex,
			Operation:           DiffOperationDrop,
			Path:                fmt.Sprintf("%s.%s.%s", index.Schema, index.Table, index.Name),
			Source:              index,
			CanRunInTransaction: true,
		}
		collector.collect(context, sql)
	}

	// Add triggers - already sorted by the Diff operation
	for _, trigger := range td.AddedTriggers {
		sql := generateTriggerSQLWithMode(trigger, targetSchema)

		context := &diffContext{
			Type:                DiffTypeTableTrigger,
			Operation:           DiffOperationCreate,
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
			Type:                DiffTypeTablePolicy,
			Operation:           DiffOperationCreate,
			Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, policy.Name),
			Source:              policy,
			CanRunInTransaction: true,
		}
		collector.collect(context, sql)
	}

	// Process index replacements with improved concurrent approach for zero-downtime
	// Sort indexes by name for deterministic output
	var sortedOnlineIndexNames []string
	for indexName := range onlineReplacements {
		sortedOnlineIndexNames = append(sortedOnlineIndexNames, indexName)
	}
	sort.Strings(sortedOnlineIndexNames)

	for _, indexName := range sortedOnlineIndexNames {
		newIndex := onlineReplacements[indexName]
		tempIndexName := generateTempIndexName(indexName)

		// Step 1: DROP old index, Step 2: CREATE new index (canonical approach - has downtime)
		dropSQL := fmt.Sprintf("DROP INDEX IF EXISTS %s;", qualifyEntityName(newIndex.Schema, indexName, targetSchema))
		canonicalSQL := generateIndexSQL(newIndex, targetSchema, false) // Regular CREATE INDEX

		// Create statements for the canonical approach (DROP + CREATE - with downtime)
		statements := []SQLStatement{
			{
				SQL:                 dropSQL,
				CanRunInTransaction: true,
			},
			{
				SQL:                 canonicalSQL,
				CanRunInTransaction: true,
			},
		}

		// Improved concurrent approach for zero-downtime:
		// Step 1: CREATE new index with temp name
		// Step 2: DROP old index
		// Step 3: RENAME new index to final name
		concurrentSQL := generateIndexSQLWithName(newIndex, tempIndexName, targetSchema, true) // CREATE CONCURRENTLY with temp name
		waitSQL := generateIndexWaitQueryWithName(tempIndexName)
		dropOldSQL := fmt.Sprintf("DROP INDEX %s;", qualifyEntityName(newIndex.Schema, indexName, targetSchema)) // No IF EXISTS for safety
		renameSQL := generateIndexRenameSQL(tempIndexName, indexName, targetSchema)

		rewrite := &DiffRewrite{
			Statements: []RewriteStatement{
				{
					SQL:                 concurrentSQL,
					CanRunInTransaction: false, // CONCURRENTLY cannot run in transaction
				},
				{
					SQL:                 waitSQL,
					CanRunInTransaction: true,
					Directive: &Directive{
						Type:    "wait",
						Message: fmt.Sprintf("Creating index %s", tempIndexName),
					},
				},
				{
					SQL:                 dropOldSQL,
					CanRunInTransaction: true,
				},
				{
					SQL:                 renameSQL,
					CanRunInTransaction: true,
				},
			},
		}

		alterContext := &diffContext{
			Type:                DiffTypeTableIndex,
			Operation:           DiffOperationAlter,
			Path:                fmt.Sprintf("%s.%s.%s", newIndex.Schema, newIndex.Table, indexName),
			Source:              newIndex,
			CanRunInTransaction: true,
		}
		collector.collectMultipleStatements(alterContext, statements, rewrite)

		// Add index comment if present as a separate operation
		if newIndex.Comment != "" {
			qualifiedIndexName := qualifyEntityName(newIndex.Schema, indexName, targetSchema)
			sql := fmt.Sprintf("COMMENT ON INDEX %s IS %s;", qualifiedIndexName, quoteString(newIndex.Comment))

			context := &diffContext{
				Type:                DiffTypeTableIndexComment,
				Operation:           DiffOperationCreate,
				Path:                fmt.Sprintf("%s.%s.%s", newIndex.Schema, newIndex.Table, indexName),
				Source:              newIndex,
				CanRunInTransaction: true,
			}
			collector.collect(context, sql)
		}
	}

	// Add regular indexes (not part of online replacement)
	regularAdds := []*ir.Index{}
	for _, index := range td.AddedIndexes {
		if _, isReplacement := onlineReplacements[index.Name]; !isReplacement {
			regularAdds = append(regularAdds, index)
		}
	}

	for _, index := range regularAdds {
		canonicalSQL := generateIndexSQL(index, targetSchema, false) // Always generate canonical form

		// Generate rewrite for online operations
		concurrentSQL := generateIndexSQL(index, targetSchema, true) // With CONCURRENTLY
		waitSQL := generateIndexWaitQueryWithName(index.Name)

		rewrite := &DiffRewrite{
			Statements: []RewriteStatement{
				{
					SQL:                 concurrentSQL,
					CanRunInTransaction: false, // CONCURRENTLY cannot run in transaction
				},
				{
					SQL:                 waitSQL,
					CanRunInTransaction: true,
					Directive: &Directive{
						Type:    "wait",
						Message: fmt.Sprintf("Creating index %s", index.Name),
					},
				},
			},
		}

		context := &diffContext{
			Type:                DiffTypeTableIndex,
			Operation:           DiffOperationCreate,
			Path:                fmt.Sprintf("%s.%s.%s", index.Schema, index.Table, index.Name),
			Source:              index,
			CanRunInTransaction: true,
		}

		collector.collectWithRewrite(context, canonicalSQL, rewrite)

		// Add index comment if present
		if index.Comment != "" {
			indexName := qualifyEntityName(index.Schema, index.Name, targetSchema)
			sql := fmt.Sprintf("COMMENT ON INDEX %s IS %s;", indexName, quoteString(index.Comment))

			context := &diffContext{
				Type:                DiffTypeTableIndexComment,
				Operation:           DiffOperationCreate,
				Path:                fmt.Sprintf("%s.%s.%s", index.Schema, index.Table, index.Name),
				Source:              index,
				CanRunInTransaction: true,
			}
			collector.collect(context, sql)
		}
	}

	// Modify triggers - already sorted by the Diff operation
	for _, triggerDiff := range td.ModifiedTriggers {
		// Use CREATE OR REPLACE for modified triggers
		sql := generateTriggerSQLWithMode(triggerDiff.New, targetSchema)

		context := &diffContext{
			Type:                DiffTypeTableTrigger,
			Operation:           DiffOperationAlter,
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
				Type:                DiffTypeTablePolicy,
				Operation:           DiffOperationDrop,
				Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, policyDiff.Old.Name),
				Source:              policyDiff,
				CanRunInTransaction: true,
			}
			collector.collect(context, sql)

			sql = generatePolicySQL(policyDiff.New, targetSchema)
			context = &diffContext{
				Type:                DiffTypeTablePolicy,
				Operation:           DiffOperationCreate,
				Path:                fmt.Sprintf("%s.%s.%s", td.Table.Schema, td.Table.Name, policyDiff.New.Name),
				Source:              policyDiff,
				CanRunInTransaction: true,
			}
			collector.collect(context, sql)
		} else {
			// Use ALTER POLICY for simple changes
			sql := generateAlterPolicySQL(policyDiff.Old, policyDiff.New, targetSchema)

			context := &diffContext{
				Type:                DiffTypeTablePolicy,
				Operation:           DiffOperationAlter,
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
			Type:                DiffTypeTableComment,
			Operation:           DiffOperationAlter,
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
				Type:                DiffTypeTableColumnComment,
				Operation:           DiffOperationAlter,
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
				Type:                DiffTypeTableIndexComment,
				Operation:           DiffOperationAlter,
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
			// Use CheckClause as-is since it's already normalized and formatted
			builder.WriteString(fmt.Sprintf(" %s", constraint.CheckClause))
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

// indexesStructurallyEqual compares two indexes for structural equality
// excluding comments and other metadata that don't require index recreation
func indexesStructurallyEqual(oldIndex, newIndex *ir.Index) bool {
	// Compare basic properties that would require recreation
	if oldIndex.Type != newIndex.Type ||
		oldIndex.Method != newIndex.Method ||
		oldIndex.IsPartial != newIndex.IsPartial ||
		oldIndex.IsExpression != newIndex.IsExpression ||
		oldIndex.Where != newIndex.Where {
		return false
	}

	// Compare column count
	if len(oldIndex.Columns) != len(newIndex.Columns) {
		return false
	}

	// Compare each column's properties
	for i, oldCol := range oldIndex.Columns {
		newCol := newIndex.Columns[i]
		if oldCol.Name != newCol.Name ||
			oldCol.Position != newCol.Position ||
			oldCol.Direction != newCol.Direction ||
			oldCol.Operator != newCol.Operator {
			return false
		}
	}

	return true
}

// generateTempIndexName generates a temporary name for an index during online replacement
func generateTempIndexName(originalName string) string {
	// Use pgschema-specific suffix to avoid conflicts
	return originalName + "_pgschema_new"
}
