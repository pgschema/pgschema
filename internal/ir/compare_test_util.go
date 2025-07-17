package ir

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// IRComparisonInput represents the input for semantic IR comparison
type IRComparisonInput struct {
	IR          *IR
	Description string // e.g., "Database IR (from pgdump.sql -> database -> ir/inspector -> IR)"
}

// CompareIRSemanticEquivalence performs enhanced semantic comparison between two IR representations
// This function focuses on semantic equivalence rather than exact structural matching
func CompareIRSemanticEquivalence(t *testing.T, input1, input2 IRComparisonInput) {
	t.Logf("=== SEMANTIC EQUIVALENCE ANALYSIS ===")
	t.Logf("Comparing: %s", input1.Description)
	t.Logf("With:      %s", input2.Description)

	// Log detailed object counts first
	logDetailedObjectCounts(t, input1, input2)

	// Compare top-level schema counts
	if len(input1.IR.Schemas) != len(input2.IR.Schemas) {
		t.Errorf("Schema count mismatch: %s has %d, %s has %d",
			input1.Description, len(input1.IR.Schemas),
			input2.Description, len(input2.IR.Schemas))
	}

	// Compare each schema for semantic equivalence
	for schemaName, schema1 := range input1.IR.Schemas {
		schema2, exists := input2.IR.Schemas[schemaName]
		if !exists {
			t.Errorf("Schema %s not found in %s", schemaName, input2.Description)
			continue
		}

		t.Logf("--- Comparing schema: %s ---", schemaName)
		compareDBSchemaSemanticEquivalence(t, schemaName, schema1, schema2, input1.Description, input2.Description)
	}

	// Check for extra schemas in second IR
	for schemaName := range input2.IR.Schemas {
		if _, exists := input1.IR.Schemas[schemaName]; !exists {
			t.Errorf("Unexpected schema %s found in %s", schemaName, input2.Description)
		}
	}

	// Compare extensions
	compareExtensions(t, input1.IR.Extensions, input2.IR.Extensions, input1.Description, input2.Description)

	t.Logf("=== SEMANTIC EQUIVALENCE ANALYSIS COMPLETED ===")
}

// logDetailedObjectCounts logs detailed object counts for both IR inputs
func logDetailedObjectCounts(t *testing.T, input1, input2 IRComparisonInput) {
	t.Logf("%s has %d schemas", input1.Description, len(input1.IR.Schemas))
	t.Logf("%s has %d schemas", input2.Description, len(input2.IR.Schemas))

	// Detailed object count logging
	for schemaName, schema1 := range input1.IR.Schemas {
		schema2 := input2.IR.Schemas[schemaName]
		if schema2 != nil {
			indexCount1 := countTableLevelIndexes(schema1)
			indexCount2 := countTableLevelIndexes(schema2)
			t.Logf("Schema '%s': %s[tables=%d, views=%d, funcs=%d, seqs=%d, indexes=%d] vs %s[tables=%d, views=%d, funcs=%d, seqs=%d, indexes=%d]",
				schemaName,
				getShortDescription(input1.Description), len(schema1.Tables), len(schema1.Views), len(schema1.Functions), len(schema1.Sequences), indexCount1,
				getShortDescription(input2.Description), len(schema2.Tables), len(schema2.Views), len(schema2.Functions), len(schema2.Sequences), indexCount2)
		}
	}
}

// getShortDescription extracts a short identifier from a long description
func getShortDescription(description string) string {
	if strings.Contains(description, "Database IR") {
		return "DB"
	}
	if strings.Contains(description, "Parser IR") {
		return "Parser"
	}
	return "IR"
}

