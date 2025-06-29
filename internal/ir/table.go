package ir

import (
	"fmt"
	"sort"
	"strings"
)

// canonicalizeTypeName converts internal PostgreSQL type names to their canonical SQL names
// This matches pg_dump behavior for type name output
func canonicalizeTypeName(typeName string) string {
	typeMapping := map[string]string{
		// Integer types
		"int2": "smallint",
		"int4": "integer", 
		"int8": "bigint",
		// Float types
		"float4": "real",
		"float8": "double precision",
		// Boolean type
		"bool": "boolean",
		// Character types
		"varchar": "character varying",
		"bpchar": "character",
		// Date/time types
		"timestamptz": "timestamp with time zone",
		"timetz": "time with time zone",
		// Other common internal names
		"numeric": "numeric", // keep as-is
		"text": "text",       // keep as-is
		// Serial types (keep as uppercase)
		"serial": "SERIAL",
		"smallserial": "SMALLSERIAL", 
		"bigserial": "BIGSERIAL",
	}
	
	if canonical, exists := typeMapping[typeName]; exists {
		return canonical
	}
	return typeName
}

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

// GetSortedConstraintNames returns constraint names sorted alphabetically
func (t *Table) GetSortedConstraintNames() []string {
	var names []string
	for name := range t.Constraints {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetCheckConstraints returns CHECK constraints sorted by name
func (t *Table) GetCheckConstraints() []*Constraint {
	var checkConstraints []*Constraint
	constraintNames := t.GetSortedConstraintNames()

	for _, name := range constraintNames {
		constraint := t.Constraints[name]
		if constraint.Type == ConstraintTypeCheck {
			checkConstraints = append(checkConstraints, constraint)
		}
	}
	return checkConstraints
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

// GenerateSQL for Table
func (t *Table) GenerateSQL() string {
	return t.GenerateSQLWithOptions(true)
}

// GenerateSQLWithOptions for Table with configurable comment inclusion
func (t *Table) GenerateSQLWithOptions(includeComments bool) string {
	if t.Type != TableTypeBase {
		return "" // Skip views here, they're handled separately
	}

	w := NewSQLWriterWithComments(includeComments)

	// Table definition
	w.WriteComment("TABLE", t.Name, t.Schema, "")
	w.WriteString("\n")
	w.WriteString(fmt.Sprintf("CREATE TABLE %s.%s (\n", t.Schema, t.Name))

	// Columns
	columns := t.SortColumnsByPosition()
	checkConstraints := t.GetCheckConstraints()
	hasCheckConstraints := len(checkConstraints) > 0

	for i, column := range columns {
		w.WriteString("    ")
		t.writeColumnDefinition(w, column)
		// Add comma after every column except the last one when there are no CHECK constraints
		if i < len(columns)-1 || hasCheckConstraints {
			w.WriteString(",")
		}
		w.WriteString("\n")
	}

	// Check constraints inline
	for i, constraint := range checkConstraints {
		// CheckClause already contains "CHECK (...)" from pg_get_constraintdef
		w.WriteString(fmt.Sprintf("    CONSTRAINT %s %s", constraint.Name, constraint.CheckClause))
		if i < len(checkConstraints)-1 {
			w.WriteString(",")
		}
		w.WriteString("\n")
	}

	w.WriteString(")")
	
	// Add partition clause if table is partitioned
	if t.IsPartitioned && t.PartitionStrategy != "" && t.PartitionKey != "" {
		w.WriteString(fmt.Sprintf("\nPARTITION BY %s (%s)", t.PartitionStrategy, t.PartitionKey))
	}
	
	w.WriteString(";\n")

	// Generate COMMENT ON TABLE statement if comment exists
	if t.Comment != "" && t.Comment != "<nil>" {
		w.WriteDDLSeparator()
		
		// Escape single quotes in comment
		escapedComment := strings.ReplaceAll(t.Comment, "'", "''")
		commentStmt := fmt.Sprintf("COMMENT ON TABLE %s.%s IS '%s';", t.Schema, t.Name, escapedComment)
		w.WriteStatementWithComment("COMMENT", "TABLE "+t.Name, t.Schema, "", commentStmt)
	}

	// Generate COMMENT ON COLUMN statements for columns with comments
	for _, column := range columns {
		if column.Comment != "" && column.Comment != "<nil>" {
			w.WriteDDLSeparator()
			
			// Escape single quotes in comment
			escapedComment := strings.ReplaceAll(column.Comment, "'", "''")
			commentStmt := fmt.Sprintf("COMMENT ON COLUMN %s.%s.%s IS '%s';", t.Schema, t.Name, column.Name, escapedComment)
			w.WriteStatementWithComment("COMMENT", "COLUMN "+t.Name+"."+column.Name, t.Schema, "", commentStmt)
		}
	}

	return w.String()
}


func (t *Table) writeColumnDefinition(w *SQLWriter, column *Column) {
	w.WriteString(column.Name)
	w.WriteString(" ")

	// Data type - handle array types and precision/scale for appropriate types
	dataType := column.DataType
	
	// Handle USER-DEFINED types and domains: use UDTName instead of base type
	if (dataType == "USER-DEFINED" && column.UDTName != "") || strings.Contains(column.UDTName, ".") {
		dataType = column.UDTName
		// Canonicalize internal type names (e.g., int4 -> integer, int8 -> bigint)
		dataType = canonicalizeTypeName(dataType)
		// If the UDTName doesn't contain a schema qualifier and it's not a built-in type,
		// we should add the schema qualifier. For most cases, this will be the same schema as the table.
		// If the type is in a different schema, it would already be qualified in UDTName.
		if !strings.Contains(dataType, ".") && !isBuiltInType(dataType) {
			// For custom types and extension types, use the table's schema as the type schema
			// This matches pg_dump behavior for types in the same schema
			dataType = t.Schema + "." + dataType
		}
	} else {
		// Canonicalize built-in type names (e.g., int4 -> integer, int8 -> bigint)
		dataType = canonicalizeTypeName(dataType)
	}
	
	// Handle array types: if data_type is "ARRAY", use udt_name with [] suffix
	if column.DataType == "ARRAY" && column.UDTName != "" {
		// Remove the underscore prefix from udt_name for array types
		// PostgreSQL stores array element types with a leading underscore
		elementType := column.UDTName
		if strings.HasPrefix(elementType, "_") {
			elementType = elementType[1:]
		}
		// Canonicalize internal type names for array elements (e.g., int4 -> integer, int8 -> bigint)
		elementType = canonicalizeTypeName(elementType)
		// For custom/extension element types, add schema qualification
		if !strings.Contains(elementType, ".") && !isBuiltInType(elementType) {
			elementType = t.Schema + "." + elementType
		}
		dataType = elementType + "[]"
	} else if column.MaxLength != nil && (dataType == "character varying" || dataType == "varchar") {
		dataType = fmt.Sprintf("character varying(%d)", *column.MaxLength)
	} else if column.MaxLength != nil && dataType == "character" {
		dataType = fmt.Sprintf("character(%d)", *column.MaxLength)
	} else if column.Precision != nil && column.Scale != nil && (dataType == "numeric" || dataType == "decimal") {
		dataType = fmt.Sprintf("%s(%d,%d)", dataType, *column.Precision, *column.Scale)
	} else if column.Precision != nil && (dataType == "numeric" || dataType == "decimal") {
		dataType = fmt.Sprintf("%s(%d)", dataType, *column.Precision)
	}
	// For integer types like "integer", "bigint", "smallint", do not add precision/scale

	w.WriteString(dataType)

	// Identity columns
	if column.IsIdentity {
		if column.IdentityGeneration == "ALWAYS" {
			w.WriteString(" GENERATED ALWAYS AS IDENTITY")
		} else if column.IdentityGeneration == "BY DEFAULT" {
			w.WriteString(" GENERATED BY DEFAULT AS IDENTITY")
		}
	}

	// Default (include all defaults inline)
	if column.DefaultValue != nil && !column.IsIdentity {
		w.WriteString(fmt.Sprintf(" DEFAULT %s", *column.DefaultValue))
	}

	// Not null
	if !column.IsNullable {
		w.WriteString(" NOT NULL")
	}

	// Handle inline constraints (PRIMARY KEY, UNIQUE)
	t.writeInlineConstraints(w, column)
}


// writeInlineConstraints writes inline constraints for a column (PRIMARY KEY, UNIQUE)
func (t *Table) writeInlineConstraints(w *SQLWriter, column *Column) {
	// Look for single-column constraints that can be written inline
	for _, constraint := range t.Constraints {
		if len(constraint.Columns) == 1 && constraint.Columns[0].Name == column.Name {
			switch constraint.Type {
			case ConstraintTypePrimaryKey:
				w.WriteString(" PRIMARY KEY")
			case ConstraintTypeUnique:
				w.WriteString(" UNIQUE")
			}
		}
	}
}

// GetColumnsWithSequenceDefaults returns columns that have defaults referencing sequences
func (t *Table) GetColumnsWithSequenceDefaults() []*Column {
	var columns []*Column
	sortedColumns := t.SortColumnsByPosition()
	for _, column := range sortedColumns {
		if column.DefaultValue != nil && strings.Contains(*column.DefaultValue, "nextval") {
			columns = append(columns, column)
		}
	}
	return columns
}

// GenerateColumnDefaultSQL generates SQL for a single column default
func (c *Column) GenerateColumnDefaultSQL(tableName, schemaName string) string {
	w := NewSQLWriter()
	stmt := fmt.Sprintf("ALTER TABLE ONLY %s.%s ALTER COLUMN %s SET DEFAULT %s;",
		schemaName, tableName, c.Name, *c.DefaultValue)
	w.WriteStatementWithComment("DEFAULT", fmt.Sprintf("%s %s", tableName, c.Name), schemaName, "", stmt)
	return w.String()
}

// GenerateRLSSQL generates SQL for RLS enablement
func (t *Table) GenerateRLSSQL() string {
	if !t.RLSEnabled {
		return ""
	}
	w := NewSQLWriter()
	stmt := fmt.Sprintf("ALTER TABLE %s.%s ENABLE ROW LEVEL SECURITY;", t.Schema, t.Name)
	w.WriteStatementWithComment("ROW SECURITY", t.Name, t.Schema, "", stmt)
	return w.String()
}