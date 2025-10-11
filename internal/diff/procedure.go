package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/ir"
)

// generateCreateProceduresSQL generates CREATE PROCEDURE statements
func generateCreateProceduresSQL(procedures []*ir.Procedure, targetSchema string, collector *diffCollector) {
	// Sort procedures by name for consistent ordering
	sortedProcedures := make([]*ir.Procedure, len(procedures))
	copy(sortedProcedures, procedures)
	sort.Slice(sortedProcedures, func(i, j int) bool {
		return sortedProcedures[i].Name < sortedProcedures[j].Name
	})

	for _, procedure := range sortedProcedures {
		sql := generateProcedureSQL(procedure, targetSchema)

		// Create context for this statement
		context := &diffContext{
			Type:                DiffTypeProcedure,
			Operation:           DiffOperationCreate,
			Path:                fmt.Sprintf("%s.%s", procedure.Schema, procedure.Name),
			Source:              procedure,
			CanRunInTransaction: true,
		}

		collector.collect(context, sql)
	}
}

// generateModifyProceduresSQL generates DROP and CREATE PROCEDURE statements for modified procedures
func generateModifyProceduresSQL(diffs []*procedureDiff, targetSchema string, collector *diffCollector) {
	for _, diff := range diffs {
		// Drop the old procedure first
		procedureName := qualifyEntityName(diff.Old.Schema, diff.Old.Name, targetSchema)
		var dropSQL string

		// For DROP statements, we need the full parameter signature including modes and names
		paramSignature := formatProcedureParametersForDrop(diff.Old)
		if paramSignature != "" {
			dropSQL = fmt.Sprintf("DROP PROCEDURE IF EXISTS %s(%s);", procedureName, paramSignature)
		} else {
			dropSQL = fmt.Sprintf("DROP PROCEDURE IF EXISTS %s();", procedureName)
		}

		// Create the new procedure
		createSQL := generateProcedureSQL(diff.New, targetSchema)

		// Create a single context with ALTER operation and multiple statements
		// This represents the modification as a single operation in the summary
		alterContext := &diffContext{
			Type:                DiffTypeProcedure,
			Operation:           DiffOperationAlter,
			Path:                fmt.Sprintf("%s.%s", diff.New.Schema, diff.New.Name),
			Source:              diff,
			CanRunInTransaction: true,
		}

		// Collect both DROP and CREATE as separate statements within a single diff
		statements := []SQLStatement{
			{SQL: dropSQL, CanRunInTransaction: true},
			{SQL: createSQL, CanRunInTransaction: true},
		}

		collector.collectStatements(alterContext, statements)
	}
}

// generateDropProceduresSQL generates DROP PROCEDURE statements
func generateDropProceduresSQL(procedures []*ir.Procedure, targetSchema string, collector *diffCollector) {
	// Sort procedures by name for consistent ordering
	sortedProcedures := make([]*ir.Procedure, len(procedures))
	copy(sortedProcedures, procedures)
	sort.Slice(sortedProcedures, func(i, j int) bool {
		return sortedProcedures[i].Name < sortedProcedures[j].Name
	})

	for _, procedure := range sortedProcedures {
		procedureName := qualifyEntityName(procedure.Schema, procedure.Name, targetSchema)
		var sql string

		// For DROP statements, we need the full parameter signature including modes and names
		// Extract the complete signature from the procedure
		paramSignature := formatProcedureParametersForDrop(procedure)
		if paramSignature != "" {
			sql = fmt.Sprintf("DROP PROCEDURE IF EXISTS %s(%s);", procedureName, paramSignature)
		} else {
			sql = fmt.Sprintf("DROP PROCEDURE IF EXISTS %s();", procedureName)
		}

		// Create context for this statement
		context := &diffContext{
			Type:                DiffTypeProcedure,
			Operation:           DiffOperationDrop,
			Path:                fmt.Sprintf("%s.%s", procedure.Schema, procedure.Name),
			Source:              procedure,
			CanRunInTransaction: true,
		}

		collector.collect(context, sql)
	}
}

