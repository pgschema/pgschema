package diff

import (
	"fmt"

	"github.com/pgschema/pgschema/internal/ir"
)

// generateColumnSQL generates SQL statements for column modifications
func (cd *ColumnDiff) generateColumnSQL(tableSchema, tableName string, targetSchema string) []string {
	var statements []string
	qualifiedTableName := getTableNameWithSchema(tableSchema, tableName, targetSchema)

	// Handle data type changes
	if cd.Old.DataType != cd.New.DataType {
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;",
			qualifiedTableName, cd.New.Name, cd.New.DataType))
	}

	// Handle nullable changes
	if cd.Old.IsNullable != cd.New.IsNullable {
		if cd.New.IsNullable {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;",
				qualifiedTableName, cd.New.Name))
		} else {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;",
				qualifiedTableName, cd.New.Name))
		}
	}

	// Handle default value changes
	oldDefault := cd.Old.DefaultValue
	newDefault := cd.New.DefaultValue

	if (oldDefault == nil) != (newDefault == nil) ||
		(oldDefault != nil && newDefault != nil && *oldDefault != *newDefault) {

		if newDefault == nil {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;",
				qualifiedTableName, cd.New.Name))
		} else {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;",
				qualifiedTableName, cd.New.Name, *newDefault))
		}
	}

	return statements
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

	// Compare identity columns
	if (old.Identity == nil) != (new.Identity == nil) {
		return false
	}
	if old.Identity != nil && new.Identity != nil {
		if old.Identity.Generation != new.Identity.Generation {
			return false
		}
	}

	return true
}
