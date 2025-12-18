package diff

import (
	"fmt"

	"github.com/pgschema/pgschema/ir"
)

// generateColumnSQL generates SQL statements for column modifications
func (cd *ColumnDiff) generateColumnSQL(tableSchema, tableName string, targetSchema string) []string {
	var statements []string
	qualifiedTableName := getTableNameWithSchema(tableSchema, tableName, targetSchema)

	// Handle data type changes - normalize types by stripping target schema prefix
	oldType := stripSchemaPrefix(cd.Old.DataType, targetSchema)
	newType := stripSchemaPrefix(cd.New.DataType, targetSchema)
	if oldType != newType {
		sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;",
			qualifiedTableName, cd.New.Name, newType)
		statements = append(statements, sql)
	}

	// Handle nullable changes
	if cd.Old.IsNullable != cd.New.IsNullable {
		if cd.New.IsNullable {
			// DROP NOT NULL
			sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;",
				qualifiedTableName, cd.New.Name)
			statements = append(statements, sql)
		} else {
			// ADD NOT NULL - generate canonical SQL only
			sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;",
				qualifiedTableName, cd.New.Name)
			statements = append(statements, sql)
		}
	}

	// Handle default value changes
	// Default values are already normalized by ir.normalizeColumn
	oldDefault := cd.Old.DefaultValue
	newDefault := cd.New.DefaultValue

	if (oldDefault == nil) != (newDefault == nil) ||
		(oldDefault != nil && newDefault != nil && *oldDefault != *newDefault) {

		var sql string
		if newDefault == nil {
			sql = fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;",
				qualifiedTableName, cd.New.Name)
		} else {
			sql = fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;",
				qualifiedTableName, cd.New.Name, *newDefault)
		}

		statements = append(statements, sql)
	}

	return statements
}

// columnsEqual compares two columns for equality
// targetSchema is used to normalize type names before comparison
func columnsEqual(old, new *ir.Column, targetSchema string) bool {
	if old.Name != new.Name {
		return false
	}
	// Normalize types by stripping target schema prefix before comparison
	oldType := stripSchemaPrefix(old.DataType, targetSchema)
	newType := stripSchemaPrefix(new.DataType, targetSchema)
	if oldType != newType {
		return false
	}
	if old.IsNullable != new.IsNullable {
		return false
	}

	// Compare default values (already normalized by ir.normalizeColumn)
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

	// Compare comments
	if old.Comment != new.Comment {
		return false
	}

	return true
}
