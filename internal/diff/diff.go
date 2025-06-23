package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

// DDLDiff represents the difference between two DDL states
type DDLDiff struct {
	AddedTables       []*ir.Table
	DroppedTables     []*ir.Table
	ModifiedTables    []*TableDiff
	AddedExtensions   []*ir.Extension
	DroppedExtensions []*ir.Extension
	AddedFunctions    []*ir.Function
	DroppedFunctions  []*ir.Function
	ModifiedFunctions []*FunctionDiff
}

// FunctionDiff represents changes to a function
type FunctionDiff struct {
	Old *ir.Function
	New *ir.Function
}

// TableDiff represents changes to a table
type TableDiff struct {
	Table              *ir.Table
	AddedColumns       []*ir.Column
	DroppedColumns     []*ir.Column
	ModifiedColumns    []*ColumnDiff
	AddedConstraints   []*ir.Constraint
	DroppedConstraints []*ir.Constraint
}

// ColumnDiff represents changes to a column
type ColumnDiff struct {
	Old *ir.Column
	New *ir.Column
}

// Diff compares two DDL strings and returns the differences
func Diff(oldDDL, newDDL string) (*DDLDiff, error) {
	// Parse the old DDL string to IR Schema
	oldParser := ir.NewParser()
	oldSchema, err := oldParser.ParseSQL(oldDDL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse old DDL: %w", err)
	}

	// Parse the new DDL string to IR Schema
	newParser := ir.NewParser()
	newSchema, err := newParser.ParseSQL(newDDL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse new DDL: %w", err)
	}

	// Perform the diff using the parsed schemas
	return diffSchemas(oldSchema, newSchema), nil
}

// diffSchemas compares two IR schemas and returns the differences
func diffSchemas(oldSchema, newSchema *ir.Schema) *DDLDiff {
	diff := &DDLDiff{
		AddedTables:       []*ir.Table{},
		DroppedTables:     []*ir.Table{},
		ModifiedTables:    []*TableDiff{},
		AddedExtensions:   []*ir.Extension{},
		DroppedExtensions: []*ir.Extension{},
		AddedFunctions:    []*ir.Function{},
		DroppedFunctions:  []*ir.Function{},
		ModifiedFunctions: []*FunctionDiff{},
	}

	// Build maps for efficient lookup
	oldTables := make(map[string]*ir.Table)
	newTables := make(map[string]*ir.Table)

	// Extract tables from all schemas in oldSchema
	for _, dbSchema := range oldSchema.Schemas {
		for _, table := range dbSchema.Tables {
			key := table.Schema + "." + table.Name
			oldTables[key] = table
		}
	}

	// Extract tables from all schemas in newSchema
	for _, dbSchema := range newSchema.Schemas {
		for _, table := range dbSchema.Tables {
			key := table.Schema + "." + table.Name
			newTables[key] = table
		}
	}

	// Find added tables
	for key, table := range newTables {
		if _, exists := oldTables[key]; !exists {
			diff.AddedTables = append(diff.AddedTables, table)
		}
	}

	// Find dropped tables
	for key, table := range oldTables {
		if _, exists := newTables[key]; !exists {
			diff.DroppedTables = append(diff.DroppedTables, table)
		}
	}

	// Find modified tables
	for key, newTable := range newTables {
		if oldTable, exists := oldTables[key]; exists {
			if tableDiff := diffTables(oldTable, newTable); tableDiff != nil {
				diff.ModifiedTables = append(diff.ModifiedTables, tableDiff)
			}
		}
	}

	// Compare extensions
	oldExtensions := make(map[string]*ir.Extension)
	newExtensions := make(map[string]*ir.Extension)

	if oldSchema.Extensions != nil {
		for name, ext := range oldSchema.Extensions {
			oldExtensions[name] = ext
		}
	}

	if newSchema.Extensions != nil {
		for name, ext := range newSchema.Extensions {
			newExtensions[name] = ext
		}
	}

	// Find added extensions
	for name, ext := range newExtensions {
		if _, exists := oldExtensions[name]; !exists {
			diff.AddedExtensions = append(diff.AddedExtensions, ext)
		}
	}

	// Find dropped extensions
	for name, ext := range oldExtensions {
		if _, exists := newExtensions[name]; !exists {
			diff.DroppedExtensions = append(diff.DroppedExtensions, ext)
		}
	}

	// Compare functions across all schemas
	oldFunctions := make(map[string]*ir.Function)
	newFunctions := make(map[string]*ir.Function)

	// Extract functions from all schemas in oldSchema
	for _, dbSchema := range oldSchema.Schemas {
		for funcName, function := range dbSchema.Functions {
			// Use schema.name(arguments) as key to distinguish functions with different signatures
			key := function.Schema + "." + funcName + "(" + function.Arguments + ")"
			oldFunctions[key] = function
		}
	}

	// Extract functions from all schemas in newSchema
	for _, dbSchema := range newSchema.Schemas {
		for funcName, function := range dbSchema.Functions {
			// Use schema.name(arguments) as key to distinguish functions with different signatures
			key := function.Schema + "." + funcName + "(" + function.Arguments + ")"
			newFunctions[key] = function
		}
	}

	// Find added functions
	for key, function := range newFunctions {
		if _, exists := oldFunctions[key]; !exists {
			diff.AddedFunctions = append(diff.AddedFunctions, function)
		}
	}

	// Find dropped functions
	for key, function := range oldFunctions {
		if _, exists := newFunctions[key]; !exists {
			diff.DroppedFunctions = append(diff.DroppedFunctions, function)
		}
	}

	// Find modified functions
	for key, newFunction := range newFunctions {
		if oldFunction, exists := oldFunctions[key]; exists {
			if !functionsEqual(oldFunction, newFunction) {
				diff.ModifiedFunctions = append(diff.ModifiedFunctions, &FunctionDiff{
					Old: oldFunction,
					New: newFunction,
				})
			}
		}
	}

	return diff
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

	// Return nil if no changes
	if len(diff.AddedColumns) == 0 && len(diff.DroppedColumns) == 0 &&
		len(diff.ModifiedColumns) == 0 && len(diff.AddedConstraints) == 0 &&
		len(diff.DroppedConstraints) == 0 {
		return nil
	}

	return diff
}

