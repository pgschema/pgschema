package ir

import (
	"fmt"
	"sort"
	"strings"
)

// SQLGeneratorService handles unified SQL generation from IR differences
type SQLGeneratorService struct {
	includeComments bool
}

// NewSQLGeneratorService creates a new unified SQL generator service
func NewSQLGeneratorService(includeComments bool) *SQLGeneratorService {
	return &SQLGeneratorService{
		includeComments: includeComments,
	}
}

// GenerateSchemaSQL generates SQL from an IR schema as if it were a diff from empty schema
func (s *SQLGeneratorService) GenerateSchemaSQL(schema *Schema, targetSchema string) string {
	emptySchema := NewSchema()
	return s.GenerateDiffSQL(emptySchema, schema, targetSchema)
}

// GenerateDiffSQL generates SQL from the differences between two schemas
func (s *SQLGeneratorService) GenerateDiffSQL(oldSchema, newSchema *Schema, targetSchema string) string {
	w := NewSQLWriterWithComments(s.includeComments)

	// Write header comments
	if s.includeComments {
		s.writeHeader(w, newSchema)
	}

	// Generate DDL in dependency order
	s.generateExtensionsSQL(w, oldSchema, newSchema)
	s.generateSchemasSQL(w, oldSchema, newSchema, targetSchema)
	s.generateTypesSQL(w, oldSchema, newSchema, targetSchema)
	s.generateSequencesSQL(w, oldSchema, newSchema, targetSchema)
	s.generateTablesSQL(w, oldSchema, newSchema, targetSchema)
	s.generateViewsSQL(w, oldSchema, newSchema, targetSchema)
	s.generateIndexesSQL(w, oldSchema, newSchema, targetSchema)
	s.generateConstraintsSQL(w, oldSchema, newSchema, targetSchema)
	s.generateFunctionsSQL(w, oldSchema, newSchema, targetSchema)
	s.generateAggregatesSQL(w, oldSchema, newSchema, targetSchema)
	s.generateProceduresSQL(w, oldSchema, newSchema, targetSchema)
	s.generateTriggersSQL(w, oldSchema, newSchema, targetSchema)
	s.generatePoliciesSQL(w, oldSchema, newSchema, targetSchema)

	return w.String()
}

// writeHeader writes the SQL header comments
func (s *SQLGeneratorService) writeHeader(w *SQLWriter, schema *Schema) {
	w.WriteString("--\n")
	w.WriteString("-- PostgreSQL database dump\n")
	w.WriteString("--\n")
	w.WriteString("\n")
	w.WriteString(fmt.Sprintf("-- Dumped from database version %s\n", schema.Metadata.DatabaseVersion))
	w.WriteString(fmt.Sprintf("-- Dumped by %s\n", schema.Metadata.DumpVersion))
}

// generateExtensionsSQL generates SQL for extension differences
func (s *SQLGeneratorService) generateExtensionsSQL(w *SQLWriter, oldSchema, newSchema *Schema) {
	// Get sorted extension names for consistent output
	extensionNames := newSchema.GetSortedExtensionNames()
	for _, name := range extensionNames {
		ext := newSchema.Extensions[name]
		if _, exists := oldSchema.Extensions[name]; !exists {
			w.WriteDDLSeparator()
			w.WriteStatementWithComment("EXTENSION", ext.Name, ext.Schema, "", ext.GenerateSQL(), "")
		}
	}
}

// generateSchemasSQL generates SQL for schema differences
func (s *SQLGeneratorService) generateSchemasSQL(w *SQLWriter, oldSchema, newSchema *Schema, targetSchema string) {
	// Get sorted schema names for consistent output
	schemaNames := newSchema.GetSortedSchemaNames()
	for _, name := range schemaNames {
		schema := newSchema.Schemas[name]
		if _, exists := oldSchema.Schemas[name]; !exists {
			// Skip creating the target schema if we're doing a schema-specific dump
			// as it's assumed to already exist
			if targetSchema != "" && name == targetSchema {
				continue
			}
			if sql := schema.GenerateSQL(); sql != "" {
				w.WriteDDLSeparator()
				w.WriteString(sql)
			}
		}
	}
}

