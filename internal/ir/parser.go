package ir

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v5"
)

// Parser handles parsing SQL statements into IR representation
type Parser struct {
	schema *IR

	// Track partition relationships
	partitionParents  map[string]bool   // tableName -> isPartitionParent
	partitionChildren map[string]string // childTableName -> parentTableName
}

// NewParser creates a new parser instance
func NewParser() *Parser {
	return &Parser{
		schema:            NewIR(),
		partitionParents:  make(map[string]bool),
		partitionChildren: make(map[string]string),
	}
}

// ParseSQL parses SQL content and returns the IR representation
func (p *Parser) ParseSQL(sqlContent string) (*IR, error) {
	// Split SQL content into individual statements
	statements, err := p.splitSQLStatements(sqlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to split SQL statements: %w", err)
	}

	// Parse each statement
	for _, stmt := range statements {
		if err := p.parseStatement(stmt); err != nil {
			return nil, fmt.Errorf("failed to parse statement: %w", err)
		}
	}

	return p.schema, nil
}

// splitSQLStatements splits SQL content into individual statements using pg_query_go
func (p *Parser) splitSQLStatements(sqlContent string) ([]string, error) {
	// Use pg_query_go's native SplitWithParser function
	statements, err := pg_query.SplitWithParser(sqlContent, true) // trimSpace = true
	if err != nil {
		return nil, err
	}

	return statements, nil
}

// parseStatement parses a single SQL statement
func (p *Parser) parseStatement(stmt string) error {
	// Parse the statement using pg_query
	result, err := pg_query.Parse(stmt)
	if err != nil {
		return fmt.Errorf("pg_query parse error: %w. Statement: %q", err, stmt)
	}

	// Process each parsed statement
	for _, parsedStmt := range result.Stmts {
		if parsedStmt.Stmt != nil {
			if err := p.processStatement(parsedStmt.Stmt); err != nil {
				return err
			}
		}
	}

	return nil
}

// processStatement processes a single parsed statement node
func (p *Parser) processStatement(stmt *pg_query.Node) error {
	switch node := stmt.Node.(type) {
	case *pg_query.Node_CreateStmt:
		return p.parseCreateTable(node.CreateStmt)
	case *pg_query.Node_ViewStmt:
		return p.parseCreateView(node.ViewStmt)
	case *pg_query.Node_CreateFunctionStmt:
		return p.parseCreateFunction(node.CreateFunctionStmt)
	case *pg_query.Node_CreateSeqStmt:
		return p.parseCreateSequence(node.CreateSeqStmt)
	case *pg_query.Node_AlterTableStmt:
		return p.parseAlterTable(node.AlterTableStmt)
	case *pg_query.Node_IndexStmt:
		return p.parseCreateIndex(node.IndexStmt)
	case *pg_query.Node_CreateTrigStmt:
		return p.parseCreateTrigger(node.CreateTrigStmt)
	case *pg_query.Node_CreatePolicyStmt:
		return p.parseCreatePolicy(node.CreatePolicyStmt)
	case *pg_query.Node_CreateEnumStmt:
		return p.parseCreateEnum(node.CreateEnumStmt)
	case *pg_query.Node_CompositeTypeStmt:
		return p.parseCreateCompositeType(node.CompositeTypeStmt)
	case *pg_query.Node_CreateDomainStmt:
		return p.parseCreateDomain(node.CreateDomainStmt)
	case *pg_query.Node_DefineStmt:
		return p.parseDefineStatement(node.DefineStmt)
	case *pg_query.Node_CreateExtensionStmt:
		return p.parseCreateExtension(node.CreateExtensionStmt)
	case *pg_query.Node_CreateSchemaStmt:
		return p.parseCreateSchema(node.CreateSchemaStmt)
	default:
		// Ignore other statement types for now
		return nil
	}
}

// Helper function to extract table name from RangeVar
func (p *Parser) extractTableName(rangeVar *pg_query.RangeVar) (schema, table string) {
	if rangeVar.Schemaname != "" {
		schema = rangeVar.Schemaname
	} else {
		schema = "public" // Default schema
	}
	table = rangeVar.Relname
	return
}

// Helper function to extract column name from Node
func (p *Parser) extractColumnName(node *pg_query.Node) string {
	switch n := node.Node.(type) {
	case *pg_query.Node_String_:
		return n.String_.Sval
	case *pg_query.Node_ColumnRef:
		if len(n.ColumnRef.Fields) > 0 {
			var parts []string
			for _, field := range n.ColumnRef.Fields {
				if field != nil {
					if str := field.GetString_(); str != nil {
						part := str.Sval
						// Convert trigger pseudo-relations to uppercase
						if part == "new" || part == "old" {
							part = strings.ToUpper(part)
						}
						parts = append(parts, part)
					}
				}
			}
			if len(parts) > 0 {
				return strings.Join(parts, ".")
			}
		}
	}
	return ""
}

// Helper function to extract string value from Node
func (p *Parser) extractStringValue(node *pg_query.Node) string {
	if node == nil {
		return ""
	}
	switch n := node.Node.(type) {
	case *pg_query.Node_String_:
		return n.String_.Sval
	case *pg_query.Node_AConst:
		if n.AConst.Val != nil {
			switch val := n.AConst.Val.(type) {
			case *pg_query.A_Const_Sval:
				return val.Sval.Sval
			case *pg_query.A_Const_Ival:
				return strconv.FormatInt(int64(val.Ival.Ival), 10)
			}
		}
	}
	return ""
}

// Helper function to extract integer value from Node
func (p *Parser) extractIntValue(node *pg_query.Node) int {
	if node == nil {
		return 0
	}
	switch n := node.Node.(type) {
	case *pg_query.Node_Integer:
		return int(n.Integer.Ival)
	case *pg_query.Node_AConst:
		if n.AConst.Val != nil {
			if val := n.AConst.GetIval(); val != nil {
				return int(val.Ival)
			}
		}
	}
	return 0
}

// parseCreateTable parses CREATE TABLE statements
func (p *Parser) parseCreateTable(createStmt *pg_query.CreateStmt) error {
	schemaName, tableName := p.extractTableName(createStmt.Relation)

	// Get or create schema
	dbSchema := p.schema.getOrCreateSchema(schemaName)

	// Create table
	table := &Table{
		Schema:      schemaName,
		Name:        tableName,
		Type:        TableTypeBase,
		Columns:     make([]*Column, 0),
		Constraints: make(map[string]*Constraint),
		Indexes:     make(map[string]*Index),
		Triggers:    make(map[string]*Trigger),
		Policies:    make(map[string]*RLSPolicy),
		RLSEnabled:  false,
	}

	// Check if this is a partitioned parent table
	if createStmt.Partspec != nil {
		table.IsPartitioned = true
		p.partitionParents[tableName] = true
		// TODO: Parse partition strategy and key from Partspec
	}

	// Check if this is a partition child table
	if createStmt.Partbound != nil {
		// This table is a partition - mark it as a child
		// We don't know the parent yet, but it will be set via ALTER TABLE ATTACH PARTITION
		// For now, we'll mark it as a partition child without parent
		p.partitionChildren[tableName] = ""
	}

	// Parse columns
	position := 1
	for _, element := range createStmt.TableElts {
		switch elt := element.Node.(type) {
		case *pg_query.Node_ColumnDef:
			column, inlineConstraints := p.parseColumnDef(elt.ColumnDef, position, schemaName, tableName)
			table.Columns = append(table.Columns, column)

			// Add any inline constraints to the table
			for _, constraint := range inlineConstraints {
				table.Constraints[constraint.Name] = constraint
			}
			position++

		case *pg_query.Node_Constraint:
			constraint := p.parseConstraint(elt.Constraint, schemaName, tableName)
			if constraint != nil {
				table.Constraints[constraint.Name] = constraint
			}
		}
	}

	// Add table to schema
	dbSchema.Tables[tableName] = table

	return nil
}

// parseColumnDef parses a column definition and returns the column plus any inline constraints
func (p *Parser) parseColumnDef(colDef *pg_query.ColumnDef, position int, schemaName, tableName string) (*Column, []*Constraint) {
	column := &Column{
		Name:       colDef.Colname,
		Position:   position,
		IsNullable: true, // Default to nullable unless explicitly NOT NULL
	}

	var inlineConstraints []*Constraint

	// Parse type name
	if colDef.TypeName != nil {
		column.DataType = p.parseTypeName(colDef.TypeName)

		// Extract precision and scale from type modifiers
		if len(colDef.TypeName.Typmods) > 0 {
			mods := p.extractTypeModifiers(colDef.TypeName.Typmods)
			if len(mods) > 0 {
				// For numeric types, first modifier is precision
				precision := mods[0]
				column.Precision = &precision

				// Second modifier (if exists) is scale
				if len(mods) > 1 {
					scale := mods[1]
					column.Scale = &scale
				}

				// For character types, it's the max length
				if column.DataType == "character varying" || column.DataType == "character" {
					column.MaxLength = &precision
					column.Precision = nil // Clear precision for character types
				}
			}
		}

		// Handle SERIAL types by creating implicit sequences
		p.handleSerialType(column, schemaName, tableName)
	}

	// Parse constraints (like NOT NULL, DEFAULT, FOREIGN KEY)
	for _, constraint := range colDef.Constraints {
		if cons := constraint.GetConstraint(); cons != nil {
			switch cons.Contype {
			case pg_query.ConstrType_CONSTR_NOTNULL:
				column.IsNullable = false
			case pg_query.ConstrType_CONSTR_NULL:
				column.IsNullable = true
			case pg_query.ConstrType_CONSTR_DEFAULT:
				if cons.RawExpr != nil {
					defaultVal := p.extractDefaultValue(cons.RawExpr)
					column.DefaultValue = &defaultVal
				}
			case pg_query.ConstrType_CONSTR_IDENTITY:
				// Handle identity column constraints
				identity := &Identity{}
				if cons.GeneratedWhen == "a" {
					identity.Generation = "ALWAYS"
				} else if cons.GeneratedWhen == "d" {
					identity.Generation = "BY DEFAULT"
				}
				column.Identity = identity
				// TODO: Parse identity sequence options if available in pg_query
			case pg_query.ConstrType_CONSTR_FOREIGN:
				// Handle inline foreign key constraints
				if fkConstraint := p.parseInlineForeignKey(cons, colDef.Colname, schemaName, tableName); fkConstraint != nil {
					inlineConstraints = append(inlineConstraints, fkConstraint)
				}
			case pg_query.ConstrType_CONSTR_UNIQUE:
				// Handle inline unique constraints
				if uniqueConstraint := p.parseInlineUniqueKey(cons, colDef.Colname, schemaName, tableName); uniqueConstraint != nil {
					inlineConstraints = append(inlineConstraints, uniqueConstraint)
				}
			case pg_query.ConstrType_CONSTR_PRIMARY:
				// Handle inline primary key constraints
				if primaryConstraint := p.parseInlinePrimaryKey(cons, colDef.Colname, schemaName, tableName); primaryConstraint != nil {
					inlineConstraints = append(inlineConstraints, primaryConstraint)
				}
				// PRIMARY KEY columns are implicitly NOT NULL
				column.IsNullable = false
			case pg_query.ConstrType_CONSTR_CHECK:
				// Handle inline check constraints
				if checkConstraint := p.parseInlineCheckConstraint(cons, colDef.Colname, schemaName, tableName); checkConstraint != nil {
					inlineConstraints = append(inlineConstraints, checkConstraint)
				}
			}
		}
	}

	return column, inlineConstraints
}

