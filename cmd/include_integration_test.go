package cmd

// Include Integration Tests
// These integration tests verify the multi-file dump functionality by testing
// the complete workflow from schema loading through modular file generation.
// The test uses testdata/include/ which contains a modular schema structure
// with \i include statements, then verifies that multi-file dump can recreate
// the same organized file structure.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pgschema/pgschema/cmd/dump"
	"github.com/pgschema/pgschema/testutil"
	"github.com/spf13/cobra"
)

func TestIncludeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Setup PostgreSQL container with specific database
	containerInfo := testutil.SetupPostgresContainerWithDB(ctx, t, "testdb", "testuser", "testpass")
	defer containerInfo.Terminate(ctx, t)

	// Load the include-based schema using original method
	// Note: Using original method since CLI apply has issues with complex schemas with triggers/policies
	loadIncludeSchema(t, ctx, containerInfo)

	// Create temporary directory for multi-file output
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "main.sql")

	// Execute multi-file dump using CLI command
	executeMultiFileDump(t, containerInfo, outputPath)

	// Compare dumped files with original include files
	compareIncludeFiles(t, tmpDir)
}

// loadIncludeSchema loads the testdata/include/main.sql schema into the database
func loadIncludeSchema(t *testing.T, ctx context.Context, containerInfo *testutil.ContainerInfo) {
	// Read the main.sql file which contains \i include statements
	mainSQLPath := "../testdata/include/main.sql"
	mainSQLContent, err := os.ReadFile(mainSQLPath)
	if err != nil {
		t.Fatalf("Failed to read main SQL file %s: %v", mainSQLPath, err)
	}

	// We need to process the includes manually since PostgreSQL container
	// won't have access to our local filesystem for \i commands
	processedSQL := processIncludeStatements(t, string(mainSQLContent), "../testdata/include")

	// Execute the processed SQL to create the schema
	_, err = containerInfo.Conn.ExecContext(ctx, processedSQL)
	if err != nil {
		t.Fatalf("Failed to execute include schema: %v", err)
	}

	t.Logf("✓ Successfully loaded include-based schema into database")
}

// processIncludeStatements recursively processes \i include statements
func processIncludeStatements(t *testing.T, sqlContent string, baseDir string) string {
	lines := strings.Split(sqlContent, "\n")
	var processedLines []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check if this is an include statement
		if strings.HasPrefix(trimmedLine, "\\i ") {
			// Extract the file path
			includePath := strings.TrimPrefix(trimmedLine, "\\i ")
			includePath = strings.TrimSpace(includePath)

			// Build full path
			fullPath := filepath.Join(baseDir, includePath)

			// Read the included file
			includeContent, err := os.ReadFile(fullPath)
			if err != nil {
				t.Fatalf("Failed to read include file %s: %v", fullPath, err)
			}

			// Add a comment indicating the source file
			processedLines = append(processedLines, fmt.Sprintf("-- Content from %s", includePath))

			// Recursively process includes in the included file
			processedInclude := processIncludeStatements(t, string(includeContent), baseDir)
			processedLines = append(processedLines, processedInclude)

		} else {
			// Regular SQL line, keep as-is
			processedLines = append(processedLines, line)
		}
	}

	return strings.Join(processedLines, "\n")
}

// executeMultiFileDump runs pgschema dump --multi-file using the CLI command
func executeMultiFileDump(t *testing.T, containerInfo *testutil.ContainerInfo, outputPath string) {
	// Create a new root command with dump as subcommand
	rootCmd := &cobra.Command{
		Use: "pgschema",
	}

	// Add the dump command as a subcommand
	rootCmd.AddCommand(dump.DumpCmd)

	// Set command arguments for dump
	args := []string{
		"dump",
		"--host", containerInfo.Host,
		"--port", fmt.Sprintf("%d", containerInfo.Port),
		"--db", "testdb",
		"--user", "testuser",
		"--password", "testpass",
		"--schema", "public",
		"--multi-file",
		"--file", outputPath,
	}
	rootCmd.SetArgs(args)

	// Execute the root command with dump subcommand
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to execute multi-file dump using pgschema dump: %v", err)
	}

	t.Logf("✓ Successfully executed multi-file dump using pgschema dump to %s", filepath.Dir(outputPath))
}

// compareIncludeFiles compares dumped files with original include files using direct comparison
func compareIncludeFiles(t *testing.T, dumpDir string) {
	sourceDir := "../testdata/include"

	// Compare the entire directory structure and contents
	compareDirectoryLayout(t, sourceDir, dumpDir)

	t.Logf("✓ Include file comparison completed")
}

