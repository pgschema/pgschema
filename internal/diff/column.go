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

	// Check if there's a type change AND the column has a default value
	// In this case, we need to: DROP DEFAULT -> ALTER TYPE -> SET DEFAULT
	// because PostgreSQL can't automatically cast default values during type changes
	hasTypeChange := oldType != newType
	oldDefault := cd.Old.DefaultValue
	newDefault := cd.New.DefaultValue
	hasOldDefault := oldDefault != nil && *oldDefault != ""

	// If type is changing and there's an existing default, drop the default first
	if hasTypeChange && hasOldDefault {
		sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;",
			qualifiedTableName, cd.New.Name)
		statements = append(statements, sql)
	}

	// Handle data type changes
	if hasTypeChange {
		// Check if we need a USING clause for the type conversion
		// This is required when converting from text-like types to custom types (like ENUMs)
		// because PostgreSQL cannot implicitly cast these types
		if needsUsingClause(oldType, newType) {
			sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s USING %s::%s;",
				qualifiedTableName, cd.New.Name, newType, cd.New.Name, newType)
			statements = append(statements, sql)
		} else {
			sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;",
				qualifiedTableName, cd.New.Name, newType)
			statements = append(statements, sql)
		}
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
	// If we already dropped the default due to type change, we need to re-add it if there's a new default
	// Otherwise, handle default changes normally
	if hasTypeChange && hasOldDefault {
		// Default was dropped above; add new default if specified
		if newDefault != nil && *newDefault != "" {
			sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;",
				qualifiedTableName, cd.New.Name, *newDefault)
			statements = append(statements, sql)
		}
	} else {
		// Normal default value change handling (no type change involved)
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
	}

	return statements
}

// needsUsingClause determines if a type conversion requires a USING clause.
//
// This is especially important when converting to or from custom types (like ENUMs),
// because PostgreSQL often cannot implicitly cast these types. To avoid generating
// invalid migrations, this function takes a conservative approach:
//
//   - Any conversion involving at least one non–built-in (custom) type will require
//     a USING clause.
//   - For built-in → built-in conversions we still assume PostgreSQL provides an
//     implicit cast in most cases; callers should be aware that some edge cases
//     (e.g. certain text → json conversions) may still need manual adjustment.
func needsUsingClause(oldType, newType string) bool {
	// Check if old type is text-like
	oldIsTextLike := ir.IsTextLikeType(oldType)

	// Determine whether the old/new types are PostgreSQL built-ins
	oldIsBuiltIn := ir.IsBuiltInType(oldType)
	newIsBuiltIn := ir.IsBuiltInType(newType)

	// Preserve existing behavior: text-like → non–built-in likely needs USING
	if oldIsTextLike && !newIsBuiltIn {
		return true
	}

	// Be conservative for any conversion involving custom (non–built-in) types:
	// this covers custom → custom and built-in ↔ custom conversions.
	if !oldIsBuiltIn || !newIsBuiltIn {
		return true
	}

	// For built-in → built-in types we assume an implicit cast is available.
	return false
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
