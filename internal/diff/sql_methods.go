package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

// SQL generation methods for DDLDiff that follow the SQL generator pattern

// generateDropSchemasSQL generates DROP SCHEMA statements
func (d *DDLDiff) generateDropSchemasSQL(w *SQLWriter, schemas []*ir.Schema, targetSchema string) {
	// Sort schemas by name for consistent ordering
	sortedSchemas := make([]*ir.Schema, len(schemas))
	copy(sortedSchemas, schemas)
	sort.Slice(sortedSchemas, func(i, j int) bool {
		return sortedSchemas[i].Name < sortedSchemas[j].Name
	})

	for _, schema := range sortedSchemas {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE;", schema.Name)
		w.WriteStatementWithComment("SCHEMA", schema.Name, "", "", sql, targetSchema)
	}
}

// generateCreateSchemasSQL generates CREATE SCHEMA statements
func (d *DDLDiff) generateCreateSchemasSQL(w *SQLWriter, schemas []*ir.Schema, targetSchema string) {
	// Sort schemas by name for consistent ordering
	sortedSchemas := make([]*ir.Schema, len(schemas))
	copy(sortedSchemas, schemas)
	sort.Slice(sortedSchemas, func(i, j int) bool {
		return sortedSchemas[i].Name < sortedSchemas[j].Name
	})

	for _, schema := range sortedSchemas {
		// Skip creating the target schema if we're doing a schema-specific dump
		if schema.Name == targetSchema {
			continue
		}
		if sql := schema.GenerateSQL(); sql != "" {
			w.WriteDDLSeparator()
			w.WriteStatementWithComment("SCHEMA", schema.Name, "", "", sql, targetSchema)
		}
	}
}

// generateModifySchemasSQL generates ALTER SCHEMA statements
func (d *DDLDiff) generateModifySchemasSQL(w *SQLWriter, diffs []*SchemaDiff, targetSchema string) {
	for _, diff := range diffs {
		if diff.Old.Owner != diff.New.Owner {
			w.WriteDDLSeparator()
			sql := fmt.Sprintf("ALTER SCHEMA %s OWNER TO %s;", diff.New.Name, diff.New.Owner)
			w.WriteStatementWithComment("SCHEMA", diff.New.Name, "", "", sql, targetSchema)
		}
	}
}

// generateDropExtensionsSQL generates DROP EXTENSION statements
func (d *DDLDiff) generateDropExtensionsSQL(w *SQLWriter, extensions []*ir.Extension, targetSchema string) {
	// Sort extensions by name for consistent ordering
	sortedExtensions := make([]*ir.Extension, len(extensions))
	copy(sortedExtensions, extensions)
	sort.Slice(sortedExtensions, func(i, j int) bool {
		return sortedExtensions[i].Name < sortedExtensions[j].Name
	})

	for _, ext := range sortedExtensions {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP EXTENSION IF EXISTS %s;", ext.Name)
		w.WriteStatementWithComment("EXTENSION", ext.Name, ext.Schema, "", sql, targetSchema)
	}
}

// generateCreateExtensionsSQL generates CREATE EXTENSION statements
func (d *DDLDiff) generateCreateExtensionsSQL(w *SQLWriter, extensions []*ir.Extension, targetSchema string) {
	// Sort extensions by name for consistent ordering
	sortedExtensions := make([]*ir.Extension, len(extensions))
	copy(sortedExtensions, extensions)
	sort.Slice(sortedExtensions, func(i, j int) bool {
		return sortedExtensions[i].Name < sortedExtensions[j].Name
	})

	for _, ext := range sortedExtensions {
		w.WriteDDLSeparator()
		sql := ext.GenerateSQL()
		w.WriteStatementWithComment("EXTENSION", ext.Name, ext.Schema, "", sql, targetSchema)
	}
}

