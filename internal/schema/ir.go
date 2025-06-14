package schema

import (
	"sort"
	"time"
)

// Schema represents the complete database schema intermediate representation
type Schema struct {
	Metadata   Metadata             `json:"metadata"`
	Schemas    map[string]*DBSchema `json:"schemas"`    // schema_name -> DBSchema
	Extensions map[string]*Extension `json:"extensions"` // extension_name -> Extension
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
	Name      string                `json:"name"`
	Tables    map[string]*Table     `json:"tables"`    // table_name -> Table
	Views     map[string]*View      `json:"views"`     // view_name -> View
	Functions map[string]*Function  `json:"functions"` // function_name -> Function
	Sequences map[string]*Sequence  `json:"sequences"` // sequence_name -> Sequence
	Indexes   map[string]*Index     `json:"indexes"`   // index_name -> Index
	Triggers  map[string]*Trigger   `json:"triggers"`  // trigger_name -> Trigger
	Policies  map[string]*RLSPolicy `json:"policies"`  // policy_name -> RLSPolicy
}

// Table represents a database table
type Table struct {
	Schema            string                  `json:"schema"`
	Name              string                  `json:"name"`
	Type              TableType               `json:"type"` // BASE_TABLE, VIEW, etc.
	Columns           []*Column               `json:"columns"`
	Constraints       map[string]*Constraint  `json:"constraints"` // constraint_name -> Constraint
	Indexes           map[string]*Index       `json:"indexes"`     // index_name -> Index
	Triggers          map[string]*Trigger     `json:"triggers"`    // trigger_name -> Trigger
	RLSEnabled        bool                    `json:"rls_enabled"`
	Policies          map[string]*RLSPolicy   `json:"policies"` // policy_name -> RLSPolicy
	Dependencies      []TableDependency       `json:"dependencies"`
	Comment           string                  `json:"comment,omitempty"`
}

// Column represents a table column
type Column struct {
	Name         string      `json:"name"`
	Position     int         `json:"position"` // ordinal_position
	DataType     string      `json:"data_type"`
	UDTName      string      `json:"udt_name,omitempty"`
	IsNullable   bool        `json:"is_nullable"`
	DefaultValue *string     `json:"default_value,omitempty"`
	MaxLength    *int        `json:"max_length,omitempty"`
	Precision    *int        `json:"precision,omitempty"`
	Scale        *int        `json:"scale,omitempty"`
	Comment      string      `json:"comment,omitempty"`
}

// Constraint represents a table constraint
type Constraint struct {
	Schema           string         `json:"schema"`
	Table            string         `json:"table"`
	Name             string         `json:"name"`
	Type             ConstraintType `json:"type"`
	Columns          []*ConstraintColumn `json:"columns"`
	ReferencedSchema string         `json:"referenced_schema,omitempty"`
	ReferencedTable  string         `json:"referenced_table,omitempty"`
	ReferencedColumns []*ConstraintColumn `json:"referenced_columns,omitempty"`
	CheckClause      string         `json:"check_clause,omitempty"`
	DeleteRule       string         `json:"delete_rule,omitempty"`
	UpdateRule       string         `json:"update_rule,omitempty"`
	Deferrable       bool           `json:"deferrable,omitempty"`
	InitiallyDeferred bool          `json:"initially_deferred,omitempty"`
	Comment          string         `json:"comment,omitempty"`
}

// ConstraintColumn represents a column within a constraint with its position
type ConstraintColumn struct {
	Name     string `json:"name"`
	Position int    `json:"position"` // ordinal_position within the constraint
}

// Index represents a database index
type Index struct {
	Schema     string       `json:"schema"`
	Table      string       `json:"table"`
	Name       string       `json:"name"`
	Type       IndexType    `json:"type"`
	Method     string       `json:"method"` // btree, hash, gin, gist, etc.
	Columns    []*IndexColumn `json:"columns"`
	IsUnique   bool         `json:"is_unique"`
	IsPrimary  bool         `json:"is_primary"`
	IsPartial  bool         `json:"is_partial"`
	Where      string       `json:"where,omitempty"` // partial index condition
	Definition string       `json:"definition"`      // full CREATE INDEX statement
	Comment    string       `json:"comment,omitempty"`
}

// IndexColumn represents a column within an index
type IndexColumn struct {
	Name      string `json:"name"`
	Position  int    `json:"position"`
	Direction string `json:"direction,omitempty"` // ASC, DESC
	Operator  string `json:"operator,omitempty"`  // operator class
}

// View represents a database view
type View struct {
	Schema       string            `json:"schema"`
	Name         string            `json:"name"`
	Definition   string            `json:"definition"`
	Dependencies []TableDependency `json:"dependencies"`
	Comment      string            `json:"comment,omitempty"`
}

// Function represents a database function
type Function struct {
	Schema     string         `json:"schema"`
	Name       string         `json:"name"`
	Definition string         `json:"definition"`
	ReturnType string         `json:"return_type"`
	Language   string         `json:"language"`
	Parameters []*Parameter   `json:"parameters,omitempty"`
	Comment    string         `json:"comment,omitempty"`
}

// Parameter represents a function parameter
type Parameter struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
	Mode     string `json:"mode"` // IN, OUT, INOUT
	Position int    `json:"position"`
}

// Sequence represents a database sequence
type Sequence struct {
	Schema       string `json:"schema"`
	Name         string `json:"name"`
	DataType     string `json:"data_type"`
	StartValue   int64  `json:"start_value"`
	MinValue     *int64 `json:"min_value,omitempty"`
	MaxValue     *int64 `json:"max_value,omitempty"`
	Increment    int64  `json:"increment"`
	CycleOption  bool   `json:"cycle_option"`
	OwnedByTable string `json:"owned_by_table,omitempty"`
	OwnedByColumn string `json:"owned_by_column,omitempty"`
	Comment      string `json:"comment,omitempty"`
}

