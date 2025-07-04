package ir

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/pgschema/pgschema/testutil"
)

// IR Integration Tests
// These comprehensive integration tests verify the entire IR workflow by comparing
// IR representations from two different sources:
// 1. Database inspection (pgdump.sql → database → ir/builder → IR)
// 2. SQL parsing (pgschema.sql → ir/parser → IR)
// This ensures our pgschema output accurately represents the original database schema

func TestIRIntegration_Employee(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test complete IR workflow integration for employee dataset
	runIRIntegrationTest(t, "employee")
}

func TestIRIntegration_Bytebase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test complete IR workflow integration for bytebase dataset
	runIRIntegrationTest(t, "bytebase")
}

func TestIRIntegration_Sakila(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test complete IR workflow integration for sakila dataset
	runIRIntegrationTest(t, "sakila")
}

// runIRIntegrationTest performs comprehensive IR workflow integration testing
// This function validates the complete IR workflow by comparing representations
// from database inspection and SQL parsing to ensure semantic equivalence
//
// Integration Test Flow:
// 1. Load pgdump.sql into PostgreSQL container
// 2. Build IR from database using ir/builder (database inspection)
// 3. Parse pgschema.sql into IR using ir/parser (SQL parsing)
// 4. Compare both IR representations for semantic equivalence
func runIRIntegrationTest(t *testing.T, testDataDir string) {
	ctx := context.Background()

	// Start PostgreSQL container
	containerInfo := testutil.SetupPostgresContainer(ctx, t)
	defer containerInfo.Terminate(ctx, t)

	// Get database connection
	db := containerInfo.Conn

	// FIRST IR: Load pgdump.sql and build IR from database inspection
	t.Logf("=== FIRST IR GENERATION: pgdump.sql -> database -> ir/builder -> IR ===")

	pgdumpPath := fmt.Sprintf("../../testdata/%s/pgdump.sql", testDataDir)
	pgdumpContent, err := os.ReadFile(pgdumpPath)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", pgdumpPath, err)
	}

	// Execute pgdump.sql to populate database
	_, err = db.ExecContext(ctx, string(pgdumpContent))
	if err != nil {
		t.Fatalf("Failed to execute pgdump.sql: %v", err)
	}

	// Build IR from database inspection using ir/builder
	builder := NewBuilder(db)
	dbIR, err := builder.BuildSchema(ctx, "public")
	if err != nil {
		t.Fatalf("Failed to build IR from database: %v", err)
	}

	// SECOND IR: Parse pgschema.sql directly into IR
	t.Logf("=== SECOND IR GENERATION: pgschema.sql -> ir/parser -> IR ===")

	pgschemaPath := fmt.Sprintf("../../testdata/%s/pgschema.sql", testDataDir)
	pgschemaContent, err := os.ReadFile(pgschemaPath)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", pgschemaPath, err)
	}

	// Parse pgschema.sql into IR using ir/parser
	parser := NewParser()
	parserIR, err := parser.ParseSQL(string(pgschemaContent))
	if err != nil {
		t.Fatalf("Failed to parse pgschema.sql into IR: %v", err)
	}

	// INTEGRATION VALIDATION: Compare both IR representations for semantic equivalence
	t.Logf("=== IR INTEGRATION VALIDATION ===")
	t.Logf("Database IR has %d schemas", len(dbIR.Schemas))
	t.Logf("Parser IR has %d schemas", len(parserIR.Schemas))

	// Detailed object count logging
	for schemaName, dbSchema := range dbIR.Schemas {
		parserSchema := parserIR.Schemas[schemaName]
		if parserSchema != nil {
			t.Logf("Schema '%s': DB[tables=%d, views=%d, funcs=%d, seqs=%d, indexes=%d] vs Parser[tables=%d, views=%d, funcs=%d, seqs=%d, indexes=%d]",
				schemaName,
				len(dbSchema.Tables), len(dbSchema.Views), len(dbSchema.Functions), len(dbSchema.Sequences), len(dbSchema.Indexes),
				len(parserSchema.Tables), len(parserSchema.Views), len(parserSchema.Functions), len(parserSchema.Sequences), len(parserSchema.Indexes))
		}
	}

	// Perform comprehensive IR comparison
	compareIRSemanticEquivalence(t, dbIR, parserIR)

	// Save debug output on failure
	if t.Failed() {
		saveIRDebugFiles(t, testDataDir, dbIR, parserIR)
	}

	t.Logf("=== IR INTEGRATION TEST COMPLETED ===")
}