// compareDBSchemaSemanticEquivalence compares two DBSchema objects for semantic equivalence
func compareDBSchemaSemanticEquivalence(t *testing.T, schemaName string, schema1, schema2 *Schema, desc1, desc2 string) {
	// Compare tables (focus on BASE tables for semantic equivalence)
	baseTables1 := make(map[string]*Table)
	baseTables2 := make(map[string]*Table)

	for name, table := range schema1.Tables {
		if table.Type == TableTypeBase {
			baseTables1[name] = table
		}
	}
	for name, table := range schema2.Tables {
		if table.Type == TableTypeBase {
			baseTables2[name] = table
		}
	}

	if len(baseTables1) != len(baseTables2) {
		t.Errorf("Schema %s: base table count difference: %s has %d, %s has %d (may be due to partition table handling differences)",
			schemaName, desc1, len(baseTables1), desc2, len(baseTables2))
	}

	// Compare each base table
	for tableName, table1 := range baseTables1 {
		table2, exists := baseTables2[tableName]
		if !exists {
			t.Errorf("Schema %s: base table %s not found in %s", schemaName, tableName, desc2)
			continue
		}

		compareTableSemanticEquivalence(t, schemaName, tableName, table1, table2, desc1, desc2)
	}

	// Compare views
	compareViewsSemanticEquivalence(t, schemaName, schema1.Views, schema2.Views, desc1, desc2)

	// Compare functions
	compareFunctionsSemanticEquivalence(t, schemaName, schema1.Functions, schema2.Functions, desc1, desc2)

	// Compare sequences
	compareSequencesSemanticEquivalence(t, schemaName, schema1.Sequences, schema2.Sequences, desc1, desc2)

	// Compare indexes at table level
	compareTableLevelIndexesSemanticEquivalence(t, schemaName, schema1, schema2, desc1, desc2)

	// Log comparison results with table-level index counts
	indexCount1 := countTableLevelIndexes(schema1)
	indexCount2 := countTableLevelIndexes(schema2)
	t.Logf("Schema %s semantic comparison: tables=%d/%d, views=%d/%d, functions=%d/%d, sequences=%d/%d, indexes=%d/%d",
		schemaName,
		len(baseTables2), len(baseTables1),
		len(schema2.Views), len(schema1.Views),
		len(schema2.Functions), len(schema1.Functions),
		len(schema2.Sequences), len(schema1.Sequences),
		indexCount2, indexCount1)
}

// compareTableSemanticEquivalence compares two tables for semantic equivalence
func compareTableSemanticEquivalence(t *testing.T, schemaName, tableName string, table1, table2 *Table, desc1, desc2 string) {
	// Basic properties
	if table1.Name != table2.Name {
		t.Errorf("Table %s.%s: name mismatch: %s has %s, %s has %s",
			schemaName, tableName, desc1, table1.Name, desc2, table2.Name)
	}

	if table1.Schema != table2.Schema {
		t.Errorf("Table %s.%s: schema mismatch: %s has %s, %s has %s",
			schemaName, tableName, desc1, table1.Schema, desc2, table2.Schema)
	}

	// Column count and semantic equivalence
	if len(table1.Columns) != len(table2.Columns) {
		t.Errorf("Table %s.%s: column count mismatch: %s has %d, %s has %d",
			schemaName, tableName, desc1, len(table1.Columns), desc2, len(table2.Columns))
	}

	// Create maps for easier column comparison
	columns1 := make(map[string]*Column)
	columns2 := make(map[string]*Column)

	for _, col := range table1.Columns {
		columns1[col.Name] = col
	}
	for _, col := range table2.Columns {
		columns2[col.Name] = col
	}

	// Compare each column semantically
	for colName, col1 := range columns1 {
		col2, exists := columns2[colName]
		if !exists {
			t.Errorf("Table %s.%s: column %s not found in %s",
				schemaName, tableName, colName, desc2)
			continue
		}

		compareColumnSemanticEquivalence(t, schemaName, tableName, colName, col1, col2, desc1, desc2)
	}

	// Log constraint differences
	if len(table1.Constraints) != len(table2.Constraints) {
		t.Errorf("Table %s.%s: constraint count difference: %s has %d, %s has %d",
			schemaName, tableName, desc1, len(table1.Constraints), desc2, len(table2.Constraints))
	}

	// Compare triggers
	compareTriggersSemanticEquivalence(t, schemaName, tableName, table1.Triggers, table2.Triggers, desc1, desc2)
}

