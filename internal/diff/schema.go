package diff

import (
	"fmt"
	"sort"

	"github.com/pgschema/pgschema/internal/ir"
)

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

		// Skip public schema
		if schema.Name != "public" {
			sql := fmt.Sprintf("CREATE SCHEMA %s;", schema.Name)
			if schema.Owner != "" {
				sql = fmt.Sprintf("CREATE SCHEMA %s AUTHORIZATION %s;", schema.Name, schema.Owner)
			}
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