// Trigger represents a database trigger
type Trigger struct {
	Schema    string       `json:"schema"`
	Table     string       `json:"table"`
	Name      string       `json:"name"`
	Timing    TriggerTiming `json:"timing"` // BEFORE, AFTER, INSTEAD OF
	Events    []TriggerEvent `json:"events"` // INSERT, UPDATE, DELETE
	Level     TriggerLevel `json:"level"`  // ROW, STATEMENT
	Function  string       `json:"function"`
	Condition string       `json:"condition,omitempty"` // WHEN condition
	Comment   string       `json:"comment,omitempty"`
}

// RLSPolicy represents a Row Level Security policy
type RLSPolicy struct {
	Schema    string      `json:"schema"`
	Table     string      `json:"table"`
	Name      string      `json:"name"`
	Command   PolicyCommand `json:"command"` // SELECT, INSERT, UPDATE, DELETE, ALL
	Permissive bool       `json:"permissive"`
	Roles     []string    `json:"roles,omitempty"`
	Using     string      `json:"using,omitempty"`     // USING expression
	WithCheck string      `json:"with_check,omitempty"` // WITH CHECK expression
	Comment   string      `json:"comment,omitempty"`
}

// Extension represents a PostgreSQL extension
type Extension struct {
	Name    string `json:"name"`
	Schema  string `json:"schema"`
	Version string `json:"version"`
	Comment string `json:"comment,omitempty"`
}

// TableDependency represents a dependency between database objects
type TableDependency struct {
	Schema string         `json:"schema"`
	Name   string         `json:"name"`
	Type   DependencyType `json:"type"`
}

// Enums for type safety

type TableType string

const (
	TableTypeBase  TableType = "BASE_TABLE"
	TableTypeView  TableType = "VIEW"
	TableTypeTemp  TableType = "TEMPORARY"
)

type ConstraintType string

const (
	ConstraintTypePrimaryKey ConstraintType = "PRIMARY_KEY"
	ConstraintTypeUnique     ConstraintType = "UNIQUE"
	ConstraintTypeForeignKey ConstraintType = "FOREIGN_KEY"
	ConstraintTypeCheck      ConstraintType = "CHECK"
	ConstraintTypeExclusion  ConstraintType = "EXCLUSION"
)

type IndexType string

const (
	IndexTypeRegular    IndexType = "REGULAR"
	IndexTypePrimary    IndexType = "PRIMARY"
	IndexTypeUnique     IndexType = "UNIQUE"
	IndexTypePartial    IndexType = "PARTIAL"
	IndexTypeExpression IndexType = "EXPRESSION"
)

type TriggerTiming string

const (
	TriggerTimingBefore    TriggerTiming = "BEFORE"
	TriggerTimingAfter     TriggerTiming = "AFTER"
	TriggerTimingInsteadOf TriggerTiming = "INSTEAD_OF"
)

type TriggerEvent string

const (
	TriggerEventInsert   TriggerEvent = "INSERT"
	TriggerEventUpdate   TriggerEvent = "UPDATE"
	TriggerEventDelete   TriggerEvent = "DELETE"
	TriggerEventTruncate TriggerEvent = "TRUNCATE"
)

type TriggerLevel string

const (
	TriggerLevelRow       TriggerLevel = "ROW"
	TriggerLevelStatement TriggerLevel = "STATEMENT"
)

type PolicyCommand string

const (
	PolicyCommandAll    PolicyCommand = "ALL"
	PolicyCommandSelect PolicyCommand = "SELECT"
	PolicyCommandInsert PolicyCommand = "INSERT"
	PolicyCommandUpdate PolicyCommand = "UPDATE"
	PolicyCommandDelete PolicyCommand = "DELETE"
)

type DependencyType string

const (
	DependencyTypeTable    DependencyType = "TABLE"
	DependencyTypeView     DependencyType = "VIEW"
	DependencyTypeFunction DependencyType = "FUNCTION"
	DependencyTypeSequence DependencyType = "SEQUENCE"
)

// Helper methods for Schema

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
		Name:      name,
		Tables:    make(map[string]*Table),
		Views:     make(map[string]*View),
		Functions: make(map[string]*Function),
		Sequences: make(map[string]*Sequence),
		Indexes:   make(map[string]*Index),
		Triggers:  make(map[string]*Trigger),
		Policies:  make(map[string]*RLSPolicy),
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

// Helper methods for DBSchema

// GetSortedTableNames returns table names sorted alphabetically
func (ds *DBSchema) GetSortedTableNames() []string {
	var names []string
	for name := range ds.Tables {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetSortedConstraintNames returns constraint names sorted alphabetically
func (t *Table) GetSortedConstraintNames() []string {
	var names []string
	for name := range t.Constraints {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetSortedIndexNames returns index names sorted alphabetically
func (t *Table) GetSortedIndexNames() []string {
	var names []string
	for name := range t.Indexes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// SortColumnsByPosition sorts columns by their ordinal position
func (t *Table) SortColumnsByPosition() []*Column {
	columns := make([]*Column, len(t.Columns))
	copy(columns, t.Columns)
	sort.Slice(columns, func(i, j int) bool {
		return columns[i].Position < columns[j].Position
	})
	return columns
}

// SortConstraintColumnsByPosition sorts constraint columns by their position
func (c *Constraint) SortConstraintColumnsByPosition() []*ConstraintColumn {
	columns := make([]*ConstraintColumn, len(c.Columns))
	copy(columns, c.Columns)
	sort.Slice(columns, func(i, j int) bool {
		return columns[i].Position < columns[j].Position
	})
	return columns
}