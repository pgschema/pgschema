package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlanCommand(t *testing.T) {
	// Test that the command is properly configured
	if PlanCmd.Use != "plan" {
		t.Errorf("Expected Use to be 'plan', got '%s'", PlanCmd.Use)
	}

	if PlanCmd.Short == "" {
		t.Error("Expected Short description to be set")
	}

	if PlanCmd.Long == "" {
		t.Error("Expected Long description to be set")
	}

	// Test that required flags are defined
	flags := PlanCmd.Flags()

	// Test database connection flags
	db1Flag := flags.Lookup("db1")
	if db1Flag == nil {
		t.Error("Expected --db1 flag to be defined")
	}

	user1Flag := flags.Lookup("user1")
	if user1Flag == nil {
		t.Error("Expected --user1 flag to be defined")
	}

	// Test file flags
	file1Flag := flags.Lookup("file1")
	if file1Flag == nil {
		t.Error("Expected --file1 flag to be defined")
	}

	file2Flag := flags.Lookup("file2")
	if file2Flag == nil {
		t.Error("Expected --file2 flag to be defined")
	}

	// Test schema filter flags
	schema1Flag := flags.Lookup("schema1")
	if schema1Flag == nil {
		t.Error("Expected --schema1 flag to be defined")
	}

	schema2Flag := flags.Lookup("schema2")
	if schema2Flag == nil {
		t.Error("Expected --schema2 flag to be defined")
	}

	formatFlag := flags.Lookup("format")
	if formatFlag == nil {
		t.Error("Expected --format flag to be defined")
	}
	if formatFlag.DefValue != "text" {
		t.Errorf("Expected default format to be 'text', got '%s'", formatFlag.DefValue)
	}
}

func TestPlanCommandExecution(t *testing.T) {
	// Create temporary test files
	tmpDir := t.TempDir()

	schema1Path := filepath.Join(tmpDir, "schema1.sql")
	schema2Path := filepath.Join(tmpDir, "schema2.sql")

	// Write test SQL content
	schema1SQL := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);`

	schema2SQL := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT
);

CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    user_id INTEGER REFERENCES users(id)
);`

	if err := os.WriteFile(schema1Path, []byte(schema1SQL), 0644); err != nil {
		t.Fatalf("Failed to write schema1 file: %v", err)
	}

	if err := os.WriteFile(schema2Path, []byte(schema2SQL), 0644); err != nil {
		t.Fatalf("Failed to write schema2 file: %v", err)
	}

	// Test successful execution
	tests := []struct {
		name   string
		format string
	}{
		{"text format", "text"},
		{"json format", "json"},
		{"preview format", "preview"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			file1 = schema1Path
			file2 = schema2Path
			format = tt.format

			err := runPlan(PlanCmd, []string{})
			if err != nil {
				t.Errorf("Expected plan command to succeed with %s format, but got error: %v", tt.format, err)
			}
		})
	}
}

func TestPlanValidation(t *testing.T) {
	// Save original values
	originalFile1 := file1
	originalFile2 := file2
	originalDb1 := db1
	originalUser1 := user1
	originalDb2 := db2
	originalUser2 := user2
	
	// Restore original values at the end
	defer func() {
		file1 = originalFile1
		file2 = originalFile2
		db1 = originalDb1
		user1 = originalUser1
		db2 = originalDb2
		user2 = originalUser2
	}()

	tests := []struct {
		name        string
		file1       string
		file2       string
		db1         string
		user1       string
		db2         string
		user2       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "no sources specified",
			expectError: true,
			errorMsg:    "source 1: must specify either database connection",
		},
		{
			name:        "both file and db for source 1",
			file1:       "test.sql",
			db1:         "testdb",
			user1:       "testuser",
			file2:       "test2.sql",
			expectError: true,
			errorMsg:    "source 1: cannot specify both database connection and schema file",
		},
		{
			name:        "both file and db for source 2",
			file1:       "test.sql",
			file2:       "test2.sql",
			db2:         "testdb",
			user2:       "testuser",
			expectError: true,
			errorMsg:    "source 2: cannot specify both database connection and schema file",
		},
		{
			name:        "incomplete db connection source 1",
			db1:         "testdb",
			file2:       "test2.sql",
			expectError: true,
			errorMsg:    "source 1: both --db1 and --user1 are required",
		},
		{
			name:        "incomplete db connection source 2",
			file1:       "test1.sql",
			db2:         "testdb",
			expectError: true,
			errorMsg:    "source 2: both --db2 and --user2 are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset all flags
			file1 = tt.file1
			file2 = tt.file2
			db1 = tt.db1
			user1 = tt.user1
			db2 = tt.db2
			user2 = tt.user2

			err := validateSourceInputs()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got none", tt.errorMsg)
				} else if err.Error() != tt.errorMsg && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', but got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			}
		})
	}
}

func TestPlanCommandErrors(t *testing.T) {
	// Save original values
	originalFile1 := file1
	originalFile2 := file2
	
	// Restore original values at the end
	defer func() {
		file1 = originalFile1
		file2 = originalFile2
	}()

	// Test with non-existent files
	file1 = "/non/existent/file1.sql"
	file2 = "/non/existent/file2.sql"
	format = "text"

	err := runPlan(PlanCmd, []string{})
	if err == nil {
		t.Error("Expected error when file1 doesn't exist, but got none")
	}

	// Create a valid schema1 file but keep schema2 invalid
	tmpDir := t.TempDir()
	schema1Path := filepath.Join(tmpDir, "schema1.sql")

	if err := os.WriteFile(schema1Path, []byte("CREATE TABLE test();"), 0644); err != nil {
		t.Fatalf("Failed to write schema1 file: %v", err)
	}

	file1 = schema1Path
	file2 = "/non/existent/file2.sql"

	err = runPlan(PlanCmd, []string{})
	if err == nil {
		t.Error("Expected error when file2 doesn't exist, but got none")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || hasSubstring(s, substr))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
