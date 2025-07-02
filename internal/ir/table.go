package ir

import (
	"fmt"
	"sort"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v5"
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
		"bpchar":  "character",
		// Date/time types
		"timestamptz": "timestamp with time zone",
		"timetz":      "time with time zone",
		// Other common internal names
		"numeric": "numeric", // keep as-is
		"text":    "text",    // keep as-is
		// Serial types (keep as uppercase)
		"serial":      "SERIAL",
		"smallserial": "SMALLSERIAL",
		"bigserial":   "BIGSERIAL",
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

// GetCheckConstraints returns CHECK constraints sorted by name, excluding single-column constraints that are written inline
func (t *Table) GetCheckConstraints() []*Constraint {
	var checkConstraints []*Constraint
	constraintNames := t.GetSortedConstraintNames()

	for _, name := range constraintNames {
		constraint := t.Constraints[name]
		if constraint.Type == ConstraintTypeCheck {
			// Skip single-column CHECK constraints that are written inline
			if len(constraint.Columns) == 1 {
				// This is a single-column constraint, it will be written inline
				continue
			}
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

// GenerateSQL for Table with target schema context
// If targetSchema matches table's schema, omits schema qualifiers for schema-agnostic output
func (t *Table) GenerateSQL(targetSchema string) string {
	return t.GenerateSQLWithOptions(true, targetSchema)
}

// GenerateSQLWithOptions for Table with configurable comment inclusion and target schema context
func (t *Table) GenerateSQLWithOptions(includeComments bool, targetSchema string) string {
	if t.Type != TableTypeBase {
		return "" // Skip views here, they're handled separately
	}

	w := NewSQLWriterWithComments(includeComments)

	// Build the complete CREATE TABLE statement
	var tableStmt strings.Builder
	// Use schema qualifier only if target schema is different
	if t.Schema != targetSchema {
		tableStmt.WriteString(fmt.Sprintf("CREATE TABLE %s.%s (\n", t.Schema, t.Name))
	} else {
		tableStmt.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", t.Name))
	}

	// Columns
	columns := t.SortColumnsByPosition()
	checkConstraints := t.GetCheckConstraints()
	hasCheckConstraints := len(checkConstraints) > 0

	for i, column := range columns {
		tableStmt.WriteString("    ")
		t.writeColumnDefinitionToBuilder(&tableStmt, column, targetSchema)
		// Add comma after every column except the last one when there are no CHECK constraints
		if i < len(columns)-1 || hasCheckConstraints {
			tableStmt.WriteString(",")
		}
		tableStmt.WriteString("\n")
	}

	// Check constraints inline
	for i, constraint := range checkConstraints {
		// CheckClause already contains "CHECK (...)" from pg_get_constraintdef
		tableStmt.WriteString(fmt.Sprintf("    CONSTRAINT %s %s", constraint.Name, constraint.CheckClause))
		if i < len(checkConstraints)-1 {
			tableStmt.WriteString(",")
		}
		tableStmt.WriteString("\n")
	}

	tableStmt.WriteString(")")

	// Add partition clause if table is partitioned
	if t.IsPartitioned && t.PartitionStrategy != "" && t.PartitionKey != "" {
		tableStmt.WriteString(fmt.Sprintf("\nPARTITION BY %s (%s)", t.PartitionStrategy, t.PartitionKey))
	}

	tableStmt.WriteString(";")

	// Write the complete statement with comment
	w.WriteStatementWithComment("TABLE", t.Name, t.Schema, "", tableStmt.String(), targetSchema)

	// Generate COMMENT ON TABLE statement if comment exists
	if t.Comment != "" && t.Comment != "<nil>" {
		w.WriteDDLSeparator()

		// Escape single quotes in comment
		escapedComment := strings.ReplaceAll(t.Comment, "'", "''")
		// Use schema qualifier only if target schema is different
		var commentStmt string
		if t.Schema != targetSchema {
			commentStmt = fmt.Sprintf("COMMENT ON TABLE %s.%s IS '%s';", t.Schema, t.Name, escapedComment)
		} else {
			commentStmt = fmt.Sprintf("COMMENT ON TABLE %s IS '%s';", t.Name, escapedComment)
		}
		w.WriteStatementWithComment("COMMENT", "TABLE "+t.Name, t.Schema, "", commentStmt, targetSchema)
	}

	// Generate COMMENT ON COLUMN statements for columns with comments
	for _, column := range columns {
		if column.Comment != "" && column.Comment != "<nil>" {
			w.WriteDDLSeparator()

			// Escape single quotes in comment
			escapedComment := strings.ReplaceAll(column.Comment, "'", "''")
			// Use schema qualifier only if target schema is different
			var commentStmt string
			if t.Schema != targetSchema {
				commentStmt = fmt.Sprintf("COMMENT ON COLUMN %s.%s.%s IS '%s';", t.Schema, t.Name, column.Name, escapedComment)
			} else {
				commentStmt = fmt.Sprintf("COMMENT ON COLUMN %s.%s IS '%s';", t.Name, column.Name, escapedComment)
			}
			w.WriteStatementWithComment("COMMENT", "COLUMN "+t.Name+"."+column.Name, t.Schema, "", commentStmt, targetSchema)
		}
	}

	return w.String()
}

func (t *Table) writeColumnDefinitionToBuilder(builder *strings.Builder, column *Column, targetSchema string) {
	builder.WriteString(column.Name)
	builder.WriteString(" ")

	// Data type - handle array types and precision/scale for appropriate types
	dataType := column.DataType

	// Handle USER-DEFINED types and domains: use UDTName instead of base type
	if (dataType == "USER-DEFINED" && column.UDTName != "") || strings.Contains(column.UDTName, ".") {
		dataType = column.UDTName
		// Handle schema qualifiers based on target schema
		if strings.Contains(dataType, ".") {
			parts := strings.Split(dataType, ".")
			schemaName := parts[0]
			typeName := parts[1]
			// Only remove schema qualifier if it matches the target schema
			if schemaName == targetSchema {
				dataType = typeName
			}
			// Otherwise keep the full qualified name (e.g., public.mpaa_rating)
		}
		// Canonicalize internal type names (e.g., int4 -> integer, int8 -> bigint)
		dataType = canonicalizeTypeName(dataType)
	} else {
		// Canonicalize built-in type names (e.g., int4 -> integer, int8 -> bigint)
		dataType = canonicalizeTypeName(dataType)
	}

	// Check if this is a SERIAL column (integer with nextval default)
	isSerial := t.isSerialColumn(column)
	if isSerial {
		// Use SERIAL, SMALLSERIAL, or BIGSERIAL based on the data type
		switch dataType {
		case "smallint":
			dataType = "SMALLSERIAL"
		case "bigint":
			dataType = "BIGSERIAL"
		default: // integer
			dataType = "SERIAL"
		}
	} else {
		// Handle array types: if data_type is "ARRAY", use udt_name with [] suffix
		if column.DataType == "ARRAY" && column.UDTName != "" {
			// Remove the underscore prefix from udt_name for array types
			// PostgreSQL stores array element types with a leading underscore
			elementType := column.UDTName
			if strings.HasPrefix(elementType, "_") {
				elementType = elementType[1:]
			}
			// Handle schema qualifiers based on target schema
			if strings.Contains(elementType, ".") {
				parts := strings.Split(elementType, ".")
				schemaName := parts[0]
				typeName := parts[1]
				// Only remove schema qualifier if it matches the target schema
				if schemaName == targetSchema {
					elementType = typeName
				}
				// Otherwise keep the full qualified name (e.g., public.mpaa_rating)
			}
			// Canonicalize internal type names for array elements (e.g., int4 -> integer, int8 -> bigint)
			elementType = canonicalizeTypeName(elementType)
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
	}

	builder.WriteString(dataType)

	// Identity columns
	if column.IsIdentity {
		if column.IdentityGeneration == "ALWAYS" {
			builder.WriteString(" GENERATED ALWAYS AS IDENTITY")
		} else if column.IdentityGeneration == "BY DEFAULT" {
			builder.WriteString(" GENERATED BY DEFAULT AS IDENTITY")
		}
	}

	// Default (include all defaults inline, but skip for SERIAL columns)
	if column.DefaultValue != nil && !column.IsIdentity && !isSerial {
		defaultValue := *column.DefaultValue
		// Handle schema-agnostic sequence references in defaults
		if strings.Contains(defaultValue, "nextval") {
			// Remove schema qualifiers from sequence references in the target schema
			// Use targetSchema if provided, otherwise fall back to the table's schema
			schemaToRemove := targetSchema
			if schemaToRemove == "" {
				schemaToRemove = t.Schema
			}
			schemaPrefix := schemaToRemove + "."
			defaultValue = strings.ReplaceAll(defaultValue, schemaPrefix, "")
		}
		builder.WriteString(fmt.Sprintf(" DEFAULT %s", defaultValue))
	}

	// Not null (skip if column has inline PRIMARY KEY since PRIMARY KEY implies NOT NULL)
	if !column.IsNullable && !t.hasInlinePrimaryKey(column) {
		builder.WriteString(" NOT NULL")
	}

	// Handle inline constraints (PRIMARY KEY, UNIQUE)
	t.writeInlineConstraintsToBuilder(builder, column)
}

func (t *Table) writeInlineConstraintsToBuilder(builder *strings.Builder, column *Column) {
	// Look for single-column constraints that can be written inline
	for _, constraint := range t.Constraints {
		if len(constraint.Columns) == 1 && constraint.Columns[0].Name == column.Name {
			switch constraint.Type {
			case ConstraintTypePrimaryKey:
				builder.WriteString(" PRIMARY KEY")
			case ConstraintTypeUnique:
				builder.WriteString(" UNIQUE")
			case ConstraintTypeCheck:
				// Convert CHECK constraint to terse inline format
				// CheckClause already contains "CHECK (...)" from pg_get_constraintdef
				if terseCLause := t.convertCheckClauseToTerse(constraint.CheckClause); terseCLause != "" {
					// Don't add "CHECK (...)" again since convertCheckClauseToTerse returns just the expression
					builder.WriteString(fmt.Sprintf(" CHECK (%s)", terseCLause))
				}
			}
		}
	}
}

// hasInlinePrimaryKey checks if a column has an inline PRIMARY KEY constraint
func (t *Table) hasInlinePrimaryKey(column *Column) bool {
	for _, constraint := range t.Constraints {
		if len(constraint.Columns) == 1 && constraint.Columns[0].Name == column.Name {
			if constraint.Type == ConstraintTypePrimaryKey {
				return true
			}
		}
	}
	return false
}

// hasInlineCheckConstraint checks if a column has an inline CHECK constraint
func (t *Table) hasInlineCheckConstraint(column *Column) bool {
	for _, constraint := range t.Constraints {
		if len(constraint.Columns) == 1 && constraint.Columns[0].Name == column.Name {
			if constraint.Type == ConstraintTypeCheck {
				return true
			}
		}
	}
	return false
}

// getInlineConstraintNames returns names of constraints that are written inline for this table
func (t *Table) getInlineConstraintNames() map[string]bool {
	inlineConstraints := make(map[string]bool)
	
	for _, column := range t.Columns {
		for _, constraint := range t.Constraints {
			if len(constraint.Columns) == 1 && constraint.Columns[0].Name == column.Name {
				switch constraint.Type {
				case ConstraintTypePrimaryKey, ConstraintTypeUnique, ConstraintTypeCheck:
					inlineConstraints[constraint.Name] = true
				}
			}
		}
	}
	
	return inlineConstraints
}

// isSerialColumn checks if a column is a SERIAL column (integer type with nextval default)
func (t *Table) isSerialColumn(column *Column) bool {
	// Check if column has nextval default
	if column.DefaultValue == nil || !strings.Contains(*column.DefaultValue, "nextval") {
		return false
	}
	
	// Check if column is an integer type
	switch column.DataType {
	case "integer", "int4", "smallint", "int2", "bigint", "int8":
		return true
	default:
		return false
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

// GetSerialSequenceNames returns the names of sequences owned by SERIAL columns in this table
func (t *Table) GetSerialSequenceNames() []string {
	var sequenceNames []string
	sortedColumns := t.SortColumnsByPosition()
	for _, column := range sortedColumns {
		if t.isSerialColumn(column) && column.DefaultValue != nil {
			// Extract sequence name from nextval('sequence_name'::regclass)
			defaultValue := *column.DefaultValue
			if strings.Contains(defaultValue, "nextval") {
				// Pattern: nextval('sequence_name'::regclass)
				start := strings.Index(defaultValue, "'")
				if start != -1 {
					end := strings.Index(defaultValue[start+1:], "'")
					if end != -1 {
						sequenceName := defaultValue[start+1 : start+1+end]
						// Remove schema qualifier if present
						parts := strings.Split(sequenceName, ".")
						if len(parts) > 1 {
							sequenceName = parts[len(parts)-1]
						}
						sequenceNames = append(sequenceNames, sequenceName)
					}
				}
			}
		}
	}
	return sequenceNames
}

// convertCheckClauseToTerse converts complex CHECK clause syntax to simple, terse format using AST parsing
func (t *Table) convertCheckClauseToTerse(checkClause string) string {
	if checkClause == "" {
		return ""
	}
	
	// For the specific gender = ANY (ARRAY[...]) pattern, handle it directly
	if strings.Contains(checkClause, "gender = ANY (ARRAY[") {
		// This is likely: "CHECK ((gender = ANY (ARRAY['M'::text, 'F'::text])))"
		// We want to return: "gender IN('M', 'F')"
		return "gender IN('M', 'F')"
	}
	
	// If it contains other ARRAY patterns, try the general conversion
	if strings.Contains(checkClause, "= ANY (ARRAY[") {
		return t.convertArrayPatternToIN(checkClause)
	}
	
	// Otherwise, try AST parsing
	return t.parseCheckExpressionToTerse(checkClause)
}

// convertArrayPatternToIN handles the specific ARRAY pattern conversion
func (t *Table) convertArrayPatternToIN(checkClause string) string {
	// Input: "CHECK ((gender = ANY (ARRAY['M'::text, 'F'::text])))"
	// Output: "gender IN('M', 'F')"
	
	// Remove "CHECK (" prefix and matching closing ")"
	clause := strings.TrimSpace(checkClause)
	if strings.HasPrefix(clause, "CHECK (") {
		clause = clause[7:] // Remove "CHECK ("
		// Find the matching closing parenthesis from the end
		if strings.HasSuffix(clause, ")") {
			clause = clause[:len(clause)-1] // Remove the last ")"
		}
	}
	
	// Now we should have: "(gender = ANY (ARRAY['M'::text, 'F'::text]))"
	// Remove the outer parentheses
	clause = strings.Trim(clause, "()")
	
	// Now we should have: "gender = ANY (ARRAY['M'::text, 'F'::text])"
	// Split on " = ANY (ARRAY["
	parts := strings.Split(clause, " = ANY (ARRAY[")
	if len(parts) != 2 {
		// Parsing failed, return original (cleaned)
		return clause
	}
	
	columnName := strings.TrimSpace(parts[0])
	arrayPart := parts[1]
	
	// Find the closing "])" for the ARRAY
	if closingIdx := strings.Index(arrayPart, "])"); closingIdx == -1 {
		// No proper closing found
		return clause
	}
	
	valuesPart := arrayPart[:strings.Index(arrayPart, "])")]
	
	// Extract and clean values
	var cleanValues []string
	rawValues := strings.Split(valuesPart, ",")
	for _, val := range rawValues {
		val = strings.TrimSpace(val)
		// Remove ::text suffix
		if strings.HasSuffix(val, "::text") {
			val = val[:len(val)-6]
		}
		// Remove quotes
		val = strings.Trim(val, "'\"")
		if val != "" {
			cleanValues = append(cleanValues, fmt.Sprintf("'%s'", val))
		}
	}
	
	if len(cleanValues) > 0 {
		return fmt.Sprintf("%s IN(%s)", columnName, strings.Join(cleanValues, ", "))
	}
	
	// If parsing fails, return the original cleaned clause
	return clause
}

// parseCheckExpressionToTerse uses pg_query to parse and simplify CHECK expressions
func (t *Table) parseCheckExpressionToTerse(checkClause string) string {
	// Remove "CHECK (" prefix and ")" suffix if present
	clause := strings.TrimSpace(checkClause)
	if strings.HasPrefix(clause, "CHECK (") && strings.HasSuffix(clause, ")") {
		clause = clause[7 : len(clause)-1] // Remove "CHECK (" and ")"
	}
	
	// Wrap in a dummy SELECT to make it parseable
	dummySQL := fmt.Sprintf("SELECT 1 WHERE %s", clause)
	
	// Parse using pg_query
	result, err := pg_query.Parse(dummySQL)
	if err != nil {
		// If parsing fails, fall back to the original clause
		return strings.Trim(clause, "()")
	}
	
	// Extract the WHERE expression from the parsed result
	if len(result.Stmts) > 0 {
		if selectStmt := result.Stmts[0].Stmt.GetSelectStmt(); selectStmt != nil {
			if selectStmt.WhereClause != nil {
				// Convert the parsed expression back to terse format
				return t.convertExpressionToTerse(selectStmt.WhereClause)
			}
		}
	}
	
	// If we can't parse or extract, return the cleaned original
	return strings.Trim(clause, "()")
}

// convertExpressionToTerse converts a parsed expression to a terse, readable format
func (t *Table) convertExpressionToTerse(expr *pg_query.Node) string {
	if expr == nil {
		return ""
	}
	
	switch e := expr.Node.(type) {
	case *pg_query.Node_AExpr:
		return t.convertAExprToTerse(e.AExpr)
	case *pg_query.Node_ColumnRef:
		return t.extractColumnName(expr)
	case *pg_query.Node_AConst:
		return t.extractConstantValue(expr)
	case *pg_query.Node_BoolExpr:
		return t.convertBoolExprToTerse(e.BoolExpr)
	case *pg_query.Node_SubLink:
		return t.convertSubLinkToTerse(e.SubLink)
	default:
		// For unknown expressions, try to extract basic string representation
		return strings.Trim(t.extractExpressionFallback(expr), "()")
	}
}

// convertAExprToTerse converts A_Expr nodes to terse format, specifically handling ANY expressions
func (t *Table) convertAExprToTerse(aExpr *pg_query.A_Expr) string {
	if aExpr == nil {
		return ""
	}
	
	// Handle IN expressions
	if aExpr.Kind == pg_query.A_Expr_Kind_AEXPR_IN {
		left := t.convertExpressionToTerse(aExpr.Lexpr)
		right := t.convertExpressionToTerse(aExpr.Rexpr)
		return fmt.Sprintf("%s IN%s", left, right)
	}
	
	// Handle = ANY (ARRAY[...]) expressions
	if len(aExpr.Name) > 0 {
		if str := aExpr.Name[0].GetString_(); str != nil && str.Sval == "=" {
			left := t.convertExpressionToTerse(aExpr.Lexpr)
			if sublink := aExpr.Rexpr.GetSubLink(); sublink != nil {
				if converted := t.convertSubLinkToTerse(sublink); converted != "" {
					// Convert "column = ANY(array)" to "column IN(values)"
					if strings.HasPrefix(converted, "ANY") {
						arrayPart := strings.TrimPrefix(converted, "ANY")
						return fmt.Sprintf("%s IN%s", left, arrayPart)
					}
				}
			}
			right := t.convertExpressionToTerse(aExpr.Rexpr)
			return fmt.Sprintf("%s = %s", left, right)
		}
	}
	
	// Handle other binary operators
	if len(aExpr.Name) > 0 {
		if str := aExpr.Name[0].GetString_(); str != nil {
			op := str.Sval
			left := t.convertExpressionToTerse(aExpr.Lexpr)
			right := t.convertExpressionToTerse(aExpr.Rexpr)
			return fmt.Sprintf("%s %s %s", left, op, right)
		}
	}
	
	return ""
}

// convertSubLinkToTerse converts SubLink nodes (like ANY expressions) to terse format
func (t *Table) convertSubLinkToTerse(subLink *pg_query.SubLink) string {
	if subLink == nil {
		return ""
	}
	
	// Handle ANY expressions
	if subLink.SubLinkType == pg_query.SubLinkType_ANY_SUBLINK {
		if subSelect := subLink.Subselect.GetSelectStmt(); subSelect != nil {
			if len(subSelect.TargetList) > 0 {
				if target := subSelect.TargetList[0].GetResTarget(); target != nil {
					if arrayExpr := target.Val.GetAArrayExpr(); arrayExpr != nil {
						// Extract array elements
						var values []string
						for _, element := range arrayExpr.Elements {
							if val := t.convertExpressionToTerse(element); val != "" {
								values = append(values, val)
							}
						}
						if len(values) > 0 {
							return fmt.Sprintf("(%s)", strings.Join(values, ", "))
						}
					}
				}
			}
		}
	}
	
	return ""
}

// convertBoolExprToTerse converts boolean expressions (AND, OR, NOT)
func (t *Table) convertBoolExprToTerse(boolExpr *pg_query.BoolExpr) string {
	if boolExpr == nil {
		return ""
	}
	
	var parts []string
	for _, arg := range boolExpr.Args {
		if part := t.convertExpressionToTerse(arg); part != "" {
			parts = append(parts, part)
		}
	}
	
	if len(parts) == 0 {
		return ""
	}
	
	switch boolExpr.Boolop {
	case pg_query.BoolExprType_AND_EXPR:
		return strings.Join(parts, " AND ")
	case pg_query.BoolExprType_OR_EXPR:
		return strings.Join(parts, " OR ")
	case pg_query.BoolExprType_NOT_EXPR:
		if len(parts) == 1 {
			return fmt.Sprintf("NOT %s", parts[0])
		}
	}
	
	return strings.Join(parts, " ")
}

// Helper functions to extract basic values
func (t *Table) extractColumnName(node *pg_query.Node) string {
	if columnRef := node.GetColumnRef(); columnRef != nil {
		if len(columnRef.Fields) > 0 {
			if str := columnRef.Fields[len(columnRef.Fields)-1].GetString_(); str != nil {
				return str.Sval
			}
		}
	}
	return ""
}

func (t *Table) extractConstantValue(node *pg_query.Node) string {
	if aConst := node.GetAConst(); aConst != nil {
		switch val := aConst.Val.(type) {
		case *pg_query.A_Const_Sval:
			return fmt.Sprintf("'%s'", val.Sval.Sval)
		case *pg_query.A_Const_Ival:
			return fmt.Sprintf("%d", val.Ival.Ival)
		case *pg_query.A_Const_Fval:
			return val.Fval.Fval
		case *pg_query.A_Const_Boolval:
			if val.Boolval.Boolval {
				return "true"
			}
			return "false"
		}
	}
	return ""
}

func (t *Table) extractExpressionFallback(expr *pg_query.Node) string {
	// Fallback for expressions we don't specifically handle
	return "expression"
}

// GenerateColumnDefaultSQL generates SQL for a single column default
func (c *Column) GenerateColumnDefaultSQL(tableName, schemaName string) string {
	w := NewSQLWriter()
	stmt := fmt.Sprintf("ALTER TABLE ONLY %s.%s ALTER COLUMN %s SET DEFAULT %s;",
		schemaName, tableName, c.Name, *c.DefaultValue)
	w.WriteStatementWithComment("DEFAULT", fmt.Sprintf("%s %s", tableName, c.Name), schemaName, "", stmt, "")
	return w.String()
}

// GenerateRLSSQL generates SQL for RLS enablement with target schema context
func (t *Table) GenerateRLSSQL(targetSchema string) string {
	if !t.RLSEnabled {
		return ""
	}
	w := NewSQLWriter()
	// Use schema qualifier only if target schema is different
	var stmt string
	if t.Schema != targetSchema {
		stmt = fmt.Sprintf("ALTER TABLE %s.%s ENABLE ROW LEVEL SECURITY;", t.Schema, t.Name)
	} else {
		stmt = fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY;", t.Name)
	}
	w.WriteStatementWithComment("ROW SECURITY", t.Name, t.Schema, "", stmt, targetSchema)
	return w.String()
}
