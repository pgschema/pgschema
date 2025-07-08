package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
)

// SQLGeneratorService handles unified SQL generation from DDL differences
type SQLGeneratorService struct {
	includeComments bool
	targetSchema    string
}

// NewSQLGeneratorService creates a new SQL generator service
func NewSQLGeneratorService(includeComments bool, targetSchema string) *SQLGeneratorService {
	return &SQLGeneratorService{
		includeComments: includeComments,
		targetSchema:    targetSchema,
	}
}

// GenerateMigrationSQL generates SQL from the DDL differences following the proper dependency order
func (s *SQLGeneratorService) GenerateMigrationSQL(diff *DDLDiff) string {
	w := NewSQLWriterWithComments(s.includeComments)

	// Write header comments
	if s.includeComments {
		s.writeHeader(w)
	}

	// Generate DDL in proper dependency order
	// First: Drop operations (in reverse dependency order)
	s.generateDropSQL(w, diff)
	
	// Then: Create operations (in dependency order)
	s.generateCreateSQL(w, diff)
	
	// Finally: Modify operations
	s.generateModifySQL(w, diff)

	return w.String()
}

// writeHeader writes the SQL header comments
func (s *SQLGeneratorService) writeHeader(w *SQLWriter) {
	w.WriteString("--\n")
	w.WriteString("-- PostgreSQL database migration\n")
	w.WriteString("--\n")
	w.WriteString("\n")
}

// generateDropSQL generates DROP statements in reverse dependency order
func (s *SQLGeneratorService) generateDropSQL(w *SQLWriter, diff *DDLDiff) {
	// Drop RLS policies first
	s.generateDropPoliciesSQL(w, diff.DroppedPolicies)
	
	// Drop triggers
	s.generateDropTriggersSQL(w, diff.DroppedTriggers)
	
	// Drop indexes
	s.generateDropIndexesSQL(w, diff.DroppedIndexes)
	
	// Drop functions
	s.generateDropFunctionsSQL(w, diff.DroppedFunctions)
	
	// Drop views
	s.generateDropViewsSQL(w, diff.DroppedViews)
	
	// Drop tables
	s.generateDropTablesSQL(w, diff.DroppedTables)
	
	// Drop types
	s.generateDropTypesSQL(w, diff.DroppedTypes)
	
	// Drop extensions
	s.generateDropExtensionsSQL(w, diff.DroppedExtensions)
	
	// Drop schemas
	s.generateDropSchemasSQL(w, diff.DroppedSchemas)
}

// generateCreateSQL generates CREATE statements in dependency order
func (s *SQLGeneratorService) generateCreateSQL(w *SQLWriter, diff *DDLDiff) {
	// Create schemas first
	s.generateCreateSchemasSQL(w, diff.AddedSchemas)
	
	// Create extensions
	s.generateCreateExtensionsSQL(w, diff.AddedExtensions)
	
	// Create types
	s.generateCreateTypesSQL(w, diff.AddedTypes)
	
	// Create tables
	s.generateCreateTablesSQL(w, diff.AddedTables)
	
	// Create views
	s.generateCreateViewsSQL(w, diff.AddedViews)
	
	// Create functions
	s.generateCreateFunctionsSQL(w, diff.AddedFunctions)
	
	// Create indexes
	s.generateCreateIndexesSQL(w, diff.AddedIndexes)
	
	// Create triggers
	s.generateCreateTriggersSQL(w, diff.AddedTriggers)
	
	// Create RLS policies
	s.generateCreatePoliciesSQL(w, diff.AddedPolicies)
}

// generateModifySQL generates ALTER statements
func (s *SQLGeneratorService) generateModifySQL(w *SQLWriter, diff *DDLDiff) {
	// Modify schemas
	s.generateModifySchemasSQL(w, diff.ModifiedSchemas)
	
	// Modify types
	s.generateModifyTypesSQL(w, diff.ModifiedTypes)
	
	// Modify tables
	s.generateModifyTablesSQL(w, diff.ModifiedTables)
	
	// Modify views
	s.generateModifyViewsSQL(w, diff.ModifiedViews)
	
	// Modify functions
	s.generateModifyFunctionsSQL(w, diff.ModifiedFunctions)
	
	// Modify triggers
	s.generateModifyTriggersSQL(w, diff.ModifiedTriggers)
	
	// Handle RLS enable/disable changes
	s.generateRLSChangesSQL(w, diff.RLSChanges)
	
	// Modify policies
	s.generateModifyPoliciesSQL(w, diff.ModifiedPolicies)
}

