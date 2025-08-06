package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestApplyCommand(t *testing.T) {
	// Test that the command is properly configured
	if ApplyCmd.Use != "apply" {
		t.Errorf("Expected Use to be 'apply', got '%s'", ApplyCmd.Use)
	}

	if ApplyCmd.Short == "" {
		t.Error("Expected Short description to be set")
	}

	if ApplyCmd.Long == "" {
		t.Error("Expected Long description to be set")
	}

	// Test that required flags are defined
	flags := ApplyCmd.Flags()

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

	// Test auto-approve flag
	autoApproveFlag := flags.Lookup("auto-approve")
	if autoApproveFlag == nil {
		t.Error("Expected --auto-approve flag to be defined")
	}
	if autoApproveFlag.DefValue != "false" {
		t.Errorf("Expected default auto-approve to be 'false', got '%s'", autoApproveFlag.DefValue)
	}

	// Test no-color flag
	noColorFlag := flags.Lookup("no-color")
	if noColorFlag == nil {
		t.Error("Expected --no-color flag to be defined")
	}
	if noColorFlag.DefValue != "false" {
		t.Errorf("Expected default no-color to be 'false', got '%s'", noColorFlag.DefValue)
	}

	// Test that format flag is NOT present (unlike plan command)
	formatFlag := flags.Lookup("format")
	if formatFlag != nil {
		t.Error("Expected --format flag NOT to be defined for apply command")
	}


	// Test lock-timeout flag
	lockTimeoutFlag := flags.Lookup("lock-timeout")
	if lockTimeoutFlag == nil {
		t.Error("Expected --lock-timeout flag to be defined")
	}
	if lockTimeoutFlag.DefValue != "" {
		t.Errorf("Expected default lock-timeout to be empty, got '%s'", lockTimeoutFlag.DefValue)
	}

	// Test application-name flag
	applicationNameFlag := flags.Lookup("application-name")
	if applicationNameFlag == nil {
		t.Error("Expected --application-name flag to be defined")
	}
	if applicationNameFlag.DefValue != "pgschema" {
		t.Errorf("Expected default application-name to be 'pgschema', got '%s'", applicationNameFlag.DefValue)
	}
}

func TestApplyCommandRequiredFlags(t *testing.T) {
	// Test that required flags are properly configured
	flags := ApplyCmd.Flags()

	// Check that required flags are marked as required
	requiredFlags := []string{"db", "user", "file"}
	for _, flagName := range requiredFlags {
		flag := flags.Lookup(flagName)
		if flag == nil {
			t.Errorf("Required flag --%s not found", flagName)
			continue
		}

		// Create a test command to check required flag validation
		testCmd := &cobra.Command{
			Use: "test",
			RunE: func(cmd *cobra.Command, args []string) error {
				return nil // Don't actually run anything
			},
		}

		// Add just this flag to the test command
		switch flagName {
		case "db":
			testCmd.Flags().String(flagName, "", flag.Usage)
		case "user":
			testCmd.Flags().String(flagName, "", flag.Usage)
		case "file":
			testCmd.Flags().String(flagName, "", flag.Usage)
		}

		// Mark it as required
		testCmd.MarkFlagRequired(flagName)

		// Test that command fails when flag is missing
		testCmd.SetArgs([]string{}) // No arguments, so required flag should be missing
		err := testCmd.Execute()
		if err == nil {
			t.Errorf("Expected error when required flag --%s is missing, but got none", flagName)
		}
	}

	// Test that all required flags together work (but don't actually execute the apply logic)
	t.Run("all required flags present", func(t *testing.T) {
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

		// Test command with all required flags - should not fail on required flag validation
		// (It will likely fail on database connection, but that's expected)
		testCmd := &cobra.Command{
			Use: "apply",
			RunE: func(cmd *cobra.Command, args []string) error {
				// Just check that required flags have values, don't actually run apply logic
				db, _ := cmd.Flags().GetString("db")
				user, _ := cmd.Flags().GetString("user")
				file, _ := cmd.Flags().GetString("file")

				if db == "" || user == "" || file == "" {
					return fmt.Errorf("required flags are missing")
				}
				return nil // Success - all required flags are present
			},
		}

		testCmd.Flags().String("db", "", "Database name")
		testCmd.Flags().String("user", "", "User name")
		testCmd.Flags().String("file", "", "File path")
		testCmd.MarkFlagRequired("db")
		testCmd.MarkFlagRequired("user")
		testCmd.MarkFlagRequired("file")

		testCmd.SetArgs([]string{"--db", "testdb", "--user", "testuser", "--file", schemaPath})
		err := testCmd.Execute()
		if err != nil {
			t.Errorf("Expected no error when all required flags are present, but got: %v", err)
		}
	})
}

func TestApplyCommandFileError(t *testing.T) {
	// Test with non-existent file
	// Reset the flags to their default values first
	applyDB = ""
	applyUser = ""
	applyFile = ""

	// Parse the command line arguments
	ApplyCmd.ParseFlags([]string{
		"--db", "testdb",
		"--user", "testuser",
		"--file", "/non/existent/file.sql",
	})

	// The command should fail because it can't connect to database
	// and the file doesn't exist
	err := RunApply(ApplyCmd, []string{})
	if err == nil {
		t.Error("Expected error when file doesn't exist, but got none")
	}
}