// compareIRSemanticEquivalence performs enhanced semantic comparison between two IR representations
// This function focuses on semantic equivalence rather than exact structural matching
func compareIRSemanticEquivalence(t *testing.T, expectedIR, actualIR *Schema) {
	t.Logf("=== SEMANTIC EQUIVALENCE ANALYSIS ===")

	// Compare top-level schema counts
	if len(expectedIR.Schemas) != len(actualIR.Schemas) {
		t.Errorf("Schema count mismatch: expected %d, got %d", len(expectedIR.Schemas), len(actualIR.Schemas))
	}

	// Compare each schema for semantic equivalence
	for schemaName, expectedSchema := range expectedIR.Schemas {
		actualSchema, exists := actualIR.Schemas[schemaName]
		if !exists {
			t.Errorf("Schema %s not found in actual IR", schemaName)
			continue
		}

		t.Logf("--- Comparing schema: %s ---", schemaName)
		compareDBSchemaSemanticEquivalence(t, schemaName, expectedSchema, actualSchema)
	}

	// Check for extra schemas in actual IR
	for schemaName := range actualIR.Schemas {
		if _, exists := expectedIR.Schemas[schemaName]; !exists {
			t.Errorf("Unexpected schema %s found in actual IR", schemaName)
		}
	}

	// Compare extensions
	compareExtensions(t, expectedIR.Extensions, actualIR.Extensions)

	t.Logf("=== SEMANTIC EQUIVALENCE ANALYSIS COMPLETED ===")
}

// compareDBSchemaSemanticEquivalence compares two DBSchema objects for semantic equivalence
func compareDBSchemaSemanticEquivalence(t *testing.T, schemaName string, expected, actual *DBSchema) {
	// Compare tables (focus on BASE tables for semantic equivalence)
	expectedBaseTables := make(map[string]*Table)
	actualBaseTables := make(map[string]*Table)

	for name, table := range expected.Tables {
		if table.Type == TableTypeBase {
			expectedBaseTables[name] = table
		}
	}
	for name, table := range actual.Tables {
		if table.Type == TableTypeBase {
			actualBaseTables[name] = table
		}
	}

	if len(expectedBaseTables) != len(actualBaseTables) {
		t.Errorf("Schema %s: base table count difference: expected %d, got %d (may be due to partition table handling differences)",
			schemaName, len(expectedBaseTables), len(actualBaseTables))
	}

	// Compare each base table
	for tableName, expectedTable := range expectedBaseTables {
		actualTable, exists := actualBaseTables[tableName]
		if !exists {
			t.Errorf("Schema %s: base table %s not found in actual IR", schemaName, tableName)
			continue
		}

		compareTableSemanticEquivalence(t, schemaName, tableName, expectedTable, actualTable)
	}

	// Compare views (semantic equivalence)
	compareViewsSemanticEquivalence(t, schemaName, expected.Views, actual.Views)

	// Compare functions (semantic equivalence)
	compareFunctionsSemanticEquivalence(t, schemaName, expected.Functions, actual.Functions)

	// Compare sequences (semantic equivalence)
	compareSequencesSemanticEquivalence(t, schemaName, expected.Sequences, actual.Sequences)

	// Compare indexes (semantic equivalence)
	compareIndexesSemanticEquivalence(t, schemaName, expected.Indexes, actual.Indexes)

	// Log comparison results
	t.Logf("Schema %s semantic comparison: tables=%d/%d, views=%d/%d, functions=%d/%d, sequences=%d/%d, indexes=%d/%d",
		schemaName,
		len(actualBaseTables), len(expectedBaseTables),
		len(actual.Views), len(expected.Views),
		len(actual.Functions), len(expected.Functions),
		len(actual.Sequences), len(expected.Sequences),
		len(actual.Indexes), len(expected.Indexes))
}

