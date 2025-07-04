package ir

import (
	"fmt"
	"strings"

	"github.com/pgschema/pgschema/internal/utils"
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
	return f.GenerateSQLWithSchema(f.Schema)
}

// GenerateSQLWithSchema for Function with target schema context
func (f *Function) GenerateSQLWithSchema(targetSchema string) string {
	return f.GenerateSQLWithOptions(true, targetSchema)
}

// GenerateSQLWithOptions generates SQL for a function with configurable comment inclusion
func (f *Function) GenerateSQLWithOptions(includeComments bool, targetSchema string) string {
	if f.Definition == "<nil>" || f.Definition == "" {
		return ""
	}
	w := NewSQLWriterWithComments(includeComments)

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

	// Only include function name without schema if it's in the target schema
	funcName := utils.QualifyEntityName(f.Schema, createSig, targetSchema)
	stmt := fmt.Sprintf("CREATE FUNCTION %s RETURNS %s\n    LANGUAGE %s%s\n    AS %s%s%s;",
		funcName, f.ReturnType, strings.ToLower(f.Language), qualifierStr, dollarTag, f.Definition, dollarTag)

	// For comment header, use "-" if in target schema
	commentSchema := utils.GetCommentSchemaName(f.Schema, targetSchema)
	if includeComments {
		w.WriteStatementWithComment("FUNCTION", headerSig, commentSchema, "", stmt, "")
	} else {
		w.WriteString(stmt)
	}

	// Generate COMMENT ON FUNCTION statement if comment exists
	if f.Comment != "" && f.Comment != "<nil>" && includeComments {
		w.WriteDDLSeparator()

		// Escape single quotes in comment
		escapedComment := strings.ReplaceAll(f.Comment, "'", "''")

		// Only include function name without schema if it's in the target schema
		var funcRef string
		if f.Schema == targetSchema {
			funcRef = headerSig
		} else {
			funcRef = fmt.Sprintf("%s.%s", f.Schema, headerSig)
		}
		commentStmt := fmt.Sprintf("COMMENT ON FUNCTION %s IS '%s';", funcRef, escapedComment)
		w.WriteStatementWithComment("COMMENT", "FUNCTION "+headerSig, commentSchema, "", commentStmt, "")
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
