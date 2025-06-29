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
	output         strings.Builder
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

func (w *SQLWriter) WriteComment(objectType, objectName, schemaName, owner string) {
	if !w.includeComments {
		return
	}
	w.output.WriteString("--\n")
	if owner != "" {
		w.output.WriteString(fmt.Sprintf("-- Name: %s; Type: %s; Schema: %s; Owner: %s\n", objectName, objectType, schemaName, owner))
	} else {
		w.output.WriteString(fmt.Sprintf("-- Name: %s; Type: %s; Schema: %s; Owner: -\n", objectName, objectType, schemaName))
	}
	w.output.WriteString("--\n")
}

func (w *SQLWriter) WriteStatementWithComment(objectType, objectName, schemaName, owner string, stmt string) {
	if w.includeComments {
		w.WriteComment(objectType, objectName, schemaName, owner)
		w.output.WriteString("\n")
	}
	w.output.WriteString(stmt)
	w.output.WriteString("\n")
}

func (w *SQLWriter) String() string {
	return w.output.String()
}