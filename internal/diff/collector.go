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

// Collect collects a SQL statement with its context information
func (c *diffCollector) collect(context *diffContext, stmt string) {
	if context != nil {
		step := Diff{
			SQL:                 strings.TrimSpace(stmt),
			Type:                context.Type,
			Operation:           context.Operation,
			Path:                context.Path,
			Source:              context.Source,
			CanRunInTransaction: context.CanRunInTransaction,
		}
		c.diffs = append(c.diffs, step)
	}
}
