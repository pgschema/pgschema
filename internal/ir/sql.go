package ir

import (
	"fmt"
	"strings"
)

// SQLGenerator interface for visitor pattern
type SQLGenerator interface {
	GenerateSQL() string
}

// SQLWriter is a helper for building SQL statements
type SQLWriter struct {
	output          strings.Builder
	includeComments bool
}

func NewSQLWriter() *SQLWriter {
	return &SQLWriter{includeComments: true}
}

func NewSQLWriterWithComments(includeComments bool) *SQLWriter {
	return &SQLWriter{includeComments: includeComments}
}

func (w *SQLWriter) WriteString(s string) {
	w.output.WriteString(s)
}

func (w *SQLWriter) WriteDDLSeparator() {
	w.output.WriteString("\n")
	w.output.WriteString("\n")
}

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

func (w *SQLWriter) String() string {
	return w.output.String()
}
