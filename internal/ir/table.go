package ir

import (
	"fmt"
	"sort"
	"strings"
)

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
		// CheckClause already contains "CHECK (...)" from pg_get_constraintdef
		w.WriteString(fmt.Sprintf("    CONSTRAINT %s %s", constraint.Name, constraint.CheckClause))
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

	// Data type - handle array types and precision/scale for appropriate types
	dataType := column.DataType
	
	// Handle array types: if data_type is "ARRAY", use udt_name with [] suffix
	if dataType == "ARRAY" && column.UDTName != "" {
		// Remove the underscore prefix from udt_name for array types
		// PostgreSQL stores array element types with a leading underscore
		elementType := column.UDTName
		if strings.HasPrefix(elementType, "_") {
			elementType = elementType[1:]
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

	// Default (only for simple defaults, complex ones are handled separately)
	if column.DefaultValue != nil && !strings.Contains(*column.DefaultValue, "nextval") {
		w.WriteString(fmt.Sprintf(" DEFAULT %s", *column.DefaultValue))
	}

	// Not null
	if !column.IsNullable {
		w.WriteString(" NOT NULL")
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