// compareDirectoryLayout compares the complete directory layout between source and dump
func compareDirectoryLayout(t *testing.T, sourceDir, dumpDir string) {
	// First, check that all expected directories exist in dump
	sourceEntries, err := os.ReadDir(sourceDir)
	if err != nil {
		t.Fatalf("Failed to read source directory %s: %v", sourceDir, err)
	}

	// Get all dump directories
	dumpEntries, err := os.ReadDir(dumpDir)
	if err != nil {
		t.Fatalf("Failed to read dump directory %s: %v", dumpDir, err)
	}

	// Create maps for easy comparison
	sourceSubdirs := make(map[string]bool)
	dumpSubdirs := make(map[string]bool)

	// Collect source subdirectories (skip files like main.sql, schema.sql)
	for _, entry := range sourceEntries {
		if entry.IsDir() {
			sourceSubdirs[entry.Name()] = true
		}
	}

	// Collect dump subdirectories
	for _, entry := range dumpEntries {
		if entry.IsDir() {
			dumpSubdirs[entry.Name()] = true
		}
	}

	// Check for missing directories in dump
	for dirName := range sourceSubdirs {
		if !dumpSubdirs[dirName] {
			t.Errorf("Missing directory in dump: %s", dirName)
			continue
		}

		t.Logf("✓ Directory exists: %s", dirName)

		// Compare the contents of this directory
		sourceDirPath := filepath.Join(sourceDir, dirName)
		dumpDirPath := filepath.Join(dumpDir, dirName)
		compareDirectoryContents(t, sourceDirPath, dumpDirPath, dirName)
	}

	// Check for extra directories in dump
	for dirName := range dumpSubdirs {
		if !sourceSubdirs[dirName] {
			t.Errorf("Unexpected extra directory in dump: %s", dirName)
		}
	}
}

// compareDirectoryContents compares files within a specific directory
func compareDirectoryContents(t *testing.T, sourceDir, dumpDir, dirName string) {
	// Read source directory files
	sourceEntries, err := os.ReadDir(sourceDir)
	if err != nil {
		t.Errorf("Failed to read source directory %s: %v", sourceDir, err)
		return
	}

	// Read dump directory files
	dumpEntries, err := os.ReadDir(dumpDir)
	if err != nil {
		t.Errorf("Failed to read dump directory %s: %v", dumpDir, err)
		return
	}

	// Create maps for comparison
	sourceFiles := make(map[string]bool)
	dumpFiles := make(map[string]bool)

	// Collect source files
	for _, entry := range sourceEntries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sourceFiles[entry.Name()] = true
		}
	}

	// Collect dump files
	for _, entry := range dumpEntries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			dumpFiles[entry.Name()] = true
		}
	}

	// All directories now have 1:1 file mapping after reorganization
	compareDirectory(t, sourceDir, dumpDir, dirName, sourceFiles, dumpFiles)
}

// compareDirectory compares directories with 1:1 file mapping
func compareDirectory(t *testing.T, sourceDir, dumpDir, dirName string, sourceFiles, dumpFiles map[string]bool) {
	// Check for missing files in dump
	for fileName := range sourceFiles {
		if !dumpFiles[fileName] {
			t.Errorf("Missing file in dump %s/: %s", dirName, fileName)
		} else {
			// Compare file contents
			compareFileContents(t, filepath.Join(sourceDir, fileName), filepath.Join(dumpDir, fileName), dirName+"/"+fileName)
			delete(dumpFiles, fileName) // Remove from map to track extras
		}
	}

	// Check for extra files in dump
	for fileName := range dumpFiles {
		t.Errorf("Unexpected extra file in dump %s/: %s", dirName, fileName)
	}
}

// compareFileContents compares two individual files
func compareFileContents(t *testing.T, sourceFilePath, dumpFilePath, displayName string) {
	sourceContent, err := os.ReadFile(sourceFilePath)
	if err != nil {
		t.Errorf("Failed to read source file %s: %v", sourceFilePath, err)
		return
	}

	dumpContent, err := os.ReadFile(dumpFilePath)
	if err != nil {
		t.Errorf("Failed to read dump file %s: %v", dumpFilePath, err)
		return
	}

	if string(sourceContent) == string(dumpContent) {
		t.Logf("✓ Content match for %s", displayName)
	} else {
		t.Errorf("Content mismatch for %s", displayName)
		t.Logf("\n\nExpected:\n%s\n\n", string(sourceContent))
		t.Logf("\n\nActual:\n%s\n\n", string(dumpContent))
	}
}