// compareColumnSemanticEquivalence compares columns with focus on semantic equivalence
func compareColumnSemanticEquivalence(t *testing.T, schemaName, tableName, colName string, col1, col2 *Column, desc1, desc2 string) {
	// Position should match
	if col1.Position != col2.Position {
		t.Errorf("Column %s.%s.%s: position mismatch: %s has %d, %s has %d",
			schemaName, tableName, colName, desc1, col1.Position, desc2, col2.Position)
	}

	// Data type should match
	if col1.DataType != col2.DataType {
		t.Errorf("Column %s.%s.%s: data type mismatch: %s has %s, %s has %s",
			schemaName, tableName, colName, desc1, col1.DataType, desc2, col2.DataType)
	}

	// Nullable
	if col1.IsNullable != col2.IsNullable {
		t.Errorf("Column %s.%s.%s: nullable difference: %s has %t, %s has %t (may be due to parsing limitations)",
			schemaName, tableName, colName, desc1, col1.IsNullable, desc2, col2.IsNullable)
	}

	// Default values - strict comparison
	if !areDefaultValuesEqual(col1.DefaultValue, col2.DefaultValue) {
		default1 := "NULL"
		default2 := "NULL"
		if col1.DefaultValue != nil {
			default1 = *col1.DefaultValue
		}
		if col2.DefaultValue != nil {
			default2 = *col2.DefaultValue
		}
		t.Errorf("Column %s.%s.%s: default value mismatch: %s has %q, %s has %q",
			schemaName, tableName, colName, desc1, default1, desc2, default2)
	}
}

// areDefaultValuesEqual checks if default values are semantically equivalent
func areDefaultValuesEqual(val1, val2 *string) bool {
	// Both nil
	if val1 == nil && val2 == nil {
		return true
	}

	// One nil, one not
	if (val1 == nil) != (val2 == nil) {
		return false
	}

	// Both not nil - normalize and compare semantically
	normalized1 := normalizeDefaultValue(*val1)
	normalized2 := normalizeDefaultValue(*val2)
	return normalized1 == normalized2
}

// normalizeDefaultValue normalizes default values for semantic comparison
func normalizeDefaultValue(value string) string {
	// Remove unnecessary whitespace
	value = strings.TrimSpace(value)

	// Handle nextval sequence references - remove schema qualification
	if strings.Contains(value, "nextval(") {
		// Pattern: nextval('schema_name.seq_name'::regclass) -> nextval('seq_name'::regclass)
		re := regexp.MustCompile(`nextval\('([^.]+)\.([^']+)'::regclass\)`)
		if re.MatchString(value) {
			// Replace with unqualified sequence name
			value = re.ReplaceAllString(value, "nextval('$2'::regclass)")
		}
		// Early return for nextval - don't apply type casting normalization
		return value
	}

	// Handle type casting - remove explicit type casts that are semantically equivalent
	// Pattern: ''::text -> ''
	// Pattern: '{}'::jsonb -> '{}'
	if strings.Contains(value, "::") {
		// Find the cast and remove it for simple literal values
		if strings.HasPrefix(value, "'") {
			if idx := strings.Index(value, "'::"); idx != -1 {
				// Find the closing quote
				if closeIdx := strings.Index(value[1:], "'"); closeIdx != -1 {
					literal := value[:closeIdx+2] // Include the closing quote
					if literal == "''" || literal == "'{}'" {
						value = literal
					}
				}
			}
		}
		// Pattern: 'G'::schema.type_name -> 'G'
		// Pattern: 'G'::type_name -> 'G'
		if strings.Contains(value, "'::") {
			if idx := strings.Index(value, "'::"); idx != -1 {
				value = value[:idx+1]
			}
		}
	}

	return value
}

// compareViewsSemanticEquivalence compares views for semantic equivalence
func compareViewsSemanticEquivalence(t *testing.T, schemaName string, views1, views2 map[string]*View, desc1, desc2 string) {
	if len(views1) != len(views2) {
		t.Errorf("Schema %s: view count difference: %s has %d, %s has %d",
			schemaName, desc1, len(views1), desc2, len(views2))
	}

	for viewName := range views1 {
		if _, exists := views2[viewName]; !exists {
			t.Errorf("Schema %s: view %s not found in %s", schemaName, viewName, desc2)
		}
	}
}

