package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

type DDLDiff struct {
	AddedSchemas       []*ir.Schema
	DroppedSchemas     []*ir.Schema
	ModifiedSchemas    []*SchemaDiff
	AddedTables        []*ir.Table
	DroppedTables      []*ir.Table
	ModifiedTables     []*TableDiff
	AddedViews         []*ir.View
	DroppedViews       []*ir.View
	ModifiedViews      []*ViewDiff
	AddedFunctions     []*ir.Function
	DroppedFunctions   []*ir.Function
	ModifiedFunctions  []*FunctionDiff
	AddedProcedures    []*ir.Procedure
	DroppedProcedures  []*ir.Procedure
	ModifiedProcedures []*ProcedureDiff
	AddedTypes         []*ir.Type
	DroppedTypes       []*ir.Type
	ModifiedTypes      []*TypeDiff
}

// SchemaDiff represents changes to a schema
type SchemaDiff struct {
	Old *ir.Schema
	New *ir.Schema
}

// FunctionDiff represents changes to a function
type FunctionDiff struct {
	Old *ir.Function
	New *ir.Function
}

// ProcedureDiff represents changes to a procedure
type ProcedureDiff struct {
	Old *ir.Procedure
	New *ir.Procedure
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
	AddedIndexes       []*ir.Index
	DroppedIndexes     []*ir.Index
	ModifiedIndexes    []*IndexDiff
	AddedTriggers      []*ir.Trigger
	DroppedTriggers    []*ir.Trigger
	ModifiedTriggers   []*TriggerDiff
	AddedPolicies      []*ir.RLSPolicy
	DroppedPolicies    []*ir.RLSPolicy
	ModifiedPolicies   []*PolicyDiff
	RLSChanges         []*RLSChange
	CommentChanged     bool
	OldComment         string
	NewComment         string
}

// ColumnDiff represents changes to a column
type ColumnDiff struct {
	Old *ir.Column
	New *ir.Column
}

// IndexDiff represents changes to an index
type IndexDiff struct {
	Old *ir.Index
	New *ir.Index
}

// PolicyDiff represents changes to a policy
type PolicyDiff struct {
	Old *ir.RLSPolicy
	New *ir.RLSPolicy
}

// RLSChange represents enabling/disabling Row Level Security on a table
type RLSChange struct {
	Table   *ir.Table
	Enabled bool // true to enable, false to disable
}

