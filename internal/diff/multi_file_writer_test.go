package diff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMultiFileWriter_Basic(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "schema.sql")
	
	// Create multi-file writer
	writer, err := NewMultiFileWriter(outputPath, true)
	if err != nil {
		t.Fatalf("Failed to create MultiFileWriter: %v", err)
	}
	
	// Test writing different object types
	writer.WriteStatementWithComment("TYPE", "user_status", "public", "postgres", 
		"CREATE TYPE user_status AS ENUM ('active', 'inactive');", "public")
	
	writer.WriteStatementWithComment("FUNCTION", "get_user_status", "public", "postgres",
		"CREATE FUNCTION get_user_status(user_id INT) RETURNS user_status AS $$ SELECT 'active'::user_status; $$;", "public")
	
	writer.WriteStatementWithComment("TABLE", "users", "public", "postgres",
		"CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);", "public")
	
	writer.WriteStatementWithComment("VIEW", "active_users", "public", "postgres",
		"CREATE VIEW active_users AS SELECT * FROM users;", "public")
	
	// Finalize the writer
	result := writer.String()
	
	// Check that the result is empty (no completion message)
	if result != "" {
		t.Errorf("Expected empty result from String(), got: %s", result)
	}
	
	// Check that main file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Main file was not created at %s", outputPath)
	}
	
	// Check that subdirectories were created
	expectedDirs := []string{"types", "functions", "tables", "views"}
	for _, dir := range expectedDirs {
		dirPath := filepath.Join(tmpDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("Expected directory %s was not created", dirPath)
		}
	}
	
	// Check that individual files were created
	expectedFiles := []string{
		"types/user_status.sql",
		"functions/get_user_status.sql", 
		"tables/users.sql",
		"views/active_users.sql",
	}
	for _, file := range expectedFiles {
		filePath := filepath.Join(tmpDir, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", filePath)
		}
	}
	
	// Read main file and check for include statements
	mainContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read main file: %v", err)
	}
	
	mainStr := string(mainContent)
	expectedIncludes := []string{
		"\\i types/user_status.sql",
		"\\i functions/get_user_status.sql",
		"\\i tables/users.sql", 
		"\\i views/active_users.sql",
	}
	
	for _, include := range expectedIncludes {
		if !strings.Contains(mainStr, include) {
			t.Errorf("Main file should contain include statement: %s", include)
		}
	}
}

func TestMultiFileWriter_HeaderPlacement(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "schema.sql")
	
	// Create multi-file writer
	writer, err := NewMultiFileWriter(outputPath, true)
	if err != nil {
		t.Fatalf("Failed to create MultiFileWriter: %v", err)
	}
	
	// Write header using WriteHeader method (simulating dump command behavior)
	testHeader := "--\n-- Test header\n--\n\n"
	writer.WriteHeader(testHeader)
	
	// Write some content
	writer.WriteStatementWithComment("TABLE", "test_table", "public", "postgres",
		"CREATE TABLE test_table (id SERIAL PRIMARY KEY);", "public")
	
	// Finalize
	_ = writer.String()
	
	// Read main file and verify header is at the beginning
	mainContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read main file: %v", err)
	}
	
	mainStr := string(mainContent)
	if !strings.HasPrefix(mainStr, testHeader) {
		t.Errorf("Main file should start with header. Got:\n%s", mainStr)
	}
}

