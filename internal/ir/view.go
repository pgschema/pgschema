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
	return v.GenerateSQLWithSchema(v.Schema)
}

// GenerateSQLWithSchema generates SQL for a view with target schema context
func (v *View) GenerateSQLWithSchema(targetSchema string) string {
	w := NewSQLWriter()
	
	// Only include view name without schema if it's in the target schema
	var viewName string
	if v.Schema == targetSchema {
		viewName = v.Name
	} else {
		viewName = fmt.Sprintf("%s.%s", v.Schema, v.Name)
	}
	
	stmt := fmt.Sprintf("CREATE VIEW %s AS\n%s", viewName, v.Definition)
	
	// For comment header, use "-" if in target schema
	commentSchema := v.Schema
	if v.Schema == targetSchema {
		commentSchema = "-"
	}
	w.WriteStatementWithComment("VIEW", v.Name, commentSchema, "", stmt, "")

	// Generate COMMENT ON TABLE statement for view if comment exists
	if v.Comment != "" && v.Comment != "<nil>" {
		w.WriteDDLSeparator()

		// Escape single quotes in comment
		escapedComment := strings.ReplaceAll(v.Comment, "'", "''")
		
		// Only include view name without schema if it's in the target schema
		var viewRef string
		if v.Schema == targetSchema {
			viewRef = v.Name
		} else {
			viewRef = fmt.Sprintf("%s.%s", v.Schema, v.Name)
		}
		commentStmt := fmt.Sprintf("COMMENT ON TABLE %s IS '%s';", viewRef, escapedComment)
		w.WriteStatementWithComment("COMMENT", "TABLE "+v.Name, commentSchema, "", commentStmt, "")
	}

	return w.String()
}

// GenerateSQLWithSchemaContext generates SQL for a view with schema qualification
// Deprecated: Use GenerateSQLWithSchema instead
func (v *View) GenerateSQLWithSchemaContext(schemaIR *Schema) string {
	return v.GenerateSQLWithSchema(v.Schema)
}