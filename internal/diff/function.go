package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

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

// GenerateDropFunctionSQL generates SQL for dropping functions
func GenerateDropFunctionSQL(functions []*ir.Function) []string {
	var statements []string
	
	// Sort functions by schema.name for consistent ordering
	sortedFunctions := make([]*ir.Function, len(functions))
	copy(sortedFunctions, functions)
	sort.Slice(sortedFunctions, func(i, j int) bool {
		keyI := sortedFunctions[i].Schema + "." + sortedFunctions[i].Name
		keyJ := sortedFunctions[j].Schema + "." + sortedFunctions[j].Name
		return keyI < keyJ
	})
	
	for _, function := range sortedFunctions {
		functionName := getTableNameWithSchema(function.Schema, function.Name, function.Schema)
		if function.Arguments != "" {
			statements = append(statements, fmt.Sprintf("DROP FUNCTION IF EXISTS %s(%s);", functionName, function.Arguments))
		} else {
			statements = append(statements, fmt.Sprintf("DROP FUNCTION IF EXISTS %s();", functionName))
		}
	}
	
	return statements
}

// GenerateCreateFunctionSQL generates SQL for creating functions
func GenerateCreateFunctionSQL(functions []*ir.Function) []string {
	var statements []string
	
	// Sort functions by schema.name for consistent ordering
	sortedFunctions := make([]*ir.Function, len(functions))
	copy(sortedFunctions, functions)
	sort.Slice(sortedFunctions, func(i, j int) bool {
		keyI := sortedFunctions[i].Schema + "." + sortedFunctions[i].Name
		keyJ := sortedFunctions[j].Schema + "." + sortedFunctions[j].Name
		return keyI < keyJ
	})
	
	for _, function := range sortedFunctions {
		stmt := generateFunctionSQL(function)
		statements = append(statements, stmt)
	}
	
	return statements
}

// GenerateAlterFunctionSQL generates SQL for modifying functions
func GenerateAlterFunctionSQL(functionDiffs []*FunctionDiff) []string {
	var statements []string
	
	// Sort modified functions by schema.name for consistent ordering
	sortedFunctionDiffs := make([]*FunctionDiff, len(functionDiffs))
	copy(sortedFunctionDiffs, functionDiffs)
	sort.Slice(sortedFunctionDiffs, func(i, j int) bool {
		keyI := sortedFunctionDiffs[i].New.Schema + "." + sortedFunctionDiffs[i].New.Name
		keyJ := sortedFunctionDiffs[j].New.Schema + "." + sortedFunctionDiffs[j].New.Name
		return keyI < keyJ
	})
	
	for _, functionDiff := range sortedFunctionDiffs {
		stmt := generateFunctionSQL(functionDiff.New)
		statements = append(statements, stmt)
	}
	
	return statements
}

// generateFunctionSQL generates CREATE OR REPLACE FUNCTION SQL for a function
func generateFunctionSQL(function *ir.Function) string {
	var stmt strings.Builder

	// Build the CREATE OR REPLACE FUNCTION header with schema qualification
	functionName := getTableNameWithSchema(function.Schema, function.Name, function.Schema)
	stmt.WriteString(fmt.Sprintf("CREATE OR REPLACE FUNCTION %s", functionName))

	// Add parameters
	if function.Signature != "" {
		stmt.WriteString(fmt.Sprintf("(\n    %s\n)", strings.ReplaceAll(function.Signature, ", ", ",\n    ")))
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

	// Add security definer/invoker
	if function.IsSecurityDefiner {
		stmt.WriteString("\nSECURITY DEFINER")
	} else {
		stmt.WriteString("\nSECURITY INVOKER")
	}

	// Add volatility if not default
	if function.Volatility != "" {
		stmt.WriteString(fmt.Sprintf("\n%s", function.Volatility))
	}

	// Add the function body
	if function.Definition != "" {
		stmt.WriteString(fmt.Sprintf("\nAS $$%s\n$$;", function.Definition))
	}

	return stmt.String()
}