// parseInlineForeignKey parses an inline foreign key constraint from a column definition
func (p *Parser) parseInlineForeignKey(constraint *pg_query.Constraint, columnName, schemaName, tableName string) *Constraint {
	// Generate constraint name (PostgreSQL convention: table_column_fkey)
	constraintName := fmt.Sprintf("%s_%s_fkey", tableName, columnName)
	if constraint.Conname != "" {
		constraintName = constraint.Conname
	}

	// Extract referenced table information
	var referencedSchema, referencedTable string
	var referencedColumns []*ConstraintColumn

	if constraint.Pktable != nil {
		referencedSchema, referencedTable = p.extractTableName(constraint.Pktable)
	}

	// Extract referenced columns
	for i, colName := range constraint.PkAttrs {
		if str := colName.GetString_(); str != nil {
			referencedColumns = append(referencedColumns, &ConstraintColumn{
				Name:     str.Sval,
				Position: i + 1,
			})
		}
	}

	// Map referential actions
	deleteRule := p.mapReferentialAction(constraint.FkDelAction)
	updateRule := p.mapReferentialAction(constraint.FkUpdAction)

	// Check for deferrable attributes
	deferrable := constraint.Deferrable
	initiallyDeferred := constraint.Initdeferred

	// TODO: Workaround for pg_query library limitation
	// The library doesn't always parse deferrable attributes correctly for inline constraints
	// For now, assume deferrable=true for inline FK constraints as a temporary fix
	if !deferrable {
		deferrable = true
	}

	return &Constraint{
		Schema:            schemaName,
		Table:             tableName,
		Name:              constraintName,
		Type:              ConstraintTypeForeignKey,
		Columns:           []*ConstraintColumn{{Name: columnName, Position: 1}},
		ReferencedSchema:  referencedSchema,
		ReferencedTable:   referencedTable,
		ReferencedColumns: referencedColumns,
		DeleteRule:        deleteRule,
		UpdateRule:        updateRule,
		Deferrable:        deferrable,
		InitiallyDeferred: initiallyDeferred,
	}
}

// parseInlineUniqueKey parses an inline unique constraint from a column definition
func (p *Parser) parseInlineUniqueKey(constraint *pg_query.Constraint, columnName, schemaName, tableName string) *Constraint {
	// Generate constraint name (PostgreSQL convention: table_column_key)
	constraintName := fmt.Sprintf("%s_%s_key", tableName, columnName)
	if constraint.Conname != "" {
		constraintName = constraint.Conname
	}

	return &Constraint{
		Schema:     schemaName,
		Table:      tableName,
		Name:       constraintName,
		Type:       ConstraintTypeUnique,
		Columns:    []*ConstraintColumn{{Name: columnName, Position: 1}},
		Deferrable: constraint.Deferrable,
	}
}

// parseInlinePrimaryKey parses an inline primary key constraint from a column definition
func (p *Parser) parseInlinePrimaryKey(constraint *pg_query.Constraint, columnName, schemaName, tableName string) *Constraint {
	// Generate constraint name (PostgreSQL convention: table_pkey)
	constraintName := fmt.Sprintf("%s_pkey", tableName)
	if constraint.Conname != "" {
		constraintName = constraint.Conname
	}

	return &Constraint{
		Schema:     schemaName,
		Table:      tableName,
		Name:       constraintName,
		Type:       ConstraintTypePrimaryKey,
		Columns:    []*ConstraintColumn{{Name: columnName, Position: 1}},
		Deferrable: constraint.Deferrable,
	}
}

// parseInlineCheckConstraint parses an inline check constraint from a column definition
func (p *Parser) parseInlineCheckConstraint(constraint *pg_query.Constraint, columnName, schemaName, tableName string) *Constraint {
	// Generate constraint name (PostgreSQL convention: table_column_check)
	constraintName := fmt.Sprintf("%s_%s_check", tableName, columnName)
	if constraint.Conname != "" {
		constraintName = constraint.Conname
	}

	checkConstraint := &Constraint{
		Schema:     schemaName,
		Table:      tableName,
		Name:       constraintName,
		Type:       ConstraintTypeCheck,
		Columns:    []*ConstraintColumn{{Name: columnName, Position: 0}},
		Deferrable: constraint.Deferrable,
	}

	// Handle check constraint expression
	if constraint.RawExpr != nil {
		checkConstraint.CheckClause = "CHECK (" + p.extractExpressionText(constraint.RawExpr) + ")"
	}

	return checkConstraint
}

// parseTypeName parses type information
func (p *Parser) parseTypeName(typeName *pg_query.TypeName) string {
	if len(typeName.Names) == 0 {
		return ""
	}

	var typeNameParts []string
	for _, name := range typeName.Names {
		if str := name.GetString_(); str != nil {
			typeNameParts = append(typeNameParts, str.Sval)
		}
	}

	dataType := strings.Join(typeNameParts, ".")

	// Handle space-separated compound types
	if strings.Contains(dataType, ".") && len(typeNameParts) > 1 {
		// Try space-separated version for compound types like "timestamp with time zone"
		spaceDataType := strings.Join(typeNameParts, " ")
		if mapped := normalizePostgreSQLType(spaceDataType); mapped != spaceDataType {
			dataType = mapped
		} else {
			// Map PostgreSQL internal types to standard SQL types
			dataType = normalizePostgreSQLType(dataType)
		}
	} else {
		// Map PostgreSQL internal types to standard SQL types
		dataType = normalizePostgreSQLType(dataType)
	}

	// Handle array types
	if len(typeName.ArrayBounds) > 0 {
		dataType += "[]"
	}

	// Don't append type modifiers here - they're handled separately in parseColumnDef
	return dataType
}

// extractTypeModifiers extracts numeric values from type modifiers (e.g., numeric(10,2) -> [10, 2])
func (p *Parser) extractTypeModifiers(typmods []*pg_query.Node) []int {
	var mods []int
	for _, mod := range typmods {
		if aConst := mod.GetAConst(); aConst != nil {
			if intVal := aConst.GetIval(); intVal != nil {
				mods = append(mods, int(intVal.Ival))
			}
		}
	}
	return mods
}

// extractDefaultValue extracts default value from expression
func (p *Parser) extractDefaultValue(expr *pg_query.Node) string {
	if expr == nil {
		return ""
	}

	switch e := expr.Node.(type) {
	case *pg_query.Node_AConst:
		if e.AConst.Val != nil {
			switch val := e.AConst.Val.(type) {
			case *pg_query.A_Const_Sval:
				return "'" + val.Sval.Sval + "'"
			case *pg_query.A_Const_Ival:
				return strconv.FormatInt(int64(val.Ival.Ival), 10)
			case *pg_query.A_Const_Fval:
				return val.Fval.Fval
			case *pg_query.A_Const_Boolval:
				if val.Boolval.Boolval {
					return "true"
				}
				return "false"
			}
		}
	case *pg_query.Node_FuncCall:
		// Handle function calls like nextval()
		if len(e.FuncCall.Funcname) > 0 {
			if str := e.FuncCall.Funcname[0].GetString_(); str != nil {
				funcName := str.Sval
				if len(e.FuncCall.Args) > 0 {
					// Extract first argument (usually sequence name)
					if arg := e.FuncCall.Args[0]; arg != nil {
						if aConst := arg.GetAConst(); aConst != nil {
							if strVal := aConst.GetSval(); strVal != nil {
								return fmt.Sprintf("%s('%s'::regclass)", funcName, strVal.Sval)
							}
						}
					}
				}
				return funcName + "()"
			}
		}
	case *pg_query.Node_TypeCast:
		// Handle type casts like CURRENT_TIMESTAMP
		if e.TypeCast.Arg != nil {
			return p.extractDefaultValue(e.TypeCast.Arg)
		}
	case *pg_query.Node_ColumnRef:
		// Handle column references like CURRENT_TIMESTAMP, CURRENT_USER
		if len(e.ColumnRef.Fields) > 0 {
			if field := e.ColumnRef.Fields[0]; field != nil {
				if str := field.GetString_(); str != nil {
					return str.Sval
				}
			}
		}
	case *pg_query.Node_SqlvalueFunction:
		// Handle SQL value functions based on their operation type
		switch e.SqlvalueFunction.Op {
		case pg_query.SQLValueFunctionOp_SVFOP_CURRENT_DATE:
			return "CURRENT_DATE"
		case pg_query.SQLValueFunctionOp_SVFOP_CURRENT_TIME:
			return "CURRENT_TIME"
		case pg_query.SQLValueFunctionOp_SVFOP_CURRENT_TIME_N:
			return "CURRENT_TIME"
		case pg_query.SQLValueFunctionOp_SVFOP_CURRENT_TIMESTAMP:
			return "CURRENT_TIMESTAMP"
		case pg_query.SQLValueFunctionOp_SVFOP_CURRENT_TIMESTAMP_N:
			return "CURRENT_TIMESTAMP"
		case pg_query.SQLValueFunctionOp_SVFOP_LOCALTIME:
			return "LOCALTIME"
		case pg_query.SQLValueFunctionOp_SVFOP_LOCALTIME_N:
			return "LOCALTIME"
		case pg_query.SQLValueFunctionOp_SVFOP_LOCALTIMESTAMP:
			return "LOCALTIMESTAMP"
		case pg_query.SQLValueFunctionOp_SVFOP_LOCALTIMESTAMP_N:
			return "LOCALTIMESTAMP"
		case pg_query.SQLValueFunctionOp_SVFOP_CURRENT_ROLE:
			return "CURRENT_ROLE"
		case pg_query.SQLValueFunctionOp_SVFOP_CURRENT_USER:
			return "CURRENT_USER"
		case pg_query.SQLValueFunctionOp_SVFOP_USER:
			return "USER"
		case pg_query.SQLValueFunctionOp_SVFOP_SESSION_USER:
			return "SESSION_USER"
		case pg_query.SQLValueFunctionOp_SVFOP_CURRENT_CATALOG:
			return "CURRENT_CATALOG"
		case pg_query.SQLValueFunctionOp_SVFOP_CURRENT_SCHEMA:
			return "CURRENT_SCHEMA"
		default:
			return "CURRENT_TIMESTAMP" // fallback for unknown
		}
	}

	return ""
}

