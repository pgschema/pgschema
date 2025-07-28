package diff

import (
	"fmt"
	"strings"
)

// SingleFileWriter is a helper for building SQL statements with proper formatting for single file output
type SingleFileWriter struct {
	output          strings.Builder
	includeComments bool
}

// NewSingleFileWriter creates a new SingleFileWriter with configurable comment inclusion
func NewSingleFileWriter(includeComments bool) *SingleFileWriter {
	return &SingleFileWriter{includeComments: includeComments}
}

// WriteString writes a string to the output
func (w *SingleFileWriter) WriteString(s string) {
	w.output.WriteString(s)
}

// WriteDDLSeparator writes DDL separator (two newlines)
func (w *SingleFileWriter) WriteDDLSeparator() {
	w.output.WriteString("\n")
}

// WriteStatementWithComment writes a SQL statement with optional comment header
func (w *SingleFileWriter) WriteStatementWithComment(objectType, objectName, schemaName, owner string, stmt string, targetSchema string) {
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

// String returns the accumulated SQL output with leading/trailing newlines removed
func (w *SingleFileWriter) String() string {
	result := w.output.String()
	return strings.Trim(result, "\n")
}
