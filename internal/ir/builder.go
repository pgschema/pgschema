package ir

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pgschema/pgschema/internal/queries"
	"github.com/pgschema/pgschema/internal/version"
)

// Builder builds IR from database queries
type Builder struct {
	db      *sql.DB
	queries *queries.Queries
}

// NewBuilder creates a new schema builder
func NewBuilder(db *sql.DB) *Builder {
	return &Builder{
		db:      db,
		queries: queries.New(db),
	}
}

// BuildIR builds the schema IR from the database for a specific schema
func (b *Builder) BuildIR(ctx context.Context, targetSchema string) (*IR, error) {
	schema := NewIR()

	// Set metadata
	if err := b.buildMetadata(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build metadata: %w", err)
	}

	// Validate target schema exists
	if err := b.validateSchemaExists(ctx, targetSchema); err != nil {
		return nil, err
	}

	// Build schemas (namespaces)
	if err := b.buildSchemas(ctx, schema, targetSchema); err != nil {
		return nil, fmt.Errorf("failed to build schemas: %w", err)
	}

	// Build tables and views
	if err := b.buildTables(ctx, schema, targetSchema); err != nil {
		return nil, fmt.Errorf("failed to build tables: %w", err)
	}

	// Build columns
	if err := b.buildColumns(ctx, schema, targetSchema); err != nil {
		return nil, fmt.Errorf("failed to build columns: %w", err)
	}

	// Build partition information
	if err := b.buildPartitions(ctx, schema, targetSchema); err != nil {
		return nil, fmt.Errorf("failed to build partitions: %w", err)
	}

	// Build partition attachments
	if err := b.buildPartitionAttachments(ctx, schema, targetSchema); err != nil {
		return nil, fmt.Errorf("failed to build partition attachments: %w", err)
	}

	// Build constraints
	if err := b.buildConstraints(ctx, schema, targetSchema); err != nil {
		return nil, fmt.Errorf("failed to build constraints: %w", err)
	}

	// Build indexes
	if err := b.buildIndexes(ctx, schema, targetSchema); err != nil {
		return nil, fmt.Errorf("failed to build indexes: %w", err)
	}

	// Build sequences
	if err := b.buildSequences(ctx, schema, targetSchema); err != nil {
		return nil, fmt.Errorf("failed to build sequences: %w", err)
	}

	// Build functions
	if err := b.buildFunctions(ctx, schema, targetSchema); err != nil {
		return nil, fmt.Errorf("failed to build functions: %w", err)
	}

	// Build procedures
	if err := b.buildProcedures(ctx, schema, targetSchema); err != nil {
		return nil, fmt.Errorf("failed to build procedures: %w", err)
	}

	// Build aggregates
	if err := b.buildAggregates(ctx, schema, targetSchema); err != nil {
		return nil, fmt.Errorf("failed to build aggregates: %w", err)
	}

	// Build views with dependencies
	if err := b.buildViews(ctx, schema, targetSchema); err != nil {
		return nil, fmt.Errorf("failed to build views: %w", err)
	}

	// Build triggers
	if err := b.buildTriggers(ctx, schema, targetSchema); err != nil {
		return nil, fmt.Errorf("failed to build triggers: %w", err)
	}

	// Build RLS policies
	if err := b.buildRLSPolicies(ctx, schema, targetSchema); err != nil {
		return nil, fmt.Errorf("failed to build RLS policies: %w", err)
	}

	// Build extensions
	if err := b.buildExtensions(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build extensions: %w", err)
	}

	// Build types
	if err := b.buildTypes(ctx, schema, targetSchema); err != nil {
		return nil, fmt.Errorf("failed to build types: %w", err)
	}

	return schema, nil
}

func (b *Builder) buildMetadata(ctx context.Context, schema *IR) error {
	var dbVersion string
	if err := b.db.QueryRowContext(ctx, "SELECT version()").Scan(&dbVersion); err != nil {
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
		DumpVersion:     "pgschema version " + version.Version(),
		DumpedAt:        time.Now(),
		Source:          "pgschema",
	}

	return nil
}

func (b *Builder) buildSchemas(ctx context.Context, schema *IR, targetSchema string) error {
	// Use the schema-specific query to prefilter at the database level
	schemaName, err := b.queries.GetSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	name := fmt.Sprintf("%s", schemaName)
	schema.GetOrCreateSchema(name)

	return nil
}