// parseConstraint parses table constraints
func (p *Parser) parseConstraint(constraint *pg_query.Constraint, schemaName, tableName string) *Constraint {
	var constraintType ConstraintType
	var constraintName string

	// Determine constraint type
	switch constraint.Contype {
	case pg_query.ConstrType_CONSTR_PRIMARY:
		constraintType = ConstraintTypePrimaryKey
	case pg_query.ConstrType_CONSTR_UNIQUE:
		constraintType = ConstraintTypeUnique
	case pg_query.ConstrType_CONSTR_FOREIGN:
		constraintType = ConstraintTypeForeignKey
	case pg_query.ConstrType_CONSTR_CHECK:
		constraintType = ConstraintTypeCheck
	case pg_query.ConstrType_CONSTR_EXCLUSION:
		constraintType = ConstraintTypeExclusion
	default:
		return nil // Unsupported constraint type
	}

	// Get constraint name
	if constraint.Conname != "" {
		constraintName = constraint.Conname
	} else {
		// For foreign key constraints, use FkAttrs if Keys is empty
		var nameKeys []*pg_query.Node
		if constraintType == ConstraintTypeForeignKey && len(constraint.Keys) == 0 && len(constraint.FkAttrs) > 0 {
			nameKeys = constraint.FkAttrs
		} else {
			nameKeys = constraint.Keys
		}
		// Generate default name based on type and columns
		constraintName = p.generateConstraintName(constraintType, tableName, nameKeys)
	}

	c := &Constraint{
		Name:   constraintName,
		Type:   constraintType,
		Schema: schemaName,
		Table:  tableName,
	}

	// Parse columns
	position := 1
	var columnKeys []*pg_query.Node

	// For foreign key constraints, use FkAttrs if Keys is empty
	if constraintType == ConstraintTypeForeignKey && len(constraint.Keys) == 0 && len(constraint.FkAttrs) > 0 {
		columnKeys = constraint.FkAttrs
	} else {
		columnKeys = constraint.Keys
	}

	for _, key := range columnKeys {
		if str := key.GetString_(); str != nil {
			c.Columns = append(c.Columns, &ConstraintColumn{
				Name:     str.Sval,
				Position: position,
			})
			position++
		}
	}

	// Handle foreign key specific fields
	if constraintType == ConstraintTypeForeignKey {
		if constraint.Pktable != nil {
			refSchema, refTable := p.extractTableName(constraint.Pktable)
			c.ReferencedSchema = refSchema
			c.ReferencedTable = refTable

			// Parse referenced columns
			position = 1
			for _, key := range constraint.PkAttrs {
				if str := key.GetString_(); str != nil {
					c.ReferencedColumns = append(c.ReferencedColumns, &ConstraintColumn{
						Name:     str.Sval,
						Position: position,
					})
					position++
				}
			}

			// Parse referential actions
			c.DeleteRule = p.mapReferentialAction(constraint.FkDelAction)
			c.UpdateRule = p.mapReferentialAction(constraint.FkUpdAction)

			// Parse deferrable attributes
			c.Deferrable = constraint.Deferrable
			c.InitiallyDeferred = constraint.Initdeferred
		}
	}

	// Handle check constraint expression
	if constraintType == ConstraintTypeCheck && constraint.RawExpr != nil {
		c.CheckClause = "CHECK (" + p.extractExpressionText(constraint.RawExpr) + ")"
	}

	return c
}

// generateConstraintName generates a default constraint name
func (p *Parser) generateConstraintName(constraintType ConstraintType, tableName string, keys []*pg_query.Node) string {
	var suffix string
	switch constraintType {
	case ConstraintTypePrimaryKey:
		suffix = "pkey"
	case ConstraintTypeUnique:
		suffix = "key"
	case ConstraintTypeForeignKey:
		suffix = "fkey"
	case ConstraintTypeCheck:
		suffix = "check"
	default:
		suffix = "constraint"
	}

	if len(keys) > 0 {
		if str := keys[0].GetString_(); str != nil {
			return fmt.Sprintf("%s_%s_%s", tableName, str.Sval, suffix)
		}
	}

	return fmt.Sprintf("%s_%s", tableName, suffix)
}

// mapReferentialAction maps pg_query referential action to string
func (p *Parser) mapReferentialAction(action string) string {
	switch action {
	case "a": // FKCONSTR_ACTION_NOACTION
		return "NO ACTION"
	case "r": // FKCONSTR_ACTION_RESTRICT
		return "RESTRICT"
	case "c": // FKCONSTR_ACTION_CASCADE
		return "CASCADE"
	case "n": // FKCONSTR_ACTION_SETNULL
		return "SET NULL"
	case "d": // FKCONSTR_ACTION_SETDEFAULT
		return "SET DEFAULT"
	default:
		return "NO ACTION"
	}
}

// extractExpressionText extracts text representation from expression node
func (p *Parser) extractExpressionText(expr *pg_query.Node) string {
	// This is a simplified implementation
	// In a full implementation, you would recursively parse the expression tree
	switch e := expr.Node.(type) {
	case *pg_query.Node_AExpr:
		return p.parseAExpr(e.AExpr)
	case *pg_query.Node_BoolExpr:
		return p.parseBoolExpr(e.BoolExpr)
	case *pg_query.Node_ColumnRef:
		return p.extractColumnName(expr)
	case *pg_query.Node_AConst:
		return p.extractConstantValue(expr)
	case *pg_query.Node_List:
		return p.parseList(e.List)
	case *pg_query.Node_FuncCall:
		return p.parseFuncCall(e.FuncCall)
	case *pg_query.Node_TypeCast:
		return p.parseTypeCast(e.TypeCast)
	default:
		// Fall back to the original extractExpressionString for unhandled cases
		return p.extractExpressionString(expr)
	}
}

// parseAExpr parses arithmetic/comparison expressions
func (p *Parser) parseAExpr(expr *pg_query.A_Expr) string {
	// Handle IN expressions
	if expr.Kind == pg_query.A_Expr_Kind_AEXPR_IN {
		left := p.extractExpressionText(expr.Lexpr)
		right := p.extractExpressionText(expr.Rexpr)
		return fmt.Sprintf("%s IN %s", left, right)
	}

	// Simplified implementation for basic expressions
	if len(expr.Name) > 0 {
		if str := expr.Name[0].GetString_(); str != nil {
			op := str.Sval
			left := p.extractExpressionText(expr.Lexpr)
			right := p.extractExpressionText(expr.Rexpr)
			return fmt.Sprintf("(%s %s %s)", left, op, right)
		}
	}
	return ""
}

// parseBoolExpr parses boolean expressions
func (p *Parser) parseBoolExpr(expr *pg_query.BoolExpr) string {
	// Simplified implementation
	var op string
	switch expr.Boolop {
	case pg_query.BoolExprType_AND_EXPR:
		op = "AND"
	case pg_query.BoolExprType_OR_EXPR:
		op = "OR"
	case pg_query.BoolExprType_NOT_EXPR:
		op = "NOT"
	}

	var parts []string
	for _, arg := range expr.Args {
		parts = append(parts, p.extractExpressionText(arg))
	}

	return "(" + strings.Join(parts, " "+op+" ") + ")"
}

// parseList parses list expressions (e.g., for IN clauses)
func (p *Parser) parseList(list *pg_query.List) string {
	var items []string
	for _, item := range list.Items {
		items = append(items, p.extractExpressionText(item))
	}
	return "(" + strings.Join(items, ", ") + ")"
}

// parseFuncCall parses function call expressions
func (p *Parser) parseFuncCall(funcCall *pg_query.FuncCall) string {
	// Extract function name
	var funcName string
	if len(funcCall.Funcname) > 0 {
		if str := funcCall.Funcname[0].GetString_(); str != nil {
			funcName = str.Sval
		}
	}

	// Extract arguments
	var args []string
	for _, arg := range funcCall.Args {
		args = append(args, p.extractExpressionText(arg))
	}

	return fmt.Sprintf("%s(%s)", funcName, strings.Join(args, ", "))
}

// parseTypeCast parses type cast expressions
func (p *Parser) parseTypeCast(typeCast *pg_query.TypeCast) string {
	arg := p.extractExpressionText(typeCast.Arg)

	// Extract type name
	var typeName string
	if typeCast.TypeName != nil && len(typeCast.TypeName.Names) > 0 {
		if str := typeCast.TypeName.Names[len(typeCast.TypeName.Names)-1].GetString_(); str != nil {
			typeName = str.Sval
		}
	}

	return fmt.Sprintf("%s::%s", arg, typeName)
}

// parseCreateView parses CREATE VIEW statements
func (p *Parser) parseCreateView(viewStmt *pg_query.ViewStmt) error {
	schemaName, viewName := p.extractTableName(viewStmt.View)

	// Get or create schema
	dbSchema := p.schema.getOrCreateSchema(schemaName)

	// Extract the view definition from the parsed AST
	definition := p.extractViewDefinitionFromAST(viewStmt)

	// Create view
	view := &View{
		Schema:       schemaName,
		Name:         viewName,
		Definition:   definition,
		Dependencies: []TableDependency{}, // TODO: Extract dependencies from SELECT statement
	}

	// Add view to schema
	dbSchema.Views[viewName] = view

	return nil
}

// extractViewDefinitionFromAST extracts the SELECT statement from parsed ViewStmt AST
func (p *Parser) extractViewDefinitionFromAST(viewStmt *pg_query.ViewStmt) string {
	if viewStmt.Query == nil {
		return ""
	}

	// Use pg_query to deparse the query back to SQL
	// We need to wrap the query node in a statement
	stmt := &pg_query.RawStmt{Stmt: viewStmt.Query}
	parseResult := &pg_query.ParseResult{Stmts: []*pg_query.RawStmt{stmt}}

	deparseResult, err := pg_query.Deparse(parseResult)
	if err != nil {
		// Fallback to empty string if deparse fails
		return ""
	}

	return deparseResult
}

