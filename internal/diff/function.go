package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/ir"
)

// generateCreateFunctionsSQL generates CREATE FUNCTION statements
func generateCreateFunctionsSQL(functions []*ir.Function, targetSchema string, collector *diffCollector) {
	// Sort functions by name for consistent ordering
	sortedFunctions := make([]*ir.Function, len(functions))
	copy(sortedFunctions, functions)
	sort.Slice(sortedFunctions, func(i, j int) bool {
		return sortedFunctions[i].Name < sortedFunctions[j].Name
	})

	for _, function := range sortedFunctions {
		sql := generateFunctionSQL(function, targetSchema)

		// Create context for this statement
		context := &diffContext{
			Type:                DiffTypeFunction,
			Operation:           DiffOperationCreate,
			Path:                fmt.Sprintf("%s.%s", function.Schema, function.Name),
			Source:              function,
			CanRunInTransaction: true,
		}

		collector.collect(context, sql)
	}
}

// generateModifyFunctionsSQL generates ALTER FUNCTION statements
func generateModifyFunctionsSQL(diffs []*functionDiff, targetSchema string, collector *diffCollector) {
	for _, diff := range diffs {
		sql := generateFunctionSQL(diff.New, targetSchema)

		// Create context for this statement
		context := &diffContext{
			Type:                DiffTypeFunction,
			Operation:           DiffOperationAlter,
			Path:                fmt.Sprintf("%s.%s", diff.New.Schema, diff.New.Name),
			Source:              diff,
			CanRunInTransaction: true,
		}

		collector.collect(context, sql)
	}
}

// generateDropFunctionsSQL generates DROP FUNCTION statements
func generateDropFunctionsSQL(functions []*ir.Function, targetSchema string, collector *diffCollector) {
	// Sort functions by name for consistent ordering
	sortedFunctions := make([]*ir.Function, len(functions))
	copy(sortedFunctions, functions)
	sort.Slice(sortedFunctions, func(i, j int) bool {
		return sortedFunctions[i].Name < sortedFunctions[j].Name
	})

	for _, function := range sortedFunctions {
		functionName := qualifyEntityName(function.Schema, function.Name, targetSchema)
		var sql string

		// Build argument list for DROP statement using normalized Parameters array
		var argsList string
		if len(function.Parameters) > 0 {
			// Format parameters for DROP (omit names and defaults, include only types)
			// Exclude TABLE mode parameters as they're part of RETURNS clause
			var argTypes []string
			for _, param := range function.Parameters {
				if param.Mode != "TABLE" {
					argTypes = append(argTypes, param.DataType)
				}
			}
			argsList = strings.Join(argTypes, ", ")
		} else if function.Arguments != "" {
			argsList = function.Arguments
		}

		if argsList != "" {
			sql = fmt.Sprintf("DROP FUNCTION IF EXISTS %s(%s);", functionName, argsList)
		} else {
			sql = fmt.Sprintf("DROP FUNCTION IF EXISTS %s();", functionName)
		}

		// Create context for this statement
		context := &diffContext{
			Type:                DiffTypeFunction,
			Operation:           DiffOperationDrop,
			Path:                fmt.Sprintf("%s.%s", function.Schema, function.Name),
			Source:              function,
			CanRunInTransaction: true,
		}

		collector.collect(context, sql)
	}
}

// generateFunctionSQL generates CREATE OR REPLACE FUNCTION SQL for a function
func generateFunctionSQL(function *ir.Function, targetSchema string) string {
	var stmt strings.Builder

	// Build the CREATE OR REPLACE FUNCTION header with schema qualification
	functionName := qualifyEntityName(function.Schema, function.Name, targetSchema)
	stmt.WriteString(fmt.Sprintf("CREATE OR REPLACE FUNCTION %s", functionName))

	// Add parameters - prefer structured Parameters array for normalized types
	if len(function.Parameters) > 0 {
		// Build parameter list from structured Parameters array
		// Exclude TABLE mode parameters as they're part of RETURNS clause
		var paramParts []string
		for _, param := range function.Parameters {
			if param.Mode != "TABLE" {
				paramParts = append(paramParts, formatFunctionParameter(param, true))
			}
		}
		if len(paramParts) > 0 {
			stmt.WriteString(fmt.Sprintf("(\n    %s\n)", strings.Join(paramParts, ",\n    ")))
		} else {
			stmt.WriteString("()")
		}
	} else if function.Signature != "" {
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

	// Add STRICT if specified
	if function.IsStrict {
		stmt.WriteString("\nSTRICT")
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

// formatFunctionParameter formats a single function parameter with name, type, and optional default value
// For functions, mode is typically omitted (unlike procedures) unless it's OUT/INOUT
// includeDefault controls whether DEFAULT clauses are included in the output
func formatFunctionParameter(param *ir.Parameter, includeDefault bool) string {
	var part string

	// For functions, only include mode if it's OUT or INOUT (IN is implicit)
	if param.Mode == "OUT" || param.Mode == "INOUT" || param.Mode == "VARIADIC" || param.Mode == "TABLE" {
		part = param.Mode + " "
	}

	// Add parameter name and type
	if param.Name != "" {
		part += param.Name + " " + param.DataType
	} else {
		part += param.DataType
	}

	// Add DEFAULT value if present and requested
	if includeDefault && param.DefaultValue != nil {
		part += " DEFAULT " + *param.DefaultValue
	}

	return part
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

	// For RETURNS TABLE functions, the Parameters array includes TABLE output columns
	// which can cause comparison issues. In this case, rely on ReturnType comparison instead.
	isTableReturn := strings.HasPrefix(old.ReturnType, "TABLE(") || strings.HasPrefix(new.ReturnType, "TABLE(")

	if !isTableReturn {
		// For non-TABLE functions, compare using normalized Parameters array
		// This ensures type aliases like "character varying" vs "varchar" are treated as equal
		hasOldParams := len(old.Parameters) > 0
		hasNewParams := len(new.Parameters) > 0

		if hasOldParams && hasNewParams {
			// Both have Parameters - compare them
			return parametersEqual(old.Parameters, new.Parameters)
		} else if hasOldParams || hasNewParams {
			// One has Parameters, one doesn't - they're different
			return false
		}
	}

	// For TABLE functions or functions without Parameters, fall back to Arguments/Signature
	if old.Arguments != new.Arguments {
		return false
	}
	if old.Signature != new.Signature {
		return false
	}

	return true
}

// parametersEqual compares two parameter arrays for equality
func parametersEqual(oldParams, newParams []*ir.Parameter) bool {
	if len(oldParams) != len(newParams) {
		return false
	}

	for i := range oldParams {
		if !parameterEqual(oldParams[i], newParams[i]) {
			return false
		}
	}

	return true
}

// parameterEqual compares two parameters for equality
func parameterEqual(old, new *ir.Parameter) bool {
	if old.Name != new.Name {
		return false
	}

	// Compare data types (already normalized by ir.normalizeFunction)
	if old.DataType != new.DataType {
		return false
	}

	if old.Mode != new.Mode {
		return false
	}

	// Compare default values
	if (old.DefaultValue == nil) != (new.DefaultValue == nil) {
		return false
	}
	if old.DefaultValue != nil && new.DefaultValue != nil {
		if *old.DefaultValue != *new.DefaultValue {
			return false
		}
	}

	return true
}
