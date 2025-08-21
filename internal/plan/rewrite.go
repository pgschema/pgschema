package plan

import (
	"fmt"
	"strings"

	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/ir"
)

// RewriteStep represents a single step in a rewrite operation
type RewriteStep struct {
	SQL                 string     `json:"sql,omitempty"`
	CanRunInTransaction bool       `json:"can_run_in_transaction"`
	Directive           *Directive `json:"directive,omitempty"`
}

// generateRewrite generates rewrite steps for a diff if online operations are enabled
func generateRewrite(d diff.Diff) []RewriteStep {
	// Dispatch to specific rewrite generators based on diff type and source
	switch d.Type {
	case diff.DiffTypeTableIndex:
		switch d.Operation {
		case diff.DiffOperationCreate:
			if index, ok := d.Source.(*ir.Index); ok {
				return generateIndexRewrite(index)
			}
		case diff.DiffOperationAlter:
			// For index changes, the source might be an IndexDiff or could be an Index for replacement
			if indexDiff, ok := d.Source.(*diff.IndexDiff); ok {
				return generateIndexChangeRewrite(indexDiff)
			} else if index, ok := d.Source.(*ir.Index); ok {
				// This handles index replacements where the source is the new index
				return generateIndexChangeRewriteFromIndex(index)
			}
		}
	case diff.DiffTypeTableConstraint:
		if d.Operation == diff.DiffOperationCreate {
			if constraint, ok := d.Source.(*ir.Constraint); ok {
				switch constraint.Type {
				case ir.ConstraintTypeCheck:
					return generateConstraintRewrite(constraint)
				case ir.ConstraintTypeForeignKey:
					return generateForeignKeyRewrite(constraint)
				}
			}
		}
	case diff.DiffTypeTableColumn:
		if d.Operation == diff.DiffOperationAlter {
			if columnDiff, ok := d.Source.(*diff.ColumnDiff); ok {
				// Check if this is a NOT NULL addition
				if columnDiff.Old.IsNullable && !columnDiff.New.IsNullable {
					return generateColumnNotNullRewrite(columnDiff, d.Path)
				}
			}
		}
	}

	return nil
}

// generateIndexRewrite generates rewrite steps for CREATE INDEX operations
func generateIndexRewrite(index *ir.Index) []RewriteStep {
	// Generate concurrent SQL
	concurrentSQL := generateIndexSQL(index, true) // With CONCURRENTLY
	waitSQL := generateIndexWaitQueryWithName(index.Name)

	return []RewriteStep{
		{
			SQL:                 concurrentSQL,
			CanRunInTransaction: false, // CONCURRENTLY cannot run in transaction
		},
		{
			SQL:                 waitSQL,
			CanRunInTransaction: true,
			Directive: &Directive{
				Type:    "wait",
				Message: fmt.Sprintf("Creating index %s", index.Name),
			},
		},
	}
}

// generateIndexChangeRewriteFromIndex generates rewrite steps for index replacement when source is new index
func generateIndexChangeRewriteFromIndex(index *ir.Index) []RewriteStep {
	// For index replacements, we need to create new index, wait, drop old, rename
	tempIndexName := index.Name + "_pgschema_new"

	// Create temporary index with new definition
	tempIndex := *index
	tempIndex.Name = tempIndexName
	concurrentSQL := generateIndexSQL(&tempIndex, true)
	waitSQL := generateIndexWaitQueryWithName(tempIndexName)

	// Drop old index and rename new one
	dropSQL := fmt.Sprintf("DROP INDEX %s;", index.Name)
	renameSQL := fmt.Sprintf("ALTER INDEX %s RENAME TO %s;", tempIndexName, index.Name)

	return []RewriteStep{
		{
			SQL:                 concurrentSQL,
			CanRunInTransaction: false,
		},
		{
			SQL:                 waitSQL,
			CanRunInTransaction: true,
			Directive: &Directive{
				Type:    "wait",
				Message: fmt.Sprintf("Creating index %s", tempIndexName),
			},
		},
		{
			SQL:                 dropSQL,
			CanRunInTransaction: true,
		},
		{
			SQL:                 renameSQL,
			CanRunInTransaction: true,
		},
	}
}

