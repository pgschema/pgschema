package diff

import (
	"github.com/pgschema/pgschema/internal/util"
)

// needsQuoting checks if an identifier needs to be quoted
// Deprecated: Use util.NeedsQuoting instead
func needsQuoting(identifier string) bool {
	return util.NeedsQuoting(identifier)
}

// quoteIdentifier adds quotes to an identifier if needed
// Deprecated: Use util.QuoteIdentifier instead
func quoteIdentifier(identifier string) string {
	return util.QuoteIdentifier(identifier)
}

// qualifyEntityNameWithQuotes returns the properly qualified and quoted entity name
// Deprecated: Use util.QualifyEntityNameWithQuotes instead
func qualifyEntityNameWithQuotes(entitySchema, entityName, targetSchema string) string {
	return util.QualifyEntityNameWithQuotes(entitySchema, entityName, targetSchema)
}