// compareTableSemanticEquivalence compares two tables for semantic equivalence
func compareTableSemanticEquivalence(t *testing.T, schemaName, tableName string, expected, actual *Table) {
	// Basic properties
	if expected.Name != actual.Name {
		t.Errorf("Table %s.%s: name mismatch: expected %s, got %s",
			schemaName, tableName, expected.Name, actual.Name)
	}

	if expected.Schema != actual.Schema {
		t.Errorf("Table %s.%s: schema mismatch: expected %s, got %s",
			schemaName, tableName, expected.Schema, actual.Schema)
	}

	// Column count and semantic equivalence
	if len(expected.Columns) != len(actual.Columns) {
		t.Errorf("Table %s.%s: column count mismatch: expected %d, got %d",
			schemaName, tableName, len(expected.Columns), len(actual.Columns))
	}

	// Create maps for easier column comparison
	expectedColumns := make(map[string]*Column)
	actualColumns := make(map[string]*Column)

	for _, col := range expected.Columns {
		expectedColumns[col.Name] = col
	}
	for _, col := range actual.Columns {
		actualColumns[col.Name] = col
	}

	// Compare each column semantically
	for colName, expectedCol := range expectedColumns {
		actualCol, exists := actualColumns[colName]
		if !exists {
			t.Errorf("Table %s.%s: column %s not found in actual IR",
				schemaName, tableName, colName)
			continue
		}

		compareColumnSemanticEquivalence(t, schemaName, tableName, colName, expectedCol, actualCol)
	}

	// Log constraint differences (semantic equivalence may differ in implementation details)
	if len(expected.Constraints) != len(actual.Constraints) {
		t.Errorf("Table %s.%s: constraint count difference: expected %d, got %d",
			schemaName, tableName, len(expected.Constraints), len(actual.Constraints))
	}
}

// compareColumnSemanticEquivalence compares columns with focus on semantic equivalence
func compareColumnSemanticEquivalence(t *testing.T, schemaName, tableName, colName string, expected, actual *Column) {
	// Position should match
	if expected.Position != actual.Position {
		t.Errorf("Column %s.%s.%s: position mismatch: expected %d, got %d",
			schemaName, tableName, colName, expected.Position, actual.Position)
	}

	// Data type semantic equivalence (handle variations in type representation)
	if !areDataTypesSemanticallySame(expected.DataType, actual.DataType) {
		t.Errorf("Column %s.%s.%s: data type variation: expected %s, got %s (may be due to precision or type representation differences)",
			schemaName, tableName, colName, expected.DataType, actual.DataType)
	}

	// Nullable - be lenient as parser may not handle all ALTER TABLE constraints
	if expected.IsNullable != actual.IsNullable {
		t.Errorf("Column %s.%s.%s: nullable difference: expected %t, got %t (may be due to parsing limitations)",
			schemaName, tableName, colName, expected.IsNullable, actual.IsNullable)
	}

	// Default values - be lenient as these may have format differences
	if !areDefaultValuesSemanticallySame(expected.DefaultValue, actual.DefaultValue) {
		expectedDefault := "NULL"
		actualDefault := "NULL"
		if expected.DefaultValue != nil {
			expectedDefault = *expected.DefaultValue
		}
		if actual.DefaultValue != nil {
			actualDefault = *actual.DefaultValue
		}
		t.Errorf("Column %s.%s.%s: default value difference: expected %q, got %q (may be due to format differences)",
			schemaName, tableName, colName, expectedDefault, actualDefault)
	}
}

