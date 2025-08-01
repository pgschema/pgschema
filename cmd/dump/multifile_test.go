package dump

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/ir"
)

func TestCreateMultiFileOutput(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "schema.sql")

	// Create test SQLCollector with some steps
	collector := diff.NewSQLCollector()
	
	// Create mock SQL contexts and add them to collector
	contexts := []*diff.SQLContext{
		{
			ObjectType:   "type",
			Operation:    "create", 
			ObjectPath:   "public.user_status",
			SourceChange: nil,
		},
		{
			ObjectType:   "table",
			Operation:    "create",
			ObjectPath:   "public.users", 
			SourceChange: nil,
		},
		{
			ObjectType:   "function",
			Operation:    "create",
			ObjectPath:   "public.get_user_count",
			SourceChange: nil,
		},
		{
			ObjectType:   "view",
			Operation:    "create",
			ObjectPath:   "public.active_users",
			SourceChange: nil,
		},
	}
	
	sqls := []string{
		"CREATE TYPE user_status AS ENUM ('active', 'inactive');",
		"CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);",
		"CREATE FUNCTION get_user_count() RETURNS integer AS $$ SELECT COUNT(*) FROM users; $$;",
		"CREATE VIEW active_users AS SELECT * FROM users WHERE status = 'active';",
	}
	
	for i, context := range contexts {
		collector.Collect(context, sqls[i])
	}

	// Create test IR with metadata
	testIR := ir.NewIR()
	testIR.Metadata.DatabaseVersion = "PostgreSQL 15.0"

	// Test the createMultiFileOutput function
	err := createMultiFileOutput(collector, testIR, "public", outputPath)
	if err != nil {
		t.Fatalf("createMultiFileOutput failed: %v", err)
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
		"functions/get_user_count.sql",
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
		"\\i functions/get_user_count.sql", 
		"\\i tables/users.sql",
		"\\i views/active_users.sql",
	}

	for _, include := range expectedIncludes {
		if !strings.Contains(mainStr, include) {
			t.Errorf("Main file should contain include statement: %s\nMain file content:\n%s", include, mainStr)
		}
	}

	// Check that header is present
	if !strings.Contains(mainStr, "-- pgschema database dump") {
		t.Errorf("Main file should contain header")
	}

	// Check individual file content
	typeFile := filepath.Join(tmpDir, "types", "user_status.sql")
	typeContent, err := os.ReadFile(typeFile)
	if err != nil {
		t.Fatalf("Failed to read type file: %v", err)
	}

	typeStr := string(typeContent)
	if !strings.Contains(typeStr, "CREATE TYPE user_status") {
		t.Errorf("Type file should contain the CREATE TYPE statement")
	}
	if !strings.Contains(typeStr, "-- Name: user_status; Type: TYPE; Schema: -; Owner: -") {
		t.Errorf("Type file should contain comment header")
	}
}

func TestGetObjectDirectory(t *testing.T) {
	tests := []struct {
		objectType string
		expected   string
	}{
		{"TYPE", "types"},
		{"DOMAIN", "domains"},
		{"SEQUENCE", "sequences"},
		{"FUNCTION", "functions"},
		{"PROCEDURE", "procedures"},
		{"TABLE", "tables"},
		{"VIEW", "views"},
		{"MATERIALIZED VIEW", "views"},
		{"TRIGGER", "tables"},
		{"INDEX", "tables"},
		{"CONSTRAINT", "tables"},
		{"POLICY", "tables"}, 
		{"UNKNOWN", "misc"},
	}

	for _, test := range tests {
		result := getObjectDirectory(test.objectType)
		if result != test.expected {
			t.Errorf("getObjectDirectory(%q) = %q, expected %q", test.objectType, result, test.expected)
		}
	}
}

func TestGetObjectName(t *testing.T) {
	tests := []struct {
		objectPath string
		expected   string
	}{
		{"public.users", "users"},
		{"schema1.table1", "table1"},
		{"simple_name", "simple_name"},
		{"", ""},
		{"a.b.c", "b"}, // Should take the second part
	}

	for _, test := range tests {
		result := getObjectName(test.objectPath)
		if result != test.expected {
			t.Errorf("getObjectName(%q) = %q, expected %q", test.objectPath, result, test.expected)
		}
	}
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