// generateDropSchemasSQL generates DROP SCHEMA statements
func (s *SQLGeneratorService) generateDropSchemasSQL(w *SQLWriter, schemas []*ir.Schema) {
	// Sort schemas by name for consistent ordering
	sortedSchemas := make([]*ir.Schema, len(schemas))
	copy(sortedSchemas, schemas)
	sort.Slice(sortedSchemas, func(i, j int) bool {
		return sortedSchemas[i].Name < sortedSchemas[j].Name
	})

	for _, schema := range sortedSchemas {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE;", schema.Name)
		w.WriteStatementWithComment("SCHEMA", schema.Name, "", "", sql, s.targetSchema)
	}
}

// generateCreateSchemasSQL generates CREATE SCHEMA statements
func (s *SQLGeneratorService) generateCreateSchemasSQL(w *SQLWriter, schemas []*ir.Schema) {
	// Sort schemas by name for consistent ordering
	sortedSchemas := make([]*ir.Schema, len(schemas))
	copy(sortedSchemas, schemas)
	sort.Slice(sortedSchemas, func(i, j int) bool {
		return sortedSchemas[i].Name < sortedSchemas[j].Name
	})

	for _, schema := range sortedSchemas {
		// Skip creating the target schema if we're doing a schema-specific dump
		if schema.Name == s.targetSchema {
			continue
		}
		if sql := schema.GenerateSQL(); sql != "" {
			w.WriteDDLSeparator()
			w.WriteStatementWithComment("SCHEMA", schema.Name, "", "", sql, s.targetSchema)
		}
	}
}

// generateModifySchemasSQL generates ALTER SCHEMA statements
func (s *SQLGeneratorService) generateModifySchemasSQL(w *SQLWriter, diffs []*SchemaDiff) {
	for _, diff := range diffs {
		if diff.Old.Owner != diff.New.Owner {
			w.WriteDDLSeparator()
			sql := fmt.Sprintf("ALTER SCHEMA %s OWNER TO %s;", diff.New.Name, diff.New.Owner)
			w.WriteStatementWithComment("SCHEMA", diff.New.Name, "", "", sql, s.targetSchema)
		}
	}
}

// generateDropExtensionsSQL generates DROP EXTENSION statements
func (s *SQLGeneratorService) generateDropExtensionsSQL(w *SQLWriter, extensions []*ir.Extension) {
	// Sort extensions by name for consistent ordering
	sortedExtensions := make([]*ir.Extension, len(extensions))
	copy(sortedExtensions, extensions)
	sort.Slice(sortedExtensions, func(i, j int) bool {
		return sortedExtensions[i].Name < sortedExtensions[j].Name
	})

	for _, ext := range sortedExtensions {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP EXTENSION IF EXISTS %s;", ext.Name)
		w.WriteStatementWithComment("EXTENSION", ext.Name, ext.Schema, "", sql, s.targetSchema)
	}
}

// generateCreateExtensionsSQL generates CREATE EXTENSION statements
func (s *SQLGeneratorService) generateCreateExtensionsSQL(w *SQLWriter, extensions []*ir.Extension) {
	// Sort extensions by name for consistent ordering
	sortedExtensions := make([]*ir.Extension, len(extensions))
	copy(sortedExtensions, extensions)
	sort.Slice(sortedExtensions, func(i, j int) bool {
		return sortedExtensions[i].Name < sortedExtensions[j].Name
	})

	for _, ext := range sortedExtensions {
		w.WriteDDLSeparator()
		sql := ext.GenerateSQL()
		w.WriteStatementWithComment("EXTENSION", ext.Name, ext.Schema, "", sql, s.targetSchema)
	}
}

// generateDropTypesSQL generates DROP TYPE statements
func (s *SQLGeneratorService) generateDropTypesSQL(w *SQLWriter, types []*ir.Type) {
	// Sort types by name for consistent ordering
	sortedTypes := make([]*ir.Type, len(types))
	copy(sortedTypes, types)
	sort.Slice(sortedTypes, func(i, j int) bool {
		return sortedTypes[i].Name < sortedTypes[j].Name
	})

	for _, typeObj := range sortedTypes {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP TYPE IF EXISTS %s CASCADE;", typeObj.Name)
		w.WriteStatementWithComment("TYPE", typeObj.Name, typeObj.Schema, "", sql, s.targetSchema)
	}
}