// generateTypesSQL generates SQL for type differences
func (s *SQLGeneratorService) generateTypesSQL(w *SQLWriter, oldSchema, newSchema *Schema, targetSchema string) {
	// Get sorted schema names for consistent output
	schemaNames := newSchema.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		if targetSchema != "" && schemaName != targetSchema {
			continue
		}

		schema := newSchema.Schemas[schemaName]
		oldSchemaObj := oldSchema.Schemas[schemaName]
		
		// Get all types that need to be created
		var typesToCreate []*Type
		for typeName, typeObj := range schema.Types {
			if oldSchemaObj == nil || oldSchemaObj.Types[typeName] == nil {
				typesToCreate = append(typesToCreate, typeObj)
			}
		}
		
		// Sort types: CREATE TYPE statements first, then CREATE DOMAIN statements
		// Within each category, sort alphabetically by name
		sort.Slice(typesToCreate, func(i, j int) bool {
			typeI := typesToCreate[i]
			typeJ := typesToCreate[j]
			
			// Domain types should come after non-domain types
			if typeI.Kind == TypeKindDomain && typeJ.Kind != TypeKindDomain {
				return false
			}
			if typeI.Kind != TypeKindDomain && typeJ.Kind == TypeKindDomain {
				return true
			}
			
			// Within the same category, sort alphabetically by name
			return typeI.Name < typeJ.Name
		})
		
		// Generate SQL for each type in sorted order
		for _, typeObj := range typesToCreate {
			w.WriteDDLSeparator()
			sql := typeObj.GenerateSQLWithOptions(false, targetSchema)
			
			// Use correct object type for comment
			var objectType string
			switch typeObj.Kind {
			case TypeKindDomain:
				objectType = "DOMAIN"
			default:
				objectType = "TYPE"
			}
			
			w.WriteStatementWithComment(objectType, typeObj.Name, schemaName, "", sql, targetSchema)
		}
	}
}

// generateSequencesSQL generates SQL for sequence differences
func (s *SQLGeneratorService) generateSequencesSQL(w *SQLWriter, oldSchema, newSchema *Schema, targetSchema string) {
	// Get sorted schema names for consistent output
	schemaNames := newSchema.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		if targetSchema != "" && schemaName != targetSchema {
			continue
		}

		schema := newSchema.Schemas[schemaName]
		oldSchemaObj := oldSchema.Schemas[schemaName]
		
		// Get sorted sequence names for consistent output
		sequenceNames := schema.GetSortedSequenceNames()
		for _, seqName := range sequenceNames {
			seq := schema.Sequences[seqName]
			if oldSchemaObj == nil || oldSchemaObj.Sequences[seqName] == nil {
				// Skip sequences that are owned by SERIAL columns
				if seq.OwnedByTable != "" && seq.OwnedByColumn != "" {
					continue
				}
				w.WriteDDLSeparator()
				sql := seq.GenerateSQLWithOptions(false, targetSchema)
				w.WriteStatementWithComment("SEQUENCE", seqName, schemaName, "", sql, targetSchema)
			}
		}
	}
}

// generateTablesSQL generates SQL for table differences
func (s *SQLGeneratorService) generateTablesSQL(w *SQLWriter, oldSchema, newSchema *Schema, targetSchema string) {
	// Get sorted schema names for consistent output
	schemaNames := newSchema.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		if targetSchema != "" && schemaName != targetSchema {
			continue
		}

		schema := newSchema.Schemas[schemaName]
		oldSchemaObj := oldSchema.Schemas[schemaName]
		
		// Build partition parent->children mapping for co-location
		partitionChildren := make(map[string][]string)
		childToParent := make(map[string]string)
		
		for _, attachment := range newSchema.PartitionAttachments {
			if attachment.ParentSchema == schemaName && attachment.ChildSchema == schemaName {
				partitionChildren[attachment.ParentTable] = append(partitionChildren[attachment.ParentTable], attachment.ChildTable)
				childToParent[attachment.ChildTable] = attachment.ParentTable
			}
		}
		
		// Get topologically sorted table names for dependency-aware output
		tableNames := schema.GetTopologicallySortedTableNames()
		processedTables := make(map[string]bool)
		
		// Process tables in order: partitioned parents first, then their children, then other tables
		for _, tableName := range tableNames {
			table := schema.Tables[tableName]
			
			// Skip if already processed or not a new table
			if processedTables[tableName] || (oldSchemaObj != nil && oldSchemaObj.Tables[tableName] != nil) {
				continue
			}
			
			// If this is a partition child, skip it for now (will be processed with parent)
			if _, isChild := childToParent[tableName]; isChild {
				continue
			}
			
			// Output the table
			w.WriteDDLSeparator()
			sql := table.GenerateSQLWithOptions(false, targetSchema)
			w.WriteStatementWithComment("TABLE", tableName, schemaName, "", sql, targetSchema)
			processedTables[tableName] = true
			
			// If this table has partitions, output them immediately after the parent
			if children, hasChildren := partitionChildren[tableName]; hasChildren {
				for _, childName := range children {
					if childTable := schema.Tables[childName]; childTable != nil && !processedTables[childName] {
						if oldSchemaObj == nil || oldSchemaObj.Tables[childName] == nil {
							w.WriteDDLSeparator()
							sql := childTable.GenerateSQLWithOptions(false, targetSchema)
							w.WriteStatementWithComment("TABLE", childName, schemaName, "", sql, targetSchema)
							processedTables[childName] = true
						}
					}
				}
			}
		}
	}
}