// generateDropTypesSQL generates DROP TYPE statements
func (d *DDLDiff) generateDropTypesSQL(w *SQLWriter, types []*ir.Type, targetSchema string) {
	// Sort types by name for consistent ordering
	sortedTypes := make([]*ir.Type, len(types))
	copy(sortedTypes, types)
	sort.Slice(sortedTypes, func(i, j int) bool {
		return sortedTypes[i].Name < sortedTypes[j].Name
	})

	for _, typeObj := range sortedTypes {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP TYPE IF EXISTS %s CASCADE;", typeObj.Name)
		w.WriteStatementWithComment("TYPE", typeObj.Name, typeObj.Schema, "", sql, targetSchema)
	}
}

// generateCreateTypesSQL generates CREATE TYPE statements
func (d *DDLDiff) generateCreateTypesSQL(w *SQLWriter, types []*ir.Type, targetSchema string) {
	// Sort types: CREATE TYPE statements first, then CREATE DOMAIN statements
	sortedTypes := make([]*ir.Type, len(types))
	copy(sortedTypes, types)
	sort.Slice(sortedTypes, func(i, j int) bool {
		typeI := sortedTypes[i]
		typeJ := sortedTypes[j]

		// Domain types should come after non-domain types
		if typeI.Kind == ir.TypeKindDomain && typeJ.Kind != ir.TypeKindDomain {
			return false
		}
		if typeI.Kind != ir.TypeKindDomain && typeJ.Kind == ir.TypeKindDomain {
			return true
		}

		// Within the same category, sort alphabetically by name
		return typeI.Name < typeJ.Name
	})

	for _, typeObj := range sortedTypes {
		w.WriteDDLSeparator()
		sql := typeObj.GenerateSQLWithOptions(false, targetSchema)
		
		// Use correct object type for comment
		var objectType string
		switch typeObj.Kind {
		case ir.TypeKindDomain:
			objectType = "DOMAIN"
		default:
			objectType = "TYPE"
		}
		
		w.WriteStatementWithComment(objectType, typeObj.Name, typeObj.Schema, "", sql, targetSchema)
	}
}

// generateModifyTypesSQL generates ALTER TYPE statements
func (d *DDLDiff) generateModifyTypesSQL(w *SQLWriter, diffs []*TypeDiff, targetSchema string) {
	for _, diff := range diffs {
		// Only ENUM types can be modified by adding values
		if diff.Old.Kind == ir.TypeKindEnum && diff.New.Kind == ir.TypeKindEnum {
			// Generate ALTER TYPE ... ADD VALUE statements for new enum values
			// This is a simplified implementation - in reality you'd need to diff the enum values
			w.WriteDDLSeparator()
			sql := fmt.Sprintf("-- ALTER TYPE %s ADD VALUE statements would go here", diff.New.Name)
			w.WriteStatementWithComment("TYPE", diff.New.Name, diff.New.Schema, "", sql, targetSchema)
		}
	}
}

// generateDropTablesSQL generates DROP TABLE statements
func (d *DDLDiff) generateDropTablesSQL(w *SQLWriter, tables []*ir.Table, targetSchema string) {
	// Group tables by schema for topological sorting
	tablesBySchema := make(map[string][]*ir.Table)
	for _, table := range tables {
		tablesBySchema[table.Schema] = append(tablesBySchema[table.Schema], table)
	}
	
	// Process each schema using reverse topological sorting for drops
	for schemaName, schemaTables := range tablesBySchema {
		// Build a temporary schema with just these tables for topological sorting
		tempSchema := &ir.Schema{
			Name:   schemaName,
			Tables: make(map[string]*ir.Table),
		}
		for _, table := range schemaTables {
			tempSchema.Tables[table.Name] = table
		}
		
		// Get topologically sorted table names, then reverse for drop order
		sortedTableNames := tempSchema.GetTopologicallySortedTableNames()
		
		// Reverse the order for dropping (dependencies first)
		for i := len(sortedTableNames) - 1; i >= 0; i-- {
			tableName := sortedTableNames[i]
			table := tempSchema.Tables[tableName]
			w.WriteDDLSeparator()
			sql := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", table.Name)
			w.WriteStatementWithComment("TABLE", table.Name, table.Schema, "", sql, targetSchema)
		}
	}
}

