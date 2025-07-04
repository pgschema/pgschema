package ir

import (
	"fmt"
	"sort"
	"strings"
	"time"
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
	var names []string
	for name := range s.Schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetSortedExtensionNames returns extension names sorted alphabetically
func (s *Schema) GetSortedExtensionNames() []string {
	var names []string
	for name := range s.Extensions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetSortedTableNames returns table names sorted alphabetically
func (ds *DBSchema) GetSortedTableNames() []string {
	var names []string
	for name := range ds.Tables {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetSortedPolicyNames returns policy names sorted alphabetically
func (ds *DBSchema) GetSortedPolicyNames() []string {
	var names []string
	for name := range ds.Policies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetSortedIndexNames returns index names sorted alphabetically
func (ds *DBSchema) GetSortedIndexNames() []string {
	var names []string
	for name := range ds.Indexes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetSortedSequenceNames returns sequence names sorted alphabetically
func (ds *DBSchema) GetSortedSequenceNames() []string {
	var names []string
	for name := range ds.Sequences {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetSortedFunctionNames returns function names sorted alphabetically
func (ds *DBSchema) GetSortedFunctionNames() []string {
	var names []string
	for name := range ds.Functions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetSortedAggregateNames returns aggregate names sorted alphabetically
func (ds *DBSchema) GetSortedAggregateNames() []string {
	var names []string
	for name := range ds.Aggregates {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetSortedProcedureNames returns procedure names sorted alphabetically
func (ds *DBSchema) GetSortedProcedureNames() []string {
	var names []string
	for name := range ds.Procedures {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetSortedTriggerNames returns trigger names sorted alphabetically
func (ds *DBSchema) GetSortedTriggerNames() []string {
	var names []string
	for name := range ds.Triggers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
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
