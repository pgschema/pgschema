package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
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
	dbFlag := flags.Lookup("db")
	if dbFlag == nil {
		t.Error("Expected --db flag to be defined")
	}

	userFlag := flags.Lookup("user")
	if userFlag == nil {
		t.Error("Expected --user flag to be defined")
	}

	hostFlag := flags.Lookup("host")
	if hostFlag == nil {
		t.Error("Expected --host flag to be defined")
	}
	if hostFlag.DefValue != "localhost" {
		t.Errorf("Expected default host to be 'localhost', got '%s'", hostFlag.DefValue)
	}

	portFlag := flags.Lookup("port")
	if portFlag == nil {
		t.Error("Expected --port flag to be defined")
	}
	if portFlag.DefValue != "5432" {
		t.Errorf("Expected default port to be '5432', got '%s'", portFlag.DefValue)
	}

	passwordFlag := flags.Lookup("password")
	if passwordFlag == nil {
		t.Error("Expected --password flag to be defined")
	}

	schemaFlag := flags.Lookup("schema")
	if schemaFlag == nil {
		t.Error("Expected --schema flag to be defined")
	}
	if schemaFlag.DefValue != "public" {
		t.Errorf("Expected default schema to be 'public', got '%s'", schemaFlag.DefValue)
	}

	// Test desired state file flag
	fileFlag := flags.Lookup("file")
	if fileFlag == nil {
		t.Error("Expected --file flag to be defined")
	}

	// Test output format flag
	formatFlag := flags.Lookup("format")
	if formatFlag == nil {
		t.Error("Expected --format flag to be defined")
	}
	if formatFlag.DefValue != "text" {
		t.Errorf("Expected default format to be 'text', got '%s'", formatFlag.DefValue)
	}
}

func TestPlanCommandRequiredFlags(t *testing.T) {
	// Create a temporary schema file
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.sql")
	
	schemaSQL := `CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);`
	
	if err := os.WriteFile(schemaPath, []byte(schemaSQL), 0644); err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}

	// Test that required flags are marked as required
	tests := []struct {
		name         string
		args         []string
		expectError  bool
	}{
		{
			name:        "missing all required flags",
			args:        []string{},
			expectError: true,
		},
		{
			name:        "missing db flag",
			args:        []string{"--user", "testuser", "--file", schemaPath},
			expectError: true,
		},
		{
			name:        "missing user flag",
			args:        []string{"--db", "testdb", "--file", schemaPath},
			expectError: true,
		},
		{
			name:        "missing file flag",
			args:        []string{"--db", "testdb", "--user", "testuser"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the command to clear any previous flag values
			cmd := &cobra.Command{}
			*cmd = *PlanCmd
			cmd.SetArgs(tt.args)
			
			err := cmd.Execute()
			if tt.expectError && err == nil {
				t.Error("Expected error due to missing required flags, but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}
		})
	}
}

func TestPlanCommandFileError(t *testing.T) {
	// Test with non-existent file
	cmd := &cobra.Command{}
	*cmd = *PlanCmd
	cmd.SetArgs([]string{
		"--db", "testdb",
		"--user", "testuser",
		"--file", "/non/existent/file.sql",
	})
	
	// The command should fail because it can't connect to database
	// and the file doesn't exist
	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error when file doesn't exist, but got none")
	}
}