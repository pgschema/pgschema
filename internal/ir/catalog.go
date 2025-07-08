package ir

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pgschema/pgschema/internal/utils"
)

// Catalog represents the complete database schema intermediate representation
type Catalog struct {
	Metadata             Metadata               `json:"metadata"`
	Schemas              map[string]*Schema   `json:"schemas"`               // schema_name -> Schema
	Extensions           map[string]*Extension  `json:"extensions"`            // extension_name -> Extension
	PartitionAttachments []*PartitionAttachment `json:"partition_attachments"` // Table partition attachments
	IndexAttachments     []*IndexAttachment     `json:"index_attachments"`     // Index partition attachments
}

// Metadata contains information about the schema dump
type Metadata struct {
	DatabaseVersion string    `json:"database_version"`
	DumpVersion     string    `json:"dump_version"`
	DumpedAt        time.Time `json:"dumped_at"`
	Source          string    `json:"source"` // "pgschema", "pg_dump", etc.
}

// Schema represents a single database schema (namespace)
type Schema struct {
	Name       string                `json:"name"`
	Owner      string                `json:"owner"`      // Schema owner
	Tables     map[string]*Table     `json:"tables"`     // table_name -> Table
	Views      map[string]*View      `json:"views"`      // view_name -> View
	Functions  map[string]*Function  `json:"functions"`  // function_name -> Function
	Procedures map[string]*Procedure `json:"procedures"` // procedure_name -> Procedure
	Aggregates map[string]*Aggregate `json:"aggregates"` // aggregate_name -> Aggregate
	Sequences  map[string]*Sequence  `json:"sequences"`  // sequence_name -> Sequence
	Policies   map[string]*RLSPolicy `json:"policies"`   // policy_name -> RLSPolicy
	Types      map[string]*Type      `json:"types"`      // type_name -> Type
	// Note: Indexes and Triggers are stored at table level (Table.Indexes, Table.Triggers)
}

// NewCatalog creates a new empty catalog IR
func NewCatalog() *Catalog {
	return &Catalog{
		Schemas:    make(map[string]*Schema),
		Extensions: make(map[string]*Extension),
	}
}

// GetOrCreateSchema gets or creates a database schema by name
func (c *Catalog) GetOrCreateSchema(name string) *Schema {
	if schema, exists := c.Schemas[name]; exists {
		return schema
	}

	schema := &Schema{
		Name:       name,
		Tables:     make(map[string]*Table),
		Views:      make(map[string]*View),
		Functions:  make(map[string]*Function),
		Procedures: make(map[string]*Procedure),
		Aggregates: make(map[string]*Aggregate),
		Sequences:  make(map[string]*Sequence),
		Policies:   make(map[string]*RLSPolicy),
		Types:      make(map[string]*Type),
	}
	c.Schemas[name] = schema
	return schema
}

// GetSortedSchemaNames returns schema names sorted alphabetically
func (c *Catalog) GetSortedSchemaNames() []string {
	return utils.SortedKeys(c.Schemas)
}

// GetSortedExtensionNames returns extension names sorted alphabetically
func (c *Catalog) GetSortedExtensionNames() []string {
	return utils.SortedKeys(c.Extensions)
}

// GetSortedTableNames returns table names sorted alphabetically
func (s *Schema) GetSortedTableNames() []string {
	return utils.SortedKeys(s.Tables)
}

// GetTopologicallySortedTableNames returns table names sorted in dependency order
// Tables that are referenced by foreign keys will come before the tables that reference them
func (s *Schema) GetTopologicallySortedTableNames() []string {
	var tableNames []string
	for name := range s.Tables {
		tableNames = append(tableNames, name)
	}

	// Build dependency graph
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	// Initialize
	for _, tableName := range tableNames {
		inDegree[tableName] = 0
		adjList[tableName] = []string{}
	}

	// Build edges: if tableA has a foreign key to tableB, add edge tableB -> tableA
	for _, tableA := range tableNames {
		tableAObj := s.Tables[tableA]
		for _, constraint := range tableAObj.Constraints {
			if constraint.Type == ConstraintTypeForeignKey && constraint.ReferencedTable != "" {
				// Only consider dependencies within the same schema
				if constraint.ReferencedSchema == s.Name || constraint.ReferencedSchema == "" {
					tableB := constraint.ReferencedTable
					// Only add edge if referenced table exists in this schema
					if _, exists := s.Tables[tableB]; exists && tableA != tableB {
						adjList[tableB] = append(adjList[tableB], tableA)
						inDegree[tableA]++
					}
				}
			}
		}
	}

	// Kahn's algorithm for topological sorting
	var queue []string
	var result []string

	// Find all nodes with no incoming edges
	for tableName, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, tableName)
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

	// Check for cycles (shouldn't happen with proper foreign keys)
	if len(result) != len(tableNames) {
		// Fallback to alphabetical sorting if cycle detected
		sort.Strings(tableNames)
		return tableNames
	}

	return result
}

// GetSortedPolicyNames returns policy names sorted alphabetically
func (s *Schema) GetSortedPolicyNames() []string {
	return utils.SortedKeys(s.Policies)
}


// GetSortedSequenceNames returns sequence names sorted alphabetically
func (s *Schema) GetSortedSequenceNames() []string {
	return utils.SortedKeys(s.Sequences)
}

// GetSortedFunctionNames returns function names sorted alphabetically
func (s *Schema) GetSortedFunctionNames() []string {
	return utils.SortedKeys(s.Functions)
}

// GetSortedAggregateNames returns aggregate names sorted alphabetically
func (s *Schema) GetSortedAggregateNames() []string {
	return utils.SortedKeys(s.Aggregates)
}

// GetSortedProcedureNames returns procedure names sorted alphabetically
func (s *Schema) GetSortedProcedureNames() []string {
	return utils.SortedKeys(s.Procedures)
}


// GetTopologicallySortedViewNames returns view names sorted in dependency order
// Views that depend on other views will come after their dependencies
func (s *Schema) GetTopologicallySortedViewNames() []string {
	var viewNames []string
	for name := range s.Views {
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
		viewAObj := s.Views[viewA]
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
func viewDependsOnView(viewA *View, viewBName string) bool {
	// Simple heuristic: check if viewB name appears in viewA definition
	// This can be enhanced with proper dependency parsing later
	return strings.Contains(strings.ToLower(viewA.Definition), strings.ToLower(viewBName))
}

// GenerateSQL for Schema (schema creation)
func (s *Schema) GenerateSQL() string {
	if s.Name == "public" {
		return "" // Skip public schema
	}
	w := NewSQLWriter()
	stmt := fmt.Sprintf("CREATE SCHEMA %s;", s.Name)
	w.WriteStatementWithComment("SCHEMA", s.Name, s.Name, "", stmt, "")
	return w.String()
}
