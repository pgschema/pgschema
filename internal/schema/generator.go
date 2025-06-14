package schema

import (
	"fmt"
	"sort"
	"strings"
)

// Generator generates SQL DDL from schema IR
type Generator struct {
	schema *Schema
}

// NewGenerator creates a new SQL generator
func NewGenerator(schema *Schema) *Generator {
	return &Generator{schema: schema}
}

// GenerateSQL generates complete SQL DDL from the schema IR
func (g *Generator) GenerateSQL() string {
	var output strings.Builder
	
	// Header
	g.writeHeader(&output)
	
	// Schemas (skip public schema)
	g.writeSchemas(&output)
	
	// Functions
	g.writeFunctions(&output)
	
	// Tables and Views (dependency sorted)
	g.writeTablesAndViews(&output)
	
	// Indexes
	g.writeIndexes(&output)
	
	// Triggers
	g.writeTriggers(&output)
	
	// Foreign Key constraints
	g.writeForeignKeyConstraints(&output)
	
	// RLS
	g.writeRLS(&output)
	
	// Footer
	g.writeFooter(&output)
	
	return output.String()
}

func (g *Generator) writeHeader(output *strings.Builder) {
	output.WriteString("--\n")
	output.WriteString("-- PostgreSQL database dump\n")
	output.WriteString("--\n")
	output.WriteString("\n")
	output.WriteString(fmt.Sprintf("-- Dumped from database version %s\n", g.schema.Metadata.DatabaseVersion))
	output.WriteString(fmt.Sprintf("-- Dumped by %s\n", g.schema.Metadata.DumpVersion))
	output.WriteString("\n")
}

func (g *Generator) writeSchemas(output *strings.Builder) {
	schemaNames := g.schema.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		if schemaName != "public" {
			g.writeComment(output, "SCHEMA", schemaName, schemaName, "")
			output.WriteString(fmt.Sprintf("CREATE SCHEMA %s;\n", schemaName))
			output.WriteString("\n")
		}
	}
}

func (g *Generator) writeFunctions(output *strings.Builder) {
	schemaNames := g.schema.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := g.schema.Schemas[schemaName]
		
		var functionNames []string
		for name := range dbSchema.Functions {
			functionNames = append(functionNames, name)
		}
		sort.Strings(functionNames)
		
		for _, functionName := range functionNames {
			function := dbSchema.Functions[functionName]
			if function.Definition != "<nil>" && function.Definition != "" {
				g.writeComment(output, "FUNCTION", fmt.Sprintf("%s()", functionName), schemaName, "")
				output.WriteString(fmt.Sprintf("CREATE FUNCTION %s.%s() RETURNS %s\n", schemaName, functionName, function.ReturnType))
				output.WriteString(fmt.Sprintf("    LANGUAGE %s\n", strings.ToLower(function.Language)))
				output.WriteString(fmt.Sprintf("    AS $$%s$$;\n", function.Definition))
				output.WriteString("\n")
			}
		}
	}
}

func (g *Generator) writeTablesAndViews(output *strings.Builder) {
	// Get all objects and sort by dependencies
	objects := g.getDependencySortedObjects()
	
	for _, obj := range objects {
		switch obj.Type {
		case "table":
			g.writeTable(output, obj.Schema, obj.Name)
		case "view":
			g.writeView(output, obj.Schema, obj.Name)
		}
	}
}

func (g *Generator) writeTable(output *strings.Builder, schemaName, tableName string) {
	dbSchema := g.schema.Schemas[schemaName]
	table := dbSchema.Tables[tableName]
	
	if table.Type != TableTypeBase {
		return // Skip views here, they're handled separately
	}
	
	// Table definition
	g.writeComment(output, "TABLE", tableName, schemaName, "")
	output.WriteString(fmt.Sprintf("CREATE TABLE %s.%s (\n", schemaName, tableName))
	
	// Columns
	columns := table.SortColumnsByPosition()
	for i, column := range columns {
		output.WriteString("    ")
		g.writeColumnDefinition(output, column)
		if i < len(columns)-1 {
			output.WriteString(",")
		}
		output.WriteString("\n")
	}
	
	output.WriteString(");\n")
	output.WriteString("\n")
	
	// Sequences owned by this table
	g.writeSequencesForTable(output, schemaName, tableName)
	
	// Column defaults
	g.writeColumnDefaults(output, table)
	
	// Primary key and unique constraints
	g.writeTableConstraints(output, table)
}

