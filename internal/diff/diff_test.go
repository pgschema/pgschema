package diff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pgschema/pgschema/internal/ir"
)

// parseSQL is a helper function to convert SQL string to IR for tests
func parseSQL(t *testing.T, sql string) *ir.IR {
	parser := ir.NewParser()
	schema, err := parser.ParseSQL(sql)
	if err != nil {
		t.Fatalf("Failed to parse SQL: %v", err)
	}
	return schema
}

// TestDiffFromFiles runs file-based diff tests from testdata directory
//
// Test filtering can be controlled using the PGSCHEMA_TEST_FILTER environment variable:
//
// Examples:
//
//	# Run all tests under alter_table/ (directory prefix with slash)
//	PGSCHEMA_TEST_FILTER="alter_table/" go test -v ./internal/diff
//
//	# Run tests under alter_table/ that start with "add_column"
//	PGSCHEMA_TEST_FILTER="alter_table/add_column" go test -v ./internal/diff
//
//	# Run a specific test
//	PGSCHEMA_TEST_FILTER="alter_table/add_column_with_fk" go test -v ./internal/diff
func TestDiffFromFiles(t *testing.T) {
	testdataDir := filepath.Join("../../testdata/diff")

	// Check if testdata directory exists
	if _, err := os.Stat(testdataDir); os.IsNotExist(err) {
		t.Skip("testdata directory does not exist, skipping file-based tests")
		return
	}

	// Get test filter from environment variable
	testFilter := os.Getenv("PGSCHEMA_TEST_FILTER")

	// Walk through all statement type directories (e.g., create_table, alter_table)
	err := filepath.Walk(testdataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root testdata directory and statement type directories
		if path == testdataDir || strings.Count(path, string(os.PathSeparator)) <= strings.Count(testdataDir, string(os.PathSeparator))+1 {
			return nil
		}

		// Only process directories that contain test cases
		if !info.IsDir() {
			return nil
		}

		// Check if this directory contains the required test files
		oldFile := filepath.Join(path, "old.sql")
		newFile := filepath.Join(path, "new.sql")
		migrationFile := filepath.Join(path, "migration.sql")

		// Extract test name from path
		relPath, _ := filepath.Rel(testdataDir, path)
		testName := strings.ReplaceAll(relPath, string(os.PathSeparator), "_")

		// Apply test filter if provided
		if testFilter != "" && !matchesFilter(relPath, testFilter) {
			return nil
		}

		// Run the test case as a subtest
		t.Run(testName, func(t *testing.T) {
			runFileBasedDiffTest(t, oldFile, newFile, migrationFile)
		})

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to walk testdata directory: %v", err)
	}
}

// runFileBasedDiffTest executes a single file-based diff test
func runFileBasedDiffTest(t *testing.T, oldFile, newFile, migrationFile string) {
	// Read old DDL
	oldDDL, err := os.ReadFile(oldFile)
	if err != nil {
		t.Fatalf("Failed to read old.sql: %v", err)
	}

	// Read new DDL
	newDDL, err := os.ReadFile(newFile)
	if err != nil {
		t.Fatalf("Failed to read new.sql: %v", err)
	}

	// Read expected migration
	expectedMigration, err := os.ReadFile(migrationFile)
	if err != nil {
		t.Fatalf("Failed to read migration.sql: %v", err)
	}

	// Parse DDL to IR
	oldIR := parseSQL(t, string(oldDDL))
	newIR := parseSQL(t, string(newDDL))

	// Run diff
	diff := Diff(oldIR, newIR)

	// Generate migration SQL
	actualMigration := GenerateMigrationSQL(diff, "public")

	// Normalize whitespace for comparison
	expected := normalizeSQL(string(expectedMigration))
	actual := normalizeSQL(actualMigration)

	if actual != expected {
		t.Errorf("Migration SQL mismatch:\nExpected:\n%s\n\nActual:\n%s", expected, actual)
	}
}

// matchesFilter checks if a test should run based on the filter pattern
// It supports:
// - Directory prefix with slash: "alter_table/" (matches all tests under alter_table/)
// - Specific test pattern: "alter_table/add_column_with_fk" (matches specific test under alter_table/)
// - Prefix pattern: "alter_table/add_column" (matches tests under alter_table/ that start with add_column)
func matchesFilter(relPath, filter string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}

	// Handle directory prefix patterns with trailing slash
	if strings.HasSuffix(filter, "/") {
		// "alter_table/" matches "alter_table/add_column_with_fk"
		return strings.HasPrefix(relPath+"/", filter)
	}

	// Handle patterns with slash (both specific tests and prefix patterns)
	if strings.Contains(filter, "/") {
		// "alter_table/add_column_with_fk" matches "alter_table/add_column_with_fk"
		// "alter_table/add_column" matches "alter_table/add_column_with_fk"
		return strings.HasPrefix(relPath, filter)
	}

	// Fallback: check if filter is a substring of the path
	return strings.Contains(relPath, filter)
}

// normalizeSQL normalizes SQL for comparison by trimming whitespace and removing empty lines
func normalizeSQL(sql string) string {
	lines := strings.Split(sql, "\n")
	var normalizedLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			normalizedLines = append(normalizedLines, trimmed)
		}
	}

	return strings.Join(normalizedLines, "\n")
}
