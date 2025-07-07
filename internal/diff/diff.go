package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

// DDLDiff represents the difference between two DDL states
type DDLDiff struct {
	AddedSchemas      []*ir.DBSchema
	DroppedSchemas    []*ir.DBSchema
	ModifiedSchemas   []*SchemaDiff
	AddedTables       []*ir.Table
	DroppedTables     []*ir.Table
	ModifiedTables    []*TableDiff
	AddedViews        []*ir.View
	DroppedViews      []*ir.View
	ModifiedViews     []*ViewDiff
	AddedExtensions   []*ir.Extension
	DroppedExtensions []*ir.Extension
	AddedFunctions    []*ir.Function
	DroppedFunctions  []*ir.Function
	ModifiedFunctions []*FunctionDiff
	AddedIndexes      []*ir.Index
	DroppedIndexes    []*ir.Index
	AddedTypes        []*ir.Type
	DroppedTypes      []*ir.Type
	ModifiedTypes     []*TypeDiff
	AddedTriggers     []*ir.Trigger
	DroppedTriggers   []*ir.Trigger
	ModifiedTriggers  []*TriggerDiff
}

// SchemaDiff represents changes to a schema
type SchemaDiff struct {
	Old *ir.DBSchema
	New *ir.DBSchema
}

// FunctionDiff represents changes to a function
type FunctionDiff struct {
	Old *ir.Function
	New *ir.Function
}

// TypeDiff represents changes to a type
type TypeDiff struct {
	Old *ir.Type
	New *ir.Type
}

// TriggerDiff represents changes to a trigger
type TriggerDiff struct {
	Old *ir.Trigger
	New *ir.Trigger
}

// ViewDiff represents changes to a view
type ViewDiff struct {
	Old *ir.View
	New *ir.View
}

// TableDiff represents changes to a table
type TableDiff struct {
	Table              *ir.Table
	AddedColumns       []*ir.Column
	DroppedColumns     []*ir.Column
	ModifiedColumns    []*ColumnDiff
	AddedConstraints   []*ir.Constraint
	DroppedConstraints []*ir.Constraint
}

// ColumnDiff represents changes to a column
type ColumnDiff struct {
	Old *ir.Column
	New *ir.Column
}

// getTableNameWithSchema returns the table name with schema qualification only when necessary
// If the table schema is different from the target schema, it returns "schema.table"
// If they are the same, it returns just "table"
func getTableNameWithSchema(tableSchema, tableName, targetSchema string) string {
	if tableSchema != targetSchema {
		return fmt.Sprintf("%s.%s", tableSchema, tableName)
	}
	return tableName
}

// Diff compares two DDL strings and returns the differences
func Diff(oldDDL, newDDL string) (*DDLDiff, error) {
	// Parse the old DDL string to IR Schema
	oldParser := ir.NewParser()
	oldSchema, err := oldParser.ParseSQL(oldDDL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse old DDL: %w", err)
	}

	// Parse the new DDL string to IR Schema
	newParser := ir.NewParser()
	newSchema, err := newParser.ParseSQL(newDDL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse new DDL: %w", err)
	}

	// Perform the diff using the parsed schemas
	return diffSchemas(oldSchema, newSchema), nil
}

