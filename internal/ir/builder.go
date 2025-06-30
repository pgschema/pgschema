package ir

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pgschema/pgschema/internal/queries"
	"github.com/pgschema/pgschema/internal/version"
)

// Builder builds schema IR from database queries
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

// BuildSchema builds the complete schema IR from the database
func (b *Builder) BuildSchema(ctx context.Context) (*Schema, error) {
	schema := NewSchema()

	// Set metadata
	if err := b.buildMetadata(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build metadata: %w", err)
	}

	// Build schemas (namespaces)
	if err := b.buildSchemas(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build schemas: %w", err)
	}

	// Build tables and views
	if err := b.buildTables(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build tables: %w", err)
	}

	// Build columns
	if err := b.buildColumns(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build columns: %w", err)
	}

	// Build partition information
	if err := b.buildPartitions(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build partitions: %w", err)
	}

	// Build partition attachments
	if err := b.buildPartitionAttachments(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build partition attachments: %w", err)
	}

	// Build constraints
	if err := b.buildConstraints(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build constraints: %w", err)
	}

	// Build indexes
	if err := b.buildIndexes(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build indexes: %w", err)
	}

	// Build sequences
	if err := b.buildSequences(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build sequences: %w", err)
	}

	// Build functions
	if err := b.buildFunctions(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build functions: %w", err)
	}

	// Build procedures
	if err := b.buildProcedures(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build procedures: %w", err)
	}

	// Build aggregates
	if err := b.buildAggregates(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build aggregates: %w", err)
	}

	// Build views with dependencies
	if err := b.buildViews(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build views: %w", err)
	}

	// Build triggers
	if err := b.buildTriggers(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build triggers: %w", err)
	}

	// Build RLS policies
	if err := b.buildRLSPolicies(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build RLS policies: %w", err)
	}

	// Build extensions
	if err := b.buildExtensions(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build extensions: %w", err)
	}

	// Build types
	if err := b.buildTypes(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to build types: %w", err)
	}

	return schema, nil
}

