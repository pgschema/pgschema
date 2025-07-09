package ir

// isBuiltInType returns true if the type is a built-in PostgreSQL type
func isBuiltInType(typeName string) bool {
	builtInTypes := map[string]bool{
		// Numeric types (canonical names)
		"smallint": true, "integer": true, "bigint": true, "decimal": true, "numeric": true,
		"real": true, "double precision": true, "smallserial": true, "serial": true, "bigserial": true,
		// Numeric types (internal names)
		"int2": true, "int4": true, "int8": true, "float4": true, "float8": true,
		// Monetary types
		"money": true,
		// Character types (canonical and internal names)
		"character varying": true, "varchar": true, "character": true, "char": true, "text": true, "bpchar": true,
		// Binary types
		"bytea": true,
		// Date/time types (canonical and internal names)
		"timestamp": true, "timestamp without time zone": true, "timestamp with time zone": true,
		"date": true, "time": true, "time without time zone": true, "time with time zone": true,
		"interval": true, "timestamptz": true, "timetz": true,
		// Boolean type (canonical and internal names)
		"boolean": true, "bool": true,
		// Enumerated types (built-in enums)
		// Geometric types
		"point": true, "line": true, "lseg": true, "box": true, "path": true, "polygon": true, "circle": true,
		// Network address types
		"cidr": true, "inet": true, "macaddr": true, "macaddr8": true,
		// Bit string types
		"bit": true, "bit varying": true,
		// Text search types
		"tsvector": true, "tsquery": true,
		// UUID type
		"uuid": true,
		// XML type
		"xml": true,
		// JSON types
		"json": true, "jsonb": true,
		// Range types
		"int4range": true, "int8range": true, "numrange": true, "tsrange": true, "tstzrange": true, "daterange": true,
		// Object identifier types
		"oid": true, "regclass": true, "regconfig": true, "regdictionary": true, "regnamespace": true,
		"regoper": true, "regoperator": true, "regproc": true, "regprocedure": true, "regrole": true, "regtype": true,
		// pg_lsn type
		"pg_lsn": true,
	}
	return builtInTypes[typeName]
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
	Name               string  `json:"name"`
	Position           int     `json:"position"` // ordinal_position
	DataType           string  `json:"data_type"`
	UDTName            string  `json:"udt_name,omitempty"`
	IsNullable         bool    `json:"is_nullable"`
	DefaultValue       *string `json:"default_value,omitempty"`
	MaxLength          *int    `json:"max_length,omitempty"`
	Precision          *int    `json:"precision,omitempty"`
	Scale              *int    `json:"scale,omitempty"`
	Comment            string  `json:"comment,omitempty"`
	IsIdentity         bool    `json:"is_identity,omitempty"`
	IdentityGeneration string  `json:"identity_generation,omitempty"` // "ALWAYS" or "BY DEFAULT"
	IdentityStart      *int64  `json:"identity_start,omitempty"`
	IdentityIncrement  *int64  `json:"identity_increment,omitempty"`
	IdentityMaximum    *int64  `json:"identity_maximum,omitempty"`
	IdentityMinimum    *int64  `json:"identity_minimum,omitempty"`
	IdentityCycle      bool    `json:"identity_cycle,omitempty"`
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

