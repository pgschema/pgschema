package ir

import (
	"fmt"
	"strings"
)

// Procedure represents a database procedure
type Procedure struct {
	Schema     string       `json:"schema"`
	Name       string       `json:"name"`
	Definition string       `json:"definition"`
	Language   string       `json:"language"`
	Arguments  string       `json:"arguments,omitempty"`
	Signature  string       `json:"signature,omitempty"`
	Parameters []*Parameter `json:"parameters,omitempty"`
	Comment    string       `json:"comment,omitempty"`
}

// GenerateSQL for Procedure
func (p *Procedure) GenerateSQL() string {
	return p.GenerateSQLWithSchema(p.Schema)
}

// GenerateSQLWithSchema for Procedure with target schema context
func (p *Procedure) GenerateSQLWithSchema(targetSchema string) string {
	return p.GenerateSQLWithOptions(true, targetSchema)
}

// GenerateSQLWithOptions generates SQL for a procedure with configurable comment inclusion
func (p *Procedure) GenerateSQLWithOptions(includeComments bool, targetSchema string) string {
	if p.Definition == "<nil>" || p.Definition == "" {
		return ""
	}
	w := NewSQLWriterWithComments(includeComments)

	// Build procedure signature for comment header (types only with schema qualifiers)
	headerSig := fmt.Sprintf("%s(%s)", p.Name, p.Arguments)

	// Build full procedure signature for CREATE statement (with parameter names)
	var createSig string
	if p.Signature != "" && p.Signature != "<nil>" {
		createSig = fmt.Sprintf("%s(%s)", p.Name, p.Signature)
	} else {
		createSig = fmt.Sprintf("%s(%s)", p.Name, p.Arguments)
	}

	// Only include procedure name without schema if it's in the target schema
	var procName string
	if p.Schema == targetSchema {
		procName = createSig
	} else {
		procName = fmt.Sprintf("%s.%s", p.Schema, createSig)
	}

	// Generate CREATE PROCEDURE statement
	stmt := fmt.Sprintf("CREATE PROCEDURE %s\n    LANGUAGE %s\n    AS $$%s$$;",
		procName, strings.ToLower(p.Language), p.Definition)

	// For comment header, use "-" if in target schema
	commentSchema := p.Schema
	if p.Schema == targetSchema {
		commentSchema = "-"
	}
	if includeComments {
		w.WriteStatementWithComment("PROCEDURE", headerSig, commentSchema, "", stmt, "")
	} else {
		w.WriteString(stmt)
	}

	// Generate COMMENT ON PROCEDURE statement if comment exists
	if p.Comment != "" && p.Comment != "<nil>" && includeComments {
		w.WriteDDLSeparator()

		// Escape single quotes in comment
		escapedComment := strings.ReplaceAll(p.Comment, "'", "''")

		// Only include procedure name without schema if it's in the target schema
		var procRef string
		if p.Schema == targetSchema {
			procRef = headerSig
		} else {
			procRef = fmt.Sprintf("%s.%s", p.Schema, headerSig)
		}
		commentStmt := fmt.Sprintf("COMMENT ON PROCEDURE %s IS '%s';", procRef, escapedComment)
		w.WriteStatementWithComment("COMMENT", "PROCEDURE "+headerSig, commentSchema, "", commentStmt, "")
	}

	return w.String()
}
