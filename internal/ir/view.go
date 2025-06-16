package ir

import (
	"fmt"
)

// View represents a database view
type View struct {
	Schema       string            `json:"schema"`
	Name         string            `json:"name"`
	Definition   string            `json:"definition"`
	Dependencies []TableDependency `json:"dependencies"`
	Comment      string            `json:"comment,omitempty"`
}

// GenerateSQL for View
func (v *View) GenerateSQL() string {
	w := NewSQLWriter()
	// For now, use the definition as-is. Schema qualification will be handled at a higher level
	stmt := fmt.Sprintf("CREATE VIEW %s.%s AS\n%s;", v.Schema, v.Name, v.Definition)
	w.WriteStatementWithComment("VIEW", v.Name, v.Schema, "", stmt)
	return w.String()
}

// GenerateSQLWithSchemaContext generates SQL for a view with schema qualification
func (v *View) GenerateSQLWithSchemaContext(schemaIR *Schema) string {
	w := NewSQLWriter()
	stmt := fmt.Sprintf("CREATE VIEW %s.%s AS\n%s;", v.Schema, v.Name, v.Definition)
	w.WriteStatementWithComment("VIEW", v.Name, v.Schema, "", stmt)
	return w.String()
}