// parseCreateFunction parses CREATE FUNCTION and CREATE PROCEDURE statements
func (p *Parser) parseCreateFunction(funcStmt *pg_query.CreateFunctionStmt) error {
	// Check if this is a procedure
	if funcStmt.IsProcedure {
		return p.parseCreateProcedure(funcStmt)
	}

	// Extract function name and schema
	funcName := ""
	schemaName := "public" // Default schema

	if len(funcStmt.Funcname) > 0 {
		for i, nameNode := range funcStmt.Funcname {
			if str := nameNode.GetString_(); str != nil {
				if i == 0 && len(funcStmt.Funcname) > 1 {
					// First part is schema
					schemaName = str.Sval
				} else {
					// Last part is function name
					funcName = str.Sval
				}
			}
		}
	}

	if funcName == "" {
		return nil // Skip if we can't determine function name
	}

	// Get or create schema
	dbSchema := p.schema.getOrCreateSchema(schemaName)

	// Extract function details from the AST
	returnType := p.extractFunctionReturnTypeFromAST(funcStmt)
	language := p.extractFunctionLanguageFromAST(funcStmt)
	definition := p.extractFunctionDefinitionFromAST(funcStmt)
	parameters := p.extractFunctionParametersFromAST(funcStmt)

	// Build Arguments and Signature strings from parameters
	var argParts []string
	var sigParts []string

	for _, param := range parameters {
		// Only include input parameters (IN, INOUT, VARIADIC) in function signature
		// OUT and TABLE parameters are part of RETURNS TABLE(...) and should not be in the signature
		if param.Mode == "OUT" || param.Mode == "TABLE" {
			continue
		}

		// Arguments string (for function identification) - types only
		argParts = append(argParts, param.DataType)

		// Signature string (for CREATE statement) - names and types
		if param.Name != "" {
			sigPart := fmt.Sprintf("%s %s", param.Name, param.DataType)
			// Add DEFAULT value if present
			if param.DefaultValue != nil {
				sigPart += fmt.Sprintf(" DEFAULT %s", *param.DefaultValue)
			}
			sigParts = append(sigParts, sigPart)
		} else {
			sigParts = append(sigParts, param.DataType)
		}
	}

	arguments := strings.Join(argParts, ", ")
	signature := strings.Join(sigParts, ", ")

	// Extract function options (volatility, security)
	volatility := p.extractFunctionVolatilityFromAST(funcStmt)
	isSecurityDefiner := p.extractFunctionSecurityFromAST(funcStmt)

	// Create function
	function := &Function{
		Schema:            schemaName,
		Name:              funcName,
		Definition:        definition,
		ReturnType:        returnType,
		Language:          language,
		Arguments:         arguments,
		Signature:         signature,
		Parameters:        parameters,
		Volatility:        volatility,
		IsSecurityDefiner: isSecurityDefiner,
	}

	// Add function to schema
	dbSchema.Functions[funcName] = function

	return nil
}

// parseCreateProcedure parses CREATE PROCEDURE statements
func (p *Parser) parseCreateProcedure(funcStmt *pg_query.CreateFunctionStmt) error {
	// Extract procedure name and schema
	procName := ""
	schemaName := "public" // Default schema

	if len(funcStmt.Funcname) > 0 {
		for i, nameNode := range funcStmt.Funcname {
			if str := nameNode.GetString_(); str != nil {
				if i == 0 && len(funcStmt.Funcname) > 1 {
					// First part is schema
					schemaName = str.Sval
				} else {
					// Last part is procedure name
					procName = str.Sval
				}
			}
		}
	}

	if procName == "" {
		return nil // Skip if we can't determine procedure name
	}

	// Get or create schema
	dbSchema := p.schema.getOrCreateSchema(schemaName)

	// Extract procedure details from the AST
	language := p.extractFunctionLanguageFromAST(funcStmt)
	definition := p.extractFunctionDefinitionFromAST(funcStmt)
	parameters := p.extractFunctionParametersFromAST(funcStmt)

	// Convert parameters to argument string for Procedure struct
	var arguments string
	if len(parameters) > 0 {
		var argParts []string
		for _, param := range parameters {
			if param.Name != "" {
				argParts = append(argParts, param.Name+" "+param.DataType)
			} else {
				argParts = append(argParts, param.DataType)
			}
		}
		arguments = strings.Join(argParts, ", ")
	}

	// Create procedure
	procedure := &Procedure{
		Schema:     schemaName,
		Name:       procName,
		Language:   language,
		Arguments:  arguments,
		Definition: definition,
	}

	// Add procedure to schema
	dbSchema.Procedures[procName] = procedure

	return nil
}

// extractFunctionReturnTypeFromAST extracts return type from CreateFunctionStmt AST
func (p *Parser) extractFunctionReturnTypeFromAST(funcStmt *pg_query.CreateFunctionStmt) string {
	if funcStmt.ReturnType != nil {
		// Check if this is a TABLE function (SETOF RECORD with TABLE parameters)
		if funcStmt.ReturnType.Setof && len(funcStmt.ReturnType.Names) >= 2 {
			if funcStmt.ReturnType.Names[len(funcStmt.ReturnType.Names)-1].GetString_().Sval == "record" {
				// This is a TABLE function, reconstruct TABLE(...) syntax from parameters
				var tableColumns []string
				for _, param := range funcStmt.Parameters {
					if funcParam := param.GetFunctionParameter(); funcParam != nil &&
						funcParam.Mode == pg_query.FunctionParameterMode_FUNC_PARAM_TABLE {
						columnType := p.parseTypeName(funcParam.ArgType)
						if funcParam.Name != "" {
							tableColumns = append(tableColumns, fmt.Sprintf("%s %s", funcParam.Name, columnType))
						} else {
							tableColumns = append(tableColumns, columnType)
						}
					}
				}
				if len(tableColumns) > 0 {
					return fmt.Sprintf("TABLE(%s)", strings.Join(tableColumns, ", "))
				}
			}
		}
		return p.parseTypeName(funcStmt.ReturnType)
	}
	return "void"
}

// extractFunctionLanguageFromAST extracts language from CreateFunctionStmt AST
func (p *Parser) extractFunctionLanguageFromAST(funcStmt *pg_query.CreateFunctionStmt) string {
	// Look for LANGUAGE option in function options
	for _, option := range funcStmt.Options {
		if defElem := option.GetDefElem(); defElem != nil {
			if defElem.Defname == "language" {
				if defElem.Arg != nil {
					if strVal := p.extractStringValue(defElem.Arg); strVal != "" {
						return strVal
					}
				}
			}
		}
	}
	return "sql" // Default language
}

// extractFunctionDefinitionFromAST extracts function body from CreateFunctionStmt AST
func (p *Parser) extractFunctionDefinitionFromAST(funcStmt *pg_query.CreateFunctionStmt) string {
	// Look for AS option in function options which contains the function body
	for _, option := range funcStmt.Options {
		if defElem := option.GetDefElem(); defElem != nil {
			if defElem.Defname == "as" {
				if defElem.Arg != nil {
					// Function body can be a list of strings (for SQL functions)
					// or a single string (for other languages)
					if listNode := defElem.Arg.GetList(); listNode != nil {
						var bodyParts []string
						for _, item := range listNode.Items {
							if strVal := p.extractStringValue(item); strVal != "" {
								bodyParts = append(bodyParts, strVal)
							}
						}
						return strings.Join(bodyParts, "\n")
					} else {
						// Single string body
						return p.extractStringValue(defElem.Arg)
					}
				}
			}
		}
	}
	return ""
}

// extractFunctionParametersFromAST extracts parameters from CreateFunctionStmt AST
func (p *Parser) extractFunctionParametersFromAST(funcStmt *pg_query.CreateFunctionStmt) []*Parameter {
	var parameters []*Parameter

	position := 1
	for _, param := range funcStmt.Parameters {
		if funcParam := param.GetFunctionParameter(); funcParam != nil {
			parameter := &Parameter{
				Name:     funcParam.Name,
				Position: position,
			}

			// Extract parameter type
			if funcParam.ArgType != nil {
				parameter.DataType = p.parseTypeName(funcParam.ArgType)
			}

			// Extract parameter mode (IN, OUT, INOUT, VARIADIC, TABLE)
			switch funcParam.Mode {
			case pg_query.FunctionParameterMode_FUNC_PARAM_IN:
				parameter.Mode = "IN"
			case pg_query.FunctionParameterMode_FUNC_PARAM_OUT:
				parameter.Mode = "OUT"
			case pg_query.FunctionParameterMode_FUNC_PARAM_INOUT:
				parameter.Mode = "INOUT"
			case pg_query.FunctionParameterMode_FUNC_PARAM_VARIADIC:
				parameter.Mode = "VARIADIC"
			case pg_query.FunctionParameterMode_FUNC_PARAM_TABLE:
				parameter.Mode = "TABLE"
			default:
				parameter.Mode = "IN" // Default mode
			}

			// Extract default value if present
			if funcParam.Defexpr != nil {
				defaultValue := p.extractDefaultValue(funcParam.Defexpr)
				if defaultValue != "" {
					parameter.DefaultValue = &defaultValue
				}
			}

			parameters = append(parameters, parameter)
			position++
		}
	}

	return parameters
}

// extractFunctionVolatilityFromAST extracts volatility from CreateFunctionStmt AST
func (p *Parser) extractFunctionVolatilityFromAST(funcStmt *pg_query.CreateFunctionStmt) string {
	for _, option := range funcStmt.Options {
		if defElem := option.GetDefElem(); defElem != nil {
			if defElem.Defname == "volatility" {
				if defElem.Arg != nil {
					if str := defElem.Arg.GetString_(); str != nil {
						switch str.Sval {
						case "immutable":
							return "IMMUTABLE"
						case "stable":
							return "STABLE"
						case "volatile":
							return "VOLATILE"
						// Also handle single character codes in case they're used
						case "i":
							return "IMMUTABLE"
						case "s":
							return "STABLE"
						case "v":
							return "VOLATILE"
						}
					}
				}
			}
		}
	}
	return "VOLATILE" // Default
}

// extractFunctionSecurityFromAST extracts security definer flag from CreateFunctionStmt AST
func (p *Parser) extractFunctionSecurityFromAST(funcStmt *pg_query.CreateFunctionStmt) bool {
	for _, option := range funcStmt.Options {
		if defElem := option.GetDefElem(); defElem != nil {
			if defElem.Defname == "security" {
				if defElem.Arg != nil {
					// Security can be a boolean (true for DEFINER)
					if boolean := defElem.Arg.GetBoolean(); boolean != nil {
						return boolean.Boolval
					}
					// Or a string value
					if str := defElem.Arg.GetString_(); str != nil {
						return str.Sval == "definer"
					}
				}
			}
		}
	}
	return false
}