// generateCreateTablesSQL generates CREATE TABLE statements with co-located indexes, constraints, triggers, and RLS
func (d *DDLDiff) generateCreateTablesSQL(w *SQLWriter, tables []*ir.Table, targetSchema string) {
	// Group tables by schema for topological sorting
	tablesBySchema := make(map[string][]*ir.Table)
	for _, table := range tables {
		tablesBySchema[table.Schema] = append(tablesBySchema[table.Schema], table)
	}
	
	// Process each schema using topological sorting
	for schemaName, schemaTables := range tablesBySchema {
		// Build a temporary schema with just these tables for topological sorting
		tempSchema := &ir.Schema{
			Name:   schemaName,
			Tables: make(map[string]*ir.Table),
		}
		for _, table := range schemaTables {
			tempSchema.Tables[table.Name] = table
		}
		
		// Get topologically sorted table names for dependency-aware output
		sortedTableNames := tempSchema.GetTopologicallySortedTableNames()
		
		// Process tables in topological order
		for _, tableName := range sortedTableNames {
			table := tempSchema.Tables[tableName]
			
			// Create the table
			w.WriteDDLSeparator()
			sql := table.GenerateSQLWithOptions(false, targetSchema)
			w.WriteStatementWithComment("TABLE", table.Name, table.Schema, "", sql, targetSchema)
			
			// Co-locate table-related objects immediately after the table
			d.generateTableIndexes(w, table, targetSchema)
			d.generateTableConstraints(w, table, targetSchema)
			d.generateTableTriggers(w, table, targetSchema)
			d.generateTableRLS(w, table, targetSchema)
		}
	}
}

// generateModifyTablesSQL generates ALTER TABLE statements
func (d *DDLDiff) generateModifyTablesSQL(w *SQLWriter, diffs []*TableDiff, targetSchema string) {
	for _, diff := range diffs {
		statements := diff.GenerateMigrationSQL()
		for _, stmt := range statements {
			w.WriteDDLSeparator()
			w.WriteStatementWithComment("TABLE", diff.Table.Name, diff.Table.Schema, "", stmt, targetSchema)
		}
	}
}

// generateDropViewsSQL generates DROP VIEW statements
func (d *DDLDiff) generateDropViewsSQL(w *SQLWriter, views []*ir.View, targetSchema string) {
	// Group views by schema for topological sorting
	viewsBySchema := make(map[string][]*ir.View)
	for _, view := range views {
		viewsBySchema[view.Schema] = append(viewsBySchema[view.Schema], view)
	}
	
	// Process each schema using reverse topological sorting for drops
	for schemaName, schemaViews := range viewsBySchema {
		// Build a temporary schema with just these views for topological sorting
		tempSchema := &ir.Schema{
			Name:  schemaName,
			Views: make(map[string]*ir.View),
		}
		for _, view := range schemaViews {
			tempSchema.Views[view.Name] = view
		}
		
		// Get topologically sorted view names, then reverse for drop order
		sortedViewNames := tempSchema.GetTopologicallySortedViewNames()
		
		// Reverse the order for dropping (dependencies first)
		for i := len(sortedViewNames) - 1; i >= 0; i-- {
			viewName := sortedViewNames[i]
			view := tempSchema.Views[viewName]
			w.WriteDDLSeparator()
			sql := fmt.Sprintf("DROP VIEW IF EXISTS %s CASCADE;", view.Name)
			w.WriteStatementWithComment("VIEW", view.Name, view.Schema, "", sql, targetSchema)
		}
	}
}

