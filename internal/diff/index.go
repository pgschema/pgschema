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

		sql := generateIndexSQL(index, targetSchema)

		// Create context for this statement
		context := &diffContext{
			Type:                "index",
			Operation:           "create",
			Path:                fmt.Sprintf("%s.%s", index.Schema, index.Name),
			Source:              index,
			CanRunInTransaction: !index.IsConcurrent, // CREATE INDEX CONCURRENTLY cannot run in a transaction
		}

		collector.collect(context, sql)

		// Add index comment
		if index.Comment != "" {
			indexName := qualifyEntityName(index.Schema, index.Name, targetSchema)
			sql := fmt.Sprintf("COMMENT ON INDEX %s IS %s;", indexName, quoteString(index.Comment))

			// Create context for this statement
			context := &diffContext{
				Type:                "comment",
				Operation:           "create",
				Path:                fmt.Sprintf("%s.%s", index.Schema, index.Name),
				Source:              index,
				CanRunInTransaction: true, // Comments can always run in a transaction
			}

			collector.collect(context, sql)
		}
	}
}

// generateIndexSQL generates CREATE INDEX statement
func generateIndexSQL(index *ir.Index, _ string) string {
	// Generate definition from components using the consolidated function
	return generateIndexDefinition(index)
}

// generateIndexDefinition generates a CREATE INDEX statement from index components
func generateIndexDefinition(index *ir.Index) string {
	var builder strings.Builder

	// CREATE [UNIQUE] INDEX [CONCURRENTLY] IF NOT EXISTS
	builder.WriteString("CREATE ")
	if index.Type == ir.IndexTypeUnique {
		builder.WriteString("UNIQUE ")
	}
	builder.WriteString("INDEX ")
	if index.IsConcurrent {
		builder.WriteString("CONCURRENTLY ")
	}
	builder.WriteString("IF NOT EXISTS ")

	// Index name
	builder.WriteString(index.Name)
	builder.WriteString(" ON ")

	// Table name (without schema for simplified format)
	builder.WriteString(index.Table)

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
