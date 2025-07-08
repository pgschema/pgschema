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