package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

// generateCreateIndexesSQL generates CREATE INDEX statements
func generateCreateIndexesSQL(indexes []*ir.Index, targetSchema string, collector *diffCollector) {
	// Sort indexes by name for consistent ordering
	sortedIndexes := make([]*ir.Index, len(indexes))
	copy(sortedIndexes, indexes)
	sort.Slice(sortedIndexes, func(i, j int) bool {
		return sortedIndexes[i].Name < sortedIndexes[j].Name
	})

	for _, index := range sortedIndexes {
		// Skip primary key indexes as they're handled with constraints
		if index.Type == ir.IndexTypePrimary {
			continue
		}

		// Determine if we should create the index concurrently (plan mode) or not (dump mode)
		isConcurrent := collector.mode == PlanMode

		sql := generateIndexSQL(index, targetSchema, isConcurrent)

		// Create context for this statement
		context := &diffContext{
			Type:                "table.index",
			Operation:           "create",
			Path:                fmt.Sprintf("%s.%s.%s", index.Schema, index.Table, index.Name),
			Source:              index,
			CanRunInTransaction: !isConcurrent, // CREATE INDEX CONCURRENTLY cannot run in a transaction
		}

		// Bundle CREATE INDEX CONCURRENTLY with wait directive if concurrent
		if isConcurrent {
			statements := []SQLStatement{
				{
					SQL:                 sql,
					CanRunInTransaction: false, // CREATE INDEX CONCURRENTLY cannot run in a transaction
				},
				{
					SQL: generateIndexWaitQuery(index),
					Directive: &Directive{
						Type:    "wait",
						Message: fmt.Sprintf("Creating index %s", index.Name),
					},
					CanRunInTransaction: true, // Wait query can run in transaction
				},
			}
			collector.collectMultipleStatements(context, statements)
		} else {
			collector.collect(context, sql)
		}

		// Add index comment
		if index.Comment != "" {
			indexName := qualifyEntityName(index.Schema, index.Name, targetSchema)
			sql := fmt.Sprintf("COMMENT ON INDEX %s IS %s;", indexName, quoteString(index.Comment))

			// Create context for this statement
			context := &diffContext{
				Type:                "table.index.comment",
				Operation:           "create",
				Path:                fmt.Sprintf("%s.%s.%s", index.Schema, index.Table, index.Name),
				Source:              index,
				CanRunInTransaction: true, // Comments can always run in a transaction
			}

			collector.collect(context, sql)
		}
	}
}

// generateIndexSQL generates CREATE INDEX statement
func generateIndexSQL(index *ir.Index, targetSchema string, isConcurrent bool) string {
	var builder strings.Builder

	// CREATE [UNIQUE] INDEX [CONCURRENTLY] IF NOT EXISTS
	builder.WriteString("CREATE ")
	if index.Type == ir.IndexTypeUnique {
		builder.WriteString("UNIQUE ")
	}
	builder.WriteString("INDEX ")
	if isConcurrent {
		builder.WriteString("CONCURRENTLY ")
	}
	builder.WriteString("IF NOT EXISTS ")

	// Index name
	builder.WriteString(index.Name)
	builder.WriteString(" ON ")

	// Table name with proper schema qualification
	tableName := getTableNameWithSchema(index.Schema, index.Table, targetSchema)
	builder.WriteString(tableName)

	// Index method - only include if not btree (the default)
	if index.Method != "" && index.Method != "btree" {
		builder.WriteString(" USING ")
		builder.WriteString(index.Method)
	}

	// Columns
	builder.WriteString(" (")
	for i, col := range index.Columns {
		if i > 0 {
			builder.WriteString(", ")
		}

		// Handle JSON expressions with proper parentheses
		if strings.Contains(col.Name, "->>") || strings.Contains(col.Name, "->") {
			// Use double parentheses for JSON expressions for clean format
			builder.WriteString(fmt.Sprintf("((%s))", col.Name))
		} else {
			builder.WriteString(col.Name)
		}

		// Add direction if specified
		if col.Direction != "" && col.Direction != "ASC" {
			builder.WriteString(" ")
			builder.WriteString(col.Direction)
		}
	}
	builder.WriteString(")")

	// WHERE clause for partial indexes
	if index.IsPartial && index.Where != "" {
		builder.WriteString(" WHERE ")
		builder.WriteString(index.Where)
	}

	// Add semicolon at the end
	builder.WriteString(";")

	return builder.String()
}

// generateIndexWaitQuery creates a wait query for monitoring concurrent index creation
func generateIndexWaitQuery(index *ir.Index) string {
	return fmt.Sprintf(`SELECT 
    COALESCE(i.indisvalid, false) as done,
    CASE 
        WHEN p.blocks_total > 0 THEN p.blocks_done * 100 / p.blocks_total
        ELSE 0
    END as progress
FROM pg_class c
LEFT JOIN pg_index i ON c.oid = i.indexrelid
LEFT JOIN pg_stat_progress_create_index p ON c.oid = p.index_relid
WHERE c.relname = '%s'`, index.Name)
}
