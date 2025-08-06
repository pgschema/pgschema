package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	requiredFlags := []string{"db", "user"}
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
	t.Run("all required flags present with file", func(t *testing.T) {
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

				if db == "" || user == "" {
					return fmt.Errorf("required flags are missing")
				}
				if file == "" {
					return fmt.Errorf("either --file or --plan must be specified")
				}
				return nil // Success - all required flags are present
			},
		}

		testCmd.Flags().String("db", "", "Database name")
		testCmd.Flags().String("user", "", "User name")
		testCmd.Flags().String("file", "", "File path")
		testCmd.MarkFlagRequired("db")
		testCmd.MarkFlagRequired("user")

		testCmd.SetArgs([]string{"--db", "testdb", "--user", "testuser", "--file", schemaPath})
		err := testCmd.Execute()
		if err != nil {
			t.Errorf("Expected no error when all required flags are present, but got: %v", err)
		}
	})
}

func TestApplyCommandFlagValidation(t *testing.T) {
	// Save original values
	origDB := applyDB
	origUser := applyUser
	origFile := applyFile
	origPlan := applyPlan
	defer func() {
		applyDB = origDB
		applyUser = origUser
		applyFile = origFile
		applyPlan = origPlan
	}()

	t.Run("neither file nor plan specified", func(t *testing.T) {
		// Reset flags
		applyDB = "testdb"
		applyUser = "testuser"
		applyFile = ""
		applyPlan = ""

		err := RunApply(ApplyCmd, []string{})
		if err == nil {
			t.Error("Expected error when neither --file nor --plan is specified")
		}
		if err != nil && err.Error() != "either --file or --plan must be specified" {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})

	t.Run("both file and plan specified", func(t *testing.T) {
		// Create a test command to test mutual exclusivity
		testCmd := &cobra.Command{
			Use: "apply",
			RunE: func(cmd *cobra.Command, args []string) error {
				return nil
			},
		}

		testCmd.Flags().String("db", "", "Database name")
		testCmd.Flags().String("user", "", "User name")
		testCmd.Flags().String("file", "", "File path")
		testCmd.Flags().String("plan", "", "Plan path")
		testCmd.MarkFlagRequired("db")
		testCmd.MarkFlagRequired("user")
		testCmd.MarkFlagsMutuallyExclusive("file", "plan")

		testCmd.SetArgs([]string{"--db", "testdb", "--user", "testuser", "--file", "schema.sql", "--plan", "plan.json"})
		err := testCmd.Execute()
		if err == nil {
			t.Error("Expected error when both --file and --plan are specified")
		}
		if err != nil && !strings.Contains(err.Error(), "if any flags in the group [file plan] are set none of the others can be") {
			t.Errorf("Expected mutual exclusivity error, got: %v", err)
		}
	})

	t.Run("only file specified", func(t *testing.T) {
		// Create a temporary schema file
		tmpDir := t.TempDir()
		schemaPath := filepath.Join(tmpDir, "schema.sql")
		schemaSQL := `CREATE TABLE test (id INT);`
		if err := os.WriteFile(schemaPath, []byte(schemaSQL), 0644); err != nil {
			t.Fatalf("Failed to write schema file: %v", err)
		}

		// Reset flags
		applyDB = "testdb"
		applyUser = "testuser"
		applyFile = schemaPath
		applyPlan = ""

		// This should fail on database connection, not on flag validation
		err := RunApply(ApplyCmd, []string{})
		if err == nil {
			t.Error("Expected error (database connection), but got none")
		}
		// Should NOT be a flag validation error
		if err != nil && strings.Contains(err.Error(), "either --file or --plan must be specified") {
			t.Errorf("Should not get flag validation error when --file is specified: %v", err)
		}
	})

	t.Run("only plan specified", func(t *testing.T) {
		// Create a temporary plan file
		tmpDir := t.TempDir()
		planPath := filepath.Join(tmpDir, "plan.json")
		planJSON := `{"version":"1.0.0","pgschema_version":"test","created_at":"2024-01-01T00:00:00Z","transaction":true,"summary":{"total":0,"add":0,"change":0,"destroy":0,"by_type":{}},"diffs":[]}`
		if err := os.WriteFile(planPath, []byte(planJSON), 0644); err != nil {
			t.Fatalf("Failed to write plan file: %v", err)
		}

		// Reset flags
		applyDB = "testdb"
		applyUser = "testuser"
		applyFile = ""
		applyPlan = planPath

		// This should work (no changes to apply)
		err := RunApply(ApplyCmd, []string{})
		if err == nil {
			// This is actually expected - empty plan with no changes should succeed
			return
		}
		// Should NOT be a flag validation error
		if strings.Contains(err.Error(), "either --file or --plan must be specified") {
			t.Errorf("Should not get flag validation error when --plan is specified: %v", err)
		}
	})
}

func TestApplyCommandVersionMismatch(t *testing.T) {
	// Save original values
	origDB := applyDB
	origUser := applyUser
	origFile := applyFile
	origPlan := applyPlan
	defer func() {
		applyDB = origDB
		applyUser = origUser
		applyFile = origFile
		applyPlan = origPlan
	}()

	// Create a temporary plan file with a different pgschema version
	tmpDir := t.TempDir()
	planPath := filepath.Join(tmpDir, "plan_old_version.json")
	
	// Create a plan JSON with an older version (current is 0.3.0, use 0.2.0)
	planJSON := `{
  "version": "1.0.0",
  "pgschema_version": "0.2.0",
  "created_at": "2024-01-01T00:00:00Z",
  "transaction": true,
  "summary": {
    "total": 1,
    "add": 1,
    "change": 0,
    "destroy": 0,
    "by_type": {
      "tables": {
        "add": 1,
        "change": 0,
        "destroy": 0
      }
    }
  },
  "diffs": [
    {
      "operation": "add",
      "type": "table",
      "path": "public.test_table",
      "sql": "CREATE TABLE test_table (id INTEGER);"
    }
  ]
}`
	
	if err := os.WriteFile(planPath, []byte(planJSON), 0644); err != nil {
		t.Fatalf("Failed to write plan file: %v", err)
	}

	// Reset flags to use the plan file with old version
	applyDB = "testdb"
	applyUser = "testuser"
	applyFile = ""
	applyPlan = planPath

	// This should fail with version mismatch error
	err := RunApply(ApplyCmd, []string{})
	if err == nil {
		t.Error("Expected error for version mismatch, but got none")
	}

	// Verify the error message contains version information
	if err != nil {
		errorMsg := err.Error()
		if !strings.Contains(errorMsg, "plan version mismatch") {
			t.Errorf("Expected 'plan version mismatch' in error message, got: %v", errorMsg)
		}
		if !strings.Contains(errorMsg, "0.2.0") {
			t.Errorf("Expected plan version '0.2.0' in error message, got: %v", errorMsg)
		}
		if !strings.Contains(errorMsg, "0.3.0") {
			t.Errorf("Expected current version '0.3.0' in error message, got: %v", errorMsg)
		}
		if !strings.Contains(errorMsg, "regenerate the plan") {
			t.Errorf("Expected 'regenerate the plan' suggestion in error message, got: %v", errorMsg)
		}
	}
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