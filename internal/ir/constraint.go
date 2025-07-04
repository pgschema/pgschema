package ir

import (
	"fmt"
	"sort"
	"strings"
)

// Constraint represents a table constraint
type Constraint struct {
	Schema            string              `json:"schema"`
	Table             string              `json:"table"`
	Name              string              `json:"name"`
	Type              ConstraintType      `json:"type"`
	Columns           []*ConstraintColumn `json:"columns"`
	ReferencedSchema  string              `json:"referenced_schema,omitempty"`
	ReferencedTable   string              `json:"referenced_table,omitempty"`
	ReferencedColumns []*ConstraintColumn `json:"referenced_columns,omitempty"`
	CheckClause       string              `json:"check_clause,omitempty"`
	DeleteRule        string              `json:"delete_rule,omitempty"`
	UpdateRule        string              `json:"update_rule,omitempty"`
	Deferrable        bool                `json:"deferrable,omitempty"`
	InitiallyDeferred bool                `json:"initially_deferred,omitempty"`
	Comment           string              `json:"comment,omitempty"`
}

// ConstraintColumn represents a column within a constraint with its position
type ConstraintColumn struct {
	Name     string `json:"name"`
	Position int    `json:"position"` // ordinal_position within the constraint
}

// SortConstraintColumnsByPosition sorts constraint columns by their position
func (c *Constraint) SortConstraintColumnsByPosition() []*ConstraintColumn {
	columns := make([]*ConstraintColumn, len(c.Columns))
	copy(columns, c.Columns)
	sort.Slice(columns, func(i, j int) bool {
		return columns[i].Position < columns[j].Position
	})
	return columns
}

// GenerateSQL for Constraint with target schema context
func (c *Constraint) GenerateSQL(targetSchema string) string {
	return c.GenerateSQLWithOptions(true, targetSchema)
}

// GenerateSQLWithOptions generates SQL for a constraint with configurable comment inclusion
func (c *Constraint) GenerateSQLWithOptions(includeComments bool, targetSchema string) string {
	w := NewSQLWriterWithComments(includeComments)
	var stmt string

	switch c.Type {
	case ConstraintTypePrimaryKey, ConstraintTypeUnique:
		// Build constraint statement
		var constraintTypeStr string
		switch c.Type {
		case ConstraintTypePrimaryKey:
			constraintTypeStr = "PRIMARY KEY"
		case ConstraintTypeUnique:
			constraintTypeStr = "UNIQUE"
		}

		// Sort columns by position
		columns := c.SortConstraintColumnsByPosition()
		var columnNames []string
		for _, col := range columns {
			columnNames = append(columnNames, col.Name)
		}
		columnList := strings.Join(columnNames, ", ")

		// Use schema qualifier only if target schema is different
		if c.Schema != targetSchema {
			stmt = fmt.Sprintf("ALTER TABLE ONLY %s.%s\n    ADD CONSTRAINT %s %s (%s);",
				c.Schema, c.Table, c.Name, constraintTypeStr, columnList)
		} else {
			stmt = fmt.Sprintf("ALTER TABLE ONLY %s\n    ADD CONSTRAINT %s %s (%s);",
				c.Table, c.Name, constraintTypeStr, columnList)
		}

	case ConstraintTypeCheck:
		// Handle CHECK constraints
		// CheckClause already contains "CHECK (...)" from pg_get_constraintdef
		if c.Schema != targetSchema {
			stmt = fmt.Sprintf("ALTER TABLE ONLY %s.%s\n    ADD CONSTRAINT %s %s;",
				c.Schema, c.Table, c.Name, c.CheckClause)
		} else {
			stmt = fmt.Sprintf("ALTER TABLE ONLY %s\n    ADD CONSTRAINT %s %s;",
				c.Table, c.Name, c.CheckClause)
		}

	case ConstraintTypeForeignKey:
		// Sort columns by position
		columns := c.SortConstraintColumnsByPosition()
		var columnNames []string
		for _, col := range columns {
			columnNames = append(columnNames, col.Name)
		}
		columnList := strings.Join(columnNames, ", ")

		// Sort referenced columns by position
		var refColumnNames []string
		if len(c.ReferencedColumns) > 0 {
			refColumns := make([]*ConstraintColumn, len(c.ReferencedColumns))
			copy(refColumns, c.ReferencedColumns)
			sort.Slice(refColumns, func(i, j int) bool {
				return refColumns[i].Position < refColumns[j].Position
			})
			for _, col := range refColumns {
				refColumnNames = append(refColumnNames, col.Name)
			}
		}
		refColumnList := strings.Join(refColumnNames, ", ")

		// Build referential actions
		var actions []string
		if c.UpdateRule != "" && c.UpdateRule != "NO ACTION" {
			actions = append(actions, fmt.Sprintf("ON UPDATE %s", c.UpdateRule))
		}
		if c.DeleteRule != "" && c.DeleteRule != "NO ACTION" {
			actions = append(actions, fmt.Sprintf("ON DELETE %s", c.DeleteRule))
		}

		actionStr := ""
		if len(actions) > 0 {
			actionStr = " " + strings.Join(actions, " ")
		}

		// Build deferrable clause
		deferrableStr := ""
		if c.Deferrable {
			if c.InitiallyDeferred {
				deferrableStr = " DEFERRABLE INITIALLY DEFERRED"
			} else {
				deferrableStr = " DEFERRABLE"
			}
		}

		// Handle table qualification
		var tableRef string
		if c.Schema != targetSchema {
			tableRef = fmt.Sprintf("%s.%s", c.Schema, c.Table)
		} else {
			tableRef = c.Table
		}

		// Handle referenced table qualification - always qualify if different schema or cross-schema reference
		var refTableRef string
		if c.ReferencedSchema != targetSchema {
			refTableRef = fmt.Sprintf("%s.%s", c.ReferencedSchema, c.ReferencedTable)
		} else {
			refTableRef = c.ReferencedTable
		}

		stmt = fmt.Sprintf("ALTER TABLE ONLY %s\n    ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s)%s%s;",
			tableRef, c.Name, columnList, refTableRef, refColumnList, actionStr, deferrableStr)

	default:
		return "" // Unsupported constraint type
	}

	constraintTypeStr := "CONSTRAINT"
	if c.Type == ConstraintTypeForeignKey {
		constraintTypeStr = "FK CONSTRAINT"
	}

	if includeComments {
		w.WriteStatementWithComment(constraintTypeStr, fmt.Sprintf("%s %s", c.Table, c.Name), c.Schema, "", stmt, targetSchema)
	} else {
		w.WriteString(stmt)
	}
	return w.String()
}
