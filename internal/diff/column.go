package diff

import (
	"fmt"

	"github.com/pgschema/pgschema/ir"
)

// generateColumnSQL generates SQL statements for column modifications
func (cd *ColumnDiff) generateColumnSQL(tableSchema, tableName string, targetSchema string) []string {
	var statements []string
	qualifiedTableName := getTableNameWithSchema(tableSchema, tableName, targetSchema)

	// Handle data type changes
	if cd.Old.DataType != cd.New.DataType {
		sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;",
			qualifiedTableName, cd.New.Name, cd.New.DataType)
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
	oldDefault := cd.Old.DefaultValue
	newDefault := cd.New.DefaultValue

	// Normalize both defaults for comparison to match PostgreSQL's pg_get_expr() behavior
	var normalizedOldDefault, normalizedNewDefault *string
	if oldDefault != nil {
		normalized := normalizeDefaultExpr(*oldDefault, tableSchema)
		normalizedOldDefault = &normalized
	}
	if newDefault != nil {
		normalized := normalizeDefaultExpr(*newDefault, tableSchema)
		normalizedNewDefault = &normalized
	}

	if (normalizedOldDefault == nil) != (normalizedNewDefault == nil) ||
		(normalizedOldDefault != nil && normalizedNewDefault != nil && *normalizedOldDefault != *normalizedNewDefault) {

		var sql string
		if normalizedNewDefault == nil {
			sql = fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;",
				qualifiedTableName, cd.New.Name)
		} else {
			sql = fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;",
				qualifiedTableName, cd.New.Name, *normalizedNewDefault)
		}

		statements = append(statements, sql)
	}

	return statements
}

// columnsEqual compares two columns for equality
// tableSchema is used to normalize default expressions for comparison
func columnsEqual(old, new *ir.Column, tableSchema string) bool {
	if old.Name != new.Name {
		return false
	}
	if old.DataType != new.DataType {
		return false
	}
	if old.IsNullable != new.IsNullable {
		return false
	}

	// Compare default values with normalization
	// This ensures that "public.func()" and "func()" are treated as equal
	// when comparing columns in the "public" schema
	oldDefaultNormalized := ""
	newDefaultNormalized := ""
	if old.DefaultValue != nil {
		oldDefaultNormalized = normalizeDefaultExpr(*old.DefaultValue, tableSchema)
	}
	if new.DefaultValue != nil {
		newDefaultNormalized = normalizeDefaultExpr(*new.DefaultValue, tableSchema)
	}

	if (old.DefaultValue == nil) != (new.DefaultValue == nil) {
		return false
	}
	if old.DefaultValue != nil && new.DefaultValue != nil && oldDefaultNormalized != newDefaultNormalized {
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
