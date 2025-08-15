package diff

import (
	"strings"
)

// diffContext provides context about the SQL statement being generated
type diffContext struct {
	Type                string // e.g., "table", "view", "function"
	Operation           string // e.g., "create", "alter", "drop"
	Path                string // e.g., "schema.table" or "schema.table.column"
	Source              any    // The ddlDiff element that generated this SQL
	CanRunInTransaction bool   // Whether this SQL can run in a transaction
}

// diffCollector collects SQL statements with their context information
type diffCollector struct {
	diffs []Diff
}

// newDiffCollector creates a new diffCollector
func newDiffCollector() *diffCollector {
	return &diffCollector{
		diffs: []Diff{},
	}
}

// collect collects a single SQL statement with its context information
func (c *diffCollector) collect(context *diffContext, stmt string) {
	if context != nil {
		cleanSQL := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(stmt), ";"))
		
		step := Diff{
			Statements: []SQLStatement{{
				SQL:                 cleanSQL,
				CanRunInTransaction: context.CanRunInTransaction,
			}},
			Type:      context.Type,
			Operation: context.Operation,
			Path:      context.Path,
			Source:    context.Source,
		}
		c.diffs = append(c.diffs, step)
	}
}

// collectMultipleStatements collects multiple related SQL statements as a single Diff
func (c *diffCollector) collectMultipleStatements(context *diffContext, statements []SQLStatement) {
	if context != nil && len(statements) > 0 {
		step := Diff{
			Statements: statements,
			Type:       context.Type,
			Operation:  context.Operation,
			Path:       context.Path,
			Source:     context.Source,
		}
		c.diffs = append(c.diffs, step)
	}
}
