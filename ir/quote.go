package ir

import (
	"strings"
	"unicode"
)

// PostgreSQL reserved words that need quoting
// Based on PostgreSQL 17 documentation: https://www.postgresql.org/docs/current/sql-keywords-appendix.html
var reservedWords = map[string]bool{
	// A-C
	"all":              true,
	"and":              true,
	"any":              true,
	"array":            true,
	"as":               true,
	"asymmetric":       true,
	"authorization":    true,
	"between":          true,
	"binary":           true,
	"both":             true,
	"case":             true,
	"cast":             true,
	"check":            true,
	"collate":          true,
	"column":           true,
	"constraint":       true,
	"create":           true,
	"cross":            true,
	"current_catalog":  true,
	"current_date":     true,
	"current_role":     true,
	"current_schema":   true,
	"current_time":     true,
	"current_timestamp": true,
	"current_user":     true,
	// D-F
	"default":     true,
	"deferrable":  true,
	"distinct":    true,
	"do":          true,
	"else":        true,
	"end":         true,
	"except":      true,
	"exists":      true,
	"false":       true,
	"fetch":       true,
	"foreign":     true,
	"from":        true,
	// G-L
	"grant":       true,
	"group":       true,
	"having":      true,
	"in":          true,
	"initially":   true,
	"intersect":   true,
	"into":        true,
	"is":          true,
	"isnull":      true,
	"join":        true,
	"lateral":     true,
	"left":        true,
	"like":        true,
	// N-P
	"not":         true,
	"null":        true,
	"of":          true,
	"on":          true,
	"only":        true,
	"or":          true,
	"order":       true,
	"primary":     true,
	// R-S
	"references":  true,
	"right":       true,
	"select":      true,
	"symmetric":   true,
	"system_user": true,
	// T-W
	"table":       true,
	"then":        true,
	"to":          true,
	"true":        true,
	"union":       true,
	"unique":      true,
	"user":        true,
	"using":       true,
	"verbose":     true,
	"when":        true,
	"where":       true,
	"window":      true,
	"with":        true,
}

// NeedsQuoting checks if an identifier needs to be quoted
func NeedsQuoting(identifier string) bool {
	if identifier == "" {
		return false
	}

	// Check if it's a reserved word
	if reservedWords[strings.ToLower(identifier)] {
		return true
	}

	// Check if it contains uppercase letters (PostgreSQL folds unquoted to lowercase)
	for _, r := range identifier {
		if unicode.IsUpper(r) {
			return true
		}
	}

	// Check if it starts with non-letter or contains special characters
	for i, r := range identifier {
		if i == 0 && !unicode.IsLetter(r) && r != '_' {
			return true
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return true
		}
	}

	return false
}

// QuoteIdentifier adds quotes to an identifier if needed
func QuoteIdentifier(identifier string) string {
	if NeedsQuoting(identifier) {
		return `"` + identifier + `"`
	}
	return identifier
}

// QualifyEntityNameWithQuotes returns the properly qualified and quoted entity name
func QualifyEntityNameWithQuotes(entitySchema, entityName, targetSchema string) string {
	quotedName := QuoteIdentifier(entityName)

	if entitySchema == targetSchema {
		return quotedName
	}

	quotedSchema := QuoteIdentifier(entitySchema)
	return quotedSchema + "." + quotedName
}