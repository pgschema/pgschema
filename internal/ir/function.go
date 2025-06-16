package ir

import (
	"fmt"
	"strings"
)

// Function represents a database function
type Function struct {
	Schema     string       `json:"schema"`
	Name       string       `json:"name"`
	Definition string       `json:"definition"`
	ReturnType string       `json:"return_type"`
	Language   string       `json:"language"`
	Parameters []*Parameter `json:"parameters,omitempty"`
	Comment    string       `json:"comment,omitempty"`
}

// Parameter represents a function parameter
type Parameter struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
	Mode     string `json:"mode"` // IN, OUT, INOUT
	Position int    `json:"position"`
}

// GenerateSQL for Function
func (f *Function) GenerateSQL() string {
	if f.Definition == "<nil>" || f.Definition == "" {
		return ""
	}
	w := NewSQLWriter()
	stmt := fmt.Sprintf("CREATE FUNCTION %s.%s() RETURNS %s\n    LANGUAGE %s\n    AS $$%s$$;",
		f.Schema, f.Name, f.ReturnType, strings.ToLower(f.Language), f.Definition)
	w.WriteStatementWithComment("FUNCTION", fmt.Sprintf("%s()", f.Name), f.Schema, "", stmt)
	return w.String()
}