func (b *Builder) buildTables(ctx context.Context, schema *IR, targetSchema string) error {
	tables, err := b.queries.GetTablesForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
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

		dbSchema := schema.GetOrCreateSchema(schemaName)

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

		dbSchema.Tables[tableName] = t
	}

	return nil
}

func (b *Builder) buildColumns(ctx context.Context, schema *IR, targetSchema string) error {
	columns, err := b.queries.GetColumnsForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
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

		dbSchema := schema.GetOrCreateSchema(schemaName)
		table, exists := dbSchema.Tables[tableName]
		if !exists {
			continue // Skip columns for non-existent tables
		}

		// Get the resolved type and strip schema prefix if it matches the current schema
		resolvedType := b.safeInterfaceToString(col.ResolvedType)
		dataType := b.stripSchemaPrefix(resolvedType, targetSchema)

		// Normalize PostgreSQL internal types to SQL standard types
		dataType = b.normalizePostgreSQLType(dataType)

		column := &Column{
			Name:       columnName,
			Position:   b.safeInterfaceToInt(col.OrdinalPosition, 0),
			DataType:   dataType,
			UDTName:    resolvedType,
			IsNullable: fmt.Sprintf("%s", col.IsNullable) == "YES",
			Comment:    comment,
		}

		// Handle default value
		if defaultVal := b.safeInterfaceToString(col.ColumnDefault); defaultVal != "" && defaultVal != "<nil>" {
			column.DefaultValue = &defaultVal
		}

		// Handle max length
		if maxLen := b.safeInterfaceToInt64(col.CharacterMaximumLength, -1); maxLen > 0 {
			maxLenInt := int(maxLen)
			column.MaxLength = &maxLenInt
		}

		// Handle numeric precision and scale
		if precision := b.safeInterfaceToInt64(col.NumericPrecision, -1); precision > 0 {
			precisionInt := int(precision)
			column.Precision = &precisionInt
		}

		if scale := b.safeInterfaceToInt64(col.NumericScale, -1); scale >= 0 {
			scaleInt := int(scale)
			column.Scale = &scaleInt
		}

		// Handle identity columns
		if fmt.Sprintf("%s", col.IsIdentity) == "YES" {
			column.IsIdentity = true
			column.IdentityGeneration = b.safeInterfaceToString(col.IdentityGeneration)

			if start := b.safeInterfaceToInt64(col.IdentityStart, -1); start >= 0 {
				column.IdentityStart = &start
			}

			if increment := b.safeInterfaceToInt64(col.IdentityIncrement, -1); increment >= 0 {
				column.IdentityIncrement = &increment
			}

			if maximum := b.safeInterfaceToInt64(col.IdentityMaximum, -1); maximum >= 0 {
				column.IdentityMaximum = &maximum
			}

			if minimum := b.safeInterfaceToInt64(col.IdentityMinimum, -1); minimum >= 0 {
				column.IdentityMinimum = &minimum
			}

			column.IdentityCycle = fmt.Sprintf("%s", col.IdentityCycle) == "YES"
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

func (b *Builder) buildPartitions(ctx context.Context, schema *IR, targetSchema string) error {
	// Use the schema-specific query to prefilter at the database level
	partitions, err := b.queries.GetPartitionedTablesForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
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

		dbSchema := schema.GetOrCreateSchema(schemaName)
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

func (b *Builder) buildPartitionAttachments(ctx context.Context, schema *IR, targetSchema string) error {
	// Build table partition attachments
	children, err := b.queries.GetPartitionChildren(ctx)
	if err != nil {
		return err
	}

	for _, child := range children {
		parentSchema := fmt.Sprintf("%s", child.ParentSchema)
		childSchema := fmt.Sprintf("%s", child.ChildSchema)

		// Only include attachments where at least one schema matches the target
		if parentSchema != targetSchema && childSchema != targetSchema {
			continue
		}

		attachment := &PartitionAttachment{
			ParentSchema: parentSchema,
			ParentTable:  fmt.Sprintf("%s", child.ParentTable),
			ChildSchema:  childSchema,
			ChildTable:   fmt.Sprintf("%s", child.ChildTable),
			PartitionBound: func() string {
				if child.PartitionBound.Valid {
					return child.PartitionBound.String
				}
				return ""
			}(),
		}
		schema.PartitionAttachments = append(schema.PartitionAttachments, attachment)
	}

	// Build index partition attachments
	indexAttachments, err := b.queries.GetPartitionIndexAttachments(ctx)
	if err != nil {
		return err
	}

	for _, indexAttachment := range indexAttachments {
		parentSchema := fmt.Sprintf("%s", indexAttachment.ParentSchema)
		childSchema := fmt.Sprintf("%s", indexAttachment.ChildSchema)

		// Only include attachments where at least one schema matches the target
		if parentSchema != targetSchema && childSchema != targetSchema {
			continue
		}

		attachment := &IndexAttachment{
			ParentSchema: parentSchema,
			ParentIndex:  fmt.Sprintf("%s", indexAttachment.ParentIndex),
			ChildSchema:  childSchema,
			ChildIndex:   fmt.Sprintf("%s", indexAttachment.ChildIndex),
		}
		schema.IndexAttachments = append(schema.IndexAttachments, attachment)
	}

	return nil
}

func (b *Builder) buildConstraints(ctx context.Context, schema *IR, targetSchema string) error {
	constraints, err := b.queries.GetConstraintsForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
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
		schemaName := fmt.Sprintf("%s", constraint.TableSchema)
		tableName := fmt.Sprintf("%s", constraint.TableName)
		constraintName := fmt.Sprintf("%s", constraint.ConstraintName)
		constraintType := ""
		if constraint.ConstraintType.Valid {
			constraintType = constraint.ConstraintType.String
		}
		columnName := fmt.Sprintf("%s", constraint.ColumnName)

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
				if refSchema := b.safeInterfaceToString(constraint.ForeignTableSchema); refSchema != "" && refSchema != "<nil>" {
					c.ReferencedSchema = refSchema
				}
				if refTable := b.safeInterfaceToString(constraint.ForeignTableName); refTable != "" && refTable != "<nil>" {
					c.ReferencedTable = refTable
				}
				if deleteRule := b.safeInterfaceToString(constraint.DeleteRule); deleteRule != "" && deleteRule != "<nil>" {
					c.DeleteRule = deleteRule
				}
				if updateRule := b.safeInterfaceToString(constraint.UpdateRule); updateRule != "" && updateRule != "<nil>" {
					c.UpdateRule = updateRule
				}
				// Handle deferrable attributes for foreign key constraints
				c.Deferrable = constraint.Deferrable
				c.InitiallyDeferred = constraint.InitiallyDeferred
			}

			// Handle check constraints
			if cType == ConstraintTypeCheck {
				if checkClause := b.safeInterfaceToString(constraint.CheckClause); checkClause != "" && checkClause != "<nil>" {
					// Skip system-generated NOT NULL constraints as they're redundant with column definitions
					if strings.Contains(checkClause, "IS NOT NULL") {
						continue
					}
					c.CheckClause = checkClause
				}
			}

			constraintGroups[key] = c
		}

		// Get column position in constraint
		position := b.getConstraintColumnPosition(ctx, schemaName, constraintName, columnName)

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
			if refColumnName := b.safeInterfaceToString(constraint.ForeignColumnName); refColumnName != "" && refColumnName != "<nil>" {
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
	partitionMapping := b.buildPartitionMapping(ctx, schema, targetSchema)

	// Add constraints to tables
	for key, constraint := range constraintGroups {
		dbSchema := schema.GetOrCreateSchema(key.schema)
		table, exists := dbSchema.Tables[key.table]
		if exists {
			table.Constraints[key.name] = constraint
			
			// For partitioned tables, ensure primary key columns are ordered with partition key first
			if constraint.Type == ConstraintTypePrimaryKey && table.IsPartitioned && table.PartitionKey != "" {
				b.sortPrimaryKeyColumnsForPartitionedTable(constraint, table.PartitionKey)
			}
			
			// For partition tables (children of partitioned tables), use parent's partition key
			if constraint.Type == ConstraintTypePrimaryKey && !table.IsPartitioned {
				if parentPartitionKey, isPartitionTable := partitionMapping[key.table]; isPartitionTable {
					b.sortPrimaryKeyColumnsForPartitionedTable(constraint, parentPartitionKey)
				}
			}
		}
	}

	return nil
}

// buildPartitionMapping builds a mapping from partition table names to their parent's partition keys
func (b *Builder) buildPartitionMapping(ctx context.Context, schema *IR, targetSchema string) map[string]string {
	partitionMapping := make(map[string]string)
	
	// Get partition children information
	partitionChildren, err := b.queries.GetPartitionChildren(ctx)
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
		dbSchema := schema.GetOrCreateSchema(targetSchema)
		if parentTableInfo, exists := dbSchema.Tables[parentTable]; exists && parentTableInfo.IsPartitioned {
			partitionMapping[childTable] = parentTableInfo.PartitionKey
		}
	}
	
	return partitionMapping
}

// sortPrimaryKeyColumnsForPartitionedTable sorts primary key constraint columns
// to ensure partition key columns come first
func (b *Builder) sortPrimaryKeyColumnsForPartitionedTable(constraint *Constraint, partitionKey string) {
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

func (b *Builder) buildIndexes(ctx context.Context, schema *IR, targetSchema string) error {
	// Get indexes for the target schema
	indexes, err := b.queries.GetIndexesForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	for _, indexRow := range indexes {
		schemaName := indexRow.Schemaname
		tableName := indexRow.Tablename
		indexName := indexRow.Indexname

		dbSchema := schema.GetOrCreateSchema(schemaName)

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
		if hasExpressions {
			indexType = IndexTypeExpression
		} else if isUnique {
			indexType = IndexTypeUnique
		}

		index := &Index{
			Schema:     schemaName,
			Table:      tableName,
			Name:       indexName,
			Type:       indexType,
			IsUnique:   isUnique,
			IsPrimary:  isPrimary,
			IsPartial:  isPartial,
			Method:     method,
			Definition: definition,
			Where:      "",
			Columns:    []*IndexColumn{},
		}

		// Set WHERE clause for partial indexes
		if isPartial && indexRow.PartialPredicate.Valid {
			// Use the predicate as-is from pg_get_expr, which already has proper formatting
			index.Where = indexRow.PartialPredicate.String
		}

		// Parse index definition to extract columns
		if err := b.parseIndexDefinition(index); err != nil {
			// If parsing fails, just continue with empty columns
			// This ensures backward compatibility
			continue
		}

		// Store the original definition - simplification will be done during read time in diff module
		index.Definition = definition

		// Add index to table only
		if table, exists := dbSchema.Tables[tableName]; exists {
			table.Indexes[indexName] = index
		}
	}

	return nil
}

// parseIndexDefinition parses an index definition string to extract method and columns
// Expected format: "CREATE [UNIQUE] INDEX index_name ON [schema.]table USING method (column1 [ASC|DESC], column2, ...)"
func (b *Builder) parseIndexDefinition(index *Index) error {
	definition := index.Definition
	if definition == "" {
		return fmt.Errorf("empty index definition")
	}

	// Extract USING method (e.g., "USING btree")
	usingRegex := regexp.MustCompile(`USING\s+(\w+)`)
	if matches := usingRegex.FindStringSubmatch(definition); len(matches) > 1 {
		index.Method = matches[1]
	} else {
		// Default to btree if not specified
		index.Method = "btree"
	}

	// Extract columns from parentheses - handle nested parentheses properly
	// Find the column list parentheses after USING method
	usingPos := strings.Index(definition, "USING")
	if usingPos == -1 {
		return fmt.Errorf("USING clause not found in index definition")
	}

	// Find the opening parenthesis after USING method
	parenStart := strings.Index(definition[usingPos:], "(")
	if parenStart == -1 {
		return fmt.Errorf("column list not found in index definition")
	}
	parenStart += usingPos

	// Find the matching closing parenthesis, handling nesting
	parenCount := 0
	parenEnd := -1
	for i := parenStart; i < len(definition); i++ {
		if definition[i] == '(' {
			parenCount++
		} else if definition[i] == ')' {
			parenCount--
			if parenCount == 0 {
				parenEnd = i
				break
			}
		}
	}

	if parenEnd == -1 {
		return fmt.Errorf("unmatched parentheses in index definition")
	}

	// Extract the content between parentheses
	columnsStr := definition[parenStart+1 : parenEnd]

	// Split by commas and parse each column
	columnParts := strings.Split(columnsStr, ",")
	for i, part := range columnParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Parse column name and direction
		columnName, direction := b.parseIndexColumnDefinition(part)

		indexColumn := &IndexColumn{
			Name:      columnName,
			Position:  i + 1,
			Direction: direction,
		}

		index.Columns = append(index.Columns, indexColumn)
	}

	return nil
}

// parseIndexColumnDefinition parses a single column definition from index
// Input: "column_name ASC" or "column_name DESC" or just "column_name"
// Or for expressions: "(expression) ASC" or "((expression ->> 'key'::text)) DESC"
// Returns: column name and direction
func (b *Builder) parseIndexColumnDefinition(columnDef string) (string, string) {
	columnDef = strings.TrimSpace(columnDef)
	if columnDef == "" {
		return "", "ASC"
	}

	direction := "ASC" // Default direction
	var columnName string

	// Handle expressions like "((payload ->> 'method'::text))" or "(expression)"
	if strings.HasPrefix(columnDef, "(") {
		// Find the matching closing parenthesis for the expression
		parenCount := 0
		exprEnd := -1
		for i, ch := range columnDef {
			if ch == '(' {
				parenCount++
			} else if ch == ')' {
				parenCount--
				if parenCount == 0 {
					exprEnd = i
					break
				}
			}
		}

		if exprEnd > 0 {
			// Extract the full expression including parentheses
			columnName = columnDef[:exprEnd+1]

			// Check for direction after the expression
			remainder := strings.TrimSpace(columnDef[exprEnd+1:])
			if remainder != "" {
				parts := strings.Fields(remainder)
				if len(parts) > 0 {
					directionStr := strings.ToUpper(parts[0])
					if directionStr == "DESC" {
						direction = "DESC"
					}
				}
			}

			// For expression indexes, extract and simplify the expression to match parser format
			if strings.Contains(columnName, "->") || strings.Contains(columnName, "->>") {
				// Extract and simplify JSON expressions to match parser output
				columnName = b.simplifyColumnExpression(columnName)
			}
		} else {
			// Malformed expression, just use as-is
			columnName = columnDef
		}
	} else {
		// Regular column name
		parts := strings.Fields(columnDef)
		columnName = parts[0]

		if len(parts) > 1 {
			directionStr := strings.ToUpper(parts[1])
			if directionStr == "DESC" {
				direction = "DESC"
			}
		}
	}

	return columnName, direction
}

func (b *Builder) buildSequences(ctx context.Context, schema *IR, targetSchema string) error {
	sequences, err := b.queries.GetSequencesForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	for _, seq := range sequences {
		schemaName := fmt.Sprintf("%s", seq.SequenceSchema)
		sequenceName := fmt.Sprintf("%s", seq.SequenceName)

		dbSchema := schema.GetOrCreateSchema(schemaName)

		sequence := &Sequence{
			Schema:      schemaName,
			Name:        sequenceName,
			DataType:    fmt.Sprintf("%s", seq.DataType),
			StartValue:  b.safeInterfaceToInt64(seq.StartValue, 1),
			Increment:   b.safeInterfaceToInt64(seq.Increment, 1),
			CycleOption: b.safeInterfaceToBool(seq.CycleOption, false),
		}

		if minVal := b.safeInterfaceToInt64(seq.MinimumValue, -1); minVal > -1 {
			sequence.MinValue = &minVal
		}

		if maxVal := b.safeInterfaceToInt64(seq.MaximumValue, -1); maxVal > -1 {
			sequence.MaxValue = &maxVal
		}

		// Set ownership information
		if seq.OwnedByTable.Valid {
			sequence.OwnedByTable = seq.OwnedByTable.String
		}
		if seq.OwnedByColumn.Valid {
			sequence.OwnedByColumn = seq.OwnedByColumn.String
		}

		dbSchema.Sequences[sequenceName] = sequence
	}

	return nil
}

func (b *Builder) buildFunctions(ctx context.Context, schema *IR, targetSchema string) error {
	functions, err := b.queries.GetFunctionsForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
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
		arguments := b.safeInterfaceToString(fn.FunctionArguments)
		signature := b.safeInterfaceToString(fn.FunctionSignature)

		dbSchema := schema.GetOrCreateSchema(schemaName)

		// Handle volatility
		volatility := b.safeInterfaceToString(fn.Volatility)

		// Handle strictness
		isStrict := fn.IsStrict

		// Handle security definer
		isSecurityDefiner := fn.IsSecurityDefiner

		function := &Function{
			Schema:            schemaName,
			Name:              functionName,
			Definition:        fmt.Sprintf("%s", fn.RoutineDefinition),
			ReturnType:        b.safeInterfaceToString(fn.DataType),
			Language:          fmt.Sprintf("%s", fn.ExternalLanguage),
			Arguments:         arguments,
			Signature:         signature,
			Comment:           comment,
			Parameters:        []*Parameter{}, // TODO: parse parameters
			Volatility:        volatility,
			IsStrict:          isStrict,
			IsSecurityDefiner: isSecurityDefiner,
		}

		dbSchema.Functions[functionName] = function
	}

	return nil
}

