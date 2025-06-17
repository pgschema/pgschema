package ir

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pgschema/pgschema/internal/queries"
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
	
	// Infer sequence ownership from column defaults
	if err := b.inferSequenceOwnership(ctx, schema); err != nil {
		return nil, fmt.Errorf("failed to infer sequence ownership: %w", err)
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
		DumpVersion:     "pgschema version 0.0.1", // TODO: get from build info
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
		
		dbSchema := schema.GetOrCreateSchema(schemaName)
		table, exists := dbSchema.Tables[tableName]
		if !exists {
			continue // Skip columns for non-existent tables
		}
		
		column := &Column{
			Name:       columnName,
			Position:   b.safeInterfaceToInt(col.OrdinalPosition, 0),
			DataType:   fmt.Sprintf("%s", col.DataType),
			UDTName:    fmt.Sprintf("%s", col.UdtName),
			IsNullable: fmt.Sprintf("%s", col.IsNullable) == "YES",
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
		
		table.Columns = append(table.Columns, column)
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
					refConstraintCol := &ConstraintColumn{
						Name:     refColumnName,
						Position: position, // Use same position for referenced column
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
	// Use direct query since SQLC has issues with the index query
	query := `
		SELECT n.nspname as schemaname,
		       t.relname as tablename,
		       i.relname as indexname,
		       pg_get_indexdef(idx.indexrelid) as indexdef
		FROM pg_index idx
		JOIN pg_class i ON i.oid = idx.indexrelid
		JOIN pg_class t ON t.oid = idx.indrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
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
		var schemaName, tableName, indexName, definition string
		if err := rows.Scan(&schemaName, &tableName, &indexName, &definition); err != nil {
			return err
		}
		
		dbSchema := schema.GetOrCreateSchema(schemaName)
		
		index := &Index{
			Schema:     schemaName,
			Table:      tableName,
			Name:       indexName,
			Type:       IndexTypeRegular,
			Definition: definition,
			Columns:    []*IndexColumn{}, // TODO: parse columns from definition
		}
		
		dbSchema.Indexes[indexName] = index
		
		// Also add to table if it exists
		if table, exists := dbSchema.Tables[tableName]; exists {
			table.Indexes[indexName] = index
		}
	}
	
	return rows.Err()
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
		
		dbSchema := schema.GetOrCreateSchema(schemaName)
		
		function := &Function{
			Schema:     schemaName,
			Name:       functionName,
			Definition: fmt.Sprintf("%s", fn.RoutineDefinition),
			ReturnType: fmt.Sprintf("%s", fn.DataType),
			Language:   fmt.Sprintf("%s", fn.ExternalLanguage),
			Parameters: []*Parameter{}, // TODO: parse parameters
		}
		
		dbSchema.Functions[functionName] = function
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
		
		dbSchema := schema.GetOrCreateSchema(schemaName)
		
		v := &View{
			Schema:       schemaName,
			Name:         viewName,
			Definition:   fmt.Sprintf("%s", view.ViewDefinition),
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
		dbSchema.Triggers[key.name] = trigger
		
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
	// For now, return empty extensions since the current query is a placeholder
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
	// Extract function name from "EXECUTE FUNCTION function_name()"
	if strings.Contains(statement, "EXECUTE FUNCTION ") {
		parts := strings.Split(statement, "EXECUTE FUNCTION ")
		if len(parts) > 1 {
			funcPart := strings.TrimSpace(parts[1])
			if idx := strings.Index(funcPart, "("); idx > 0 {
				return funcPart[:idx]
			}
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
// inferSequenceOwnership analyzes column defaults to determine sequence ownership
func (b *Builder) inferSequenceOwnership(ctx context.Context, schema *Schema) error {
	// Iterate through all schemas
	for _, dbSchema := range schema.Schemas {
		// Iterate through all tables
		for _, table := range dbSchema.Tables {
			// Iterate through all columns
			for _, column := range table.Columns {
				// Check if column has a default value that references a sequence
				if column.DefaultValue != nil {
					sequenceName := b.extractSequenceFromDefault(*column.DefaultValue, table.Schema)
					if sequenceName != "" {
						// Find the sequence and update its ownership
						if sequence, exists := dbSchema.Sequences[sequenceName]; exists {
							sequence.OwnedByTable = table.Name
							sequence.OwnedByColumn = column.Name
						}
					}
				}
			}
		}
	}
	return nil
}

// extractSequenceFromDefault extracts sequence name from a default value like "nextval('schema.sequence_name'::regclass)"
func (b *Builder) extractSequenceFromDefault(defaultValue, tableSchema string) string {
	// Look for nextval pattern
	if !strings.Contains(defaultValue, "nextval") {
		return ""
	}
	
	// Extract sequence name from patterns like:
	// nextval('sequence_name'::regclass)
	// nextval('schema.sequence_name'::regclass)
	// nextval('"schema"."sequence_name"'::regclass)
	
	// Find the content between single quotes
	startIdx := strings.Index(defaultValue, "nextval('")
	if startIdx == -1 {
		return ""
	}
	
	nameStart := startIdx + len("nextval('")
	endIdx := strings.Index(defaultValue[nameStart:], "'")
	if endIdx == -1 {
		return ""
	}
	endIdx += nameStart
	
	sequenceRef := defaultValue[nameStart:endIdx]
	
	// Handle schema.sequence_name format
	if strings.Contains(sequenceRef, ".") {
		parts := strings.Split(sequenceRef, ".")
		if len(parts) >= 2 {
			// Take the last part as sequence name, removing any quotes
			sequenceName := strings.Trim(parts[len(parts)-1], `"`)
			return sequenceName
		}
	}
	
	// Handle unqualified sequence name
	return strings.Trim(sequenceRef, `"`)
}
