package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

type DDLDiff struct {
	addedSchemas       []*ir.Schema
	droppedSchemas     []*ir.Schema
	modifiedSchemas    []*schemaDiff
	addedTables        []*ir.Table
	droppedTables      []*ir.Table
	modifiedTables     []*tableDiff
	addedViews         []*ir.View
	droppedViews       []*ir.View
	modifiedViews      []*viewDiff
	addedFunctions     []*ir.Function
	droppedFunctions   []*ir.Function
	modifiedFunctions  []*functionDiff
	addedProcedures    []*ir.Procedure
	droppedProcedures  []*ir.Procedure
	modifiedProcedures []*procedureDiff
	addedTypes         []*ir.Type
	droppedTypes       []*ir.Type
	modifiedTypes      []*typeDiff
	addedSequences     []*ir.Sequence
	droppedSequences   []*ir.Sequence
	modifiedSequences  []*sequenceDiff
}

// schemaDiff represents changes to a schema
type schemaDiff struct {
	Old *ir.Schema
	New *ir.Schema
}

// functionDiff represents changes to a function
type functionDiff struct {
	Old *ir.Function
	New *ir.Function
}

// procedureDiff represents changes to a procedure
type procedureDiff struct {
	Old *ir.Procedure
	New *ir.Procedure
}

// typeDiff represents changes to a type
type typeDiff struct {
	Old *ir.Type
	New *ir.Type
}

// sequenceDiff represents changes to a sequence
type sequenceDiff struct {
	Old *ir.Sequence
	New *ir.Sequence
}

// triggerDiff represents changes to a trigger
type triggerDiff struct {
	Old *ir.Trigger
	New *ir.Trigger
}

// viewDiff represents changes to a view
type viewDiff struct {
	Old            *ir.View
	New            *ir.View
	CommentChanged bool
	OldComment     string
	NewComment     string
}

// tableDiff represents changes to a table
type tableDiff struct {
	Table              *ir.Table
	AddedColumns       []*ir.Column
	DroppedColumns     []*ir.Column
	ModifiedColumns    []*columnDiff
	AddedConstraints   []*ir.Constraint
	DroppedConstraints []*ir.Constraint
	AddedIndexes       []*ir.Index
	DroppedIndexes     []*ir.Index
	ModifiedIndexes    []*indexDiff
	AddedTriggers      []*ir.Trigger
	DroppedTriggers    []*ir.Trigger
	ModifiedTriggers   []*triggerDiff
	AddedPolicies      []*ir.RLSPolicy
	DroppedPolicies    []*ir.RLSPolicy
	ModifiedPolicies   []*policyDiff
	RLSChanges         []*rlsChange
	CommentChanged     bool
	OldComment         string
	NewComment         string
}

// columnDiff represents changes to a column
type columnDiff struct {
	Old *ir.Column
	New *ir.Column
}

// indexDiff represents changes to an index
type indexDiff struct {
	Old *ir.Index
	New *ir.Index
}

// policyDiff represents changes to a policy
type policyDiff struct {
	Old *ir.RLSPolicy
	New *ir.RLSPolicy
}

// rlsChange represents enabling/disabling Row Level Security on a table
type rlsChange struct {
	Table   *ir.Table
	Enabled bool // true to enable, false to disable
}

