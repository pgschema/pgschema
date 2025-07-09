package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
	"github.com/pgschema/pgschema/internal/utils"
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
		if sql := d.generateSchemaSQL(schema); sql != "" {
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
		sql := d.generateExtensionSQL(ext)
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
		sql := d.generateTypeSQL(typeObj, targetSchema)
		
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
			sql := d.generateTableSQL(table, targetSchema)
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
			sql := d.generateViewSQL(view, targetSchema)
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
		sql := d.generateFunctionSQL(function, targetSchema)
		w.WriteStatementWithComment("FUNCTION", function.Name, function.Schema, "", sql, targetSchema)
	}
}

// generateModifyFunctionsSQL generates ALTER FUNCTION statements
func (d *DDLDiff) generateModifyFunctionsSQL(w *SQLWriter, diffs []*FunctionDiff, targetSchema string) {
	for _, diff := range diffs {
		w.WriteDDLSeparator()
		sql := d.generateFunctionSQL(diff.New, targetSchema)
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
		sql := d.generateIndexSQL(index, targetSchema)
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
		sql := d.generateTriggerSQL(trigger, targetSchema)
		w.WriteStatementWithComment("TRIGGER", trigger.Name, trigger.Schema, "", sql, targetSchema)
	}
}

// generateModifyTriggersSQL generates ALTER TRIGGER statements
func (d *DDLDiff) generateModifyTriggersSQL(w *SQLWriter, diffs []*TriggerDiff, targetSchema string) {
	for _, diff := range diffs {
		w.WriteDDLSeparator()
		sql := d.generateTriggerSQL(diff.New, targetSchema)
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
		sql := d.generatePolicySQL(policy, targetSchema)
		w.WriteStatementWithComment("POLICY", policy.Name, policy.Schema, "", sql, targetSchema)
	}
}

