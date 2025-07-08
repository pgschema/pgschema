package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
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

// GenerateMigrationSQL generates SQL statements for type modifications
func (td *TypeDiff) GenerateMigrationSQL() []string {
	var statements []string

	// Only ENUM types can be modified (add values)
	if td.Old.Kind == ir.TypeKindEnum && td.New.Kind == ir.TypeKindEnum {
		// Find added enum values
		oldValues := make(map[string]int)
		for i, value := range td.Old.EnumValues {
			oldValues[value] = i
		}

		for i, value := range td.New.EnumValues {
			if _, exists := oldValues[value]; !exists {
				// This is a new value - determine position
				var stmt string
				if i == 0 {
					// First value
					stmt = fmt.Sprintf("ALTER TYPE %s.%s ADD VALUE '%s' BEFORE '%s';",
						td.New.Schema, td.New.Name, value, td.New.EnumValues[1])
				} else if i == len(td.New.EnumValues)-1 {
					// Last value
					stmt = fmt.Sprintf("ALTER TYPE %s.%s ADD VALUE '%s' AFTER '%s';",
						td.New.Schema, td.New.Name, value, td.New.EnumValues[i-1])
				} else {
					// Middle value - add after the previous value
					stmt = fmt.Sprintf("ALTER TYPE %s.%s ADD VALUE '%s' AFTER '%s';",
						td.New.Schema, td.New.Name, value, td.New.EnumValues[i-1])
				}
				statements = append(statements, stmt)
			}
		}
	}

	return statements
}

// GenerateDropTypeSQL generates SQL for dropping types
func GenerateDropTypeSQL(types []*ir.Type) []string {
	var statements []string
	
	// Sort types by schema.name for consistent ordering
	sortedTypes := make([]*ir.Type, len(types))
	copy(sortedTypes, types)
	sort.Slice(sortedTypes, func(i, j int) bool {
		keyI := sortedTypes[i].Schema + "." + sortedTypes[i].Name
		keyJ := sortedTypes[j].Schema + "." + sortedTypes[j].Name
		return keyI < keyJ
	})
	
	for _, typeObj := range sortedTypes {
		statements = append(statements, fmt.Sprintf("DROP TYPE IF EXISTS %s.%s;", typeObj.Schema, typeObj.Name))
	}
	
	return statements
}

// GenerateCreateTypeSQL generates SQL for creating types
func GenerateCreateTypeSQL(types []*ir.Type) []string {
	var statements []string
	
	// Sort types by schema.name for consistent ordering
	sortedTypes := make([]*ir.Type, len(types))
	copy(sortedTypes, types)
	sort.Slice(sortedTypes, func(i, j int) bool {
		keyI := sortedTypes[i].Schema + "." + sortedTypes[i].Name
		keyJ := sortedTypes[j].Schema + "." + sortedTypes[j].Name
		return keyI < keyJ
	})
	
	for _, typeObj := range sortedTypes {
		// Generate CREATE TYPE statement without comments for migration
		switch typeObj.Kind {
		case ir.TypeKindEnum:
			var values []string
			for _, value := range typeObj.EnumValues {
				values = append(values, fmt.Sprintf("   '%s'", value))
			}
			stmt := fmt.Sprintf("CREATE TYPE %s.%s AS ENUM (\n%s\n);",
				typeObj.Schema, typeObj.Name, strings.Join(values, ",\n"))
			statements = append(statements, stmt)
		case ir.TypeKindComposite:
			var columns []string
			for _, col := range typeObj.Columns {
				columns = append(columns, fmt.Sprintf("\t%s %s", col.Name, col.DataType))
			}
			stmt := fmt.Sprintf("CREATE TYPE %s.%s AS (\n%s\n);",
				typeObj.Schema, typeObj.Name, strings.Join(columns, ",\n"))
			statements = append(statements, stmt)
		case ir.TypeKindDomain:
			var parts []string
			parts = append(parts, fmt.Sprintf("CREATE DOMAIN %s.%s AS %s", typeObj.Schema, typeObj.Name, typeObj.BaseType))
			if typeObj.Default != "" {
				parts = append(parts, fmt.Sprintf("DEFAULT %s", typeObj.Default))
			}
			if typeObj.NotNull {
				parts = append(parts, "NOT NULL")
			}
			for _, constraint := range typeObj.Constraints {
				if constraint.Name != "" {
					parts = append(parts, fmt.Sprintf("\tCONSTRAINT %s %s", constraint.Name, constraint.Definition))
				} else {
					parts = append(parts, fmt.Sprintf("\t%s", constraint.Definition))
				}
			}
			stmt := strings.Join(parts, "\n") + ";"
			statements = append(statements, stmt)
		}
	}
	
	return statements
}

// GenerateAlterTypeSQL generates SQL for modifying types
func GenerateAlterTypeSQL(typeDiffs []*TypeDiff) []string {
	var statements []string
	
	// Sort modified types by schema.name for consistent ordering
	sortedTypeDiffs := make([]*TypeDiff, len(typeDiffs))
	copy(sortedTypeDiffs, typeDiffs)
	sort.Slice(sortedTypeDiffs, func(i, j int) bool {
		keyI := sortedTypeDiffs[i].New.Schema + "." + sortedTypeDiffs[i].New.Name
		keyJ := sortedTypeDiffs[j].New.Schema + "." + sortedTypeDiffs[j].New.Name
		return keyI < keyJ
	})
	
	for _, typeDiff := range sortedTypeDiffs {
		statements = append(statements, typeDiff.GenerateMigrationSQL()...)
	}
	
	return statements
}