// columnsEqual compares two columns for equality
func columnsEqual(old, new *ir.Column) bool {
	if old.Name != new.Name {
		return false
	}
	if old.DataType != new.DataType {
		return false
	}
	if old.IsNullable != new.IsNullable {
		return false
	}

	// Compare default values
	if (old.DefaultValue == nil) != (new.DefaultValue == nil) {
		return false
	}
	if old.DefaultValue != nil && new.DefaultValue != nil && *old.DefaultValue != *new.DefaultValue {
		return false
	}

	// Compare max length
	if (old.MaxLength == nil) != (new.MaxLength == nil) {
		return false
	}
	if old.MaxLength != nil && new.MaxLength != nil && *old.MaxLength != *new.MaxLength {
		return false
	}

	return true
}

// functionsEqual compares two functions for equality
func functionsEqual(old, new *ir.Function) bool {
	if old.Schema != new.Schema {
		return false
	}
	if old.Name != new.Name {
		return false
	}
	if old.Definition != new.Definition {
		return false
	}
	if old.ReturnType != new.ReturnType {
		return false
	}
	if old.Language != new.Language {
		return false
	}
	if old.Arguments != new.Arguments {
		return false
	}
	if old.Signature != new.Signature {
		return false
	}
	if old.Volatility != new.Volatility {
		return false
	}
	if old.IsSecurityDefiner != new.IsSecurityDefiner {
		return false
	}
	if old.IsStrict != new.IsStrict {
		return false
	}

	return true
}

