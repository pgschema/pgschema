package diff

import (
	"strings"
)

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