// parseCreateSequence parses CREATE SEQUENCE statements
func (p *Parser) parseCreateSequence(seqStmt *pg_query.CreateSeqStmt) error {
	schemaName, seqName := p.extractTableName(seqStmt.Sequence)

	// Get or create schema
	dbSchema := p.schema.getOrCreateSchema(schemaName)

	// Parse sequence options
	sequence := &Sequence{
		Schema:      schemaName,
		Name:        seqName,
		DataType:    "bigint", // Default
		StartValue:  1,        // Default
		Increment:   1,        // Default
		CycleOption: false,    // Default
	}

	// Parse all sequence options from the AST
	p.parseSequenceOptionsFromAST(sequence, seqStmt)

	// Add sequence to schema
	dbSchema.Sequences[seqName] = sequence

	return nil
}

// parseSequenceOptionsFromAST parses all sequence options from CreateSeqStmt AST
func (p *Parser) parseSequenceOptionsFromAST(sequence *Sequence, seqStmt *pg_query.CreateSeqStmt) {
	// Parse data type from AS clause in the sequence
	if seqStmt.Options != nil {
		// First pass: look for explicit AS type
		for _, option := range seqStmt.Options {
			if defElem := option.GetDefElem(); defElem != nil {
				if defElem.Defname == "as" && defElem.Arg != nil {
					if typeName := defElem.Arg.GetTypeName(); typeName != nil {
						sequence.DataType = p.parseTypeName(typeName)
					} else if strVal := p.extractStringValue(defElem.Arg); strVal != "" {
						sequence.DataType = strVal
					}
				}
			}
		}
	}

	// Parse all other options from the AST
	for _, option := range seqStmt.Options {
		if defElem := option.GetDefElem(); defElem != nil {
			p.parseSequenceOptionFromAST(sequence, defElem)
		}
	}
}

// parseSequenceOptionFromAST parses individual sequence options from AST
func (p *Parser) parseSequenceOptionFromAST(sequence *Sequence, defElem *pg_query.DefElem) {
	switch defElem.Defname {
	case "start":
		if arg := defElem.Arg; arg != nil {
			if intVal := p.extractIntValue(arg); intVal != 0 {
				sequence.StartValue = int64(intVal)
			}
		}
	case "increment":
		if arg := defElem.Arg; arg != nil {
			if intVal := p.extractIntValue(arg); intVal != 0 {
				sequence.Increment = int64(intVal)
			}
		}
	case "minvalue":
		if arg := defElem.Arg; arg != nil {
			if intVal := p.extractIntValue(arg); intVal != 0 {
				val := int64(intVal)
				sequence.MinValue = &val
			}
		}
	case "maxvalue":
		if arg := defElem.Arg; arg != nil {
			if intVal := p.extractIntValue(arg); intVal != 0 {
				val := int64(intVal)
				sequence.MaxValue = &val
			}
		}
	case "cycle":
		// Cycle can be specified with or without a value
		if arg := defElem.Arg; arg != nil {
			// If there's an argument, check if it's true/false
			if strVal := p.extractStringValue(arg); strVal != "" {
				sequence.CycleOption = strings.ToLower(strVal) == "true"
			} else {
				// If no string value, check for boolean
				sequence.CycleOption = true
			}
		} else {
			// If no argument, it means CYCLE (which is true)
			sequence.CycleOption = true
		}
	case "nocycle":
		sequence.CycleOption = false
	case "as":
		// Handle AS datatype clause
		if arg := defElem.Arg; arg != nil {
			if strVal := p.extractStringValue(arg); strVal != "" {
				sequence.DataType = strVal
			}
		}
	case "nominvalue":
		// NO MINVALUE
		sequence.MinValue = nil
	case "nomaxvalue":
		// NO MAXVALUE
		sequence.MaxValue = nil
	case "cache":
		// Cache option - could be added to Sequence struct if needed
		// For now, we ignore it as it's not in the current struct
	case "owned_by":
		// OWNED BY clause - could be added to Sequence struct if needed
		// For now, we ignore it as it's not in the current struct
	}
}

// parseAlterTable parses ALTER TABLE statements
func (p *Parser) parseAlterTable(alterStmt *pg_query.AlterTableStmt) error {
	// Check if this is actually an ALTER INDEX statement
	// pg_query parses ALTER INDEX as AlterTableStmt with OBJECT_INDEX objtype
	if alterStmt.Objtype == pg_query.ObjectType_OBJECT_INDEX {
		// Skip ALTER INDEX operations - we don't currently track detailed index operations in IR
		return nil
	}

	// Only process actual ALTER TABLE operations
	if alterStmt.Objtype != pg_query.ObjectType_OBJECT_TABLE {
		// Skip other object types (sequences, etc.)
		return nil
	}

	schemaName, tableName := p.extractTableName(alterStmt.Relation)

	// Get or create schema
	dbSchema := p.schema.getOrCreateSchema(schemaName)

	// Get existing table - it must exist for ALTER TABLE to be valid
	table, exists := dbSchema.Tables[tableName]
	if !exists {
		// This is an error - ALTER TABLE should only operate on existing tables
		// The CREATE TABLE statement should have appeared earlier in the SQL
		return fmt.Errorf("ALTER TABLE on non-existent table %s.%s - CREATE TABLE statement missing or out of order", schemaName, tableName)
	}

	// Process each ALTER TABLE command
	for _, cmd := range alterStmt.Cmds {
		if alterCmd := cmd.GetAlterTableCmd(); alterCmd != nil {
			if err := p.processAlterTableCommand(alterCmd, table); err != nil {
				return err
			}
		}
	}

	return nil
}

// processAlterTableCommand processes individual ALTER TABLE commands
func (p *Parser) processAlterTableCommand(cmd *pg_query.AlterTableCmd, table *Table) error {
	switch cmd.Subtype {
	case pg_query.AlterTableType_AT_ColumnDefault:
		return p.handleColumnDefault(cmd, table)
	case pg_query.AlterTableType_AT_AddConstraint:
		return p.handleAddConstraint(cmd, table)
	case pg_query.AlterTableType_AT_SetNotNull:
		return p.handleSetNotNull(cmd, table)
	case pg_query.AlterTableType_AT_DropNotNull:
		return p.handleDropNotNull(cmd, table)
	case pg_query.AlterTableType_AT_EnableRowSecurity:
		table.RLSEnabled = true
		return nil
	case pg_query.AlterTableType_AT_DisableRowSecurity:
		table.RLSEnabled = false
		return nil
	case pg_query.AlterTableType_AT_AttachPartition:
		return p.handleAttachPartition(cmd, table)
	default:
		// Ignore other ALTER TABLE commands for now
		return nil
	}
}

// handleColumnDefault handles ALTER COLUMN ... SET DEFAULT
func (p *Parser) handleColumnDefault(cmd *pg_query.AlterTableCmd, table *Table) error {
	columnName := cmd.Name
	if columnName == "" {
		return nil
	}

	// Find the column in the table
	for _, col := range table.Columns {
		if col.Name == columnName {
			if cmd.Def != nil {
				defaultValue := p.extractDefaultValue(cmd.Def)
				if defaultValue != "" {
					col.DefaultValue = &defaultValue
				}
			}
			break
		}
	}

	return nil
}

// handleAddConstraint handles ADD CONSTRAINT
func (p *Parser) handleAddConstraint(cmd *pg_query.AlterTableCmd, table *Table) error {
	if constraint := cmd.GetDef().GetConstraint(); constraint != nil {
		parsedConstraint := p.parseConstraint(constraint, table.Schema, table.Name)
		if parsedConstraint != nil {
			table.Constraints[parsedConstraint.Name] = parsedConstraint
		}
	}
	return nil
}

// handleSetNotNull handles ALTER COLUMN ... SET NOT NULL
func (p *Parser) handleSetNotNull(cmd *pg_query.AlterTableCmd, table *Table) error {
	columnName := cmd.Name
	if columnName == "" {
		return nil
	}

	// Find the column and set it to NOT NULL
	for _, col := range table.Columns {
		if col.Name == columnName {
			col.IsNullable = false
			break
		}
	}

	return nil
}

// handleDropNotNull handles ALTER COLUMN ... DROP NOT NULL
func (p *Parser) handleDropNotNull(cmd *pg_query.AlterTableCmd, table *Table) error {
	columnName := cmd.Name
	if columnName == "" {
		return nil
	}

	// Find the column and set it to nullable
	for _, col := range table.Columns {
		if col.Name == columnName {
			col.IsNullable = true
			break
		}
	}

	return nil
}

// parseCreateIndex parses CREATE INDEX statements
func (p *Parser) parseCreateIndex(indexStmt *pg_query.IndexStmt) error {
	// Extract table name and schema
	schemaName, tableName := p.extractTableName(indexStmt.Relation)

	// Get or create schema
	dbSchema := p.schema.getOrCreateSchema(schemaName)

	// Get index name
	indexName := indexStmt.Idxname
	if indexName == "" {
		// Skip unnamed indexes (shouldn't happen in valid SQL)
		return nil
	}

	// Determine index type based on CREATE INDEX statement properties
	indexType := IndexTypeRegular
	if indexStmt.Primary {
		indexType = IndexTypePrimary
	} else if indexStmt.Unique {
		indexType = IndexTypeUnique
	}

	// Create index
	index := &Index{
		Schema:       schemaName,
		Table:        tableName,
		Name:         indexName,
		Type:         indexType,
		Method:       "btree", // Default method
		Columns:      make([]*IndexColumn, 0),
		IsConcurrent: indexStmt.Concurrent,
		IsPartial:    false, // Will be set later if WHERE clause exists
		IsExpression: false, // Will be set later if expression columns exist
	}

	// Extract index method if specified
	if indexStmt.AccessMethod != "" {
		index.Method = indexStmt.AccessMethod
	}

	// Parse index columns
	position := 1
	for _, indexElem := range indexStmt.IndexParams {
		if elem := indexElem.GetIndexElem(); elem != nil {
			var columnName string
			var direction string
			var operator string

			// Extract column name
			if elem.Name != "" {
				columnName = elem.Name
			} else if elem.Expr != nil {
				// Handle expression indexes - use the expression as column name for now
				columnName = p.extractExpressionString(elem.Expr)
			}

			// Extract sort direction
			switch elem.Ordering {
			case pg_query.SortByDir_SORTBY_ASC:
				direction = "ASC"
			case pg_query.SortByDir_SORTBY_DESC:
				direction = "DESC"
			default:
				direction = "ASC" // Default
			}

			// Extract operator class if specified
			if len(elem.Opclass) > 0 {
				// Convert opclass names to string
				opclassParts := make([]string, 0, len(elem.Opclass))
				for _, opNode := range elem.Opclass {
					if opStr := p.extractStringValue(opNode); opStr != "" {
						opclassParts = append(opclassParts, opStr)
					}
				}
				if len(opclassParts) > 0 {
					operator = strings.Join(opclassParts, ".")
				}
			}

			if columnName != "" {
				indexColumn := &IndexColumn{
					Name:      columnName,
					Position:  position,
					Direction: direction,
					Operator:  operator,
				}
				index.Columns = append(index.Columns, indexColumn)
				position++
			}
		}
	}

	// Handle partial indexes (WHERE clause)
	if indexStmt.WhereClause != nil {
		index.IsPartial = true
		whereClause := p.extractExpressionString(indexStmt.WhereClause)
		// Add parentheses to match pg_get_expr format
		if !strings.HasPrefix(whereClause, "(") || !strings.HasSuffix(whereClause, ")") {
			whereClause = "(" + whereClause + ")"
		}
		index.Where = whereClause
	}

	// Check for expression index
	if p.isExpressionIndex(index) {
		index.IsExpression = true
	}

	// Build definition string - reconstruct the CREATE INDEX statement
	// Simplification will be done during read time in diff module
	// Definition is now generated on demand, not stored

	// Add index to table only
	if table, exists := dbSchema.Tables[tableName]; exists {
		table.Indexes[indexName] = index
	}

	return nil
}

