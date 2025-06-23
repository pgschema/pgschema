package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

// DDLDiff represents the difference between two DDL states
type DDLDiff struct {
	AddedTables   []*ir.Table
	DroppedTables []*ir.Table
	ModifiedTables []*TableDiff
}

// TableDiff represents changes to a table
type TableDiff struct {
	Table            *ir.Table
	AddedColumns     []*ir.Column
	DroppedColumns   []*ir.Column
	ModifiedColumns  []*ColumnDiff
	AddedConstraints []*ir.Constraint
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
		AddedTables:   []*ir.Table{},
		DroppedTables: []*ir.Table{},
		ModifiedTables: []*TableDiff{},
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

	return diff
}

// diffTables compares two tables and returns the differences
func diffTables(oldTable, newTable *ir.Table) *TableDiff {
	diff := &TableDiff{
		Table:            newTable,
		AddedColumns:     []*ir.Column{},
		DroppedColumns:   []*ir.Column{},
		ModifiedColumns:  []*ColumnDiff{},
		AddedConstraints: []*ir.Constraint{},
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

// GenerateMigrationSQL generates SQL statements for the migration
func (d *DDLDiff) GenerateMigrationSQL() string {
	var statements []string

	// Drop tables first
	for _, table := range d.DroppedTables {
		statements = append(statements, fmt.Sprintf("DROP TABLE %s.%s;", table.Schema, table.Name))
	}

	// Create new tables
	for _, table := range d.AddedTables {
		statements = append(statements, table.GenerateSQL())
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

	// Drop columns
	for _, column := range td.DroppedColumns {
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s.%s DROP COLUMN %s;", 
			td.Table.Schema, td.Table.Name, column.Name))
	}

	// Add new columns
	for _, column := range td.AddedColumns {
		// Check if this column has an associated foreign key constraint
		var fkConstraint *ir.Constraint
		for _, constraint := range td.Table.Constraints {
			if constraint.Type == ir.ConstraintTypeForeignKey && 
				len(constraint.Columns) == 1 && 
				constraint.Columns[0].Name == column.Name {
				fkConstraint = constraint
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

	// Modify existing columns
	for _, columnDiff := range td.ModifiedColumns {
		statements = append(statements, columnDiff.GenerateMigrationSQL(td.Table.Schema, td.Table.Name)...)
	}

	// Add new constraints
	for _, constraint := range td.AddedConstraints {
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