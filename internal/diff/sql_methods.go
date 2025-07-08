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
	// Sort tables by name for consistent ordering
	sortedTables := make([]*ir.Table, len(tables))
	copy(sortedTables, tables)
	sort.Slice(sortedTables, func(i, j int) bool {
		return sortedTables[i].Name < sortedTables[j].Name
	})

	for _, table := range sortedTables {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", table.Name)
		w.WriteStatementWithComment("TABLE", table.Name, table.Schema, "", sql, targetSchema)
	}
}

// generateCreateTablesSQL generates CREATE TABLE statements
func (d *DDLDiff) generateCreateTablesSQL(w *SQLWriter, tables []*ir.Table, targetSchema string) {
	// Sort tables by name for consistent ordering
	sortedTables := make([]*ir.Table, len(tables))
	copy(sortedTables, tables)
	sort.Slice(sortedTables, func(i, j int) bool {
		return sortedTables[i].Name < sortedTables[j].Name
	})

	for _, table := range sortedTables {
		w.WriteDDLSeparator()
		sql := table.GenerateSQLWithOptions(false, targetSchema)
		w.WriteStatementWithComment("TABLE", table.Name, table.Schema, "", sql, targetSchema)
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
	// Sort views by name for consistent ordering
	sortedViews := make([]*ir.View, len(views))
	copy(sortedViews, views)
	sort.Slice(sortedViews, func(i, j int) bool {
		return sortedViews[i].Name < sortedViews[j].Name
	})

	for _, view := range sortedViews {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP VIEW IF EXISTS %s CASCADE;", view.Name)
		w.WriteStatementWithComment("VIEW", view.Name, view.Schema, "", sql, targetSchema)
	}
}

// generateCreateViewsSQL generates CREATE VIEW statements
func (d *DDLDiff) generateCreateViewsSQL(w *SQLWriter, views []*ir.View, targetSchema string) {
	// Sort views by name for consistent ordering
	sortedViews := make([]*ir.View, len(views))
	copy(sortedViews, views)
	sort.Slice(sortedViews, func(i, j int) bool {
		return sortedViews[i].Name < sortedViews[j].Name
	})

	for _, view := range sortedViews {
		w.WriteDDLSeparator()
		sql := view.GenerateSQLWithOptions(false, targetSchema)
		w.WriteStatementWithComment("VIEW", view.Name, view.Schema, "", sql, targetSchema)
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