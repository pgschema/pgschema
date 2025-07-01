package ir

import (
	"fmt"
	"strings"
)

// Function represents a database function
type Function struct {
	Schema            string       `json:"schema"`
	Name              string       `json:"name"`
	Definition        string       `json:"definition"`
	ReturnType        string       `json:"return_type"`
	Language          string       `json:"language"`
	Arguments         string       `json:"arguments,omitempty"`
	Signature         string       `json:"signature,omitempty"`
	Parameters        []*Parameter `json:"parameters,omitempty"`
	Comment           string       `json:"comment,omitempty"`
	Volatility        string       `json:"volatility,omitempty"`          // IMMUTABLE, STABLE, VOLATILE
	IsStrict          bool         `json:"is_strict,omitempty"`           // STRICT or null behavior
	IsSecurityDefiner bool         `json:"is_security_definer,omitempty"` // SECURITY DEFINER
}

// Parameter represents a function parameter
type Parameter struct {
	Name         string  `json:"name"`
	DataType     string  `json:"data_type"`
	Mode         string  `json:"mode"` // IN, OUT, INOUT
	Position     int     `json:"position"`
	DefaultValue *string `json:"default_value,omitempty"`
}

// GenerateSQL for Function
func (f *Function) GenerateSQL() string {
	if f.Definition == "<nil>" || f.Definition == "" {
		return ""
	}
	w := NewSQLWriter()

	// Build function signature for comment header (types only with schema qualifiers)
	headerSig := fmt.Sprintf("%s(%s)", f.Name, f.Arguments)

	// Build full function signature for CREATE statement (with parameter names)
	var createSig string
	if f.Signature != "" && f.Signature != "<nil>" {
		createSig = fmt.Sprintf("%s(%s)", f.Name, f.Signature)
	} else {
		createSig = fmt.Sprintf("%s(%s)", f.Name, f.Arguments)
	}

	// Build qualifiers (volatility and strictness)
	var qualifiers []string
	if f.Volatility != "" && f.Volatility != "VOLATILE" {
		// Only include non-default volatility (VOLATILE is the default, so omit it)
		qualifiers = append(qualifiers, f.Volatility)
	}
	if f.IsStrict {
		qualifiers = append(qualifiers, "STRICT")
	}
	if f.IsSecurityDefiner {
		qualifiers = append(qualifiers, "SECURITY DEFINER")
	}

	qualifierStr := ""
	if len(qualifiers) > 0 {
		qualifierStr = " " + strings.Join(qualifiers, " ")
	}

	// Generate CREATE FUNCTION statement with proper dollar quoting
	dollarTag := generateDollarQuoteTag(f.Definition)
	stmt := fmt.Sprintf("CREATE FUNCTION %s.%s RETURNS %s\n    LANGUAGE %s%s\n    AS %s%s%s;",
		f.Schema, createSig, f.ReturnType, strings.ToLower(f.Language), qualifierStr, dollarTag, f.Definition, dollarTag)
	w.WriteStatementWithComment("FUNCTION", headerSig, f.Schema, "", stmt, "")

	// Generate COMMENT ON FUNCTION statement if comment exists
	if f.Comment != "" && f.Comment != "<nil>" {
		w.WriteDDLSeparator()

		// Escape single quotes in comment
		escapedComment := strings.ReplaceAll(f.Comment, "'", "''")
		commentStmt := fmt.Sprintf("COMMENT ON FUNCTION %s.%s IS '%s';", f.Schema, headerSig, escapedComment)
		w.WriteStatementWithComment("COMMENT", "FUNCTION "+headerSig, f.Schema, "", commentStmt, "")
	}

	return w.String()
}

// generateDollarQuoteTag creates a safe dollar quote tag that doesn't conflict with the function body content.
// This implements the same algorithm used by pg_dump to avoid conflicts.
func generateDollarQuoteTag(body string) string {
	// Check if the body contains potential conflicts with $$ quoting:
	// 1. Direct $$ sequences
	// 2. Parameter references like $1, $2, etc. that could be ambiguous
	needsTagged := strings.Contains(body, "$$") || containsParameterReferences(body)

	if !needsTagged {
		return "$$"
	}

	// Start with the pg_dump preferred tag
	candidates := []string{"$_$", "$function$", "$body$", "$pgdump$"}

	// Try each candidate tag
	for _, tag := range candidates {
		if !strings.Contains(body, tag) {
			return tag
		}
	}

	// If all predefined tags conflict, generate a unique one
	// Use a simple incrementing number approach like pg_dump does
	for i := 1; i < 1000; i++ {
		tag := fmt.Sprintf("$tag%d$", i)
		if !strings.Contains(body, tag) {
			return tag
		}
	}

	// Fallback - this should rarely happen
	return "$fallback$"
}

// containsParameterReferences checks if the body contains PostgreSQL parameter references ($1, $2, etc.)
// that could be confused with dollar quoting delimiters
func containsParameterReferences(body string) bool {
	// Simple check for $digit patterns which are PostgreSQL parameter references
	for i := 0; i < len(body)-1; i++ {
		if body[i] == '$' && i+1 < len(body) && body[i+1] >= '0' && body[i+1] <= '9' {
			return true
		}
	}
	return false
}
