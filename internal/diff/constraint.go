package diff

import (
	"fmt"
	"strings"

	"github.com/pgschema/pgschema/ir"
)

// generateConstraintSQL generates constraint definition for inline table constraints
func generateConstraintSQL(constraint *ir.Constraint, _ string) string {
	// Helper function to get column names from ConstraintColumn array
	getColumnNames := func(columns []*ir.ConstraintColumn) []string {
		var names []string
		for _, col := range columns {
			names = append(names, ir.QuoteIdentifier(col.Name))
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
			ir.QuoteIdentifier(constraint.ReferencedTable), strings.Join(getColumnNames(constraint.ReferencedColumns), ", "))
		// Only add ON UPDATE/DELETE if they are not the default "NO ACTION"
		if constraint.UpdateRule != "" && constraint.UpdateRule != "NO ACTION" {
			stmt += fmt.Sprintf(" ON UPDATE %s", constraint.UpdateRule)
		}
		if constraint.DeleteRule != "" && constraint.DeleteRule != "NO ACTION" {
			stmt += fmt.Sprintf(" ON DELETE %s", constraint.DeleteRule)
		}
		return stmt
	case ir.ConstraintTypeCheck:
		// Generate CHECK constraint with proper NOT VALID placement
		// The CheckClause is normalized to exclude NOT VALID (stripped in normalize.go)
		// We append NOT VALID based on IsValid field, mimicking pg_dump behavior
		result := fmt.Sprintf("CONSTRAINT %s %s", constraint.Name, constraint.CheckClause)
		if !constraint.IsValid {
			result += " NOT VALID"
		}
		return result
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
			// Only include multi-column unique constraints as inline constraints
			// Single-column unique constraints are handled inline with the column definition
			if len(constraint.Columns) > 1 {
				uniques = append(uniques, constraint)
			}
		case ir.ConstraintTypeForeignKey:
			// Only include multi-column foreign key constraints as inline constraints
			// Single-column foreign key constraints are handled inline with the column definition
			if len(constraint.Columns) > 1 {
				foreignKeys = append(foreignKeys, constraint)
			}
		case ir.ConstraintTypeCheck:
			// Always include ALL CHECK constraints as table-level named constraints
			// This eliminates complexity and makes constraints explicit and manageable
			checkConstraints = append(checkConstraints, constraint)
		}
	}

	// Add constraints in order: PRIMARY KEY, UNIQUE, FOREIGN KEY, CHECK
	inlineConstraints = append(inlineConstraints, primaryKeys...)
	inlineConstraints = append(inlineConstraints, uniques...)
	inlineConstraints = append(inlineConstraints, foreignKeys...)
	inlineConstraints = append(inlineConstraints, checkConstraints...)

	return inlineConstraints
}

// constraintsEqual compares two constraints for equality
func constraintsEqual(old, new *ir.Constraint) bool {
	// Basic properties
	if old.Name != new.Name {
		return false
	}
	if old.Type != new.Type {
		return false
	}
	if old.ReferencedSchema != new.ReferencedSchema {
		return false
	}
	if old.ReferencedTable != new.ReferencedTable {
		return false
	}
	if old.CheckClause != new.CheckClause {
		return false
	}
	
	// Foreign key specific properties (this is the key fix!)
	if old.DeleteRule != new.DeleteRule {
		return false
	}
	if old.UpdateRule != new.UpdateRule {
		return false
	}
	if old.Deferrable != new.Deferrable {
		return false
	}
	if old.InitiallyDeferred != new.InitiallyDeferred {
		return false
	}

	// Validation status - only compare for CHECK and FOREIGN KEY constraints
	// PRIMARY KEY and UNIQUE constraints are always valid (IsValid is not meaningful for them)
	if old.Type == ir.ConstraintTypeCheck || old.Type == ir.ConstraintTypeForeignKey {
		if old.IsValid != new.IsValid {
			return false
		}
	}

	// Comments
	if old.Comment != new.Comment {
		return false
	}
	
	// Compare columns (skip for CHECK constraints as column detection may differ between parser and inspector)
	if old.Type != ir.ConstraintTypeCheck {
		if len(old.Columns) != len(new.Columns) {
			return false
		}
		for i, oldCol := range old.Columns {
			newCol := new.Columns[i]
			if oldCol.Name != newCol.Name || oldCol.Position != newCol.Position {
				return false
			}
		}
	}
	
	// Compare referenced columns
	if len(old.ReferencedColumns) != len(new.ReferencedColumns) {
		return false
	}
	for i, oldCol := range old.ReferencedColumns {
		newCol := new.ReferencedColumns[i]
		if oldCol.Name != newCol.Name || oldCol.Position != newCol.Position {
			return false
		}
	}
	
	return true
}
