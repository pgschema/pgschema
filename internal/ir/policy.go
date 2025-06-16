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
	w := NewSQLWriter()
	policyStmt := fmt.Sprintf("CREATE POLICY %s ON %s.%s", p.Name, p.Schema, p.Table)

	// Add command type if specified
	if p.Command != PolicyCommandAll {
		policyStmt += fmt.Sprintf(" FOR %s", p.Command)
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
	w.WriteStatementWithComment("POLICY", fmt.Sprintf("%s %s", p.Table, p.Name), p.Schema, "", policyStmt)
	return w.String()
}