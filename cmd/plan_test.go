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
			schema1File = schema1Path
			schema2File = schema2Path
			format = tt.format
			
			err := runPlan(PlanCmd, []string{})
			if err != nil {
				t.Errorf("Expected plan command to succeed with %s format, but got error: %v", tt.format, err)
			}
		})
	}
}

func TestPlanCommandErrors(t *testing.T) {
	// Test with non-existent files
	schema1File = "/non/existent/file1.sql"
	schema2File = "/non/existent/file2.sql"
	format = "text"
	
	err := runPlan(PlanCmd, []string{})
	if err == nil {
		t.Error("Expected error when schema1 file doesn't exist, but got none")
	}
	
	// Create a valid schema1 file but keep schema2 invalid
	tmpDir := t.TempDir()
	schema1Path := filepath.Join(tmpDir, "schema1.sql")
	
	if err := os.WriteFile(schema1Path, []byte("CREATE TABLE test();"), 0644); err != nil {
		t.Fatalf("Failed to write schema1 file: %v", err)
	}
	
	schema1File = schema1Path
	schema2File = "/non/existent/file2.sql"
	
	err = runPlan(PlanCmd, []string{})
	if err == nil {
		t.Error("Expected error when schema2 file doesn't exist, but got none")
	}
}