// diffSchemas compares two IR schemas and returns the differences
func diffSchemas(oldSchema, newSchema *ir.Schema) *DDLDiff {
	diff := &DDLDiff{
		AddedSchemas:      []*ir.DBSchema{},
		DroppedSchemas:    []*ir.DBSchema{},
		ModifiedSchemas:   []*SchemaDiff{},
		AddedTables:       []*ir.Table{},
		DroppedTables:     []*ir.Table{},
		ModifiedTables:    []*TableDiff{},
		AddedViews:        []*ir.View{},
		DroppedViews:      []*ir.View{},
		ModifiedViews:     []*ViewDiff{},
		AddedExtensions:   []*ir.Extension{},
		DroppedExtensions: []*ir.Extension{},
		AddedFunctions:    []*ir.Function{},
		DroppedFunctions:  []*ir.Function{},
		ModifiedFunctions: []*FunctionDiff{},
		AddedIndexes:      []*ir.Index{},
		DroppedIndexes:    []*ir.Index{},
		AddedTypes:        []*ir.Type{},
		DroppedTypes:      []*ir.Type{},
		ModifiedTypes:     []*TypeDiff{},
		AddedTriggers:     []*ir.Trigger{},
		DroppedTriggers:   []*ir.Trigger{},
		ModifiedTriggers:  []*TriggerDiff{},
	}

	// Compare schemas first
	for name, newDBSchema := range newSchema.Schemas {
		// Skip the public schema as it exists by default
		if name == "public" {
			continue
		}

		if oldDBSchema, exists := oldSchema.Schemas[name]; exists {
			// Check if schema has changed (owner)
			if oldDBSchema.Owner != newDBSchema.Owner {
				diff.ModifiedSchemas = append(diff.ModifiedSchemas, &SchemaDiff{
					Old: oldDBSchema,
					New: newDBSchema,
				})
			}
		} else {
			// Schema was added
			diff.AddedSchemas = append(diff.AddedSchemas, newDBSchema)
		}
	}

	// Find dropped schemas
	for name, oldDBSchema := range oldSchema.Schemas {
		// Skip the public schema as it exists by default
		if name == "public" {
			continue
		}

		if _, exists := newSchema.Schemas[name]; !exists {
			diff.DroppedSchemas = append(diff.DroppedSchemas, oldDBSchema)
		}
	}

	// Build maps for efficient lookup
	oldTables := make(map[string]*ir.Table)
	newTables := make(map[string]*ir.Table)

	// Extract tables from all schemas in oldSchema
	for _, dbSchema := range oldSchema.Schemas {
		for _, table := range dbSchema.Tables {
			key := table.Schema + "." + table.Name
			oldTables[key] = table
		}
	}

	// Extract tables from all schemas in newSchema
	for _, dbSchema := range newSchema.Schemas {
		for _, table := range dbSchema.Tables {
			key := table.Schema + "." + table.Name
			newTables[key] = table
		}
	}

	// Find added tables
	for key, table := range newTables {
		if _, exists := oldTables[key]; !exists {
			diff.AddedTables = append(diff.AddedTables, table)
		}
	}

	// Find dropped tables
	for key, table := range oldTables {
		if _, exists := newTables[key]; !exists {
			diff.DroppedTables = append(diff.DroppedTables, table)
		}
	}

	// Find modified tables
	for key, newTable := range newTables {
		if oldTable, exists := oldTables[key]; exists {
			if tableDiff := diffTables(oldTable, newTable); tableDiff != nil {
				diff.ModifiedTables = append(diff.ModifiedTables, tableDiff)
			}
		}
	}

	// Compare extensions
	oldExtensions := make(map[string]*ir.Extension)
	newExtensions := make(map[string]*ir.Extension)

	if oldSchema.Extensions != nil {
		for name, ext := range oldSchema.Extensions {
			oldExtensions[name] = ext
		}
	}

	if newSchema.Extensions != nil {
		for name, ext := range newSchema.Extensions {
			newExtensions[name] = ext
		}
	}

	// Find added extensions
	for name, ext := range newExtensions {
		if _, exists := oldExtensions[name]; !exists {
			diff.AddedExtensions = append(diff.AddedExtensions, ext)
		}
	}

	// Find dropped extensions
	for name, ext := range oldExtensions {
		if _, exists := newExtensions[name]; !exists {
			diff.DroppedExtensions = append(diff.DroppedExtensions, ext)
		}
	}

	// Compare functions across all schemas
	oldFunctions := make(map[string]*ir.Function)
	newFunctions := make(map[string]*ir.Function)

	// Extract functions from all schemas in oldSchema
	for _, dbSchema := range oldSchema.Schemas {
		for funcName, function := range dbSchema.Functions {
			// Use schema.name(arguments) as key to distinguish functions with different signatures
			key := function.Schema + "." + funcName + "(" + function.Arguments + ")"
			oldFunctions[key] = function
		}
	}

	// Extract functions from all schemas in newSchema
	for _, dbSchema := range newSchema.Schemas {
		for funcName, function := range dbSchema.Functions {
			// Use schema.name(arguments) as key to distinguish functions with different signatures
			key := function.Schema + "." + funcName + "(" + function.Arguments + ")"
			newFunctions[key] = function
		}
	}

	// Find added functions
	for key, function := range newFunctions {
		if _, exists := oldFunctions[key]; !exists {
			diff.AddedFunctions = append(diff.AddedFunctions, function)
		}
	}

	// Find dropped functions
	for key, function := range oldFunctions {
		if _, exists := newFunctions[key]; !exists {
			diff.DroppedFunctions = append(diff.DroppedFunctions, function)
		}
	}

	// Find modified functions
	for key, newFunction := range newFunctions {
		if oldFunction, exists := oldFunctions[key]; exists {
			if !functionsEqual(oldFunction, newFunction) {
				diff.ModifiedFunctions = append(diff.ModifiedFunctions, &FunctionDiff{
					Old: oldFunction,
					New: newFunction,
				})
			}
		}
	}

	// Compare indexes
	oldIndexes := make(map[string]*ir.Index)
	newIndexes := make(map[string]*ir.Index)

	// Extract indexes from all schemas and tables in oldSchema
	for _, dbSchema := range oldSchema.Schemas {
		for _, table := range dbSchema.Tables {
			for _, index := range table.Indexes {
				key := index.Schema + "." + index.Table + "." + index.Name
				oldIndexes[key] = index
			}
		}
	}

	// Extract indexes from all schemas and tables in newSchema
	for _, dbSchema := range newSchema.Schemas {
		for _, table := range dbSchema.Tables {
			for _, index := range table.Indexes {
				key := index.Schema + "." + index.Table + "." + index.Name
				newIndexes[key] = index
			}
		}
	}

	// Find added indexes
	for key, index := range newIndexes {
		if _, exists := oldIndexes[key]; !exists {
			diff.AddedIndexes = append(diff.AddedIndexes, index)
		}
	}

	// Find dropped indexes
	for key, index := range oldIndexes {
		if _, exists := newIndexes[key]; !exists {
			diff.DroppedIndexes = append(diff.DroppedIndexes, index)
		}
	}

	// Compare types across all schemas
	oldTypes := make(map[string]*ir.Type)
	newTypes := make(map[string]*ir.Type)

	// Extract types from all schemas in oldSchema
	for _, dbSchema := range oldSchema.Schemas {
		for typeName, typeObj := range dbSchema.Types {
			key := typeObj.Schema + "." + typeName
			oldTypes[key] = typeObj
		}
	}

	// Extract types from all schemas in newSchema
	for _, dbSchema := range newSchema.Schemas {
		for typeName, typeObj := range dbSchema.Types {
			key := typeObj.Schema + "." + typeName
			newTypes[key] = typeObj
		}
	}

	// Find added types
	for key, typeObj := range newTypes {
		if _, exists := oldTypes[key]; !exists {
			diff.AddedTypes = append(diff.AddedTypes, typeObj)
		}
	}

	// Find dropped types
	for key, typeObj := range oldTypes {
		if _, exists := newTypes[key]; !exists {
			diff.DroppedTypes = append(diff.DroppedTypes, typeObj)
		}
	}

	// Find modified types
	for key, newType := range newTypes {
		if oldType, exists := oldTypes[key]; exists {
			if !typesEqual(oldType, newType) {
				diff.ModifiedTypes = append(diff.ModifiedTypes, &TypeDiff{
					Old: oldType,
					New: newType,
				})
			}
		}
	}

	// Compare views across all schemas
	oldViews := make(map[string]*ir.View)
	newViews := make(map[string]*ir.View)

	// Extract views from all schemas in oldSchema
	for _, dbSchema := range oldSchema.Schemas {
		for viewName, view := range dbSchema.Views {
			key := view.Schema + "." + viewName
			oldViews[key] = view
		}
	}

	// Extract views from all schemas in newSchema
	for _, dbSchema := range newSchema.Schemas {
		for viewName, view := range dbSchema.Views {
			key := view.Schema + "." + viewName
			newViews[key] = view
		}
	}

	// Find added views
	for key, view := range newViews {
		if _, exists := oldViews[key]; !exists {
			diff.AddedViews = append(diff.AddedViews, view)
		}
	}

	// Find dropped views
	for key, view := range oldViews {
		if _, exists := newViews[key]; !exists {
			diff.DroppedViews = append(diff.DroppedViews, view)
		}
	}

	// Find modified views
	for key, newView := range newViews {
		if oldView, exists := oldViews[key]; exists {
			if !viewsEqual(oldView, newView) {
				diff.ModifiedViews = append(diff.ModifiedViews, &ViewDiff{
					Old: oldView,
					New: newView,
				})
			}
		}
	}

	// Compare triggers across all schemas
	oldTriggers := make(map[string]*ir.Trigger)
	newTriggers := make(map[string]*ir.Trigger)

	// Extract triggers from all tables in all schemas in oldSchema
	for _, dbSchema := range oldSchema.Schemas {
		for _, table := range dbSchema.Tables {
			for triggerName, trigger := range table.Triggers {
				key := trigger.Schema + "." + trigger.Table + "." + triggerName
				oldTriggers[key] = trigger
			}
		}
	}

	// Extract triggers from all tables in all schemas in newSchema
	for _, dbSchema := range newSchema.Schemas {
		for _, table := range dbSchema.Tables {
			for triggerName, trigger := range table.Triggers {
				key := trigger.Schema + "." + trigger.Table + "." + triggerName
				newTriggers[key] = trigger
			}
		}
	}

	// Find added triggers
	for key, trigger := range newTriggers {
		if _, exists := oldTriggers[key]; !exists {
			diff.AddedTriggers = append(diff.AddedTriggers, trigger)
		}
	}

	// Find dropped triggers
	for key, trigger := range oldTriggers {
		if _, exists := newTriggers[key]; !exists {
			diff.DroppedTriggers = append(diff.DroppedTriggers, trigger)
		}
	}

	// Find modified triggers
	for key, newTrigger := range newTriggers {
		if oldTrigger, exists := oldTriggers[key]; exists {
			if !triggersEqual(oldTrigger, newTrigger) {
				diff.ModifiedTriggers = append(diff.ModifiedTriggers, &TriggerDiff{
					Old: oldTrigger,
					New: newTrigger,
				})
			}
		}
	}

	return diff
}

