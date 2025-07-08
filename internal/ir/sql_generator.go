package ir

import (
	"fmt"
	"sort"
	"strings"
)

// SQLGeneratorService handles unified SQL generation from IR differences
type SQLGeneratorService struct {
	includeComments bool
	targetSchema    string
}

// NewSQLGeneratorService creates a new unified SQL generator service
func NewSQLGeneratorService(includeComments bool, targetSchema string) *SQLGeneratorService {
	return &SQLGeneratorService{
		includeComments: includeComments,
		targetSchema:    targetSchema,
	}
}

// GenerateDiff generates SQL from the differences between two schemas
func (s *SQLGeneratorService) GenerateDiff(oldIR, newIR *IR) string {
	w := NewSQLWriterWithComments(s.includeComments)

	// Write header comments
	if s.includeComments {
		s.writeHeader(w, newIR)
	}

	// Generate DDL in dependency order
	s.generateExtensionsSQL(w, oldIR, newIR)
	s.generateSchemasSQL(w, oldIR, newIR)
	s.generateTypesSQL(w, oldIR, newIR)
	s.generateSequencesSQL(w, oldIR, newIR)
	s.generateTablesSQL(w, oldIR, newIR) // Now includes indexes, constraints, and triggers
	s.generateViewsSQL(w, oldIR, newIR)
	s.generateFunctionsSQL(w, oldIR, newIR)
	s.generateAggregatesSQL(w, oldIR, newIR)
	s.generateProceduresSQL(w, oldIR, newIR)

	return w.String()
}

// writeHeader writes the SQL header comments
func (s *SQLGeneratorService) writeHeader(w *SQLWriter, schema *IR) {
	w.WriteString("--\n")
	w.WriteString("-- PostgreSQL database dump\n")
	w.WriteString("--\n")
	w.WriteString("\n")
	w.WriteString(fmt.Sprintf("-- Dumped from database version %s\n", schema.Metadata.DatabaseVersion))
	w.WriteString(fmt.Sprintf("-- Dumped by %s\n", schema.Metadata.DumpVersion))
}

// generateExtensionsSQL generates SQL for extension differences
func (s *SQLGeneratorService) generateExtensionsSQL(w *SQLWriter, oldIR, newIR *IR) {
	// Get sorted extension names for consistent output
	extensionNames := newIR.GetSortedExtensionNames()
	for _, name := range extensionNames {
		ext := newIR.Extensions[name]
		if _, exists := oldIR.Extensions[name]; !exists {
			w.WriteDDLSeparator()
			w.WriteStatementWithComment("EXTENSION", ext.Name, ext.Schema, "", ext.GenerateSQL(), "")
		}
	}
}