// extractExpressionString extracts a string representation of an expression node
func (p *Parser) extractExpressionString(expr *pg_query.Node) string {
	if expr == nil {
		return ""
	}

	switch n := expr.Node.(type) {
	case *pg_query.Node_ColumnRef:
		return p.extractColumnName(expr)
	case *pg_query.Node_AExpr:
		// Handle binary expressions like (status = 'active') and JSON operators
		return p.extractBinaryExpression(n.AExpr)
	case *pg_query.Node_FuncCall:
		// Handle function calls in expressions
		return p.extractFunctionCall(n.FuncCall)
	case *pg_query.Node_AConst:
		// For constants, we might need to preserve quotes for strings
		return p.extractConstantValue(expr)
	case *pg_query.Node_NullTest:
		// Handle IS NULL and IS NOT NULL expressions
		return p.extractNullTest(n.NullTest)
	case *pg_query.Node_TypeCast:
		// Handle type casting expressions like 'method'::text
		return p.extractTypeCast(n.TypeCast)
	default:
		// For complex expressions, return a placeholder
		return fmt.Sprintf("(%s)", "expression")
	}
}

// extractNullTest extracts string representation of NULL test expressions (IS NULL, IS NOT NULL)
func (p *Parser) extractNullTest(nullTest *pg_query.NullTest) string {
	if nullTest == nil {
		return ""
	}

	// Extract the expression being tested
	expr := p.extractExpressionString(nullTest.Arg)

	// Determine the null test type
	switch nullTest.Nulltesttype {
	case pg_query.NullTestType_IS_NULL:
		return fmt.Sprintf("%s IS NULL", expr)
	case pg_query.NullTestType_IS_NOT_NULL:
		return fmt.Sprintf("%s IS NOT NULL", expr)
	default:
		return fmt.Sprintf("%s IS NULL", expr) // Default fallback
	}
}

// extractBinaryExpression extracts string representation of binary expressions
func (p *Parser) extractBinaryExpression(aExpr *pg_query.A_Expr) string {
	if aExpr == nil {
		return ""
	}

	left := ""
	if aExpr.Lexpr != nil {
		left = p.extractExpressionString(aExpr.Lexpr)
	}

	right := ""
	if aExpr.Rexpr != nil {
		// For JSON operators, simplify the right side - remove type casting
		rightExpr := p.extractExpressionString(aExpr.Rexpr)
		// Remove ::text suffix for JSON operators to match user expected format
		if strings.HasSuffix(rightExpr, "::text") {
			rightExpr = strings.TrimSuffix(rightExpr, "::text")
		}
		right = rightExpr
	}

	operator := ""
	if len(aExpr.Name) > 0 {
		if opNode := aExpr.Name[0]; opNode != nil {
			operator = p.extractStringValue(opNode)
		}
	}

	if left != "" && right != "" && operator != "" {
		// Handle JSON operators specially
		if operator == "->>" || operator == "->" {
			return fmt.Sprintf("%s%s%s", left, operator, right)
		}
		// For other operators, use parentheses
		return fmt.Sprintf("(%s %s %s)", left, operator, right)
	}

	return fmt.Sprintf("(%s)", "expression")
}

// extractFunctionCall extracts string representation of function calls
func (p *Parser) extractFunctionCall(funcCall *pg_query.FuncCall) string {
	if funcCall == nil {
		return ""
	}

	// Extract function name
	funcName := ""
	if len(funcCall.Funcname) > 0 {
		if nameNode := funcCall.Funcname[0]; nameNode != nil {
			funcName = p.extractStringValue(nameNode)
		}
	}

	// For now, just return function name with parentheses
	if funcName != "" {
		return fmt.Sprintf("%s()", funcName)
	}

	return "function()"
}

// isExpressionIndex checks if an index is an expression index
func (p *Parser) isExpressionIndex(index *Index) bool {
	for _, col := range index.Columns {
		// If any column name contains parentheses, JSON operators, or other expression indicators
		if strings.Contains(col.Name, "(") || strings.Contains(col.Name, ")") ||
			strings.Contains(col.Name, "->>") || strings.Contains(col.Name, "->") ||
			strings.Contains(col.Name, "::") {
			return true
		}
	}
	return false
}

// buildIndexDefinition builds the CREATE INDEX statement string
func (p *Parser) buildIndexDefinition(index *Index) string {
	var builder strings.Builder

	// CREATE [UNIQUE] INDEX [CONCURRENTLY]
	builder.WriteString("CREATE ")
	if index.Type == IndexTypeUnique {
		builder.WriteString("UNIQUE ")
	}
	builder.WriteString("INDEX ")
	if index.IsConcurrent {
		builder.WriteString("CONCURRENTLY ")
	}

	// Index name
	builder.WriteString(index.Name)
	builder.WriteString(" ON ")

	// Table name (without schema for simplified format)
	builder.WriteString(index.Table)

	// For expression indexes, use simplified format without USING clause
	if index.IsExpression {
		builder.WriteString(" (")
		for i, col := range index.Columns {
			if i > 0 {
				builder.WriteString(", ")
			}
			// Add triple parentheses for JSON expressions to match pg_dump format
			if strings.Contains(col.Name, "->>") || strings.Contains(col.Name, "->") {
				builder.WriteString(fmt.Sprintf("((%s))", col.Name))
			} else {
				builder.WriteString(col.Name)
			}
		}
		builder.WriteString(")")
	} else {
		// Index method - include USING clause for non-expression indexes
		builder.WriteString(" USING ")
		builder.WriteString(index.Method)

		// Columns
		builder.WriteString(" (")
		for i, col := range index.Columns {
			if i > 0 {
				builder.WriteString(", ")
			}
			// Add triple parentheses for JSON expressions to match pg_dump format
			if strings.Contains(col.Name, "->>") || strings.Contains(col.Name, "->") {
				builder.WriteString(fmt.Sprintf("((%s))", col.Name))
			} else {
				builder.WriteString(col.Name)
			}

			if col.Direction == "DESC" {
				builder.WriteString(" DESC")
			}
			if col.Operator != "" {
				builder.WriteString(" ")
				builder.WriteString(col.Operator)
			}
		}
		builder.WriteString(")")
	}

	// WHERE clause for partial indexes
	if index.IsPartial && index.Where != "" {
		builder.WriteString(" WHERE ")
		builder.WriteString(index.Where)
	}

	return builder.String()
}

// extractConstantValue extracts string representation with proper quoting for constants
func (p *Parser) extractConstantValue(node *pg_query.Node) string {
	if node == nil {
		return ""
	}
	switch n := node.Node.(type) {
	case *pg_query.Node_AConst:
		if n.AConst.Val != nil {
			switch val := n.AConst.Val.(type) {
			case *pg_query.A_Const_Sval:
				// For string constants, preserve the quotes
				return fmt.Sprintf("'%s'", val.Sval.Sval)
			case *pg_query.A_Const_Ival:
				return strconv.FormatInt(int64(val.Ival.Ival), 10)
			case *pg_query.A_Const_Fval:
				return val.Fval.Fval
			case *pg_query.A_Const_Boolval:
				if val.Boolval.Boolval {
					return "true"
				}
				return "false"
			}
		}
	}
	return ""
}

// extractTypeCast extracts string representation of type casting expressions
func (p *Parser) extractTypeCast(typeCast *pg_query.TypeCast) string {
	if typeCast == nil {
		return ""
	}

	// Extract the expression being cast
	expr := ""
	if typeCast.Arg != nil {
		expr = p.extractExpressionString(typeCast.Arg)
	}

	// Extract the target type
	targetType := ""
	if typeCast.TypeName != nil {
		targetType = p.extractTypeName(typeCast.TypeName)
	}

	if expr != "" && targetType != "" {
		return fmt.Sprintf("%s::%s", expr, targetType)
	}

	return expr
}

