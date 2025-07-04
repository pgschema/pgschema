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
	EnumValues  []string            `json:"enum_values,omitempty"` // For ENUM types
	Columns     []*TypeColumn       `json:"columns,omitempty"`     // For composite types
	BaseType    string              `json:"base_type,omitempty"`   // For DOMAIN types
	NotNull     bool                `json:"not_null,omitempty"`    // For DOMAIN types
	Default     string              `json:"default,omitempty"`     // For DOMAIN types
	Constraints []*DomainConstraint `json:"constraints,omitempty"` // For DOMAIN types
}

// GenerateSQL generates CREATE TYPE statement
func (t *Type) GenerateSQL() string {
	return t.GenerateSQLWithSchema(t.Schema)
}

// GenerateSQLWithSchema generates SQL for a type with target schema context
func (t *Type) GenerateSQLWithSchema(targetSchema string) string {
	return t.GenerateSQLWithOptions(true, targetSchema)
}

// GenerateSQLWithOptions generates SQL for a type with configurable comment inclusion
func (t *Type) GenerateSQLWithOptions(includeComments bool, targetSchema string) string {
	w := NewSQLWriterWithComments(includeComments)

	var stmt string
	var objectType string
	switch t.Kind {
	case TypeKindEnum:
		stmt = t.generateEnumSQLWithSchema(targetSchema)
		objectType = "TYPE"
	case TypeKindComposite:
		stmt = t.generateCompositeSQLWithSchema(targetSchema)
		objectType = "TYPE"
	case TypeKindDomain:
		stmt = t.generateDomainSQLWithSchema(targetSchema)
		objectType = "DOMAIN"
	default:
		return ""
	}

	// For comment header, use "-" if in target schema
	commentSchema := t.Schema
	if t.Schema == targetSchema {
		commentSchema = "-"
	}
	if includeComments {
		w.WriteStatementWithComment(objectType, t.Name, commentSchema, "-", stmt, "")
	} else {
		w.WriteString(stmt)
	}

	// Add comment if present
	if t.Comment != "" && includeComments {
		w.WriteDDLSeparator()
		var commentStmt string

		// Only include type name without schema if it's in the target schema
		var typeName string
		if t.Schema == targetSchema {
			typeName = t.Name
		} else {
			typeName = fmt.Sprintf("%s.%s", t.Schema, t.Name)
		}

		if t.Kind == TypeKindDomain {
			commentStmt = fmt.Sprintf("COMMENT ON DOMAIN %s IS '%s';", typeName, t.Comment)
			w.WriteStatementWithComment("COMMENT", "DOMAIN "+t.Name, commentSchema, "-", commentStmt, "")
		} else {
			commentStmt = fmt.Sprintf("COMMENT ON TYPE %s IS '%s';", typeName, t.Comment)
			w.WriteStatementWithComment("COMMENT", "TYPE "+t.Name, commentSchema, "-", commentStmt, "")
		}
	}

	return w.String()
}

// generateEnumSQLWithSchema generates CREATE TYPE ... AS ENUM statement with target schema context
func (t *Type) generateEnumSQLWithSchema(targetSchema string) string {
	var values []string
	for _, value := range t.EnumValues {
		values = append(values, fmt.Sprintf("    '%s'", value))
	}

	// Only include type name without schema if it's in the target schema
	var typeName string
	if t.Schema == targetSchema {
		typeName = t.Name
	} else {
		typeName = fmt.Sprintf("%s.%s", t.Schema, t.Name)
	}

	return fmt.Sprintf("CREATE TYPE %s AS ENUM (\n%s\n);",
		typeName, strings.Join(values, ",\n"))
}

// generateCompositeSQLWithSchema generates CREATE TYPE ... AS (...) statement with target schema context
func (t *Type) generateCompositeSQLWithSchema(targetSchema string) string {
	var columns []string
	for _, col := range t.Columns {
		columns = append(columns, fmt.Sprintf("\t%s %s", col.Name, col.DataType))
	}

	// Only include type name without schema if it's in the target schema
	var typeName string
	if t.Schema == targetSchema {
		typeName = t.Name
	} else {
		typeName = fmt.Sprintf("%s.%s", t.Schema, t.Name)
	}

	return fmt.Sprintf("CREATE TYPE %s AS (\n%s\n);",
		typeName, strings.Join(columns, ",\n"))
}

// generateDomainSQLWithSchema generates CREATE DOMAIN statement with target schema context
func (t *Type) generateDomainSQLWithSchema(targetSchema string) string {
	var parts []string

	// Only include domain name without schema if it's in the target schema
	var domainName string
	if t.Schema == targetSchema {
		domainName = t.Name
	} else {
		domainName = fmt.Sprintf("%s.%s", t.Schema, t.Name)
	}

	parts = append(parts, fmt.Sprintf("CREATE DOMAIN %s AS %s", domainName, t.BaseType))

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
