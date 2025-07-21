package ir

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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

	pgdumpPath := fmt.Sprintf("../../testdata/dump/%s/pgdump.sql", testDataDir)
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

	pgschemaPath := fmt.Sprintf("../../testdata/dump/%s/pgschema.sql", testDataDir)
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

	// Perform comprehensive IR comparison
	dbInput := IRComparisonInput{
		IR:          dbIR,
		Description: "Database IR (pgdump.sql → database → ir/inspector → IR)",
	}
	parserInput := IRComparisonInput{
		IR:          parserIR,
		Description: "Parser IR (pgschema.sql → ir/parser → IR)",
	}

	CompareIRSemanticEquivalence(t, dbInput, parserInput)

	// Save debug output on failure
	if t.Failed() {
		saveIRDebugFiles(t, testDataDir, dbInput, parserInput)
	}

	t.Logf("=== IR INTEGRATION TEST COMPLETED ===")
}

// SaveIRDebugFiles saves IR representations to files for debugging
func saveIRDebugFiles(t *testing.T, testDataDir string, input1, input2 IRComparisonInput) {
	// Save first IR
	ir1Path := fmt.Sprintf("%s_ir1_debug.json", testDataDir)
	if ir1JSON, err := json.MarshalIndent(input1.IR, "", "  "); err == nil {
		if err := os.WriteFile(ir1Path, ir1JSON, 0644); err == nil {
			t.Logf("Debug: First IR (%s) written to %s", input1.Description, ir1Path)
		}
	}

	// Save second IR
	ir2Path := fmt.Sprintf("%s_ir2_debug.json", testDataDir)
	if ir2JSON, err := json.MarshalIndent(input2.IR, "", "  "); err == nil {
		if err := os.WriteFile(ir2Path, ir2JSON, 0644); err == nil {
			t.Logf("Debug: Second IR (%s) written to %s", input2.Description, ir2Path)
		}
	}

	t.Logf("Debug files saved for detailed IR comparison analysis")
}
