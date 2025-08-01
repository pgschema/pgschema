package diff

import (
	"strings"
)

// SQLContext provides context about the SQL statement being generated
type SQLContext struct {
	ObjectType   string // e.g., "table", "view", "function"
	Operation    string // e.g., "create", "alter", "drop"
	ObjectPath   string // e.g., "schema.table" or "schema.table.column"
	SourceChange any    // The DDLDiff element that generated this SQL
}

// PlanStep represents a single SQL statement with its source change
type PlanStep struct {
	SQL          string `json:"sql"`
	ObjectType   string `json:"object_type"`
	Operation    string `json:"operation"` // create, alter, drop
	ObjectPath   string `json:"object_path"`
	SourceChange any    `json:"source_change"`
}

// SQLCollector collects SQL statements with their context information
type SQLCollector struct {
	steps []PlanStep
}

// NewSQLCollector creates a new SQLCollector
func NewSQLCollector() *SQLCollector {
	return &SQLCollector{
		steps: []PlanStep{},
	}
}

// Collect collects a SQL statement with its context information
func (c *SQLCollector) Collect(context *SQLContext, stmt string) {
	if context != nil {
		step := PlanStep{
			SQL:          strings.TrimSpace(stmt),
			ObjectType:   context.ObjectType,
			Operation:    context.Operation,
			ObjectPath:   context.ObjectPath,
			SourceChange: context.SourceChange,
		}
		c.steps = append(c.steps, step)
	}
}

// GetSteps returns all collected plan steps
func (c *SQLCollector) GetSteps() []PlanStep {
	return c.steps
}