func TestMultiFileWriter_NewlineHandling(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "schema.sql")
	
	// Create multi-file writer
	writer, err := NewMultiFileWriter(outputPath, true)
	if err != nil {
		t.Fatalf("Failed to create MultiFileWriter: %v", err)
	}
	
	// Write header
	testHeader := "--\n-- Test header\n--\n\n"
	writer.WriteHeader(testHeader)
	
	// Write a table statement
	writer.WriteStatementWithComment("TABLE", "test_table", "public", "postgres",
		"CREATE TABLE test_table (id SERIAL PRIMARY KEY);", "public")
	
	// Finalize
	_ = writer.String()
	
	// Check main file doesn't have extra newlines
	mainContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read main file: %v", err)
	}
	
	mainStr := string(mainContent)
	// Should not end with multiple newlines
	if strings.HasSuffix(mainStr, "\n\n\n") {
		t.Errorf("Main file should not end with multiple newlines. Content:\n%q", mainStr)
	}
	
	// Check individual table file doesn't have extra newlines
	tableFile := filepath.Join(tmpDir, "tables", "test_table.sql")
	tableContent, err := os.ReadFile(tableFile)
	if err != nil {
		t.Fatalf("Failed to read table file: %v", err)
	}
	
	tableStr := string(tableContent)
	// Should NOT end with any trailing newlines (like SingleFileWriter)
	if strings.HasSuffix(tableStr, "\n") {
		t.Errorf("Table file should not end with trailing newlines. Content:\n%q", tableStr)
	}
	
	// Should not have extra newlines between content
	lines := strings.Split(strings.TrimRight(tableStr, "\n"), "\n")
	for i, line := range lines {
		if line == "" && i > 0 && i < len(lines)-1 {
			// Empty lines in the middle might indicate extra newlines
			if lines[i-1] == "" || lines[i+1] == "" {
				t.Errorf("Table file seems to have consecutive empty lines around line %d", i+1)
			}
		}
	}
}

func TestMultiFileWriter_TrailingNewlines(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "main.sql")
	
	// Create multi-file writer
	writer, err := NewMultiFileWriter(outputPath, true)
	if err != nil {
		t.Fatalf("Failed to create MultiFileWriter: %v", err)
	}
	
	// Write header (simulating actual dump command)
	testHeader := "--\n-- pgschema database dump\n--\n\n-- Dumped from database version PostgreSQL 15.0\n-- Dumped by pgschema version test\n\n\n"
	writer.WriteHeader(testHeader)
	
	// Write some test content
	writer.WriteStatementWithComment("TYPE", "user_status", "public", "postgres",
		"CREATE TYPE user_status AS ENUM ('active', 'inactive');", "public")
	
	writer.WriteStatementWithComment("TABLE", "users", "public", "postgres",
		"CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);", "public")
	
	// Finalize
	result := writer.String()
	t.Logf("Result: %q", result)
	
	// Check main file for trailing newlines
	mainContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read main file: %v", err)
	}
	
	// Count trailing newlines in main file
	trailingNewlines := 0
	for i := len(mainContent) - 1; i >= 0 && mainContent[i] == '\n'; i-- {
		trailingNewlines++
	}
	t.Logf("Main file content: %q", string(mainContent))
	t.Logf("Main file trailing newlines: %d", trailingNewlines)
	
	// Check individual files
	// Check table file
	tableFile := filepath.Join(tmpDir, "tables", "users.sql")
	tableContent, err := os.ReadFile(tableFile)
	if err != nil {
		t.Fatalf("Failed to read table file: %v", err)
	}
	
	trailingNewlines = 0
	for i := len(tableContent) - 1; i >= 0 && tableContent[i] == '\n'; i-- {
		trailingNewlines++
	}
	t.Logf("Table file content: %q", string(tableContent))
	t.Logf("Table file trailing newlines: %d", trailingNewlines)
	
	// Check type file
	typeFile := filepath.Join(tmpDir, "types", "user_status.sql")
	typeContent, err := os.ReadFile(typeFile)
	if err != nil {
		t.Fatalf("Failed to read type file: %v", err)
	}
	
	trailingNewlines = 0
	for i := len(typeContent) - 1; i >= 0 && typeContent[i] == '\n'; i-- {
		trailingNewlines++
	}
	t.Logf("Type file content: %q", string(typeContent))
	t.Logf("Type file trailing newlines: %d", trailingNewlines)
}

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple_name", "simple_name"},
		{"name-with-dashes", "name-with-dashes"},
		{"name.with.dots", "name_with_dots"},
		{"name with spaces", "name_with_spaces"},
		{"name@#$%symbols", "name____symbols"},
		{"_leading_underscore_", "leading_underscore"},
		{"MixedCase", "mixedcase"},
	}
	
	for _, test := range tests {
		result := sanitizeFileName(test.input)
		if result != test.expected {
			t.Errorf("sanitizeFileName(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}