// generateSchemasSQL generates SQL for schema differences
func (s *SQLGeneratorService) generateSchemasSQL(w *SQLWriter, oldIR, newIR *IR) {
	// Get sorted schema names for consistent output
	schemaNames := newIR.GetSortedSchemaNames()
	for _, name := range schemaNames {
		schema := newIR.Schemas[name]
		if _, exists := oldIR.Schemas[name]; !exists {
			// Skip creating the target schema if we're doing a schema-specific dump
			// as it's assumed to already exist
			if s.targetSchema != "" && name == s.targetSchema {
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
func (s *SQLGeneratorService) generateTypesSQL(w *SQLWriter, oldIR, newIR *IR) {
	// Get sorted schema names for consistent output
	schemaNames := newIR.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		if s.targetSchema != "" && schemaName != s.targetSchema {
			continue
		}

		schema := newIR.Schemas[schemaName]
		oldIRSchema := oldIR.Schemas[schemaName]

		// Get all types that need to be created
		var typesToCreate []*Type
		for typeName, typeObj := range schema.Types {
			if oldIRSchema == nil || oldIRSchema.Types[typeName] == nil {
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
			sql := typeObj.GenerateSQLWithOptions(false, s.targetSchema)

			// Use correct object type for comment
			var objectType string
			switch typeObj.Kind {
			case TypeKindDomain:
				objectType = "DOMAIN"
			default:
				objectType = "TYPE"
			}

			w.WriteStatementWithComment(objectType, typeObj.Name, schemaName, "", sql, s.targetSchema)
		}
	}
}

// generateSequencesSQL generates SQL for sequence differences
func (s *SQLGeneratorService) generateSequencesSQL(w *SQLWriter, oldIR, newIR *IR) {
	// Get sorted schema names for consistent output
	schemaNames := newIR.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		if s.targetSchema != "" && schemaName != s.targetSchema {
			continue
		}

		schema := newIR.Schemas[schemaName]
		oldIRSchema := oldIR.Schemas[schemaName]

		// Get sorted sequence names for consistent output
		sequenceNames := schema.GetSortedSequenceNames()
		for _, seqName := range sequenceNames {
			seq := schema.Sequences[seqName]
			if oldIRSchema == nil || oldIRSchema.Sequences[seqName] == nil {
				// Skip sequences that are owned by SERIAL columns
				if seq.OwnedByTable != "" && seq.OwnedByColumn != "" {
					continue
				}
				w.WriteDDLSeparator()
				sql := seq.GenerateSQLWithOptions(false, s.targetSchema)
				w.WriteStatementWithComment("SEQUENCE", seqName, schemaName, "", sql, s.targetSchema)
			}
		}
	}
}

// generateTablesSQL generates SQL for table differences
func (s *SQLGeneratorService) generateTablesSQL(w *SQLWriter, oldIR, newIR *IR) {
	// Get sorted schema names for consistent output
	schemaNames := newIR.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		if s.targetSchema != "" && schemaName != s.targetSchema {
			continue
		}

		schema := newIR.Schemas[schemaName]
		oldIRSchema := oldIR.Schemas[schemaName]

		// Build partition parent->children mapping for co-location
		partitionChildren := make(map[string][]string)
		childToParent := make(map[string]string)

		for _, attachment := range newIR.PartitionAttachments {
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
			if processedTables[tableName] || (oldIRSchema != nil && oldIRSchema.Tables[tableName] != nil) {
				continue
			}

			// If this is a partition child, skip it for now (will be processed with parent)
			if _, isChild := childToParent[tableName]; isChild {
				continue
			}

			// Output the table
			w.WriteDDLSeparator()
			sql := table.GenerateSQLWithOptions(false, s.targetSchema)
			w.WriteStatementWithComment("TABLE", tableName, schemaName, "", sql, s.targetSchema)
			processedTables[tableName] = true

			// Output indexes for this table (excluding primary key indexes)
			s.generateTableIndexes(w, table, schemaName)

			// Output constraints for this table (excluding inline constraints)
			s.generateTableConstraints(w, table, schemaName)

			// Output triggers for this table
			s.generateTableTriggers(w, table, schemaName)

			// Output RLS enablement and policies for this table
			s.generateTableRLS(w, table, schemaName)

			// If this table has partitions, output them immediately after the parent
			if children, hasChildren := partitionChildren[tableName]; hasChildren {
				for _, childName := range children {
					if childTable := schema.Tables[childName]; childTable != nil && !processedTables[childName] {
						if oldIRSchema == nil || oldIRSchema.Tables[childName] == nil {
							w.WriteDDLSeparator()
							sql := childTable.GenerateSQLWithOptions(false, s.targetSchema)
							w.WriteStatementWithComment("TABLE", childName, schemaName, "", sql, s.targetSchema)
							processedTables[childName] = true

							// Output indexes, constraints, triggers, and RLS for the partition table
							s.generateTableIndexes(w, childTable, schemaName)
							s.generateTableConstraints(w, childTable, schemaName)
							s.generateTableTriggers(w, childTable, schemaName)
							s.generateTableRLS(w, childTable, schemaName)
						}
					}
				}
			}
		}
	}
}

// generateViewsSQL generates SQL for view differences
func (s *SQLGeneratorService) generateViewsSQL(w *SQLWriter, oldIR, newIR *IR) {
	// Get sorted schema names for consistent output
	schemaNames := newIR.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		if s.targetSchema != "" && schemaName != s.targetSchema {
			continue
		}

		schema := newIR.Schemas[schemaName]
		oldIRSchema := oldIR.Schemas[schemaName]
		// Get topologically sorted view names to handle dependencies
		viewNames := schema.GetTopologicallySortedViewNames()
		for _, viewName := range viewNames {
			view := schema.Views[viewName]
			if oldIRSchema == nil || oldIRSchema.Views[viewName] == nil {
				w.WriteDDLSeparator()
				sql := view.GenerateSQLWithOptions(false, s.targetSchema)
				w.WriteStatementWithComment("VIEW", viewName, schemaName, "", sql, s.targetSchema)
			}
		}
	}
}

// generateFunctionsSQL generates SQL for function differences
func (s *SQLGeneratorService) generateFunctionsSQL(w *SQLWriter, oldIR, newIR *IR) {
	// Get sorted schema names for consistent output
	schemaNames := newIR.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		if s.targetSchema != "" && schemaName != s.targetSchema {
			continue
		}

		schema := newIR.Schemas[schemaName]
		oldIRSchema := oldIR.Schemas[schemaName]

		// Get sorted function names for consistent output
		functionNames := schema.GetSortedFunctionNames()
		for _, funcName := range functionNames {
			function := schema.Functions[funcName]
			if oldIRSchema == nil || oldIRSchema.Functions[funcName] == nil {
				w.WriteDDLSeparator()
				sql := function.GenerateSQLWithOptions(false, s.targetSchema)
				w.WriteStatementWithComment("FUNCTION", funcName, schemaName, "", sql, s.targetSchema)
			}
		}
	}
}

