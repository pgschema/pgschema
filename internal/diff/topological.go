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
	var insertionOrder []string
	for _, table := range tables {
		key := table.Schema + "." + table.Name
		tableMap[key] = table
		insertionOrder = append(insertionOrder, key)
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

	// Kahn's algorithm with deterministic cycle breaking
	var queue []string
	var result []string
	processed := make(map[string]bool, len(tableMap))

	// Seed queue with nodes that have no incoming edges
	for key, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, key)
		}
	}
	sort.Strings(queue)

	for len(result) < len(tableMap) {
		if len(queue) == 0 {
			// Cycle detected: pick the next unprocessed table using original insertion order
			next := nextInOrder(insertionOrder, processed)
			if next == "" {
				break
			}
			queue = append(queue, next)
			inDegree[next] = 0
		}

		current := queue[0]
		queue = queue[1:]
		if processed[current] {
			continue
		}
		processed[current] = true
		result = append(result, current)

		neighbors := append([]string(nil), adjList[current]...)
		sort.Strings(neighbors)

		for _, neighbor := range neighbors {
			inDegree[neighbor]--
			if inDegree[neighbor] <= 0 && !processed[neighbor] {
				queue = append(queue, neighbor)
				sort.Strings(queue)
			}
		}
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
	var insertionOrder []string
	for _, view := range views {
		key := view.Schema + "." + view.Name
		viewMap[key] = view
		insertionOrder = append(insertionOrder, key)
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

	// Kahn's algorithm with deterministic cycle breaking
	var queue []string
	var result []string
	processed := make(map[string]bool, len(viewMap))

	for key, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, key)
		}
	}
	sort.Strings(queue)

	for len(result) < len(viewMap) {
		if len(queue) == 0 {
			next := nextInOrder(insertionOrder, processed)
			if next == "" {
				break
			}
			queue = append(queue, next)
			inDegree[next] = 0
		}

		current := queue[0]
		queue = queue[1:]
		if processed[current] {
			continue
		}
		processed[current] = true
		result = append(result, current)

		neighbors := append([]string(nil), adjList[current]...)
		sort.Strings(neighbors)

		for _, neighbor := range neighbors {
			inDegree[neighbor]--
			if inDegree[neighbor] <= 0 && !processed[neighbor] {
				queue = append(queue, neighbor)
				sort.Strings(queue)
			}
		}
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

func nextInOrder(order []string, processed map[string]bool) string {
	for _, key := range order {
		if !processed[key] {
			return key
		}
	}
	return ""
}