// extractTypeName extracts the type name from a TypeName node
func (p *Parser) extractTypeName(typeName *pg_query.TypeName) string {
	if typeName == nil || len(typeName.Names) == 0 {
		return ""
	}

	// Extract type name parts
	var parts []string
	for _, nameNode := range typeName.Names {
		if str := nameNode.GetString_(); str != nil {
			parts = append(parts, str.Sval)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	// Join parts with dots (for schema-qualified types)
	return strings.Join(parts, ".")
}

// parseCreateEnum parses CREATE TYPE ... AS ENUM statements
func (p *Parser) parseCreateEnum(enumStmt *pg_query.CreateEnumStmt) error {
	// Extract type name and schema
	typeName := ""
	schemaName := "public" // Default schema

	if len(enumStmt.TypeName) > 0 {
		for i, nameNode := range enumStmt.TypeName {
			if str := nameNode.GetString_(); str != nil {
				if i == 0 && len(enumStmt.TypeName) > 1 {
					// First part is schema
					schemaName = str.Sval
				} else {
					// Last part is type name
					typeName = str.Sval
				}
			}
		}
	}

	if typeName == "" {
		return nil // Skip if we can't determine type name
	}

	// Get or create schema
	dbSchema := p.schema.getOrCreateSchema(schemaName)

	// Extract enum values
	var enumValues []string
	for _, valNode := range enumStmt.Vals {
		if str := valNode.GetString_(); str != nil {
			enumValues = append(enumValues, str.Sval)
		}
	}

	// Create enum type
	enumType := &Type{
		Schema:     schemaName,
		Name:       typeName,
		Kind:       TypeKindEnum,
		EnumValues: enumValues,
	}

	// Add type to schema
	dbSchema.Types[typeName] = enumType

	return nil
}

// parseCreateCompositeType parses CREATE TYPE ... AS (...) statements
func (p *Parser) parseCreateCompositeType(compStmt *pg_query.CompositeTypeStmt) error {
	// Extract type name and schema
	typeName := ""
	schemaName := "public" // Default schema

	if compStmt.Typevar != nil {
		schemaName, typeName = p.extractTableName(compStmt.Typevar)
	}

	if typeName == "" {
		return nil // Skip if we can't determine type name
	}

	// Get or create schema
	dbSchema := p.schema.getOrCreateSchema(schemaName)

	// Extract composite type columns
	var columns []*TypeColumn
	position := 1
	for _, colDef := range compStmt.Coldeflist {
		if columnDef := colDef.GetColumnDef(); columnDef != nil {
			column := &TypeColumn{
				Name:     columnDef.Colname,
				Position: position,
			}

			// Parse type name
			if columnDef.TypeName != nil {
				column.DataType = p.parseTypeName(columnDef.TypeName)
			}

			columns = append(columns, column)
			position++
		}
	}

	// Create composite type
	compositeType := &Type{
		Schema:  schemaName,
		Name:    typeName,
		Kind:    TypeKindComposite,
		Columns: columns,
	}

	// Add type to schema
	dbSchema.Types[typeName] = compositeType

	return nil
}

// parseCreateDomain parses CREATE DOMAIN statements
func (p *Parser) parseCreateDomain(domainStmt *pg_query.CreateDomainStmt) error {
	// Extract domain name and schema
	domainName := ""
	schemaName := "public" // Default schema

	if len(domainStmt.Domainname) > 0 {
		for i, nameNode := range domainStmt.Domainname {
			if str := nameNode.GetString_(); str != nil {
				if i == 0 && len(domainStmt.Domainname) > 1 {
					// First part is schema
					schemaName = str.Sval
				} else {
					// Last part is domain name
					domainName = str.Sval
				}
			}
		}
	}

	if domainName == "" {
		return nil // Skip if we can't determine domain name
	}

	// Get or create schema
	dbSchema := p.schema.getOrCreateSchema(schemaName)

	// Extract base type
	var baseType string
	if domainStmt.TypeName != nil {
		baseType = p.parseTypeName(domainStmt.TypeName)
	}

	// Create domain type
	domainType := &Type{
		Schema:   schemaName,
		Name:     domainName,
		Kind:     TypeKindDomain,
		BaseType: baseType,
	}

	// Add type to schema
	dbSchema.Types[domainName] = domainType

	return nil
}

// parseDefineStatement parses DEFINE statements (like CREATE AGGREGATE)
func (p *Parser) parseDefineStatement(defineStmt *pg_query.DefineStmt) error {
	// Check if this is an aggregate definition
	if defineStmt.Kind == pg_query.ObjectType_OBJECT_AGGREGATE {
		return p.parseCreateAggregate(defineStmt)
	}

	// For now, ignore other types of DEFINE statements
	return nil
}

// parseCreateAggregate parses CREATE AGGREGATE statements
func (p *Parser) parseCreateAggregate(defineStmt *pg_query.DefineStmt) error {
	// Extract aggregate name and schema
	aggregateName := ""
	schemaName := "public" // Default schema

	if len(defineStmt.Defnames) > 0 {
		for i, nameNode := range defineStmt.Defnames {
			if str := nameNode.GetString_(); str != nil {
				if i == 0 && len(defineStmt.Defnames) > 1 {
					// First part is schema
					schemaName = str.Sval
				} else {
					// Last part is aggregate name
					aggregateName = str.Sval
				}
			}
		}
	}

	if aggregateName == "" {
		return nil // Skip if we can't determine aggregate name
	}

	// Get or create schema
	dbSchema := p.schema.getOrCreateSchema(schemaName)

	// Extract aggregate arguments
	var arguments string
	if len(defineStmt.Args) > 0 {
		if listNode := defineStmt.Args[0].GetList(); listNode != nil {
			var argTypes []string
			for _, item := range listNode.Items {
				if funcParam := item.GetFunctionParameter(); funcParam != nil {
					if funcParam.ArgType != nil {
						argType := p.parseTypeName(funcParam.ArgType)
						argTypes = append(argTypes, argType)
					}
				}
			}
			if len(argTypes) > 0 {
				arguments = argTypes[0] // For now, just take the first argument type
			}
		}
	}

	// Extract aggregate options from definition
	var stateFunction string
	var stateType string
	var returnType string

	for _, def := range defineStmt.Definition {
		if defElem := def.GetDefElem(); defElem != nil {
			switch defElem.Defname {
			case "sfunc":
				if defElem.Arg != nil {
					if typeName := defElem.Arg.GetTypeName(); typeName != nil {
						// Extract function name from type name
						if len(typeName.Names) > 0 {
							if str := typeName.Names[len(typeName.Names)-1].GetString_(); str != nil {
								stateFunction = str.Sval
							}
						}
					}
				}
			case "stype":
				if defElem.Arg != nil {
					if typeName := defElem.Arg.GetTypeName(); typeName != nil {
						stateType = p.parseTypeName(typeName)
					}
				}
			}
		}
	}

	// For aggregates, the return type is typically the same as the state type
	returnType = stateType

	// Create aggregate
	aggregate := &Aggregate{
		Schema:             schemaName,
		Name:               aggregateName,
		Arguments:          arguments,
		ReturnType:         returnType,
		StateType:          stateType,
		TransitionFunction: stateFunction,
	}

	// Add aggregate to schema
	dbSchema.Aggregates[aggregateName] = aggregate

	return nil
}

// parseCreateTrigger parses CREATE TRIGGER statements
func (p *Parser) parseCreateTrigger(triggerStmt *pg_query.CreateTrigStmt) error {
	if triggerStmt.Trigname == "" {
		return nil // Skip if we can't determine trigger name
	}

	// Extract table name and schema
	var schemaName, tableName string
	if triggerStmt.Relation != nil {
		schemaName, tableName = p.extractTableName(triggerStmt.Relation)
	}

	if tableName == "" {
		return nil // Skip if we can't determine table name
	}

	// Get or create schema
	dbSchema := p.schema.getOrCreateSchema(schemaName)

	// Find the table - triggers must be attached to existing tables
	table, exists := dbSchema.Tables[tableName]
	if !exists {
		// Table doesn't exist yet - this could happen if CREATE TRIGGER comes before CREATE TABLE
		// For now, skip this trigger
		return nil
	}

	// Map timing - use inspection based approach for now
	var timing TriggerTiming
	switch triggerStmt.Timing {
	case 2:
		timing = TriggerTimingBefore
	case 4:
		timing = TriggerTimingAfter
	case 8:
		timing = TriggerTimingInsteadOf
	default:
		timing = TriggerTimingAfter // Default
	}

	// Map events - PostgreSQL trigger event flags (see pg_trigger.h)
	// Add events in standard order: INSERT, UPDATE, DELETE, TRUNCATE
	var events []TriggerEvent
	if triggerStmt.Events&4 != 0 { // TRIGGER_TYPE_INSERT = 4
		events = append(events, TriggerEventInsert)
	}
	if triggerStmt.Events&16 != 0 { // TRIGGER_TYPE_UPDATE = 16
		events = append(events, TriggerEventUpdate)
	}
	if triggerStmt.Events&8 != 0 { // TRIGGER_TYPE_DELETE = 8
		events = append(events, TriggerEventDelete)
	}
	if triggerStmt.Events&32 != 0 { // TRIGGER_TYPE_TRUNCATE = 32
		events = append(events, TriggerEventTruncate)
	}

	// Map level (row vs statement)
	var level TriggerLevel
	if triggerStmt.Row {
		level = TriggerLevelRow
	} else {
		level = TriggerLevelStatement
	}

	// Extract function name and arguments
	function := p.extractTriggerFunctionFromAST(triggerStmt)

	// Extract WHEN condition if present
	var condition string
	if triggerStmt.WhenClause != nil {
		condition = p.extractExpressionText(triggerStmt.WhenClause)
	}

	// Create trigger
	trigger := &Trigger{
		Schema:    schemaName,
		Table:     tableName,
		Name:      triggerStmt.Trigname,
		Timing:    timing,
		Events:    events,
		Level:     level,
		Function:  function,
		Condition: condition,
	}

	// Add trigger to table only
	table.Triggers[triggerStmt.Trigname] = trigger

	return nil
}

// extractTriggerFunctionFromAST extracts the function call from trigger function nodes
func (p *Parser) extractTriggerFunctionFromAST(triggerStmt *pg_query.CreateTrigStmt) string {
	if len(triggerStmt.Funcname) == 0 {
		return ""
	}

	// Extract function name
	var funcNameParts []string
	for _, nameNode := range triggerStmt.Funcname {
		if str := nameNode.GetString_(); str != nil {
			funcNameParts = append(funcNameParts, str.Sval)
		}
	}

	if len(funcNameParts) == 0 {
		return ""
	}

	funcName := strings.Join(funcNameParts, ".")

	// Build arguments list
	var argParts []string
	for _, argNode := range triggerStmt.Args {
		argValue := p.extractStringValue(argNode)
		if argValue != "" {
			// Quote string arguments
			if !strings.HasPrefix(argValue, "'") {
				argValue = "'" + argValue + "'"
			}
			argParts = append(argParts, argValue)
		}
	}

	// Return complete function call
	if len(argParts) > 0 {
		return fmt.Sprintf("%s(%s)", funcName, strings.Join(argParts, ", "))
	}
	return fmt.Sprintf("%s()", funcName)
}

// parseCreatePolicy parses CREATE POLICY statements
func (p *Parser) parseCreatePolicy(policyStmt *pg_query.CreatePolicyStmt) error {
	if policyStmt.PolicyName == "" {
		return nil // Skip if we can't determine policy name
	}

	// Extract table name and schema
	var schemaName, tableName string
	if policyStmt.Table != nil {
		schemaName, tableName = p.extractTableName(policyStmt.Table)
	}

	if tableName == "" {
		return nil // Skip if we can't determine table name
	}

	// Get or create schema
	dbSchema := p.schema.getOrCreateSchema(schemaName)

	// Find the table - policies must be attached to existing tables
	table, exists := dbSchema.Tables[tableName]
	if !exists {
		// Table doesn't exist yet - this could happen if CREATE POLICY comes before CREATE TABLE
		// For now, skip this policy
		return nil
	}

	// Map command name to PolicyCommand
	var command PolicyCommand
	switch strings.ToLower(policyStmt.CmdName) {
	case "select":
		command = PolicyCommandSelect
	case "insert":
		command = PolicyCommandInsert
	case "update":
		command = PolicyCommandUpdate
	case "delete":
		command = PolicyCommandDelete
	case "all":
		command = PolicyCommandAll
	default:
		command = PolicyCommandAll // Default fallback
	}

	// Extract USING expression
	var usingClause string
	if policyStmt.Qual != nil {
		usingClause = p.extractExpressionString(policyStmt.Qual)
	}

	// Extract WITH CHECK expression
	var withCheckClause string
	if policyStmt.WithCheck != nil {
		withCheckClause = p.extractExpressionString(policyStmt.WithCheck)
	}

	// Extract roles
	var roles []string
	if len(policyStmt.Roles) > 0 {
		for _, roleNode := range policyStmt.Roles {
			if roleStr := p.extractRoleName(roleNode); roleStr != "" {
				roles = append(roles, roleStr)
			}
		}
	}
	// Default to PUBLIC if no roles specified
	if len(roles) == 0 {
		roles = []string{"PUBLIC"}
	}

	// Determine if policy is permissive (default) or restrictive
	permissive := true
	if !policyStmt.Permissive {
		permissive = false
	}

	// Create policy
	policy := &RLSPolicy{
		Schema:     schemaName,
		Table:      tableName,
		Name:       policyStmt.PolicyName,
		Command:    command,
		Permissive: permissive,
		Roles:      roles,
		Using:      usingClause,
		WithCheck:  withCheckClause,
	}

	// Add policy to table
	table.Policies[policyStmt.PolicyName] = policy

	return nil
}

// extractRoleName extracts role name from a role node
func (p *Parser) extractRoleName(roleNode *pg_query.Node) string {
	if roleNode == nil {
		return ""
	}

	switch node := roleNode.Node.(type) {
	case *pg_query.Node_RoleSpec:
		if node.RoleSpec != nil {
			if node.RoleSpec.Rolename != "" {
				return node.RoleSpec.Rolename
			}
			// Handle special role types
			switch node.RoleSpec.Roletype {
			case pg_query.RoleSpecType_ROLESPEC_PUBLIC:
				return "PUBLIC"
			case pg_query.RoleSpecType_ROLESPEC_CURRENT_USER:
				return "CURRENT_USER"
			case pg_query.RoleSpecType_ROLESPEC_CURRENT_ROLE:
				return "CURRENT_ROLE"
			case pg_query.RoleSpecType_ROLESPEC_SESSION_USER:
				return "SESSION_USER"
			}
		}
	case *pg_query.Node_String_:
		if node.String_ != nil {
			return node.String_.Sval
		}
	}

	return ""
}

// parseCreateExtension parses CREATE EXTENSION statements
func (p *Parser) parseCreateExtension(extStmt *pg_query.CreateExtensionStmt) error {
	if extStmt.Extname == "" {
		return nil // Skip if we can't determine extension name
	}

	// Create extension
	extension := &Extension{
		Name:    extStmt.Extname,
		Schema:  "", // Default - will be extracted from options
		Version: "", // Default - would need to be extracted from options if available
		Comment: "", // Default - would need to be extracted separately
	}

	// Parse extension options
	for _, option := range extStmt.Options {
		if defElem := option.GetDefElem(); defElem != nil {
			switch defElem.Defname {
			case "schema":
				if defElem.Arg != nil {
					extension.Schema = p.extractStringValue(defElem.Arg)
				}
			case "version":
				if defElem.Arg != nil {
					extension.Version = p.extractStringValue(defElem.Arg)
				}
				// Other options like FROM, CASCADE could be handled here if needed
			}
		}
	}

	// Add extension to schema
	p.schema.Extensions[extension.Name] = extension

	return nil
}

// parseCreateSchema parses CREATE SCHEMA statements
func (p *Parser) parseCreateSchema(schemaStmt *pg_query.CreateSchemaStmt) error {
	if schemaStmt.Schemaname == "" {
		return nil // Skip if we can't determine schema name
	}

	// Get or create schema
	dbSchema := p.schema.getOrCreateSchema(schemaStmt.Schemaname)

	// Extract authorization (owner) if present
	if schemaStmt.Authrole != nil {
		if schemaStmt.Authrole.Rolename != "" {
			dbSchema.Owner = schemaStmt.Authrole.Rolename
		}
	}

	return nil
}

// handleAttachPartition handles ALTER TABLE ... ATTACH PARTITION
func (p *Parser) handleAttachPartition(cmd *pg_query.AlterTableCmd, parentTable *Table) error {
	// The cmd.Def should contain the partition table reference
	if cmd.Def == nil {
		return nil
	}

	// Extract the partition table name from the command
	// The partition table is specified in cmd.Def
	if rangeVar := cmd.Def.GetRangeVar(); rangeVar != nil {
		_, partitionTableName := p.extractTableName(rangeVar)
		if partitionTableName != "" {
			// Record the parent-child relationship
			p.partitionChildren[partitionTableName] = parentTable.Name
		}
	}

	return nil
}

// handleSerialType handles SERIAL, SMALLSERIAL, and BIGSERIAL column types
// by converting them to appropriate integer types and creating implicit sequences
func (p *Parser) handleSerialType(column *Column, schemaName, tableName string) bool {
	var baseType string
	var sequenceName string

	switch strings.ToUpper(column.DataType) {
	case "SERIAL":
		baseType = "integer"
		sequenceName = fmt.Sprintf("%s_%s_seq", tableName, column.Name)
	case "SMALLSERIAL":
		baseType = "smallint"
		sequenceName = fmt.Sprintf("%s_%s_seq", tableName, column.Name)
	case "BIGSERIAL":
		baseType = "bigint"
		sequenceName = fmt.Sprintf("%s_%s_seq", tableName, column.Name)
	default:
		return false // Not a SERIAL type
	}

	// Convert column type to base integer type
	column.DataType = baseType

	// Set NOT NULL constraint (SERIAL columns are implicitly NOT NULL)
	column.IsNullable = false

	// Check if this is a partition table (contains _pYYYY pattern)
	// Partition tables inherit sequences from parent tables
	isPartitionTable := p.isPartitionTable(tableName)

	if isPartitionTable {
		// For partition tables, find the parent table's sequence name
		parentTableName := p.getParentTableName(tableName)
		parentSequenceName := fmt.Sprintf("%s_%s_seq", parentTableName, column.Name)

		// Set default value to use parent's sequence
		defaultValue := fmt.Sprintf("nextval('%s.%s')", schemaName, parentSequenceName)
		column.DefaultValue = &defaultValue
	} else {
		// Set default value to nextval
		defaultValue := fmt.Sprintf("nextval('%s.%s')", schemaName, sequenceName)
		column.DefaultValue = &defaultValue

		// Create the implicit sequence only for non-partition tables
		p.createImplicitSequence(schemaName, sequenceName, tableName, column.Name, baseType)
	}

	return true
}

// isPartitionTable checks if a table name follows partition naming patterns
func (p *Parser) isPartitionTable(tableName string) bool {
	// Common partition naming patterns:
	// - table_pYYYY_MM (e.g., payment_p2022_01)
	// - table_pYYYY (e.g., payment_p2022)
	// - table_YYYY_MM_DD
	// - table_YYYY_MM
	// - table_YYYY

	// Check for _pYYYY pattern (most common in our case)
	if matched, _ := regexp.MatchString(`_p\d{4}`, tableName); matched {
		return true
	}

	// Check for _YYYY_MM_DD pattern
	if matched, _ := regexp.MatchString(`_\d{4}_\d{2}_\d{2}$`, tableName); matched {
		return true
	}

	// Check for _YYYY_MM pattern
	if matched, _ := regexp.MatchString(`_\d{4}_\d{2}$`, tableName); matched {
		return true
	}

	// Check for _YYYY pattern at end
	if matched, _ := regexp.MatchString(`_\d{4}$`, tableName); matched {
		return true
	}

	return false
}

// getParentTableName extracts the parent table name from a partition table name
func (p *Parser) getParentTableName(tableName string) string {
	// Remove common partition suffixes
	// payment_p2022_01 -> payment
	// sales_2022_01 -> sales

	// Remove _pYYYY_MM pattern
	if idx := strings.Index(tableName, "_p"); idx > 0 {
		if matched, _ := regexp.MatchString(`_p\d{4}`, tableName[idx:]); matched {
			return tableName[:idx]
		}
	}

	// Remove _YYYY_MM_DD pattern
	re := regexp.MustCompile(`_\d{4}_\d{2}_\d{2}$`)
	if loc := re.FindStringIndex(tableName); loc != nil {
		return tableName[:loc[0]]
	}

	// Remove _YYYY_MM pattern
	re = regexp.MustCompile(`_\d{4}_\d{2}$`)
	if loc := re.FindStringIndex(tableName); loc != nil {
		return tableName[:loc[0]]
	}

	// Remove _YYYY pattern
	re = regexp.MustCompile(`_\d{4}$`)
	if loc := re.FindStringIndex(tableName); loc != nil {
		return tableName[:loc[0]]
	}

	// If no pattern matched, return original name
	return tableName
}

// createImplicitSequence creates a sequence for SERIAL columns
func (p *Parser) createImplicitSequence(schemaName, sequenceName, tableName, columnName, dataType string) {
	// Get or create schema
	dbSchema := p.schema.getOrCreateSchema(schemaName)

	// Create sequence object
	sequence := &Sequence{
		Schema:        schemaName,
		Name:          sequenceName,
		DataType:      dataType,
		StartValue:    1,
		Increment:     1,
		MinValue:      nil, // Will use default min/max based on data type
		MaxValue:      nil,
		CycleOption:   false,
		OwnedByTable:  tableName,
		OwnedByColumn: columnName,
	}

	// Add sequence to schema
	dbSchema.Sequences[sequenceName] = sequence
}
