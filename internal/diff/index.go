package diff

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

// GenerateDropIndexSQL generates SQL for dropping indexes
func GenerateDropIndexSQL(indexes []*ir.Index) []string {
	var statements []string
	
	// Sort indexes by schema.table.name for consistent ordering
	sortedIndexes := make([]*ir.Index, len(indexes))
	copy(sortedIndexes, indexes)
	sort.Slice(sortedIndexes, func(i, j int) bool {
		keyI := sortedIndexes[i].Schema + "." + sortedIndexes[i].Table + "." + sortedIndexes[i].Name
		keyJ := sortedIndexes[j].Schema + "." + sortedIndexes[j].Table + "." + sortedIndexes[j].Name
		return keyI < keyJ
	})
	
	for _, index := range sortedIndexes {
		statements = append(statements, fmt.Sprintf("DROP INDEX IF EXISTS %s.%s;", index.Schema, index.Name))
	}
	
	return statements
}

// GenerateCreateIndexSQL generates SQL for creating indexes
func (d *DDLDiff) GenerateCreateIndexSQL(indexes []*ir.Index) []string {
	var statements []string
	
	// Sort indexes by schema.table.name for consistent ordering
	sortedIndexes := make([]*ir.Index, len(indexes))
	copy(sortedIndexes, indexes)
	sort.Slice(sortedIndexes, func(i, j int) bool {
		keyI := sortedIndexes[i].Schema + "." + sortedIndexes[i].Table + "." + sortedIndexes[i].Name
		keyJ := sortedIndexes[j].Schema + "." + sortedIndexes[j].Table + "." + sortedIndexes[j].Name
		return keyI < keyJ
	})
	
	for _, index := range sortedIndexes {
		// Generate clean migration SQL without schema qualifiers and USING btree
		indexSQL := d.generateIndexSQL(index, "")
		// Remove any comment headers and trailing newlines
		indexSQL = strings.TrimSpace(indexSQL)
		// Extract just the CREATE INDEX statement
		lines := strings.Split(indexSQL, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "CREATE") && strings.Contains(line, "INDEX") {
				statements = append(statements, line)
				break
			}
		}
	}
	
	return statements
}

// simplifyExpressionIndexDefinition converts an expression index definition to simplified format
// This function removes USING btree clauses, simplifies type casts, and normalizes JSON operators
// This is called during read time when dumping content to ensure we only process it once
func simplifyExpressionIndexDefinition(definition, tableName string) string {
	// Use regex to extract the index name and expression
	// Pattern: CREATE [UNIQUE] INDEX indexname ON [schema.]table USING method (expression)
	re := regexp.MustCompile(`CREATE\s+(UNIQUE\s+)?INDEX\s+(\w+)\s+ON\s+(?:(\w+)\.)?(\w+)\s+USING\s+(\w+)\s+\((.+)\)(?:\s+WHERE\s+.+)?`)
	matches := re.FindStringSubmatch(definition)

	if len(matches) >= 7 {
		isUnique := strings.TrimSpace(matches[1]) != ""
		indexName := matches[2]
		// matches[3] is schema (optional), matches[4] is table name, matches[5] is method, matches[6] is expression
		method := matches[5]
		expression := matches[6]

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

		if method == "btree" {
			return fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)", uniqueKeyword, indexName, tableName, expression)
		} else {
			return fmt.Sprintf("CREATE %sINDEX %s ON %s USING %s (%s)", uniqueKeyword, indexName, tableName, method, expression)
		}
	}

	// If regex doesn't match, return original definition unchanged
	return definition
}