// generateAggregatesSQL generates SQL for aggregate differences
func (s *SQLGeneratorService) generateAggregatesSQL(w *SQLWriter, oldIR, newIR *IR) {
	// Get sorted schema names for consistent output
	schemaNames := newIR.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		if s.targetSchema != "" && schemaName != s.targetSchema {
			continue
		}

		schema := newIR.Schemas[schemaName]
		oldIRSchema := oldIR.Schemas[schemaName]

		// Get sorted aggregate names for consistent output
		aggregateNames := schema.GetSortedAggregateNames()
		for _, aggName := range aggregateNames {
			aggregate := schema.Aggregates[aggName]
			if oldIRSchema == nil || oldIRSchema.Aggregates[aggName] == nil {
				w.WriteDDLSeparator()
				sql := aggregate.GenerateSQLWithOptions(false, s.targetSchema)
				w.WriteStatementWithComment("AGGREGATE", aggName, schemaName, "", sql, s.targetSchema)
			}
		}
	}
}

// generateProceduresSQL generates SQL for procedure differences
func (s *SQLGeneratorService) generateProceduresSQL(w *SQLWriter, oldIR, newIR *IR) {
	// Get sorted schema names for consistent output
	schemaNames := newIR.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		if s.targetSchema != "" && schemaName != s.targetSchema {
			continue
		}

		schema := newIR.Schemas[schemaName]
		oldIRSchema := oldIR.Schemas[schemaName]

		// Get sorted procedure names for consistent output
		procedureNames := schema.GetSortedProcedureNames()
		for _, procName := range procedureNames {
			procedure := schema.Procedures[procName]
			if oldIRSchema == nil || oldIRSchema.Procedures[procName] == nil {
				w.WriteDDLSeparator()
				sql := procedure.GenerateSQLWithOptions(false, s.targetSchema)
				w.WriteStatementWithComment("PROCEDURE", procName, schemaName, "", sql, s.targetSchema)
			}
		}
	}
}

// generateIndexSQL generates SQL for an index with unified formatting
func (s *SQLGeneratorService) generateIndexSQL(index *Index) string {
	sql := SimplifyExpressionIndexDefinition(index.Definition, index.Table)
	if !strings.HasSuffix(sql, ";") {
		sql += ";"
	}
	return sql
}

// generateTableIndexes generates SQL for indexes belonging to a specific table
func (s *SQLGeneratorService) generateTableIndexes(w *SQLWriter, table *Table, schemaName string) {
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

		w.WriteDDLSeparator()
		sql := s.generateIndexSQL(index)
		w.WriteStatementWithComment("INDEX", indexName, schemaName, "", sql, s.targetSchema)
	}
}

// generateTableConstraints generates SQL for constraints belonging to a specific table
func (s *SQLGeneratorService) generateTableConstraints(w *SQLWriter, table *Table, schemaName string) {
	// Get sorted constraint names for consistent output
	constraintNames := make([]string, 0, len(table.Constraints))
	for constraintName := range table.Constraints {
		constraintNames = append(constraintNames, constraintName)
	}
	sort.Strings(constraintNames)

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
		constraintSQL := constraint.GenerateSQLWithOptions(false, s.targetSchema)
		w.WriteStatementWithComment("CONSTRAINT", constraintName, schemaName, "", constraintSQL, s.targetSchema)
	}
}

// generateTableTriggers generates SQL for triggers belonging to a specific table
func (s *SQLGeneratorService) generateTableTriggers(w *SQLWriter, table *Table, schemaName string) {
	// Get sorted trigger names for consistent output
	triggerNames := make([]string, 0, len(table.Triggers))
	for triggerName := range table.Triggers {
		triggerNames = append(triggerNames, triggerName)
	}
	sort.Strings(triggerNames)

	for _, triggerName := range triggerNames {
		trigger := table.Triggers[triggerName]
		w.WriteDDLSeparator()
		sql := trigger.GenerateSQLWithOptions(false, s.targetSchema)
		w.WriteStatementWithComment("TRIGGER", triggerName, schemaName, "", sql, s.targetSchema)
	}
}

// generateTableRLS generates RLS enablement and policies for a specific table
func (s *SQLGeneratorService) generateTableRLS(w *SQLWriter, table *Table, schemaName string) {
	tableName := table.Name

	// Generate ALTER TABLE ... ENABLE ROW LEVEL SECURITY if needed
	if table.RLSEnabled {
		w.WriteDDLSeparator()
		var fullTableName string
		if schemaName == s.targetSchema {
			fullTableName = tableName
		} else {
			fullTableName = fmt.Sprintf("%s.%s", schemaName, tableName)
		}
		sql := fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY;", fullTableName)
		w.WriteStatementWithComment("TABLE", tableName, schemaName, "", sql, "")
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
		w.WriteDDLSeparator()
		sql := policy.GenerateSQLWithOptions(false, s.targetSchema)
		w.WriteStatementWithComment("POLICY", policyName, schemaName, "", sql, s.targetSchema)
	}
}
