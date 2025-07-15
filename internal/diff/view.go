package diff

import (
	"fmt"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

// generateCreateViewsSQL generates CREATE VIEW statements
// Views are assumed to be pre-sorted in topological order for dependency-aware creation
func generateCreateViewsSQL(w *SQLWriter, views []*ir.View, targetSchema string, compare bool) {
	// Process views in the provided order (already topologically sorted)
	for _, view := range views {
		w.WriteDDLSeparator()
		// If compare mode, CREATE OR REPLACE, otherwise CREATE
		sql := generateViewSQL(view, targetSchema, compare)
		w.WriteStatementWithComment("VIEW", view.Name, view.Schema, "", sql, targetSchema)
	}
}

// generateModifyViewsSQL generates CREATE OR REPLACE VIEW statements
func generateModifyViewsSQL(w *SQLWriter, diffs []*ViewDiff, targetSchema string) {
	for _, diff := range diffs {
		w.WriteDDLSeparator()
		sql := generateViewSQL(diff.New, targetSchema, true) // Use OR REPLACE for modified views
		w.WriteStatementWithComment("VIEW", diff.New.Name, diff.New.Schema, "", sql, targetSchema)
	}
}

// generateDropViewsSQL generates DROP VIEW statements
// Views are assumed to be pre-sorted in reverse topological order for dependency-aware dropping
func generateDropViewsSQL(w *SQLWriter, views []*ir.View, targetSchema string) {
	// Process views in the provided order (already reverse topologically sorted)
	for _, view := range views {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP VIEW IF EXISTS %s CASCADE;", view.Name)
		w.WriteStatementWithComment("VIEW", view.Name, view.Schema, "", sql, targetSchema)
	}
}

// generateViewSQL generates CREATE [OR REPLACE] VIEW statement
func generateViewSQL(view *ir.View, targetSchema string, useReplace bool) string {
	// Determine view name based on context
	var viewName string
	if targetSchema != "" {
		// For diff scenarios, use schema qualification logic
		viewName = qualifyEntityName(view.Schema, view.Name, targetSchema)
	} else {
		// For dump scenarios, always include schema prefix
		viewName = fmt.Sprintf("%s.%s", view.Schema, view.Name)
	}

	// Start with the raw definition
	definition := view.Definition

	// Remove any existing trailing semicolons to avoid duplication
	definition = strings.TrimSuffix(strings.TrimSpace(definition), ";")

	// Simple heuristic: if the definition references tables without schema qualification,
	// and the view is in public schema, add public. prefix to table references
	// Only do this for dump scenarios (when targetSchema is empty)
	if targetSchema == "" && view.Schema == "public" && !strings.Contains(definition, "public.") {
		// This is a simple approach - a more robust solution would parse the SQL
		definition = strings.ReplaceAll(definition, "FROM employees", "FROM public.employees")
	}

	// Format the definition with proper indentation
	var formattedDef string
	if strings.Contains(definition, "SELECT") && !strings.Contains(definition, "\n") {
		// Simple formatting: add newlines after SELECT and before FROM/WHERE
		formattedDef = strings.ReplaceAll(definition, "SELECT ", "SELECT \n    ")
		formattedDef = strings.ReplaceAll(formattedDef, ", ", ",\n    ")
		formattedDef = strings.ReplaceAll(formattedDef, " FROM ", "\nFROM ")
		formattedDef = strings.ReplaceAll(formattedDef, " WHERE ", "\nWHERE ")
	} else {
		// Keep existing formatting but ensure proper indentation for multi-line definitions
		formattedDef = definition
		// Add leading space to SELECT if it's at the beginning of the definition
		if strings.HasPrefix(formattedDef, "SELECT") {
			formattedDef = " " + formattedDef
		}
	}

	// Determine CREATE statement type
	createClause := "CREATE VIEW"
	if useReplace {
		createClause = "CREATE OR REPLACE VIEW"
	}

	return fmt.Sprintf("%s %s AS\n%s;", createClause, viewName, formattedDef)
}

// viewsEqual compares two views for equality
func viewsEqual(old, new *ir.View) bool {
	if old.Schema != new.Schema {
		return false
	}
	if old.Name != new.Name {
		return false
	}
	if old.Definition != new.Definition {
		return false
	}
	return true
}


// viewDependsOnView checks if viewA depends on viewB
func viewDependsOnView(viewA *ir.View, viewBName string) bool {
	// Simple heuristic: check if viewB name appears in viewA definition
	// This can be enhanced with proper dependency parsing later
	return strings.Contains(strings.ToLower(viewA.Definition), strings.ToLower(viewBName))
}
