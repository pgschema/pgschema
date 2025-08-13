package diff

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pgschema/pgschema/internal/ir"
	"github.com/pgschema/pgschema/testutil"
)

// TestDiffInspectorAndParser tests the complete IR (Intermediate Representation) workflow
// using test cases from testdata/diff/.
//
// The test performs these validations for each test case:
// 1. Apply old.sql to database → inspect to get oldIR
// 2. Parse new.sql → get newIR
// 3. Diff oldIR and newIR → generate migration SQL
// 4. Compare generated migration SQL with expected plan.sql (exact match validation)
// 5. Apply the migration SQL to the database
// 6. Inspect database again to get finalIR
// 7. Compare finalIR with newIR (validates IR round-trip correctness)
// 8. Generate migration from finalIR to newIR (should be empty - validates true idempotency)
//
// Test filtering can be controlled using the PGSCHEMA_TEST_FILTER environment variable:
//
// Examples:
//
//	# Run all tests under create_table/ (directory prefix with slash)
//	PGSCHEMA_TEST_FILTER="create_table/" go test -v ./internal/diff -run TestDiffInspectorAndParser
//
//	# Run tests under create_table/ that start with "add_column"
//	PGSCHEMA_TEST_FILTER="create_table/add_column" go test -v ./internal/diff -run TestDiffInspectorAndParser
//
//	# Run a specific test
//	PGSCHEMA_TEST_FILTER="create_table/add_table" go test -v ./internal/diff -run TestDiffInspectorAndParser
func TestDiffInspectorAndParser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDataRoot := "../../testdata/diff"

	// Get test filter from environment variable
	testFilter := os.Getenv("PGSCHEMA_TEST_FILTER")

	// Walk through all test case directories
	err := filepath.Walk(testDataRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory or if it's the root
		if !info.IsDir() || path == testDataRoot {
			return nil
		}

		// Check if this directory contains the required files
		oldFile := filepath.Join(path, "old.sql")
		newFile := filepath.Join(path, "new.sql")
		planFile := filepath.Join(path, "plan.sql")

		// Skip directories that don't contain the required test files
		if _, err := os.Stat(oldFile); os.IsNotExist(err) {
			return nil
		}
		if _, err := os.Stat(newFile); os.IsNotExist(err) {
			return nil
		}
		if _, err := os.Stat(planFile); os.IsNotExist(err) {
			return nil
		}

		// Get relative path for test name
		relPath, _ := filepath.Rel(testDataRoot, path)
		testName := strings.ReplaceAll(relPath, string(filepath.Separator), "_")

		// Apply test filter if provided
		if testFilter != "" && !matchesFilter(relPath, testFilter) {
			return nil
		}

		t.Run(testName, func(t *testing.T) {
			runDiffIntegrationTest(t, oldFile, newFile, planFile)
		})

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to walk test data directory: %v", err)
	}
}

// runDiffIntegrationTest executes a single diff integration test case.
// See TestDiffInspectorAndParser for complete documentation of the validation steps.
func runDiffIntegrationTest(t *testing.T, oldFile, newFile, planFile string) {
	ctx := context.Background()

	// Start PostgreSQL container
	containerInfo := testutil.SetupPostgresContainer(ctx, t)
	defer containerInfo.Terminate(ctx, t)

	// Get database connection
	db := containerInfo.Conn

	t.Logf("=== DIFF INTEGRATION TEST: %s → %s ===", filepath.Base(oldFile), filepath.Base(newFile))

	// STEP 1: Apply old.sql to database and build oldIR from inspection
	t.Logf("--- Applying old.sql and building oldIR via database inspection ---")

	oldContent, err := os.ReadFile(oldFile)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", oldFile, err)
	}

	// Execute old.sql to populate database
	if len(strings.TrimSpace(string(oldContent))) > 0 {
		_, err = db.ExecContext(ctx, string(oldContent))
		if err != nil {
			t.Fatalf("Failed to execute old.sql: %v", err)
		}
	}

	// Build oldIR from database inspection
	inspector := ir.NewInspector(db)
	oldIR, err := inspector.BuildIR(ctx, "public")
	if err != nil {
		t.Fatalf("Failed to build oldIR from database: %v", err)
	}

	// STEP 2: Parse new.sql into newIR
	t.Logf("--- Parsing new.sql into newIR ---")

	newContent, err := os.ReadFile(newFile)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", newFile, err)
	}

	// Parse new.sql into newIR
	parser := ir.NewParser()
	newIR, err := parser.ParseSQL(string(newContent))
	if err != nil {
		t.Fatalf("Failed to parse new.sql into IR: %v", err)
	}

	// STEP 3: Generate migration SQL from IR diff
	t.Logf("--- Generating migration SQL from IR diff ---")

	diffs := GenerateMigration(oldIR, newIR, "public")
	actualMigrationSQL := buildSQLFromSteps(diffs)

	// STEP 4: Compare with expected plan.sql
	t.Logf("--- Comparing generated migration SQL with expected ---")

	expectedContent, err := os.ReadFile(planFile)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", planFile, err)
	}

	expectedMigrationSQL := strings.TrimSpace(string(expectedContent))
	actualMigrationSQL = strings.TrimSpace(actualMigrationSQL)

	if expectedMigrationSQL != actualMigrationSQL {
		t.Errorf("Migration SQL mismatch\nExpected:\n%s\n\nActual:\n%s", expectedMigrationSQL, actualMigrationSQL)

		// Save debug files on failure
		saveIRDiffDebugFiles(t, filepath.Base(oldFile), oldIR, newIR, expectedMigrationSQL, actualMigrationSQL)
	}

	// STEP 5: Apply the migration SQL to the database
	t.Logf("--- Applying migration SQL to database ---")

	if len(strings.TrimSpace(actualMigrationSQL)) > 0 {
		_, err = db.ExecContext(ctx, actualMigrationSQL)
		if err != nil {
			t.Fatalf("Failed to apply migration SQL: %v", err)
		}
	}

	// STEP 6: Inspect database again to get finalIR
	t.Logf("--- Inspecting database after migration to get finalIR ---")

	finalIR, err := inspector.BuildIR(ctx, "public")
	if err != nil {
		t.Fatalf("Failed to build finalIR from database after migration: %v", err)
	}

	// STEP 7: Compare finalIR with newIR - they should be identical
	t.Logf("--- Comparing finalIR with newIR ---")

	if !compareIRs(t, newIR, finalIR) {
		t.Errorf("Final IR after migration does not match expected new IR")
		saveIRComparisonDebugFiles(t, filepath.Base(oldFile), newIR, finalIR)
	}

	// STEP 8: Generate migration from finalIR to newIR - should be empty
	t.Logf("--- Generating migration from finalIR to newIR (should be empty) ---")

	finalDiffs := GenerateMigration(finalIR, newIR, "public")
	finalMigrationSQL := buildSQLFromSteps(finalDiffs)
	finalMigrationSQL = strings.TrimSpace(finalMigrationSQL)

	if finalMigrationSQL != "" {
		t.Errorf("Expected empty migration after applying changes, but got:\n%s", finalMigrationSQL)
		t.Logf("This indicates that the round-trip (parse -> diff -> apply -> inspect) is not idempotent.")
		t.Logf("Possible causes:")
		t.Logf("1. The inspector doesn't capture all database state correctly")
		t.Logf("2. The parser doesn't generate DDL that matches PostgreSQL's normalization")
		t.Logf("3. The diff algorithm is missing some edge cases")
	}

	t.Logf("=== DIFF INTEGRATION TEST COMPLETED ===")
}

