package schema

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Schema represents the complete database schema intermediate representation
type Schema struct {
	Metadata   Metadata              `json:"metadata"`
	Schemas    map[string]*DBSchema  `json:"schemas"`    // schema_name -> DBSchema
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
	Schema       string                 `json:"schema"`
	Name         string                 `json:"name"`
	Type         TableType              `json:"type"` // BASE_TABLE, VIEW, etc.
	Columns      []*Column              `json:"columns"`
	Constraints  map[string]*Constraint `json:"constraints"` // constraint_name -> Constraint
	Indexes      map[string]*Index      `json:"indexes"`     // index_name -> Index
	Triggers     map[string]*Trigger    `json:"triggers"`    // trigger_name -> Trigger
	RLSEnabled   bool                   `json:"rls_enabled"`
	Policies     map[string]*RLSPolicy  `json:"policies"` // policy_name -> RLSPolicy
	Dependencies []TableDependency      `json:"dependencies"`
	Comment      string                 `json:"comment,omitempty"`
}

// Column represents a table column
type Column struct {
	Name         string  `json:"name"`
	Position     int     `json:"position"` // ordinal_position
	DataType     string  `json:"data_type"`
	UDTName      string  `json:"udt_name,omitempty"`
	IsNullable   bool    `json:"is_nullable"`
	DefaultValue *string `json:"default_value,omitempty"`
	MaxLength    *int    `json:"max_length,omitempty"`
	Precision    *int    `json:"precision,omitempty"`
	Scale        *int    `json:"scale,omitempty"`
	Comment      string  `json:"comment,omitempty"`
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

// Index represents a database index
type Index struct {
	Schema     string         `json:"schema"`
	Table      string         `json:"table"`
	Name       string         `json:"name"`
	Type       IndexType      `json:"type"`
	Method     string         `json:"method"` // btree, hash, gin, gist, etc.
	Columns    []*IndexColumn `json:"columns"`
	IsUnique   bool           `json:"is_unique"`
	IsPrimary  bool           `json:"is_primary"`
	IsPartial  bool           `json:"is_partial"`
	Where      string         `json:"where,omitempty"` // partial index condition
	Definition string         `json:"definition"`      // full CREATE INDEX statement
	Comment    string         `json:"comment,omitempty"`
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
	Schema     string       `json:"schema"`
	Name       string       `json:"name"`
	Definition string       `json:"definition"`
	ReturnType string       `json:"return_type"`
	Language   string       `json:"language"`
	Parameters []*Parameter `json:"parameters,omitempty"`
	Comment    string       `json:"comment,omitempty"`
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
	TableTypeBase TableType = "BASE_TABLE"
	TableTypeView TableType = "VIEW"
	TableTypeTemp TableType = "TEMPORARY"
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

// SortConstraintColumnsByPosition sorts constraint columns by their position
func (c *Constraint) SortConstraintColumnsByPosition() []*ConstraintColumn {
	columns := make([]*ConstraintColumn, len(c.Columns))
	copy(columns, c.Columns)
	sort.Slice(columns, func(i, j int) bool {
		return columns[i].Position < columns[j].Position
	})
	return columns
}

// SQLGenerator interface for visitor pattern
type SQLGenerator interface {
	GenerateSQL() string
}


// SQLGenerator implementations for each database resource type

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

// GenerateSQL for Function
func (f *Function) GenerateSQL() string {
	if f.Definition == "<nil>" || f.Definition == "" {
		return ""
	}
	w := NewSQLWriter()
	stmt := fmt.Sprintf("CREATE FUNCTION %s.%s() RETURNS %s\n    LANGUAGE %s\n    AS $$%s$$;",
		f.Schema, f.Name, f.ReturnType, strings.ToLower(f.Language), f.Definition)
	w.WriteStatementWithComment("FUNCTION", fmt.Sprintf("%s()", f.Name), f.Schema, "", stmt)
	return w.String()
}

// GenerateSQL for Sequence (CREATE SEQUENCE only)
func (s *Sequence) GenerateSQL() string {
	w := NewSQLWriter()

	// Build sequence statement
	var stmt strings.Builder
	stmt.WriteString(fmt.Sprintf("CREATE SEQUENCE %s.%s\n", s.Schema, s.Name))
	if s.DataType != "" && s.DataType != "bigint" {
		stmt.WriteString(fmt.Sprintf("    AS %s\n", s.DataType))
	}
	stmt.WriteString(fmt.Sprintf("    START WITH %d\n", s.StartValue))
	stmt.WriteString(fmt.Sprintf("    INCREMENT BY %d\n", s.Increment))

	if s.MinValue != nil {
		stmt.WriteString(fmt.Sprintf("    MINVALUE %d\n", *s.MinValue))
	} else {
		stmt.WriteString("    NO MINVALUE\n")
	}

	if s.MaxValue != nil {
		stmt.WriteString(fmt.Sprintf("    MAXVALUE %d\n", *s.MaxValue))
	} else {
		stmt.WriteString("    NO MAXVALUE\n")
	}

	stmt.WriteString("    CACHE 1")
	if s.CycleOption {
		stmt.WriteString("\n    CYCLE")
	}
	stmt.WriteString(";")

	w.WriteStatementWithComment("SEQUENCE", s.Name, s.Schema, "", stmt.String())
	return w.String()
}

// GenerateOwnershipSQL generates ALTER SEQUENCE OWNED BY statement
func (s *Sequence) GenerateOwnershipSQL() string {
	if s.OwnedByTable == "" || s.OwnedByColumn == "" {
		return ""
	}
	w := NewSQLWriter()
	ownedStmt := fmt.Sprintf("ALTER SEQUENCE %s.%s OWNED BY %s.%s.%s;",
		s.Schema, s.Name, s.Schema, s.OwnedByTable, s.OwnedByColumn)
	w.WriteStatementWithComment("SEQUENCE OWNED BY", s.Name, s.Schema, "", ownedStmt)
	return w.String()
}

// GenerateSQL for Table
func (t *Table) GenerateSQL() string {
	if t.Type != TableTypeBase {
		return "" // Skip views here, they're handled separately
	}

	w := NewSQLWriter()

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
		w.WriteString(fmt.Sprintf("    CONSTRAINT %s CHECK (%s)", constraint.Name, constraint.CheckClause))
		if i < len(checkConstraints)-1 {
			w.WriteString(",")
		}
		w.WriteString("\n")
	}

	w.WriteString(");\n")

	return w.String()
}

func (t *Table) writeColumnDefinition(w *SQLWriter, column *Column) {
	w.WriteString(column.Name)
	w.WriteString(" ")

	// Data type - only add precision/scale for appropriate types
	dataType := column.DataType
	if column.MaxLength != nil && (dataType == "character varying" || dataType == "varchar") {
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

	// Not null
	if !column.IsNullable {
		w.WriteString(" NOT NULL")
	}

	// Default (only for simple defaults, complex ones are handled separately)
	if column.DefaultValue != nil && !strings.Contains(*column.DefaultValue, "nextval") {
		w.WriteString(fmt.Sprintf(" DEFAULT %s", *column.DefaultValue))
	}
}

// GenerateSQL for View
func (v *View) GenerateSQL() string {
	w := NewSQLWriter()
	// For now, use the definition as-is. Schema qualification will be handled at a higher level
	stmt := fmt.Sprintf("CREATE VIEW %s.%s AS\n%s;", v.Schema, v.Name, v.Definition)
	w.WriteStatementWithComment("VIEW", v.Name, v.Schema, "", stmt)
	return w.String()
}

// GenerateSQLWithSchemaContext generates SQL for a view with schema qualification
func (v *View) GenerateSQLWithSchemaContext(schemaIR *Schema) string {
	w := NewSQLWriter()
	stmt := fmt.Sprintf("CREATE VIEW %s.%s AS\n%s;", v.Schema, v.Name, v.Definition)
	w.WriteStatementWithComment("VIEW", v.Name, v.Schema, "", stmt)
	return w.String()
}

// GenerateSQL for Index
func (i *Index) GenerateSQL() string {
	w := NewSQLWriter()
	stmt := fmt.Sprintf("%s;", i.Definition)
	w.WriteStatementWithComment("INDEX", i.Name, i.Schema, "", stmt)
	return w.String()
}

// GenerateSQL for Trigger
func (tr *Trigger) GenerateSQL() string {
	w := NewSQLWriter()

	// Build event list
	var events []string
	for _, event := range tr.Events {
		events = append(events, string(event))
	}
	eventList := strings.Join(events, " OR ")

	stmt := fmt.Sprintf("CREATE TRIGGER %s %s %s ON %s.%s FOR EACH %s EXECUTE FUNCTION %s.%s();",
		tr.Name, tr.Timing, eventList, tr.Schema, tr.Table, tr.Level, tr.Schema, tr.Function)
	w.WriteStatementWithComment("TRIGGER", fmt.Sprintf("%s %s", tr.Table, tr.Name), tr.Schema, "", stmt)
	return w.String()
}

// GenerateSQL for Constraint
func (c *Constraint) GenerateSQL() string {
	w := NewSQLWriter()
	var stmt string

	switch c.Type {
	case ConstraintTypePrimaryKey, ConstraintTypeUnique:
		// Build constraint statement
		var constraintTypeStr string
		switch c.Type {
		case ConstraintTypePrimaryKey:
			constraintTypeStr = "PRIMARY KEY"
		case ConstraintTypeUnique:
			constraintTypeStr = "UNIQUE"
		}

		// Sort columns by position
		columns := c.SortConstraintColumnsByPosition()
		var columnNames []string
		for _, col := range columns {
			columnNames = append(columnNames, col.Name)
		}
		columnList := strings.Join(columnNames, ", ")

		stmt = fmt.Sprintf("ALTER TABLE ONLY %s.%s\n    ADD CONSTRAINT %s %s (%s);",
			c.Schema, c.Table, c.Name, constraintTypeStr, columnList)

	case ConstraintTypeCheck:
		// Handle CHECK constraints
		stmt = fmt.Sprintf("ALTER TABLE ONLY %s.%s\n    ADD CONSTRAINT %s CHECK (%s);",
			c.Schema, c.Table, c.Name, c.CheckClause)

	case ConstraintTypeForeignKey:
		// Sort columns by position
		columns := c.SortConstraintColumnsByPosition()
		var columnNames []string
		for _, col := range columns {
			columnNames = append(columnNames, col.Name)
		}
		columnList := strings.Join(columnNames, ", ")

		// Sort referenced columns by position
		var refColumnNames []string
		if len(c.ReferencedColumns) > 0 {
			refColumns := make([]*ConstraintColumn, len(c.ReferencedColumns))
			copy(refColumns, c.ReferencedColumns)
			sort.Slice(refColumns, func(i, j int) bool {
				return refColumns[i].Position < refColumns[j].Position
			})
			for _, col := range refColumns {
				refColumnNames = append(refColumnNames, col.Name)
			}
		}
		refColumnList := strings.Join(refColumnNames, ", ")

		// Build referential actions
		var actions []string
		if c.DeleteRule != "" && c.DeleteRule != "NO ACTION" {
			actions = append(actions, fmt.Sprintf("ON DELETE %s", c.DeleteRule))
		}
		if c.UpdateRule != "" && c.UpdateRule != "NO ACTION" {
			actions = append(actions, fmt.Sprintf("ON UPDATE %s", c.UpdateRule))
		}

		actionStr := ""
		if len(actions) > 0 {
			actionStr = " " + strings.Join(actions, " ")
		}

		stmt = fmt.Sprintf("ALTER TABLE ONLY %s.%s\n    ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s.%s(%s)%s;",
			c.Schema, c.Table, c.Name, columnList, c.ReferencedSchema, c.ReferencedTable, refColumnList, actionStr)

	default:
		return "" // Unsupported constraint type
	}

	constraintTypeStr := "CONSTRAINT"
	if c.Type == ConstraintTypeForeignKey {
		constraintTypeStr = "FK CONSTRAINT"
	}

	w.WriteStatementWithComment(constraintTypeStr, fmt.Sprintf("%s %s", c.Table, c.Name), c.Schema, "", stmt)
	return w.String()
}

// GenerateSQL for RLSPolicy
func (p *RLSPolicy) GenerateSQL() string {
	w := NewSQLWriter()
	policyStmt := fmt.Sprintf("CREATE POLICY %s ON %s.%s", p.Name, p.Schema, p.Table)

	// Add command type if specified
	if p.Command != PolicyCommandAll {
		policyStmt += fmt.Sprintf(" FOR %s", p.Command)
	}

	// Add USING clause if present
	if p.Using != "" {
		policyStmt += fmt.Sprintf(" USING (%s)", p.Using)
	}

	// Add WITH CHECK clause if present
	if p.WithCheck != "" {
		policyStmt += fmt.Sprintf(" WITH CHECK (%s)", p.WithCheck)
	}

	policyStmt += ";"
	w.WriteStatementWithComment("POLICY", fmt.Sprintf("%s %s", p.Table, p.Name), p.Schema, "", policyStmt)
	return w.String()
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