// GenerateMigrationSQL generates SQL statements for the migration
func (d *DDLDiff) GenerateMigrationSQL() string {
	var statements []string

	// Drop extensions first (before dropping tables that might depend on them)
	// Sort extensions by name for consistent ordering
	sortedDroppedExtensions := make([]*ir.Extension, len(d.DroppedExtensions))
	copy(sortedDroppedExtensions, d.DroppedExtensions)
	sort.Slice(sortedDroppedExtensions, func(i, j int) bool {
		return sortedDroppedExtensions[i].Name < sortedDroppedExtensions[j].Name
	})
	for _, ext := range sortedDroppedExtensions {
		statements = append(statements, fmt.Sprintf("DROP EXTENSION IF EXISTS %s;", ext.Name))
	}

	// Drop tables
	for _, table := range d.DroppedTables {
		statements = append(statements, fmt.Sprintf("DROP TABLE %s.%s;", table.Schema, table.Name))
	}

	// Create extensions (before creating tables that might depend on them)
	// Sort extensions by name for consistent ordering
	sortedExtensions := make([]*ir.Extension, len(d.AddedExtensions))
	copy(sortedExtensions, d.AddedExtensions)
	sort.Slice(sortedExtensions, func(i, j int) bool {
		return sortedExtensions[i].Name < sortedExtensions[j].Name
	})
	for _, ext := range sortedExtensions {
		if ext.Schema != "" {
			statements = append(statements, fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s WITH SCHEMA %s;", ext.Name, ext.Schema))
		} else {
			statements = append(statements, fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s;", ext.Name))
		}
	}

	// Drop functions
	// Sort functions by schema.name for consistent ordering
	sortedDroppedFunctions := make([]*ir.Function, len(d.DroppedFunctions))
	copy(sortedDroppedFunctions, d.DroppedFunctions)
	sort.Slice(sortedDroppedFunctions, func(i, j int) bool {
		keyI := sortedDroppedFunctions[i].Schema + "." + sortedDroppedFunctions[i].Name
		keyJ := sortedDroppedFunctions[j].Schema + "." + sortedDroppedFunctions[j].Name
		return keyI < keyJ
	})
	for _, function := range sortedDroppedFunctions {
		if function.Arguments != "" {
			statements = append(statements, fmt.Sprintf("DROP FUNCTION IF EXISTS %s.%s(%s);", function.Schema, function.Name, function.Arguments))
		} else {
			statements = append(statements, fmt.Sprintf("DROP FUNCTION IF EXISTS %s.%s();", function.Schema, function.Name))
		}
	}

	// Create new tables
	for _, table := range d.AddedTables {
		statements = append(statements, table.GenerateSQL())
	}

	// Create functions (after tables, in case they reference tables)
	// Sort functions by schema.name for consistent ordering
	sortedAddedFunctions := make([]*ir.Function, len(d.AddedFunctions))
	copy(sortedAddedFunctions, d.AddedFunctions)
	sort.Slice(sortedAddedFunctions, func(i, j int) bool {
		keyI := sortedAddedFunctions[i].Schema + "." + sortedAddedFunctions[i].Name
		keyJ := sortedAddedFunctions[j].Schema + "." + sortedAddedFunctions[j].Name
		return keyI < keyJ
	})
	for _, function := range sortedAddedFunctions {
		// Build the CREATE FUNCTION statement from parsed data
		var stmt strings.Builder

		// Build the CREATE FUNCTION header with schema qualification
		if function.Schema != "" {
			stmt.WriteString(fmt.Sprintf("CREATE OR REPLACE FUNCTION %s.%s", function.Schema, function.Name))
		} else {
			stmt.WriteString(fmt.Sprintf("CREATE OR REPLACE FUNCTION %s", function.Name))
		}

		// Add parameters
		if function.Signature != "" {
			stmt.WriteString(fmt.Sprintf("(\n    %s\n)", strings.ReplaceAll(function.Signature, ", ", ",\n    ")))
		} else {
			stmt.WriteString("()")
		}

		// Add return type
		if function.ReturnType != "" {
			stmt.WriteString(fmt.Sprintf("\nRETURNS %s", function.ReturnType))
		}

		// Add language
		if function.Language != "" {
			stmt.WriteString(fmt.Sprintf("\nLANGUAGE %s", function.Language))
		}

		// Add security definer if set
		if function.IsSecurityDefiner {
			stmt.WriteString("\nSECURITY DEFINER")
		}

		// Add volatility if not default
		if function.Volatility != "" {
			stmt.WriteString(fmt.Sprintf("\n%s", function.Volatility))
		}

		// Add the function body
		if function.Definition != "" {
			stmt.WriteString(fmt.Sprintf("\nAS $$%s\n$$;", function.Definition))
		}

		statements = append(statements, stmt.String())
	}

	// Modify existing functions (using CREATE OR REPLACE)
	// Sort modified functions by schema.name for consistent ordering
	sortedModifiedFunctions := make([]*FunctionDiff, len(d.ModifiedFunctions))
	copy(sortedModifiedFunctions, d.ModifiedFunctions)
	sort.Slice(sortedModifiedFunctions, func(i, j int) bool {
		keyI := sortedModifiedFunctions[i].New.Schema + "." + sortedModifiedFunctions[i].New.Name
		keyJ := sortedModifiedFunctions[j].New.Schema + "." + sortedModifiedFunctions[j].New.Name
		return keyI < keyJ
	})
	for _, functionDiff := range sortedModifiedFunctions {
		function := functionDiff.New
		// Build the CREATE OR REPLACE FUNCTION statement from parsed data
		var stmt strings.Builder

		// Build the CREATE OR REPLACE FUNCTION header with schema qualification
		if function.Schema != "" {
			stmt.WriteString(fmt.Sprintf("CREATE OR REPLACE FUNCTION %s.%s", function.Schema, function.Name))
		} else {
			stmt.WriteString(fmt.Sprintf("CREATE OR REPLACE FUNCTION %s", function.Name))
		}

		// Add parameters
		if function.Signature != "" {
			stmt.WriteString(fmt.Sprintf("(\n    %s\n)", strings.ReplaceAll(function.Signature, ", ", ",\n    ")))
		} else {
			stmt.WriteString("()")
		}

		// Add return type
		if function.ReturnType != "" {
			stmt.WriteString(fmt.Sprintf("\nRETURNS %s", function.ReturnType))
		}

		// Add language
		if function.Language != "" {
			stmt.WriteString(fmt.Sprintf("\nLANGUAGE %s", function.Language))
		}

		// Add security definer/invoker
		if function.IsSecurityDefiner {
			stmt.WriteString("\nSECURITY DEFINER")
		} else {
			stmt.WriteString("\nSECURITY INVOKER")
		}

		// Add volatility if not default
		if function.Volatility != "" {
			stmt.WriteString(fmt.Sprintf("\n%s", function.Volatility))
		}

		// Add the function body
		if function.Definition != "" {
			stmt.WriteString(fmt.Sprintf("\nAS $$%s\n$$;", function.Definition))
		}

		statements = append(statements, stmt.String())
	}

	// Modify existing tables
	for _, tableDiff := range d.ModifiedTables {
		statements = append(statements, tableDiff.GenerateMigrationSQL()...)
	}

	return strings.Join(statements, "\n")
}

// GenerateMigrationSQL generates SQL statements for table modifications
func (td *TableDiff) GenerateMigrationSQL() []string {
	var statements []string

	// Drop constraints first (before dropping columns)
	for _, constraint := range td.DroppedConstraints {
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s.%s DROP CONSTRAINT %s;",
			td.Table.Schema, td.Table.Name, constraint.Name))
	}

	// Drop columns (sort by position for consistent ordering)
	sortedDroppedColumns := make([]*ir.Column, len(td.DroppedColumns))
	copy(sortedDroppedColumns, td.DroppedColumns)
	sort.Slice(sortedDroppedColumns, func(i, j int) bool {
		return sortedDroppedColumns[i].Position < sortedDroppedColumns[j].Position
	})
	for _, column := range sortedDroppedColumns {
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s.%s DROP COLUMN %s;",
			td.Table.Schema, td.Table.Name, column.Name))
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
		if fkConstraint != nil {
			stmt = fmt.Sprintf("ALTER TABLE %s.%s\nADD COLUMN %s %s",
				td.Table.Schema, td.Table.Name, column.Name, column.DataType)
		} else {
			stmt = fmt.Sprintf("ALTER TABLE %s.%s ADD COLUMN %s %s",
				td.Table.Schema, td.Table.Name, column.Name, column.DataType)
		}

		// Add foreign key reference inline if present
		if fkConstraint != nil {
			stmt += fmt.Sprintf(" REFERENCES %s.%s", fkConstraint.ReferencedSchema, fkConstraint.ReferencedTable)

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

		if column.DefaultValue != nil {
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
		statements = append(statements, columnDiff.GenerateMigrationSQL(td.Table.Schema, td.Table.Name)...)
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
			columns := constraint.SortConstraintColumnsByPosition()
			var columnNames []string
			for _, col := range columns {
				columnNames = append(columnNames, col.Name)
			}
			stmt := fmt.Sprintf("ALTER TABLE %s.%s \nADD CONSTRAINT %s UNIQUE (%s);",
				td.Table.Schema, td.Table.Name, constraint.Name, strings.Join(columnNames, ", "))
			statements = append(statements, stmt)

		case ir.ConstraintTypeCheck:
			// CheckClause already contains "CHECK (...)" from the constraint definition
			stmt := fmt.Sprintf("ALTER TABLE %s.%s \nADD CONSTRAINT %s %s;",
				td.Table.Schema, td.Table.Name, constraint.Name, constraint.CheckClause)
			statements = append(statements, stmt)

		case ir.ConstraintTypeForeignKey:
			// Sort columns by position
			columns := constraint.SortConstraintColumnsByPosition()
			var columnNames []string
			for _, col := range columns {
				columnNames = append(columnNames, col.Name)
			}

			// Sort referenced columns by position
			var refColumnNames []string
			if len(constraint.ReferencedColumns) > 0 {
				refColumns := make([]*ir.ConstraintColumn, len(constraint.ReferencedColumns))
				copy(refColumns, constraint.ReferencedColumns)
				sort.Slice(refColumns, func(i, j int) bool {
					return refColumns[i].Position < refColumns[j].Position
				})
				for _, col := range refColumns {
					refColumnNames = append(refColumnNames, col.Name)
				}
			}

			stmt := fmt.Sprintf("ALTER TABLE %s.%s \nADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s.%s(%s)",
				td.Table.Schema, td.Table.Name, constraint.Name,
				strings.Join(columnNames, ", "),
				constraint.ReferencedSchema, constraint.ReferencedTable,
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
			columns := constraint.SortConstraintColumnsByPosition()
			var columnNames []string
			for _, col := range columns {
				columnNames = append(columnNames, col.Name)
			}
			stmt := fmt.Sprintf("ALTER TABLE %s.%s \nADD CONSTRAINT %s PRIMARY KEY (%s);",
				td.Table.Schema, td.Table.Name, constraint.Name, strings.Join(columnNames, ", "))
			statements = append(statements, stmt)
		}
	}

	return statements
}

