package util

import (
	"strings"
	"unicode"
)

// PostgreSQL reserved words that need quoting
var reservedWords = map[string]bool{
	"user":   true,
	"order":  true,
	"group":  true,
	"select": true,
	"from":   true,
	"where":  true,
	"table":  true,
	// Add more as needed
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