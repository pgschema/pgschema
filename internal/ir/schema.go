package ir

import (
	"fmt"
	"sort"
	"time"
)

// Schema represents the complete database schema intermediate representation
type Schema struct {
	Metadata            Metadata               `json:"metadata"`
	Schemas             map[string]*DBSchema   `json:"schemas"`             // schema_name -> DBSchema
	Extensions          map[string]*Extension  `json:"extensions"`          // extension_name -> Extension
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

// GetSortedTableNames returns table names sorted alphabetically
func (ds *DBSchema) GetSortedTableNames() []string {
	var names []string
	for name := range ds.Tables {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GenerateSQL for DBSchema (schema creation)
func (ds *DBSchema) GenerateSQL() string {
	if ds.Name == "public" {
		return "" // Skip public schema
	}
	w := NewSQLWriter()
	stmt := fmt.Sprintf("CREATE SCHEMA %s;", ds.Name)
	w.WriteStatementWithComment("SCHEMA", ds.Name, ds.Name, "", stmt)
	return w.String()
}