// saveIRDiffDebugFiles saves IR representations and migration SQL for debugging
func saveIRDiffDebugFiles(t *testing.T, testName string, oldIR, newIR *ir.IR, expectedSQL, actualSQL string) {
	// Save oldIR
	if oldIRJson, err := json.MarshalIndent(oldIR, "", "  "); err == nil {
		oldIRPath := fmt.Sprintf("%s_oldIR_debug.json", testName)
		if err := os.WriteFile(oldIRPath, oldIRJson, 0644); err == nil {
			t.Logf("Debug: oldIR written to %s", oldIRPath)
		}
	}

	// Save newIR
	if newIRJson, err := json.MarshalIndent(newIR, "", "  "); err == nil {
		newIRPath := fmt.Sprintf("%s_newIR_debug.json", testName)
		if err := os.WriteFile(newIRPath, newIRJson, 0644); err == nil {
			t.Logf("Debug: newIR written to %s", newIRPath)
		}
	}

	// Save expected migration SQL
	expectedPath := fmt.Sprintf("%s_expected_migration_debug.sql", testName)
	if err := os.WriteFile(expectedPath, []byte(expectedSQL), 0644); err == nil {
		t.Logf("Debug: expected migration SQL written to %s", expectedPath)
	}

	// Save actual migration SQL
	actualPath := fmt.Sprintf("%s_actual_migration_debug.sql", testName)
	if err := os.WriteFile(actualPath, []byte(actualSQL), 0644); err == nil {
		t.Logf("Debug: actual migration SQL written to %s", actualPath)
	}

	t.Logf("Debug files saved for detailed analysis")
}

// compareIRs compares two IR structures for equality
// This comparison is semantic rather than literal, as parser and inspector
// produce different default values for metadata fields
func compareIRs(t *testing.T, expected, actual *ir.IR) bool {
	// Instead of comparing the full IR, generate a diff and check if it's empty
	// This is the most accurate way to verify semantic equivalence
	diffs := GenerateMigration(expected, actual, "public")
	migrationSQL := buildSQLFromSteps(diffs)
	migrationSQL = strings.TrimSpace(migrationSQL)

	if migrationSQL != "" {
		t.Logf("IR comparison failed - migration SQL needed:")
		t.Logf("%s", migrationSQL)

		// Also log the JSON for debugging
		expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
		actualJSON, _ := json.MarshalIndent(actual, "", "  ")
		t.Logf("Expected IR:\n%s", string(expectedJSON))
		t.Logf("Actual IR:\n%s", string(actualJSON))

		return false
	}

	return true
}

// saveIRComparisonDebugFiles saves IR representations for debugging comparison failures
func saveIRComparisonDebugFiles(t *testing.T, testName string, expectedIR, actualIR *ir.IR) {
	// Save expected IR
	if expectedIRJson, err := json.MarshalIndent(expectedIR, "", "  "); err == nil {
		expectedPath := fmt.Sprintf("%s_expected_IR_debug.json", testName)
		if err := os.WriteFile(expectedPath, expectedIRJson, 0644); err == nil {
			t.Logf("Debug: expected IR written to %s", expectedPath)
		}
	}

	// Save actual IR
	if actualIRJson, err := json.MarshalIndent(actualIR, "", "  "); err == nil {
		actualPath := fmt.Sprintf("%s_actual_IR_debug.json", testName)
		if err := os.WriteFile(actualPath, actualIRJson, 0644); err == nil {
			t.Logf("Debug: actual IR written to %s", actualPath)
		}
	}

	t.Logf("IR comparison debug files saved for detailed analysis")
}