func (b *Builder) buildProcedures(ctx context.Context, schema *IR, targetSchema string) error {
	procedures, err := b.queries.GetProceduresForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
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
		arguments := b.safeInterfaceToString(proc.ProcedureArguments)
		signature := b.safeInterfaceToString(proc.ProcedureSignature)

		dbSchema := schema.GetOrCreateSchema(schemaName)

		procedure := &Procedure{
			Schema:     schemaName,
			Name:       procedureName,
			Definition: fmt.Sprintf("%s", proc.RoutineDefinition),
			Language:   fmt.Sprintf("%s", proc.ExternalLanguage),
			Arguments:  arguments,
			Signature:  signature,
			Comment:    comment,
			Parameters: []*Parameter{}, // TODO: parse parameters
		}

		dbSchema.Procedures[procedureName] = procedure
	}

	return nil
}

func (b *Builder) buildAggregates(ctx context.Context, schema *IR, targetSchema string) error {
	aggregates, err := b.queries.GetAggregatesForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	for _, agg := range aggregates {
		schemaName := fmt.Sprintf("%s", agg.AggregateSchema)
		aggregateName := fmt.Sprintf("%s", agg.AggregateName)
		comment := ""
		if agg.AggregateComment.Valid {
			comment = agg.AggregateComment.String
		}
		arguments := b.safeInterfaceToString(agg.AggregateArguments)
		signature := b.safeInterfaceToString(agg.AggregateSignature)
		returnType := b.safeInterfaceToString(agg.AggregateReturnType)
		transitionFunction := b.safeInterfaceToString(agg.TransitionFunction)
		transitionFunctionSchema := b.safeInterfaceToString(agg.TransitionFunctionSchema)
		stateType := b.safeInterfaceToString(agg.StateType)
		initialCondition := b.safeInterfaceToString(agg.InitialCondition)
		finalFunction := b.safeInterfaceToString(agg.FinalFunction)
		finalFunctionSchema := b.safeInterfaceToString(agg.FinalFunctionSchema)

		dbSchema := schema.GetOrCreateSchema(schemaName)

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

		dbSchema.Aggregates[aggregateName] = aggregate
	}

	return nil
}

