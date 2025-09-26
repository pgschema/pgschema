package include

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewProcessor(t *testing.T) {
	processor := NewProcessor("/tmp/test")
	if processor.baseDir != "/tmp/test" {
		t.Errorf("Expected baseDir to be /tmp/test, got %s", processor.baseDir)
	}
	if processor.visited == nil {
		t.Error("Expected visited map to be initialized")
	}
}

func TestProcessFile_BasicInclude(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create main file
	mainFile := filepath.Join(tempDir, "main.sql")
	mainContent := `-- Main file
CREATE TABLE users (id SERIAL PRIMARY KEY);
\i tables/orders.sql
-- End of main file`

	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	// Create subdirectory and included file
	tablesDir := filepath.Join(tempDir, "tables")
	if err := os.MkdirAll(tablesDir, 0755); err != nil {
		t.Fatalf("Failed to create tables dir: %v", err)
	}

	ordersFile := filepath.Join(tablesDir, "orders.sql")
	ordersContent := `-- Orders table
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id)
);`

	if err := os.WriteFile(ordersFile, []byte(ordersContent), 0644); err != nil {
		t.Fatalf("Failed to write orders file: %v", err)
	}

	// Process the file
	processor := NewProcessor(tempDir)
	result, err := processor.ProcessFile(mainFile)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	// Check that the include was processed
	if !strings.Contains(result, "CREATE TABLE users") {
		t.Error("Main file content not found in result")
	}
	if !strings.Contains(result, "CREATE TABLE orders") {
		t.Error("Included file content not found in result")
	}
	if !strings.Contains(result, "user_id INTEGER REFERENCES users(id)") {
		t.Error("Orders table definition not found in result")
	}

	// Check that \i directive was replaced
	if strings.Contains(result, "\\i tables/orders.sql") {
		t.Error("Include directive should have been replaced")
	}
}

func TestProcessFile_NestedInclude(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create main file
	mainFile := filepath.Join(tempDir, "main.sql")
	mainContent := `-- Main file
\i level1.sql
-- End of main file`

	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	// Create level1 file that includes level2
	level1File := filepath.Join(tempDir, "level1.sql")
	level1Content := `-- Level 1
CREATE TABLE level1_table (id SERIAL);
\i level2.sql
-- End level 1`

	if err := os.WriteFile(level1File, []byte(level1Content), 0644); err != nil {
		t.Fatalf("Failed to write level1 file: %v", err)
	}

	// Create level2 file
	level2File := filepath.Join(tempDir, "level2.sql")
	level2Content := `-- Level 2
CREATE TABLE level2_table (id SERIAL);`

	if err := os.WriteFile(level2File, []byte(level2Content), 0644); err != nil {
		t.Fatalf("Failed to write level2 file: %v", err)
	}

	// Process the file
	processor := NewProcessor(tempDir)
	result, err := processor.ProcessFile(mainFile)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	// Check that all content is included
	if !strings.Contains(result, "CREATE TABLE level1_table") {
		t.Error("Level1 content not found in result")
	}
	if !strings.Contains(result, "CREATE TABLE level2_table") {
		t.Error("Level2 content not found in result")
	}

	// Check that include directives were replaced
	if strings.Contains(result, "\\i level1.sql") || strings.Contains(result, "\\i level2.sql") {
		t.Error("Include directives should have been replaced")
	}
}

func TestProcessFile_CircularDependency(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create file1 that includes file2
	file1 := filepath.Join(tempDir, "file1.sql")
	file1Content := `-- File 1
\i file2.sql`

	if err := os.WriteFile(file1, []byte(file1Content), 0644); err != nil {
		t.Fatalf("Failed to write file1: %v", err)
	}

	// Create file2 that includes file1 (circular dependency)
	file2 := filepath.Join(tempDir, "file2.sql")
	file2Content := `-- File 2
\i file1.sql`

	if err := os.WriteFile(file2, []byte(file2Content), 0644); err != nil {
		t.Fatalf("Failed to write file2: %v", err)
	}

	// Process the file - should detect circular dependency
	processor := NewProcessor(tempDir)
	_, err := processor.ProcessFile(file1)
	if err == nil {
		t.Fatal("Expected circular dependency error")
	}
	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("Expected 'circular dependency' error, got: %v", err)
	}
}