// generateCreateViewsSQL generates CREATE VIEW statements
func (d *DDLDiff) generateCreateViewsSQL(w *SQLWriter, views []*ir.View, targetSchema string) {
	// Group views by schema for topological sorting
	viewsBySchema := make(map[string][]*ir.View)
	for _, view := range views {
		viewsBySchema[view.Schema] = append(viewsBySchema[view.Schema], view)
	}
	
	// Process each schema using topological sorting
	for schemaName, schemaViews := range viewsBySchema {
		// Build a temporary schema with just these views for topological sorting
		tempSchema := &ir.Schema{
			Name:  schemaName,
			Views: make(map[string]*ir.View),
		}
		for _, view := range schemaViews {
			tempSchema.Views[view.Name] = view
		}
		
		// Get topologically sorted view names for dependency-aware output
		sortedViewNames := tempSchema.GetTopologicallySortedViewNames()
		
		// Process views in topological order
		for _, viewName := range sortedViewNames {
			view := tempSchema.Views[viewName]
			w.WriteDDLSeparator()
			sql := view.GenerateSQLWithOptions(false, targetSchema)
			w.WriteStatementWithComment("VIEW", view.Name, view.Schema, "", sql, targetSchema)
		}
	}
}

// generateModifyViewsSQL generates ALTER VIEW statements
func (d *DDLDiff) generateModifyViewsSQL(w *SQLWriter, diffs []*ViewDiff, targetSchema string) {
	for _, diff := range diffs {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("CREATE OR REPLACE VIEW %s AS %s;", diff.New.Name, diff.New.Definition)
		w.WriteStatementWithComment("VIEW", diff.New.Name, diff.New.Schema, "", sql, targetSchema)
	}
}

// generateDropFunctionsSQL generates DROP FUNCTION statements
func (d *DDLDiff) generateDropFunctionsSQL(w *SQLWriter, functions []*ir.Function, targetSchema string) {
	// Sort functions by name for consistent ordering
	sortedFunctions := make([]*ir.Function, len(functions))
	copy(sortedFunctions, functions)
	sort.Slice(sortedFunctions, func(i, j int) bool {
		return sortedFunctions[i].Name < sortedFunctions[j].Name
	})

	for _, function := range sortedFunctions {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP FUNCTION IF EXISTS %s(%s) CASCADE;", function.Name, function.Arguments)
		w.WriteStatementWithComment("FUNCTION", function.Name, function.Schema, "", sql, targetSchema)
	}
}

// generateCreateFunctionsSQL generates CREATE FUNCTION statements
func (d *DDLDiff) generateCreateFunctionsSQL(w *SQLWriter, functions []*ir.Function, targetSchema string) {
	// Sort functions by name for consistent ordering
	sortedFunctions := make([]*ir.Function, len(functions))
	copy(sortedFunctions, functions)
	sort.Slice(sortedFunctions, func(i, j int) bool {
		return sortedFunctions[i].Name < sortedFunctions[j].Name
	})

	for _, function := range sortedFunctions {
		w.WriteDDLSeparator()
		sql := function.GenerateSQLWithOptions(false, targetSchema)
		w.WriteStatementWithComment("FUNCTION", function.Name, function.Schema, "", sql, targetSchema)
	}
}

// generateModifyFunctionsSQL generates ALTER FUNCTION statements
func (d *DDLDiff) generateModifyFunctionsSQL(w *SQLWriter, diffs []*FunctionDiff, targetSchema string) {
	for _, diff := range diffs {
		w.WriteDDLSeparator()
		sql := diff.New.GenerateSQLWithOptions(false, targetSchema)
		w.WriteStatementWithComment("FUNCTION", diff.New.Name, diff.New.Schema, "", sql, targetSchema)
	}
}