// Diff compares two IR schemas directly and returns the differences
func Diff(oldIR, newIR *ir.IR) *DDLDiff {
	diff := &DDLDiff{
		addedSchemas:       []*ir.Schema{},
		droppedSchemas:     []*ir.Schema{},
		modifiedSchemas:    []*schemaDiff{},
		addedTables:        []*ir.Table{},
		droppedTables:      []*ir.Table{},
		modifiedTables:     []*tableDiff{},
		addedViews:         []*ir.View{},
		droppedViews:       []*ir.View{},
		modifiedViews:      []*viewDiff{},
		addedFunctions:     []*ir.Function{},
		droppedFunctions:   []*ir.Function{},
		modifiedFunctions:  []*functionDiff{},
		addedProcedures:    []*ir.Procedure{},
		droppedProcedures:  []*ir.Procedure{},
		modifiedProcedures: []*procedureDiff{},
		addedTypes:         []*ir.Type{},
		droppedTypes:       []*ir.Type{},
		modifiedTypes:      []*typeDiff{},
		addedSequences:     []*ir.Sequence{},
		droppedSequences:   []*ir.Sequence{},
		modifiedSequences:  []*sequenceDiff{},
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
				diff.modifiedSchemas = append(diff.modifiedSchemas, &schemaDiff{
					Old: oldDBSchema,
					New: newDBSchema,
				})
			}
		} else {
			// Schema was added
			diff.addedSchemas = append(diff.addedSchemas, newDBSchema)
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
			diff.droppedSchemas = append(diff.droppedSchemas, oldDBSchema)
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
			diff.addedTables = append(diff.addedTables, table)
		}
	}

	// Find dropped tables
	for key, table := range oldTables {
		if _, exists := newTables[key]; !exists {
			diff.droppedTables = append(diff.droppedTables, table)
		}
	}

	// Find modified tables
	for key, newTable := range newTables {
		if oldTable, exists := oldTables[key]; exists {
			if tableDiff := diffTables(oldTable, newTable); tableDiff != nil {
				diff.modifiedTables = append(diff.modifiedTables, tableDiff)
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
			diff.addedFunctions = append(diff.addedFunctions, function)
		}
	}

	// Find dropped functions in deterministic order
	oldFunctionKeys := sortedKeys(oldFunctions)
	for _, key := range oldFunctionKeys {
		function := oldFunctions[key]
		if _, exists := newFunctions[key]; !exists {
			diff.droppedFunctions = append(diff.droppedFunctions, function)
		}
	}

	// Find modified functions in deterministic order
	for _, key := range functionKeys {
		newFunction := newFunctions[key]
		if oldFunction, exists := oldFunctions[key]; exists {
			if !functionsEqual(oldFunction, newFunction) {
				diff.modifiedFunctions = append(diff.modifiedFunctions, &functionDiff{
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
			diff.addedProcedures = append(diff.addedProcedures, procedure)
		}
	}

	// Find dropped procedures in deterministic order
	oldProcedureKeys := sortedKeys(oldProcedures)
	for _, key := range oldProcedureKeys {
		procedure := oldProcedures[key]
		if _, exists := newProcedures[key]; !exists {
			diff.droppedProcedures = append(diff.droppedProcedures, procedure)
		}
	}

	// Find modified procedures in deterministic order
	for _, key := range procedureKeys {
		newProcedure := newProcedures[key]
		if oldProcedure, exists := oldProcedures[key]; exists {
			if !proceduresEqual(oldProcedure, newProcedure) {
				diff.modifiedProcedures = append(diff.modifiedProcedures, &procedureDiff{
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
			diff.addedTypes = append(diff.addedTypes, typeObj)
		}
	}

	// Find dropped types in deterministic order
	oldTypeKeys := sortedKeys(oldTypes)
	for _, key := range oldTypeKeys {
		typeObj := oldTypes[key]
		if _, exists := newTypes[key]; !exists {
			diff.droppedTypes = append(diff.droppedTypes, typeObj)
		}
	}

	// Find modified types in deterministic order
	for _, key := range typeKeys {
		newType := newTypes[key]
		if oldType, exists := oldTypes[key]; exists {
			if !typesEqual(oldType, newType) {
				diff.modifiedTypes = append(diff.modifiedTypes, &typeDiff{
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
			diff.addedViews = append(diff.addedViews, view)
		}
	}

	// Find dropped views in deterministic order
	oldViewKeys := sortedKeys(oldViews)
	for _, key := range oldViewKeys {
		view := oldViews[key]
		if _, exists := newViews[key]; !exists {
			diff.droppedViews = append(diff.droppedViews, view)
		}
	}

	// Find modified views in deterministic order
	for _, key := range viewKeys {
		newView := newViews[key]
		if oldView, exists := oldViews[key]; exists {
			structurallyDifferent := !viewsEqual(oldView, newView)
			commentChanged := oldView.Comment != newView.Comment

			if structurallyDifferent || commentChanged {
				// For materialized views with structural changes, use DROP + CREATE approach
				if newView.Materialized && structurallyDifferent {
					// Add old materialized view to dropped views
					diff.droppedViews = append(diff.droppedViews, oldView)
					// Add new materialized view to added views
					diff.addedViews = append(diff.addedViews, newView)
				} else {
					// For regular views or comment-only changes, use the modify approach
					viewDiff := &viewDiff{
						Old: oldView,
						New: newView,
					}

					// Check for comment changes
					if commentChanged {
						viewDiff.CommentChanged = true
						viewDiff.OldComment = oldView.Comment
						viewDiff.NewComment = newView.Comment
					}

					diff.modifiedViews = append(diff.modifiedViews, viewDiff)
				}
			}
		}
	}

	// Compare sequences across all schemas
	oldSequences := make(map[string]*ir.Sequence)
	newSequences := make(map[string]*ir.Sequence)

	// Extract sequences from all schemas in oldIR in deterministic order
	for _, dbSchema := range oldIR.Schemas {
		seqNames := sortedKeys(dbSchema.Sequences)
		for _, seqName := range seqNames {
			seq := dbSchema.Sequences[seqName]
			key := seq.Schema + "." + seqName
			oldSequences[key] = seq
		}
	}

	// Extract sequences from all schemas in newIR in deterministic order
	for _, dbSchema := range newIR.Schemas {
		seqNames := sortedKeys(dbSchema.Sequences)
		for _, seqName := range seqNames {
			seq := dbSchema.Sequences[seqName]
			key := seq.Schema + "." + seqName
			newSequences[key] = seq
		}
	}

	// Find added sequences in deterministic order
	seqKeys := sortedKeys(newSequences)
	for _, key := range seqKeys {
		seq := newSequences[key]
		if _, exists := oldSequences[key]; !exists {
			// Skip sequences owned by table columns (created by SERIAL)
			if seq.OwnedByTable != "" && seq.OwnedByColumn != "" {
				continue
			}
			diff.addedSequences = append(diff.addedSequences, seq)
		}
	}

	// Find dropped sequences in deterministic order
	oldSeqKeys := sortedKeys(oldSequences)
	for _, key := range oldSeqKeys {
		seq := oldSequences[key]
		if _, exists := newSequences[key]; !exists {
			// Skip sequences owned by table columns (created by SERIAL)
			if seq.OwnedByTable != "" && seq.OwnedByColumn != "" {
				continue
			}
			diff.droppedSequences = append(diff.droppedSequences, seq)
		}
	}

	// Find modified sequences in deterministic order
	for _, key := range seqKeys {
		newSeq := newSequences[key]
		if oldSeq, exists := oldSequences[key]; exists {
			// Skip sequences owned by table columns (created by SERIAL)
			if (oldSeq.OwnedByTable != "" && oldSeq.OwnedByColumn != "") ||
				(newSeq.OwnedByTable != "" && newSeq.OwnedByColumn != "") {
				continue
			}
			if !sequencesEqual(oldSeq, newSeq) {
				diff.modifiedSequences = append(diff.modifiedSequences, &sequenceDiff{
					Old: oldSeq,
					New: newSeq,
				})
			}
		}
	}

	// Sort tables and views topologically for consistent ordering
	diff.addedTables = topologicallySortTables(diff.addedTables)
	diff.droppedTables = reverseSlice(topologicallySortTables(diff.droppedTables))
	diff.addedViews = topologicallySortViews(diff.addedViews)
	diff.droppedViews = reverseSlice(topologicallySortViews(diff.droppedViews))

	// Sort ModifiedTables alphabetically for consistent ordering
	// (topological sorting isn't needed for modified tables since they already exist)
	sortModifiedTables(diff.modifiedTables)

	// Sort individual table objects (indexes, triggers, policies, constraints) within each table
	sortTableObjects(diff.modifiedTables)

	return diff
}

// CollectMigrationSQL populates the collector with SQL statements for the diff
// The collector must not be nil
func (d *DDLDiff) CollectMigrationSQL(targetSchema string, collector *SQLCollector) {
	// First: Drop operations (in reverse dependency order)
	d.generateDropSQL(targetSchema, collector)

	// Then: Create operations (in dependency order)
	d.generateCreateSQL(targetSchema, collector)

	// Finally: Modify operations
	d.generateModifySQL(targetSchema, collector)
}

// generateCreateSQL generates CREATE statements in dependency order
func (d *DDLDiff) generateCreateSQL(targetSchema string, collector *SQLCollector) {
	// Note: Schema creation is out of scope for schema-level comparisons

	// Create types
	generateCreateTypesSQL(d.addedTypes, targetSchema, collector)

	// Create sequences
	generateCreateSequencesSQL(d.addedSequences, targetSchema, collector)

	// Create tables with co-located indexes, constraints, triggers, and RLS
	generateCreateTablesSQL(d.addedTables, targetSchema, collector)

	// Create functions (functions may depend on tables)
	generateCreateFunctionsSQL(d.addedFunctions, targetSchema, collector)

	// Create procedures (procedures may depend on tables)
	generateCreateProceduresSQL(d.addedProcedures, targetSchema, collector)

	// Create triggers (triggers may depend on functions/procedures)
	generateCreateTriggersFromTables(d.addedTables, targetSchema, collector)

	// Create views
	generateCreateViewsSQL(d.addedViews, targetSchema, collector)
}

// generateModifySQL generates ALTER statements
func (d *DDLDiff) generateModifySQL(targetSchema string, collector *SQLCollector) {
	// Modify schemas
	// Note: Schema modification is out of scope for schema-level comparisons

	// Modify types
	generateModifyTypesSQL(d.modifiedTypes, targetSchema, collector)

	// Modify sequences
	generateModifySequencesSQL(d.modifiedSequences, targetSchema, collector)

	// Modify tables
	generateModifyTablesSQL(d.modifiedTables, targetSchema, collector)

	// Modify views
	generateModifyViewsSQL(d.modifiedViews, targetSchema, collector)

	// Modify functions
	generateModifyFunctionsSQL(d.modifiedFunctions, targetSchema, collector)

	// Modify procedures
	generateModifyProceduresSQL(d.modifiedProcedures, targetSchema, collector)

}

// generateDropSQL generates DROP statements in reverse dependency order
func (d *DDLDiff) generateDropSQL(targetSchema string, collector *SQLCollector) {

	// Drop functions
	generateDropFunctionsSQL(d.droppedFunctions, targetSchema, collector)

	// Drop procedures
	generateDropProceduresSQL(d.droppedProcedures, targetSchema, collector)

	// Drop views
	generateDropViewsSQL(d.droppedViews, targetSchema, collector)

	// Drop tables
	generateDropTablesSQL(d.droppedTables, targetSchema, collector)

	// Drop sequences
	generateDropSequencesSQL(d.droppedSequences, targetSchema, collector)

	// Drop types
	generateDropTypesSQL(d.droppedTypes, targetSchema, collector)

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
func sortModifiedTables(tables []*tableDiff) {
	sort.Slice(tables, func(i, j int) bool {
		// First sort by schema, then by table name
		if tables[i].Table.Schema != tables[j].Table.Schema {
			return tables[i].Table.Schema < tables[j].Table.Schema
		}
		return tables[i].Table.Name < tables[j].Table.Name
	})
}

// sortTableObjects sorts the objects within each table diff for consistent ordering
func sortTableObjects(tables []*tableDiff) {
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

		// Sort modified indexes
		sort.Slice(tableDiff.ModifiedIndexes, func(i, j int) bool {
			return tableDiff.ModifiedIndexes[i].New.Name < tableDiff.ModifiedIndexes[j].New.Name
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