func TestProcessFile_DirectoryTraversal(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create main file with directory traversal attempt
	mainFile := filepath.Join(tempDir, "main.sql")
	mainContent := `-- Main file
\i ../../../etc/passwd
-- End of main file`

	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	// Process the file - should reject directory traversal
	processor := NewProcessor(tempDir)
	_, err := processor.ProcessFile(mainFile)
	if err == nil {
		t.Fatal("Expected directory traversal error")
	}
	if !strings.Contains(err.Error(), "directory traversal not allowed") {
		t.Errorf("Expected 'directory traversal not allowed' error, got: %v", err)
	}
}

func TestProcessFile_FileNotFound(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create main file that includes non-existent file
	mainFile := filepath.Join(tempDir, "main.sql")
	mainContent := `-- Main file
\i nonexistent.sql
-- End of main file`

	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	// Process the file - should report file not found
	processor := NewProcessor(tempDir)
	_, err := processor.ProcessFile(mainFile)
	if err == nil {
		t.Fatal("Expected file not found error")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected 'does not exist' error, got: %v", err)
	}
}

func TestProcessFile_WithSemicolon(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create main file with semicolon after include
	mainFile := filepath.Join(tempDir, "main.sql")
	mainContent := `-- Main file
\i test.sql;
-- End of main file`

	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	// Create included file
	testFile := filepath.Join(tempDir, "test.sql")
	testContent := `CREATE TABLE test (id SERIAL);`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Process the file
	processor := NewProcessor(tempDir)
	result, err := processor.ProcessFile(mainFile)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	// Check that the include was processed
	if !strings.Contains(result, "CREATE TABLE test") {
		t.Error("Included file content not found in result")
	}
	if strings.Contains(result, "\\i test.sql;") {
		t.Error("Include directive should have been replaced")
	}
}

func TestProcessFile_NoIncludes(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create file without includes
	mainFile := filepath.Join(tempDir, "main.sql")
	mainContent := `-- Main file
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);
-- End of main file`

	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	// Process the file
	processor := NewProcessor(tempDir)
	result, err := processor.ProcessFile(mainFile)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	// Check that content is returned as-is
	if result != mainContent {
		t.Error("Content should be returned unchanged when no includes present")
	}
}

func TestProcessFile_FolderInclude(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create main file with folder include
	mainFile := filepath.Join(tempDir, "main.sql")
	mainContent := `-- Main file
\i types/
-- End of main file`

	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	// Create types directory with multiple files
	typesDir := filepath.Join(tempDir, "types")
	if err := os.MkdirAll(typesDir, 0755); err != nil {
		t.Fatalf("Failed to create types dir: %v", err)
	}

	// Create type files (should be processed in alphabetical order)
	files := map[string]string{
		"zoo.sql":    "CREATE TYPE zoo AS ENUM ('open', 'closed');",
		"animal.sql": "CREATE TYPE animal AS ENUM ('cat', 'dog');",
		"bird.sql":   "CREATE TYPE bird AS ENUM ('eagle', 'robin');",
	}

	for filename, content := range files {
		filePath := filepath.Join(typesDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", filename, err)
		}
	}

	// Process the file
	processor := NewProcessor(tempDir)
	result, err := processor.ProcessFile(mainFile)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	// Check that the folder include was processed
	if !strings.Contains(result, "CREATE TYPE animal") {
		t.Error("animal.sql content not found in result")
	}
	if !strings.Contains(result, "CREATE TYPE bird") {
		t.Error("bird.sql content not found in result")
	}
	if !strings.Contains(result, "CREATE TYPE zoo") {
		t.Error("zoo.sql content not found in result")
	}

	// Check that files are processed in alphabetical order
	animalIdx := strings.Index(result, "CREATE TYPE animal")
	birdIdx := strings.Index(result, "CREATE TYPE bird")
	zooIdx := strings.Index(result, "CREATE TYPE zoo")

	if animalIdx == -1 || birdIdx == -1 || zooIdx == -1 {
		t.Fatal("Not all type definitions found")
	}

	if !(animalIdx < birdIdx && birdIdx < zooIdx) {
		t.Error("Files not processed in alphabetical order")
		t.Logf("Order found: animal=%d, bird=%d, zoo=%d", animalIdx, birdIdx, zooIdx)
	}

	// Check that folder include directive was replaced
	if strings.Contains(result, "\\i types/") {
		t.Error("Folder include directive should have been replaced")
	}
}