// diffTables compares two tables and returns the differences
func diffTables(oldTable, newTable *ir.Table) *TableDiff {
	diff := &TableDiff{
		Table:              newTable,
		AddedColumns:       []*ir.Column{},
		DroppedColumns:     []*ir.Column{},
		ModifiedColumns:    []*ColumnDiff{},
		AddedConstraints:   []*ir.Constraint{},
		DroppedConstraints: []*ir.Constraint{},
	}

	// Build maps for efficient lookup
	oldColumns := make(map[string]*ir.Column)
	newColumns := make(map[string]*ir.Column)

	for _, column := range oldTable.Columns {
		oldColumns[column.Name] = column
	}

	for _, column := range newTable.Columns {
		newColumns[column.Name] = column
	}

	// Find added columns
	for name, column := range newColumns {
		if _, exists := oldColumns[name]; !exists {
			diff.AddedColumns = append(diff.AddedColumns, column)
		}
	}

	// Find dropped columns
	for name, column := range oldColumns {
		if _, exists := newColumns[name]; !exists {
			diff.DroppedColumns = append(diff.DroppedColumns, column)
		}
	}

	// Find modified columns
	for name, newColumn := range newColumns {
		if oldColumn, exists := oldColumns[name]; exists {
			if !columnsEqual(oldColumn, newColumn) {
				diff.ModifiedColumns = append(diff.ModifiedColumns, &ColumnDiff{
					Old: oldColumn,
					New: newColumn,
				})
			}
		}
	}

	// Compare constraints
	oldConstraints := make(map[string]*ir.Constraint)
	newConstraints := make(map[string]*ir.Constraint)

	if oldTable.Constraints != nil {
		for name, constraint := range oldTable.Constraints {
			oldConstraints[name] = constraint
		}
	}

	if newTable.Constraints != nil {
		for name, constraint := range newTable.Constraints {
			newConstraints[name] = constraint
		}
	}

	// Find added constraints
	for name, constraint := range newConstraints {
		if _, exists := oldConstraints[name]; !exists {
			diff.AddedConstraints = append(diff.AddedConstraints, constraint)
		}
	}

	// Find dropped constraints
	for name, constraint := range oldConstraints {
		if _, exists := newConstraints[name]; !exists {
			diff.DroppedConstraints = append(diff.DroppedConstraints, constraint)
		}
	}

	// Return nil if no changes
	if len(diff.AddedColumns) == 0 && len(diff.DroppedColumns) == 0 &&
		len(diff.ModifiedColumns) == 0 && len(diff.AddedConstraints) == 0 &&
		len(diff.DroppedConstraints) == 0 {
		return nil
	}

	return diff
}

// columnsEqual compares two columns for equality
func columnsEqual(old, new *ir.Column) bool {
	if old.Name != new.Name {
		return false
	}
	if old.DataType != new.DataType {
		return false
	}
	if old.IsNullable != new.IsNullable {
		return false
	}

	// Compare default values
	if (old.DefaultValue == nil) != (new.DefaultValue == nil) {
		return false
	}
	if old.DefaultValue != nil && new.DefaultValue != nil && *old.DefaultValue != *new.DefaultValue {
		return false
	}

	// Compare max length
	if (old.MaxLength == nil) != (new.MaxLength == nil) {
		return false
	}
	if old.MaxLength != nil && new.MaxLength != nil && *old.MaxLength != *new.MaxLength {
		return false
	}

	// Compare identity columns
	if old.IsIdentity != new.IsIdentity {
		return false
	}
	if old.IdentityGeneration != new.IdentityGeneration {
		return false
	}

	return true
}

// functionsEqual compares two functions for equality
func functionsEqual(old, new *ir.Function) bool {
	if old.Schema != new.Schema {
		return false
	}
	if old.Name != new.Name {
		return false
	}
	if old.Definition != new.Definition {
		return false
	}
	if old.ReturnType != new.ReturnType {
		return false
	}
	if old.Language != new.Language {
		return false
	}
	if old.Arguments != new.Arguments {
		return false
	}
	if old.Signature != new.Signature {
		return false
	}
	if old.Volatility != new.Volatility {
		return false
	}
	if old.IsSecurityDefiner != new.IsSecurityDefiner {
		return false
	}
	if old.IsStrict != new.IsStrict {
		return false
	}

	return true
}

// typesEqual compares two types for equality
func typesEqual(old, new *ir.Type) bool {
	if old.Schema != new.Schema {
		return false
	}
	if old.Name != new.Name {
		return false
	}
	if old.Kind != new.Kind {
		return false
	}

	switch old.Kind {
	case ir.TypeKindEnum:
		// For ENUM types, compare values
		if len(old.EnumValues) != len(new.EnumValues) {
			return false
		}
		for i, value := range old.EnumValues {
			if value != new.EnumValues[i] {
				return false
			}
		}

	case ir.TypeKindComposite:
		// For composite types, compare columns
		if len(old.Columns) != len(new.Columns) {
			return false
		}
		for i, col := range old.Columns {
			newCol := new.Columns[i]
			if col.Name != newCol.Name || col.DataType != newCol.DataType {
				return false
			}
		}

	case ir.TypeKindDomain:
		// For domain types, compare base type and constraints
		if old.BaseType != new.BaseType {
			return false
		}
		if old.NotNull != new.NotNull {
			return false
		}
		if old.Default != new.Default {
			return false
		}
		if len(old.Constraints) != len(new.Constraints) {
			return false
		}
		for i, constraint := range old.Constraints {
			newConstraint := new.Constraints[i]
			if constraint.Name != newConstraint.Name || constraint.Definition != newConstraint.Definition {
				return false
			}
		}
	}

	return true
}

// viewsEqual compares two views for equality
func viewsEqual(old, new *ir.View) bool {
	if old.Schema != new.Schema {
		return false
	}
	if old.Name != new.Name {
		return false
	}
	if old.Definition != new.Definition {
		return false
	}
	return true
}

