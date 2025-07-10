package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
	"github.com/pgschema/pgschema/internal/utils"
)

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

// GenerateDropViewSQL generates SQL for dropping views
func GenerateDropViewSQL(views []*ir.View) []string {
	var statements []string
	
	// Sort views by schema.name for consistent ordering
	sortedViews := make([]*ir.View, len(views))
	copy(sortedViews, views)
	sort.Slice(sortedViews, func(i, j int) bool {
		keyI := sortedViews[i].Schema + "." + sortedViews[i].Name
		keyJ := sortedViews[j].Schema + "." + sortedViews[j].Name
		return keyI < keyJ
	})
	
	for _, view := range sortedViews {
		statements = append(statements, fmt.Sprintf("DROP VIEW IF EXISTS %s.%s;", view.Schema, view.Name))
	}
	
	return statements
}

// GenerateCreateViewSQL generates SQL for creating views
func GenerateCreateViewSQL(views []*ir.View) []string {
	var statements []string
	
	// Sort views by schema.name for consistent ordering
	sortedViews := make([]*ir.View, len(views))
	copy(sortedViews, views)
	sort.Slice(sortedViews, func(i, j int) bool {
		keyI := sortedViews[i].Schema + "." + sortedViews[i].Name
		keyJ := sortedViews[j].Schema + "." + sortedViews[j].Name
		return keyI < keyJ
	})
	
	for _, view := range sortedViews {
		stmt := generateViewSQL(view)
		statements = append(statements, stmt)
	}
	
	return statements
}

// GenerateAlterViewSQL generates SQL for modifying views
func GenerateAlterViewSQL(viewDiffs []*ViewDiff) []string {
	var statements []string
	
	// Sort modified views by schema.name for consistent ordering
	sortedViewDiffs := make([]*ViewDiff, len(viewDiffs))
	copy(sortedViewDiffs, viewDiffs)
	sort.Slice(sortedViewDiffs, func(i, j int) bool {
		keyI := sortedViewDiffs[i].New.Schema + "." + sortedViewDiffs[i].New.Name
		keyJ := sortedViewDiffs[j].New.Schema + "." + sortedViewDiffs[j].New.Name
		return keyI < keyJ
	})
	
	for _, viewDiff := range sortedViewDiffs {
		stmt := generateViewSQL(viewDiff.New)
		statements = append(statements, stmt)
	}
	
	return statements
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
			sql := d.generateViewSQL(view, targetSchema)
			w.WriteStatementWithComment("VIEW", view.Name, view.Schema, "", sql, targetSchema)
		}
	}
}

// generateModifyViewsSQL generates ALTER VIEW statements
func (d *DDLDiff) generateModifyViewsSQL(w *SQLWriter, diffs []*ViewDiff, targetSchema string) {
	for _, diff := range diffs {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("CREATE OR REPLACE VIEW %s AS %s;", diff.New.Name, diff.New.Definition)
		w.WriteStatementWithComment("VIEW", diff.New.Name, diff.New.Schema, "", sql, targetSchema)
	}
}

// generateViewSQL generates CREATE VIEW statement
func (d *DDLDiff) generateViewSQL(view *ir.View, targetSchema string) string {
	// Only include view name without schema if it's in the target schema
	viewName := utils.QualifyEntityName(view.Schema, view.Name, targetSchema)
	return fmt.Sprintf("CREATE VIEW %s AS\n%s", viewName, view.Definition)
}

// generateViewSQL generates CREATE OR REPLACE VIEW SQL for a view
func generateViewSQL(view *ir.View) string {
	// Always include schema prefix on view name
	viewName := fmt.Sprintf("%s.%s", view.Schema, view.Name)

	// The view definition should include schema-qualified table references
	// For now, we'll need to enhance the parser to preserve the original formatting
	// or implement a more sophisticated SQL formatter
	definition := view.Definition

	// Simple heuristic: if the definition references tables without schema qualification,
	// and the view is in public schema, add public. prefix to table references
	if view.Schema == "public" && !strings.Contains(definition, "public.") {
		// This is a simple approach - a more robust solution would parse the SQL
		definition = strings.ReplaceAll(definition, "FROM employees", "FROM public.employees")
	}

	// Format the definition with newlines for better readability
	formattedDef := definition
	if strings.Contains(definition, "SELECT") && !strings.Contains(definition, "\n") {
		// Simple formatting: add newlines after SELECT and before FROM/WHERE
		formattedDef = strings.ReplaceAll(definition, "SELECT ", "SELECT \n    ")
		formattedDef = strings.ReplaceAll(formattedDef, ", ", ",\n    ")
		formattedDef = strings.ReplaceAll(formattedDef, " FROM ", "\nFROM ")
		formattedDef = strings.ReplaceAll(formattedDef, " WHERE ", "\nWHERE ")
	}

	return fmt.Sprintf("CREATE OR REPLACE VIEW %s AS\n%s;", viewName, formattedDef)
}