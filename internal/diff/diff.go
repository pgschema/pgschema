package diff

import (
	"fmt"

	"github.com/pgschema/pgschema/internal/ir"
)

// getTableNameWithSchema returns the table name with schema qualification only when necessary
// If the table schema is different from the target schema, it returns "schema.table"
// If they are the same, it returns just "table"
func getTableNameWithSchema(tableSchema, tableName, targetSchema string) string {
	if tableSchema != targetSchema {
		return fmt.Sprintf("%s.%s", tableSchema, tableName)
	}
	return tableName
}

// Diff compares two IR schemas directly and returns the differences
func Diff(oldIR, newIR *ir.IR) *DDLDiff {
	diff := &DDLDiff{
		AddedSchemas:      []*ir.Schema{},
		DroppedSchemas:    []*ir.Schema{},
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
		AddedPolicies:     []*ir.RLSPolicy{},
		DroppedPolicies:   []*ir.RLSPolicy{},
		ModifiedPolicies:  []*PolicyDiff{},
		RLSChanges:        []*RLSChange{},
	}

	// Compare schemas first
	for name, newDBSchema := range newIR.Schemas {
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

	// Find dropped schemas
	for name, oldDBSchema := range oldIR.Schemas {
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

	// Compare extensions
	oldExtensions := make(map[string]*ir.Extension)
	newExtensions := make(map[string]*ir.Extension)

	if oldIR.Extensions != nil {
		for name, ext := range oldIR.Extensions {
			oldExtensions[name] = ext
		}
	}

	if newIR.Extensions != nil {
		for name, ext := range newIR.Extensions {
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

	// Extract functions from all schemas in oldIR
	for _, dbSchema := range oldIR.Schemas {
		for funcName, function := range dbSchema.Functions {
			// Use schema.name(arguments) as key to distinguish functions with different signatures
			key := function.Schema + "." + funcName + "(" + function.Arguments + ")"
			oldFunctions[key] = function
		}
	}

	// Extract functions from all schemas in newIR
	for _, dbSchema := range newIR.Schemas {
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

	// Extract indexes from all schemas and tables in oldIR
	for _, dbSchema := range oldIR.Schemas {
		for _, table := range dbSchema.Tables {
			for _, index := range table.Indexes {
				key := index.Schema + "." + index.Table + "." + index.Name
				oldIndexes[key] = index
			}
		}
	}

	// Extract indexes from all schemas and tables in newIR
	for _, dbSchema := range newIR.Schemas {
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

	// Extract types from all schemas in oldIR
	for _, dbSchema := range oldIR.Schemas {
		for typeName, typeObj := range dbSchema.Types {
			key := typeObj.Schema + "." + typeName
			oldTypes[key] = typeObj
		}
	}

	// Extract types from all schemas in newIR
	for _, dbSchema := range newIR.Schemas {
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

	// Extract views from all schemas in oldIR
	for _, dbSchema := range oldIR.Schemas {
		for viewName, view := range dbSchema.Views {
			key := view.Schema + "." + viewName
			oldViews[key] = view
		}
	}

	// Extract views from all schemas in newIR
	for _, dbSchema := range newIR.Schemas {
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

	// Extract triggers from all tables in all schemas in oldIR
	for _, dbSchema := range oldIR.Schemas {
		for _, table := range dbSchema.Tables {
			for triggerName, trigger := range table.Triggers {
				key := trigger.Schema + "." + trigger.Table + "." + triggerName
				oldTriggers[key] = trigger
			}
		}
	}

	// Extract triggers from all tables in all schemas in newIR
	for _, dbSchema := range newIR.Schemas {
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

	// Compare RLS policies across all tables
	oldPolicies := make(map[string]*ir.RLSPolicy)
	newPolicies := make(map[string]*ir.RLSPolicy)

	// Extract policies from all tables in all schemas in oldIR
	for _, dbSchema := range oldIR.Schemas {
		for _, table := range dbSchema.Tables {
			for policyName, policy := range table.Policies {
				key := policy.Schema + "." + policy.Table + "." + policyName
				oldPolicies[key] = policy
			}
		}
	}

	// Extract policies from all tables in all schemas in newIR
	for _, dbSchema := range newIR.Schemas {
		for _, table := range dbSchema.Tables {
			for policyName, policy := range table.Policies {
				key := policy.Schema + "." + policy.Table + "." + policyName
				newPolicies[key] = policy
			}
		}
	}

	// Find added policies
	for key, policy := range newPolicies {
		if _, exists := oldPolicies[key]; !exists {
			diff.AddedPolicies = append(diff.AddedPolicies, policy)
		}
	}

	// Find dropped policies
	for key, policy := range oldPolicies {
		if _, exists := newPolicies[key]; !exists {
			diff.DroppedPolicies = append(diff.DroppedPolicies, policy)
		}
	}

	// Find modified policies
	for key, newPolicy := range newPolicies {
		if oldPolicy, exists := oldPolicies[key]; exists {
			if !policiesEqual(oldPolicy, newPolicy) {
				diff.ModifiedPolicies = append(diff.ModifiedPolicies, &PolicyDiff{
					Old: oldPolicy,
					New: newPolicy,
				})
			}
		}
	}

	// Check for RLS enable/disable changes
	for key, newTable := range newTables {
		if oldTable, exists := oldTables[key]; exists {
			if oldTable.RLSEnabled != newTable.RLSEnabled {
				diff.RLSChanges = append(diff.RLSChanges, &RLSChange{
					Table:   newTable,
					Enabled: newTable.RLSEnabled,
				})
			}
		}
	}

	return diff
}

// GenerateMigrationSQL generates SQL statements for the migration using the unified SQL generator approach
func (d *DDLDiff) GenerateMigrationSQL() string {
	return d.GenerateMigrationSQLWithOptions(false, "public")
}

// GenerateMigrationSQLWithOptions generates SQL statements using the unified SQL generator approach
func (d *DDLDiff) GenerateMigrationSQLWithOptions(includeComments bool, targetSchema string) string {
	w := NewSQLWriterWithComments(includeComments)

	// Generate DDL in proper dependency order following SQL generator pattern

	// First: Drop operations (in reverse dependency order)
	d.generateDropSQL(w, targetSchema)

	// Then: Create operations (in dependency order)
	d.generateCreateSQL(w, targetSchema)

	// Finally: Modify operations
	d.generateModifySQL(w, targetSchema)

	return w.String()
}

// GenerateDumpSQL generates a complete database dump SQL from an IR schema
// This is equivalent to diff between the schema and an empty schema
func GenerateDumpSQL(schema *ir.IR, includeComments bool, targetSchema string) string {
	// Create an empty schema for comparison
	emptyIR := ir.NewIR()

	// Generate diff between the schema and empty schema
	diff := Diff(emptyIR, schema)

	w := NewSQLWriterWithComments(includeComments)

	// Dump only contains Create statement
	diff.generateCreateSQL(w, targetSchema)

	return w.String()
}

// GenerateMigrationSQL generates migration SQL between two schemas
func GenerateMigrationSQL(oldSchema, newSchema *ir.IR, includeComments bool, targetSchema string) string {
	// Generate diff between old and new schemas
	diff := Diff(oldSchema, newSchema)

	// Generate SQL using the unified diff approach
	return diff.GenerateMigrationSQLWithOptions(includeComments, targetSchema)
}

// generateDropSQL generates DROP statements in reverse dependency order
func (d *DDLDiff) generateDropSQL(w *SQLWriter, targetSchema string) {
	// Handle RLS disable changes first (before dropping policies)
	generateRLSDisableChangesSQL(w, d.RLSChanges, targetSchema)

	// Drop RLS policies 
	generateDropPoliciesSQL(w, d.DroppedPolicies, targetSchema)

	// Drop triggers
	d.generateDropTriggersSQL(w, d.DroppedTriggers, targetSchema)

	// Drop indexes
	generateDropIndexesSQL(w, d.DroppedIndexes, targetSchema)

	// Drop functions
	generateDropFunctionsSQL(w, d.DroppedFunctions, targetSchema)

	// Drop views
	d.generateDropViewsSQL(w, d.DroppedViews, targetSchema)

	// Drop tables
	d.generateDropTablesSQL(w, d.DroppedTables, targetSchema)

	// Drop types
	generateDropTypesSQL(w, d.DroppedTypes, targetSchema)

	// Drop extensions
	generateDropExtensionsSQL(w, d.DroppedExtensions, targetSchema)

	// Drop schemas
	generateDropSchemasSQL(w, d.DroppedSchemas, targetSchema)
}

// generateCreateSQL generates CREATE statements in dependency order
func (d *DDLDiff) generateCreateSQL(w *SQLWriter, targetSchema string) {
	// Create schemas first
	generateCreateSchemasSQL(w, d.AddedSchemas, targetSchema)

	// Create extensions
	generateCreateExtensionsSQL(w, d.AddedExtensions, targetSchema)

	// Create types
	generateCreateTypesSQL(w, d.AddedTypes, targetSchema)

	// Create tables with co-located indexes, constraints, triggers, and RLS
	d.generateCreateTablesSQL(w, d.AddedTables, targetSchema)

	// Create views
	d.generateCreateViewsSQL(w, d.AddedViews, targetSchema)

	// Create functions
	generateCreateFunctionsSQL(w, d.AddedFunctions, targetSchema)

	// Create indexes (for indexes added to existing tables)
	// Skip if this is a dump scenario (only added tables, no dropped/modified) - indexes are already generated with tables
	isDumpScenario := len(d.AddedTables) > 0 && len(d.DroppedTables) == 0 && len(d.ModifiedTables) == 0
	if !isDumpScenario {
		generateCreateIndexesSQL(w, d.AddedIndexes, targetSchema)
		
		// Handle RLS enable changes (before creating policies) - only for diff scenarios
		generateRLSEnableChangesSQL(w, d.RLSChanges, targetSchema)

		// Create policies - only for diff scenarios
		generateCreatePoliciesSQL(w, d.AddedPolicies, targetSchema)

		// Create triggers - only for diff scenarios
		d.generateCreateTriggersSQL(w, d.AddedTriggers, targetSchema)
	}
}

// generateModifySQL generates ALTER statements
func (d *DDLDiff) generateModifySQL(w *SQLWriter, targetSchema string) {
	// Modify schemas
	generateModifySchemasSQL(w, d.ModifiedSchemas, targetSchema)

	// Modify types
	generateModifyTypesSQL(w, d.ModifiedTypes, targetSchema)

	// Modify tables
	d.generateModifyTablesSQL(w, d.ModifiedTables, targetSchema)

	// Modify views
	d.generateModifyViewsSQL(w, d.ModifiedViews, targetSchema)

	// Modify functions
	generateModifyFunctionsSQL(w, d.ModifiedFunctions, targetSchema)

	// Modify triggers
	d.generateModifyTriggersSQL(w, d.ModifiedTriggers, targetSchema)

	// Modify policies
	generateModifyPoliciesSQL(w, d.ModifiedPolicies, targetSchema)
}
