package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
	"github.com/pgschema/pgschema/internal/utils"
)

// typesEqual compares two types for equality
func typesEqual(old, new *ir.Type) bool {
	if old.Schema != new.Schema {
		return false
	}
	if old.Name != new.Name {
		return false
	}
	if old.Kind != new.Kind {
		return false
	}

	switch old.Kind {
	case ir.TypeKindEnum:
		// For ENUM types, compare values
		if len(old.EnumValues) != len(new.EnumValues) {
			return false
		}
		for i, value := range old.EnumValues {
			if value != new.EnumValues[i] {
				return false
			}
		}

	case ir.TypeKindComposite:
		// For composite types, compare columns
		if len(old.Columns) != len(new.Columns) {
			return false
		}
		for i, col := range old.Columns {
			newCol := new.Columns[i]
			if col.Name != newCol.Name || col.DataType != newCol.DataType {
				return false
			}
		}

	case ir.TypeKindDomain:
		// For domain types, compare base type and constraints
		if old.BaseType != new.BaseType {
			return false
		}
		if old.NotNull != new.NotNull {
			return false
		}
		if old.Default != new.Default {
			return false
		}
		if len(old.Constraints) != len(new.Constraints) {
			return false
		}
		for i, constraint := range old.Constraints {
			newConstraint := new.Constraints[i]
			if constraint.Name != newConstraint.Name || constraint.Definition != newConstraint.Definition {
				return false
			}
		}
	}

	return true
}


// generateDropTypesSQL generates DROP TYPE statements
func generateDropTypesSQL(w *SQLWriter, types []*ir.Type, targetSchema string) {
	// Sort types by name for consistent ordering
	sortedTypes := make([]*ir.Type, len(types))
	copy(sortedTypes, types)
	sort.Slice(sortedTypes, func(i, j int) bool {
		return sortedTypes[i].Name < sortedTypes[j].Name
	})

	for _, typeObj := range sortedTypes {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP TYPE IF EXISTS %s CASCADE;", typeObj.Name)
		w.WriteStatementWithComment("TYPE", typeObj.Name, typeObj.Schema, "", sql, targetSchema)
	}
}

// generateCreateTypesSQL generates CREATE TYPE statements
func generateCreateTypesSQL(w *SQLWriter, types []*ir.Type, targetSchema string) {
	// Sort types: CREATE TYPE statements first, then CREATE DOMAIN statements
	sortedTypes := make([]*ir.Type, len(types))
	copy(sortedTypes, types)
	sort.Slice(sortedTypes, func(i, j int) bool {
		typeI := sortedTypes[i]
		typeJ := sortedTypes[j]

		// Domain types should come after non-domain types
		if typeI.Kind == ir.TypeKindDomain && typeJ.Kind != ir.TypeKindDomain {
			return false
		}
		if typeI.Kind != ir.TypeKindDomain && typeJ.Kind == ir.TypeKindDomain {
			return true
		}

		// Within the same category, sort alphabetically by name
		return typeI.Name < typeJ.Name
	})

	for _, typeObj := range sortedTypes {
		w.WriteDDLSeparator()
		sql := generateTypeSQL(typeObj, targetSchema)

		// Use correct object type for comment
		var objectType string
		switch typeObj.Kind {
		case ir.TypeKindDomain:
			objectType = "DOMAIN"
		default:
			objectType = "TYPE"
		}

		w.WriteStatementWithComment(objectType, typeObj.Name, typeObj.Schema, "", sql, targetSchema)
	}
}

// generateModifyTypesSQL generates ALTER TYPE statements
func generateModifyTypesSQL(w *SQLWriter, diffs []*TypeDiff, targetSchema string) {
	for _, diff := range diffs {
		// Only ENUM types can be modified by adding values
		if diff.Old.Kind == ir.TypeKindEnum && diff.New.Kind == ir.TypeKindEnum {
			// Generate ALTER TYPE ... ADD VALUE statements for new enum values
			alterStatements := generateAlterTypeEnumStatements(diff.Old, diff.New, targetSchema)
			for _, stmt := range alterStatements {
				w.WriteDDLSeparator()
				w.WriteString(stmt) // No comments for diff scenarios
			}
		}
	}
}

