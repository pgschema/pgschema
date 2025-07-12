package diff

import (
	"fmt"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

// generateCreateViewsSQL generates CREATE VIEW statements
func (d *DDLDiff) generateCreateViewsSQL(w *SQLWriter, views []*ir.View, targetSchema string) {
	// Group views by schema for topological sorting
	viewsBySchema := make(map[string][]*ir.View)
	for _, view := range views {
		viewsBySchema[view.Schema] = append(viewsBySchema[view.Schema], view)
	}

	// Process each schema using topological sorting
	for schemaName, schemaViews := range viewsBySchema {
		// Build a temporary schema with just these views for topological sorting
		tempSchema := &ir.Schema{
			Name:  schemaName,
			Views: make(map[string]*ir.View),
		}
		for _, view := range schemaViews {
			tempSchema.Views[view.Name] = view
		}

		// Get topologically sorted view names for dependency-aware output
		sortedViewNames := tempSchema.GetTopologicallySortedViewNames()

		// Process views in topological order
		for _, viewName := range sortedViewNames {
			view := tempSchema.Views[viewName]
			w.WriteDDLSeparator()
			// For CREATE scenarios (adding new views), detect if this is a dump or diff
			isDumpScenario := len(d.AddedTables) > 0 && len(d.DroppedTables) == 0 && len(d.ModifiedTables) == 0
			sql := d.generateViewSQLWithMode(view, targetSchema, !isDumpScenario)
			w.WriteStatementWithComment("VIEW", view.Name, view.Schema, "", sql, targetSchema)
		}
	}
}

// generateModifyViewsSQL generates ALTER VIEW statements
func (d *DDLDiff) generateModifyViewsSQL(w *SQLWriter, diffs []*ViewDiff, targetSchema string) {
	for _, diff := range diffs {
		w.WriteDDLSeparator()
		// For modify scenarios, always use CREATE OR REPLACE
		sql := d.generateViewSQLWithMode(diff.New, targetSchema, true)
		w.WriteStatementWithComment("VIEW", diff.New.Name, diff.New.Schema, "", sql, targetSchema)
	}
}

// generateDropViewsSQL generates DROP VIEW statements
func (d *DDLDiff) generateDropViewsSQL(w *SQLWriter, views []*ir.View, targetSchema string) {
	// Group views by schema for topological sorting
	viewsBySchema := make(map[string][]*ir.View)
	for _, view := range views {
		viewsBySchema[view.Schema] = append(viewsBySchema[view.Schema], view)
	}

	// Process each schema using reverse topological sorting for drops
	for schemaName, schemaViews := range viewsBySchema {
		// Build a temporary schema with just these views for topological sorting
		tempSchema := &ir.Schema{
			Name:  schemaName,
			Views: make(map[string]*ir.View),
		}
		for _, view := range schemaViews {
			tempSchema.Views[view.Name] = view
		}

		// Get topologically sorted view names, then reverse for drop order
		sortedViewNames := tempSchema.GetTopologicallySortedViewNames()

		// Reverse the order for dropping (dependencies first)
		for i := len(sortedViewNames) - 1; i >= 0; i-- {
			viewName := sortedViewNames[i]
			view := tempSchema.Views[viewName]
			w.WriteDDLSeparator()
			sql := fmt.Sprintf("DROP VIEW IF EXISTS %s CASCADE;", view.Name)
			w.WriteStatementWithComment("VIEW", view.Name, view.Schema, "", sql, targetSchema)
		}
	}
}

// generateViewSQLWithMode generates CREATE [OR REPLACE] VIEW statement
func (d *DDLDiff) generateViewSQLWithMode(view *ir.View, targetSchema string, useReplace bool) string {
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
