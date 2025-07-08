package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
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