// areDataTypesSemanticallySame checks if two data types are semantically equivalent
func areDataTypesSemanticallySame(expected, actual string) bool {
	// Direct match
	if expected == actual {
		return true
	}

	// Handle array type variations: "ARRAY" vs "type[]"
	if expected == "ARRAY" && strings.HasSuffix(actual, "[]") {
		return true
	}
	if strings.HasSuffix(expected, "[]") && actual == "ARRAY" {
		return true
	}

	// Handle numeric precision variations: "numeric" vs "numeric(5,2)"
	if strings.HasPrefix(expected, "numeric") && strings.HasPrefix(actual, "numeric") {
		return true
	}

	// Handle character precision variations: "character" vs "character(20)"
	if strings.HasPrefix(expected, "character") && strings.HasPrefix(actual, "character") {
		return true
	}

	// Handle user-defined types: "USER-DEFINED" from database vs actual type name from parser
	if expected == "USER-DEFINED" && strings.Contains(actual, ".") {
		return true // parser shows schema-qualified type name, database shows "USER-DEFINED"
	}

	// Handle common PostgreSQL type aliases
	typeAliases := map[string][]string{
		"integer": {"int", "int4"},
		"bigint":  {"int8"},
		"text":    {"varchar"},
		"boolean": {"bool"},
	}

	for canonical, aliases := range typeAliases {
		if expected == canonical {
			for _, alias := range aliases {
				if actual == alias {
					return true
				}
			}
		}
		if actual == canonical {
			for _, alias := range aliases {
				if expected == alias {
					return true
				}
			}
		}
	}

	return false
}

// areDefaultValuesSemanticallySame checks if default values are semantically equivalent
func areDefaultValuesSemanticallySame(expected, actual *string) bool {
	// Both nil
	if expected == nil && actual == nil {
		return true
	}

	// One nil, one not
	if (expected == nil) != (actual == nil) {
		return false
	}

	// Both not nil - normalize and compare
	expectedNorm := normalizeDefaultValue(*expected)
	actualNorm := normalizeDefaultValue(*actual)

	return expectedNorm == actualNorm
}

// normalizeDefaultValue normalizes default value strings for comparison
func normalizeDefaultValue(value string) string {
	// Remove extra whitespace
	normalized := strings.TrimSpace(value)

	// Handle common PostgreSQL default variations
	// e.g., "CURRENT_TIMESTAMP" vs "now()" vs "CURRENT_TIMESTAMP()"
	normalized = strings.ReplaceAll(normalized, "CURRENT_TIMESTAMP", "now()")
	normalized = strings.ReplaceAll(normalized, "CURRENT_DATE", "now()")
	normalized = strings.ReplaceAll(normalized, "now()", "now()")

	// Handle type-cast variations: "'G'::public.mpaa_rating" vs "'G'"
	if strings.Contains(normalized, "::") {
		parts := strings.Split(normalized, "::")
		if len(parts) > 0 {
			normalized = parts[0]
		}
	}

	// Handle nextval variations: "nextval('seq'::regclass)" vs "nextval()"
	if strings.HasPrefix(normalized, "nextval(") {
		normalized = "nextval()"
	}

	return normalized
}

// compareViewsSemanticEquivalence compares views for semantic equivalence
func compareViewsSemanticEquivalence(t *testing.T, schemaName string, expected, actual map[string]*View) {
	if len(expected) != len(actual) {
		t.Errorf("Schema %s: view count difference: expected %d, got %d",
			schemaName, len(expected), len(actual))
	}

	for viewName := range expected {
		if _, exists := actual[viewName]; !exists {
			t.Errorf("Schema %s: view %s not found in actual IR", schemaName, viewName)
		}
	}
}

// compareFunctionsSemanticEquivalence compares functions for semantic equivalence
func compareFunctionsSemanticEquivalence(t *testing.T, schemaName string, expected, actual map[string]*Function) {
	if len(expected) != len(actual) {
		t.Errorf("Schema %s: function count difference: expected %d, got %d",
			schemaName, len(expected), len(actual))
	}

	for funcName := range expected {
		if _, exists := actual[funcName]; !exists {
			t.Errorf("Schema %s: function %s not found in actual IR", schemaName, funcName)
		}
	}
}