// generateViewsSQL generates SQL for view differences
func (s *SQLGeneratorService) generateViewsSQL(w *SQLWriter, oldSchema, newSchema *Schema, targetSchema string) {
	// Get sorted schema names for consistent output
	schemaNames := newSchema.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		if targetSchema != "" && schemaName != targetSchema {
			continue
		}

		schema := newSchema.Schemas[schemaName]
		oldSchemaObj := oldSchema.Schemas[schemaName]
		// Get topologically sorted view names to handle dependencies
		viewNames := schema.GetTopologicallySortedViewNames()
		for _, viewName := range viewNames {
			view := schema.Views[viewName]
			if oldSchemaObj == nil || oldSchemaObj.Views[viewName] == nil {
				w.WriteDDLSeparator()
				sql := view.GenerateSQLWithOptions(false, targetSchema)
				w.WriteStatementWithComment("VIEW", viewName, schemaName, "", sql, targetSchema)
			}
		}
	}
}

// generateIndexesSQL generates SQL for index differences
func (s *SQLGeneratorService) generateIndexesSQL(w *SQLWriter, oldSchema, newSchema *Schema, targetSchema string) {
	// Get sorted schema names for consistent output
	schemaNames := newSchema.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		if targetSchema != "" && schemaName != targetSchema {
			continue
		}

		schema := newSchema.Schemas[schemaName]
		oldSchemaObj := oldSchema.Schemas[schemaName]
		// Get sorted index names for consistent output
		indexNames := schema.GetSortedIndexNames()
		for _, indexName := range indexNames {
			index := schema.Indexes[indexName]
			if oldSchemaObj == nil || oldSchemaObj.Indexes[indexName] == nil {
				// Skip primary key indexes as they're handled with constraints
				if index.IsPrimary {
					continue
				}

				w.WriteDDLSeparator()
				sql := s.generateIndexSQL(index, targetSchema)
				// No need to strip comments for indexes as generateIndexSQL doesn't add them
				w.WriteStatementWithComment("INDEX", indexName, schemaName, "", sql, targetSchema)
			}
		}
	}
}

// generateConstraintsSQL generates SQL for constraint differences
func (s *SQLGeneratorService) generateConstraintsSQL(w *SQLWriter, oldSchema, newSchema *Schema, targetSchema string) {
	// Get sorted schema names for consistent output
	schemaNames := newSchema.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		if targetSchema != "" && schemaName != targetSchema {
			continue
		}

		schema := newSchema.Schemas[schemaName]
		oldSchemaObj := oldSchema.Schemas[schemaName]
		
		// Get sorted table names for consistent output
		tableNames := schema.GetSortedTableNames()
		for _, tableName := range tableNames {
			table := schema.Tables[tableName]
			if oldSchemaObj == nil || oldSchemaObj.Tables[tableName] == nil {
				// Get sorted constraint names for consistent output
				constraintNames := table.GetSortedConstraintNames()
				for _, constraintName := range constraintNames {
					constraint := table.Constraints[constraintName]
					// Skip PRIMARY KEY, UNIQUE, FOREIGN KEY, and CHECK constraints as they are now inline in CREATE TABLE
					if constraint.Type == ConstraintTypePrimaryKey || 
					   constraint.Type == ConstraintTypeUnique || 
					   constraint.Type == ConstraintTypeForeignKey ||
					   constraint.Type == ConstraintTypeCheck {
						continue
					}
					w.WriteDDLSeparator()
					constraintSQL := constraint.GenerateSQLWithOptions(false, targetSchema)
					w.WriteStatementWithComment("CONSTRAINT", constraintName, schemaName, "", constraintSQL, targetSchema)
				}
			}
		}
	}
}

// generateFunctionsSQL generates SQL for function differences
func (s *SQLGeneratorService) generateFunctionsSQL(w *SQLWriter, oldSchema, newSchema *Schema, targetSchema string) {
	// Get sorted schema names for consistent output
	schemaNames := newSchema.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		if targetSchema != "" && schemaName != targetSchema {
			continue
		}

		schema := newSchema.Schemas[schemaName]
		oldSchemaObj := oldSchema.Schemas[schemaName]
		
		// Get sorted function names for consistent output
		functionNames := schema.GetSortedFunctionNames()
		for _, funcName := range functionNames {
			function := schema.Functions[funcName]
			if oldSchemaObj == nil || oldSchemaObj.Functions[funcName] == nil {
				w.WriteDDLSeparator()
				sql := function.GenerateSQLWithOptions(false, targetSchema)
				w.WriteStatementWithComment("FUNCTION", funcName, schemaName, "", sql, targetSchema)
			}
		}
	}
}

