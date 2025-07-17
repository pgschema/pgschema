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

// TestDiffInspectorAndParser tests IR integration using internal/diff/testdata test cases
// This test validates the complete workflow:
// 1. Apply old.sql to database → inspect to get oldIR
// 2. Parse new.sql → get newIR
// 3. Diff oldIR and newIR → generate migration SQL
// 4. Compare generated migration SQL with expected migration.sql
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

	testDataRoot := "testdata"

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
		migrationFile := filepath.Join(path, "migration.sql")

		// Get relative path for test name
		relPath, _ := filepath.Rel(testDataRoot, path)
		testName := strings.ReplaceAll(relPath, string(filepath.Separator), "_")

		// Apply test filter if provided
		if testFilter != "" && !matchesFilter(relPath, testFilter) {
			return nil
		}

		t.Run(testName, func(t *testing.T) {
			runDiffIntegrationTest(t, oldFile, newFile, migrationFile)
		})

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to walk test data directory: %v", err)
	}
}

// runDiffIntegrationTest performs IR integration testing using diff test case data
// This validates the complete IR workflow:
// 1. Apply old.sql to database → inspect to get oldIR
// 2. Parse new.sql → get newIR
// 3. Diff oldIR and newIR → generate migration SQL
// 4. Compare generated migration SQL with expected migration.sql
func runDiffIntegrationTest(t *testing.T, oldFile, newFile, migrationFile string) {
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

	ddlDiff := Diff(oldIR, newIR)
	actualMigrationSQL := GenerateMigrationSQL(ddlDiff, "public")

	// STEP 4: Compare with expected migration.sql
	t.Logf("--- Comparing generated migration SQL with expected ---")

	expectedContent, err := os.ReadFile(migrationFile)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", migrationFile, err)
	}

	expectedMigrationSQL := strings.TrimSpace(string(expectedContent))
	actualMigrationSQL = strings.TrimSpace(actualMigrationSQL)

	if expectedMigrationSQL != actualMigrationSQL {
		t.Errorf("Migration SQL mismatch\nExpected:\n%s\n\nActual:\n%s", expectedMigrationSQL, actualMigrationSQL)

		// Save debug files on failure
		saveIRDiffDebugFiles(t, filepath.Base(oldFile), oldIR, newIR, expectedMigrationSQL, actualMigrationSQL)
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