// compareFunctionsSemanticEquivalence compares functions for semantic equivalence
func compareFunctionsSemanticEquivalence(t *testing.T, schemaName string, funcs1, funcs2 map[string]*Function, desc1, desc2 string) {
	if len(funcs1) != len(funcs2) {
		t.Errorf("Schema %s: function count difference: %s has %d, %s has %d",
			schemaName, desc1, len(funcs1), desc2, len(funcs2))
	}

	for funcName := range funcs1 {
		if _, exists := funcs2[funcName]; !exists {
			t.Errorf("Schema %s: function %s not found in %s", schemaName, funcName, desc2)
		}
	}
}

// compareSequencesSemanticEquivalence compares sequences for semantic equivalence
func compareSequencesSemanticEquivalence(t *testing.T, schemaName string, seqs1, seqs2 map[string]*Sequence, desc1, desc2 string) {
	if len(seqs1) != len(seqs2) {
		t.Errorf("Schema %s: sequence count difference: %s has %d, %s has %d",
			schemaName, desc1, len(seqs1), desc2, len(seqs2))
	}

	for seqName := range seqs1 {
		if _, exists := seqs2[seqName]; !exists {
			t.Errorf("Schema %s: sequence %s not found in %s", schemaName, seqName, desc2)
		}
	}
}

// compareExtensions compares extensions for semantic equivalence
func compareExtensions(t *testing.T, exts1, exts2 map[string]*Extension, desc1, desc2 string) {
	if len(exts1) != len(exts2) {
		t.Errorf("Extension count difference: %s has %d, %s has %d", desc1, len(exts1), desc2, len(exts2))
	}

	for extName := range exts1 {
		if _, exists := exts2[extName]; !exists {
			t.Errorf("Extension %s not found in %s", extName, desc2)
		}
	}
}

// countTableLevelIndexes counts all indexes stored at table level within a schema
func countTableLevelIndexes(schema *Schema) int {
	count := 0
	for _, table := range schema.Tables {
		count += len(table.Indexes)
	}
	return count
}

// compareTableLevelIndexesSemanticEquivalence compares indexes stored at table level
func compareTableLevelIndexesSemanticEquivalence(t *testing.T, schemaName string, schema1, schema2 *Schema, desc1, desc2 string) {
	// Collect all indexes from tables in first schema
	indexes1 := make(map[string]*Index)
	for tableName, table := range schema1.Tables {
		for indexName, index := range table.Indexes {
			// Use table.index format as key to ensure uniqueness across tables
			key := fmt.Sprintf("%s.%s", tableName, indexName)
			indexes1[key] = index
		}
	}

	// Collect all indexes from tables in second schema
	indexes2 := make(map[string]*Index)
	for tableName, table := range schema2.Tables {
		for indexName, index := range table.Indexes {
			// Use table.index format as key to ensure uniqueness across tables
			key := fmt.Sprintf("%s.%s", tableName, indexName)
			indexes2[key] = index
		}
	}

	// Compare index counts
	if len(indexes1) != len(indexes2) {
		t.Errorf("Schema %s: table-level index count difference: %s has %d, %s has %d",
			schemaName, desc1, len(indexes1), desc2, len(indexes2))
	}

	// Compare each index
	for indexKey, index1 := range indexes1 {
		index2, exists := indexes2[indexKey]
		if !exists {
			t.Errorf("Schema %s: table-level index %s not found in %s", schemaName, indexKey, desc2)
			continue
		}

		compareIndexSemanticEquivalence(t, schemaName, indexKey, index1, index2, desc1, desc2)
	}

	// Check for extra indexes in second schema
	for indexKey := range indexes2 {
		if _, exists := indexes1[indexKey]; !exists {
			t.Errorf("Schema %s: unexpected table-level index %s found in %s", schemaName, indexKey, desc2)
		}
	}
}

