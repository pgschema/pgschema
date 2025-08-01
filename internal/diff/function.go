package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

// generateCreateFunctionsSQL generates CREATE FUNCTION statements
func generateCreateFunctionsSQL(functions []*ir.Function, targetSchema string, collector *SQLCollector) {
	// Sort functions by name for consistent ordering
	sortedFunctions := make([]*ir.Function, len(functions))
	copy(sortedFunctions, functions)
	sort.Slice(sortedFunctions, func(i, j int) bool {
		return sortedFunctions[i].Name < sortedFunctions[j].Name
	})

	for _, function := range sortedFunctions {
		sql := generateFunctionSQL(function, targetSchema)
		
		// Create context for this statement
		context := &SQLContext{
			ObjectType:   "function",
			Operation:    "create",
			ObjectPath:   fmt.Sprintf("%s.%s", function.Schema, function.Name),
			SourceChange: function,
		}
		
		if collector != nil {
			collector.Collect(context, sql)
		}
	}
}

// generateModifyFunctionsSQL generates ALTER FUNCTION statements
func generateModifyFunctionsSQL(diffs []*FunctionDiff, targetSchema string, collector *SQLCollector) {
	for _, diff := range diffs {
		sql := generateFunctionSQL(diff.New, targetSchema)
		
		// Create context for this statement
		context := &SQLContext{
			ObjectType:   "function",
			Operation:    "alter",
			ObjectPath:   fmt.Sprintf("%s.%s", diff.New.Schema, diff.New.Name),
			SourceChange: diff,
		}
		
		if collector != nil {
			collector.Collect(context, sql)
		}
	}
}

// generateDropFunctionsSQL generates DROP FUNCTION statements
func generateDropFunctionsSQL(functions []*ir.Function, targetSchema string, collector *SQLCollector) {
	// Sort functions by name for consistent ordering
	sortedFunctions := make([]*ir.Function, len(functions))
	copy(sortedFunctions, functions)
	sort.Slice(sortedFunctions, func(i, j int) bool {
		return sortedFunctions[i].Name < sortedFunctions[j].Name
	})

	for _, function := range sortedFunctions {
		functionName := qualifyEntityName(function.Schema, function.Name, targetSchema)
		var sql string
		if function.Arguments != "" {
			sql = fmt.Sprintf("DROP FUNCTION IF EXISTS %s(%s);", functionName, function.Arguments)
		} else {
			sql = fmt.Sprintf("DROP FUNCTION IF EXISTS %s();", functionName)
		}
		
		// Create context for this statement
		context := &SQLContext{
			ObjectType:   "function",
			Operation:    "drop",
			ObjectPath:   fmt.Sprintf("%s.%s", function.Schema, function.Name),
			SourceChange: function,
		}
		
		if collector != nil {
			collector.Collect(context, sql)
		}
	}
}

// generateFunctionSQL generates CREATE OR REPLACE FUNCTION SQL for a function
func generateFunctionSQL(function *ir.Function, targetSchema string) string {
	var stmt strings.Builder

	// Build the CREATE OR REPLACE FUNCTION header with schema qualification
	functionName := qualifyEntityName(function.Schema, function.Name, targetSchema)
	stmt.WriteString(fmt.Sprintf("CREATE OR REPLACE FUNCTION %s", functionName))

	// Add parameters using detailed signature if available
	if function.Signature != "" {
		stmt.WriteString(fmt.Sprintf("(\n    %s\n)", strings.ReplaceAll(function.Signature, ", ", ",\n    ")))
	} else if function.Arguments != "" {
		stmt.WriteString(fmt.Sprintf("(%s)", function.Arguments))
	} else {
		stmt.WriteString("()")
	}

	// Add return type
	if function.ReturnType != "" {
		stmt.WriteString(fmt.Sprintf("\nRETURNS %s", function.ReturnType))
	}

	// Add language
	if function.Language != "" {
		stmt.WriteString(fmt.Sprintf("\nLANGUAGE %s", function.Language))
	}

	// Add security definer/invoker - PostgreSQL default is INVOKER
	if function.IsSecurityDefiner {
		stmt.WriteString("\nSECURITY DEFINER")
	} else {
		stmt.WriteString("\nSECURITY INVOKER")
	}

	// Add volatility if not default
	if function.Volatility != "" {
		stmt.WriteString(fmt.Sprintf("\n%s", function.Volatility))
	}

	// Add the function body with proper dollar quoting
	if function.Definition != "" {
		tag := generateDollarQuoteTag(function.Definition)
		stmt.WriteString(fmt.Sprintf("\nAS %s%s%s;", tag, function.Definition, tag))
	} else {
		stmt.WriteString("\nAS $$$$;")
	}

	return stmt.String()
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

// functionsEqual compares two functions for equality
func functionsEqual(old, new *ir.Function) bool {
	if old.Schema != new.Schema {
		return false
	}
	if old.Name != new.Name {
		return false
	}
	if old.Definition != new.Definition {
		return false
	}
	if old.ReturnType != new.ReturnType {
		return false
	}
	if old.Language != new.Language {
		return false
	}
	if old.Arguments != new.Arguments {
		return false
	}
	if old.Signature != new.Signature {
		return false
	}
	return true
}
