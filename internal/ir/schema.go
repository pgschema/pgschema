package ir

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pgschema/pgschema/internal/utils"
)

// Schema represents the complete database schema intermediate representation
type Schema struct {
	Metadata             Metadata               `json:"metadata"`
	Schemas              map[string]*DBSchema   `json:"schemas"`               // schema_name -> DBSchema
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

// DBSchema represents a single database schema (namespace)
type DBSchema struct {
	Name       string                `json:"name"`
	Owner      string                `json:"owner"`      // Schema owner
	Tables     map[string]*Table     `json:"tables"`     // table_name -> Table
	Views      map[string]*View      `json:"views"`      // view_name -> View
	Functions  map[string]*Function  `json:"functions"`  // function_name -> Function
	Procedures map[string]*Procedure `json:"procedures"` // procedure_name -> Procedure
	Aggregates map[string]*Aggregate `json:"aggregates"` // aggregate_name -> Aggregate
	Sequences  map[string]*Sequence  `json:"sequences"`  // sequence_name -> Sequence
	Indexes    map[string]*Index     `json:"indexes"`    // index_name -> Index
	Triggers   map[string]*Trigger   `json:"triggers"`   // trigger_name -> Trigger
	Policies   map[string]*RLSPolicy `json:"policies"`   // policy_name -> RLSPolicy
	Types      map[string]*Type      `json:"types"`      // type_name -> Type
}

// NewSchema creates a new empty schema IR
func NewSchema() *Schema {
	return &Schema{
		Schemas:    make(map[string]*DBSchema),
		Extensions: make(map[string]*Extension),
	}
}

// GetOrCreateSchema gets or creates a database schema by name
func (s *Schema) GetOrCreateSchema(name string) *DBSchema {
	if schema, exists := s.Schemas[name]; exists {
		return schema
	}

	schema := &DBSchema{
		Name:       name,
		Tables:     make(map[string]*Table),
		Views:      make(map[string]*View),
		Functions:  make(map[string]*Function),
		Procedures: make(map[string]*Procedure),
		Aggregates: make(map[string]*Aggregate),
		Sequences:  make(map[string]*Sequence),
		Indexes:    make(map[string]*Index),
		Triggers:   make(map[string]*Trigger),
		Policies:   make(map[string]*RLSPolicy),
		Types:      make(map[string]*Type),
	}
	s.Schemas[name] = schema
	return schema
}

// GetSortedSchemaNames returns schema names sorted alphabetically
func (s *Schema) GetSortedSchemaNames() []string {
	return utils.SortedKeys(s.Schemas)
}

// GetSortedExtensionNames returns extension names sorted alphabetically
func (s *Schema) GetSortedExtensionNames() []string {
	return utils.SortedKeys(s.Extensions)
}

// GetSortedTableNames returns table names sorted alphabetically
func (ds *DBSchema) GetSortedTableNames() []string {
	return utils.SortedKeys(ds.Tables)
}

// GetTopologicallySortedTableNames returns table names sorted in dependency order
// Tables that are referenced by foreign keys will come before the tables that reference them
func (ds *DBSchema) GetTopologicallySortedTableNames() []string {
	var tableNames []string
	for name := range ds.Tables {
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
		tableAObj := ds.Tables[tableA]
		for _, constraint := range tableAObj.Constraints {
			if constraint.Type == ConstraintTypeForeignKey && constraint.ReferencedTable != "" {
				// Only consider dependencies within the same schema
				if constraint.ReferencedSchema == ds.Name || constraint.ReferencedSchema == "" {
					tableB := constraint.ReferencedTable
					// Only add edge if referenced table exists in this schema
					if _, exists := ds.Tables[tableB]; exists && tableA != tableB {
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
func (ds *DBSchema) GetSortedPolicyNames() []string {
	return utils.SortedKeys(ds.Policies)
}

// GetSortedIndexNames returns index names sorted alphabetically
func (ds *DBSchema) GetSortedIndexNames() []string {
	return utils.SortedKeys(ds.Indexes)
}

// GetSortedSequenceNames returns sequence names sorted alphabetically
func (ds *DBSchema) GetSortedSequenceNames() []string {
	return utils.SortedKeys(ds.Sequences)
}

// GetSortedFunctionNames returns function names sorted alphabetically
func (ds *DBSchema) GetSortedFunctionNames() []string {
	return utils.SortedKeys(ds.Functions)
}

// GetSortedAggregateNames returns aggregate names sorted alphabetically
func (ds *DBSchema) GetSortedAggregateNames() []string {
	return utils.SortedKeys(ds.Aggregates)
}

// GetSortedProcedureNames returns procedure names sorted alphabetically
func (ds *DBSchema) GetSortedProcedureNames() []string {
	return utils.SortedKeys(ds.Procedures)
}

// GetSortedTriggerNames returns trigger names sorted alphabetically
func (ds *DBSchema) GetSortedTriggerNames() []string {
	return utils.SortedKeys(ds.Triggers)
}

// GetTopologicallySortedViewNames returns view names sorted in dependency order
// Views that depend on other views will come after their dependencies
func (ds *DBSchema) GetTopologicallySortedViewNames() []string {
	var viewNames []string
	for name := range ds.Views {
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
		viewAObj := ds.Views[viewA]
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

// GenerateSQL for DBSchema (schema creation)
func (ds *DBSchema) GenerateSQL() string {
	if ds.Name == "public" {
		return "" // Skip public schema
	}
	w := NewSQLWriter()
	stmt := fmt.Sprintf("CREATE SCHEMA %s;", ds.Name)
	w.WriteStatementWithComment("SCHEMA", ds.Name, ds.Name, "", stmt, "")
	return w.String()
}
