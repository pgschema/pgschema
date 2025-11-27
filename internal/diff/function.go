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

		// Build argument list for DROP statement using GetArguments()
		argsList := function.GetArguments()

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

	// Add parameters from structured Parameters array
	// Exclude TABLE mode parameters as they're part of RETURNS clause
	var paramParts []string
	for _, param := range function.Parameters {
		if param.Mode != "TABLE" {
			paramParts = append(paramParts, formatFunctionParameter(param, true, targetSchema))
		}
	}
	if len(paramParts) > 0 {
		stmt.WriteString(fmt.Sprintf("(\n    %s\n)", strings.Join(paramParts, ",\n    ")))
	} else {
		stmt.WriteString("()")
	}

	// Add return type
	if function.ReturnType != "" {
		// Strip schema prefix from return type if it matches the target schema
		returnType := stripSchemaPrefix(function.ReturnType, targetSchema)
		stmt.WriteString(fmt.Sprintf("\nRETURNS %s", returnType))
	}

	// Add language
	if function.Language != "" {
		stmt.WriteString(fmt.Sprintf("\nLANGUAGE %s", function.Language))
	}

	// Add volatility if not default
	if function.Volatility != "" {
		stmt.WriteString(fmt.Sprintf("\n%s", function.Volatility))
	}

	// Add STRICT if specified
	if function.IsStrict {
		stmt.WriteString("\nSTRICT")
	}

	// Add SECURITY DEFINER if true (INVOKER is default and not output)
	if function.IsSecurityDefiner {
		stmt.WriteString("\nSECURITY DEFINER")
	}

	// Add LEAKPROOF if true
	if function.IsLeakproof {
		stmt.WriteString("\nLEAKPROOF")
	}
	// Note: Don't output NOT LEAKPROOF (it's the default)

	// Add PARALLEL if not default (UNSAFE)
	if function.Parallel == "SAFE" {
		stmt.WriteString("\nPARALLEL SAFE")
	} else if function.Parallel == "RESTRICTED" {
		stmt.WriteString("\nPARALLEL RESTRICTED")
	}
	// Note: Don't output PARALLEL UNSAFE (it's the default)

	// Add the function body
	if function.Definition != "" {
		// Check if this uses RETURN clause syntax (PG14+)
		// pg_get_function_sqlbody returns "RETURN expression" which should not be wrapped
		// Use case-insensitive comparison to handle all variations
		trimmedDef := strings.TrimSpace(function.Definition)
		if len(trimmedDef) >= 7 && strings.EqualFold(trimmedDef[:7], "RETURN ") {
			stmt.WriteString(fmt.Sprintf("\n%s;", trimmedDef))
		} else {
			// Traditional AS $$ ... $$ syntax
			tag := generateDollarQuoteTag(function.Definition)
			stmt.WriteString(fmt.Sprintf("\nAS %s%s%s;", tag, function.Definition, tag))
		}
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
func formatFunctionParameter(param *ir.Parameter, includeDefault bool, targetSchema string) string {
	var part string

	// For functions, only include mode if it's OUT or INOUT (IN is implicit)
	if param.Mode == "OUT" || param.Mode == "INOUT" || param.Mode == "VARIADIC" {
		part = param.Mode + " "
	}

	// Add parameter name and type
	// Strip schema prefix from data type if it matches the target schema
	dataType := stripSchemaPrefix(param.DataType, targetSchema)
	if param.Name != "" {
		part += param.Name + " " + dataType
	} else {
		part += dataType
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
	if old.Volatility != new.Volatility {
		return false
	}
	if old.IsStrict != new.IsStrict {
		return false
	}
	if old.IsSecurityDefiner != new.IsSecurityDefiner {
		return false
	}

	// Compare using normalized Parameters array
	// This ensures type aliases like "character varying" vs "varchar" are treated as equal
	// For RETURNS TABLE functions, exclude TABLE mode parameters (they're in ReturnType)
	// Only compare input parameters (IN, INOUT, VARIADIC, OUT)
	oldInputParams := filterNonTableParameters(old.Parameters)
	newInputParams := filterNonTableParameters(new.Parameters)
	return parametersEqual(oldInputParams, newInputParams)
}

// filterNonTableParameters filters out TABLE mode parameters
// TABLE parameters are output columns in RETURNS TABLE() and shouldn't be compared as input parameters
func filterNonTableParameters(params []*ir.Parameter) []*ir.Parameter {
	var filtered []*ir.Parameter
	for _, param := range params {
		if param.Mode != "TABLE" {
			filtered = append(filtered, param)
		}
	}
	return filtered
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
