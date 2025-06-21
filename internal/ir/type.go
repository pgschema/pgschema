package ir

import (
	"fmt"
	"strings"
)

// TypeKind represents the kind of PostgreSQL type
type TypeKind string

const (
	TypeKindEnum      TypeKind = "ENUM"
	TypeKindComposite TypeKind = "COMPOSITE"
	TypeKindDomain    TypeKind = "DOMAIN"
)

// TypeColumn represents a column in a composite type
type TypeColumn struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
	Position int    `json:"position"`
}

// DomainConstraint represents a constraint on a domain
type DomainConstraint struct {
	Name       string `json:"name"`
	Definition string `json:"definition"`
}

// Type represents a PostgreSQL user-defined type
type Type struct {
	Schema      string              `json:"schema"`
	Name        string              `json:"name"`
	Kind        TypeKind            `json:"kind"`
	Comment     string              `json:"comment,omitempty"`
	EnumValues  []string            `json:"enum_values,omitempty"`     // For ENUM types
	Columns     []*TypeColumn       `json:"columns,omitempty"`         // For composite types
	BaseType    string              `json:"base_type,omitempty"`       // For DOMAIN types
	NotNull     bool                `json:"not_null,omitempty"`        // For DOMAIN types
	Default     string              `json:"default,omitempty"`         // For DOMAIN types
	Constraints []*DomainConstraint `json:"constraints,omitempty"`     // For DOMAIN types
}

// GenerateSQL generates CREATE TYPE statement
func (t *Type) GenerateSQL() string {
	w := NewSQLWriter()

	var stmt string
	var objectType string
	switch t.Kind {
	case TypeKindEnum:
		stmt = t.generateEnumSQL()
		objectType = "TYPE"
	case TypeKindComposite:
		stmt = t.generateCompositeSQL()
		objectType = "TYPE"
	case TypeKindDomain:
		stmt = t.generateDomainSQL()
		objectType = "DOMAIN"
	default:
		return ""
	}

	w.WriteStatementWithComment(objectType, t.Name, t.Schema, "-", stmt)

	// Add comment if present
	if t.Comment != "" {
		w.WriteDDLSeparator()
		var commentStmt string
		if t.Kind == TypeKindDomain {
			commentStmt = fmt.Sprintf("COMMENT ON DOMAIN %s.%s IS '%s';", t.Schema, t.Name, t.Comment)
			w.WriteStatementWithComment("COMMENT", "DOMAIN "+t.Name, t.Schema, "-", commentStmt)
		} else {
			commentStmt = fmt.Sprintf("COMMENT ON TYPE %s.%s IS '%s';", t.Schema, t.Name, t.Comment)
			w.WriteStatementWithComment("COMMENT", "TYPE "+t.Name, t.Schema, "-", commentStmt)
		}
	}

	return w.String()
}

// generateEnumSQL generates CREATE TYPE ... AS ENUM statement
func (t *Type) generateEnumSQL() string {
	var values []string
	for _, value := range t.EnumValues {
		values = append(values, fmt.Sprintf("    '%s'", value))
	}

	return fmt.Sprintf("CREATE TYPE %s.%s AS ENUM (\n%s\n);",
		t.Schema, t.Name, strings.Join(values, ",\n"))
}

// generateCompositeSQL generates CREATE TYPE ... AS (...) statement
func (t *Type) generateCompositeSQL() string {
	var columns []string
	for _, col := range t.Columns {
		columns = append(columns, fmt.Sprintf("\t%s %s", col.Name, col.DataType))
	}

	return fmt.Sprintf("CREATE TYPE %s.%s AS (\n%s\n);",
		t.Schema, t.Name, strings.Join(columns, ",\n"))
}

// generateDomainSQL generates CREATE DOMAIN statement
func (t *Type) generateDomainSQL() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("CREATE DOMAIN %s.%s AS %s", t.Schema, t.Name, t.BaseType))
	
	if t.Default != "" {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", t.Default))
	}
	
	if t.NotNull {
		parts = append(parts, "NOT NULL")
	}
	
	for _, constraint := range t.Constraints {
		if constraint.Name != "" {
			parts = append(parts, fmt.Sprintf("\tCONSTRAINT %s %s", constraint.Name, constraint.Definition))
		} else {
			parts = append(parts, fmt.Sprintf("\t%s", constraint.Definition))
		}
	}
	
	return strings.Join(parts, "\n") + ";"
}