// generateDropIndexesSQL generates DROP INDEX statements
func (d *DDLDiff) generateDropIndexesSQL(w *SQLWriter, indexes []*ir.Index, targetSchema string) {
	// Sort indexes by name for consistent ordering
	sortedIndexes := make([]*ir.Index, len(indexes))
	copy(sortedIndexes, indexes)
	sort.Slice(sortedIndexes, func(i, j int) bool {
		return sortedIndexes[i].Name < sortedIndexes[j].Name
	})

	for _, index := range sortedIndexes {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP INDEX IF EXISTS %s;", index.Name)
		w.WriteStatementWithComment("INDEX", index.Name, index.Schema, "", sql, targetSchema)
	}
}

// generateCreateIndexesSQL generates CREATE INDEX statements
func (d *DDLDiff) generateCreateIndexesSQL(w *SQLWriter, indexes []*ir.Index, targetSchema string) {
	// Sort indexes by name for consistent ordering
	sortedIndexes := make([]*ir.Index, len(indexes))
	copy(sortedIndexes, indexes)
	sort.Slice(sortedIndexes, func(i, j int) bool {
		return sortedIndexes[i].Name < sortedIndexes[j].Name
	})

	for _, index := range sortedIndexes {
		// Skip primary key indexes as they're handled with constraints
		if index.IsPrimary {
			continue
		}
		
		w.WriteDDLSeparator()
		sql := index.Definition
		if !strings.HasSuffix(sql, ";") {
			sql += ";"
		}
		w.WriteStatementWithComment("INDEX", index.Name, index.Schema, "", sql, targetSchema)
	}
}

// generateDropTriggersSQL generates DROP TRIGGER statements
func (d *DDLDiff) generateDropTriggersSQL(w *SQLWriter, triggers []*ir.Trigger, targetSchema string) {
	// Sort triggers by name for consistent ordering
	sortedTriggers := make([]*ir.Trigger, len(triggers))
	copy(sortedTriggers, triggers)
	sort.Slice(sortedTriggers, func(i, j int) bool {
		return sortedTriggers[i].Name < sortedTriggers[j].Name
	})

	for _, trigger := range sortedTriggers {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s;", trigger.Name, trigger.Table)
		w.WriteStatementWithComment("TRIGGER", trigger.Name, trigger.Schema, "", sql, targetSchema)
	}
}

// generateCreateTriggersSQL generates CREATE TRIGGER statements
func (d *DDLDiff) generateCreateTriggersSQL(w *SQLWriter, triggers []*ir.Trigger, targetSchema string) {
	// Sort triggers by name for consistent ordering
	sortedTriggers := make([]*ir.Trigger, len(triggers))
	copy(sortedTriggers, triggers)
	sort.Slice(sortedTriggers, func(i, j int) bool {
		return sortedTriggers[i].Name < sortedTriggers[j].Name
	})

	for _, trigger := range sortedTriggers {
		w.WriteDDLSeparator()
		sql := trigger.GenerateSQLWithOptions(false, targetSchema)
		w.WriteStatementWithComment("TRIGGER", trigger.Name, trigger.Schema, "", sql, targetSchema)
	}
}

// generateModifyTriggersSQL generates ALTER TRIGGER statements
func (d *DDLDiff) generateModifyTriggersSQL(w *SQLWriter, diffs []*TriggerDiff, targetSchema string) {
	for _, diff := range diffs {
		w.WriteDDLSeparator()
		sql := diff.New.GenerateSQLWithOptions(false, targetSchema)
		w.WriteStatementWithComment("TRIGGER", diff.New.Name, diff.New.Schema, "", sql, targetSchema)
	}
}

// generateDropPoliciesSQL generates DROP POLICY statements
func (d *DDLDiff) generateDropPoliciesSQL(w *SQLWriter, policies []*ir.RLSPolicy, targetSchema string) {
	// Sort policies by name for consistent ordering
	sortedPolicies := make([]*ir.RLSPolicy, len(policies))
	copy(sortedPolicies, policies)
	sort.Slice(sortedPolicies, func(i, j int) bool {
		return sortedPolicies[i].Name < sortedPolicies[j].Name
	})

	for _, policy := range sortedPolicies {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s;", policy.Name, policy.Table)
		w.WriteStatementWithComment("POLICY", policy.Name, policy.Schema, "", sql, targetSchema)
	}
}