// generateIndexChangeRewrite generates rewrite steps for index modifications
func generateIndexChangeRewrite(indexDiff *diff.IndexDiff) []RewriteStep {
	// For index changes, we need to create new index, wait, drop old, rename
	tempIndexName := indexDiff.New.Name + "_pgschema_new"

	// Create temporary index with new definition
	tempIndex := *indexDiff.New
	tempIndex.Name = tempIndexName
	concurrentSQL := generateIndexSQL(&tempIndex, true)
	waitSQL := generateIndexWaitQueryWithName(tempIndexName)

	// Drop old index and rename new one
	dropSQL := fmt.Sprintf("DROP INDEX %s;", indexDiff.Old.Name)
	renameSQL := fmt.Sprintf("ALTER INDEX %s RENAME TO %s;", tempIndexName, indexDiff.New.Name)

	return []RewriteStep{
		{
			SQL:                 concurrentSQL,
			CanRunInTransaction: false,
		},
		{
			SQL:                 waitSQL,
			CanRunInTransaction: true,
			Directive: &Directive{
				Type:    "wait",
				Message: fmt.Sprintf("Creating index %s", tempIndexName),
			},
		},
		{
			SQL:                 dropSQL,
			CanRunInTransaction: true,
		},
		{
			SQL:                 renameSQL,
			CanRunInTransaction: true,
		},
	}
}

// generateConstraintRewrite generates rewrite steps for CHECK constraint operations
func generateConstraintRewrite(constraint *ir.Constraint) []RewriteStep {
	tableName := getTableNameWithSchema(constraint.Schema, constraint.Table)

	notValidSQL := fmt.Sprintf("ALTER TABLE %s\nADD CONSTRAINT %s %s NOT VALID;",
		tableName, constraint.Name, constraint.CheckClause)
	validateSQL := fmt.Sprintf("ALTER TABLE %s VALIDATE CONSTRAINT %s;",
		tableName, constraint.Name)

	return []RewriteStep{
		{
			SQL:                 notValidSQL,
			CanRunInTransaction: true,
		},
		{
			SQL:                 validateSQL,
			CanRunInTransaction: true,
		},
	}
}

// generateForeignKeyRewrite generates rewrite steps for FOREIGN KEY constraint operations
func generateForeignKeyRewrite(constraint *ir.Constraint) []RewriteStep {
	tableName := getTableNameWithSchema(constraint.Schema, constraint.Table)

	// Build foreign key clause
	var columnNames []string
	for _, col := range constraint.Columns {
		columnNames = append(columnNames, col.Name)
	}

	var refColumnNames []string
	for _, col := range constraint.ReferencedColumns {
		refColumnNames = append(refColumnNames, col.Name)
	}

	refTableName := getTableNameWithSchema(constraint.ReferencedSchema, constraint.ReferencedTable)

	fkClause := fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s (%s)",
		joinStrings(columnNames, ", "),
		refTableName,
		joinStrings(refColumnNames, ", "))

	// Add ON UPDATE/DELETE clauses if specified (in correct order)
	if constraint.UpdateRule != "" && constraint.UpdateRule != "NO ACTION" {
		fkClause += fmt.Sprintf(" ON UPDATE %s", constraint.UpdateRule)
	}
	if constraint.DeleteRule != "" && constraint.DeleteRule != "NO ACTION" {
		fkClause += fmt.Sprintf(" ON DELETE %s", constraint.DeleteRule)
	}

	// Add DEFERRABLE clauses if specified
	if constraint.Deferrable {
		fkClause += " DEFERRABLE"
		if constraint.InitiallyDeferred {
			fkClause += " INITIALLY DEFERRED"
		}
	}

	notValidSQL := fmt.Sprintf("ALTER TABLE %s\nADD CONSTRAINT %s %s NOT VALID;",
		tableName, constraint.Name, fkClause)
	validateSQL := fmt.Sprintf("ALTER TABLE %s VALIDATE CONSTRAINT %s;",
		tableName, constraint.Name)

	return []RewriteStep{
		{
			SQL:                 notValidSQL,
			CanRunInTransaction: true,
		},
		{
			SQL:                 validateSQL,
			CanRunInTransaction: true,
		},
	}
}

