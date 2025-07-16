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
// 1. Database inspection (pgdump.sql → database → ir/inspector → IR)
// 2. SQL parsing (pgschema.sql → ir/parser → IR)
// This ensures our pgschema output accurately represents the original database schema
//
// Test Workflow:
//   pgdump.sql → Database → [INSPECTOR] → IR
//                                          ↓
//                                 Semantic Equivalence?
//                                          ↑
//                pgschema.sql → [PARSER] → IR
//
// Both paths should produce semantically equivalent IR representations

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
// 2. Build IR from database using ir/inspector (database inspection)
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
	t.Logf("=== FIRST IR GENERATION: pgdump.sql -> database -> ir/inspector -> IR ===")

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

	// Build IR from database inspection using ir/inspector
	inspector := NewInspector(db)
	dbIR, err := inspector.BuildIR(ctx, "public")
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
			dbIndexCount := countTableLevelIndexes(dbSchema)
			parserIndexCount := countTableLevelIndexes(parserSchema)
			t.Logf("Schema '%s': DB[tables=%d, views=%d, funcs=%d, seqs=%d, indexes=%d] vs Parser[tables=%d, views=%d, funcs=%d, seqs=%d, indexes=%d]",
				schemaName,
				len(dbSchema.Tables), len(dbSchema.Views), len(dbSchema.Functions), len(dbSchema.Sequences), dbIndexCount,
				len(parserSchema.Tables), len(parserSchema.Views), len(parserSchema.Functions), len(parserSchema.Sequences), parserIndexCount)
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
// inspectorIR comes from database inspection, parserIR comes from SQL parsing
func compareIRSemanticEquivalence(t *testing.T, inspectorIR, parserIR *IR) {
	t.Logf("=== SEMANTIC EQUIVALENCE ANALYSIS ===")

	// Compare top-level schema counts
	if len(inspectorIR.Schemas) != len(parserIR.Schemas) {
		t.Errorf("Schema count mismatch: inspector %d, parser %d", len(inspectorIR.Schemas), len(parserIR.Schemas))
	}

	// Compare each schema for semantic equivalence
	for schemaName, inspectorSchema := range inspectorIR.Schemas {
		parserSchema, exists := parserIR.Schemas[schemaName]
		if !exists {
			t.Errorf("Schema %s not found in parser IR", schemaName)
			continue
		}

		t.Logf("--- Comparing schema: %s ---", schemaName)
		compareDBSchemaSemanticEquivalence(t, schemaName, inspectorSchema, parserSchema)
	}

	// Check for extra schemas in parser IR
	for schemaName := range parserIR.Schemas {
		if _, exists := inspectorIR.Schemas[schemaName]; !exists {
			t.Errorf("Unexpected schema %s found in parser IR", schemaName)
		}
	}

	// Compare extensions
	compareExtensions(t, inspectorIR.Extensions, parserIR.Extensions)

	t.Logf("=== SEMANTIC EQUIVALENCE ANALYSIS COMPLETED ===")
}

// compareDBSchemaSemanticEquivalence compares two DBSchema objects for semantic equivalence
func compareDBSchemaSemanticEquivalence(t *testing.T, schemaName string, inspector, parser *Schema) {
	// Compare tables (focus on BASE tables for semantic equivalence)
	inspectorBaseTables := make(map[string]*Table)
	parserBaseTables := make(map[string]*Table)

	for name, table := range inspector.Tables {
		if table.Type == TableTypeBase {
			inspectorBaseTables[name] = table
		}
	}
	for name, table := range parser.Tables {
		if table.Type == TableTypeBase {
			parserBaseTables[name] = table
		}
	}

	if len(inspectorBaseTables) != len(parserBaseTables) {
		t.Errorf("Schema %s: base table count difference: inspector %d, parser %d (may be due to partition table handling differences)",
			schemaName, len(inspectorBaseTables), len(parserBaseTables))
	}

	// Compare each base table
	for tableName, inspectorTable := range inspectorBaseTables {
		parserTable, exists := parserBaseTables[tableName]
		if !exists {
			t.Errorf("Schema %s: base table %s not found in parser IR", schemaName, tableName)
			continue
		}

		compareTableSemanticEquivalence(t, schemaName, tableName, inspectorTable, parserTable)
	}

	// Compare views (semantic equivalence)
	compareViewsSemanticEquivalence(t, schemaName, inspector.Views, parser.Views)

	// Compare functions (semantic equivalence)
	compareFunctionsSemanticEquivalence(t, schemaName, inspector.Functions, parser.Functions)

	// Compare sequences (semantic equivalence)
	compareSequencesSemanticEquivalence(t, schemaName, inspector.Sequences, parser.Sequences)

	// Compare indexes at table level (semantic equivalence)
	compareTableLevelIndexesSemanticEquivalence(t, schemaName, inspector, parser)

	// Log comparison results with table-level index counts
	inspectorIndexCount := countTableLevelIndexes(inspector)
	parserIndexCount := countTableLevelIndexes(parser)
	t.Logf("Schema %s semantic comparison: tables=%d/%d, views=%d/%d, functions=%d/%d, sequences=%d/%d, indexes=%d/%d",
		schemaName,
		len(parserBaseTables), len(inspectorBaseTables),
		len(parser.Views), len(inspector.Views),
		len(parser.Functions), len(inspector.Functions),
		len(parser.Sequences), len(inspector.Sequences),
		parserIndexCount, inspectorIndexCount)
}

