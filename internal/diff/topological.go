package diff

import (
	"sort"

	"github.com/pgschema/pgschema/ir"
)

// topologicallySortTables sorts tables across all schemas in dependency order
// Tables that are referenced by foreign keys will come before the tables that reference them
func topologicallySortTables(tables []*ir.Table) []*ir.Table {
	if len(tables) <= 1 {
		return tables
	}

	// Build maps for efficient lookup
	tableMap := make(map[string]*ir.Table)
	for _, table := range tables {
		key := table.Schema + "." + table.Name
		tableMap[key] = table
	}

	// Build dependency graph
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	// Initialize
	for key := range tableMap {
		inDegree[key] = 0
		adjList[key] = []string{}
	}

	// Build edges: if tableA has a foreign key to tableB, add edge tableB -> tableA
	for keyA, tableA := range tableMap {
		for _, constraint := range tableA.Constraints {
			if constraint.Type == ir.ConstraintTypeForeignKey && constraint.ReferencedTable != "" {
				// Build referenced table key
				referencedSchema := constraint.ReferencedSchema
				if referencedSchema == "" {
					referencedSchema = tableA.Schema // Default to same schema
				}
				keyB := referencedSchema + "." + constraint.ReferencedTable

				// Only add edge if referenced table exists in our set and is different
				if _, exists := tableMap[keyB]; exists && keyA != keyB {
					adjList[keyB] = append(adjList[keyB], keyA)
					inDegree[keyA]++
				}
			}
		}
	}

	// Kahn's algorithm for topological sorting
	var queue []string
	var result []string

	// Find all nodes with no incoming edges
	for key, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, key)
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

	// Check for cycles
	if len(result) != len(tableMap) {
		// Fallback to alphabetical sorting if cycle detected
		sortedTables := make([]*ir.Table, len(tables))
		copy(sortedTables, tables)
		sort.Slice(sortedTables, func(i, j int) bool {
			keyI := sortedTables[i].Schema + "." + sortedTables[i].Name
			keyJ := sortedTables[j].Schema + "." + sortedTables[j].Name
			return keyI < keyJ
		})
		return sortedTables
	}

	// Convert result back to table slice
	sortedTables := make([]*ir.Table, 0, len(result))
	for _, key := range result {
		sortedTables = append(sortedTables, tableMap[key])
	}

	return sortedTables
}

// topologicallySortViews sorts views across all schemas in dependency order
// Views that depend on other views will come after their dependencies
func topologicallySortViews(views []*ir.View) []*ir.View {
	if len(views) <= 1 {
		return views
	}

	// Build maps for efficient lookup
	viewMap := make(map[string]*ir.View)
	for _, view := range views {
		key := view.Schema + "." + view.Name
		viewMap[key] = view
	}

	// Build dependency graph
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	// Initialize
	for key := range viewMap {
		inDegree[key] = 0
		adjList[key] = []string{}
	}

	// Build edges: if viewA depends on viewB, add edge viewB -> viewA
	for keyA, viewA := range viewMap {
		for keyB, viewB := range viewMap {
			if keyA != keyB && viewDependsOnView(viewA, viewB.Name) {
				adjList[keyB] = append(adjList[keyB], keyA)
				inDegree[keyA]++
			}
		}
	}

	// Kahn's algorithm for topological sorting
	var queue []string
	var result []string

	// Find all nodes with no incoming edges
	for key, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, key)
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

	// Check for cycles
	if len(result) != len(viewMap) {
		// Fallback to alphabetical sorting if cycle detected
		sortedViews := make([]*ir.View, len(views))
		copy(sortedViews, views)
		sort.Slice(sortedViews, func(i, j int) bool {
			keyI := sortedViews[i].Schema + "." + sortedViews[i].Name
			keyJ := sortedViews[j].Schema + "." + sortedViews[j].Name
			return keyI < keyJ
		})
		return sortedViews
	}

	// Convert result back to view slice
	sortedViews := make([]*ir.View, 0, len(result))
	for _, key := range result {
		sortedViews = append(sortedViews, viewMap[key])
	}

	return sortedViews
}

// reverseSlice returns a new slice with elements in reverse order
func reverseSlice[T any](slice []T) []T {
	reversed := make([]T, len(slice))
	for i, v := range slice {
		reversed[len(slice)-1-i] = v
	}
	return reversed
}