func (g *Generator) writeView(output *strings.Builder, schemaName, viewName string) {
	dbSchema := g.schema.Schemas[schemaName]
	view := dbSchema.Views[viewName]
	
	g.writeComment(output, "VIEW", viewName, schemaName, "")
	
	// Add schema qualifiers to view definition
	qualifiedDef := g.addSchemaQualifiers(view.Definition, schemaName)
	
	output.WriteString(fmt.Sprintf("CREATE VIEW %s.%s AS\n%s;\n", schemaName, viewName, qualifiedDef))
	output.WriteString("\n")
}

func (g *Generator) writeSequencesForTable(output *strings.Builder, schemaName, tableName string) {
	dbSchema := g.schema.Schemas[schemaName]
	
	var sequenceNames []string
	for name, sequence := range dbSchema.Sequences {
		if sequence.OwnedByTable == tableName {
			sequenceNames = append(sequenceNames, name)
		}
	}
	sort.Strings(sequenceNames)
	
	for _, sequenceName := range sequenceNames {
		sequence := dbSchema.Sequences[sequenceName]
		g.writeSequence(output, sequence)
	}
}

func (g *Generator) writeSequence(output *strings.Builder, sequence *Sequence) {
	g.writeComment(output, "SEQUENCE", sequence.Name, sequence.Schema, "")
	
	output.WriteString(fmt.Sprintf("CREATE SEQUENCE %s.%s\n", sequence.Schema, sequence.Name))
	if sequence.DataType != "" && sequence.DataType != "bigint" {
		output.WriteString(fmt.Sprintf("    AS %s\n", sequence.DataType))
	}
	output.WriteString(fmt.Sprintf("    START WITH %d\n", sequence.StartValue))
	output.WriteString(fmt.Sprintf("    INCREMENT BY %d\n", sequence.Increment))
	
	if sequence.MinValue != nil {
		output.WriteString(fmt.Sprintf("    MINVALUE %d\n", *sequence.MinValue))
	} else {
		output.WriteString("    NO MINVALUE\n")
	}
	
	if sequence.MaxValue != nil {
		output.WriteString(fmt.Sprintf("    MAXVALUE %d\n", *sequence.MaxValue))
	} else {
		output.WriteString("    NO MAXVALUE\n")
	}
	
	output.WriteString("    CACHE 1")
	if sequence.CycleOption {
		output.WriteString("\n    CYCLE")
	}
	output.WriteString(";\n")
	output.WriteString("\n")
	
	// Sequence ownership
	if sequence.OwnedByTable != "" && sequence.OwnedByColumn != "" {
		g.writeComment(output, "SEQUENCE OWNED BY", sequence.Name, sequence.Schema, "")
		output.WriteString(fmt.Sprintf("ALTER SEQUENCE %s.%s OWNED BY %s.%s.%s;\n",
			sequence.Schema, sequence.Name, sequence.Schema, sequence.OwnedByTable, sequence.OwnedByColumn))
		output.WriteString("\n")
	}
}

func (g *Generator) writeColumnDefinition(output *strings.Builder, column *Column) {
	output.WriteString(column.Name)
	output.WriteString(" ")
	
	// Data type
	dataType := column.DataType
	if column.MaxLength != nil && (dataType == "character varying" || dataType == "varchar") {
		dataType = fmt.Sprintf("character varying(%d)", *column.MaxLength)
	} else if column.MaxLength != nil && dataType == "character" {
		dataType = fmt.Sprintf("character(%d)", *column.MaxLength)
	} else if column.Precision != nil && column.Scale != nil {
		dataType = fmt.Sprintf("%s(%d,%d)", dataType, *column.Precision, *column.Scale)
	} else if column.Precision != nil {
		dataType = fmt.Sprintf("%s(%d)", dataType, *column.Precision)
	}
	
	output.WriteString(dataType)
	
	// Not null
	if !column.IsNullable {
		output.WriteString(" NOT NULL")
	}
	
	// Default (only for simple defaults, complex ones are handled separately)
	if column.DefaultValue != nil && !strings.Contains(*column.DefaultValue, "nextval") {
		output.WriteString(fmt.Sprintf(" DEFAULT %s", *column.DefaultValue))
	}
}