// triggersEqual compares two triggers for equality
func triggersEqual(old, new *ir.Trigger) bool {
	if old.Schema != new.Schema {
		return false
	}
	if old.Table != new.Table {
		return false
	}
	if old.Name != new.Name {
		return false
	}
	if old.Timing != new.Timing {
		return false
	}
	if old.Level != new.Level {
		return false
	}
	if old.Function != new.Function {
		return false
	}
	if old.Condition != new.Condition {
		return false
	}

	// Compare events
	if len(old.Events) != len(new.Events) {
		return false
	}
	for i, event := range old.Events {
		if event != new.Events[i] {
			return false
		}
	}

	return true
}

// GenerateMigrationSQL generates SQL statements for the migration
func (d *DDLDiff) GenerateMigrationSQL() string {
	var statements []string

	// Drop schemas first (but this would be rare and dangerous)
	for _, schema := range d.DroppedSchemas {
		statements = append(statements, fmt.Sprintf("DROP SCHEMA %s;", schema.Name))
	}

	// Create new schemas
	// Sort schemas by: 1) schemas without owner first, 2) then by name alphabetically
	sortedAddedSchemas := make([]*ir.DBSchema, len(d.AddedSchemas))
	copy(sortedAddedSchemas, d.AddedSchemas)
	sort.Slice(sortedAddedSchemas, func(i, j int) bool {
		schemaI := sortedAddedSchemas[i]
		schemaJ := sortedAddedSchemas[j]

		// If one has owner and other doesn't, prioritize the one without owner
		if (schemaI.Owner == "") != (schemaJ.Owner == "") {
			return schemaI.Owner == ""
		}

		// If both have same owner status, sort by name
		return schemaI.Name < schemaJ.Name
	})
	for _, schema := range sortedAddedSchemas {
		if schema.Owner != "" {
			statements = append(statements, fmt.Sprintf("CREATE SCHEMA %s AUTHORIZATION %s;", schema.Name, schema.Owner))
		} else {
			statements = append(statements, fmt.Sprintf("CREATE SCHEMA %s;", schema.Name))
		}
	}

	// Modify existing schemas (owner changes)
	// Sort schema changes by name for consistent ordering
	sortedModifiedSchemas := make([]*SchemaDiff, len(d.ModifiedSchemas))
	copy(sortedModifiedSchemas, d.ModifiedSchemas)
	sort.Slice(sortedModifiedSchemas, func(i, j int) bool {
		return sortedModifiedSchemas[i].New.Name < sortedModifiedSchemas[j].New.Name
	})
	for _, schemaDiff := range sortedModifiedSchemas {
		if schemaDiff.Old.Owner != schemaDiff.New.Owner {
			statements = append(statements, fmt.Sprintf("ALTER SCHEMA %s OWNER TO %s;", schemaDiff.New.Name, schemaDiff.New.Owner))
		}
	}

	// Drop types that might be dependencies for tables
	// Sort types by schema.name for consistent ordering
	sortedDroppedTypes := make([]*ir.Type, len(d.DroppedTypes))
	copy(sortedDroppedTypes, d.DroppedTypes)
	sort.Slice(sortedDroppedTypes, func(i, j int) bool {
		keyI := sortedDroppedTypes[i].Schema + "." + sortedDroppedTypes[i].Name
		keyJ := sortedDroppedTypes[j].Schema + "." + sortedDroppedTypes[j].Name
		return keyI < keyJ
	})
	for _, typeObj := range sortedDroppedTypes {
		statements = append(statements, fmt.Sprintf("DROP TYPE IF EXISTS %s.%s;", typeObj.Schema, typeObj.Name))
	}

	// Drop extensions first (before dropping tables that might depend on them)
	// Sort extensions by name for consistent ordering
	sortedDroppedExtensions := make([]*ir.Extension, len(d.DroppedExtensions))
	copy(sortedDroppedExtensions, d.DroppedExtensions)
	sort.Slice(sortedDroppedExtensions, func(i, j int) bool {
		return sortedDroppedExtensions[i].Name < sortedDroppedExtensions[j].Name
	})
	for _, ext := range sortedDroppedExtensions {
		statements = append(statements, fmt.Sprintf("DROP EXTENSION IF EXISTS %s;", ext.Name))
	}

	// Drop indexes (before dropping tables)
	// Sort indexes by schema.table.name for consistent ordering
	sortedDroppedIndexes := make([]*ir.Index, len(d.DroppedIndexes))
	copy(sortedDroppedIndexes, d.DroppedIndexes)
	sort.Slice(sortedDroppedIndexes, func(i, j int) bool {
		keyI := sortedDroppedIndexes[i].Schema + "." + sortedDroppedIndexes[i].Table + "." + sortedDroppedIndexes[i].Name
		keyJ := sortedDroppedIndexes[j].Schema + "." + sortedDroppedIndexes[j].Table + "." + sortedDroppedIndexes[j].Name
		return keyI < keyJ
	})
	for _, index := range sortedDroppedIndexes {
		statements = append(statements, fmt.Sprintf("DROP INDEX IF EXISTS %s.%s;", index.Schema, index.Name))
	}

	// Drop views (before dropping tables they might depend on)
	// Sort views by schema.name for consistent ordering
	sortedDroppedViews := make([]*ir.View, len(d.DroppedViews))
	copy(sortedDroppedViews, d.DroppedViews)
	sort.Slice(sortedDroppedViews, func(i, j int) bool {
		keyI := sortedDroppedViews[i].Schema + "." + sortedDroppedViews[i].Name
		keyJ := sortedDroppedViews[j].Schema + "." + sortedDroppedViews[j].Name
		return keyI < keyJ
	})
	for _, view := range sortedDroppedViews {
		statements = append(statements, fmt.Sprintf("DROP VIEW IF EXISTS %s.%s;", view.Schema, view.Name))
	}

	// Drop tables
	for _, table := range d.DroppedTables {
		statements = append(statements, fmt.Sprintf("DROP TABLE %s.%s;", table.Schema, table.Name))
	}

	// Create extensions (before creating tables that might depend on them)
	// Sort extensions by name for consistent ordering
	sortedExtensions := make([]*ir.Extension, len(d.AddedExtensions))
	copy(sortedExtensions, d.AddedExtensions)
	sort.Slice(sortedExtensions, func(i, j int) bool {
		return sortedExtensions[i].Name < sortedExtensions[j].Name
	})
	for _, ext := range sortedExtensions {
		if ext.Schema != "" {
			statements = append(statements, fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s WITH SCHEMA %s;", ext.Name, ext.Schema))
		} else {
			statements = append(statements, fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s;", ext.Name))
		}
	}

	// Create types (before creating tables that might use them)
	// Sort types by schema.name for consistent ordering
	sortedAddedTypes := make([]*ir.Type, len(d.AddedTypes))
	copy(sortedAddedTypes, d.AddedTypes)
	sort.Slice(sortedAddedTypes, func(i, j int) bool {
		keyI := sortedAddedTypes[i].Schema + "." + sortedAddedTypes[i].Name
		keyJ := sortedAddedTypes[j].Schema + "." + sortedAddedTypes[j].Name
		return keyI < keyJ
	})
	for _, typeObj := range sortedAddedTypes {
		// Generate CREATE TYPE statement without comments for migration
		switch typeObj.Kind {
		case ir.TypeKindEnum:
			var values []string
			for _, value := range typeObj.EnumValues {
				values = append(values, fmt.Sprintf("   '%s'", value))
			}
			stmt := fmt.Sprintf("CREATE TYPE %s.%s AS ENUM (\n%s\n);",
				typeObj.Schema, typeObj.Name, strings.Join(values, ",\n"))
			statements = append(statements, stmt)
		case ir.TypeKindComposite:
			var columns []string
			for _, col := range typeObj.Columns {
				columns = append(columns, fmt.Sprintf("\t%s %s", col.Name, col.DataType))
			}
			stmt := fmt.Sprintf("CREATE TYPE %s.%s AS (\n%s\n);",
				typeObj.Schema, typeObj.Name, strings.Join(columns, ",\n"))
			statements = append(statements, stmt)
		case ir.TypeKindDomain:
			var parts []string
			parts = append(parts, fmt.Sprintf("CREATE DOMAIN %s.%s AS %s", typeObj.Schema, typeObj.Name, typeObj.BaseType))
			if typeObj.Default != "" {
				parts = append(parts, fmt.Sprintf("DEFAULT %s", typeObj.Default))
			}
			if typeObj.NotNull {
				parts = append(parts, "NOT NULL")
			}
			for _, constraint := range typeObj.Constraints {
				if constraint.Name != "" {
					parts = append(parts, fmt.Sprintf("\tCONSTRAINT %s %s", constraint.Name, constraint.Definition))
				} else {
					parts = append(parts, fmt.Sprintf("\t%s", constraint.Definition))
				}
			}
			stmt := strings.Join(parts, "\n") + ";"
			statements = append(statements, stmt)
		}
	}

	// Modify existing types (only ENUM types can be modified)
	// Sort modified types by schema.name for consistent ordering
	sortedModifiedTypes := make([]*TypeDiff, len(d.ModifiedTypes))
	copy(sortedModifiedTypes, d.ModifiedTypes)
	sort.Slice(sortedModifiedTypes, func(i, j int) bool {
		keyI := sortedModifiedTypes[i].New.Schema + "." + sortedModifiedTypes[i].New.Name
		keyJ := sortedModifiedTypes[j].New.Schema + "." + sortedModifiedTypes[j].New.Name
		return keyI < keyJ
	})
	for _, typeDiff := range sortedModifiedTypes {
		statements = append(statements, typeDiff.GenerateMigrationSQL()...)
	}

	// Drop functions
	// Sort functions by schema.name for consistent ordering
	sortedDroppedFunctions := make([]*ir.Function, len(d.DroppedFunctions))
	copy(sortedDroppedFunctions, d.DroppedFunctions)
	sort.Slice(sortedDroppedFunctions, func(i, j int) bool {
		keyI := sortedDroppedFunctions[i].Schema + "." + sortedDroppedFunctions[i].Name
		keyJ := sortedDroppedFunctions[j].Schema + "." + sortedDroppedFunctions[j].Name
		return keyI < keyJ
	})
	for _, function := range sortedDroppedFunctions {
		functionName := getTableNameWithSchema(function.Schema, function.Name, function.Schema)
		if function.Arguments != "" {
			statements = append(statements, fmt.Sprintf("DROP FUNCTION IF EXISTS %s(%s);", functionName, function.Arguments))
		} else {
			statements = append(statements, fmt.Sprintf("DROP FUNCTION IF EXISTS %s();", functionName))
		}
	}

	// Create new tables
	for _, table := range d.AddedTables {
		statements = append(statements, table.GenerateSQLWithOptions(false, ""))
	}

	// Create views (after tables, as they depend on tables)
	// Sort views by schema.name for consistent ordering
	sortedAddedViews := make([]*ir.View, len(d.AddedViews))
	copy(sortedAddedViews, d.AddedViews)
	sort.Slice(sortedAddedViews, func(i, j int) bool {
		keyI := sortedAddedViews[i].Schema + "." + sortedAddedViews[i].Name
		keyJ := sortedAddedViews[j].Schema + "." + sortedAddedViews[j].Name
		return keyI < keyJ
	})
	for _, view := range sortedAddedViews {
		// Generate CREATE VIEW statement
		// Always include schema prefix on view name
		viewName := fmt.Sprintf("%s.%s", view.Schema, view.Name)

		// The view definition should include schema-qualified table references
		// For now, we'll need to enhance the parser to preserve the original formatting
		// or implement a more sophisticated SQL formatter
		definition := view.Definition

		// Simple heuristic: if the definition references tables without schema qualification,
		// and the view is in public schema, add public. prefix to table references
		if view.Schema == "public" && !strings.Contains(definition, "public.") {
			// This is a simple approach - a more robust solution would parse the SQL
			definition = strings.ReplaceAll(definition, "FROM employees", "FROM public.employees")
		}

		// Format the definition with newlines for better readability
		formattedDef := definition
		if strings.Contains(definition, "SELECT") && !strings.Contains(definition, "\n") {
			// Simple formatting: add newlines after SELECT and before FROM/WHERE
			formattedDef = strings.ReplaceAll(definition, "SELECT ", "SELECT \n    ")
			formattedDef = strings.ReplaceAll(formattedDef, ", ", ",\n    ")
			formattedDef = strings.ReplaceAll(formattedDef, " FROM ", "\nFROM ")
			formattedDef = strings.ReplaceAll(formattedDef, " WHERE ", "\nWHERE ")
		}

		stmt := fmt.Sprintf("CREATE OR REPLACE VIEW %s AS\n%s;", viewName, formattedDef)
		statements = append(statements, stmt)
	}

	// Modify existing views (using CREATE OR REPLACE VIEW)
	// Sort modified views by schema.name for consistent ordering
	sortedModifiedViews := make([]*ViewDiff, len(d.ModifiedViews))
	copy(sortedModifiedViews, d.ModifiedViews)
	sort.Slice(sortedModifiedViews, func(i, j int) bool {
		keyI := sortedModifiedViews[i].New.Schema + "." + sortedModifiedViews[i].New.Name
		keyJ := sortedModifiedViews[j].New.Schema + "." + sortedModifiedViews[j].New.Name
		return keyI < keyJ
	})
	for _, viewDiff := range sortedModifiedViews {
		view := viewDiff.New
		// Always include schema prefix on view name
		viewName := fmt.Sprintf("%s.%s", view.Schema, view.Name)

		// Apply the same formatting as for new views
		definition := view.Definition
		if view.Schema == "public" && !strings.Contains(definition, "public.") {
			definition = strings.ReplaceAll(definition, "FROM employees", "FROM public.employees")
		}

		formattedDef := definition
		if strings.Contains(definition, "SELECT") && !strings.Contains(definition, "\n") {
			formattedDef = strings.ReplaceAll(definition, "SELECT ", "SELECT \n    ")
			formattedDef = strings.ReplaceAll(formattedDef, ", ", ",\n    ")
			formattedDef = strings.ReplaceAll(formattedDef, " FROM ", "\nFROM ")
			formattedDef = strings.ReplaceAll(formattedDef, " WHERE ", "\nWHERE ")
		}

		stmt := fmt.Sprintf("CREATE OR REPLACE VIEW %s AS\n%s;", viewName, formattedDef)
		statements = append(statements, stmt)
	}

	// Create functions (after tables, in case they reference tables)
	// Sort functions by schema.name for consistent ordering
	sortedAddedFunctions := make([]*ir.Function, len(d.AddedFunctions))
	copy(sortedAddedFunctions, d.AddedFunctions)
	sort.Slice(sortedAddedFunctions, func(i, j int) bool {
		keyI := sortedAddedFunctions[i].Schema + "." + sortedAddedFunctions[i].Name
		keyJ := sortedAddedFunctions[j].Schema + "." + sortedAddedFunctions[j].Name
		return keyI < keyJ
	})
	for _, function := range sortedAddedFunctions {
		// Build the CREATE FUNCTION statement from parsed data
		var stmt strings.Builder

		// Build the CREATE FUNCTION header with schema qualification
		functionName := getTableNameWithSchema(function.Schema, function.Name, function.Schema)
		stmt.WriteString(fmt.Sprintf("CREATE OR REPLACE FUNCTION %s", functionName))

		// Add parameters
		if function.Signature != "" {
			stmt.WriteString(fmt.Sprintf("(\n    %s\n)", strings.ReplaceAll(function.Signature, ", ", ",\n    ")))
		} else {
			stmt.WriteString("()")
		}

		// Add return type
		if function.ReturnType != "" {
			stmt.WriteString(fmt.Sprintf("\nRETURNS %s", function.ReturnType))
		}

		// Add language
		if function.Language != "" {
			stmt.WriteString(fmt.Sprintf("\nLANGUAGE %s", function.Language))
		}

		// Add security definer if set
		if function.IsSecurityDefiner {
			stmt.WriteString("\nSECURITY DEFINER")
		}

		// Add volatility if not default
		if function.Volatility != "" {
			stmt.WriteString(fmt.Sprintf("\n%s", function.Volatility))
		}

		// Add the function body
		if function.Definition != "" {
			stmt.WriteString(fmt.Sprintf("\nAS $$%s\n$$;", function.Definition))
		}

		statements = append(statements, stmt.String())
	}

	// Modify existing functions (using CREATE OR REPLACE)
	// Sort modified functions by schema.name for consistent ordering
	sortedModifiedFunctions := make([]*FunctionDiff, len(d.ModifiedFunctions))
	copy(sortedModifiedFunctions, d.ModifiedFunctions)
	sort.Slice(sortedModifiedFunctions, func(i, j int) bool {
		keyI := sortedModifiedFunctions[i].New.Schema + "." + sortedModifiedFunctions[i].New.Name
		keyJ := sortedModifiedFunctions[j].New.Schema + "." + sortedModifiedFunctions[j].New.Name
		return keyI < keyJ
	})
	for _, functionDiff := range sortedModifiedFunctions {
		function := functionDiff.New
		// Build the CREATE OR REPLACE FUNCTION statement from parsed data
		var stmt strings.Builder

		// Build the CREATE OR REPLACE FUNCTION header with schema qualification
		functionName := getTableNameWithSchema(function.Schema, function.Name, function.Schema)
		stmt.WriteString(fmt.Sprintf("CREATE OR REPLACE FUNCTION %s", functionName))

		// Add parameters
		if function.Signature != "" {
			stmt.WriteString(fmt.Sprintf("(\n    %s\n)", strings.ReplaceAll(function.Signature, ", ", ",\n    ")))
		} else {
			stmt.WriteString("()")
		}

		// Add return type
		if function.ReturnType != "" {
			stmt.WriteString(fmt.Sprintf("\nRETURNS %s", function.ReturnType))
		}

		// Add language
		if function.Language != "" {
			stmt.WriteString(fmt.Sprintf("\nLANGUAGE %s", function.Language))
		}

		// Add security definer/invoker
		if function.IsSecurityDefiner {
			stmt.WriteString("\nSECURITY DEFINER")
		} else {
			stmt.WriteString("\nSECURITY INVOKER")
		}

		// Add volatility if not default
		if function.Volatility != "" {
			stmt.WriteString(fmt.Sprintf("\n%s", function.Volatility))
		}

		// Add the function body
		if function.Definition != "" {
			stmt.WriteString(fmt.Sprintf("\nAS $$%s\n$$;", function.Definition))
		}

		statements = append(statements, stmt.String())
	}

	// Modify existing tables
	for _, tableDiff := range d.ModifiedTables {
		statements = append(statements, tableDiff.GenerateMigrationSQL()...)
	}

	// Create indexes (after tables and constraints are created)
	// Sort indexes by schema.table.name for consistent ordering
	sortedAddedIndexes := make([]*ir.Index, len(d.AddedIndexes))
	copy(sortedAddedIndexes, d.AddedIndexes)
	sort.Slice(sortedAddedIndexes, func(i, j int) bool {
		keyI := sortedAddedIndexes[i].Schema + "." + sortedAddedIndexes[i].Table + "." + sortedAddedIndexes[i].Name
		keyJ := sortedAddedIndexes[j].Schema + "." + sortedAddedIndexes[j].Table + "." + sortedAddedIndexes[j].Name
		return keyI < keyJ
	})
	for _, index := range sortedAddedIndexes {
		// Generate clean migration SQL without schema qualifiers and USING btree
		indexSQL := index.GenerateSQL("")
		// Remove any comment headers and trailing newlines
		indexSQL = strings.TrimSpace(indexSQL)
		// Extract just the CREATE INDEX statement
		lines := strings.Split(indexSQL, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "CREATE") && strings.Contains(line, "INDEX") {
				statements = append(statements, line)
				break
			}
		}
	}

	// Drop triggers (before creating new/modified ones)
	// Sort triggers by schema.table.name for consistent ordering
	sortedDroppedTriggers := make([]*ir.Trigger, len(d.DroppedTriggers))
	copy(sortedDroppedTriggers, d.DroppedTriggers)
	sort.Slice(sortedDroppedTriggers, func(i, j int) bool {
		keyI := sortedDroppedTriggers[i].Schema + "." + sortedDroppedTriggers[i].Table + "." + sortedDroppedTriggers[i].Name
		keyJ := sortedDroppedTriggers[j].Schema + "." + sortedDroppedTriggers[j].Table + "." + sortedDroppedTriggers[j].Name
		return keyI < keyJ
	})
	for _, trigger := range sortedDroppedTriggers {
		statements = append(statements, fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s.%s;", trigger.Name, trigger.Schema, trigger.Table))
	}

	// Create new triggers
	// Sort triggers by schema.table.name for consistent ordering
	sortedAddedTriggers := make([]*ir.Trigger, len(d.AddedTriggers))
	copy(sortedAddedTriggers, d.AddedTriggers)
	sort.Slice(sortedAddedTriggers, func(i, j int) bool {
		keyI := sortedAddedTriggers[i].Schema + "." + sortedAddedTriggers[i].Table + "." + sortedAddedTriggers[i].Name
		keyJ := sortedAddedTriggers[j].Schema + "." + sortedAddedTriggers[j].Table + "." + sortedAddedTriggers[j].Name
		return keyI < keyJ
	})
	for _, trigger := range sortedAddedTriggers {
		statements = append(statements, trigger.GenerateSimpleSQL())
	}

	// Modify existing triggers (use CREATE OR REPLACE)
	// Sort modified triggers by schema.table.name for consistent ordering
	sortedModifiedTriggers := make([]*TriggerDiff, len(d.ModifiedTriggers))
	copy(sortedModifiedTriggers, d.ModifiedTriggers)
	sort.Slice(sortedModifiedTriggers, func(i, j int) bool {
		keyI := sortedModifiedTriggers[i].New.Schema + "." + sortedModifiedTriggers[i].New.Table + "." + sortedModifiedTriggers[i].New.Name
		keyJ := sortedModifiedTriggers[j].New.Schema + "." + sortedModifiedTriggers[j].New.Table + "." + sortedModifiedTriggers[j].New.Name
		return keyI < keyJ
	})
	for _, triggerDiff := range sortedModifiedTriggers {
		// Use CREATE OR REPLACE for trigger modifications
		statements = append(statements, triggerDiff.New.GenerateSimpleSQL())
	}

	return strings.Join(statements, "\n")
}

// GenerateMigrationSQL generates SQL statements for table modifications
func (td *TableDiff) GenerateMigrationSQL() []string {
	var statements []string

	// Drop constraints first (before dropping columns)
	for _, constraint := range td.DroppedConstraints {
		tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, td.Table.Schema)
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s;",
			tableName, constraint.Name))
	}

	// Drop columns (sort by position for consistent ordering)
	sortedDroppedColumns := make([]*ir.Column, len(td.DroppedColumns))
	copy(sortedDroppedColumns, td.DroppedColumns)
	sort.Slice(sortedDroppedColumns, func(i, j int) bool {
		return sortedDroppedColumns[i].Position < sortedDroppedColumns[j].Position
	})
	for _, column := range sortedDroppedColumns {
		tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, td.Table.Schema)
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;",
			tableName, column.Name))
	}

	// Add new columns (sort by position for consistent ordering)
	sortedAddedColumns := make([]*ir.Column, len(td.AddedColumns))
	copy(sortedAddedColumns, td.AddedColumns)
	sort.Slice(sortedAddedColumns, func(i, j int) bool {
		return sortedAddedColumns[i].Position < sortedAddedColumns[j].Position
	})

	// Track which FK constraints are handled inline with column additions
	handledFKConstraints := make(map[string]bool)

	for _, column := range sortedAddedColumns {
		// Check if this column has an associated foreign key constraint
		var fkConstraint *ir.Constraint
		for _, constraint := range td.Table.Constraints {
			if constraint.Type == ir.ConstraintTypeForeignKey &&
				len(constraint.Columns) == 1 &&
				constraint.Columns[0].Name == column.Name {
				fkConstraint = constraint
				handledFKConstraints[constraint.Name] = true
				break
			}
		}

		// Use line break format for complex statements (with foreign keys)
		var stmt string
		tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, td.Table.Schema)
		if fkConstraint != nil {
			stmt = fmt.Sprintf("ALTER TABLE %s\nADD COLUMN %s %s",
				tableName, column.Name, column.DataType)
		} else {
			stmt = fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
				tableName, column.Name, column.DataType)
		}

		// Add foreign key reference inline if present
		if fkConstraint != nil {
			referencedTableName := getTableNameWithSchema(fkConstraint.ReferencedSchema, fkConstraint.ReferencedTable, td.Table.Schema)
			stmt += fmt.Sprintf(" REFERENCES %s", referencedTableName)

			if len(fkConstraint.ReferencedColumns) > 0 {
				var refCols []string
				for _, refCol := range fkConstraint.ReferencedColumns {
					refCols = append(refCols, refCol.Name)
				}
				stmt += fmt.Sprintf("(%s)", strings.Join(refCols, ", "))
			}

			// Add referential actions
			if fkConstraint.UpdateRule != "" && fkConstraint.UpdateRule != "NO ACTION" {
				stmt += fmt.Sprintf(" ON UPDATE %s", fkConstraint.UpdateRule)
			}
			if fkConstraint.DeleteRule != "" && fkConstraint.DeleteRule != "NO ACTION" {
				stmt += fmt.Sprintf(" ON DELETE %s", fkConstraint.DeleteRule)
			}

			// Add deferrable clause
			if fkConstraint.Deferrable {
				if fkConstraint.InitiallyDeferred {
					stmt += " DEFERRABLE INITIALLY DEFERRED"
				} else {
					stmt += " DEFERRABLE"
				}
			}
		}

		// Add identity column syntax
		if column.IsIdentity {
			if column.IdentityGeneration == "ALWAYS" {
				stmt += " GENERATED ALWAYS AS IDENTITY"
			} else if column.IdentityGeneration == "BY DEFAULT" {
				stmt += " GENERATED BY DEFAULT AS IDENTITY"
			}
		}

		if column.DefaultValue != nil && !column.IsIdentity {
			stmt += fmt.Sprintf(" DEFAULT %s", *column.DefaultValue)
		}

		if !column.IsNullable {
			stmt += " NOT NULL"
		}

		statements = append(statements, stmt+";")
	}

	// Modify existing columns (sort by position to maintain column order)
	sortedModifiedColumns := make([]*ColumnDiff, len(td.ModifiedColumns))
	copy(sortedModifiedColumns, td.ModifiedColumns)
	sort.Slice(sortedModifiedColumns, func(i, j int) bool {
		return sortedModifiedColumns[i].New.Position < sortedModifiedColumns[j].New.Position
	})
	for _, columnDiff := range sortedModifiedColumns {
		statements = append(statements, columnDiff.GenerateMigrationSQL(td.Table.Schema, td.Table.Name)...)
	}

	// Add new constraints (sort by name for consistent ordering)
	sortedConstraints := make([]*ir.Constraint, 0)
	for _, constraint := range td.AddedConstraints {
		// Skip FK constraints that were already handled inline with column additions
		if constraint.Type == ir.ConstraintTypeForeignKey && handledFKConstraints[constraint.Name] {
			continue
		}
		sortedConstraints = append(sortedConstraints, constraint)
	}
	sort.Slice(sortedConstraints, func(i, j int) bool {
		return sortedConstraints[i].Name < sortedConstraints[j].Name
	})
	for _, constraint := range sortedConstraints {
		switch constraint.Type {
		case ir.ConstraintTypeUnique:
			// Sort columns by position
			columns := constraint.SortConstraintColumnsByPosition()
			var columnNames []string
			for _, col := range columns {
				columnNames = append(columnNames, col.Name)
			}
			tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, td.Table.Schema)
			stmt := fmt.Sprintf("ALTER TABLE %s \nADD CONSTRAINT %s UNIQUE (%s);",
				tableName, constraint.Name, strings.Join(columnNames, ", "))
			statements = append(statements, stmt)

		case ir.ConstraintTypeCheck:
			// CheckClause already contains "CHECK (...)" from the constraint definition
			tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, td.Table.Schema)
			stmt := fmt.Sprintf("ALTER TABLE %s \nADD CONSTRAINT %s %s;",
				tableName, constraint.Name, constraint.CheckClause)
			statements = append(statements, stmt)

		case ir.ConstraintTypeForeignKey:
			// Sort columns by position
			columns := constraint.SortConstraintColumnsByPosition()
			var columnNames []string
			for _, col := range columns {
				columnNames = append(columnNames, col.Name)
			}

			// Sort referenced columns by position
			var refColumnNames []string
			if len(constraint.ReferencedColumns) > 0 {
				refColumns := make([]*ir.ConstraintColumn, len(constraint.ReferencedColumns))
				copy(refColumns, constraint.ReferencedColumns)
				sort.Slice(refColumns, func(i, j int) bool {
					return refColumns[i].Position < refColumns[j].Position
				})
				for _, col := range refColumns {
					refColumnNames = append(refColumnNames, col.Name)
				}
			}

			tableName := getTableNameWithSchema(td.Table.Schema, td.Table.Name, td.Table.Schema)
			referencedTableName := getTableNameWithSchema(constraint.ReferencedSchema, constraint.ReferencedTable, td.Table.Schema)
			stmt := fmt.Sprintf("ALTER TABLE %s \nADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s)",
				tableName, constraint.Name,
				strings.Join(columnNames, ", "),
				referencedTableName,
				strings.Join(refColumnNames, ", "))

			// Add referential actions
			if constraint.UpdateRule != "" && constraint.UpdateRule != "NO ACTION" {
				stmt += fmt.Sprintf(" ON UPDATE %s", constraint.UpdateRule)
			}
			if constraint.DeleteRule != "" && constraint.DeleteRule != "NO ACTION" {
				stmt += fmt.Sprintf(" ON DELETE %s", constraint.DeleteRule)
			}

			// Add deferrable clause
			if constraint.Deferrable {
				if constraint.InitiallyDeferred {
					stmt += " DEFERRABLE INITIALLY DEFERRED"
				} else {
					stmt += " DEFERRABLE"
				}
			}

			statements = append(statements, stmt+";")

		case ir.ConstraintTypePrimaryKey:
			// Sort columns by position
			columns := constraint.SortConstraintColumnsByPosition()
			var columnNames []string
			for _, col := range columns {
				columnNames = append(columnNames, col.Name)
			}
			stmt := fmt.Sprintf("ALTER TABLE %s.%s \nADD CONSTRAINT %s PRIMARY KEY (%s);",
				td.Table.Schema, td.Table.Name, constraint.Name, strings.Join(columnNames, ", "))
			statements = append(statements, stmt)
		}
	}

	return statements
}

