package ir


// IR represents the complete database schema intermediate representation
type IR struct {
	Metadata             Metadata               `json:"metadata"`
	Schemas              map[string]*Schema     `json:"schemas"`               // schema_name -> Schema
	PartitionAttachments []*PartitionAttachment `json:"partition_attachments"` // Table partition attachments
}

// Metadata contains information about the schema dump
type Metadata struct {
	DatabaseVersion string `json:"database_version"`
}

// Schema represents a single database schema (namespace)
type Schema struct {
	Name  string `json:"name"`
	Owner string `json:"owner"` // Schema owner
	// Note: Indexes, Triggers, and RLS Policies are stored at table level (Table.Indexes, Table.Triggers, Table.Policies)
	Tables     map[string]*Table     `json:"tables"`     // table_name -> Table
	Views      map[string]*View      `json:"views"`      // view_name -> View
	Functions  map[string]*Function  `json:"functions"`  // function_name -> Function
	Procedures map[string]*Procedure `json:"procedures"` // procedure_name -> Procedure
	Aggregates map[string]*Aggregate `json:"aggregates"` // aggregate_name -> Aggregate
	Sequences  map[string]*Sequence  `json:"sequences"`  // sequence_name -> Sequence
	Types      map[string]*Type      `json:"types"`      // type_name -> Type
}

// Table represents a database table
type Table struct {
	Schema            string                 `json:"schema"`
	Name              string                 `json:"name"`
	Type              TableType              `json:"type"` // BASE_TABLE, VIEW, etc.
	Columns           []*Column              `json:"columns"`
	Constraints       map[string]*Constraint `json:"constraints"` // constraint_name -> Constraint
	Indexes           map[string]*Index      `json:"indexes"`     // index_name -> Index
	Triggers          map[string]*Trigger    `json:"triggers"`    // trigger_name -> Trigger
	RLSEnabled        bool                   `json:"rls_enabled"`
	Policies          map[string]*RLSPolicy  `json:"policies"` // policy_name -> RLSPolicy
	Dependencies      []TableDependency      `json:"dependencies"`
	Comment           string                 `json:"comment,omitempty"`
	IsPartitioned     bool                   `json:"is_partitioned"`
	PartitionStrategy string                 `json:"partition_strategy,omitempty"` // RANGE, LIST, HASH
	PartitionKey      string                 `json:"partition_key,omitempty"`      // Column(s) used for partitioning
}

// Column represents a table column
type Column struct {
	Name         string    `json:"name"`
	Position     int       `json:"position"` // ordinal_position
	DataType     string    `json:"data_type"`
	IsNullable   bool      `json:"is_nullable"`
	DefaultValue *string   `json:"default_value,omitempty"`
	MaxLength    *int      `json:"max_length,omitempty"`
	Precision    *int      `json:"precision,omitempty"`
	Scale        *int      `json:"scale,omitempty"`
	Comment      string    `json:"comment,omitempty"`
	Identity     *Identity `json:"identity,omitempty"`
}

// Identity represents PostgreSQL identity column configuration
type Identity struct {
	Generation string `json:"generation,omitempty"` // "ALWAYS" or "BY DEFAULT"
	Start      *int64 `json:"start,omitempty"`
	Increment  *int64 `json:"increment,omitempty"`
	Maximum    *int64 `json:"maximum,omitempty"`
	Minimum    *int64 `json:"minimum,omitempty"`
	Cycle      bool   `json:"cycle,omitempty"`
}

// TableType represents different types of table objects
type TableType string

const (
	TableTypeBase TableType = "BASE_TABLE"
	TableTypeView TableType = "VIEW"
	TableTypeTemp TableType = "TEMPORARY"
)

// DependencyType represents different types of database object dependencies
type DependencyType string

const (
	DependencyTypeTable    DependencyType = "TABLE"
	DependencyTypeView     DependencyType = "VIEW"
	DependencyTypeFunction DependencyType = "FUNCTION"
	DependencyTypeSequence DependencyType = "SEQUENCE"
)

// TableDependency represents a dependency between database objects
type TableDependency struct {
	Schema string         `json:"schema"`
	Name   string         `json:"name"`
	Type   DependencyType `json:"type"`
}

// View represents a database view
type View struct {
	Schema       string `json:"schema"`
	Name         string `json:"name"`
	Definition   string `json:"definition"`
	Comment      string `json:"comment,omitempty"`
	Materialized bool   `json:"materialized,omitempty"`
}

// Function represents a database function
type Function struct {
	Schema            string       `json:"schema"`
	Name              string       `json:"name"`
	Definition        string       `json:"definition"`
	ReturnType        string       `json:"return_type"`
	Language          string       `json:"language"`
	Arguments         string       `json:"arguments,omitempty"`
	Signature         string       `json:"signature,omitempty"`
	Parameters        []*Parameter `json:"parameters,omitempty"`
	Comment           string       `json:"comment,omitempty"`
	Volatility        string       `json:"volatility,omitempty"`          // IMMUTABLE, STABLE, VOLATILE
	IsStrict          bool         `json:"is_strict,omitempty"`           // STRICT or null behavior
	IsSecurityDefiner bool         `json:"is_security_definer,omitempty"` // SECURITY DEFINER
}

