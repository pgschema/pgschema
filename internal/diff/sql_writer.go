package diff

import (
	"fmt"
	"strings"
)

// SQLWriter is a helper for building SQL statements with proper formatting
type SQLWriter struct {
	output          strings.Builder
	includeComments bool
}

// NewSQLWriter creates a new SQLWriter with comments enabled by default
func NewSQLWriter() *SQLWriter {
	return &SQLWriter{includeComments: true}
}

// NewSQLWriterWithComments creates a new SQLWriter with configurable comment inclusion
func NewSQLWriterWithComments(includeComments bool) *SQLWriter {
	return &SQLWriter{includeComments: includeComments}
}

// WriteString writes a string to the output
func (w *SQLWriter) WriteString(s string) {
	w.output.WriteString(s)
}

// WriteDDLSeparator writes DDL separator (two newlines)
func (w *SQLWriter) WriteDDLSeparator() {
	w.output.WriteString("\n")
	w.output.WriteString("\n")
}

// WriteStatementWithComment writes a SQL statement with optional comment header
func (w *SQLWriter) WriteStatementWithComment(objectType, objectName, schemaName, owner string, stmt string, targetSchema string) {
	if w.includeComments {
		w.output.WriteString("--\n")
		// For schema-agnostic dumps, use generic schema name in comments
		commentSchemaName := schemaName
		if targetSchema != "" && schemaName == targetSchema {
			commentSchemaName = "-"
		}
		if owner != "" {
			w.output.WriteString(fmt.Sprintf("-- Name: %s; Type: %s; Schema: %s; Owner: %s\n", objectName, objectType, commentSchemaName, owner))
		} else {
			w.output.WriteString(fmt.Sprintf("-- Name: %s; Type: %s; Schema: %s; Owner: -\n", objectName, objectType, commentSchemaName))
		}
		w.output.WriteString("--\n")
		w.output.WriteString("\n")
	}
	w.output.WriteString(stmt)
	w.output.WriteString("\n")
}

// String returns the accumulated SQL output
func (w *SQLWriter) String() string {
	return w.output.String()
}