func (b *Builder) buildMetadata(ctx context.Context, schema *Schema) error {
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

func (b *Builder) buildSchemas(ctx context.Context, schema *Schema) error {
	schemaNames, err := b.queries.GetSchemas(ctx)
	if err != nil {
		return err
	}

	for _, schemaName := range schemaNames {
		name := fmt.Sprintf("%s", schemaName)
		schema.GetOrCreateSchema(name)
	}

	return nil
}

func (b *Builder) buildTables(ctx context.Context, schema *Schema) error {
	tables, err := b.queries.GetTables(ctx)
	if err != nil {
		return err
	}

	for _, table := range tables {
		schemaName := fmt.Sprintf("%s", table.TableSchema)
		tableName := fmt.Sprintf("%s", table.TableName)
		tableType := fmt.Sprintf("%s", table.TableType)
		comment := b.safeInterfaceToString(table.TableComment)

		dbSchema := schema.GetOrCreateSchema(schemaName)

		var tType TableType
		switch tableType {
		case "BASE TABLE":
			tType = TableTypeBase
		case "VIEW":
			tType = TableTypeView
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

func (b *Builder) buildColumns(ctx context.Context, schema *Schema) error {
	columns, err := b.queries.GetColumns(ctx)
	if err != nil {
		return err
	}

	for _, col := range columns {
		schemaName := fmt.Sprintf("%s", col.TableSchema)
		tableName := fmt.Sprintf("%s", col.TableName)
		columnName := fmt.Sprintf("%s", col.ColumnName)
		comment := b.safeInterfaceToString(col.ColumnComment)

		dbSchema := schema.GetOrCreateSchema(schemaName)
		table, exists := dbSchema.Tables[tableName]
		if !exists {
			continue // Skip columns for non-existent tables
		}

		column := &Column{
			Name:       columnName,
			Position:   b.safeInterfaceToInt(col.OrdinalPosition, 0),
			DataType:   fmt.Sprintf("%s", col.DataType),
			UDTName:    b.safeInterfaceToString(col.ResolvedType),
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

		table.Columns = append(table.Columns, column)
	}

	return nil
}

func (b *Builder) buildPartitions(ctx context.Context, schema *Schema) error {
	partitions, err := b.queries.GetPartitionedTables(ctx)
	if err != nil {
		return err
	}

	for _, partition := range partitions {
		schemaName := fmt.Sprintf("%s", partition.TableSchema)
		tableName := fmt.Sprintf("%s", partition.TableName)
		partitionStrategy := fmt.Sprintf("%s", partition.PartitionStrategy)
		partitionKey := b.safeStringPointerToString(partition.PartitionKey)

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

func (b *Builder) buildPartitionAttachments(ctx context.Context, schema *Schema) error {
	// Build table partition attachments
	children, err := b.queries.GetPartitionChildren(ctx)
	if err != nil {
		return err
	}

	for _, child := range children {
		attachment := &PartitionAttachment{
			ParentSchema:   fmt.Sprintf("%s", child.ParentSchema),
			ParentTable:    fmt.Sprintf("%s", child.ParentTable),
			ChildSchema:    fmt.Sprintf("%s", child.ChildSchema),
			ChildTable:     fmt.Sprintf("%s", child.ChildTable),
			PartitionBound: b.safeStringPointerToString(child.PartitionBound),
		}
		schema.PartitionAttachments = append(schema.PartitionAttachments, attachment)
	}

	// Build index partition attachments
	indexAttachments, err := b.queries.GetPartitionIndexAttachments(ctx)
	if err != nil {
		return err
	}

	for _, indexAttachment := range indexAttachments {
		attachment := &IndexAttachment{
			ParentSchema: fmt.Sprintf("%s", indexAttachment.ParentSchema),
			ParentIndex:  fmt.Sprintf("%s", indexAttachment.ParentIndex),
			ChildSchema:  fmt.Sprintf("%s", indexAttachment.ChildSchema),
			ChildIndex:   fmt.Sprintf("%s", indexAttachment.ChildIndex),
		}
		schema.IndexAttachments = append(schema.IndexAttachments, attachment)
	}

	return nil
}

func (b *Builder) buildConstraints(ctx context.Context, schema *Schema) error {
	constraints, err := b.queries.GetConstraints(ctx)
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
		constraintType := fmt.Sprintf("%s", constraint.ConstraintType)
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
				if deferrable := constraint.Deferrable; deferrable != nil {
					if deferrableBool, ok := deferrable.(bool); ok {
						c.Deferrable = deferrableBool
					}
				}
				if initiallyDeferred := constraint.InitiallyDeferred; initiallyDeferred != nil {
					if initiallyDeferredBool, ok := initiallyDeferred.(bool); ok {
						c.InitiallyDeferred = initiallyDeferredBool
					}
				}
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
					if constraint.ForeignOrdinalPosition != nil {
						if foreignOrdinalPos, ok := constraint.ForeignOrdinalPosition.(int32); ok {
							refPosition = int(foreignOrdinalPos)
						} else if foreignOrdinalPos, ok := constraint.ForeignOrdinalPosition.(int); ok {
							refPosition = foreignOrdinalPos
						}
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

	// Add constraints to tables
	for key, constraint := range constraintGroups {
		dbSchema := schema.GetOrCreateSchema(key.schema)
		table, exists := dbSchema.Tables[key.table]
		if exists {
			table.Constraints[key.name] = constraint
		}
	}

	return nil
}

func (b *Builder) buildIndexes(ctx context.Context, schema *Schema) error {
	// Use enhanced query to extract comprehensive index information
	query := `
		SELECT n.nspname as schemaname,
		       t.relname as tablename,
		       i.relname as indexname,
		       idx.indisunique as is_unique,
		       idx.indisprimary as is_primary,
		       (idx.indpred IS NOT NULL) as is_partial,
		       am.amname as method,
		       pg_get_indexdef(idx.indexrelid) as indexdef,
		       CASE 
		           WHEN idx.indpred IS NOT NULL THEN pg_get_expr(idx.indpred, idx.indrelid)
		           ELSE NULL
		       END as partial_predicate,
		       CASE 
		           WHEN idx.indexprs IS NOT NULL THEN true
		           ELSE false
		       END as has_expressions
		FROM pg_index idx
		JOIN pg_class i ON i.oid = idx.indexrelid
		JOIN pg_class t ON t.oid = idx.indrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_am am ON am.oid = i.relam
		WHERE NOT idx.indisprimary
		  AND NOT EXISTS (
		      SELECT 1 FROM pg_constraint c 
		      WHERE c.conindid = idx.indexrelid 
		      AND c.contype IN ('u', 'p')
		  )
		  AND n.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
		  AND n.nspname NOT LIKE 'pg_temp_%'
		  AND n.nspname NOT LIKE 'pg_toast_temp_%'
		ORDER BY n.nspname, t.relname, i.relname`

	rows, err := b.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, tableName, indexName, definition, method string
		var partialPredicate *string
		var isUnique, isPrimary, isPartial, hasExpressions bool
		if err := rows.Scan(&schemaName, &tableName, &indexName, &isUnique, &isPrimary, &isPartial, &method, &definition, &partialPredicate, &hasExpressions); err != nil {
			return err
		}

		dbSchema := schema.GetOrCreateSchema(schemaName)

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
		if isPartial && partialPredicate != nil {
			// Add parentheses to match parser output format
			index.Where = "(" + *partialPredicate + ")"
		}

		// Parse index definition to extract columns
		if err := b.parseIndexDefinition(index); err != nil {
			// If parsing fails, just continue with empty columns
			// This ensures backward compatibility
			continue
		}

		dbSchema.Indexes[indexName] = index

		// Also add to table if it exists
		if table, exists := dbSchema.Tables[tableName]; exists {
			table.Indexes[indexName] = index
		}
	}

	return rows.Err()
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
			
			// For expression indexes, use a simplified name for compatibility
			if strings.Contains(columnName, "->") || strings.Contains(columnName, "->>") {
				// This is a JSON expression, use a generic name to match parser output
				columnName = "(payload ->> (expression))"
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

func (b *Builder) buildSequences(ctx context.Context, schema *Schema) error {
	sequences, err := b.queries.GetSequences(ctx)
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

		dbSchema.Sequences[sequenceName] = sequence
	}

	return nil
}

func (b *Builder) buildFunctions(ctx context.Context, schema *Schema) error {
	functions, err := b.queries.GetFunctions(ctx)
	if err != nil {
		return err
	}

	for _, fn := range functions {
		schemaName := fmt.Sprintf("%s", fn.RoutineSchema)
		functionName := fmt.Sprintf("%s", fn.RoutineName)
		comment := b.safeInterfaceToString(fn.FunctionComment)
		arguments := b.safeInterfaceToString(fn.FunctionArguments)
		signature := b.safeInterfaceToString(fn.FunctionSignature)

		dbSchema := schema.GetOrCreateSchema(schemaName)

		// Handle volatility
		volatility := b.safeInterfaceToString(fn.Volatility)

		// Handle strictness
		isStrict := false
		if fn.IsStrict != nil {
			if strictBool, ok := fn.IsStrict.(bool); ok {
				isStrict = strictBool
			}
		}

		// Handle security definer
		isSecurityDefiner := false
		if fn.IsSecurityDefiner != nil {
			if secDefBool, ok := fn.IsSecurityDefiner.(bool); ok {
				isSecurityDefiner = secDefBool
			}
		}

		function := &Function{
			Schema:            schemaName,
			Name:              functionName,
			Definition:        fmt.Sprintf("%s", fn.RoutineDefinition),
			ReturnType:        fmt.Sprintf("%s", fn.DataType),
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

func (b *Builder) buildProcedures(ctx context.Context, schema *Schema) error {
	procedures, err := b.queries.GetProcedures(ctx)
	if err != nil {
		return err
	}

	for _, proc := range procedures {
		schemaName := fmt.Sprintf("%s", proc.RoutineSchema)
		procedureName := fmt.Sprintf("%s", proc.RoutineName)
		comment := b.safeInterfaceToString(proc.ProcedureComment)
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

func (b *Builder) buildAggregates(ctx context.Context, schema *Schema) error {
	aggregates, err := b.queries.GetAggregates(ctx)
	if err != nil {
		return err
	}

	for _, agg := range aggregates {
		schemaName := fmt.Sprintf("%s", agg.AggregateSchema)
		aggregateName := fmt.Sprintf("%s", agg.AggregateName)
		comment := b.safeInterfaceToString(agg.AggregateComment)
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

func (b *Builder) buildViews(ctx context.Context, schema *Schema) error {
	views, err := b.queries.GetViews(ctx)
	if err != nil {
		return err
	}

	for _, view := range views {
		schemaName := fmt.Sprintf("%s", view.TableSchema)
		viewName := fmt.Sprintf("%s", view.TableName)
		comment := b.safeInterfaceToString(view.ViewComment)

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

func (b *Builder) buildTriggers(ctx context.Context, schema *Schema) error {
	triggers, err := b.queries.GetTriggers(ctx)
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

	// Add triggers to schema and tables
	for key, trigger := range triggerGroups {
		dbSchema := schema.GetOrCreateSchema(key.schema)
		// Use table.trigger format as key to ensure uniqueness across tables
		triggerKey := fmt.Sprintf("%s.%s", key.table, key.name)
		dbSchema.Triggers[triggerKey] = trigger

		if table, exists := dbSchema.Tables[key.table]; exists {
			table.Triggers[key.name] = trigger
		}
	}

	return nil
}

func (b *Builder) buildRLSPolicies(ctx context.Context, schema *Schema) error {
	// Check RLS enabled tables
	rlsQuery := `
		SELECT n.nspname AS schemaname, c.relname AS tablename
		FROM pg_class c
		JOIN pg_namespace n ON c.relnamespace = n.oid
		WHERE c.relkind = 'r'
		  AND n.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
		  AND n.nspname NOT LIKE 'pg_temp_%'
		  AND n.nspname NOT LIKE 'pg_toast_temp_%'
		  AND c.relrowsecurity = true
		ORDER BY n.nspname, c.relname`

	rows, err := b.db.QueryContext(ctx, rlsQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, tableName string
		if err := rows.Scan(&schemaName, &tableName); err != nil {
			return err
		}

		dbSchema := schema.GetOrCreateSchema(schemaName)
		if table, exists := dbSchema.Tables[tableName]; exists {
			table.RLSEnabled = true
		}
	}

	// Get RLS policies
	policyQuery := `
		SELECT n.nspname AS schemaname,
		       c.relname AS tablename,
		       pol.polname AS policyname,
		       pol.polcmd AS cmd,
		       pg_get_expr(pol.polqual, pol.polrelid) AS qual,
		       pg_get_expr(pol.polwithcheck, pol.polrelid) AS with_check
		FROM pg_policy pol
		JOIN pg_class c ON pol.polrelid = c.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		WHERE n.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
		  AND n.nspname NOT LIKE 'pg_temp_%'
		  AND n.nspname NOT LIKE 'pg_toast_temp_%'
		ORDER BY n.nspname, c.relname, pol.polname`

	rows, err = b.db.QueryContext(ctx, policyQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var schemaName, tableName, policyName string
		var cmd sql.NullString
		var qual, withCheck sql.NullString

		if err := rows.Scan(&schemaName, &tableName, &policyName, &cmd, &qual, &withCheck); err != nil {
			return err
		}

		var pCommand PolicyCommand
		if cmd.Valid {
			switch cmd.String {
			case "r":
				pCommand = PolicyCommandSelect
			case "a":
				pCommand = PolicyCommandInsert
			case "w":
				pCommand = PolicyCommandUpdate
			case "d":
				pCommand = PolicyCommandDelete
			case "*":
				pCommand = PolicyCommandAll
			default:
				pCommand = PolicyCommandAll
			}
		}

		policy := &RLSPolicy{
			Schema:     schemaName,
			Table:      tableName,
			Name:       policyName,
			Command:    pCommand,
			Permissive: true, // Assuming permissive for now
		}

		if qual.Valid {
			policy.Using = qual.String
		}

		if withCheck.Valid {
			policy.WithCheck = withCheck.String
		}

		dbSchema := schema.GetOrCreateSchema(schemaName)
		dbSchema.Policies[policyName] = policy

		if table, exists := dbSchema.Tables[tableName]; exists {
			table.Policies[policyName] = policy
		}
	}

	return nil
}

func (b *Builder) buildExtensions(ctx context.Context, schema *Schema) error {
	extensions, err := b.queries.GetExtensions(ctx)
	if err != nil {
		return err
	}

	for _, ext := range extensions {
		extensionName := fmt.Sprintf("%s", ext.ExtensionName)
		schemaName := fmt.Sprintf("%s", ext.SchemaName)
		version := fmt.Sprintf("%s", ext.ExtensionVersion)
		comment := b.safeInterfaceToString(ext.ExtensionComment)

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

func (b *Builder) buildTypes(ctx context.Context, schema *Schema) error {
	types, err := b.queries.GetTypes(ctx)
	if err != nil {
		return err
	}

	// Get domains
	domains, err := b.queries.GetDomains(ctx)
	if err != nil {
		return err
	}

	// Get domain constraints
	domainConstraints, err := b.queries.GetDomainConstraints(ctx)
	if err != nil {
		return err
	}

	// Get enum values for ENUM types
	enumValues, err := b.queries.GetEnumValues(ctx)
	if err != nil {
		return err
	}

	// Get columns for composite types
	compositeColumns, err := b.queries.GetCompositeTypeColumns(ctx)
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

		typeCol := &TypeColumn{
			Name:     col.ColumnName,
			DataType: col.ColumnType,
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
		typeKind := TypeKind(t.TypeKind)
		comment := b.safeInterfaceToString(t.TypeComment)

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
		comment := b.safeInterfaceToString(d.DomainComment)

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

// safeStringPointerToString safely converts a string pointer to string
func (b *Builder) safeStringPointerToString(val *string) string {
	if val == nil {
		return ""
	}
	return *val
}