// generateColumnNotNullRewrite generates rewrite steps for SET NOT NULL operations
func generateColumnNotNullRewrite(_ *diff.ColumnDiff, path string) []RewriteStep {
	// Parse path (schema.table.column) to extract schema, table, and column names
	parts := strings.Split(path, ".")
	if len(parts) != 3 {
		// Fallback: should not happen, but return empty if path format is unexpected
		return nil
	}

	schema := parts[0]
	table := parts[1]
	column := parts[2]

	tableName := getTableNameWithSchema(schema, table)
	constraintName := fmt.Sprintf("%s_not_null", column)

	// Step 1: Add CHECK constraint with NOT VALID
	addConstraintSQL := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s CHECK (%s IS NOT NULL) NOT VALID;",
		tableName, constraintName, column)

	// Step 2: Validate the constraint
	validateConstraintSQL := fmt.Sprintf("ALTER TABLE %s VALIDATE CONSTRAINT %s;",
		tableName, constraintName)

	// Step 3: Set column to NOT NULL
	setNotNullSQL := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;",
		tableName, column)

	return []RewriteStep{
		{
			SQL:                 addConstraintSQL,
			CanRunInTransaction: true,
		},
		{
			SQL:                 validateConstraintSQL,
			CanRunInTransaction: true,
		},
		{
			SQL:                 setNotNullSQL,
			CanRunInTransaction: true,
		},
	}
}

// generateIndexSQL generates CREATE INDEX statement
func generateIndexSQL(index *ir.Index, isConcurrent bool) string {
	var sql strings.Builder

	sql.WriteString("CREATE")
	if index.Type == ir.IndexTypeUnique {
		sql.WriteString(" UNIQUE")
	}
	sql.WriteString(" INDEX")
	if isConcurrent {
		sql.WriteString(" CONCURRENTLY")
	}
	sql.WriteString(" IF NOT EXISTS ")
	sql.WriteString(index.Name)
	sql.WriteString(" ON ")

	tableName := getTableNameWithSchema(index.Schema, index.Table)
	sql.WriteString(tableName)

	if index.Method != "" && index.Method != "btree" {
		sql.WriteString(" USING ")
		sql.WriteString(index.Method)
	}

	sql.WriteString(" (")

	var columnParts []string
	for _, col := range index.Columns {
		part := col.Name
		if col.Direction != "" && col.Direction != "ASC" {
			part += " " + col.Direction
		}
		if col.Operator != "" {
			part += " " + col.Operator
		}
		columnParts = append(columnParts, part)
	}

	sql.WriteString(joinStrings(columnParts, ", "))
	sql.WriteString(")")

	if index.Where != "" {
		sql.WriteString(" WHERE ")
		sql.WriteString(index.Where)
	}

	sql.WriteString(";")
	return sql.String()
}

// generateIndexWaitQueryWithName creates a wait query for monitoring concurrent index creation
func generateIndexWaitQueryWithName(indexName string) string {
	return fmt.Sprintf(`SELECT 
    COALESCE(i.indisvalid, false) as done,
    CASE 
        WHEN p.blocks_total > 0 THEN p.blocks_done * 100 / p.blocks_total
        ELSE 0
    END as progress
FROM pg_class c
LEFT JOIN pg_index i ON c.oid = i.indexrelid
LEFT JOIN pg_stat_progress_create_index p ON c.oid = p.index_relid
WHERE c.relname = '%s';`, indexName)
}

// Helper functions

func getTableNameWithSchema(schema, table string) string {
	if schema != "" && schema != "public" {
		return fmt.Sprintf("%s.%s", schema, table)
	}
	return table
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	var result strings.Builder
	result.WriteString(strs[0])
	for _, s := range strs[1:] {
		result.WriteString(sep)
		result.WriteString(s)
	}
	return result.String()
}