// compareTableSemanticEquivalence compares two tables for semantic equivalence
func compareTableSemanticEquivalence(t *testing.T, schemaName, tableName string, inspector, parser *Table) {
	// Basic properties
	if inspector.Name != parser.Name {
		t.Errorf("Table %s.%s: name mismatch: inspector %s, parser %s",
			schemaName, tableName, inspector.Name, parser.Name)
	}

	if inspector.Schema != parser.Schema {
		t.Errorf("Table %s.%s: schema mismatch: inspector %s, parser %s",
			schemaName, tableName, inspector.Schema, parser.Schema)
	}

	// Column count and semantic equivalence
	if len(inspector.Columns) != len(parser.Columns) {
		t.Errorf("Table %s.%s: column count mismatch: inspector %d, parser %d",
			schemaName, tableName, len(inspector.Columns), len(parser.Columns))
	}

	// Create maps for easier column comparison
	inspectorColumns := make(map[string]*Column)
	parserColumns := make(map[string]*Column)

	for _, col := range inspector.Columns {
		inspectorColumns[col.Name] = col
	}
	for _, col := range parser.Columns {
		parserColumns[col.Name] = col
	}

	// Compare each column semantically
	for colName, inspectorCol := range inspectorColumns {
		parserCol, exists := parserColumns[colName]
		if !exists {
			t.Errorf("Table %s.%s: column %s not found in parser IR",
				schemaName, tableName, colName)
			continue
		}

		compareColumnSemanticEquivalence(t, schemaName, tableName, colName, inspectorCol, parserCol)
	}

	// Log constraint differences (semantic equivalence may differ in implementation details)
	if len(inspector.Constraints) != len(parser.Constraints) {
		t.Errorf("Table %s.%s: constraint count difference: inspector %d, parser %d",
			schemaName, tableName, len(inspector.Constraints), len(parser.Constraints))
	}
}

// compareColumnSemanticEquivalence compares columns with focus on semantic equivalence
func compareColumnSemanticEquivalence(t *testing.T, schemaName, tableName, colName string, inspector, parser *Column) {
	// Position should match
	if inspector.Position != parser.Position {
		t.Errorf("Column %s.%s.%s: position mismatch: inspector %d, parser %d",
			schemaName, tableName, colName, inspector.Position, parser.Position)
	}

	// Data type should match exactly now that we've fixed type mapping
	if inspector.DataType != parser.DataType {
		t.Errorf("Column %s.%s.%s: data type mismatch: inspector %s, parser %s",
			schemaName, tableName, colName, inspector.DataType, parser.DataType)
	}

	// Nullable - be lenient as parser may not handle all ALTER TABLE constraints
	if inspector.IsNullable != parser.IsNullable {
		t.Errorf("Column %s.%s.%s: nullable difference: inspector %t, parser %t (may be due to parsing limitations)",
			schemaName, tableName, colName, inspector.IsNullable, parser.IsNullable)
	}

	// Default values - strict comparison
	if !areDefaultValuesEqual(inspector.DefaultValue, parser.DefaultValue) {
		inspectorDefault := "NULL"
		parserDefault := "NULL"
		if inspector.DefaultValue != nil {
			inspectorDefault = *inspector.DefaultValue
		}
		if parser.DefaultValue != nil {
			parserDefault = *parser.DefaultValue
		}
		t.Errorf("Column %s.%s.%s: default value mismatch: inspector %q, parser %q",
			schemaName, tableName, colName, inspectorDefault, parserDefault)
	}
}