// GenerateMigrationSQL generates SQL statements for column modifications
func (cd *ColumnDiff) GenerateMigrationSQL(schema, tableName string) []string {
	var statements []string

	// Handle data type changes
	if cd.Old.DataType != cd.New.DataType {
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s.%s ALTER COLUMN %s TYPE %s;",
			schema, tableName, cd.New.Name, cd.New.DataType))
	}

	// Handle nullable changes
	if cd.Old.IsNullable != cd.New.IsNullable {
		if cd.New.IsNullable {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s.%s ALTER COLUMN %s DROP NOT NULL;",
				schema, tableName, cd.New.Name))
		} else {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s.%s ALTER COLUMN %s SET NOT NULL;",
				schema, tableName, cd.New.Name))
		}
	}

	// Handle default value changes
	oldDefault := cd.Old.DefaultValue
	newDefault := cd.New.DefaultValue

	if (oldDefault == nil) != (newDefault == nil) ||
		(oldDefault != nil && newDefault != nil && *oldDefault != *newDefault) {

		if newDefault == nil {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s.%s ALTER COLUMN %s DROP DEFAULT;",
				schema, tableName, cd.New.Name))
		} else {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s.%s ALTER COLUMN %s SET DEFAULT %s;",
				schema, tableName, cd.New.Name, *newDefault))
		}
	}

	return statements
}