// generateCreatePoliciesSQL generates CREATE POLICY statements
func (d *DDLDiff) generateCreatePoliciesSQL(w *SQLWriter, policies []*ir.RLSPolicy, targetSchema string) {
	// Sort policies by name for consistent ordering
	sortedPolicies := make([]*ir.RLSPolicy, len(policies))
	copy(sortedPolicies, policies)
	sort.Slice(sortedPolicies, func(i, j int) bool {
		return sortedPolicies[i].Name < sortedPolicies[j].Name
	})

	for _, policy := range sortedPolicies {
		w.WriteDDLSeparator()
		sql := policy.GenerateSQLWithOptions(false, targetSchema)
		w.WriteStatementWithComment("POLICY", policy.Name, policy.Schema, "", sql, targetSchema)
	}
}

// generateModifyPoliciesSQL generates ALTER POLICY statements
func (d *DDLDiff) generateModifyPoliciesSQL(w *SQLWriter, diffs []*PolicyDiff, targetSchema string) {
	for _, diff := range diffs {
		w.WriteDDLSeparator()
		sql := diff.New.GenerateSQLWithOptions(false, targetSchema)
		w.WriteStatementWithComment("POLICY", diff.New.Name, diff.New.Schema, "", sql, targetSchema)
	}
}

// generateRLSChangesSQL generates RLS enable/disable statements
func (d *DDLDiff) generateRLSChangesSQL(w *SQLWriter, changes []*RLSChange, targetSchema string) {
	for _, change := range changes {
		w.WriteDDLSeparator()
		var sql string
		if change.Enabled {
			sql = fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY;", change.Table.Name)
		} else {
			sql = fmt.Sprintf("ALTER TABLE %s DISABLE ROW LEVEL SECURITY;", change.Table.Name)
		}
		w.WriteStatementWithComment("TABLE", change.Table.Name, change.Table.Schema, "", sql, targetSchema)
	}
}

// generateTableIndexes generates SQL for indexes belonging to a specific table
func (d *DDLDiff) generateTableIndexes(w *SQLWriter, table *ir.Table, targetSchema string) {
	// Get sorted index names for consistent output
	indexNames := make([]string, 0, len(table.Indexes))
	for indexName := range table.Indexes {
		indexNames = append(indexNames, indexName)
	}
	sort.Strings(indexNames)

	for _, indexName := range indexNames {
		index := table.Indexes[indexName]
		// Skip primary key indexes as they're handled with constraints
		if index.IsPrimary {
			continue
		}
		
		// Include all indexes for this table (for dump scenarios) or only added indexes (for diff scenarios)
		if d.isIndexInAddedList(index) {
			w.WriteDDLSeparator()
			sql := index.Definition
			if !strings.HasSuffix(sql, ";") {
				sql += ";"
			}
			w.WriteStatementWithComment("INDEX", indexName, table.Schema, "", sql, targetSchema)
		}
	}
}

// generateTableConstraints generates SQL for constraints belonging to a specific table
func (d *DDLDiff) generateTableConstraints(w *SQLWriter, table *ir.Table, targetSchema string) {
	// Get sorted constraint names for consistent output
	constraintNames := make([]string, 0, len(table.Constraints))
	for constraintName := range table.Constraints {
		constraintNames = append(constraintNames, constraintName)
	}
	sort.Strings(constraintNames)

	for _, constraintName := range constraintNames {
		constraint := table.Constraints[constraintName]
		// Skip PRIMARY KEY, UNIQUE, FOREIGN KEY, and CHECK constraints as they are now inline in CREATE TABLE
		if constraint.Type == ir.ConstraintTypePrimaryKey ||
			constraint.Type == ir.ConstraintTypeUnique ||
			constraint.Type == ir.ConstraintTypeForeignKey ||
			constraint.Type == ir.ConstraintTypeCheck {
			continue
		}
		
		// Only include constraints that would be in the added list
		w.WriteDDLSeparator()
		constraintSQL := constraint.GenerateSQLWithOptions(false, targetSchema)
		w.WriteStatementWithComment("CONSTRAINT", constraintName, table.Schema, "", constraintSQL, targetSchema)
	}
}

