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
)

// TypeColumn represents a column in a composite type
type TypeColumn struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
	Position int    `json:"position"`
}

// Type represents a PostgreSQL user-defined type
type Type struct {
	Schema     string        `json:"schema"`
	Name       string        `json:"name"`
	Kind       TypeKind      `json:"kind"`
	Comment    string        `json:"comment,omitempty"`
	EnumValues []string      `json:"enum_values,omitempty"` // For ENUM types
	Columns    []*TypeColumn `json:"columns,omitempty"`     // For composite types
}

// GenerateSQL generates CREATE TYPE statement
func (t *Type) GenerateSQL() string {
	w := NewSQLWriter()

	var stmt string
	switch t.Kind {
	case TypeKindEnum:
		stmt = t.generateEnumSQL()
	case TypeKindComposite:
		stmt = t.generateCompositeSQL()
	default:
		return ""
	}

	w.WriteStatementWithComment("TYPE", t.Name, t.Schema, "-", stmt)

	// Add comment if present
	if t.Comment != "" {
		w.WriteDDLSeparator()
		commentStmt := fmt.Sprintf("COMMENT ON TYPE %s.%s IS '%s';", t.Schema, t.Name, t.Comment)
		w.WriteStatementWithComment("COMMENT", "TYPE "+t.Name, t.Schema, "-", commentStmt)
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
