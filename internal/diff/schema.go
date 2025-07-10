package diff

import (
	"fmt"
	"sort"

	"github.com/pgschema/pgschema/internal/ir"
)

// GenerateDropSchemaSQL generates SQL for dropping schemas
func GenerateDropSchemaSQL(schemas []*ir.Schema) []string {
	var statements []string
	for _, schema := range schemas {
		statements = append(statements, fmt.Sprintf("DROP SCHEMA %s;", schema.Name))
	}
	return statements
}

// GenerateCreateSchemaSQL generates SQL for creating schemas
func GenerateCreateSchemaSQL(schemas []*ir.Schema) []string {
	var statements []string
	
	// Sort schemas by: 1) schemas without owner first, 2) then by name alphabetically
	sortedSchemas := make([]*ir.Schema, len(schemas))
	copy(sortedSchemas, schemas)
	sort.Slice(sortedSchemas, func(i, j int) bool {
		schemaI := sortedSchemas[i]
		schemaJ := sortedSchemas[j]

		// If one has owner and other doesn't, prioritize the one without owner
		if (schemaI.Owner == "") != (schemaJ.Owner == "") {
			return schemaI.Owner == ""
		}

		// If both have same owner status, sort by name
		return schemaI.Name < schemaJ.Name
	})
	
	for _, schema := range sortedSchemas {
		if schema.Owner != "" {
			statements = append(statements, fmt.Sprintf("CREATE SCHEMA %s AUTHORIZATION %s;", schema.Name, schema.Owner))
		} else {
			statements = append(statements, fmt.Sprintf("CREATE SCHEMA %s;", schema.Name))
		}
	}
	
	return statements
}

// GenerateAlterSchemaSQL generates SQL for modifying schemas
func GenerateAlterSchemaSQL(schemaDiffs []*SchemaDiff) []string {
	var statements []string
	
	// Sort schema changes by name for consistent ordering
	sortedSchemaDiffs := make([]*SchemaDiff, len(schemaDiffs))
	copy(sortedSchemaDiffs, schemaDiffs)
	sort.Slice(sortedSchemaDiffs, func(i, j int) bool {
		return sortedSchemaDiffs[i].New.Name < sortedSchemaDiffs[j].New.Name
	})
	
	for _, schemaDiff := range sortedSchemaDiffs {
		if schemaDiff.Old.Owner != schemaDiff.New.Owner {
			statements = append(statements, fmt.Sprintf("ALTER SCHEMA %s OWNER TO %s;", schemaDiff.New.Name, schemaDiff.New.Owner))
		}
	}
	
	return statements
}

// generateDropSchemasSQL generates DROP SCHEMA statements
func generateDropSchemasSQL(w *SQLWriter, schemas []*ir.Schema, targetSchema string) {
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
func generateCreateSchemasSQL(w *SQLWriter, schemas []*ir.Schema, targetSchema string) {
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
		if sql := generateSchemaSQL(schema); sql != "" {
			w.WriteDDLSeparator()
			w.WriteStatementWithComment("SCHEMA", schema.Name, "", "", sql, targetSchema)
		}
	}
}

// generateModifySchemasSQL generates ALTER SCHEMA statements
func generateModifySchemasSQL(w *SQLWriter, diffs []*SchemaDiff, targetSchema string) {
	for _, diff := range diffs {
		if diff.Old.Owner != diff.New.Owner {
			w.WriteDDLSeparator()
			sql := fmt.Sprintf("ALTER SCHEMA %s OWNER TO %s;", diff.New.Name, diff.New.Owner)
			w.WriteStatementWithComment("SCHEMA", diff.New.Name, "", "", sql, targetSchema)
		}
	}
}

// generateSchemaSQL generates CREATE SCHEMA statement
func generateSchemaSQL(schema *ir.Schema) string {
	if schema.Name == "public" {
		return "" // Skip public schema
	}
	return fmt.Sprintf("CREATE SCHEMA %s;", schema.Name)
}