// generateCreateTypesSQL generates CREATE TYPE statements
func (s *SQLGeneratorService) generateCreateTypesSQL(w *SQLWriter, types []*ir.Type) {
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
		sql := typeObj.GenerateSQLWithOptions(false, s.targetSchema)
		
		// Use correct object type for comment
		var objectType string
		switch typeObj.Kind {
		case ir.TypeKindDomain:
			objectType = "DOMAIN"
		default:
			objectType = "TYPE"
		}
		
		w.WriteStatementWithComment(objectType, typeObj.Name, typeObj.Schema, "", sql, s.targetSchema)
	}
}

// generateModifyTypesSQL generates ALTER TYPE statements
func (s *SQLGeneratorService) generateModifyTypesSQL(w *SQLWriter, diffs []*TypeDiff) {
	for _, diff := range diffs {
		// Only ENUM types can be modified by adding values
		if diff.Old.Kind == ir.TypeKindEnum && diff.New.Kind == ir.TypeKindEnum {
			// Generate ALTER TYPE ... ADD VALUE statements for new enum values
			// This is a simplified implementation - in reality you'd need to diff the enum values
			w.WriteDDLSeparator()
			sql := fmt.Sprintf("-- ALTER TYPE %s ADD VALUE statements would go here", diff.New.Name)
			w.WriteStatementWithComment("TYPE", diff.New.Name, diff.New.Schema, "", sql, s.targetSchema)
		}
	}
}

// generateDropTablesSQL generates DROP TABLE statements
func (s *SQLGeneratorService) generateDropTablesSQL(w *SQLWriter, tables []*ir.Table) {
	// Sort tables by name for consistent ordering
	sortedTables := make([]*ir.Table, len(tables))
	copy(sortedTables, tables)
	sort.Slice(sortedTables, func(i, j int) bool {
		return sortedTables[i].Name < sortedTables[j].Name
	})

	for _, table := range sortedTables {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", table.Name)
		w.WriteStatementWithComment("TABLE", table.Name, table.Schema, "", sql, s.targetSchema)
	}
}

// generateCreateTablesSQL generates CREATE TABLE statements
func (s *SQLGeneratorService) generateCreateTablesSQL(w *SQLWriter, tables []*ir.Table) {
	// Sort tables by name for consistent ordering
	sortedTables := make([]*ir.Table, len(tables))
	copy(sortedTables, tables)
	sort.Slice(sortedTables, func(i, j int) bool {
		return sortedTables[i].Name < sortedTables[j].Name
	})

	for _, table := range sortedTables {
		w.WriteDDLSeparator()
		sql := table.GenerateSQLWithOptions(false, s.targetSchema)
		w.WriteStatementWithComment("TABLE", table.Name, table.Schema, "", sql, s.targetSchema)
	}
}

// generateModifyTablesSQL generates ALTER TABLE statements
func (s *SQLGeneratorService) generateModifyTablesSQL(w *SQLWriter, diffs []*TableDiff) {
	for _, diff := range diffs {
		statements := diff.GenerateMigrationSQL()
		for _, stmt := range statements {
			w.WriteDDLSeparator()
			w.WriteStatementWithComment("TABLE", diff.Table.Name, diff.Table.Schema, "", stmt, s.targetSchema)
		}
	}
}

// generateDropViewsSQL generates DROP VIEW statements
func (s *SQLGeneratorService) generateDropViewsSQL(w *SQLWriter, views []*ir.View) {
	// Sort views by name for consistent ordering
	sortedViews := make([]*ir.View, len(views))
	copy(sortedViews, views)
	sort.Slice(sortedViews, func(i, j int) bool {
		return sortedViews[i].Name < sortedViews[j].Name
	})

	for _, view := range sortedViews {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP VIEW IF EXISTS %s CASCADE;", view.Name)
		w.WriteStatementWithComment("VIEW", view.Name, view.Schema, "", sql, s.targetSchema)
	}
}