func (b *Builder) buildViews(ctx context.Context, schema *IR, targetSchema string) error {
	views, err := b.queries.GetViewsForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	for _, view := range views {
		schemaName := fmt.Sprintf("%s", view.TableSchema)
		viewName := fmt.Sprintf("%s", view.TableName)
		comment := ""
		if view.ViewComment.Valid {
			comment = view.ViewComment.String
		}

		dbSchema := schema.GetOrCreateSchema(schemaName)

		v := &View{
			Schema:       schemaName,
			Name:         viewName,
			Definition:   fmt.Sprintf("%s", view.ViewDefinition),
			Comment:      comment,
			Dependencies: []TableDependency{}, // TODO: parse dependencies
		}

		dbSchema.Views[viewName] = v
	}

	return nil
}

func (b *Builder) buildTriggers(ctx context.Context, schema *IR, targetSchema string) error {
	triggers, err := b.queries.GetTriggersForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
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

			t = &Trigger{
				Schema:   schemaName,
				Table:    tableName,
				Name:     triggerName,
				Timing:   tTiming,
				Events:   []TriggerEvent{},
				Level:    TriggerLevelRow, // Assuming ROW level for now
				Function: b.extractFunctionFromStatement(statement),
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
		dbSchema := schema.GetOrCreateSchema(key.schema)

		if table, exists := dbSchema.Tables[key.table]; exists {
			table.Triggers[key.name] = trigger
		}
	}

	return nil
}

