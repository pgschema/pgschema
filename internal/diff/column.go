package diff

import (
	"fmt"

	"github.com/pgschema/pgschema/internal/ir"
)

// ColumnSQLResult contains the result of column SQL generation
type ColumnSQLResult struct {
	Statements []string
	Rewrite    *DiffRewrite
}

// generateColumnSQL generates SQL statements for column modifications
func (cd *columnDiff) generateColumnSQL(tableSchema, tableName string, targetSchema string) []ColumnSQLResult {
	var results []ColumnSQLResult
	qualifiedTableName := getTableNameWithSchema(tableSchema, tableName, targetSchema)

	// Handle data type changes
	if cd.Old.DataType != cd.New.DataType {
		sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;",
			qualifiedTableName, cd.New.Name, cd.New.DataType)
		results = append(results, ColumnSQLResult{
			Statements: []string{sql},
			Rewrite:    nil, // No rewrite support for type changes yet
		})
	}

	// Handle nullable changes
	if cd.Old.IsNullable != cd.New.IsNullable {
		if cd.New.IsNullable {
			// DROP NOT NULL - no rewrite needed
			sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;",
				qualifiedTableName, cd.New.Name)
			results = append(results, ColumnSQLResult{
				Statements: []string{sql},
				Rewrite:    nil,
			})
		} else {
			// ADD NOT NULL - generate rewrite for online operations
			canonicalSQL := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;",
				qualifiedTableName, cd.New.Name)

			// Generate rewrite for online operations
			constraintName := fmt.Sprintf("%s_not_null", cd.New.Name)
			checkSQL := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s CHECK (%s IS NOT NULL) NOT VALID;",
				qualifiedTableName, constraintName, cd.New.Name)
			validateSQL := fmt.Sprintf("ALTER TABLE %s VALIDATE CONSTRAINT %s;",
				qualifiedTableName, constraintName)
			setNotNullSQL := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;",
				qualifiedTableName, cd.New.Name)

			rewrite := &DiffRewrite{
				Statements: []RewriteStatement{
					{
						SQL:                 checkSQL,
						CanRunInTransaction: true,
					},
					{
						SQL:                 validateSQL,
						CanRunInTransaction: true,
					},
					{
						SQL:                 setNotNullSQL,
						CanRunInTransaction: true,
					},
				},
			}

			results = append(results, ColumnSQLResult{
				Statements: []string{canonicalSQL},
				Rewrite:    rewrite,
			})
		}
	}

	// Handle default value changes
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

		results = append(results, ColumnSQLResult{
			Statements: []string{sql},
			Rewrite:    nil, // No rewrite support for default changes yet
		})
	}

	return results
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

	// Compare comments
	if old.Comment != new.Comment {
		return false
	}

	return true
}
