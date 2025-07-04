package ir

import (
	"fmt"
)

// RLSPolicy represents a Row Level Security policy
type RLSPolicy struct {
	Schema     string        `json:"schema"`
	Table      string        `json:"table"`
	Name       string        `json:"name"`
	Command    PolicyCommand `json:"command"` // SELECT, INSERT, UPDATE, DELETE, ALL
	Permissive bool          `json:"permissive"`
	Roles      []string      `json:"roles,omitempty"`
	Using      string        `json:"using,omitempty"`      // USING expression
	WithCheck  string        `json:"with_check,omitempty"` // WITH CHECK expression
	Comment    string        `json:"comment,omitempty"`
}

// GenerateSQL for RLSPolicy
func (p *RLSPolicy) GenerateSQL() string {
	return p.GenerateSQLWithSchema(p.Schema)
}

// GenerateSQLWithSchema generates SQL for a policy with target schema context
func (p *RLSPolicy) GenerateSQLWithSchema(targetSchema string) string {
	return p.GenerateSQLWithOptions(true, targetSchema)
}

// GenerateSQLWithOptions generates SQL for a policy with configurable comment inclusion
func (p *RLSPolicy) GenerateSQLWithOptions(includeComments bool, targetSchema string) string {
	w := NewSQLWriterWithComments(includeComments)

	// Only include table name without schema if it's in the target schema
	var tableName string
	if p.Schema == targetSchema {
		tableName = p.Table
	} else {
		tableName = fmt.Sprintf("%s.%s", p.Schema, p.Table)
	}

	policyStmt := fmt.Sprintf("CREATE POLICY %s ON %s", p.Name, tableName)

	// Add command type if specified
	if p.Command != PolicyCommandAll {
		policyStmt += fmt.Sprintf(" FOR %s", p.Command)
	}

	// Add roles if specified
	if len(p.Roles) > 0 {
		policyStmt += " TO "
		for i, role := range p.Roles {
			if i > 0 {
				policyStmt += ", "
			}
			policyStmt += role
		}
	}

	// Add USING clause if present
	if p.Using != "" {
		policyStmt += fmt.Sprintf(" USING (%s)", p.Using)
	}

	// Add WITH CHECK clause if present
	if p.WithCheck != "" {
		policyStmt += fmt.Sprintf(" WITH CHECK (%s)", p.WithCheck)
	}

	policyStmt += ";"

	// For comment header, use "-" if in target schema
	commentSchema := p.Schema
	if p.Schema == targetSchema {
		commentSchema = "-"
	}
	if includeComments {
		w.WriteStatementWithComment("POLICY", fmt.Sprintf("%s %s", p.Table, p.Name), commentSchema, "", policyStmt, "")
	} else {
		w.WriteString(policyStmt)
	}
	return w.String()
}