func (b *Builder) buildRLSPolicies(ctx context.Context, schema *IR, targetSchema string) error {
	// Get RLS enabled tables for the target schema
	rlsTables, err := b.queries.GetRLSTablesForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
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

		dbSchema := schema.GetOrCreateSchema(schemaName)
		if table, exists := dbSchema.Tables[tableName]; exists {
			table.RLSEnabled = true
		}
	}

	// Get RLS policies for the target schema
	policies, err := b.queries.GetRLSPoliciesForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
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

		dbSchema := schema.GetOrCreateSchema(schemaName)
		dbSchema.Policies[policyName] = policy

		if table, exists := dbSchema.Tables[tableName]; exists {
			table.Policies[policyName] = policy
		}
	}

	return nil
}

func (b *Builder) buildExtensions(ctx context.Context, schema *IR) error {
	extensions, err := b.queries.GetExtensions(ctx)
	if err != nil {
		return err
	}

	for _, ext := range extensions {
		extensionName := fmt.Sprintf("%s", ext.ExtensionName)
		schemaName := fmt.Sprintf("%s", ext.SchemaName)
		version := fmt.Sprintf("%s", ext.ExtensionVersion)
		comment := ""
		if ext.ExtensionComment.Valid {
			comment = ext.ExtensionComment.String
		}

		extension := &Extension{
			Name:    extensionName,
			Schema:  schemaName,
			Version: version,
			Comment: comment,
		}

		schema.Extensions[extensionName] = extension
	}

	return nil
}