// formatParameterString formats a single parameter with mode, name, type, and optional default value
// includeDefault controls whether DEFAULT clauses are included in the output
func formatParameterString(param *ir.Parameter, includeDefault bool) string {
	var part string
	// Always include mode for clarity (IN is default but we make it explicit)
	if param.Mode != "" {
		part = param.Mode + " "
	} else {
		part = "IN "
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

// generateProcedureSQL generates CREATE OR REPLACE PROCEDURE SQL for a procedure
func generateProcedureSQL(procedure *ir.Procedure, targetSchema string) string {
	var stmt strings.Builder

	// Build the CREATE OR REPLACE PROCEDURE header with schema qualification
	procedureName := qualifyEntityName(procedure.Schema, procedure.Name, targetSchema)
	stmt.WriteString(fmt.Sprintf("CREATE OR REPLACE PROCEDURE %s", procedureName))

	// Add parameters - prefer structured Parameters array, then signature, then arguments
	if len(procedure.Parameters) > 0 {
		// Build parameter list from structured Parameters array
		// Always include mode explicitly (matching pg_dump behavior)
		var paramParts []string
		for _, param := range procedure.Parameters {
			paramParts = append(paramParts, formatParameterString(param, true))
		}
		if len(paramParts) > 0 {
			stmt.WriteString(fmt.Sprintf("(\n    %s\n)", strings.Join(paramParts, ",\n    ")))
		} else {
			stmt.WriteString("()")
		}
	} else if procedure.Signature != "" {
		// Use detailed signature if available
		stmt.WriteString(fmt.Sprintf("(\n    %s\n)", strings.ReplaceAll(procedure.Signature, ", ", ",\n    ")))
	} else if procedure.Arguments != "" {
		// Format Arguments field with newlines if it contains multiple parameters
		args := procedure.Arguments
		if strings.Contains(args, ", ") {
			args = strings.ReplaceAll(args, ", ", ",\n    ")
			stmt.WriteString(fmt.Sprintf("(\n    %s\n)", args))
		} else {
			stmt.WriteString(fmt.Sprintf("(%s)", args))
		}
	} else {
		stmt.WriteString("()")
	}

	// Add language
	if procedure.Language != "" {
		stmt.WriteString(fmt.Sprintf("\nLANGUAGE %s", procedure.Language))
	}

	// Note: Procedures don't have SECURITY DEFINER/INVOKER in PostgreSQL
	// This is a function-only feature

	// Add the procedure body with proper dollar quoting
	if procedure.Definition != "" {
		tag := generateProcedureDollarQuoteTag(procedure.Definition)
		stmt.WriteString(fmt.Sprintf("\nAS %s%s%s;", tag, procedure.Definition, tag))
	} else {
		stmt.WriteString("\nAS $$$$;")
	}

	return stmt.String()
}

// generateProcedureDollarQuoteTag creates a safe dollar quote tag that doesn't conflict with the procedure body content.
// This implements the same algorithm used by pg_dump to avoid conflicts.
func generateProcedureDollarQuoteTag(body string) string {
	// Check if the body contains potential conflicts with $$ quoting:
	// 1. Direct $$ sequences
	// 2. Parameter references like $1, $2, etc. that could be ambiguous
	needsTagged := strings.Contains(body, "$$") || containsProcedureParameterReferences(body)

	if !needsTagged {
		return "$$"
	}

	// Start with the pg_dump preferred tag
	candidates := []string{"$_$", "$procedure$", "$body$", "$pgdump$"}

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

// containsProcedureParameterReferences checks if the body contains PostgreSQL parameter references ($1, $2, etc.)
// that could be confused with dollar quoting delimiters
func containsProcedureParameterReferences(body string) bool {
	// Simple check for $digit patterns which are PostgreSQL parameter references
	for i := 0; i < len(body)-1; i++ {
		if body[i] == '$' && i+1 < len(body) && body[i+1] >= '0' && body[i+1] <= '9' {
			return true
		}
	}
	return false
}

// proceduresEqual compares two procedures for equality
func proceduresEqual(old, new *ir.Procedure) bool {
	if old.Schema != new.Schema {
		return false
	}
	if old.Name != new.Name {
		return false
	}
	if old.Definition != new.Definition {
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

// formatProcedureParametersForDrop formats procedure parameters for DROP PROCEDURE statements
// Returns the full parameter signature including mode and name (e.g., "IN order_id integer, IN amount numeric")
// This is necessary for proper procedure identification in PostgreSQL
func formatProcedureParametersForDrop(procedure *ir.Procedure) string {
	// First, try to use the structured Parameters array if available
	if len(procedure.Parameters) > 0 {
		var paramParts []string
		for _, param := range procedure.Parameters {
			// Use helper function with includeDefault=false for DROP statements
			paramParts = append(paramParts, formatParameterString(param, false))
		}
		return strings.Join(paramParts, ", ")
	}

	// Fallback to Signature field if Parameters not available
	if procedure.Signature != "" {
		// Signature should already have the mode information
		// Just need to remove DEFAULT clauses
		var paramParts []string
		params := strings.Split(procedure.Signature, ",")
		for _, param := range params {
			param = strings.TrimSpace(param)
			// Remove DEFAULT clauses
			if idx := strings.Index(param, " DEFAULT "); idx != -1 {
				param = param[:idx]
			}
			paramParts = append(paramParts, param)
		}
		return strings.Join(paramParts, ", ")
	}

	// Last resort: try to parse Arguments field and add IN mode
	if procedure.Arguments != "" {
		var paramParts []string
		params := strings.Split(procedure.Arguments, ",")
		for _, param := range params {
			param = strings.TrimSpace(param)
			// Remove DEFAULT clauses
			if idx := strings.Index(param, " DEFAULT "); idx != -1 {
				param = param[:idx]
			}
			// Add IN mode prefix if not already present
			if !strings.HasPrefix(param, "IN ") && !strings.HasPrefix(param, "OUT ") && !strings.HasPrefix(param, "INOUT ") {
				param = "IN " + param
			}
			paramParts = append(paramParts, param)
		}
		return strings.Join(paramParts, ", ")
	}

	return ""
}
