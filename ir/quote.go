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

// QuoteIdentifierWithForce adds quotes to an identifier based on forceQuote flag.
// When forceQuote is true, all identifiers are quoted regardless of whether they need it.
// When forceQuote is false, only identifiers that require quoting (reserved words, mixed case, etc.) are quoted.
//
// Parameters:
//   - identifier: The identifier to potentially quote
//   - forceQuote: Whether to force quoting of all identifiers, regardless of whether they are reserved words
//
// Returns the identifier with quotes added if necessary or forced.
func QuoteIdentifierWithForce(identifier string, forceQuote bool) string {
	if identifier == "" {
		return ""
	}
	if forceQuote || NeedsQuoting(identifier) {
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

// QualifyEntityNameWithQuotesAndForce returns the properly qualified and quoted entity name with forceQuote option.
// This function combines schema qualification logic with optional forced quoting of all identifiers.
//
// Parameters:
//   - entitySchema: The schema of the entity
//   - entityName: The name of the entity
//   - targetSchema: The target schema for qualification (if same as entitySchema, schema prefix is omitted)
//   - forceQuote: Whether to force quoting of all identifiers, regardless of whether they are reserved words
//
// Returns a properly qualified and quoted entity name (e.g., "schema"."table" or just "table" if in target schema).
func QualifyEntityNameWithQuotesAndForce(entitySchema, entityName, targetSchema string, forceQuote bool) string {
	quotedName := QuoteIdentifierWithForce(entityName, forceQuote)

	if entitySchema == targetSchema {
		return quotedName
	}

	quotedSchema := QuoteIdentifierWithForce(entitySchema, forceQuote)
	return quotedSchema + "." + quotedName
}