// generateCreateViewsSQL generates CREATE VIEW statements
func (s *SQLGeneratorService) generateCreateViewsSQL(w *SQLWriter, views []*ir.View) {
	// Sort views by name for consistent ordering
	sortedViews := make([]*ir.View, len(views))
	copy(sortedViews, views)
	sort.Slice(sortedViews, func(i, j int) bool {
		return sortedViews[i].Name < sortedViews[j].Name
	})

	for _, view := range sortedViews {
		w.WriteDDLSeparator()
		sql := view.GenerateSQLWithOptions(false, s.targetSchema)
		w.WriteStatementWithComment("VIEW", view.Name, view.Schema, "", sql, s.targetSchema)
	}
}

// generateModifyViewsSQL generates ALTER VIEW statements
func (s *SQLGeneratorService) generateModifyViewsSQL(w *SQLWriter, diffs []*ViewDiff) {
	for _, diff := range diffs {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("CREATE OR REPLACE VIEW %s AS %s;", diff.New.Name, diff.New.Definition)
		w.WriteStatementWithComment("VIEW", diff.New.Name, diff.New.Schema, "", sql, s.targetSchema)
	}
}

// generateDropFunctionsSQL generates DROP FUNCTION statements
func (s *SQLGeneratorService) generateDropFunctionsSQL(w *SQLWriter, functions []*ir.Function) {
	// Sort functions by name for consistent ordering
	sortedFunctions := make([]*ir.Function, len(functions))
	copy(sortedFunctions, functions)
	sort.Slice(sortedFunctions, func(i, j int) bool {
		return sortedFunctions[i].Name < sortedFunctions[j].Name
	})

	for _, function := range sortedFunctions {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP FUNCTION IF EXISTS %s(%s) CASCADE;", function.Name, function.Arguments)
		w.WriteStatementWithComment("FUNCTION", function.Name, function.Schema, "", sql, s.targetSchema)
	}
}

// generateCreateFunctionsSQL generates CREATE FUNCTION statements
func (s *SQLGeneratorService) generateCreateFunctionsSQL(w *SQLWriter, functions []*ir.Function) {
	// Sort functions by name for consistent ordering
	sortedFunctions := make([]*ir.Function, len(functions))
	copy(sortedFunctions, functions)
	sort.Slice(sortedFunctions, func(i, j int) bool {
		return sortedFunctions[i].Name < sortedFunctions[j].Name
	})

	for _, function := range sortedFunctions {
		w.WriteDDLSeparator()
		sql := function.GenerateSQLWithOptions(false, s.targetSchema)
		w.WriteStatementWithComment("FUNCTION", function.Name, function.Schema, "", sql, s.targetSchema)
	}
}

// generateModifyFunctionsSQL generates ALTER FUNCTION statements
func (s *SQLGeneratorService) generateModifyFunctionsSQL(w *SQLWriter, diffs []*FunctionDiff) {
	for _, diff := range diffs {
		w.WriteDDLSeparator()
		sql := diff.New.GenerateSQLWithOptions(false, s.targetSchema)
		w.WriteStatementWithComment("FUNCTION", diff.New.Name, diff.New.Schema, "", sql, s.targetSchema)
	}
}

// generateDropIndexesSQL generates DROP INDEX statements
func (s *SQLGeneratorService) generateDropIndexesSQL(w *SQLWriter, indexes []*ir.Index) {
	// Sort indexes by name for consistent ordering
	sortedIndexes := make([]*ir.Index, len(indexes))
	copy(sortedIndexes, indexes)
	sort.Slice(sortedIndexes, func(i, j int) bool {
		return sortedIndexes[i].Name < sortedIndexes[j].Name
	})

	for _, index := range sortedIndexes {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP INDEX IF EXISTS %s;", index.Name)
		w.WriteStatementWithComment("INDEX", index.Name, index.Schema, "", sql, s.targetSchema)
	}
}

// generateCreateIndexesSQL generates CREATE INDEX statements
func (s *SQLGeneratorService) generateCreateIndexesSQL(w *SQLWriter, indexes []*ir.Index) {
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
		w.WriteStatementWithComment("INDEX", index.Name, index.Schema, "", sql, s.targetSchema)
	}
}

