package ir

import (
	"fmt"
	"regexp"
	"strings"
)

// Index represents a database index
type Index struct {
	Schema       string         `json:"schema"`
	Table        string         `json:"table"`
	Name         string         `json:"name"`
	Type         IndexType      `json:"type"`
	Method       string         `json:"method"` // btree, hash, gin, gist, etc.
	Columns      []*IndexColumn `json:"columns"`
	IsUnique     bool           `json:"is_unique"`
	IsPrimary    bool           `json:"is_primary"`
	IsPartial    bool           `json:"is_partial"`
	IsConcurrent bool           `json:"is_concurrent"`
	Where        string         `json:"where,omitempty"` // partial index condition
	Definition   string         `json:"definition"`      // full CREATE INDEX statement
	Comment      string         `json:"comment,omitempty"`
}

// IndexColumn represents a column within an index
type IndexColumn struct {
	Name      string `json:"name"`
	Position  int    `json:"position"`
	Direction string `json:"direction,omitempty"` // ASC, DESC
	Operator  string `json:"operator,omitempty"`  // operator class
}


// SimplifyExpressionIndexDefinition converts an expression index definition to simplified format
// This function removes USING btree clauses, simplifies type casts, and normalizes JSON operators
func SimplifyExpressionIndexDefinition(definition, tableName string) string {
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