// generateTableTriggers generates SQL for triggers belonging to a specific table
func (d *DDLDiff) generateTableTriggers(w *SQLWriter, table *ir.Table, targetSchema string) {
	// Get sorted trigger names for consistent output
	triggerNames := make([]string, 0, len(table.Triggers))
	for triggerName := range table.Triggers {
		triggerNames = append(triggerNames, triggerName)
	}
	sort.Strings(triggerNames)

	for _, triggerName := range triggerNames {
		trigger := table.Triggers[triggerName]
		// Include all triggers for this table (for dump scenarios) or only added triggers (for diff scenarios)
		if d.isTriggerInAddedList(trigger) {
			w.WriteDDLSeparator()
			sql := trigger.GenerateSQLWithOptions(false, targetSchema)
			w.WriteStatementWithComment("TRIGGER", triggerName, table.Schema, "", sql, targetSchema)
		}
	}
}

// generateTableRLS generates RLS enablement and policies for a specific table
func (d *DDLDiff) generateTableRLS(w *SQLWriter, table *ir.Table, targetSchema string) {
	// Generate ALTER TABLE ... ENABLE ROW LEVEL SECURITY if needed
	if table.RLSEnabled {
		w.WriteDDLSeparator()
		var fullTableName string
		if table.Schema == targetSchema {
			fullTableName = table.Name
		} else {
			fullTableName = fmt.Sprintf("%s.%s", table.Schema, table.Name)
		}
		sql := fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY;", fullTableName)
		w.WriteStatementWithComment("TABLE", table.Name, table.Schema, "", sql, "")
	}

	// Generate policies for this table
	// Get sorted policy names for consistent output
	policyNames := make([]string, 0, len(table.Policies))
	for policyName := range table.Policies {
		policyNames = append(policyNames, policyName)
	}
	sort.Strings(policyNames)

	for _, policyName := range policyNames {
		policy := table.Policies[policyName]
		// Include all policies for this table (for dump scenarios) or only added policies (for diff scenarios)
		if d.isPolicyInAddedList(policy) {
			w.WriteDDLSeparator()
			sql := policy.GenerateSQLWithOptions(false, targetSchema)
			w.WriteStatementWithComment("POLICY", policyName, table.Schema, "", sql, targetSchema)
		}
	}
}

// Helper methods to check if objects are in the added lists
func (d *DDLDiff) isIndexInAddedList(index *ir.Index) bool {
	for _, addedIndex := range d.AddedIndexes {
		if addedIndex.Name == index.Name && addedIndex.Schema == index.Schema && addedIndex.Table == index.Table {
			return true
		}
	}
	return false
}

func (d *DDLDiff) isTriggerInAddedList(trigger *ir.Trigger) bool {
	for _, addedTrigger := range d.AddedTriggers {
		if addedTrigger.Name == trigger.Name && addedTrigger.Schema == trigger.Schema && addedTrigger.Table == trigger.Table {
			return true
		}
	}
	return false
}

func (d *DDLDiff) isPolicyInAddedList(policy *ir.RLSPolicy) bool {
	for _, addedPolicy := range d.AddedPolicies {
		if addedPolicy.Name == policy.Name && addedPolicy.Schema == policy.Schema && addedPolicy.Table == policy.Table {
			return true
		}
	}
	return false
}

func (d *DDLDiff) isRLSEnabledInChanges(table *ir.Table) bool {
	for _, change := range d.RLSChanges {
		if change.Table.Name == table.Name && change.Table.Schema == table.Schema && change.Enabled {
			return true
		}
	}
	return false
}