// generateDropTriggersSQL generates DROP TRIGGER statements
func (s *SQLGeneratorService) generateDropTriggersSQL(w *SQLWriter, triggers []*ir.Trigger) {
	// Sort triggers by name for consistent ordering
	sortedTriggers := make([]*ir.Trigger, len(triggers))
	copy(sortedTriggers, triggers)
	sort.Slice(sortedTriggers, func(i, j int) bool {
		return sortedTriggers[i].Name < sortedTriggers[j].Name
	})

	for _, trigger := range sortedTriggers {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s;", trigger.Name, trigger.Table)
		w.WriteStatementWithComment("TRIGGER", trigger.Name, trigger.Schema, "", sql, s.targetSchema)
	}
}

// generateCreateTriggersSQL generates CREATE TRIGGER statements
func (s *SQLGeneratorService) generateCreateTriggersSQL(w *SQLWriter, triggers []*ir.Trigger) {
	// Sort triggers by name for consistent ordering
	sortedTriggers := make([]*ir.Trigger, len(triggers))
	copy(sortedTriggers, triggers)
	sort.Slice(sortedTriggers, func(i, j int) bool {
		return sortedTriggers[i].Name < sortedTriggers[j].Name
	})

	for _, trigger := range sortedTriggers {
		w.WriteDDLSeparator()
		sql := trigger.GenerateSQLWithOptions(false, s.targetSchema)
		w.WriteStatementWithComment("TRIGGER", trigger.Name, trigger.Schema, "", sql, s.targetSchema)
	}
}

// generateModifyTriggersSQL generates ALTER TRIGGER statements
func (s *SQLGeneratorService) generateModifyTriggersSQL(w *SQLWriter, diffs []*TriggerDiff) {
	for _, diff := range diffs {
		w.WriteDDLSeparator()
		sql := diff.New.GenerateSQLWithOptions(false, s.targetSchema)
		w.WriteStatementWithComment("TRIGGER", diff.New.Name, diff.New.Schema, "", sql, s.targetSchema)
	}
}

// generateDropPoliciesSQL generates DROP POLICY statements
func (s *SQLGeneratorService) generateDropPoliciesSQL(w *SQLWriter, policies []*ir.RLSPolicy) {
	// Sort policies by name for consistent ordering
	sortedPolicies := make([]*ir.RLSPolicy, len(policies))
	copy(sortedPolicies, policies)
	sort.Slice(sortedPolicies, func(i, j int) bool {
		return sortedPolicies[i].Name < sortedPolicies[j].Name
	})

	for _, policy := range sortedPolicies {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s;", policy.Name, policy.Table)
		w.WriteStatementWithComment("POLICY", policy.Name, policy.Schema, "", sql, s.targetSchema)
	}
}

// generateCreatePoliciesSQL generates CREATE POLICY statements
func (s *SQLGeneratorService) generateCreatePoliciesSQL(w *SQLWriter, policies []*ir.RLSPolicy) {
	// Sort policies by name for consistent ordering
	sortedPolicies := make([]*ir.RLSPolicy, len(policies))
	copy(sortedPolicies, policies)
	sort.Slice(sortedPolicies, func(i, j int) bool {
		return sortedPolicies[i].Name < sortedPolicies[j].Name
	})

	for _, policy := range sortedPolicies {
		w.WriteDDLSeparator()
		sql := policy.GenerateSQLWithOptions(false, s.targetSchema)
		w.WriteStatementWithComment("POLICY", policy.Name, policy.Schema, "", sql, s.targetSchema)
	}
}

// generateModifyPoliciesSQL generates ALTER POLICY statements
func (s *SQLGeneratorService) generateModifyPoliciesSQL(w *SQLWriter, diffs []*PolicyDiff) {
	for _, diff := range diffs {
		w.WriteDDLSeparator()
		sql := diff.New.GenerateSQLWithOptions(false, s.targetSchema)
		w.WriteStatementWithComment("POLICY", diff.New.Name, diff.New.Schema, "", sql, s.targetSchema)
	}
}

// generateRLSChangesSQL generates RLS enable/disable statements
func (s *SQLGeneratorService) generateRLSChangesSQL(w *SQLWriter, changes []*RLSChange) {
	for _, change := range changes {
		w.WriteDDLSeparator()
		var sql string
		if change.Enabled {
			sql = fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY;", change.Table.Name)
		} else {
			sql = fmt.Sprintf("ALTER TABLE %s DISABLE ROW LEVEL SECURITY;", change.Table.Name)
		}
		w.WriteStatementWithComment("TABLE", change.Table.Name, change.Table.Schema, "", sql, s.targetSchema)
	}
}