func (g *Generator) writeColumnDefaults(output *strings.Builder, table *Table) {
	columns := table.SortColumnsByPosition()
	for _, column := range columns {
		if column.DefaultValue != nil && strings.Contains(*column.DefaultValue, "nextval") {
			g.writeComment(output, "DEFAULT", fmt.Sprintf("%s %s", table.Name, column.Name), table.Schema, "")
			
			// Add schema qualification to nextval
			qualifiedDefault := g.addSchemaQualifiersToNextval(*column.DefaultValue, table.Schema)
			
			output.WriteString(fmt.Sprintf("ALTER TABLE ONLY %s.%s ALTER COLUMN %s SET DEFAULT %s;\n",
				table.Schema, table.Name, column.Name, qualifiedDefault))
			output.WriteString("\n")
			output.WriteString("\n")
		}
	}
}

func (g *Generator) writeTableConstraints(output *strings.Builder, table *Table) {
	constraintNames := table.GetSortedConstraintNames()
	
	for _, constraintName := range constraintNames {
		constraint := table.Constraints[constraintName]
		if constraint.Type == ConstraintTypePrimaryKey || constraint.Type == ConstraintTypeUnique {
			g.writeConstraint(output, constraint)
		}
	}
}

func (g *Generator) writeConstraint(output *strings.Builder, constraint *Constraint) {
	var constraintTypeStr string
	switch constraint.Type {
	case ConstraintTypePrimaryKey:
		constraintTypeStr = "PRIMARY KEY"
	case ConstraintTypeUnique:
		constraintTypeStr = "UNIQUE"
	case ConstraintTypeForeignKey:
		constraintTypeStr = "FOREIGN KEY"
	case ConstraintTypeCheck:
		constraintTypeStr = "CHECK"
	default:
		return
	}
	
	g.writeComment(output, "CONSTRAINT", fmt.Sprintf("%s %s", constraint.Table, constraint.Name), constraint.Schema, "")
	
	// Sort columns by position
	columns := constraint.SortConstraintColumnsByPosition()
	var columnNames []string
	for _, col := range columns {
		columnNames = append(columnNames, col.Name)
	}
	columnList := strings.Join(columnNames, ", ")
	
	output.WriteString(fmt.Sprintf("ALTER TABLE ONLY %s.%s\n", constraint.Schema, constraint.Table))
	
	if constraint.Type == ConstraintTypeCheck {
		output.WriteString(fmt.Sprintf("    ADD CONSTRAINT %s CHECK (%s);\n", constraint.Name, constraint.CheckClause))
	} else {
		output.WriteString(fmt.Sprintf("    ADD CONSTRAINT %s %s (%s);\n", constraint.Name, constraintTypeStr, columnList))
	}
	
	output.WriteString("\n")
	output.WriteString("\n")
}

func (g *Generator) writeIndexes(output *strings.Builder) {
	schemaNames := g.schema.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := g.schema.Schemas[schemaName]
		
		var indexNames []string
		for name := range dbSchema.Indexes {
			indexNames = append(indexNames, name)
		}
		sort.Strings(indexNames)
		
		for _, indexName := range indexNames {
			index := dbSchema.Indexes[indexName]
			g.writeComment(output, "INDEX", indexName, schemaName, "")
			output.WriteString(fmt.Sprintf("%s;\n", index.Definition))
			output.WriteString("\n")
			output.WriteString("\n")
		}
	}
}

func (g *Generator) writeTriggers(output *strings.Builder) {
	schemaNames := g.schema.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := g.schema.Schemas[schemaName]
		
		var triggerNames []string
		for name := range dbSchema.Triggers {
			triggerNames = append(triggerNames, name)
		}
		sort.Strings(triggerNames)
		
		for _, triggerName := range triggerNames {
			trigger := dbSchema.Triggers[triggerName]
			g.writeTrigger(output, trigger)
		}
	}
}

func (g *Generator) writeTrigger(output *strings.Builder, trigger *Trigger) {
	g.writeComment(output, "TRIGGER", fmt.Sprintf("%s %s", trigger.Table, trigger.Name), trigger.Schema, "")
	
	// Build event list
	var events []string
	for _, event := range trigger.Events {
		events = append(events, string(event))
	}
	eventList := strings.Join(events, " OR ")
	
	// Add schema qualification to function
	qualifiedFunction := g.addSchemaQualifiersToTrigger(fmt.Sprintf("EXECUTE FUNCTION %s()", trigger.Function), trigger.Schema)
	
	output.WriteString(fmt.Sprintf("CREATE TRIGGER %s %s %s ON %s.%s FOR EACH %s %s;\n",
		trigger.Name, trigger.Timing, eventList, trigger.Schema, trigger.Table, trigger.Level, qualifiedFunction))
	output.WriteString("\n")
	output.WriteString("\n")
}