// generateAggregatesSQL generates SQL for aggregate differences
func (s *SQLGeneratorService) generateAggregatesSQL(w *SQLWriter, oldSchema, newSchema *Schema, targetSchema string) {
	// Get sorted schema names for consistent output
	schemaNames := newSchema.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		if targetSchema != "" && schemaName != targetSchema {
			continue
		}

		schema := newSchema.Schemas[schemaName]
		oldSchemaObj := oldSchema.Schemas[schemaName]
		
		// Get sorted aggregate names for consistent output
		aggregateNames := schema.GetSortedAggregateNames()
		for _, aggName := range aggregateNames {
			aggregate := schema.Aggregates[aggName]
			if oldSchemaObj == nil || oldSchemaObj.Aggregates[aggName] == nil {
				w.WriteDDLSeparator()
				sql := aggregate.GenerateSQLWithOptions(false, targetSchema)
				w.WriteStatementWithComment("AGGREGATE", aggName, schemaName, "", sql, targetSchema)
			}
		}
	}
}

// generateProceduresSQL generates SQL for procedure differences
func (s *SQLGeneratorService) generateProceduresSQL(w *SQLWriter, oldSchema, newSchema *Schema, targetSchema string) {
	// Get sorted schema names for consistent output
	schemaNames := newSchema.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		if targetSchema != "" && schemaName != targetSchema {
			continue
		}

		schema := newSchema.Schemas[schemaName]
		oldSchemaObj := oldSchema.Schemas[schemaName]
		
		// Get sorted procedure names for consistent output
		procedureNames := schema.GetSortedProcedureNames()
		for _, procName := range procedureNames {
			procedure := schema.Procedures[procName]
			if oldSchemaObj == nil || oldSchemaObj.Procedures[procName] == nil {
				w.WriteDDLSeparator()
				sql := procedure.GenerateSQLWithOptions(false, targetSchema)
				w.WriteStatementWithComment("PROCEDURE", procName, schemaName, "", sql, targetSchema)
			}
		}
	}
}

// generateTriggersSQL generates SQL for trigger differences
func (s *SQLGeneratorService) generateTriggersSQL(w *SQLWriter, oldSchema, newSchema *Schema, targetSchema string) {
	// Get sorted schema names for consistent output
	schemaNames := newSchema.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		if targetSchema != "" && schemaName != targetSchema {
			continue
		}

		schema := newSchema.Schemas[schemaName]
		oldSchemaObj := oldSchema.Schemas[schemaName]
		
		// Get sorted trigger names for consistent output
		triggerNames := schema.GetSortedTriggerNames()
		for _, triggerName := range triggerNames {
			trigger := schema.Triggers[triggerName]
			if oldSchemaObj == nil || oldSchemaObj.Triggers[triggerName] == nil {
				w.WriteDDLSeparator()
				sql := trigger.GenerateSQLWithOptions(false, targetSchema)
				w.WriteStatementWithComment("TRIGGER", triggerName, schemaName, "", sql, targetSchema)
			}
		}
	}
}

// generatePoliciesSQL generates SQL for policy differences
func (s *SQLGeneratorService) generatePoliciesSQL(w *SQLWriter, oldSchema, newSchema *Schema, targetSchema string) {
	// Get sorted schema names for consistent output
	schemaNames := newSchema.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		if targetSchema != "" && schemaName != targetSchema {
			continue
		}

		schema := newSchema.Schemas[schemaName]
		oldSchemaObj := oldSchema.Schemas[schemaName]
		// Get sorted policy names for consistent output
		policyNames := schema.GetSortedPolicyNames()
		for _, policyName := range policyNames {
			policy := schema.Policies[policyName]
			if oldSchemaObj == nil || oldSchemaObj.Policies[policyName] == nil {
				w.WriteDDLSeparator()
				sql := policy.GenerateSQLWithOptions(false, targetSchema)
				w.WriteStatementWithComment("POLICY", policyName, schemaName, "", sql, targetSchema)
			}
		}
	}
}

// generateIndexSQL generates SQL for an index with unified formatting
func (s *SQLGeneratorService) generateIndexSQL(index *Index, targetSchema string) string {
	sql := SimplifyExpressionIndexDefinition(index.Definition, index.Table)
	if !strings.HasSuffix(sql, ";") {
		sql += ";"
	}
	return sql
}

// generateConstraintSQL generates SQL for a constraint with unified formatting
func (s *SQLGeneratorService) generateConstraintSQL(constraint *Constraint, targetSchema string) string {
	return constraint.GenerateSQL(targetSchema)
}