// areDefaultValuesEqual checks if default values are exactly equal
func areDefaultValuesEqual(inspector, parser *string) bool {
	// Both nil
	if inspector == nil && parser == nil {
		return true
	}

	// One nil, one not
	if (inspector == nil) != (parser == nil) {
		return false
	}

	// Both not nil - strict string comparison
	return *inspector == *parser
}

// compareViewsSemanticEquivalence compares views for semantic equivalence
func compareViewsSemanticEquivalence(t *testing.T, schemaName string, inspector, parser map[string]*View) {
	if len(inspector) != len(parser) {
		t.Errorf("Schema %s: view count difference: inspector %d, parser %d",
			schemaName, len(inspector), len(parser))
	}

	for viewName := range inspector {
		if _, exists := parser[viewName]; !exists {
			t.Errorf("Schema %s: view %s not found in parser IR", schemaName, viewName)
		}
	}
}

// compareFunctionsSemanticEquivalence compares functions for semantic equivalence
func compareFunctionsSemanticEquivalence(t *testing.T, schemaName string, inspector, parser map[string]*Function) {
	if len(inspector) != len(parser) {
		t.Errorf("Schema %s: function count difference: inspector %d, parser %d",
			schemaName, len(inspector), len(parser))
	}

	for funcName := range inspector {
		if _, exists := parser[funcName]; !exists {
			t.Errorf("Schema %s: function %s not found in parser IR", schemaName, funcName)
		}
	}
}

// compareSequencesSemanticEquivalence compares sequences for semantic equivalence
func compareSequencesSemanticEquivalence(t *testing.T, schemaName string, inspector, parser map[string]*Sequence) {
	if len(inspector) != len(parser) {
		t.Errorf("Schema %s: sequence count difference: inspector %d, parser %d",
			schemaName, len(inspector), len(parser))
	}

	for seqName := range inspector {
		if _, exists := parser[seqName]; !exists {
			t.Errorf("Schema %s: sequence %s not found in parser IR", schemaName, seqName)
		}
	}
}