func (g *Generator) writeForeignKeyConstraints(output *strings.Builder) {
	schemaNames := g.schema.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := g.schema.Schemas[schemaName]
		
		// Collect all foreign key constraints
		var foreignKeyConstraints []*Constraint
		for _, table := range dbSchema.Tables {
			for _, constraint := range table.Constraints {
				if constraint.Type == ConstraintTypeForeignKey {
					foreignKeyConstraints = append(foreignKeyConstraints, constraint)
				}
			}
		}
		
		// Sort by table name, then constraint name
		sort.Slice(foreignKeyConstraints, func(i, j int) bool {
			if foreignKeyConstraints[i].Table != foreignKeyConstraints[j].Table {
				return foreignKeyConstraints[i].Table < foreignKeyConstraints[j].Table
			}
			return foreignKeyConstraints[i].Name < foreignKeyConstraints[j].Name
		})
		
		for _, constraint := range foreignKeyConstraints {
			g.writeForeignKeyConstraint(output, constraint)
		}
	}
}

func (g *Generator) writeForeignKeyConstraint(output *strings.Builder, constraint *Constraint) {
	g.writeComment(output, "FK CONSTRAINT", fmt.Sprintf("%s %s", constraint.Table, constraint.Name), constraint.Schema, "")
	
	// Sort columns by position
	columns := constraint.SortConstraintColumnsByPosition()
	var columnNames []string
	for _, col := range columns {
		columnNames = append(columnNames, col.Name)
	}
	columnList := strings.Join(columnNames, ", ")
	
	// Sort referenced columns by position
	var refColumnNames []string
	if len(constraint.ReferencedColumns) > 0 {
		refColumns := make([]*ConstraintColumn, len(constraint.ReferencedColumns))
		copy(refColumns, constraint.ReferencedColumns)
		sort.Slice(refColumns, func(i, j int) bool {
			return refColumns[i].Position < refColumns[j].Position
		})
		for _, col := range refColumns {
			refColumnNames = append(refColumnNames, col.Name)
		}
	}
	refColumnList := strings.Join(refColumnNames, ", ")
	
	output.WriteString(fmt.Sprintf("ALTER TABLE ONLY %s.%s\n", constraint.Schema, constraint.Table))
	
	// Build referential actions
	var actions []string
	if constraint.DeleteRule != "" && constraint.DeleteRule != "NO ACTION" {
		actions = append(actions, fmt.Sprintf("ON DELETE %s", constraint.DeleteRule))
	}
	if constraint.UpdateRule != "" && constraint.UpdateRule != "NO ACTION" {
		actions = append(actions, fmt.Sprintf("ON UPDATE %s", constraint.UpdateRule))
	}
	
	actionStr := ""
	if len(actions) > 0 {
		actionStr = " " + strings.Join(actions, " ")
	}
	
	output.WriteString(fmt.Sprintf("    ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s.%s(%s)%s;\n",
		constraint.Name, columnList, constraint.ReferencedSchema, constraint.ReferencedTable, refColumnList, actionStr))
	output.WriteString("\n")
	output.WriteString("\n")
}

func (g *Generator) writeRLS(output *strings.Builder) {
	// RLS enabled tables
	schemaNames := g.schema.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := g.schema.Schemas[schemaName]
		
		var rlsTables []string
		for tableName, table := range dbSchema.Tables {
			if table.RLSEnabled {
				rlsTables = append(rlsTables, tableName)
			}
		}
		sort.Strings(rlsTables)
		
		for _, tableName := range rlsTables {
			g.writeComment(output, "ROW SECURITY", tableName, schemaName, "")
			output.WriteString(fmt.Sprintf("ALTER TABLE %s.%s ENABLE ROW LEVEL SECURITY;\n", schemaName, tableName))
			output.WriteString("\n")
		}
	}
	
	// RLS policies
	for _, schemaName := range schemaNames {
		dbSchema := g.schema.Schemas[schemaName]
		
		var policyNames []string
		for name := range dbSchema.Policies {
			policyNames = append(policyNames, name)
		}
		sort.Strings(policyNames)
		
		for _, policyName := range policyNames {
			policy := dbSchema.Policies[policyName]
			g.writeRLSPolicy(output, policy)
		}
	}
}