// compareSequencesSemanticEquivalence compares sequences for semantic equivalence
func compareSequencesSemanticEquivalence(t *testing.T, schemaName string, expected, actual map[string]*Sequence) {
	if len(expected) != len(actual) {
		t.Errorf("Schema %s: sequence count difference: expected %d, got %d",
			schemaName, len(expected), len(actual))
	}

	for seqName := range expected {
		if _, exists := actual[seqName]; !exists {
			t.Errorf("Schema %s: sequence %s not found in actual IR", schemaName, seqName)
		}
	}
}

// compareIndexesSemanticEquivalence compares indexes for semantic equivalence
func compareIndexesSemanticEquivalence(t *testing.T, schemaName string, expected, actual map[string]*Index) {
	if len(expected) != len(actual) {
		t.Errorf("Schema %s: index count difference: expected %d, got %d",
			schemaName, len(expected), len(actual))
	}

	for indexName, expectedIndex := range expected {
		actualIndex, exists := actual[indexName]
		if !exists {
			t.Errorf("Schema %s: index %s not found in actual IR", schemaName, indexName)
			continue
		}

		compareIndexSemanticEquivalence(t, schemaName, indexName, expectedIndex, actualIndex)
	}
}

// compareIndexSemanticEquivalence compares two indexes for semantic equivalence
func compareIndexSemanticEquivalence(t *testing.T, schemaName, indexName string, expected, actual *Index) {
	// Basic properties
	if expected.Name != actual.Name {
		t.Errorf("Index %s.%s: name mismatch: expected %s, got %s",
			schemaName, indexName, expected.Name, actual.Name)
	}

	if expected.Schema != actual.Schema {
		t.Errorf("Index %s.%s: schema mismatch: expected %s, got %s",
			schemaName, indexName, expected.Schema, actual.Schema)
	}

	if expected.Table != actual.Table {
		t.Errorf("Index %s.%s: table mismatch: expected %s, got %s",
			schemaName, indexName, expected.Table, actual.Table)
	}

	// Index type and flags
	if expected.Type != actual.Type {
		t.Errorf("Index %s.%s: type difference: expected %s, got %s (may be acceptable due to semantic differences)",
			schemaName, indexName, expected.Type, actual.Type)
	}

	if expected.IsUnique != actual.IsUnique {
		t.Errorf("Index %s.%s: unique flag mismatch: expected %t, got %t",
			schemaName, indexName, expected.IsUnique, actual.IsUnique)
	}

	if expected.IsPrimary != actual.IsPrimary {
		t.Errorf("Index %s.%s: primary flag mismatch: expected %t, got %t",
			schemaName, indexName, expected.IsPrimary, actual.IsPrimary)
	}

	if expected.IsPartial != actual.IsPartial {
		t.Errorf("Index %s.%s: partial flag difference: expected %t, got %t",
			schemaName, indexName, expected.IsPartial, actual.IsPartial)
	}

	// Index method (btree, hash, gin, etc.)
	if expected.Method != actual.Method {
		t.Errorf("Index %s.%s: method difference: expected %s, got %s",
			schemaName, indexName, expected.Method, actual.Method)
	}

	// Column count
	if len(expected.Columns) != len(actual.Columns) {
		t.Errorf("Index %s.%s: column count mismatch: expected %d, got %d",
			schemaName, indexName, len(expected.Columns), len(actual.Columns))
	}

	// Compare columns semantically
	expectedColumnsMap := make(map[int]*IndexColumn)
	actualColumnsMap := make(map[int]*IndexColumn)

	for _, col := range expected.Columns {
		expectedColumnsMap[col.Position] = col
	}
	for _, col := range actual.Columns {
		actualColumnsMap[col.Position] = col
	}

	for position, expectedCol := range expectedColumnsMap {
		actualCol, exists := actualColumnsMap[position]
		if !exists {
			t.Errorf("Index %s.%s: column at position %d not found in actual IR",
				schemaName, indexName, position)
			continue
		}

		compareIndexColumnSemanticEquivalence(t, schemaName, indexName, position, expectedCol, actualCol)
	}

	// Partial index WHERE clause - normalize for comparison
	if expected.IsPartial || actual.IsPartial {
		expectedWhere := strings.TrimSpace(expected.Where)
		actualWhere := strings.TrimSpace(actual.Where)
		if expectedWhere != actualWhere {
			t.Errorf("Index %s.%s: WHERE clause difference: expected %q, got %q (may be due to format differences)",
				schemaName, indexName, expectedWhere, actualWhere)
		}
	}

	// Definition comparison - normalize for semantic equivalence
	if !areIndexDefinitionsSemanticallySame(expected.Definition, actual.Definition) {
		t.Errorf("Index %s.%s: definition difference: expected %q, got %q (may be due to format variations)", schemaName, indexName, expected.Definition, actual.Definition)
	}
}