// compareExtensions compares extensions for semantic equivalence
func compareExtensions(t *testing.T, inspector, parser map[string]*Extension) {
	if len(inspector) != len(parser) {
		t.Errorf("Extension count difference: inspector %d, parser %d", len(inspector), len(parser))
	}

	for extName := range inspector {
		if _, exists := parser[extName]; !exists {
			t.Errorf("Extension %s not found in parser IR", extName)
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
func compareTableLevelIndexesSemanticEquivalence(t *testing.T, schemaName string, inspector, parser *Schema) {
	// Collect all indexes from tables in inspector schema
	inspectorIndexes := make(map[string]*Index)
	for tableName, table := range inspector.Tables {
		for indexName, index := range table.Indexes {
			// Use table.index format as key to ensure uniqueness across tables
			key := fmt.Sprintf("%s.%s", tableName, indexName)
			inspectorIndexes[key] = index
		}
	}

	// Collect all indexes from tables in parser schema
	parserIndexes := make(map[string]*Index)
	for tableName, table := range parser.Tables {
		for indexName, index := range table.Indexes {
			// Use table.index format as key to ensure uniqueness across tables
			key := fmt.Sprintf("%s.%s", tableName, indexName)
			parserIndexes[key] = index
		}
	}

	// Compare index counts
	if len(inspectorIndexes) != len(parserIndexes) {
		t.Errorf("Schema %s: table-level index count difference: inspector %d, parser %d",
			schemaName, len(inspectorIndexes), len(parserIndexes))
	}

	// Compare each index
	for indexKey, inspectorIndex := range inspectorIndexes {
		parserIndex, exists := parserIndexes[indexKey]
		if !exists {
			t.Errorf("Schema %s: table-level index %s not found in parser IR", schemaName, indexKey)
			continue
		}

		compareIndexSemanticEquivalence(t, schemaName, indexKey, inspectorIndex, parserIndex)
	}

	// Check for extra indexes in parser
	for indexKey := range parserIndexes {
		if _, exists := inspectorIndexes[indexKey]; !exists {
			t.Errorf("Schema %s: unexpected table-level index %s found in parser IR", schemaName, indexKey)
		}
	}
}

// compareIndexSemanticEquivalence compares two indexes for semantic equivalence
func compareIndexSemanticEquivalence(t *testing.T, schemaName, indexName string, inspector, parser *Index) {
	// Basic properties
	if inspector.Name != parser.Name {
		t.Errorf("Index %s.%s: name mismatch: inspector %s, parser %s",
			schemaName, indexName, inspector.Name, parser.Name)
	}

	if inspector.Schema != parser.Schema {
		t.Errorf("Index %s.%s: schema mismatch: inspector %s, parser %s",
			schemaName, indexName, inspector.Schema, parser.Schema)
	}

	if inspector.Table != parser.Table {
		t.Errorf("Index %s.%s: table mismatch: inspector %s, parser %s",
			schemaName, indexName, inspector.Table, parser.Table)
	}

	// Index type and flags
	if inspector.Type != parser.Type {
		t.Errorf("Index %s.%s: type difference: inspector %s, parser %s (may be acceptable due to semantic differences)",
			schemaName, indexName, inspector.Type, parser.Type)
	}

	inspectorIsUnique := inspector.Type == IndexTypeUnique || inspector.Type == IndexTypePrimary
	parserIsUnique := parser.Type == IndexTypeUnique || parser.Type == IndexTypePrimary
	if inspectorIsUnique != parserIsUnique {
		t.Errorf("Index %s.%s: unique flag mismatch: inspector %t, parser %t",
			schemaName, indexName, inspectorIsUnique, parserIsUnique)
	}

	if inspector.Type == IndexTypePrimary != (parser.Type == IndexTypePrimary) {
		t.Errorf("Index %s.%s: primary flag mismatch: inspector %t, parser %t",
			schemaName, indexName, inspector.Type == IndexTypePrimary, parser.Type == IndexTypePrimary)
	}

	if inspector.IsPartial != parser.IsPartial {
		t.Errorf("Index %s.%s: partial flag difference: inspector %t, parser %t",
			schemaName, indexName, inspector.IsPartial, parser.IsPartial)
	}

	// Index method (btree, hash, gin, etc.)
	if inspector.Method != parser.Method {
		t.Errorf("Index %s.%s: method difference: inspector %s, parser %s",
			schemaName, indexName, inspector.Method, parser.Method)
	}

	// Column count
	if len(inspector.Columns) != len(parser.Columns) {
		t.Errorf("Index %s.%s: column count mismatch: inspector %d, parser %d",
			schemaName, indexName, len(inspector.Columns), len(parser.Columns))
	}

	// Compare columns semantically
	inspectorColumnsMap := make(map[int]*IndexColumn)
	parserColumnsMap := make(map[int]*IndexColumn)

	for _, col := range inspector.Columns {
		inspectorColumnsMap[col.Position] = col
	}
	for _, col := range parser.Columns {
		parserColumnsMap[col.Position] = col
	}

	for position, inspectorCol := range inspectorColumnsMap {
		parserCol, exists := parserColumnsMap[position]
		if !exists {
			t.Errorf("Index %s.%s: column at position %d not found in parser IR",
				schemaName, indexName, position)
			continue
		}

		compareIndexColumnSemanticEquivalence(t, schemaName, indexName, position, inspectorCol, parserCol)
	}

	// Partial index WHERE clause - normalize for comparison
	if inspector.IsPartial || parser.IsPartial {
		inspectorWhere := strings.TrimSpace(inspector.Where)
		parserWhere := strings.TrimSpace(parser.Where)
		if inspectorWhere != parserWhere {
			t.Errorf("Index %s.%s: WHERE clause difference: inspector %q, parser %q (may be due to format differences)",
				schemaName, indexName, inspectorWhere, parserWhere)
		}
	}
}

// compareIndexColumnSemanticEquivalence compares index columns for semantic equivalence
func compareIndexColumnSemanticEquivalence(t *testing.T, schemaName, indexName string, position int, inspector, parser *IndexColumn) {
	if inspector.Name != parser.Name {
		t.Errorf("Index %s.%s column at position %d: name mismatch: inspector %s, parser %s",
			schemaName, indexName, position, inspector.Name, parser.Name)
	}

	if inspector.Position != parser.Position {
		t.Errorf("Index %s.%s column %s: position mismatch: inspector %d, parser %d",
			schemaName, indexName, inspector.Name, inspector.Position, parser.Position)
	}

	// Direction and operator may have variations
	if inspector.Direction != parser.Direction {
		t.Errorf("Index %s.%s column %s: direction difference: inspector %s, parser %s",
			schemaName, indexName, inspector.Name, inspector.Direction, parser.Direction)
	}

	if inspector.Operator != parser.Operator {
		t.Errorf("Index %s.%s column %s: operator difference: inspector %s, parser %s",
			schemaName, indexName, inspector.Name, inspector.Operator, parser.Operator)
	}
}

// saveIRDebugFiles saves IR representations to files for debugging
func saveIRDebugFiles(t *testing.T, testDataDir string, dbIR, parserIR *IR) {
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