// compareIndexSemanticEquivalence compares two indexes for semantic equivalence
func compareIndexSemanticEquivalence(t *testing.T, schemaName, indexName string, index1, index2 *Index, desc1, desc2 string) {
	// Basic properties
	if index1.Name != index2.Name {
		t.Errorf("Index %s.%s: name mismatch: %s has %s, %s has %s",
			schemaName, indexName, desc1, index1.Name, desc2, index2.Name)
	}

	if index1.Schema != index2.Schema {
		t.Errorf("Index %s.%s: schema mismatch: %s has %s, %s has %s",
			schemaName, indexName, desc1, index1.Schema, desc2, index2.Schema)
	}

	if index1.Table != index2.Table {
		t.Errorf("Index %s.%s: table mismatch: %s has %s, %s has %s",
			schemaName, indexName, desc1, index1.Table, desc2, index2.Table)
	}

	// Index type and flags
	if index1.Type != index2.Type {
		t.Errorf("Index %s.%s: type difference: %s has %s, %s has %s (may be acceptable due to semantic differences)",
			schemaName, indexName, desc1, index1.Type, desc2, index2.Type)
	}

	isUnique1 := index1.Type == IndexTypeUnique || index1.Type == IndexTypePrimary
	isUnique2 := index2.Type == IndexTypeUnique || index2.Type == IndexTypePrimary
	if isUnique1 != isUnique2 {
		t.Errorf("Index %s.%s: unique flag mismatch: %s has %t, %s has %t",
			schemaName, indexName, desc1, isUnique1, desc2, isUnique2)
	}

	if index1.Type == IndexTypePrimary != (index2.Type == IndexTypePrimary) {
		t.Errorf("Index %s.%s: primary flag mismatch: %s has %t, %s has %t",
			schemaName, indexName, desc1, index1.Type == IndexTypePrimary, desc2, index2.Type == IndexTypePrimary)
	}

	if index1.IsPartial != index2.IsPartial {
		t.Errorf("Index %s.%s: partial flag difference: %s has %t, %s has %t",
			schemaName, indexName, desc1, index1.IsPartial, desc2, index2.IsPartial)
	}

	// Index method
	if index1.Method != index2.Method {
		t.Errorf("Index %s.%s: method difference: %s has %s, %s has %s",
			schemaName, indexName, desc1, index1.Method, desc2, index2.Method)
	}

	// Column count
	if len(index1.Columns) != len(index2.Columns) {
		t.Errorf("Index %s.%s: column count mismatch: %s has %d, %s has %d",
			schemaName, indexName, desc1, len(index1.Columns), desc2, len(index2.Columns))
	}

	// Compare columns semantically
	columnsMap1 := make(map[int]*IndexColumn)
	columnsMap2 := make(map[int]*IndexColumn)

	for _, col := range index1.Columns {
		columnsMap1[col.Position] = col
	}
	for _, col := range index2.Columns {
		columnsMap2[col.Position] = col
	}

	for position, col1 := range columnsMap1 {
		col2, exists := columnsMap2[position]
		if !exists {
			t.Errorf("Index %s.%s: column at position %d not found in %s",
				schemaName, indexName, position, desc2)
			continue
		}

		compareIndexColumnSemanticEquivalence(t, schemaName, indexName, position, col1, col2, desc1, desc2)
	}

	// Partial index WHERE clause - normalize for comparison
	if index1.IsPartial || index2.IsPartial {
		where1 := strings.TrimSpace(index1.Where)
		where2 := strings.TrimSpace(index2.Where)
		if where1 != where2 {
			t.Errorf("Index %s.%s: WHERE clause difference: %s has %q, %s has %q (may be due to format differences)",
				schemaName, indexName, desc1, where1, desc2, where2)
		}
	}
}

// compareIndexColumnSemanticEquivalence compares index columns for semantic equivalence
func compareIndexColumnSemanticEquivalence(t *testing.T, schemaName, indexName string, position int, col1, col2 *IndexColumn, desc1, desc2 string) {
	if col1.Name != col2.Name {
		t.Errorf("Index %s.%s column at position %d: name mismatch: %s has %s, %s has %s",
			schemaName, indexName, position, desc1, col1.Name, desc2, col2.Name)
	}

	if col1.Position != col2.Position {
		t.Errorf("Index %s.%s column %s: position mismatch: %s has %d, %s has %d",
			schemaName, indexName, col1.Name, desc1, col1.Position, desc2, col2.Position)
	}

	// Direction and operator may have variations
	if col1.Direction != col2.Direction {
		t.Errorf("Index %s.%s column %s: direction difference: %s has %s, %s has %s",
			schemaName, indexName, col1.Name, desc1, col1.Direction, desc2, col2.Direction)
	}

	if col1.Operator != col2.Operator {
		t.Errorf("Index %s.%s column %s: operator difference: %s has %s, %s has %s",
			schemaName, indexName, col1.Name, desc1, col1.Operator, desc2, col2.Operator)
	}
}