func (g *Generator) writeRLSPolicy(output *strings.Builder, policy *RLSPolicy) {
	g.writeComment(output, "POLICY", fmt.Sprintf("%s %s", policy.Table, policy.Name), policy.Schema, "")
	
	policyStmt := fmt.Sprintf("CREATE POLICY %s ON %s.%s", policy.Name, policy.Schema, policy.Table)
	
	// Add command type if specified
	if policy.Command != PolicyCommandAll {
		policyStmt += fmt.Sprintf(" FOR %s", policy.Command)
	}
	
	// Add USING clause if present
	if policy.Using != "" {
		policyStmt += fmt.Sprintf(" USING (%s)", policy.Using)
	}
	
	// Add WITH CHECK clause if present
	if policy.WithCheck != "" {
		policyStmt += fmt.Sprintf(" WITH CHECK (%s)", policy.WithCheck)
	}
	
	policyStmt += ";"
	output.WriteString(policyStmt)
	output.WriteString("\n")
	output.WriteString("\n")
	output.WriteString("\n")
}

func (g *Generator) writeFooter(output *strings.Builder) {
	output.WriteString("--\n")
	output.WriteString("-- PostgreSQL database dump complete\n")
	output.WriteString("--\n")
	output.WriteString("\n")
}

func (g *Generator) writeComment(output *strings.Builder, objectType, objectName, schemaName, owner string) {
	output.WriteString("\n")
	output.WriteString("--\n")
	if owner != "" {
		output.WriteString(fmt.Sprintf("-- Name: %s; Type: %s; Schema: %s; Owner: %s\n", objectName, objectType, schemaName, owner))
	} else {
		output.WriteString(fmt.Sprintf("-- Name: %s; Type: %s; Schema: %s; Owner: -\n", objectName, objectType, schemaName))
	}
	output.WriteString("--\n")
	output.WriteString("\n")
}

// Helper methods for dependency sorting and schema qualification

type dependencyObject struct {
	Schema string
	Name   string
	Type   string
}

func (g *Generator) getDependencySortedObjects() []dependencyObject {
	var objects []dependencyObject
	
	// Add all tables first (they have no dependencies)
	schemaNames := g.schema.GetSortedSchemaNames()
	for _, schemaName := range schemaNames {
		dbSchema := g.schema.Schemas[schemaName]
		
		tableNames := dbSchema.GetSortedTableNames()
		for _, tableName := range tableNames {
			table := dbSchema.Tables[tableName]
			if table.Type == TableTypeBase {
				objects = append(objects, dependencyObject{
					Schema: schemaName,
					Name:   tableName,
					Type:   "table",
				})
			}
		}
	}
	
	// Add views (TODO: implement proper dependency sorting)
	for _, schemaName := range schemaNames {
		dbSchema := g.schema.Schemas[schemaName]
		
		var viewNames []string
		for name := range dbSchema.Views {
			viewNames = append(viewNames, name)
		}
		sort.Strings(viewNames)
		
		for _, viewName := range viewNames {
			objects = append(objects, dependencyObject{
				Schema: schemaName,
				Name:   viewName,
				Type:   "view",
			})
		}
	}
	
	return objects
}

func (g *Generator) addSchemaQualifiers(sqlText, schemaName string) string {
	// TODO: Implement proper schema qualification
	return sqlText
}

func (g *Generator) addSchemaQualifiersToNextval(defaultValue, schemaName string) string {
	result := defaultValue
	nextvalStart := "nextval('"
	
	startIdx := strings.Index(result, nextvalStart)
	if startIdx == -1 {
		return result
	}
	
	nameStart := startIdx + len(nextvalStart)
	endIdx := strings.Index(result[nameStart:], "'")
	if endIdx == -1 {
		return result
	}
	endIdx += nameStart
	
	seqName := result[nameStart:endIdx]
	
	if !strings.Contains(seqName, ".") {
		qualifiedName := fmt.Sprintf("%s.%s", schemaName, seqName)
		result = result[:nameStart] + qualifiedName + result[endIdx:]
	}
	
	return result
}

func (g *Generator) addSchemaQualifiersToTrigger(triggerStmt, schemaName string) string {
	result := triggerStmt
	executeKeyword := "EXECUTE FUNCTION "
	
	startIdx := strings.Index(result, executeKeyword)
	if startIdx == -1 {
		return result
	}
	
	nameStart := startIdx + len(executeKeyword)
	parenIdx := strings.Index(result[nameStart:], "(")
	if parenIdx == -1 {
		return result
	}
	parenIdx += nameStart
	
	funcName := strings.TrimSpace(result[nameStart:parenIdx])
	
	if !strings.Contains(funcName, ".") {
		qualifiedName := fmt.Sprintf("%s.%s", schemaName, funcName)
		result = result[:nameStart] + qualifiedName + result[parenIdx:]
	}
	
	return result
}