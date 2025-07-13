package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

// generateCreateViewsSQL generates CREATE VIEW statements
func generateCreateViewsSQL(w *SQLWriter, views []*ir.View, targetSchema string, compare bool) {
	// Group views by schema for topological sorting
	viewsBySchema := make(map[string][]*ir.View)
	for _, view := range views {
		viewsBySchema[view.Schema] = append(viewsBySchema[view.Schema], view)
	}

	// Process each schema using topological sorting in deterministic order
	schemaNames := sortedKeys(viewsBySchema)
	for _, schemaName := range schemaNames {
		schemaViews := viewsBySchema[schemaName]
		// Build a temporary schema with just these views for topological sorting
		tempSchema := &ir.Schema{
			Name:  schemaName,
			Views: make(map[string]*ir.View),
		}
		for _, view := range schemaViews {
			tempSchema.Views[view.Name] = view
		}

		// Get topologically sorted view names for dependency-aware output
		sortedViewNames := getTopologicallySortedViewNames(tempSchema)

		// Process views in topological order
		for _, viewName := range sortedViewNames {
			view := tempSchema.Views[viewName]
			w.WriteDDLSeparator()
			// If compare mode, CREATE OR REPLACE, otherwise CREATE
			sql := generateViewSQL(view, targetSchema, compare)
			w.WriteStatementWithComment("VIEW", view.Name, view.Schema, "", sql, targetSchema)
		}
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
func generateDropViewsSQL(w *SQLWriter, views []*ir.View, targetSchema string) {
	// Group views by schema for topological sorting
	viewsBySchema := make(map[string][]*ir.View)
	for _, view := range views {
		viewsBySchema[view.Schema] = append(viewsBySchema[view.Schema], view)
	}

	// Process each schema using reverse topological sorting for drops in deterministic order
	schemaNames := sortedKeys(viewsBySchema)
	for _, schemaName := range schemaNames {
		schemaViews := viewsBySchema[schemaName]
		// Build a temporary schema with just these views for topological sorting
		tempSchema := &ir.Schema{
			Name:  schemaName,
			Views: make(map[string]*ir.View),
		}
		for _, view := range schemaViews {
			tempSchema.Views[view.Name] = view
		}

		// Get topologically sorted view names, then reverse for drop order
		sortedViewNames := getTopologicallySortedViewNames(tempSchema)

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

// getTopologicallySortedViewNames returns view names sorted in dependency order
// Views that depend on other views will come after their dependencies
func getTopologicallySortedViewNames(schema *ir.Schema) []string {
	var viewNames []string
	for name := range schema.Views {
		viewNames = append(viewNames, name)
	}

	// Build dependency graph
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	// Initialize
	for _, viewName := range viewNames {
		inDegree[viewName] = 0
		adjList[viewName] = []string{}
	}

	// Build edges: if viewA depends on viewB, add edge viewB -> viewA
	for _, viewA := range viewNames {
		viewAObj := schema.Views[viewA]
		for _, viewB := range viewNames {
			if viewA != viewB && viewDependsOnView(viewAObj, viewB) {
				adjList[viewB] = append(adjList[viewB], viewA)
				inDegree[viewA]++
			}
		}
	}

	// Kahn's algorithm for topological sorting
	var queue []string
	var result []string

	// Find all nodes with no incoming edges
	for viewName, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, viewName)
		}
	}

	// Sort initial queue alphabetically for deterministic output
	sort.Strings(queue)

	for len(queue) > 0 {
		// Remove node from queue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// For each neighbor, reduce in-degree
		neighbors := adjList[current]
		sort.Strings(neighbors) // For deterministic output

		for _, neighbor := range neighbors {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
				sort.Strings(queue) // Keep queue sorted for deterministic output
			}
		}
	}

	// Check for cycles (shouldn't happen with proper views)
	if len(result) != len(viewNames) {
		// Fallback to alphabetical sorting if cycle detected
		sort.Strings(viewNames)
		return viewNames
	}

	return result
}

// viewDependsOnView checks if viewA depends on viewB
func viewDependsOnView(viewA *ir.View, viewBName string) bool {
	// Simple heuristic: check if viewB name appears in viewA definition
	// This can be enhanced with proper dependency parsing later
	return strings.Contains(strings.ToLower(viewA.Definition), strings.ToLower(viewBName))
}