func TestProcessFile_NestedFolderInclude(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create main file
	mainFile := filepath.Join(tempDir, "main.sql")
	mainContent := `-- Main file
\i schema/
-- End of main file`

	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	// Create nested directory structure
	schemaDir := filepath.Join(tempDir, "schema")
	typesDir := filepath.Join(schemaDir, "types")
	tablesDir := filepath.Join(schemaDir, "tables")

	for _, dir := range []string{schemaDir, typesDir, tablesDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	// Create files in different directories
	files := map[string]string{
		filepath.Join(schemaDir, "main.sql"):         "-- Schema main file",
		filepath.Join(typesDir, "user_type.sql"):    "CREATE TYPE user_type AS ENUM ('admin', 'user');",
		filepath.Join(tablesDir, "users.sql"):       "CREATE TABLE users (id SERIAL);",
		filepath.Join(typesDir, "status_type.sql"):  "CREATE TYPE status_type AS ENUM ('active', 'inactive');",
	}

	for filePath, content := range files {
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", filePath, err)
		}
	}

	// Process the file
	processor := NewProcessor(tempDir)
	result, err := processor.ProcessFile(mainFile)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	// Check that all content is included using DFS order
	expected := []string{
		"-- Schema main file",
		"CREATE TABLE users", // tables/ comes before types/ alphabetically
		"CREATE TYPE status_type", // status_type.sql comes before user_type.sql
		"CREATE TYPE user_type",
	}

	lastIndex := -1
	for _, expectedContent := range expected {
		index := strings.Index(result, expectedContent)
		if index == -1 {
			t.Errorf("Expected content not found: %s", expectedContent)
			continue
		}
		if index < lastIndex {
			t.Errorf("Content out of expected order: %s at position %d, previous was at %d",
				expectedContent, index, lastIndex)
		}
		lastIndex = index
	}
}

func TestProcessFile_FolderNotFound(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create main file that includes non-existent folder
	mainFile := filepath.Join(tempDir, "main.sql")
	mainContent := `-- Main file
\i nonexistent/
-- End of main file`

	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	// Process the file - should report folder not found
	processor := NewProcessor(tempDir)
	_, err := processor.ProcessFile(mainFile)
	if err == nil {
		t.Fatal("Expected folder not found error")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected 'does not exist' error, got: %v", err)
	}
}

func TestProcessFile_EmptyFolder(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create main file with folder include
	mainFile := filepath.Join(tempDir, "main.sql")
	mainContent := `-- Main file
\i empty/
-- End of main file`

	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	// Create empty directory
	emptyDir := filepath.Join(tempDir, "empty")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("Failed to create empty dir: %v", err)
	}

	// Process the file
	processor := NewProcessor(tempDir)
	result, err := processor.ProcessFile(mainFile)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	// Check that only main file content remains
	expected := `-- Main file
-- End of main file`
	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestProcessFile_ExpectedFileButFoundFolder(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create main file trying to include a folder as file
	mainFile := filepath.Join(tempDir, "main.sql")
	mainContent := `-- Main file
\i somefolder
-- End of main file`

	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	// Create a directory with the same name
	folderPath := filepath.Join(tempDir, "somefolder")
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		t.Fatalf("Failed to create folder: %v", err)
	}

	// Process the file - should report type mismatch
	processor := NewProcessor(tempDir)
	_, err := processor.ProcessFile(mainFile)
	if err == nil {
		t.Fatal("Expected type mismatch error")
	}
	if !strings.Contains(err.Error(), "expected file but found folder") {
		t.Errorf("Expected 'expected file but found folder' error, got: %v", err)
	}
}

