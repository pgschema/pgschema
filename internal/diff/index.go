package diff

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

// generateDropIndexesSQL generates DROP INDEX statements
func generateDropIndexesSQL(w *SQLWriter, indexes []*ir.Index, targetSchema string) {
	// Sort indexes by name for consistent ordering
	sortedIndexes := make([]*ir.Index, len(indexes))
	copy(sortedIndexes, indexes)
	sort.Slice(sortedIndexes, func(i, j int) bool {
		return sortedIndexes[i].Name < sortedIndexes[j].Name
	})

	for _, index := range sortedIndexes {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP INDEX IF EXISTS %s;", index.Name)
		w.WriteStatementWithComment("INDEX", index.Name, index.Schema, "", sql, targetSchema)
	}
}

// generateCreateIndexesSQL generates CREATE INDEX statements
func generateCreateIndexesSQL(w *SQLWriter, indexes []*ir.Index, targetSchema string) {
	// Sort indexes by name for consistent ordering
	sortedIndexes := make([]*ir.Index, len(indexes))
	copy(sortedIndexes, indexes)
	sort.Slice(sortedIndexes, func(i, j int) bool {
		return sortedIndexes[i].Name < sortedIndexes[j].Name
	})

	for _, index := range sortedIndexes {
		// Skip primary key indexes as they're handled with constraints
		if index.IsPrimary {
			continue
		}

		w.WriteDDLSeparator()
		sql := generateIndexSQL(index, targetSchema)
		w.WriteStatementWithComment("INDEX", index.Name, index.Schema, "", sql, targetSchema)
	}
}

// generateIndexSQL generates CREATE INDEX statement
func generateIndexSQL(index *ir.Index, targetSchema string) string {
	var stmt string
	if index.Schema != targetSchema {
		// Use the definition as-is
		stmt = index.Definition
	} else {
		// Remove schema qualifiers from the definition for schema-agnostic output
		definition := index.Definition
		schemaPrefix := index.Schema + "."
		// Remove schema qualifiers that match the target schema
		definition = strings.ReplaceAll(definition, schemaPrefix, "")
		stmt = definition
	}

	// Apply expression index simplification during read time
	stmt = simplifyExpressionIndexDefinition(stmt, index.Table)

	if !strings.HasSuffix(stmt, ";") {
		stmt += ";"
	}

	return stmt
}

// parseIndexDefinition parses a CREATE INDEX statement and returns the captured groups
// This handles nested parentheses properly
func parseIndexDefinition(definition string) []string {
	// First use regex to extract the basic structure
	re := regexp.MustCompile(`^CREATE\s+(UNIQUE\s+)?INDEX\s+(CONCURRENTLY\s+)?(\w+)\s+ON\s+(?:(\w+)\.)?(\w+)\s+USING\s+(\w+)\s+\(`)
	basicMatches := re.FindStringSubmatch(definition)
	if basicMatches == nil {
		return nil
	}

	// Find the start of the expression (after "USING method (")
	startIdx := re.FindStringIndex(definition)
	if startIdx == nil {
		return nil
	}
	exprStart := startIdx[1] // Position after the opening parenthesis

	// Find the matching closing parenthesis for the expression
	parenCount := 1
	exprEnd := exprStart
	for i := exprStart; i < len(definition) && parenCount > 0; i++ {
		if definition[i] == '(' {
			parenCount++
		} else if definition[i] == ')' {
			parenCount--
			if parenCount == 0 {
				exprEnd = i
				break
			}
		}
	}

	if parenCount != 0 {
		// Unbalanced parentheses, return nil
		return nil
	}

	// Extract the expression
	expression := definition[exprStart:exprEnd]

	// Check for WHERE clause
	remainingDef := strings.TrimSpace(definition[exprEnd+1:])
	whereClause := ""
	if strings.HasPrefix(remainingDef, "WHERE ") {
		whereClause = remainingDef[6:] // Remove "WHERE "
	}

	// Build result array matching the original regex groups
	result := make([]string, 9)
	result[0] = definition // Full match
	result[1] = basicMatches[1] // UNIQUE
	result[2] = basicMatches[2] // CONCURRENTLY  
	result[3] = basicMatches[3] // index name
	result[4] = basicMatches[4] // schema (optional)
	result[5] = basicMatches[5] // table name
	result[6] = basicMatches[6] // method
	result[7] = expression      // expression with proper parentheses
	result[8] = whereClause     // WHERE clause

	return result
}

// simplifyExpressionIndexDefinition converts an expression index definition to simplified format
// This function removes USING btree clauses, simplifies type casts, and normalizes JSON operators
// This is called during read time when dumping content to ensure we only process it once
func simplifyExpressionIndexDefinition(definition, tableName string) string {
	// Parse the index definition to extract components
	// We need to handle nested parentheses in expressions properly
	match := parseIndexDefinition(definition)
	if match == nil {
		// If parsing fails, return original definition unchanged
		return definition
	}

	if len(match) >= 8 {
		isUnique := strings.TrimSpace(match[1]) != ""
		isConcurrent := strings.TrimSpace(match[2]) != ""
		indexName := match[3]
		// match[4] is schema (optional), match[5] is table name, match[6] is method, match[7] is expression, match[8] is WHERE clause
		method := match[6]
		expression := match[7]
		whereClause := ""
		if len(match) > 8 && match[8] != "" {
			whereClause = match[8]
		}

		// Simplify the expression - remove ::text type casts
		expression = strings.ReplaceAll(expression, "::text", "")

		// Remove spaces around JSON operators for consistency
		expression = strings.ReplaceAll(expression, " ->> ", "->>")
		expression = strings.ReplaceAll(expression, " -> ", "->")

		// Rebuild in simplified format - preserve UNIQUE keyword and only omit USING clause for btree (default)
		uniqueKeyword := ""
		if isUnique {
			uniqueKeyword = "UNIQUE "
		}

		concurrentlyKeyword := ""
		if isConcurrent {
			concurrentlyKeyword = "CONCURRENTLY "
		}

		// Build the WHERE clause part
		whereClausePart := ""
		if whereClause != "" {
			// Check if the WHERE clause already has parentheses
			whereClause = strings.TrimSpace(whereClause)
			if strings.HasPrefix(whereClause, "(") && strings.HasSuffix(whereClause, ")") {
				whereClausePart = fmt.Sprintf(" WHERE %s", whereClause)
			} else {
				whereClausePart = fmt.Sprintf(" WHERE (%s)", whereClause)
			}
		}

		if method == "btree" {
			return fmt.Sprintf("CREATE %sINDEX %s%s ON %s (%s)%s", uniqueKeyword, concurrentlyKeyword, indexName, tableName, expression, whereClausePart)
		} else {
			return fmt.Sprintf("CREATE %sINDEX %s%s ON %s USING %s (%s)%s", uniqueKeyword, concurrentlyKeyword, indexName, tableName, method, expression, whereClausePart)
		}
	}

	// If regex doesn't match, return original definition unchanged
	return definition
}