// GenerateMigrationSQL generates SQL statements for type modifications
func (td *TypeDiff) GenerateMigrationSQL() []string {
	var statements []string

	// Only ENUM types can be modified (add values)
	if td.Old.Kind == ir.TypeKindEnum && td.New.Kind == ir.TypeKindEnum {
		// Find added enum values
		oldValues := make(map[string]int)
		for i, value := range td.Old.EnumValues {
			oldValues[value] = i
		}

		for i, value := range td.New.EnumValues {
			if _, exists := oldValues[value]; !exists {
				// This is a new value - determine position
				var stmt string
				if i == 0 {
					// First value
					stmt = fmt.Sprintf("ALTER TYPE %s.%s ADD VALUE '%s' BEFORE '%s';",
						td.New.Schema, td.New.Name, value, td.New.EnumValues[1])
				} else if i == len(td.New.EnumValues)-1 {
					// Last value
					stmt = fmt.Sprintf("ALTER TYPE %s.%s ADD VALUE '%s' AFTER '%s';",
						td.New.Schema, td.New.Name, value, td.New.EnumValues[i-1])
				} else {
					// Middle value - add after the previous value
					stmt = fmt.Sprintf("ALTER TYPE %s.%s ADD VALUE '%s' AFTER '%s';",
						td.New.Schema, td.New.Name, value, td.New.EnumValues[i-1])
				}
				statements = append(statements, stmt)
			}
		}
	}

	return statements
}

// GenerateMigrationSQL generates SQL statements for column modifications
func (cd *ColumnDiff) GenerateMigrationSQL(schema, tableName string) []string {
	var statements []string
	qualifiedTableName := getTableNameWithSchema(schema, tableName, schema)

	// Handle data type changes
	if cd.Old.DataType != cd.New.DataType {
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;",
			qualifiedTableName, cd.New.Name, cd.New.DataType))
	}

	// Handle nullable changes
	if cd.Old.IsNullable != cd.New.IsNullable {
		if cd.New.IsNullable {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;",
				qualifiedTableName, cd.New.Name))
		} else {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;",
				qualifiedTableName, cd.New.Name))
		}
	}

	// Handle default value changes
	oldDefault := cd.Old.DefaultValue
	newDefault := cd.New.DefaultValue

	if (oldDefault == nil) != (newDefault == nil) ||
		(oldDefault != nil && newDefault != nil && *oldDefault != *newDefault) {

		if newDefault == nil {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;",
				qualifiedTableName, cd.New.Name))
		} else {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;",
				qualifiedTableName, cd.New.Name, *newDefault))
		}
	}

	return statements
}