// generateModifyPoliciesSQL generates ALTER POLICY statements
func (d *DDLDiff) generateModifyPoliciesSQL(w *SQLWriter, diffs []*PolicyDiff, targetSchema string) {
	for _, diff := range diffs {
		w.WriteDDLSeparator()
		sql := d.generatePolicySQL(diff.New, targetSchema)
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
			sql := d.generateIndexSQL(index, targetSchema)
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
		constraintSQL := d.generateConstraintSQL(constraint, targetSchema)
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
			sql := d.generateTriggerSQL(trigger, targetSchema)
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
			sql := d.generatePolicySQL(policy, targetSchema)
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

// SQL generation functions for individual IR objects
// These replace the GenerateSQL methods that were previously in the IR module

// generateSchemaSQL generates CREATE SCHEMA statement
func (d *DDLDiff) generateSchemaSQL(schema *ir.Schema) string {
	if schema.Name == "public" {
		return "" // Skip public schema
	}
	return fmt.Sprintf("CREATE SCHEMA %s;", schema.Name)
}

// generateExtensionSQL generates CREATE EXTENSION statement
func (d *DDLDiff) generateExtensionSQL(ext *ir.Extension) string {
	if ext.Schema != "" {
		return fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s WITH SCHEMA %s;", ext.Name, ext.Schema)
	} else {
		return fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s;", ext.Name)
	}
}

// generateTypeSQL generates CREATE TYPE statement
func (d *DDLDiff) generateTypeSQL(typeObj *ir.Type, targetSchema string) string {
	// Only include type name without schema if it's in the target schema
	typeName := utils.QualifyEntityName(typeObj.Schema, typeObj.Name, targetSchema)

	switch typeObj.Kind {
	case ir.TypeKindEnum:
		var values []string
		for _, value := range typeObj.EnumValues {
			values = append(values, fmt.Sprintf("'%s'", value))
		}
		return fmt.Sprintf("CREATE TYPE %s AS ENUM (%s);", typeName, strings.Join(values, ", "))
	case ir.TypeKindComposite:
		var attributes []string
		for _, attr := range typeObj.Columns {
			attributes = append(attributes, fmt.Sprintf("%s %s", attr.Name, attr.DataType))
		}
		return fmt.Sprintf("CREATE TYPE %s AS (%s);", typeName, strings.Join(attributes, ", "))
	case ir.TypeKindDomain:
		stmt := fmt.Sprintf("CREATE DOMAIN %s AS %s", typeName, typeObj.BaseType)
		if typeObj.Default != "" {
			stmt += fmt.Sprintf(" DEFAULT %s", typeObj.Default)
		}
		if typeObj.NotNull {
			stmt += " NOT NULL"
		}
		// Add domain constraints (CHECK constraints)
		for _, constraint := range typeObj.Constraints {
			if constraint.Name != "" {
				stmt += fmt.Sprintf(" CONSTRAINT %s %s", constraint.Name, constraint.Definition)
			} else {
				stmt += fmt.Sprintf(" %s", constraint.Definition)
			}
		}
		return stmt + ";"
	default:
		return fmt.Sprintf("CREATE TYPE %s;", typeName)
	}
}

// generateTableSQL generates CREATE TABLE statement
func (d *DDLDiff) generateTableSQL(table *ir.Table, targetSchema string) string {
	// Only include table name without schema if it's in the target schema
	tableName := utils.QualifyEntityName(table.Schema, table.Name, targetSchema)

	var parts []string
	parts = append(parts, fmt.Sprintf("CREATE TABLE %s (", tableName))

	// Add columns
	var columnParts []string
	for _, column := range table.Columns {
		// Build column definition with SERIAL detection
		var builder strings.Builder
		writeColumnDefinitionToBuilder(&builder, table, column, targetSchema)
		columnParts = append(columnParts, fmt.Sprintf("    %s", builder.String()))
	}

	// Add constraints inline in the correct order (PRIMARY KEY, UNIQUE, FOREIGN KEY)
	inlineConstraints := getInlineConstraintsForTable(table)
	for _, constraint := range inlineConstraints {
		constraintDef := d.generateConstraintSQL(constraint, targetSchema)
		if constraintDef != "" {
			columnParts = append(columnParts, fmt.Sprintf("    %s", constraintDef))
		}
	}

	parts = append(parts, strings.Join(columnParts, ",\n"))
	
	// Add partition clause for partitioned tables
	if table.IsPartitioned && table.PartitionStrategy != "" && table.PartitionKey != "" {
		parts = append(parts, fmt.Sprintf(")\nPARTITION BY %s (%s);", table.PartitionStrategy, table.PartitionKey))
	} else {
		parts = append(parts, ");")
	}

	return strings.Join(parts, "\n")
}

// generateViewSQL generates CREATE VIEW statement
func (d *DDLDiff) generateViewSQL(view *ir.View, targetSchema string) string {
	// Only include view name without schema if it's in the target schema
	viewName := utils.QualifyEntityName(view.Schema, view.Name, targetSchema)
	return fmt.Sprintf("CREATE VIEW %s AS\n%s", viewName, view.Definition)
}

// generateFunctionSQL generates CREATE FUNCTION statement
func (d *DDLDiff) generateFunctionSQL(function *ir.Function, targetSchema string) string {
	// Only include function name without schema if it's in the target schema
	functionName := utils.QualifyEntityName(function.Schema, function.Name, targetSchema)

	stmt := fmt.Sprintf("CREATE OR REPLACE FUNCTION %s(%s) RETURNS %s",
		functionName, function.Arguments, function.ReturnType)

	if function.Language != "" {
		stmt += fmt.Sprintf(" LANGUAGE %s", function.Language)
	}

	if function.Volatility != "" {
		stmt += fmt.Sprintf(" %s", function.Volatility)
	}

	if function.IsSecurityDefiner {
		stmt += " SECURITY DEFINER"
	}

	// Add the function body with proper dollar quoting
	if function.Definition != "" {
		tag := generateDollarQuoteTag(function.Definition)
		stmt += fmt.Sprintf("\nAS %s%s%s;", tag, function.Definition, tag)
	} else {
		stmt += "\nAS $$$$;"
	}

	return stmt
}

// generateSequenceSQL generates CREATE SEQUENCE statement
func (d *DDLDiff) generateSequenceSQL(sequence *ir.Sequence, targetSchema string) string {
	// Only include sequence name without schema if it's in the target schema
	sequenceName := utils.QualifyEntityName(sequence.Schema, sequence.Name, targetSchema)

	stmt := fmt.Sprintf("CREATE SEQUENCE %s", sequenceName)

	if sequence.DataType != "" && sequence.DataType != "bigint" {
		stmt += fmt.Sprintf(" AS %s", sequence.DataType)
	}

	if sequence.StartValue != 1 {
		stmt += fmt.Sprintf(" START %d", sequence.StartValue)
	}

	if sequence.Increment != 1 {
		stmt += fmt.Sprintf(" INCREMENT %d", sequence.Increment)
	}

	if sequence.MinValue != nil && *sequence.MinValue != 1 {
		stmt += fmt.Sprintf(" MINVALUE %d", *sequence.MinValue)
	}

	if sequence.MaxValue != nil && *sequence.MaxValue != 9223372036854775807 {
		stmt += fmt.Sprintf(" MAXVALUE %d", *sequence.MaxValue)
	}

	if sequence.CycleOption {
		stmt += " CYCLE"
	}

	return stmt + ";"
}

// generateTriggerSQL generates CREATE TRIGGER statement
func (d *DDLDiff) generateTriggerSQL(trigger *ir.Trigger, targetSchema string) string {
	// Build event list in standard order: INSERT, UPDATE, DELETE
	var events []string
	eventOrder := []ir.TriggerEvent{ir.TriggerEventInsert, ir.TriggerEventUpdate, ir.TriggerEventDelete}
	for _, orderEvent := range eventOrder {
		for _, triggerEvent := range trigger.Events {
			if triggerEvent == orderEvent {
				events = append(events, string(triggerEvent))
				break
			}
		}
	}
	eventList := strings.Join(events, " OR ")

	// Only include table name without schema if it's in the target schema
	tableName := utils.QualifyEntityName(trigger.Schema, trigger.Table, targetSchema)

	// Function field should contain the complete function call including parameters
	return fmt.Sprintf("CREATE TRIGGER %s %s %s ON %s FOR EACH %s EXECUTE FUNCTION %s;",
		trigger.Name, trigger.Timing, eventList, tableName, trigger.Level, trigger.Function)
}

// generatePolicySQL generates CREATE POLICY statement
func (d *DDLDiff) generatePolicySQL(policy *ir.RLSPolicy, targetSchema string) string {
	// Only include table name without schema if it's in the target schema
	tableName := utils.QualifyEntityName(policy.Schema, policy.Table, targetSchema)

	policyStmt := fmt.Sprintf("CREATE POLICY %s ON %s", policy.Name, tableName)

	// Add command type if specified
	if policy.Command != ir.PolicyCommandAll {
		policyStmt += fmt.Sprintf(" FOR %s", policy.Command)
	}

	// Add roles if specified
	if len(policy.Roles) > 0 {
		policyStmt += " TO "
		for i, role := range policy.Roles {
			if i > 0 {
				policyStmt += ", "
			}
			policyStmt += role
		}
	}

	// Add USING clause if present
	if policy.Using != "" {
		policyStmt += fmt.Sprintf(" USING (%s)", policy.Using)
	}

	// Add WITH CHECK clause if present
	if policy.WithCheck != "" {
		policyStmt += fmt.Sprintf(" WITH CHECK (%s)", policy.WithCheck)
	}

	return policyStmt + ";"
}

// getSortedConstraintNames returns constraint names sorted alphabetically
func getSortedConstraintNames(constraints map[string]*ir.Constraint) []string {
	return utils.SortedKeys(constraints)
}

// getInlineConstraintsForTable returns constraints in the correct order: PRIMARY KEY, UNIQUE, FOREIGN KEY
func getInlineConstraintsForTable(table *ir.Table) []*ir.Constraint {
	var inlineConstraints []*ir.Constraint

	// Get constraint names sorted for consistent output
	constraintNames := getSortedConstraintNames(table.Constraints)

	// Separate constraints by type for proper ordering
	var primaryKeys []*ir.Constraint
	var uniques []*ir.Constraint
	var foreignKeys []*ir.Constraint
	var checkConstraints []*ir.Constraint

	for _, constraintName := range constraintNames {
		constraint := table.Constraints[constraintName]

		// Categorize constraints by type
		switch constraint.Type {
		case ir.ConstraintTypePrimaryKey:
			primaryKeys = append(primaryKeys, constraint)
		case ir.ConstraintTypeUnique:
			uniques = append(uniques, constraint)
		case ir.ConstraintTypeForeignKey:
			foreignKeys = append(foreignKeys, constraint)
		case ir.ConstraintTypeCheck:
			// Only include table-level CHECK constraints (not column-level ones)
			// Column-level CHECK constraints are handled inline with the column definition
			if len(constraint.Columns) != 1 {
				checkConstraints = append(checkConstraints, constraint)
			}
		}
	}

	// Add constraints in order: PRIMARY KEY, UNIQUE, FOREIGN KEY, CHECK
	inlineConstraints = append(inlineConstraints, primaryKeys...)
	inlineConstraints = append(inlineConstraints, uniques...)
	inlineConstraints = append(inlineConstraints, foreignKeys...)
	inlineConstraints = append(inlineConstraints, checkConstraints...)

	return inlineConstraints
}

// generateConstraintSQL generates constraint definition for inline table constraints
func (d *DDLDiff) generateConstraintSQL(constraint *ir.Constraint, targetSchema string) string {
	// Helper function to get column names from ConstraintColumn array
	getColumnNames := func(columns []*ir.ConstraintColumn) []string {
		var names []string
		for _, col := range columns {
			names = append(names, col.Name)
		}
		return names
	}

	switch constraint.Type {
	case ir.ConstraintTypePrimaryKey:
		return fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(getColumnNames(constraint.Columns), ", "))
	case ir.ConstraintTypeUnique:
		return fmt.Sprintf("UNIQUE (%s)", strings.Join(getColumnNames(constraint.Columns), ", "))
	case ir.ConstraintTypeForeignKey:
		stmt := fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s (%s)",
			strings.Join(getColumnNames(constraint.Columns), ", "),
			constraint.ReferencedTable, strings.Join(getColumnNames(constraint.ReferencedColumns), ", "))
		// Only add ON DELETE/UPDATE if they are not the default "NO ACTION"
		if constraint.DeleteRule != "" && constraint.DeleteRule != "NO ACTION" {
			stmt += fmt.Sprintf(" ON DELETE %s", constraint.DeleteRule)
		}
		if constraint.UpdateRule != "" && constraint.UpdateRule != "NO ACTION" {
			stmt += fmt.Sprintf(" ON UPDATE %s", constraint.UpdateRule)
		}
		return stmt
	case ir.ConstraintTypeCheck:
		return constraint.CheckClause
	default:
		return ""
	}
}

// generateIndexSQL generates CREATE INDEX statement
func (d *DDLDiff) generateIndexSQL(index *ir.Index, targetSchema string) string {
	var stmt string
	if index.Schema != targetSchema {
		// Use the definition as-is
		stmt = index.Definition
	} else {
		// Remove schema qualifiers from the definition for schema-agnostic output
		definition := index.Definition
		schemaPrefix := index.Schema + "."
		// Remove schema qualifiers that match the target schema
		definition = strings.ReplaceAll(definition, schemaPrefix, "")
		stmt = definition
	}

	// Remove "USING btree" since btree is the default index method
	if index.Method == "btree" {
		stmt = strings.ReplaceAll(stmt, " USING btree", "")
	}

	if !strings.HasSuffix(stmt, ";") {
		stmt += ";"
	}

	return stmt
}