// Parameter represents a function parameter
type Parameter struct {
	Name         string  `json:"name"`
	DataType     string  `json:"data_type"`
	Mode         string  `json:"mode"` // IN, OUT, INOUT
	Position     int     `json:"position"`
	DefaultValue *string `json:"default_value,omitempty"`
}

// Sequence represents a database sequence
type Sequence struct {
	Schema        string `json:"schema"`
	Name          string `json:"name"`
	DataType      string `json:"data_type"`
	StartValue    int64  `json:"start_value"`
	MinValue      *int64 `json:"min_value,omitempty"`
	MaxValue      *int64 `json:"max_value,omitempty"`
	Increment     int64  `json:"increment"`
	CycleOption   bool   `json:"cycle_option"`
	OwnedByTable  string `json:"owned_by_table,omitempty"`
	OwnedByColumn string `json:"owned_by_column,omitempty"`
	Comment       string `json:"comment,omitempty"`
}

// Constraint represents a table constraint
type Constraint struct {
	Schema            string              `json:"schema"`
	Table             string              `json:"table"`
	Name              string              `json:"name"`
	Type              ConstraintType      `json:"type"`
	Columns           []*ConstraintColumn `json:"columns"`
	ReferencedSchema  string              `json:"referenced_schema,omitempty"`
	ReferencedTable   string              `json:"referenced_table,omitempty"`
	ReferencedColumns []*ConstraintColumn `json:"referenced_columns,omitempty"`
	CheckClause       string              `json:"check_clause,omitempty"`
	DeleteRule        string              `json:"delete_rule,omitempty"`
	UpdateRule        string              `json:"update_rule,omitempty"`
	Deferrable        bool                `json:"deferrable,omitempty"`
	InitiallyDeferred bool                `json:"initially_deferred,omitempty"`
	Comment           string              `json:"comment,omitempty"`
}

// ConstraintColumn represents a column within a constraint with its position
type ConstraintColumn struct {
	Name     string `json:"name"`
	Position int    `json:"position"` // ordinal_position within the constraint
}

// ConstraintType represents different types of database constraints
type ConstraintType string

const (
	ConstraintTypePrimaryKey ConstraintType = "PRIMARY_KEY"
	ConstraintTypeUnique     ConstraintType = "UNIQUE"
	ConstraintTypeForeignKey ConstraintType = "FOREIGN_KEY"
	ConstraintTypeCheck      ConstraintType = "CHECK"
	ConstraintTypeExclusion  ConstraintType = "EXCLUSION"
)

// Index represents a database index
type Index struct {
	Schema       string         `json:"schema"`
	Table        string         `json:"table"`
	Name         string         `json:"name"`
	Type         IndexType      `json:"type"`
	Method       string         `json:"method"` // btree, hash, gin, gist, etc.
	Columns      []*IndexColumn `json:"columns"`
	IsConcurrent bool           `json:"is_concurrent"`
	IsPartial    bool           `json:"is_partial"`      // has a WHERE clause
	IsExpression bool           `json:"is_expression"`   // functional/expression index
	Where        string         `json:"where,omitempty"` // partial index condition
	Comment      string         `json:"comment,omitempty"`
}

// IndexColumn represents a column within an index
type IndexColumn struct {
	Name      string `json:"name"`
	Position  int    `json:"position"`
	Direction string `json:"direction,omitempty"` // ASC, DESC
	Operator  string `json:"operator,omitempty"`  // operator class
}

// IndexType represents different types of database indexes
type IndexType string

const (
	IndexTypeRegular IndexType = "REGULAR"
	IndexTypePrimary IndexType = "PRIMARY"
	IndexTypeUnique  IndexType = "UNIQUE"
)

// Trigger represents a database trigger
type Trigger struct {
	Schema    string         `json:"schema"`
	Table     string         `json:"table"`
	Name      string         `json:"name"`
	Timing    TriggerTiming  `json:"timing"` // BEFORE, AFTER, INSTEAD OF
	Events    []TriggerEvent `json:"events"` // INSERT, UPDATE, DELETE
	Level     TriggerLevel   `json:"level"`  // ROW, STATEMENT
	Function  string         `json:"function"`
	Condition string         `json:"condition,omitempty"` // WHEN condition
	Comment   string         `json:"comment,omitempty"`
}

// TriggerTiming represents the timing of trigger execution
type TriggerTiming string

const (
	TriggerTimingBefore    TriggerTiming = "BEFORE"
	TriggerTimingAfter     TriggerTiming = "AFTER"
	TriggerTimingInsteadOf TriggerTiming = "INSTEAD_OF"
)