func (b *Builder) buildTypes(ctx context.Context, schema *IR, targetSchema string) error {
	types, err := b.queries.GetTypesForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	// Get domains
	domains, err := b.queries.GetDomainsForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	// Get domain constraints
	domainConstraints, err := b.queries.GetDomainConstraintsForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	// Get enum values for ENUM types
	enumValues, err := b.queries.GetEnumValuesForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
	if err != nil {
		return err
	}

	// Get columns for composite types
	compositeColumns, err := b.queries.GetCompositeTypeColumnsForSchema(ctx, sql.NullString{String: targetSchema, Valid: true})
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
		position := b.safeInterfaceToInt(col.ColumnPosition, 0)

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
		key := fmt.Sprintf("%s.%s", b.safeInterfaceToString(constraint.DomainSchema), b.safeInterfaceToString(constraint.DomainName))
		constraintName := b.safeInterfaceToString(constraint.ConstraintName)
		constraintDef := b.safeInterfaceToString(constraint.ConstraintDefinition)

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

		dbSchema := schema.GetOrCreateSchema(schemaName)

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

		dbSchema.Types[typeName] = customType
	}

	// Create domains
	for _, d := range domains {
		schemaName := b.safeInterfaceToString(d.DomainSchema)
		domainName := b.safeInterfaceToString(d.DomainName)
		baseType := b.safeInterfaceToString(d.BaseType)
		notNull := b.safeInterfaceToBool(d.NotNull, false)
		defaultValue := b.safeInterfaceToString(d.DefaultValue)
		comment := ""
		if d.DomainComment.Valid {
			comment = d.DomainComment.String
		}

		dbSchema := schema.GetOrCreateSchema(schemaName)

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

		dbSchema.Types[domainName] = domainType
	}

	return nil
}

