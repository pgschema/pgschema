package ir

import "fmt"

// Extension represents a PostgreSQL extension
type Extension struct {
	Name    string `json:"name"`
	Schema  string `json:"schema"`
	Version string `json:"version"`
	Comment string `json:"comment,omitempty"`
}

// GenerateSQL generates CREATE EXTENSION statement
func (e *Extension) GenerateSQL() string {
	w := NewSQLWriter()

	var stmt string
	if e.Schema != "" {
		stmt = fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s WITH SCHEMA %s;", e.Name, e.Schema)
	} else {
		stmt = fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s;", e.Name)
	}

	w.WriteStatementWithComment("EXTENSION", e.Name, "-", "-", stmt)

	// Add comment if present
	if e.Comment != "" {
		w.WriteDDLSeparator()
		commentStmt := fmt.Sprintf("COMMENT ON EXTENSION %s IS '%s';", e.Name, e.Comment)
		w.WriteStatementWithComment("COMMENT", "EXTENSION "+e.Name, "-", "-", commentStmt)
	}

	return w.String()
}