// compareIndexColumnSemanticEquivalence compares index columns for semantic equivalence
func compareIndexColumnSemanticEquivalence(t *testing.T, schemaName, indexName string, position int, expected, actual *IndexColumn) {
	if expected.Name != actual.Name {
		t.Errorf("Index %s.%s column at position %d: name mismatch: expected %s, got %s",
			schemaName, indexName, position, expected.Name, actual.Name)
	}

	if expected.Position != actual.Position {
		t.Errorf("Index %s.%s column %s: position mismatch: expected %d, got %d",
			schemaName, indexName, expected.Name, expected.Position, actual.Position)
	}

	// Direction and operator may have variations
	if expected.Direction != actual.Direction {
		t.Errorf("Index %s.%s column %s: direction difference: expected %s, got %s",
			schemaName, indexName, expected.Name, expected.Direction, actual.Direction)
	}

	if expected.Operator != actual.Operator {
		t.Errorf("Index %s.%s column %s: operator difference: expected %s, got %s",
			schemaName, indexName, expected.Name, expected.Operator, actual.Operator)
	}
}

// areIndexDefinitionsSemanticallySame checks if two index definitions are semantically equivalent
func areIndexDefinitionsSemanticallySame(expected, actual string) bool {
	// Direct match
	if expected == actual {
		return true
	}

	// Normalize whitespace and compare
	expectedNorm := strings.Join(strings.Fields(expected), " ")
	actualNorm := strings.Join(strings.Fields(actual), " ")

	if expectedNorm == actualNorm {
		return true
	}

	// Handle schema qualification differences
	// Expected might have explicit schema qualification while actual might not
	expectedNorm = strings.ReplaceAll(expectedNorm, "public.", "")
	actualNorm = strings.ReplaceAll(actualNorm, "public.", "")

	return expectedNorm == actualNorm
}

// compareExtensions compares extensions for semantic equivalence
func compareExtensions(t *testing.T, expected, actual map[string]*Extension) {
	if len(expected) != len(actual) {
		t.Errorf("Extension count difference: expected %d, got %d", len(expected), len(actual))
	}

	for extName := range expected {
		if _, exists := actual[extName]; !exists {
			t.Errorf("Extension %s not found in actual IR", extName)
		}
	}
}

// saveIRDebugFiles saves IR representations to files for debugging
func saveIRDebugFiles(t *testing.T, testDataDir string, dbIR, parserIR *Schema) {
	// Save database IR
	dbIRPath := fmt.Sprintf("%s_db_ir_debug.json", testDataDir)
	if dbJSON, err := json.MarshalIndent(dbIR, "", "  "); err == nil {
		if err := os.WriteFile(dbIRPath, dbJSON, 0644); err == nil {
			t.Logf("Debug: Database IR written to %s", dbIRPath)
		}
	}

	// Save parser IR
	parserIRPath := fmt.Sprintf("%s_parser_ir_debug.json", testDataDir)
	if parserJSON, err := json.MarshalIndent(parserIR, "", "  "); err == nil {
		if err := os.WriteFile(parserIRPath, parserJSON, 0644); err == nil {
			t.Logf("Debug: Parser IR written to %s", parserIRPath)
		}
	}

	t.Logf("Debug files saved for detailed IR comparison analysis")
}
