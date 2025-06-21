package ir

import (
	"fmt"
	"strings"
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
	
	// Generate COMMENT ON TABLE statement for view if comment exists
	if v.Comment != "" && v.Comment != "<nil>" {
		w.WriteDDLSeparator()
		
		// Escape single quotes in comment
		escapedComment := strings.ReplaceAll(v.Comment, "'", "''")
		commentStmt := fmt.Sprintf("COMMENT ON TABLE %s.%s IS '%s';", v.Schema, v.Name, escapedComment)
		w.WriteStatementWithComment("COMMENT", "TABLE "+v.Name, v.Schema, "", commentStmt)
	}
	
	return w.String()
}

// GenerateSQLWithSchemaContext generates SQL for a view with schema qualification
func (v *View) GenerateSQLWithSchemaContext(schemaIR *Schema) string {
	w := NewSQLWriter()
	stmt := fmt.Sprintf("CREATE VIEW %s.%s AS\n%s;", v.Schema, v.Name, v.Definition)
	w.WriteStatementWithComment("VIEW", v.Name, v.Schema, "", stmt)
	
	// Generate COMMENT ON TABLE statement for view if comment exists
	if v.Comment != "" && v.Comment != "<nil>" {
		w.WriteDDLSeparator()
		
		// Escape single quotes in comment
		escapedComment := strings.ReplaceAll(v.Comment, "'", "''")
		commentStmt := fmt.Sprintf("COMMENT ON TABLE %s.%s IS '%s';", v.Schema, v.Name, escapedComment)
		w.WriteStatementWithComment("COMMENT", "TABLE "+v.Name, v.Schema, "", commentStmt)
	}
	
	return w.String()
}