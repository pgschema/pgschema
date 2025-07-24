package diff

import (
	"fmt"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

// generateConstraintSQL generates constraint definition for inline table constraints
func generateConstraintSQL(constraint *ir.Constraint, targetSchema string) string {
	// Helper function to get column names from ConstraintColumn array
	getColumnNames := func(columns []*ir.ConstraintColumn) []string {
		var names []string
		for _, col := range columns {
			names = append(names, col.Name)
		}
		return names
	}

	switch constraint.Type {
	case ir.ConstraintTypePrimaryKey:
		return fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(getColumnNames(constraint.Columns), ", "))
	case ir.ConstraintTypeUnique:
		return fmt.Sprintf("UNIQUE (%s)", strings.Join(getColumnNames(constraint.Columns), ", "))
	case ir.ConstraintTypeForeignKey:
		stmt := fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s (%s)",
			strings.Join(getColumnNames(constraint.Columns), ", "),
			constraint.ReferencedTable, strings.Join(getColumnNames(constraint.ReferencedColumns), ", "))
		// Only add ON DELETE/UPDATE if they are not the default "NO ACTION"
		if constraint.DeleteRule != "" && constraint.DeleteRule != "NO ACTION" {
			stmt += fmt.Sprintf(" ON DELETE %s", constraint.DeleteRule)
		}
		if constraint.UpdateRule != "" && constraint.UpdateRule != "NO ACTION" {
			stmt += fmt.Sprintf(" ON UPDATE %s", constraint.UpdateRule)
		}
		return stmt
	case ir.ConstraintTypeCheck:
		return constraint.CheckClause
	default:
		return ""
	}
}

// getInlineConstraintsForTable returns constraints in the correct order: PRIMARY KEY, UNIQUE, FOREIGN KEY
func getInlineConstraintsForTable(table *ir.Table) []*ir.Constraint {
	var inlineConstraints []*ir.Constraint

	// Get constraint names sorted for consistent output (sorting handled by IR)
	constraintNames := sortedKeys(table.Constraints)

	// Separate constraints by type for proper ordering
	var primaryKeys []*ir.Constraint
	var uniques []*ir.Constraint
	var foreignKeys []*ir.Constraint
	var checkConstraints []*ir.Constraint

	for _, constraintName := range constraintNames {
		constraint := table.Constraints[constraintName]

		// Categorize constraints by type
		switch constraint.Type {
		case ir.ConstraintTypePrimaryKey:
			// Only include multi-column primary keys as inline constraints
			// Single-column primary keys are handled inline with the column definition
			if len(constraint.Columns) > 1 {
				primaryKeys = append(primaryKeys, constraint)
			}
		case ir.ConstraintTypeUnique:
			uniques = append(uniques, constraint)
		case ir.ConstraintTypeForeignKey:
			foreignKeys = append(foreignKeys, constraint)
		case ir.ConstraintTypeCheck:
			// Only include table-level CHECK constraints (not column-level ones)
			// Column-level CHECK constraints are handled inline with the column definition
			if len(constraint.Columns) != 1 {
				checkConstraints = append(checkConstraints, constraint)
			}
		}
	}

	// Add constraints in order: PRIMARY KEY, UNIQUE, FOREIGN KEY, CHECK
	inlineConstraints = append(inlineConstraints, primaryKeys...)
	inlineConstraints = append(inlineConstraints, uniques...)
	inlineConstraints = append(inlineConstraints, foreignKeys...)
	inlineConstraints = append(inlineConstraints, checkConstraints...)

	return inlineConstraints
}
