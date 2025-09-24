package ir

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/pgschema/pgschema/ir/queries"
	"golang.org/x/sync/errgroup"
)

// Inspector builds IR from database queries
type Inspector struct {
	db           *sql.DB
	queries      *queries.Queries
	ignoreConfig *IgnoreConfig
}

// NewInspector creates a new schema inspector with optional ignore configuration
func NewInspector(db *sql.DB, ignoreConfig *IgnoreConfig) *Inspector {
	return &Inspector{
		db:           db,
		queries:      queries.New(db),
		ignoreConfig: ignoreConfig,
	}
}

// queryGroup represents a group of queries that can be executed concurrently
// BuildIR builds the schema IR from the database for a specific schema
func (i *Inspector) BuildIR(ctx context.Context, targetSchema string) (*IR, error) {
	schema := NewIR()

	// Sequential prerequisites
	if err := i.buildMetadata(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build metadata: %w", err)
	}

	if err := i.validateSchemaExists(ctx, targetSchema); err != nil {
		return nil, err
	}

	if err := i.buildSchemas(ctx, schema, targetSchema); err != nil {
		return nil, fmt.Errorf("failed to build schemas: %w", err)
	}

	if err := i.buildTables(ctx, schema, targetSchema); err != nil {
		return nil, fmt.Errorf("failed to build tables: %w", err)
	}

	// Concurrent Group 1: Table Details
	group1 := queryGroup{
		name: "table details",
		funcs: []func(context.Context, *IR, string) error{
			i.buildColumns,
			i.buildConstraints,
			i.buildIndexes,
			i.buildPartitions,
		},
	}

	// Concurrent Group 2: Independent Objects
	group2 := queryGroup{
		name: "independent objects",
		funcs: []func(context.Context, *IR, string) error{
			i.buildSequences,
			i.buildFunctions,
			i.buildProcedures,
			i.buildAggregates,
			i.buildTypes,
		},
	}

	// Concurrent Group 3: Table-Dependent Objects
	group3 := queryGroup{
		name: "table-dependent objects",
		funcs: []func(context.Context, *IR, string) error{
			i.buildViews,
			i.buildTriggers,
			i.buildRLSPolicies,
		},
	}

	// Execute groups concurrently where possible
	var eg errgroup.Group

	// Group 1 & 2 can run in parallel
	eg.Go(func() error {
		return i.executeConcurrentGroup(ctx, schema, targetSchema, group1)
	})

	eg.Go(func() error {
		return i.executeConcurrentGroup(ctx, schema, targetSchema, group2)
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	// Group 3 runs after table details are loaded
	if err := i.executeConcurrentGroup(ctx, schema, targetSchema, group3); err != nil {
		return nil, err
	}

	// Normalize the IR
	normalizeIR(schema)

	return schema, nil
}

type queryGroup struct {
	name  string
	funcs []func(context.Context, *IR, string) error
}

// executeConcurrentGroup executes a group of functions concurrently
func (i *Inspector) executeConcurrentGroup(ctx context.Context, schema *IR, targetSchema string, group queryGroup) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(group.funcs))

	for _, fn := range group.funcs {
		wg.Add(1)
		go func(f func(context.Context, *IR, string) error) {
			defer wg.Done()
			if err := f(ctx, schema, targetSchema); err != nil {
				errChan <- err
			}
		}(fn)
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	for err := range errChan {
		if err != nil {
			return fmt.Errorf("%s: %w", group.name, err)
		}
	}
	return nil
}

func (i *Inspector) buildMetadata(ctx context.Context, schema *IR) error {
	var dbVersion string
	if err := i.db.QueryRowContext(ctx, "SELECT version()").Scan(&dbVersion); err != nil {
		return err
	}

	// Extract version number from the version string
	if strings.Contains(dbVersion, "PostgreSQL") {
		parts := strings.Fields(dbVersion)
		if len(parts) >= 2 {
			dbVersion = "PostgreSQL " + parts[1]
		}
	}

	schema.Metadata = Metadata{
		DatabaseVersion: dbVersion,
	}

	return nil
}

func (i *Inspector) buildSchemas(ctx context.Context, schema *IR, targetSchema string) error {
	// Use the schema-specific query to prefilter at the database level
	schemaName, err := i.queries.GetSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	name := fmt.Sprintf("%s", schemaName)
	schema.getOrCreateSchema(name)

	return nil
}

func (i *Inspector) buildTables(ctx context.Context, schema *IR, targetSchema string) error {
	tables, err := i.queries.GetTablesForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	for _, table := range tables {
		schemaName := fmt.Sprintf("%s", table.TableSchema)
		tableName := fmt.Sprintf("%s", table.TableName)
		tableType := fmt.Sprintf("%s", table.TableType)
		comment := ""
		if table.TableComment.Valid {
			comment = table.TableComment.String
		}

		// No need to filter by schema since query is already schema-specific

		// Check if table should be ignored
		if i.ignoreConfig != nil && i.ignoreConfig.ShouldIgnoreTable(tableName) {
			continue
		}

		dbSchema := schema.getOrCreateSchema(schemaName)

		// Skip views as they are handled by buildViews function
		if tableType == "VIEW" {
			continue
		}

		var tType TableType
		switch tableType {
		case "BASE TABLE":
			tType = TableTypeBase
		default:
			tType = TableTypeBase
		}

		t := &Table{
			Schema:      schemaName,
			Name:        tableName,
			Type:        tType,
			Comment:     comment,
			Columns:     []*Column{},
			Constraints: make(map[string]*Constraint),
			Indexes:     make(map[string]*Index),
			Triggers:    make(map[string]*Trigger),
			Policies:    make(map[string]*RLSPolicy),
		}

		dbSchema.SetTable(tableName, t)
	}

	return nil
}

func (i *Inspector) buildColumns(ctx context.Context, schema *IR, targetSchema string) error {
	columns, err := i.queries.GetColumnsForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	for _, col := range columns {
		schemaName := fmt.Sprintf("%s", col.TableSchema)
		tableName := fmt.Sprintf("%s", col.TableName)
		columnName := fmt.Sprintf("%s", col.ColumnName)
		comment := ""
		if col.ColumnComment.Valid {
			comment = col.ColumnComment.String
		}

		// No need to filter by schema since query is already schema-specific

		dbSchema := schema.getOrCreateSchema(schemaName)
		table, exists := dbSchema.Tables[tableName]
		if !exists {
			continue // Skip columns for non-existent tables
		}

		// Get the resolved type - schema prefix and type normalization is now handled during read time
		resolvedType := i.safeInterfaceToString(col.ResolvedType)
		// Map internal PostgreSQL types to standard SQL types
		dataType := normalizePostgreSQLType(resolvedType)

		column := &Column{
			Name:       columnName,
			Position:   i.safeInterfaceToInt(col.OrdinalPosition, 0),
			DataType:   dataType,
			IsNullable: fmt.Sprintf("%s", col.IsNullable) == "YES",
			Comment:    comment,
		}

		// Handle default value - keep original value as stored in database
		if defaultVal := i.safeInterfaceToString(col.ColumnDefault); defaultVal != "" && defaultVal != "<nil>" {
			column.DefaultValue = &defaultVal
		}

		// Handle max length
		if maxLen := i.safeInterfaceToInt64(col.CharacterMaximumLength, -1); maxLen > 0 {
			maxLenInt := int(maxLen)
			column.MaxLength = &maxLenInt
		}

		// Handle numeric precision and scale
		if precision := i.safeInterfaceToInt64(col.NumericPrecision, -1); precision > 0 {
			precisionInt := int(precision)
			column.Precision = &precisionInt
		}

		if scale := i.safeInterfaceToInt64(col.NumericScale, -1); scale >= 0 {
			scaleInt := int(scale)
			column.Scale = &scaleInt
		}

		// Handle identity columns
		if fmt.Sprintf("%s", col.IsIdentity) == "YES" {
			identity := &Identity{
				Generation: i.safeInterfaceToString(col.IdentityGeneration),
				Cycle:      fmt.Sprintf("%s", col.IdentityCycle) == "YES",
			}

			if start := i.safeInterfaceToInt64(col.IdentityStart, -1); start >= 0 {
				identity.Start = &start
			}

			if increment := i.safeInterfaceToInt64(col.IdentityIncrement, -1); increment >= 0 {
				identity.Increment = &increment
			}

			if maximum := i.safeInterfaceToInt64(col.IdentityMaximum, -1); maximum >= 0 {
				identity.Maximum = &maximum
			}

			if minimum := i.safeInterfaceToInt64(col.IdentityMinimum, -1); minimum >= 0 {
				identity.Minimum = &minimum
			}

			column.Identity = identity
		}

		// Check if column already exists to avoid duplicates
		columnExists := false
		for _, existingCol := range table.Columns {
			if existingCol.Name == columnName {
				columnExists = true
				break
			}
		}

		// Only add column if it doesn't already exist
		if !columnExists {
			table.Columns = append(table.Columns, column)
		}
	}

	return nil
}

func (i *Inspector) buildPartitions(ctx context.Context, schema *IR, targetSchema string) error {
	// Use the schema-specific query to prefilter at the database level
	partitions, err := i.queries.GetPartitionedTablesForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	for _, partition := range partitions {
		schemaName := partition.TableSchema
		tableName := partition.TableName
		partitionStrategy := ""
		if partition.PartitionStrategy.Valid {
			partitionStrategy = partition.PartitionStrategy.String
		}
		partitionKey := ""
		if partition.PartitionKey.Valid {
			partitionKey = partition.PartitionKey.String
		}

		// No need to filter by schema since query is already schema-specific

		dbSchema := schema.getOrCreateSchema(schemaName)
		table, exists := dbSchema.Tables[tableName]
		if !exists {
			continue // Skip partitions for non-existent tables
		}

		table.IsPartitioned = true
		table.PartitionStrategy = partitionStrategy
		table.PartitionKey = partitionKey
	}

	return nil
}


func (i *Inspector) buildConstraints(ctx context.Context, schema *IR, targetSchema string) error {
	constraints, err := i.queries.GetConstraintsForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	// Group constraints by key to handle composite constraints
	type constraintKey struct {
		schema string
		table  string
		name   string
	}
	constraintGroups := make(map[constraintKey]*Constraint)

	for _, constraint := range constraints {
		schemaName := constraint.TableSchema
		tableName := constraint.TableName
		constraintName := constraint.ConstraintName
		constraintType := ""
		if constraint.ConstraintType.Valid {
			constraintType = constraint.ConstraintType.String
		}
		columnName := constraint.ColumnName

		if columnName == "<nil>" {
			continue // Skip constraints without columns
		}

		key := constraintKey{
			schema: schemaName,
			table:  tableName,
			name:   constraintName,
		}

		// Get or create constraint
		c, exists := constraintGroups[key]
		if !exists {
			var cType ConstraintType
			switch constraintType {
			case "PRIMARY KEY":
				cType = ConstraintTypePrimaryKey
			case "UNIQUE":
				cType = ConstraintTypeUnique
			case "FOREIGN KEY":
				cType = ConstraintTypeForeignKey
			case "CHECK":
				cType = ConstraintTypeCheck
			default:
				continue // Skip unknown constraint types
			}

			c = &Constraint{
				Schema:  schemaName,
				Table:   tableName,
				Name:    constraintName,
				Type:    cType,
				Columns: []*ConstraintColumn{},
			}

			// Handle foreign key references
			if cType == ConstraintTypeForeignKey {
				if refSchema := i.safeInterfaceToString(constraint.ForeignTableSchema); refSchema != "" && refSchema != "<nil>" {
					c.ReferencedSchema = refSchema
				}
				if refTable := i.safeInterfaceToString(constraint.ForeignTableName); refTable != "" && refTable != "<nil>" {
					c.ReferencedTable = refTable
				}
				if deleteRule := i.safeInterfaceToString(constraint.DeleteRule); deleteRule != "" && deleteRule != "<nil>" {
					c.DeleteRule = deleteRule
				}
				if updateRule := i.safeInterfaceToString(constraint.UpdateRule); updateRule != "" && updateRule != "<nil>" {
					c.UpdateRule = updateRule
				}
				// Handle deferrable attributes for foreign key constraints
				c.Deferrable = constraint.Deferrable
				c.InitiallyDeferred = constraint.InitiallyDeferred
			}

			// Handle check constraints
			if cType == ConstraintTypeCheck {
				if checkClause := i.safeInterfaceToString(constraint.CheckClause); checkClause != "" && checkClause != "<nil>" {
					// Skip system-generated NOT NULL constraints as they're redundant with column definitions
					if strings.Contains(checkClause, "IS NOT NULL") {
						continue
					}
					c.CheckClause = checkClause
				}
			}

			// Set validation state from database
			c.IsValid = constraint.IsValid

			constraintGroups[key] = c
		}

		// Get column position in constraint
		position := i.getConstraintColumnPosition(ctx, schemaName, constraintName, columnName)

		// Check if column already exists in constraint to avoid duplicates
		columnExists := false
		for _, existingCol := range c.Columns {
			if existingCol.Name == columnName {
				columnExists = true
				break
			}
		}

		// Add column to constraint only if it doesn't exist
		if !columnExists {
			constraintCol := &ConstraintColumn{
				Name:     columnName,
				Position: position,
			}
			c.Columns = append(c.Columns, constraintCol)
		}

		// Handle foreign key referenced columns
		if c.Type == ConstraintTypeForeignKey {
			if refColumnName := i.safeInterfaceToString(constraint.ForeignColumnName); refColumnName != "" && refColumnName != "<nil>" {
				// Check if referenced column already exists to avoid duplicates
				refColumnExists := false
				for _, existingRefCol := range c.ReferencedColumns {
					if existingRefCol.Name == refColumnName {
						refColumnExists = true
						break
					}
				}

				// Add referenced column only if it doesn't exist
				if !refColumnExists {
					// Get the foreign ordinal position for proper ordering
					refPosition := position // Default fallback to source position
					if constraint.ForeignOrdinalPosition.Valid {
						refPosition = int(constraint.ForeignOrdinalPosition.Int32)
					}

					refConstraintCol := &ConstraintColumn{
						Name:     refColumnName,
						Position: refPosition, // Use foreign ordinal position for referenced column
					}
					c.ReferencedColumns = append(c.ReferencedColumns, refConstraintCol)
				}
			}
		}
	}

	// Build a mapping of partition tables to their parent's partition keys
	partitionMapping := i.buildPartitionMapping(ctx, schema, targetSchema)

	// Add constraints to tables
	for key, constraint := range constraintGroups {
		dbSchema := schema.getOrCreateSchema(key.schema)
		table, exists := dbSchema.Tables[key.table]
		if exists {
			// Sort constraint columns by their position to preserve original order from database
			// This ensures constraints maintain the correct column order as defined in PostgreSQL
			if requiresPositionSorting(constraint.Type) {
				sort.Slice(constraint.Columns, func(i, j int) bool {
					return constraint.Columns[i].Position < constraint.Columns[j].Position
				})
				
				// Also sort referenced columns for foreign keys
				if constraint.Type == ConstraintTypeForeignKey && len(constraint.ReferencedColumns) > 0 {
					sort.Slice(constraint.ReferencedColumns, func(i, j int) bool {
						return constraint.ReferencedColumns[i].Position < constraint.ReferencedColumns[j].Position
					})
				}
			}
			
			table.Constraints[key.name] = constraint

			// For partitioned tables, ensure primary key columns are ordered with partition key first
			// This special handling overrides the position-based sorting for partitioned tables
			if constraint.Type == ConstraintTypePrimaryKey && table.IsPartitioned && table.PartitionKey != "" {
				i.sortPrimaryKeyColumnsForPartitionedTable(constraint, table.PartitionKey)
			}

			// For partition tables (children of partitioned tables), use parent's partition key
			if constraint.Type == ConstraintTypePrimaryKey && !table.IsPartitioned {
				if parentPartitionKey, isPartitionTable := partitionMapping[key.table]; isPartitionTable {
					i.sortPrimaryKeyColumnsForPartitionedTable(constraint, parentPartitionKey)
				}
			}
		}
	}

	return nil
}

// requiresPositionSorting returns true if the constraint type requires columns to be sorted by position
func requiresPositionSorting(constraintType ConstraintType) bool {
	switch constraintType {
	case ConstraintTypeUnique, ConstraintTypePrimaryKey, ConstraintTypeForeignKey:
		return true
	default:
		return false
	}
}

// buildPartitionMapping builds a mapping from partition table names to their parent's partition keys
func (i *Inspector) buildPartitionMapping(ctx context.Context, schema *IR, targetSchema string) map[string]string {
	partitionMapping := make(map[string]string)

	// Get partition children information
	partitionChildren, err := i.queries.GetPartitionChildren(ctx)
	if err != nil {
		// If we can't get partition info, return empty mapping
		return partitionMapping
	}

	for _, child := range partitionChildren {
		// Only process children in the target schema
		if child.ChildSchema != targetSchema {
			continue
		}

		childTable := child.ChildTable
		parentTable := child.ParentTable

		// Find the parent table's partition key
		dbSchema := schema.getOrCreateSchema(targetSchema)
		if parentTableInfo, exists := dbSchema.Tables[parentTable]; exists && parentTableInfo.IsPartitioned {
			partitionMapping[childTable] = parentTableInfo.PartitionKey
		}
	}

	return partitionMapping
}

// sortPrimaryKeyColumnsForPartitionedTable sorts primary key constraint columns
// to ensure partition key columns come first
func (i *Inspector) sortPrimaryKeyColumnsForPartitionedTable(constraint *Constraint, partitionKey string) {
	if constraint.Type != ConstraintTypePrimaryKey || len(constraint.Columns) <= 1 {
		return
	}

	// Parse partition key to handle multi-column partitions
	partitionColumns := make(map[string]bool)
	for _, col := range strings.Split(partitionKey, ",") {
		partitionColumns[strings.TrimSpace(col)] = true
	}

	// Separate partition columns from non-partition columns
	var partitionCols []*ConstraintColumn
	var nonPartitionCols []*ConstraintColumn

	for _, col := range constraint.Columns {
		if partitionColumns[col.Name] {
			partitionCols = append(partitionCols, col)
		} else {
			nonPartitionCols = append(nonPartitionCols, col)
		}
	}

	// Sort partition columns by their position to maintain consistent ordering
	sort.Slice(partitionCols, func(i, j int) bool {
		return partitionCols[i].Position < partitionCols[j].Position
	})

	// Sort non-partition columns by their position
	sort.Slice(nonPartitionCols, func(i, j int) bool {
		return nonPartitionCols[i].Position < nonPartitionCols[j].Position
	})

	// Rebuild the columns list with partition columns first
	constraint.Columns = append(partitionCols, nonPartitionCols...)
}

func (i *Inspector) buildIndexes(ctx context.Context, schema *IR, targetSchema string) error {
	// Get indexes for the target schema
	indexes, err := i.queries.GetIndexesForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	for _, indexRow := range indexes {
		schemaName := indexRow.Schemaname
		tableName := indexRow.Tablename
		indexName := indexRow.Indexname

		dbSchema := schema.getOrCreateSchema(schemaName)

		// Extract values with null safety
		isUnique := indexRow.IsUnique
		isPrimary := indexRow.IsPrimary
		isPartial := indexRow.IsPartial.Valid && indexRow.IsPartial.Bool
		hasExpressions := indexRow.HasExpressions.Valid && indexRow.HasExpressions.Bool
		method := indexRow.Method
		definition := ""
		if indexRow.Indexdef.Valid {
			definition = indexRow.Indexdef.String
		}

		// Determine index type based on properties
		indexType := IndexTypeRegular
		if isPrimary {
			indexType = IndexTypePrimary
		} else if isUnique {
			indexType = IndexTypeUnique
		}

		// Extract comment
		comment := ""
		if indexRow.IndexComment.Valid {
			comment = indexRow.IndexComment.String
		}

		index := &Index{
			Schema:       schemaName,
			Table:        tableName,
			Name:         indexName,
			Type:         indexType,
			Method:       method,
			IsPartial:    isPartial,
			IsExpression: hasExpressions,
			Where:        "",
			Comment:      comment,
			Columns:      []*IndexColumn{},
		}

		// Set WHERE clause for partial indexes
		if isPartial && indexRow.PartialPredicate.Valid {
			// Use the predicate as-is from pg_get_expr, which already has proper formatting
			index.Where = indexRow.PartialPredicate.String
		}

		// Parse index definition to extract columns
		if err := i.parseIndexDefinition(index, definition); err != nil {
			// If parsing fails, just continue with empty columns
			// This ensures backward compatibility
			continue
		}

		// Store the original definition - simplification will be done during read time in diff module
		// Definition is now generated on demand, not stored

		// Add index to table only
		if table, exists := dbSchema.Tables[tableName]; exists {
			table.Indexes[indexName] = index
		}
	}

	return nil
}

// parseIndexDefinition parses an index definition string to extract method and columns using pg_query_go
// Expected format: "CREATE [UNIQUE] INDEX index_name ON [schema.]table USING method (column1 [ASC|DESC], column2, ...)"
func (i *Inspector) parseIndexDefinition(index *Index, definition string) error {
	if definition == "" {
		return fmt.Errorf("empty index definition")
	}

	// Parse the definition string using pg_query
	result, err := pg_query.Parse(definition)
	if err != nil {
		return fmt.Errorf("failed to parse index definition: %w", err)
	}

	// Find the CREATE INDEX statement in the parsed result
	var indexStmt *pg_query.IndexStmt
	for _, stmt := range result.Stmts {
		if node := stmt.GetStmt(); node != nil {
			if indexNode := node.GetIndexStmt(); indexNode != nil {
				indexStmt = indexNode
				break
			}
		}
	}

	if indexStmt == nil {
		return fmt.Errorf("no CREATE INDEX statement found in definition")
	}

	// Extract index method
	if indexStmt.AccessMethod != "" {
		index.Method = indexStmt.AccessMethod
	} else {
		// Default to btree if not specified
		index.Method = "btree"
	}

	// Parse index columns from IndexParams
	for idx, indexElem := range indexStmt.IndexParams {
		if elem := indexElem.GetIndexElem(); elem != nil {
			var columnName string
			var direction string

			// Extract column name or expression directly from AST
			if elem.Name != "" {
				// Simple column name
				columnName = elem.Name
			} else if elem.Expr != nil {
				// Expression column - extract directly from AST
				columnName = i.extractExpressionFromAST(elem.Expr)
			}

			// Extract sort direction directly from AST
			switch elem.Ordering {
			case pg_query.SortByDir_SORTBY_ASC:
				direction = "ASC"
			case pg_query.SortByDir_SORTBY_DESC:
				direction = "DESC"
			default:
				direction = "ASC" // Default
			}

			if columnName != "" {
				indexColumn := &IndexColumn{
					Name:      columnName,
					Position:  idx + 1,
					Direction: direction,
				}

				index.Columns = append(index.Columns, indexColumn)
			}
		}
	}

	return nil
}

// extractExpressionFromAST extracts a string representation of an expression node for index definitions
func (i *Inspector) extractExpressionFromAST(expr *pg_query.Node) string {
	if expr == nil {
		return ""
	}

	switch n := expr.Node.(type) {
	case *pg_query.Node_ColumnRef:
		return i.extractColumnNameFromAST(expr)
	case *pg_query.Node_AExpr:
		// Handle binary expressions like JSON operators
		return i.extractBinaryExpressionFromAST(n.AExpr)
	case *pg_query.Node_FuncCall:
		// Handle function calls in expressions
		return i.extractFunctionCallFromAST(n.FuncCall)
	case *pg_query.Node_AConst:
		// Handle constants
		return i.extractConstantValueFromAST(expr)
	case *pg_query.Node_TypeCast:
		// Handle type casting expressions like 'method'::text
		return i.extractTypeCastFromAST(n.TypeCast)
	default:
		// For unhandled cases, return a placeholder
		return "(expression)"
	}
}

// extractColumnNameFromAST extracts column name from a ColumnRef node
func (i *Inspector) extractColumnNameFromAST(node *pg_query.Node) string {
	if columnRef := node.GetColumnRef(); columnRef != nil {
		if len(columnRef.Fields) > 0 {
			var parts []string
			for _, field := range columnRef.Fields {
				if field != nil {
					if str := field.GetString_(); str != nil {
						parts = append(parts, str.Sval)
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

// extractBinaryExpressionFromAST extracts string representation of binary expressions
func (i *Inspector) extractBinaryExpressionFromAST(aExpr *pg_query.A_Expr) string {
	if aExpr == nil {
		return ""
	}

	left := ""
	if aExpr.Lexpr != nil {
		left = i.extractExpressionFromAST(aExpr.Lexpr)
	}

	right := ""
	if aExpr.Rexpr != nil {
		right = i.extractExpressionFromAST(aExpr.Rexpr)
	}

	operator := ""
	if len(aExpr.Name) > 0 {
		if opNode := aExpr.Name[0]; opNode != nil {
			if str := opNode.GetString_(); str != nil {
				operator = str.Sval
			}
		}
	}

	if left != "" && right != "" && operator != "" {
		// Handle JSON operators specially - don't add extra parentheses
		if operator == "->>" || operator == "->" {
			return fmt.Sprintf("%s%s%s", left, operator, right)
		}
		return fmt.Sprintf("(%s %s %s)", left, operator, right)
	}

	return fmt.Sprintf("(%s)", left)
}

// extractFunctionCallFromAST extracts string representation of function calls
func (i *Inspector) extractFunctionCallFromAST(funcCall *pg_query.FuncCall) string {
	if funcCall == nil {
		return ""
	}

	// Extract function name
	funcName := ""
	if len(funcCall.Funcname) > 0 {
		if nameNode := funcCall.Funcname[0]; nameNode != nil {
			if str := nameNode.GetString_(); str != nil {
				funcName = str.Sval
			}
		}
	}

	if funcName == "" {
		return "function()"
	}

	// Extract function arguments
	var args []string
	if len(funcCall.Args) > 0 {
		for _, argNode := range funcCall.Args {
			if argNode != nil {
				argStr := i.extractExpressionFromAST(argNode)
				if argStr != "" {
					args = append(args, argStr)
				}
			}
		}
	}

	// Build function call with arguments
	if len(args) > 0 {
		return fmt.Sprintf("%s(%s)", funcName, strings.Join(args, ", "))
	}
	return fmt.Sprintf("%s()", funcName)
}

// extractConstantValueFromAST extracts string representation of constants
func (i *Inspector) extractConstantValueFromAST(node *pg_query.Node) string {
	if aConst := node.GetAConst(); aConst != nil {
		if aConst.Isnull {
			return "NULL"
		}
		if aConst.Val != nil {
			switch val := aConst.Val.(type) {
			case *pg_query.A_Const_Sval:
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

// extractTypeCastFromAST extracts string representation of type cast expressions
func (i *Inspector) extractTypeCastFromAST(typeCast *pg_query.TypeCast) string {
	if typeCast == nil {
		return ""
	}

	// Extract the expression being cast
	expr := ""
	if typeCast.Arg != nil {
		expr = i.extractExpressionFromAST(typeCast.Arg)
	}

	return expr
}

func (i *Inspector) buildSequences(ctx context.Context, schema *IR, targetSchema string) error {
	sequences, err := i.queries.GetSequencesForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	for _, seq := range sequences {
		schemaName := seq.SequenceSchema.String
		sequenceName := seq.SequenceName.String

		// Check if sequence should be ignored
		if i.ignoreConfig != nil && i.ignoreConfig.ShouldIgnoreSequence(sequenceName) {
			continue
		}

		dbSchema := schema.getOrCreateSchema(schemaName)

		// Set empty DataType for sequences that use PostgreSQL's implicit bigint default
		dataType := fmt.Sprintf("%s", seq.DataType)
		if dataType == "bigint" {
			// Check if this is a default bigint by looking at min/max values
			// Default bigint sequences have min_value=1 and max_value=9223372036854775807
			if seq.MinimumValue.Valid && seq.MinimumValue.Int64 == 1 && 
			   seq.MaximumValue.Valid && seq.MaximumValue.Int64 == 9223372036854775807 {
				dataType = "" // This means it was not explicitly specified
			}
		}

		sequence := &Sequence{
			Schema:      schemaName,
			Name:        sequenceName,
			DataType:    dataType,
			StartValue:  seq.StartValue.Int64,
			Increment:   seq.Increment.Int64,
			CycleOption: seq.CycleOption.Bool,
		}

		// Set default values if not valid
		if !seq.StartValue.Valid {
			sequence.StartValue = 1
		}
		if !seq.Increment.Valid {
			sequence.Increment = 1
		}

		// Only set MinValue/MaxValue if they differ from the data type defaults
		if seq.MinimumValue.Valid {
			minVal := seq.MinimumValue.Int64
			// Only set if not the default (1) for this data type
			if minVal != 1 {
				sequence.MinValue = &minVal
			}
		}

		if seq.MaximumValue.Valid {
			maxVal := seq.MaximumValue.Int64
			var defaultMax int64
			switch dataType {
			case "smallint":
				defaultMax = 32767 // smallint max
			case "integer":
				defaultMax = 2147483647 // integer max
			case "bigint", "":
				defaultMax = 9223372036854775807 // bigint max (math.MaxInt64)
			default:
				defaultMax = 9223372036854775807 // bigint max (math.MaxInt64)
			}
			// Only set if not the default for this data type
			if maxVal != defaultMax {
				sequence.MaxValue = &maxVal
			}
		}

		// Set cache value if it's different from default (1)
		if seq.CacheSize.Valid && seq.CacheSize.Int64 != 1 {
			cacheVal := seq.CacheSize.Int64
			sequence.Cache = &cacheVal
		}

		// Set ownership information
		if seq.OwnedByTable.Valid {
			sequence.OwnedByTable = seq.OwnedByTable.String
		}
		if seq.OwnedByColumn.Valid {
			sequence.OwnedByColumn = seq.OwnedByColumn.String
		}

		// Skip sequences that are owned by identity columns
		// Identity sequences should be managed through the identity column, not as separate sequences
		if sequence.OwnedByTable != "" && sequence.OwnedByColumn != "" {
			// Check if the owning column is an identity column
			if i.isIdentityColumn(ctx, seq.SequenceSchema.String, sequence.OwnedByTable, sequence.OwnedByColumn) {
				// Skip this sequence - it's managed by the identity column
				continue
			}
		}

		dbSchema.SetSequence(sequenceName, sequence)
	}

	return nil
}

// isIdentityColumn checks if a column is an identity column
func (i *Inspector) isIdentityColumn(ctx context.Context, schemaName, tableName, columnName string) bool {
	query := `
		SELECT is_identity 
		FROM information_schema.columns 
		WHERE table_schema = $1 
		  AND table_name = $2 
		  AND column_name = $3`
	
	var isIdentity string
	err := i.db.QueryRowContext(ctx, query, schemaName, tableName, columnName).Scan(&isIdentity)
	if err != nil {
		return false
	}
	
	return isIdentity == "YES"
}

func (i *Inspector) buildFunctions(ctx context.Context, schema *IR, targetSchema string) error {
	functions, err := i.queries.GetFunctionsForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	for _, fn := range functions {
		schemaName := fmt.Sprintf("%s", fn.RoutineSchema)
		functionName := fmt.Sprintf("%s", fn.RoutineName)
		comment := ""
		if fn.FunctionComment.Valid {
			comment = fn.FunctionComment.String
		}
		arguments := i.safeInterfaceToString(fn.FunctionArguments)
		signature := i.safeInterfaceToString(fn.FunctionSignature)

		// Check if function should be ignored
		if i.ignoreConfig != nil && i.ignoreConfig.ShouldIgnoreFunction(functionName) {
			continue
		}

		// Get function definition from pg_get_functiondef
		definition := i.safeInterfaceToString(fn.RoutineDefinition)

		dbSchema := schema.getOrCreateSchema(schemaName)

		// Handle volatility
		volatility := i.safeInterfaceToString(fn.Volatility)

		// Handle strictness
		isStrict := fn.IsStrict

		// Handle security definer
		isSecurityDefiner := fn.IsSecurityDefiner

		// Parse parameters from system catalog arrays (preferred method)
		var parameters []*Parameter
		if arrayParams := i.parseParametersFromProcArrays(fn.Proargmodes, fn.Proargnames, fn.Proallargtypes); arrayParams != nil {
			parameters = arrayParams
		} else {
			// Fall back to parsing from signature for simple functions without modes
			parameters = i.parseParametersFromSignature(signature)
		}

		function := &Function{
			Schema:            schemaName,
			Name:              functionName,
			Definition:        definition,
			ReturnType:        i.safeInterfaceToString(fn.DataType),
			Language:          i.safeInterfaceToString(fn.ExternalLanguage),
			Arguments:         arguments,
			Signature:         signature,
			Comment:           comment,
			Parameters:        parameters,
			Volatility:        volatility,
			IsStrict:          isStrict,
			IsSecurityDefiner: isSecurityDefiner,
		}

		dbSchema.SetFunction(functionName, function)
	}

	return nil
}

// parseParametersFromSignature parses function signature string into Parameter structs
// Example signature: "order_id integer, discount_percent numeric DEFAULT 0"
func (i *Inspector) parseParametersFromSignature(signature string) []*Parameter {
	if signature == "" {
		return nil
	}

	var parameters []*Parameter
	position := 1

	// Split by comma to get individual parameters
	paramStrings := strings.Split(signature, ",")
	for _, paramStr := range paramStrings {
		paramStr = strings.TrimSpace(paramStr)
		if paramStr == "" {
			continue
		}

		param := &Parameter{
			Mode:     "IN", // Default mode for inspector
			Position: position,
		}

		// Look for DEFAULT clause
		defaultIdx := strings.Index(strings.ToUpper(paramStr), " DEFAULT ")
		if defaultIdx != -1 {
			// Extract default value
			defaultValue := strings.TrimSpace(paramStr[defaultIdx+9:]) // " DEFAULT " is 9 chars
			param.DefaultValue = &defaultValue
			paramStr = strings.TrimSpace(paramStr[:defaultIdx])
		}

		// Split into name and type
		parts := strings.Fields(paramStr)
		if len(parts) >= 2 {
			param.Name = parts[0]
			param.DataType = strings.Join(parts[1:], " ")
		} else if len(parts) == 1 {
			// Only type, no name (shouldn't happen but handle gracefully)
			param.DataType = parts[0]
		}

		parameters = append(parameters, param)
		position++
	}

	return parameters
}

// parseParametersFromProcArrays parses function parameters from PostgreSQL system catalog arrays
// proargmodes: parameter modes (i=IN, o=OUT, b=INOUT, v=VARIADIC, t=TABLE)
// proargnames: parameter names
// proallargtypes: parameter type OIDs (as text)
func (i *Inspector) parseParametersFromProcArrays(proargmodes []string, proargnames []string, proallargtypes []string) []*Parameter {
	// If no modes are specified, all parameters are IN parameters
	if len(proargmodes) == 0 {
		// Fall back to parsing from signature for simple functions
		return nil
	}

	var parameters []*Parameter
	position := 1

	for idx, modeStr := range proargmodes {

		// Convert PostgreSQL mode characters to our mode strings
		var mode string
		switch modeStr {
		case "i":
			mode = "IN"
		case "o":
			mode = "OUT"
		case "b":
			mode = "INOUT"
		case "v":
			mode = "VARIADIC"
		case "t":
			mode = "TABLE"
		default:
			mode = "IN" // Default to IN for unknown modes
		}

		parameter := &Parameter{
			Mode:     mode,
			Position: position,
		}

		// Extract parameter name if available
		if idx < len(proargnames) && proargnames[idx] != "" {
			parameter.Name = proargnames[idx]
		}

		// Extract parameter type if available
		if idx < len(proallargtypes) {
			// Convert OID string to integer and look up type name
			if oidStr := proallargtypes[idx]; oidStr != "" {
				// Try to convert to integer OID first
				if typeOid, err := strconv.ParseInt(oidStr, 10, 64); err == nil {
					parameter.DataType = i.lookupTypeNameFromOID(typeOid)
				} else {
					// If it's not a number, treat as a direct type name
					parameter.DataType = oidStr
				}
			}
		}

		parameters = append(parameters, parameter)
		position++
	}

	return parameters
}

// lookupTypeNameFromOID converts PostgreSQL type OID to type name
func (i *Inspector) lookupTypeNameFromOID(oid int64) string {
	// Common type OID mappings (can be extended as needed)
	typeMap := map[int64]string{
		16:   "boolean",
		20:   "bigint",
		21:   "smallint",
		23:   "integer",
		25:   "text",
		1043: "character varying",
		1082: "date",
		1114: "timestamp without time zone", // Will be normalized later
		1184: "timestamp with time zone",
		1700: "numeric",
		2950: "uuid",
	}

	if typeName, exists := typeMap[oid]; exists {
		return typeName
	}

	// For unknown OIDs, return a placeholder
	// In a real implementation, this could query pg_type
	return fmt.Sprintf("oid_%d", oid)
}

func (i *Inspector) buildProcedures(ctx context.Context, schema *IR, targetSchema string) error {
	procedures, err := i.queries.GetProceduresForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	for _, proc := range procedures {
		schemaName := fmt.Sprintf("%s", proc.RoutineSchema)
		procedureName := fmt.Sprintf("%s", proc.RoutineName)
		comment := ""
		if proc.ProcedureComment.Valid {
			comment = proc.ProcedureComment.String
		}
		arguments := i.safeInterfaceToString(proc.ProcedureArguments)
		signature := i.safeInterfaceToString(proc.ProcedureSignature)

		// Check if procedure should be ignored
		if i.ignoreConfig != nil && i.ignoreConfig.ShouldIgnoreProcedure(procedureName) {
			continue
		}

		// Get procedure definition from pg_get_functiondef
		definition := i.safeInterfaceToString(proc.RoutineDefinition)

		dbSchema := schema.getOrCreateSchema(schemaName)

		procedure := &Procedure{
			Schema:     schemaName,
			Name:       procedureName,
			Definition: definition,
			Language:   i.safeInterfaceToString(proc.ExternalLanguage),
			Arguments:  arguments,
			Signature:  signature,
			Comment:    comment,
			Parameters: []*Parameter{}, // TODO: parse parameters
		}

		dbSchema.SetProcedure(procedureName, procedure)
	}

	return nil
}

func (i *Inspector) buildAggregates(ctx context.Context, schema *IR, targetSchema string) error {
	aggregates, err := i.queries.GetAggregatesForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	for _, agg := range aggregates {
		schemaName := agg.AggregateSchema
		aggregateName := agg.AggregateName
		comment := ""
		if agg.AggregateComment.Valid {
			comment = agg.AggregateComment.String
		}
		arguments := i.safeInterfaceToString(agg.AggregateArguments)
		signature := i.safeInterfaceToString(agg.AggregateSignature)
		returnType := i.safeInterfaceToString(agg.AggregateReturnType)
		transitionFunction := i.safeInterfaceToString(agg.TransitionFunction)
		transitionFunctionSchema := i.safeInterfaceToString(agg.TransitionFunctionSchema)
		stateType := i.safeInterfaceToString(agg.StateType)
		initialCondition := i.safeInterfaceToString(agg.InitialCondition)
		finalFunction := i.safeInterfaceToString(agg.FinalFunction)
		finalFunctionSchema := i.safeInterfaceToString(agg.FinalFunctionSchema)

		dbSchema := schema.getOrCreateSchema(schemaName)

		aggregate := &Aggregate{
			Schema:                   schemaName,
			Name:                     aggregateName,
			Arguments:                arguments,
			Signature:                signature,
			ReturnType:               returnType,
			TransitionFunction:       transitionFunction,
			TransitionFunctionSchema: transitionFunctionSchema,
			StateType:                stateType,
			InitialCondition:         initialCondition,
			FinalFunction:            finalFunction,
			FinalFunctionSchema:      finalFunctionSchema,
			Comment:                  comment,
		}

		dbSchema.SetAggregate(aggregateName, aggregate)
	}

	return nil
}

func (i *Inspector) buildViews(ctx context.Context, schema *IR, targetSchema string) error {
	views, err := i.queries.GetViewsForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	for _, view := range views {
		schemaName := view.TableSchema
		viewName := view.TableName
		comment := ""
		if view.ViewComment.Valid {
			comment = view.ViewComment.String
		}

		// Check if view should be ignored
		if i.ignoreConfig != nil && i.ignoreConfig.ShouldIgnoreView(viewName) {
			continue
		}

		dbSchema := schema.getOrCreateSchema(schemaName)

		var definition string
		if view.ViewDefinition.Valid {
			definition = view.ViewDefinition.String
			// Strip trailing semicolon from pg_get_viewdef() output for consistency with parser
			definition = strings.TrimSuffix(definition, ";")
		}

		v := &View{
			Schema:       schemaName,
			Name:         viewName,
			Definition:   definition,
			Comment:      comment,
			Materialized: view.IsMaterialized.Valid && view.IsMaterialized.Bool,
		}

		dbSchema.SetView(viewName, v)
	}

	return nil
}

func (i *Inspector) buildTriggers(ctx context.Context, schema *IR, targetSchema string) error {
	triggers, err := i.queries.GetTriggersForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	// Group triggers by name to handle multiple events
	type triggerKey struct {
		schema string
		table  string
		name   string
	}
	triggerGroups := make(map[triggerKey]*Trigger)

	for _, trigger := range triggers {
		schemaName := fmt.Sprintf("%s", trigger.TriggerSchema)
		tableName := fmt.Sprintf("%s", trigger.EventObjectTable)
		triggerName := fmt.Sprintf("%s", trigger.TriggerName)
		timing := fmt.Sprintf("%s", trigger.ActionTiming)
		event := fmt.Sprintf("%s", trigger.EventManipulation)
		statement := fmt.Sprintf("%s", trigger.ActionStatement)
		orientation := fmt.Sprintf("%s", trigger.ActionOrientation)

		// Handle the action_condition field which may be nil
		var condition string
		if trigger.ActionCondition != nil {
			rawCondition := fmt.Sprintf("%s", trigger.ActionCondition)
			condition = i.normalizeTriggerCondition(rawCondition)
		}

		key := triggerKey{
			schema: schemaName,
			table:  tableName,
			name:   triggerName,
		}

		t, exists := triggerGroups[key]
		if !exists {
			var tTiming TriggerTiming
			switch timing {
			case "BEFORE":
				tTiming = TriggerTimingBefore
			case "AFTER":
				tTiming = TriggerTimingAfter
			case "INSTEAD OF":
				tTiming = TriggerTimingInsteadOf
			default:
				tTiming = TriggerTimingAfter
			}

			// Determine trigger level from orientation
			var level TriggerLevel
			switch orientation {
			case "ROW":
				level = TriggerLevelRow
			case "STATEMENT":
				level = TriggerLevelStatement
			default:
				level = TriggerLevelRow // Default to ROW if unknown
			}

			t = &Trigger{
				Schema:    schemaName,
				Table:     tableName,
				Name:      triggerName,
				Timing:    tTiming,
				Events:    []TriggerEvent{},
				Level:     level,
				Function:  i.extractFunctionFromStatement(statement),
				Condition: condition,
			}

			triggerGroups[key] = t
		}

		// Add event
		var tEvent TriggerEvent
		switch event {
		case "INSERT":
			tEvent = TriggerEventInsert
		case "UPDATE":
			tEvent = TriggerEventUpdate
		case "DELETE":
			tEvent = TriggerEventDelete
		case "TRUNCATE":
			tEvent = TriggerEventTruncate
		default:
			continue
		}

		// Check if event already exists
		eventExists := false
		for _, existingEvent := range t.Events {
			if existingEvent == tEvent {
				eventExists = true
				break
			}
		}
		if !eventExists {
			t.Events = append(t.Events, tEvent)
		}
	}

	// Add triggers to tables only
	for key, trigger := range triggerGroups {
		dbSchema := schema.getOrCreateSchema(key.schema)

		if table, exists := dbSchema.Tables[key.table]; exists {
			table.Triggers[key.name] = trigger
		}
	}

	return nil
}

// normalizeTriggerCondition normalizes the trigger condition from the database
// to match the format expected by the parser
func (i *Inspector) normalizeTriggerCondition(rawCondition string) string {
	if rawCondition == "" {
		return ""
	}

	// Remove outer parentheses if they wrap the entire condition
	condition := strings.TrimSpace(rawCondition)
	if len(condition) > 2 && condition[0] == '(' && condition[len(condition)-1] == ')' {
		// Check if these are outer parentheses by counting balanced parens
		parenCount := 0
		canRemove := true
		for _, char := range condition[1 : len(condition)-1] {
			if char == '(' {
				parenCount++
			} else if char == ')' {
				parenCount--
				if parenCount < 0 {
					canRemove = false
					break
				}
			}
		}
		if canRemove && parenCount == 0 {
			condition = strings.TrimSpace(condition[1 : len(condition)-1])
		}
	}

	return condition
}

func (i *Inspector) buildRLSPolicies(ctx context.Context, schema *IR, targetSchema string) error {
	// Get RLS enabled tables for the target schema
	rlsTables, err := i.queries.GetRLSTablesForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	// Mark tables as RLS enabled
	for _, rlsTable := range rlsTables {
		schemaName := ""
		if rlsTable.Schemaname.Valid {
			schemaName = rlsTable.Schemaname.String
		}
		tableName := ""
		if rlsTable.Tablename.Valid {
			tableName = rlsTable.Tablename.String
		}

		dbSchema := schema.getOrCreateSchema(schemaName)
		if table, exists := dbSchema.Tables[tableName]; exists {
			table.RLSEnabled = true
		}
	}

	// Get RLS policies for the target schema
	policies, err := i.queries.GetRLSPoliciesForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	// Process policies
	for _, policyRow := range policies {
		schemaName := ""
		if policyRow.Schemaname.Valid {
			schemaName = policyRow.Schemaname.String
		}
		tableName := ""
		if policyRow.Tablename.Valid {
			tableName = policyRow.Tablename.String
		}
		policyName := ""
		if policyRow.Policyname.Valid {
			policyName = policyRow.Policyname.String
		}

		var pCommand PolicyCommand
		if policyRow.Cmd.Valid {
			switch policyRow.Cmd.String {
			case "SELECT":
				pCommand = PolicyCommandSelect
			case "INSERT":
				pCommand = PolicyCommandInsert
			case "UPDATE":
				pCommand = PolicyCommandUpdate
			case "DELETE":
				pCommand = PolicyCommandDelete
			case "ALL":
				pCommand = PolicyCommandAll
			default:
				pCommand = PolicyCommandAll
			}
		} else {
			pCommand = PolicyCommandAll
		}

		// Determine if policy is permissive
		permissive := true // Default
		if policyRow.Permissive.Valid {
			permissive = policyRow.Permissive.String == "PERMISSIVE"
		}

		policy := &RLSPolicy{
			Schema:     schemaName,
			Table:      tableName,
			Name:       policyName,
			Command:    pCommand,
			Permissive: permissive,
			Roles:      policyRow.Roles,
		}

		if policyRow.Qual.Valid {
			policy.Using = policyRow.Qual.String
		}

		if policyRow.WithCheck.Valid {
			policy.WithCheck = policyRow.WithCheck.String
		}

		dbSchema := schema.getOrCreateSchema(schemaName)

		if table, exists := dbSchema.Tables[tableName]; exists {
			table.Policies[policyName] = policy
		}
	}

	return nil
}

func (i *Inspector) buildTypes(ctx context.Context, schema *IR, targetSchema string) error {
	types, err := i.queries.GetTypesForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	// Get domains
	domains, err := i.queries.GetDomainsForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	// Get domain constraints
	domainConstraints, err := i.queries.GetDomainConstraintsForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	// Get enum values for ENUM types
	enumValues, err := i.queries.GetEnumValuesForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	// Get columns for composite types
	compositeColumns, err := i.queries.GetCompositeTypeColumnsForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	// Create maps for efficient lookup
	enumValuesMap := make(map[string][]string)
	compositeColumnsMap := make(map[string][]*TypeColumn)
	domainConstraintsMap := make(map[string][]*DomainConstraint)

	// Process enum values
	for _, enumVal := range enumValues {
		key := fmt.Sprintf("%s.%s", enumVal.TypeSchema, enumVal.TypeName)
		enumValuesMap[key] = append(enumValuesMap[key], enumVal.EnumValue)
	}

	// Process composite columns
	for _, col := range compositeColumns {
		key := fmt.Sprintf("%s.%s", col.TypeSchema, col.TypeName)
		position := i.safeInterfaceToInt(col.ColumnPosition, 0)

		dataType := ""
		if col.ColumnType.Valid {
			dataType = col.ColumnType.String
		}

		typeCol := &TypeColumn{
			Name:     col.ColumnName,
			DataType: dataType,
			Position: position,
		}

		compositeColumnsMap[key] = append(compositeColumnsMap[key], typeCol)
	}

	// Process domain constraints
	for _, constraint := range domainConstraints {
		key := fmt.Sprintf("%s.%s", i.safeInterfaceToString(constraint.DomainSchema), i.safeInterfaceToString(constraint.DomainName))
		constraintName := i.safeInterfaceToString(constraint.ConstraintName)
		constraintDef := i.safeInterfaceToString(constraint.ConstraintDefinition)

		// Skip NOT NULL constraints as they are already captured in the NotNull boolean field
		if constraintDef == "NOT NULL" {
			continue
		}

		domainConstraint := &DomainConstraint{
			Name:       constraintName,
			Definition: constraintDef,
		}

		domainConstraintsMap[key] = append(domainConstraintsMap[key], domainConstraint)
	}

	// Create types
	for _, t := range types {
		schemaName := t.TypeSchema
		typeName := t.TypeName
		typeKindStr := ""
		if t.TypeKind.Valid {
			typeKindStr = t.TypeKind.String
		}
		typeKind := TypeKind(typeKindStr)
		comment := ""
		if t.TypeComment.Valid {
			comment = t.TypeComment.String
		}

		// Check if type should be ignored
		if i.ignoreConfig != nil && i.ignoreConfig.ShouldIgnoreType(typeName) {
			continue
		}

		dbSchema := schema.getOrCreateSchema(schemaName)

		customType := &Type{
			Schema:  schemaName,
			Name:    typeName,
			Kind:    typeKind,
			Comment: comment,
		}

		key := fmt.Sprintf("%s.%s", schemaName, typeName)

		switch typeKind {
		case TypeKindEnum:
			customType.EnumValues = enumValuesMap[key]
		case TypeKindComposite:
			customType.Columns = compositeColumnsMap[key]
		}

		dbSchema.SetType(typeName, customType)
	}

	// Create domains
	for _, d := range domains {
		schemaName := i.safeInterfaceToString(d.DomainSchema)
		domainName := i.safeInterfaceToString(d.DomainName)
		baseType := i.safeInterfaceToString(d.BaseType)
		notNull := i.safeInterfaceToBool(d.NotNull, false)
		defaultValue := i.safeInterfaceToString(d.DefaultValue)
		comment := ""
		if d.DomainComment.Valid {
			comment = d.DomainComment.String
		}

		// Check if domain (type) should be ignored
		if i.ignoreConfig != nil && i.ignoreConfig.ShouldIgnoreType(domainName) {
			continue
		}

		dbSchema := schema.getOrCreateSchema(schemaName)

		key := fmt.Sprintf("%s.%s", schemaName, domainName)
		constraints := domainConstraintsMap[key]

		domainType := &Type{
			Schema:      schemaName,
			Name:        domainName,
			Kind:        TypeKindDomain,
			Comment:     comment,
			BaseType:    baseType,
			NotNull:     notNull,
			Default:     defaultValue,
			Constraints: constraints,
		}

		dbSchema.SetType(domainName, domainType)
	}

	return nil
}

// Helper methods

func (i *Inspector) getConstraintColumnPosition(ctx context.Context, schemaName, constraintName, columnName string) int {
	query := `
		SELECT kcu.ordinal_position
		FROM information_schema.key_column_usage kcu
		WHERE kcu.table_schema = $1
		  AND kcu.constraint_name = $2
		  AND kcu.column_name = $3`

	var position int
	err := i.db.QueryRowContext(ctx, query, schemaName, constraintName, columnName).Scan(&position)
	if err != nil {
		return 0 // Default position if query fails
	}

	return position
}

// pgCatalogTriggerFunctions contains the hard-coded list of built-in PostgreSQL trigger functions
// that exist in the pg_catalog schema. These functions should always be schema-qualified
// to maintain consistency between parser and inspector outputs.
var pgCatalogTriggerFunctions = map[string]bool{
	"suppress_redundant_updates_trigger": true,
	"tsvector_update_trigger":            true,
	"tsvector_update_trigger_column":     true,
}

func (i *Inspector) extractFunctionFromStatement(statement string) string {
	// Extract complete function call from "EXECUTE FUNCTION function_name(...)"
	if strings.Contains(statement, "EXECUTE FUNCTION ") {
		parts := strings.Split(statement, "EXECUTE FUNCTION ")
		if len(parts) > 1 {
			funcPart := strings.TrimSpace(parts[1])

			// Check if this is a pg_catalog built-in function that needs schema qualification
			if needsSchemaQualification := i.shouldAddPgCatalogPrefix(funcPart); needsSchemaQualification {
				// Add pg_catalog prefix if it's missing
				if !strings.HasPrefix(funcPart, "pg_catalog.") {
					funcPart = "pg_catalog." + funcPart
				}
			}

			// Return the complete function call including parameters
			return funcPart
		}
	}
	return statement
}

// shouldAddPgCatalogPrefix determines if a function name should be prefixed with pg_catalog.
// It extracts the base function name and checks if it's in the list of built-in trigger functions.
func (i *Inspector) shouldAddPgCatalogPrefix(funcCall string) bool {
	// Extract just the function name (before the opening parenthesis)
	funcName := funcCall
	if parenIndex := strings.Index(funcCall, "("); parenIndex != -1 {
		funcName = funcCall[:parenIndex]
	}

	// Remove any existing schema qualification to get the base name
	if dotIndex := strings.LastIndex(funcName, "."); dotIndex != -1 {
		funcName = funcName[dotIndex+1:]
	}

	// Check if it's a pg_catalog built-in function
	return pgCatalogTriggerFunctions[funcName]
}

// validateSchemaExists checks if a schema exists in the database
func (i *Inspector) validateSchemaExists(ctx context.Context, schemaName string) error {
	query := `
		SELECT 1 
		FROM information_schema.schemata 
		WHERE schema_name = $1
		  AND schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
		  AND schema_name NOT LIKE 'pg_temp_%'
		  AND schema_name NOT LIKE 'pg_toast_temp_%'`

	var exists int
	err := i.db.QueryRowContext(ctx, query, schemaName).Scan(&exists)
	if err == sql.ErrNoRows {
		return fmt.Errorf("schema '%s' does not exist in the database", schemaName)
	}
	if err != nil {
		return fmt.Errorf("failed to check if schema '%s' exists: %w", schemaName, err)
	}

	return nil
}

// Helper functions for safe type conversion from interface{}

func (i *Inspector) safeInterfaceToString(val interface{}) string {
	if val == nil {
		return ""
	}
	if sqlVal, ok := val.(sql.NullString); ok {
		if sqlVal.Valid {
			return sqlVal.String
		}
		return ""
	}
	return fmt.Sprintf("%s", val)
}

func (i *Inspector) safeInterfaceToInt(val interface{}, defaultVal int) int {
	if val == nil {
		return defaultVal
	}
	if sqlVal, ok := val.(sql.NullInt64); ok {
		if sqlVal.Valid {
			return int(sqlVal.Int64)
		}
		return defaultVal
	}
	if intVal, ok := val.(int64); ok {
		return int(intVal)
	}
	if intVal, ok := val.(int32); ok {
		return int(intVal)
	}
	if intVal, ok := val.(int); ok {
		return intVal
	}
	return defaultVal
}

func (i *Inspector) safeInterfaceToInt64(val interface{}, defaultVal int64) int64 {
	if val == nil {
		return defaultVal
	}
	if sqlVal, ok := val.(sql.NullInt64); ok {
		if sqlVal.Valid {
			return sqlVal.Int64
		}
		return defaultVal
	}
	if intVal, ok := val.(int64); ok {
		return intVal
	}
	if intVal, ok := val.(int32); ok {
		return int64(intVal)
	}
	if intVal, ok := val.(int); ok {
		return int64(intVal)
	}
	// Handle string types (information_schema.sequences returns numeric values as strings)
	if strVal, ok := val.(string); ok {
		if parsedVal, err := strconv.ParseInt(strVal, 10, 64); err == nil {
			return parsedVal
		}
	}
	return defaultVal
}

func (i *Inspector) safeInterfaceToBool(val interface{}, defaultVal bool) bool {
	if val == nil {
		return defaultVal
	}
	if sqlVal, ok := val.(sql.NullBool); ok {
		if sqlVal.Valid {
			return sqlVal.Bool
		}
		return defaultVal
	}
	if boolVal, ok := val.(bool); ok {
		return boolVal
	}
	if strVal := i.safeInterfaceToString(val); strVal != "" {
		return strVal == "YES" || strVal == "true" || strVal == "t"
	}
	return defaultVal
}