// Diff compares two IR schemas directly and returns the differences
func Diff(oldIR, newIR *ir.IR) *DDLDiff {
	diff := &DDLDiff{
		AddedSchemas:       []*ir.Schema{},
		DroppedSchemas:     []*ir.Schema{},
		ModifiedSchemas:    []*SchemaDiff{},
		AddedTables:        []*ir.Table{},
		DroppedTables:      []*ir.Table{},
		ModifiedTables:     []*TableDiff{},
		AddedViews:         []*ir.View{},
		DroppedViews:       []*ir.View{},
		ModifiedViews:      []*ViewDiff{},
		AddedFunctions:     []*ir.Function{},
		DroppedFunctions:   []*ir.Function{},
		ModifiedFunctions:  []*FunctionDiff{},
		AddedProcedures:    []*ir.Procedure{},
		DroppedProcedures:  []*ir.Procedure{},
		ModifiedProcedures: []*ProcedureDiff{},
		AddedTypes:         []*ir.Type{},
		DroppedTypes:       []*ir.Type{},
		ModifiedTypes:      []*TypeDiff{},
	}

	// Compare schemas first in deterministic order
	schemaNames := sortedKeys(newIR.Schemas)
	for _, name := range schemaNames {
		newDBSchema := newIR.Schemas[name]
		// Skip the public schema as it exists by default
		if name == "public" {
			continue
		}

		if oldDBSchema, exists := oldIR.Schemas[name]; exists {
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

	// Find dropped schemas in deterministic order
	oldSchemaNames := sortedKeys(oldIR.Schemas)
	for _, name := range oldSchemaNames {
		oldDBSchema := oldIR.Schemas[name]
		// Skip the public schema as it exists by default
		if name == "public" {
			continue
		}

		if _, exists := newIR.Schemas[name]; !exists {
			diff.DroppedSchemas = append(diff.DroppedSchemas, oldDBSchema)
		}
	}

	// Build maps for efficient lookup
	oldTables := make(map[string]*ir.Table)
	newTables := make(map[string]*ir.Table)

	// Extract tables from all schemas in oldIR
	for _, dbSchema := range oldIR.Schemas {
		for _, table := range dbSchema.Tables {
			key := table.Schema + "." + table.Name
			oldTables[key] = table
		}
	}

	// Extract tables from all schemas in newIR
	for _, dbSchema := range newIR.Schemas {
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


	// Compare functions across all schemas
	oldFunctions := make(map[string]*ir.Function)
	newFunctions := make(map[string]*ir.Function)

	// Extract functions from all schemas in oldIR in deterministic order
	for _, dbSchema := range oldIR.Schemas {
		funcNames := sortedKeys(dbSchema.Functions)
		for _, funcName := range funcNames {
			function := dbSchema.Functions[funcName]
			// Use schema.name(arguments) as key to distinguish functions with different signatures
			key := function.Schema + "." + funcName + "(" + function.Arguments + ")"
			oldFunctions[key] = function
		}
	}

	// Extract functions from all schemas in newIR in deterministic order
	for _, dbSchema := range newIR.Schemas {
		funcNames := sortedKeys(dbSchema.Functions)
		for _, funcName := range funcNames {
			function := dbSchema.Functions[funcName]
			// Use schema.name(arguments) as key to distinguish functions with different signatures
			key := function.Schema + "." + funcName + "(" + function.Arguments + ")"
			newFunctions[key] = function
		}
	}

	// Find added functions in deterministic order
	functionKeys := sortedKeys(newFunctions)
	for _, key := range functionKeys {
		function := newFunctions[key]
		if _, exists := oldFunctions[key]; !exists {
			diff.AddedFunctions = append(diff.AddedFunctions, function)
		}
	}

	// Find dropped functions in deterministic order
	oldFunctionKeys := sortedKeys(oldFunctions)
	for _, key := range oldFunctionKeys {
		function := oldFunctions[key]
		if _, exists := newFunctions[key]; !exists {
			diff.DroppedFunctions = append(diff.DroppedFunctions, function)
		}
	}

	// Find modified functions in deterministic order
	for _, key := range functionKeys {
		newFunction := newFunctions[key]
		if oldFunction, exists := oldFunctions[key]; exists {
			if !functionsEqual(oldFunction, newFunction) {
				diff.ModifiedFunctions = append(diff.ModifiedFunctions, &FunctionDiff{
					Old: oldFunction,
					New: newFunction,
				})
			}
		}
	}

	// Compare procedures across all schemas
	oldProcedures := make(map[string]*ir.Procedure)
	newProcedures := make(map[string]*ir.Procedure)

	// Extract procedures from all schemas in oldIR in deterministic order
	for _, dbSchema := range oldIR.Schemas {
		procNames := sortedKeys(dbSchema.Procedures)
		for _, procName := range procNames {
			procedure := dbSchema.Procedures[procName]
			// Use schema.name(arguments) as key to distinguish procedures with different signatures
			key := procedure.Schema + "." + procName + "(" + procedure.Arguments + ")"
			oldProcedures[key] = procedure
		}
	}

	// Extract procedures from all schemas in newIR in deterministic order
	for _, dbSchema := range newIR.Schemas {
		procNames := sortedKeys(dbSchema.Procedures)
		for _, procName := range procNames {
			procedure := dbSchema.Procedures[procName]
			// Use schema.name(arguments) as key to distinguish procedures with different signatures
			key := procedure.Schema + "." + procName + "(" + procedure.Arguments + ")"
			newProcedures[key] = procedure
		}
	}

	// Find added procedures in deterministic order
	procedureKeys := sortedKeys(newProcedures)
	for _, key := range procedureKeys {
		procedure := newProcedures[key]
		if _, exists := oldProcedures[key]; !exists {
			diff.AddedProcedures = append(diff.AddedProcedures, procedure)
		}
	}

	// Find dropped procedures in deterministic order
	oldProcedureKeys := sortedKeys(oldProcedures)
	for _, key := range oldProcedureKeys {
		procedure := oldProcedures[key]
		if _, exists := newProcedures[key]; !exists {
			diff.DroppedProcedures = append(diff.DroppedProcedures, procedure)
		}
	}

	// Find modified procedures in deterministic order
	for _, key := range procedureKeys {
		newProcedure := newProcedures[key]
		if oldProcedure, exists := oldProcedures[key]; exists {
			if !proceduresEqual(oldProcedure, newProcedure) {
				diff.ModifiedProcedures = append(diff.ModifiedProcedures, &ProcedureDiff{
					Old: oldProcedure,
					New: newProcedure,
				})
			}
		}
	}

	// Compare types across all schemas
	oldTypes := make(map[string]*ir.Type)
	newTypes := make(map[string]*ir.Type)

	// Extract types from all schemas in oldIR in deterministic order
	for _, dbSchema := range oldIR.Schemas {
		typeNames := sortedKeys(dbSchema.Types)
		for _, typeName := range typeNames {
			typeObj := dbSchema.Types[typeName]
			key := typeObj.Schema + "." + typeName
			oldTypes[key] = typeObj
		}
	}

	// Extract types from all schemas in newIR in deterministic order
	for _, dbSchema := range newIR.Schemas {
		typeNames := sortedKeys(dbSchema.Types)
		for _, typeName := range typeNames {
			typeObj := dbSchema.Types[typeName]
			key := typeObj.Schema + "." + typeName
			newTypes[key] = typeObj
		}
	}

	// Find added types in deterministic order
	typeKeys := sortedKeys(newTypes)
	for _, key := range typeKeys {
		typeObj := newTypes[key]
		if _, exists := oldTypes[key]; !exists {
			diff.AddedTypes = append(diff.AddedTypes, typeObj)
		}
	}

	// Find dropped types in deterministic order
	oldTypeKeys := sortedKeys(oldTypes)
	for _, key := range oldTypeKeys {
		typeObj := oldTypes[key]
		if _, exists := newTypes[key]; !exists {
			diff.DroppedTypes = append(diff.DroppedTypes, typeObj)
		}
	}

	// Find modified types in deterministic order
	for _, key := range typeKeys {
		newType := newTypes[key]
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

	// Extract views from all schemas in oldIR in deterministic order
	for _, dbSchema := range oldIR.Schemas {
		viewNames := sortedKeys(dbSchema.Views)
		for _, viewName := range viewNames {
			view := dbSchema.Views[viewName]
			key := view.Schema + "." + viewName
			oldViews[key] = view
		}
	}

	// Extract views from all schemas in newIR in deterministic order
	for _, dbSchema := range newIR.Schemas {
		viewNames := sortedKeys(dbSchema.Views)
		for _, viewName := range viewNames {
			view := dbSchema.Views[viewName]
			key := view.Schema + "." + viewName
			newViews[key] = view
		}
	}

	// Find added views in deterministic order
	viewKeys := sortedKeys(newViews)
	for _, key := range viewKeys {
		view := newViews[key]
		if _, exists := oldViews[key]; !exists {
			diff.AddedViews = append(diff.AddedViews, view)
		}
	}

	// Find dropped views in deterministic order
	oldViewKeys := sortedKeys(oldViews)
	for _, key := range oldViewKeys {
		view := oldViews[key]
		if _, exists := newViews[key]; !exists {
			diff.DroppedViews = append(diff.DroppedViews, view)
		}
	}

	// Find modified views in deterministic order
	for _, key := range viewKeys {
		newView := newViews[key]
		if oldView, exists := oldViews[key]; exists {
			if !viewsEqual(oldView, newView) {
				diff.ModifiedViews = append(diff.ModifiedViews, &ViewDiff{
					Old: oldView,
					New: newView,
				})
			}
		}
	}

	// Sort tables and views topologically for consistent ordering
	diff.AddedTables = topologicallySortTables(diff.AddedTables)
	diff.DroppedTables = reverseSlice(topologicallySortTables(diff.DroppedTables))
	diff.AddedViews = topologicallySortViews(diff.AddedViews)
	diff.DroppedViews = reverseSlice(topologicallySortViews(diff.DroppedViews))
	
	// Sort ModifiedTables alphabetically for consistent ordering
	// (topological sorting isn't needed for modified tables since they already exist)
	sortModifiedTables(diff.ModifiedTables)
	
	// Sort individual table objects (indexes, triggers, policies, constraints) within each table
	sortTableObjects(diff.ModifiedTables)

	return diff
}

// GenerateMigrationSQL generates SQL statements for the diff
func GenerateMigrationSQL(d *DDLDiff, targetSchema string) string {
	w := NewSQLWriter(false)

	// First: Drop operations (in reverse dependency order)
	d.generateDropSQL(w, targetSchema)

	// Then: Create operations (in dependency order)
	d.generateCreateSQL(w, targetSchema, true)

	// Finally: Modify operations
	d.generateModifySQL(w, targetSchema)

	return w.String()
}

// GenerateDumpSQL generates a complete database dump SQL from an IR schema
// This is equivalent to diff between the schema and an empty schema
func GenerateDumpSQL(schema *ir.IR, targetSchema string) string {
	w := NewSQLWriter(true)

	// Create an empty schema for comparison
	emptyIR := ir.NewIR()

	// Generate diff between the schema and empty schema
	diff := Diff(emptyIR, schema)

	// Dump only contains Create statement
	diff.generateCreateSQL(w, targetSchema, false)

	return w.String()
}

// generateCreateSQL generates CREATE statements in dependency order
func (d *DDLDiff) generateCreateSQL(w *SQLWriter, targetSchema string, compare bool) {
	// Note: Schema creation is out of scope for schema-level comparisons


	// Create types
	generateCreateTypesSQL(w, d.AddedTypes, targetSchema)

	// Create functions
	generateCreateFunctionsSQL(w, d.AddedFunctions, targetSchema)

	// Create procedures
	generateCreateProceduresSQL(w, d.AddedProcedures, targetSchema)

	// Create tables with co-located indexes, constraints, triggers, and RLS
	generateCreateTablesSQL(w, d.AddedTables, targetSchema, compare)

	// Create views
	generateCreateViewsSQL(w, d.AddedViews, targetSchema, compare)
}

// generateModifySQL generates ALTER statements
func (d *DDLDiff) generateModifySQL(w *SQLWriter, targetSchema string) {
	// Modify schemas
	// Note: Schema modification is out of scope for schema-level comparisons

	// Modify types
	generateModifyTypesSQL(w, d.ModifiedTypes, targetSchema)

	// Modify tables
	generateModifyTablesSQL(w, d.ModifiedTables, targetSchema)

	// Modify views
	generateModifyViewsSQL(w, d.ModifiedViews, targetSchema)

	// Modify functions
	generateModifyFunctionsSQL(w, d.ModifiedFunctions, targetSchema)

	// Modify procedures
	generateModifyProceduresSQL(w, d.ModifiedProcedures, targetSchema)

}

// generateDropSQL generates DROP statements in reverse dependency order
func (d *DDLDiff) generateDropSQL(w *SQLWriter, targetSchema string) {

	// Drop functions
	generateDropFunctionsSQL(w, d.DroppedFunctions, targetSchema)

	// Drop procedures
	generateDropProceduresSQL(w, d.DroppedProcedures, targetSchema)

	// Drop views
	generateDropViewsSQL(w, d.DroppedViews, targetSchema)

	// Drop tables
	generateDropTablesSQL(w, d.DroppedTables, targetSchema)

	// Drop types
	generateDropTypesSQL(w, d.DroppedTypes, targetSchema)


	// Drop schemas
	// Note: Schema deletion is out of scope for schema-level comparisons
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

// qualifyEntityName returns the properly qualified entity name based on target schema
// If entity is in target schema, returns just the name, otherwise returns schema.name
func qualifyEntityName(entitySchema, entityName, targetSchema string) string {
	if entitySchema == targetSchema {
		return entityName
	}
	return fmt.Sprintf("%s.%s", entitySchema, entityName)
}

// quoteString properly quotes a string for SQL, handling single quotes
func quoteString(s string) string {
	// Escape single quotes by doubling them
	escaped := strings.ReplaceAll(s, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
}

// sortModifiedTables sorts modified tables alphabetically by schema then name
func sortModifiedTables(tables []*TableDiff) {
	sort.Slice(tables, func(i, j int) bool {
		// First sort by schema, then by table name
		if tables[i].Table.Schema != tables[j].Table.Schema {
			return tables[i].Table.Schema < tables[j].Table.Schema
		}
		return tables[i].Table.Name < tables[j].Table.Name
	})
}

// sortTableObjects sorts the objects within each table diff for consistent ordering
func sortTableObjects(tables []*TableDiff) {
	for _, tableDiff := range tables {
		// Sort dropped constraints
		sort.Slice(tableDiff.DroppedConstraints, func(i, j int) bool {
			return tableDiff.DroppedConstraints[i].Name < tableDiff.DroppedConstraints[j].Name
		})
		
		// Sort added constraints
		sort.Slice(tableDiff.AddedConstraints, func(i, j int) bool {
			return tableDiff.AddedConstraints[i].Name < tableDiff.AddedConstraints[j].Name
		})
		
		// Sort dropped policies
		sort.Slice(tableDiff.DroppedPolicies, func(i, j int) bool {
			return tableDiff.DroppedPolicies[i].Name < tableDiff.DroppedPolicies[j].Name
		})
		
		// Sort added policies
		sort.Slice(tableDiff.AddedPolicies, func(i, j int) bool {
			return tableDiff.AddedPolicies[i].Name < tableDiff.AddedPolicies[j].Name
		})
		
		// Sort modified policies
		sort.Slice(tableDiff.ModifiedPolicies, func(i, j int) bool {
			return tableDiff.ModifiedPolicies[i].New.Name < tableDiff.ModifiedPolicies[j].New.Name
		})
		
		// Sort dropped triggers
		sort.Slice(tableDiff.DroppedTriggers, func(i, j int) bool {
			return tableDiff.DroppedTriggers[i].Name < tableDiff.DroppedTriggers[j].Name
		})
		
		// Sort added triggers
		sort.Slice(tableDiff.AddedTriggers, func(i, j int) bool {
			return tableDiff.AddedTriggers[i].Name < tableDiff.AddedTriggers[j].Name
		})
		
		// Sort modified triggers
		sort.Slice(tableDiff.ModifiedTriggers, func(i, j int) bool {
			return tableDiff.ModifiedTriggers[i].New.Name < tableDiff.ModifiedTriggers[j].New.Name
		})
		
		// Sort dropped indexes
		sort.Slice(tableDiff.DroppedIndexes, func(i, j int) bool {
			return tableDiff.DroppedIndexes[i].Name < tableDiff.DroppedIndexes[j].Name
		})
		
		// Sort added indexes
		sort.Slice(tableDiff.AddedIndexes, func(i, j int) bool {
			return tableDiff.AddedIndexes[i].Name < tableDiff.AddedIndexes[j].Name
		})
		
		// Sort columns by position for consistent ordering
		sort.Slice(tableDiff.DroppedColumns, func(i, j int) bool {
			return tableDiff.DroppedColumns[i].Position < tableDiff.DroppedColumns[j].Position
		})
		
		sort.Slice(tableDiff.AddedColumns, func(i, j int) bool {
			return tableDiff.AddedColumns[i].Position < tableDiff.AddedColumns[j].Position
		})
		
		sort.Slice(tableDiff.ModifiedColumns, func(i, j int) bool {
			return tableDiff.ModifiedColumns[i].New.Position < tableDiff.ModifiedColumns[j].New.Position
		})
	}
}

// sortedKeys returns sorted keys from a map[string]T
func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