// compareTriggersSemanticEquivalence compares triggers for semantic equivalence
func compareTriggersSemanticEquivalence(t *testing.T, schemaName, tableName string, triggers1, triggers2 map[string]*Trigger, desc1, desc2 string) {
	// Check trigger count
	if len(triggers1) != len(triggers2) {
		t.Errorf("Table %s.%s: trigger count difference: %s has %d, %s has %d",
			schemaName, tableName, desc1, len(triggers1), desc2, len(triggers2))
	}

	// Compare each trigger
	for triggerName, trigger1 := range triggers1 {
		trigger2, exists := triggers2[triggerName]
		if !exists {
			t.Errorf("Table %s.%s: trigger %s not found in %s",
				schemaName, tableName, triggerName, desc2)
			continue
		}

		compareTriggerSemanticEquivalence(t, schemaName, tableName, triggerName, trigger1, trigger2, desc1, desc2)
	}

	// Check for extra triggers in second map
	for triggerName := range triggers2 {
		if _, exists := triggers1[triggerName]; !exists {
			t.Errorf("Table %s.%s: unexpected trigger %s found in %s",
				schemaName, tableName, triggerName, desc2)
		}
	}
}

// compareTriggerSemanticEquivalence compares individual triggers for semantic equivalence
func compareTriggerSemanticEquivalence(t *testing.T, schemaName, tableName, triggerName string, trigger1, trigger2 *Trigger, desc1, desc2 string) {
	// Compare basic properties
	if trigger1.Name != trigger2.Name {
		t.Errorf("Trigger %s.%s.%s: name mismatch: %s has %s, %s has %s",
			schemaName, tableName, triggerName, desc1, trigger1.Name, desc2, trigger2.Name)
	}

	if trigger1.Timing != trigger2.Timing {
		t.Errorf("Trigger %s.%s.%s: timing mismatch: %s has %s, %s has %s",
			schemaName, tableName, triggerName, desc1, trigger1.Timing, desc2, trigger2.Timing)
	}

	if trigger1.Level != trigger2.Level {
		t.Errorf("Trigger %s.%s.%s: level mismatch: %s has %s, %s has %s",
			schemaName, tableName, triggerName, desc1, trigger1.Level, desc2, trigger2.Level)
	}

	// Compare events
	if len(trigger1.Events) != len(trigger2.Events) {
		t.Errorf("Trigger %s.%s.%s: event count mismatch: %s has %d, %s has %d",
			schemaName, tableName, triggerName, desc1, len(trigger1.Events), desc2, len(trigger2.Events))
	} else {
		// Compare each event
		for i, event1 := range trigger1.Events {
			if i < len(trigger2.Events) && event1 != trigger2.Events[i] {
				t.Errorf("Trigger %s.%s.%s: event %d mismatch: %s has %s, %s has %s",
					schemaName, tableName, triggerName, i, desc1, event1, desc2, trigger2.Events[i])
			}
		}
	}

	// Compare function calls - this is the critical comparison
	if trigger1.Function != trigger2.Function {
		t.Errorf("Trigger %s.%s.%s: function mismatch: %s has %q, %s has %q",
			schemaName, tableName, triggerName, desc1, trigger1.Function, desc2, trigger2.Function)
	}

	// Compare conditions
	if trigger1.Condition != trigger2.Condition {
		t.Errorf("Trigger %s.%s.%s: condition mismatch: %s has %q, %s has %q",
			schemaName, tableName, triggerName, desc1, trigger1.Condition, desc2, trigger2.Condition)
	}
}