// Helper methods

func (b *Builder) getConstraintColumnPosition(ctx context.Context, schemaName, constraintName, columnName string) int {
	query := `
		SELECT kcu.ordinal_position
		FROM information_schema.key_column_usage kcu
		WHERE kcu.table_schema = $1
		  AND kcu.constraint_name = $2
		  AND kcu.column_name = $3`

	var position int
	err := b.db.QueryRowContext(ctx, query, schemaName, constraintName, columnName).Scan(&position)
	if err != nil {
		return 0 // Default position if query fails
	}

	return position
}

func (b *Builder) extractFunctionFromStatement(statement string) string {
	// Extract complete function call from "EXECUTE FUNCTION function_name(...)"
	if strings.Contains(statement, "EXECUTE FUNCTION ") {
		parts := strings.Split(statement, "EXECUTE FUNCTION ")
		if len(parts) > 1 {
			funcPart := strings.TrimSpace(parts[1])
			// Return the complete function call including parameters
			return funcPart
		}
	}
	return statement
}

// validateSchemaExists checks if a schema exists in the database
func (b *Builder) validateSchemaExists(ctx context.Context, schemaName string) error {
	query := `
		SELECT 1 
		FROM information_schema.schemata 
		WHERE schema_name = $1
		  AND schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
		  AND schema_name NOT LIKE 'pg_temp_%'
		  AND schema_name NOT LIKE 'pg_toast_temp_%'`

	var exists int
	err := b.db.QueryRowContext(ctx, query, schemaName).Scan(&exists)
	if err == sql.ErrNoRows {
		return fmt.Errorf("schema '%s' does not exist in the database", schemaName)
	}
	if err != nil {
		return fmt.Errorf("failed to check if schema '%s' exists: %w", schemaName, err)
	}

	return nil
}