func TestProcessFile_ExpectedFolderButFoundFile(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create main file trying to include a file as folder
	mainFile := filepath.Join(tempDir, "main.sql")
	mainContent := `-- Main file
\i somefile.sql/
-- End of main file`

	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	// Create a file with that name
	filePath := filepath.Join(tempDir, "somefile.sql")
	if err := os.WriteFile(filePath, []byte("CREATE TABLE test (id SERIAL);"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Process the file - should report type mismatch
	processor := NewProcessor(tempDir)
	_, err := processor.ProcessFile(mainFile)
	if err == nil {
		t.Fatal("Expected type mismatch error")
	}
	if !strings.Contains(err.Error(), "expected folder but found file") {
		t.Errorf("Expected 'expected folder but found file' error, got: %v", err)
	}
}

func TestProcessFile_MixedFilesAndFoldersInFolder(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create main file
	mainFile := filepath.Join(tempDir, "main.sql")
	mainContent := `-- Main file
\i mixed/
-- End of main file`

	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main file: %v", err)
	}

	// Create mixed directory with files and subdirectories
	mixedDir := filepath.Join(tempDir, "mixed")
	subDir := filepath.Join(mixedDir, "subdir")

	for _, dir := range []string{mixedDir, subDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	// Create files and non-SQL files
	files := map[string]string{
		filepath.Join(mixedDir, "a_file.sql"):      "-- A file content",
		filepath.Join(mixedDir, "b_file.txt"):      "This should be ignored",
		filepath.Join(mixedDir, "z_file.sql"):      "-- Z file content",
		filepath.Join(subDir, "sub_file.sql"):      "-- Sub file content",
	}

	for filePath, content := range files {
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", filePath, err)
		}
	}

	// Process the file
	processor := NewProcessor(tempDir)
	result, err := processor.ProcessFile(mainFile)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	// Check that SQL files are included in proper order
	if !strings.Contains(result, "-- A file content") {
		t.Error("a_file.sql content not found")
	}
	if !strings.Contains(result, "-- Sub file content") {
		t.Error("sub_file.sql content not found")
	}
	if !strings.Contains(result, "-- Z file content") {
		t.Error("z_file.sql content not found")
	}

	// Check that non-SQL files are ignored
	if strings.Contains(result, "This should be ignored") {
		t.Error("Non-SQL file content should be ignored")
	}

	// Check alphabetical order: a_file.sql, subdir/sub_file.sql, z_file.sql
	aIdx := strings.Index(result, "-- A file content")
	subIdx := strings.Index(result, "-- Sub file content")
	zIdx := strings.Index(result, "-- Z file content")

	if !(aIdx < subIdx && subIdx < zIdx) {
		t.Error("Files not processed in expected alphabetical order")
		t.Logf("Order found: a_file=%d, sub_file=%d, z_file=%d", aIdx, subIdx, zIdx)
	}
}

func TestProcessFile_MatchesPreGeneratedSchema(t *testing.T) {
	// Test that processing main.sql produces the same output as schema.sql
	// This ensures the include processor works correctly with real test data

	processor := NewProcessor("../../testdata/include")

	// Process the main.sql file with all includes
	processedContent, err := processor.ProcessFile("../../testdata/include/main.sql")
	if err != nil {
		t.Fatalf("Failed to process main.sql: %v", err)
	}

	// Read the pre-generated schema.sql file
	expectedContent, err := os.ReadFile("../../testdata/include/expected_full_schema.sql")
	if err != nil {
		t.Fatalf("Failed to read schema.sql: %v", err)
	}

	// Compare the contents - they should match exactly
	if processedContent != string(expectedContent) {
		t.Errorf("Processed content does not match schema.sql")
		t.Logf("Expected length: %d", len(expectedContent))
		t.Logf("Actual length: %d", len(processedContent))

		// Find the first difference for debugging
		expectedLines := strings.Split(string(expectedContent), "\n")
		actualLines := strings.Split(processedContent, "\n")

		maxLines := len(expectedLines)
		if len(actualLines) > maxLines {
			maxLines = len(actualLines)
		}

		for i := 0; i < maxLines; i++ {
			expectedLine := ""
			actualLine := ""

			if i < len(expectedLines) {
				expectedLine = expectedLines[i]
			}
			if i < len(actualLines) {
				actualLine = actualLines[i]
			}

			if expectedLine != actualLine {
				t.Logf("First difference at line %d:", i+1)
				t.Logf("Expected: %q", expectedLine)
				t.Logf("Actual:   %q", actualLine)
				break
			}
		}
	}
}
