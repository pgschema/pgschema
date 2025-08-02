package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
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
			Type:                "procedure",
			Operation:           "create",
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

		// For DROP statements, we need just the parameter types, not names
		paramTypes := extractParameterTypes(diff.Old)
		if paramTypes != "" {
			dropSQL = fmt.Sprintf("DROP PROCEDURE IF EXISTS %s(%s);", procedureName, paramTypes)
		} else {
			dropSQL = fmt.Sprintf("DROP PROCEDURE IF EXISTS %s();", procedureName)
		}

		// Create context for the drop statement
		dropContext := &diffContext{
			Type:                "procedure",
			Operation:           "drop",
			Path:                fmt.Sprintf("%s.%s", diff.Old.Schema, diff.Old.Name),
			Source:              diff,
			CanRunInTransaction: true,
		}

		collector.collect(dropContext, dropSQL)

		// Create the new procedure
		createSQL := generateProcedureSQL(diff.New, targetSchema)

		// Create context for the create statement
		createContext := &diffContext{
			Type:                "procedure",
			Operation:           "create",
			Path:                fmt.Sprintf("%s.%s", diff.New.Schema, diff.New.Name),
			Source:              diff,
			CanRunInTransaction: true,
		}

		collector.collect(createContext, createSQL)
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

		// For DROP statements, we need just the parameter types, not names
		// Extract types from the arguments/signature
		paramTypes := extractParameterTypes(procedure)
		if paramTypes != "" {
			sql = fmt.Sprintf("DROP PROCEDURE IF EXISTS %s(%s);", procedureName, paramTypes)
		} else {
			sql = fmt.Sprintf("DROP PROCEDURE IF EXISTS %s();", procedureName)
		}

		// Create context for this statement
		context := &diffContext{
			Type:                "procedure",
			Operation:           "drop",
			Path:                fmt.Sprintf("%s.%s", procedure.Schema, procedure.Name),
			Source:              procedure,
			CanRunInTransaction: true,
		}

		collector.collect(context, sql)
	}
}

// generateProcedureSQL generates CREATE OR REPLACE PROCEDURE SQL for a procedure
func generateProcedureSQL(procedure *ir.Procedure, targetSchema string) string {
	var stmt strings.Builder

	// Build the CREATE OR REPLACE PROCEDURE header with schema qualification
	procedureName := qualifyEntityName(procedure.Schema, procedure.Name, targetSchema)
	stmt.WriteString(fmt.Sprintf("CREATE OR REPLACE PROCEDURE %s", procedureName))

	// Add parameters using detailed signature if available
	if procedure.Signature != "" {
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

// extractParameterTypes extracts just the parameter types from a procedure's signature or arguments
// For example: "order_id integer, amount numeric" becomes "integer, numeric"
func extractParameterTypes(procedure *ir.Procedure) string {
	// Try to use Arguments field first as it should contain just types
	if procedure.Arguments != "" {
		// If Arguments contains parameter names (e.g., "order_id integer, amount numeric"),
		// extract just the types
		args := procedure.Arguments
		if strings.Contains(args, " ") {
			// This suggests parameter names are included, extract types
			var types []string
			params := strings.Split(args, ",")
			for _, param := range params {
				param = strings.TrimSpace(param)
				// Split by spaces and take the last part (the type)
				parts := strings.Fields(param)
				if len(parts) >= 2 {
					// Take the type (usually the second part: "name type")
					types = append(types, parts[1])
				} else if len(parts) == 1 {
					// If only one part, assume it's the type
					types = append(types, parts[0])
				}
			}
			return strings.Join(types, ", ")
		}
		// If no spaces, assume Arguments already contains just types
		return args
	}

	// Fallback to Signature field
	if procedure.Signature != "" {
		var types []string
		params := strings.Split(procedure.Signature, ",")
		for _, param := range params {
			param = strings.TrimSpace(param)
			// Remove DEFAULT clauses and extract type
			if strings.Contains(param, " DEFAULT ") {
				param = strings.Split(param, " DEFAULT ")[0]
			}
			// Split by spaces and take the type part
			parts := strings.Fields(param)
			if len(parts) >= 2 {
				types = append(types, parts[1])
			} else if len(parts) == 1 {
				types = append(types, parts[0])
			}
		}
		return strings.Join(types, ", ")
	}

	return ""
}