// Helper functions for safe type conversion from interface{}

func (b *Builder) safeInterfaceToString(val interface{}) string {
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

func (b *Builder) safeInterfaceToInt(val interface{}, defaultVal int) int {
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

func (b *Builder) safeInterfaceToInt64(val interface{}, defaultVal int64) int64 {
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
	return defaultVal
}

func (b *Builder) safeInterfaceToBool(val interface{}, defaultVal bool) bool {
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
	if strVal := b.safeInterfaceToString(val); strVal != "" {
		return strVal == "YES" || strVal == "true" || strVal == "t"
	}
	return defaultVal
}

// stripSchemaPrefix removes the schema prefix from a type name if it matches the target schema
func (b *Builder) stripSchemaPrefix(typeName, targetSchema string) string {
	if typeName == "" {
		return typeName
	}

	// Check if the type has a schema prefix
	prefix := targetSchema + "."
	if strings.HasPrefix(typeName, prefix) {
		return strings.TrimPrefix(typeName, prefix)
	}

	return typeName
}

// simplifyColumnExpression simplifies a column expression to match parser format
// Example: "((payload ->> 'method'::text))" -> "(payload->>'method')"
func (b *Builder) simplifyColumnExpression(expression string) string {
	// Remove ::text type casts
	simplified := strings.ReplaceAll(expression, "::text", "")

	// Remove unnecessary outer parentheses layers using generic approach
	simplified = b.removeExtraParentheses(simplified)

	// Remove spaces around JSON operators for consistency
	simplified = strings.ReplaceAll(simplified, " ->> ", "->>")
	simplified = strings.ReplaceAll(simplified, " -> ", "->")

	return simplified
}

// removeExtraParentheses removes unnecessary outer parentheses layers from an expression
// while preserving the core expression. It handles nested parentheses properly.
// Example: "((expression))" -> "(expression)", "(((a + b)))" -> "(a + b)"
func (b *Builder) removeExtraParentheses(expression string) string {
	if len(expression) < 2 {
		return expression
	}

	// Keep removing outer parentheses as long as:
	// 1. The expression starts and ends with parentheses
	// 2. The opening parenthesis at position 0 matches the closing parenthesis at the end
	// 3. Removing them doesn't break the expression structure
	for len(expression) >= 2 && expression[0] == '(' && expression[len(expression)-1] == ')' {
		// Check if the first '(' matches the last ')' (i.e., they form the outermost pair)
		parenCount := 0
		matchesOutermost := false

		for i := 0; i < len(expression); i++ {
			if expression[i] == '(' {
				parenCount++
			} else if expression[i] == ')' {
				parenCount--
				// If we reach 0 at the last character, the first and last parentheses are paired
				if parenCount == 0 && i == len(expression)-1 {
					matchesOutermost = true
					break
				} else if parenCount == 0 {
					// We found a closing parenthesis that pairs with an earlier opening one
					// This means the first '(' doesn't pair with the last ')'
					break
				}
			}
		}

		// Only remove the outer parentheses if they form a complete pair around the entire expression
		if matchesOutermost {
			expression = expression[1 : len(expression)-1]
		} else {
			break
		}
	}

	return expression
}

// normalizePostgreSQLType converts PostgreSQL internal type names to SQL standard names
func (b *Builder) normalizePostgreSQLType(typeName string) string {
	typeMap := map[string]string{
		// Numeric types
		"int2":   "smallint",
		"int4":   "integer",
		"int8":   "bigint",
		"float4": "real",
		"float8": "double precision",
		"bool":   "boolean",

		// Character types
		"bpchar":  "character",
		"varchar": "character varying",

		// Date/time types
		"timestamp with time zone": "timestamptz", // Convert to abbreviated form
		"time with time zone":      "timetz",      // Convert to abbreviated form
		"timestamptz":              "timestamptz", // Keep canonical form
		"timetz":                   "timetz",      // Keep canonical form

		// Array notation
		"_text": "text[]",
		"_int4": "integer[]",
		"_int2": "smallint[]",
	}

	if normalized, exists := typeMap[typeName]; exists {
		return normalized
	}

	return typeName
}
