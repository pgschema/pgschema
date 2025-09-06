package diff

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

// needsQuoting checks if an identifier needs to be quoted
func needsQuoting(identifier string) bool {
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

// quoteIdentifier adds quotes to an identifier if needed
func quoteIdentifier(identifier string) string {
	if needsQuoting(identifier) {
		return `"` + identifier + `"`
	}
	return identifier
}

// qualifyEntityNameWithQuotes returns the properly qualified and quoted entity name
func qualifyEntityNameWithQuotes(entitySchema, entityName, targetSchema string) string {
	quotedName := quoteIdentifier(entityName)
	
	if entitySchema == targetSchema {
		return quotedName
	}
	
	quotedSchema := quoteIdentifier(entitySchema)
	return quotedSchema + "." + quotedName
}