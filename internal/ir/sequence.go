package ir

import (
	"fmt"
	"strings"
)

// Sequence represents a database sequence
type Sequence struct {
	Schema        string `json:"schema"`
	Name          string `json:"name"`
	DataType      string `json:"data_type"`
	StartValue    int64  `json:"start_value"`
	MinValue      *int64 `json:"min_value,omitempty"`
	MaxValue      *int64 `json:"max_value,omitempty"`
	Increment     int64  `json:"increment"`
	CycleOption   bool   `json:"cycle_option"`
	OwnedByTable  string `json:"owned_by_table,omitempty"`
	OwnedByColumn string `json:"owned_by_column,omitempty"`
	Comment       string `json:"comment,omitempty"`
}

// GenerateSQL for Sequence (CREATE SEQUENCE only)
func (s *Sequence) GenerateSQL(targetSchema string) string {
	return s.GenerateSQLWithOptions(true, targetSchema)
}

// GenerateSQLWithOptions for Sequence with configurable comment inclusion
func (s *Sequence) GenerateSQLWithOptions(includeComments bool, targetSchema string) string {
	w := NewSQLWriterWithComments(includeComments)

	// Build sequence statement
	var stmt strings.Builder
	// Use schema qualifier only if target schema is different
	if s.Schema != targetSchema {
		stmt.WriteString(fmt.Sprintf("CREATE SEQUENCE %s.%s\n", s.Schema, s.Name))
	} else {
		stmt.WriteString(fmt.Sprintf("CREATE SEQUENCE %s\n", s.Name))
	}
	if s.DataType != "" && s.DataType != "bigint" {
		stmt.WriteString(fmt.Sprintf("    AS %s\n", s.DataType))
	}
	stmt.WriteString(fmt.Sprintf("    START WITH %d\n", s.StartValue))
	stmt.WriteString(fmt.Sprintf("    INCREMENT BY %d\n", s.Increment))

	if s.MinValue != nil {
		stmt.WriteString(fmt.Sprintf("    MINVALUE %d\n", *s.MinValue))
	} else {
		stmt.WriteString("    NO MINVALUE\n")
	}

	if s.MaxValue != nil {
		stmt.WriteString(fmt.Sprintf("    MAXVALUE %d\n", *s.MaxValue))
	} else {
		stmt.WriteString("    NO MAXVALUE\n")
	}

	stmt.WriteString("    CACHE 1")
	if s.CycleOption {
		stmt.WriteString("\n    CYCLE")
	}
	stmt.WriteString(";")

	if includeComments {
		w.WriteStatementWithComment("SEQUENCE", s.Name, s.Schema, "", stmt.String(), targetSchema)
	} else {
		w.WriteString(stmt.String())
	}
	return w.String()
}