// generateAlterTypeEnumStatements generates ALTER TYPE ADD VALUE statements for enum changes
func generateAlterTypeEnumStatements(oldType, newType *ir.Type, targetSchema string) []string {
	var statements []string

	// Create a map of old enum values for quick lookup
	oldValues := make(map[string]int)
	for i, value := range oldType.EnumValues {
		oldValues[value] = i
	}

	// Find new values and their positions
	typeName := utils.QualifyEntityName(newType.Schema, newType.Name, targetSchema)

	for i, newValue := range newType.EnumValues {
		if _, exists := oldValues[newValue]; !exists {
			// This is a new value, generate ALTER TYPE ADD VALUE statement
			var stmt string
			if i == 0 {
				// Add at the beginning
				stmt = fmt.Sprintf("ALTER TYPE %s ADD VALUE '%s' BEFORE '%s';", typeName, newValue, newType.EnumValues[1])
			} else if i == len(newType.EnumValues)-1 {
				// Add at the end
				stmt = fmt.Sprintf("ALTER TYPE %s ADD VALUE '%s' AFTER '%s';", typeName, newValue, newType.EnumValues[i-1])
			} else {
				// Add in the middle
				stmt = fmt.Sprintf("ALTER TYPE %s ADD VALUE '%s' AFTER '%s';", typeName, newValue, newType.EnumValues[i-1])
			}
			statements = append(statements, stmt)
		}
	}

	return statements
}

// generateTypeSQL generates CREATE TYPE statement
func generateTypeSQL(typeObj *ir.Type, targetSchema string) string {
	// Only include type name without schema if it's in the target schema
	typeName := utils.QualifyEntityName(typeObj.Schema, typeObj.Name, targetSchema)

	switch typeObj.Kind {
	case ir.TypeKindEnum:
		if len(typeObj.EnumValues) == 0 {
			return fmt.Sprintf("CREATE TYPE %s AS ENUM ();", typeName)
		}

		// Use multi-line format for better readability
		var lines []string
		lines = append(lines, fmt.Sprintf("CREATE TYPE %s AS ENUM (", typeName))
		for i, value := range typeObj.EnumValues {
			if i == len(typeObj.EnumValues)-1 {
				// Last value, no comma
				lines = append(lines, fmt.Sprintf("    '%s'", value))
			} else {
				// Not last value, add comma
				lines = append(lines, fmt.Sprintf("    '%s',", value))
			}
		}
		lines = append(lines, ");")
		return strings.Join(lines, "\n")
	case ir.TypeKindComposite:
		var attributes []string
		for _, attr := range typeObj.Columns {
			attributes = append(attributes, fmt.Sprintf("%s %s", attr.Name, attr.DataType))
		}
		return fmt.Sprintf("CREATE TYPE %s AS (%s);", typeName, strings.Join(attributes, ", "))
	case ir.TypeKindDomain:
		stmt := fmt.Sprintf("CREATE DOMAIN %s AS %s", typeName, typeObj.BaseType)
		if typeObj.Default != "" {
			stmt += fmt.Sprintf(" DEFAULT %s", typeObj.Default)
		}
		if typeObj.NotNull {
			stmt += " NOT NULL"
		}
		// Add domain constraints (CHECK constraints)
		for _, constraint := range typeObj.Constraints {
			if constraint.Name != "" {
				stmt += fmt.Sprintf(" CONSTRAINT %s %s", constraint.Name, constraint.Definition)
			} else {
				stmt += fmt.Sprintf(" %s", constraint.Definition)
			}
		}
		return stmt + ";"
	default:
		return fmt.Sprintf("CREATE TYPE %s;", typeName)
	}
}
