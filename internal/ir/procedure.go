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
	if p.Definition == "<nil>" || p.Definition == "" {
		return ""
	}
	w := NewSQLWriter()
	
	// Build procedure signature for comment header (types only with schema qualifiers)
	headerSig := fmt.Sprintf("%s(%s)", p.Name, p.Arguments)
	
	// Build full procedure signature for CREATE statement (with parameter names)
	var createSig string
	if p.Signature != "" && p.Signature != "<nil>" {
		createSig = fmt.Sprintf("%s(%s)", p.Name, p.Signature)
	} else {
		createSig = fmt.Sprintf("%s(%s)", p.Name, p.Arguments)
	}
	
	// Generate CREATE PROCEDURE statement
	stmt := fmt.Sprintf("CREATE PROCEDURE %s.%s\n    LANGUAGE %s\n    AS $$%s$$;",
		p.Schema, createSig, strings.ToLower(p.Language), p.Definition)
	w.WriteStatementWithComment("PROCEDURE", headerSig, p.Schema, "", stmt)
	
	// Generate COMMENT ON PROCEDURE statement if comment exists
	if p.Comment != "" && p.Comment != "<nil>" {
		w.WriteDDLSeparator()
		
		// Escape single quotes in comment
		escapedComment := strings.ReplaceAll(p.Comment, "'", "''")
		commentStmt := fmt.Sprintf("COMMENT ON PROCEDURE %s.%s IS '%s';", p.Schema, headerSig, escapedComment)
		w.WriteStatementWithComment("COMMENT", "PROCEDURE "+headerSig, p.Schema, "", commentStmt)
	}
	
	return w.String()
}