// TriggerEvent represents the event that triggers the trigger
type TriggerEvent string

const (
	TriggerEventInsert   TriggerEvent = "INSERT"
	TriggerEventUpdate   TriggerEvent = "UPDATE"
	TriggerEventDelete   TriggerEvent = "DELETE"
	TriggerEventTruncate TriggerEvent = "TRUNCATE"
)

// TriggerLevel represents the level at which the trigger fires
type TriggerLevel string

const (
	TriggerLevelRow       TriggerLevel = "ROW"
	TriggerLevelStatement TriggerLevel = "STATEMENT"
)


// RLSPolicy represents a Row Level Security policy
type RLSPolicy struct {
	Schema     string        `json:"schema"`
	Table      string        `json:"table"`
	Name       string        `json:"name"`
	Command    PolicyCommand `json:"command"` // SELECT, INSERT, UPDATE, DELETE, ALL
	Permissive bool          `json:"permissive"`
	Roles      []string      `json:"roles,omitempty"`
	Using      string        `json:"using,omitempty"`      // USING expression
	WithCheck  string        `json:"with_check,omitempty"` // WITH CHECK expression
	Comment    string        `json:"comment,omitempty"`
}

// PolicyCommand represents the command for which the policy applies
type PolicyCommand string

const (
	PolicyCommandAll    PolicyCommand = "ALL"
	PolicyCommandSelect PolicyCommand = "SELECT"
	PolicyCommandInsert PolicyCommand = "INSERT"
	PolicyCommandUpdate PolicyCommand = "UPDATE"
	PolicyCommandDelete PolicyCommand = "DELETE"
)

// TypeKind represents the kind of PostgreSQL type
type TypeKind string

const (
	TypeKindEnum      TypeKind = "ENUM"
	TypeKindComposite TypeKind = "COMPOSITE"
	TypeKindDomain    TypeKind = "DOMAIN"
)

// TypeColumn represents a column in a composite type
type TypeColumn struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
	Position int    `json:"position"`
}

// DomainConstraint represents a constraint on a domain
type DomainConstraint struct {
	Name       string `json:"name"`
	Definition string `json:"definition"`
}

// Type represents a PostgreSQL user-defined type
type Type struct {
	Schema      string              `json:"schema"`
	Name        string              `json:"name"`
	Kind        TypeKind            `json:"kind"`
	Comment     string              `json:"comment,omitempty"`
	EnumValues  []string            `json:"enum_values,omitempty"` // For ENUM types
	Columns     []*TypeColumn       `json:"columns,omitempty"`     // For composite types
	BaseType    string              `json:"base_type,omitempty"`   // For DOMAIN types
	NotNull     bool                `json:"not_null,omitempty"`    // For DOMAIN types
	Default     string              `json:"default,omitempty"`     // For DOMAIN types
	Constraints []*DomainConstraint `json:"constraints,omitempty"` // For DOMAIN types
}

// Aggregate represents a database aggregate function
type Aggregate struct {
	Schema                   string `json:"schema"`
	Name                     string `json:"name"`
	Arguments                string `json:"arguments,omitempty"`
	Signature                string `json:"signature,omitempty"`
	ReturnType               string `json:"return_type"`
	TransitionFunction       string `json:"transition_function"`
	TransitionFunctionSchema string `json:"transition_function_schema,omitempty"`
	StateType                string `json:"state_type"`
	InitialCondition         string `json:"initial_condition,omitempty"`
	FinalFunction            string `json:"final_function,omitempty"`
	FinalFunctionSchema      string `json:"final_function_schema,omitempty"`
	Comment                  string `json:"comment,omitempty"`
}

// Procedure represents a database procedure
type Procedure struct {
	Schema     string       `json:"schema"`
	Name       string       `json:"name"`
	Definition string       `json:"definition"`
	Language   string       `json:"language"`
	Arguments  string       `json:"arguments,omitempty"`
	Signature  string       `json:"signature,omitempty"`
	Parameters []*Parameter `json:"parameters,omitempty"`
	Comment    string       `json:"comment,omitempty"`
}

// PartitionAttachment represents a partition child table attachment
type PartitionAttachment struct {
	ParentSchema   string `json:"parent_schema"`
	ParentTable    string `json:"parent_table"`
	ChildSchema    string `json:"child_schema"`
	ChildTable     string `json:"child_table"`
	PartitionBound string `json:"partition_bound"`
}


// NewIR creates a new empty catalog IR
func NewIR() *IR {
	return &IR{
		Schemas:    make(map[string]*Schema),
	}
}


// getOrCreateSchema gets or creates a database schema by name
func (c *IR) getOrCreateSchema(name string) *Schema {
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
		Types:      make(map[string]*Type),
	}
	c.Schemas[name] = schema
	return schema
}
