package diff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDiffFromFiles runs file-based diff tests from testdata directory
func TestDiffFromFiles(t *testing.T) {
	testdataDir := filepath.Join("testdata")

	// Check if testdata directory exists
	if _, err := os.Stat(testdataDir); os.IsNotExist(err) {
		t.Skip("testdata directory does not exist, skipping file-based tests")
		return
	}

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

		if !fileExists(oldFile) || !fileExists(newFile) || !fileExists(migrationFile) {
			return nil // Skip incomplete test cases
		}

		// Extract test name from path
		relPath, _ := filepath.Rel(testdataDir, path)
		testName := strings.ReplaceAll(relPath, string(os.PathSeparator), "_")

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

	// Run diff
	diff, err := Diff(string(oldDDL), string(newDDL))
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	// Generate migration SQL
	actualMigration := diff.GenerateMigrationSQL()

	// Normalize whitespace for comparison
	expected := normalizeSQL(string(expectedMigration))
	actual := normalizeSQL(actualMigration)

	if actual != expected {
		t.Errorf("Migration SQL mismatch:\nExpected:\n%s\n\nActual:\n%s", expected, actual)
	}
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
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
