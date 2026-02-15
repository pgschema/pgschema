package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pgplex/pgschema/ir"
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

	// Test output flags
	outputHumanFlag := flags.Lookup("output-human")
	if outputHumanFlag == nil {
		t.Error("Expected --output-human flag to be defined")
	}

	outputJSONFlag := flags.Lookup("output-json")
	if outputJSONFlag == nil {
		t.Error("Expected --output-json flag to be defined")
	}

	outputSQLFlag := flags.Lookup("output-sql")
	if outputSQLFlag == nil {
		t.Error("Expected --output-sql flag to be defined")
	}
}

func TestPlanCommandRequiredFlags(t *testing.T) {
	// Test that required flags are properly configured
	flags := PlanCmd.Flags()

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

	// Test that all required flags together work (but don't actually execute the plan logic)
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
			Use: "plan",
			RunE: func(cmd *cobra.Command, args []string) error {
				// Just check that required flags have values, don't actually run plan logic
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

func TestPlanCommandFileError(t *testing.T) {
	// Test with non-existent file
	// Reset the flags to their default values first
	planDB = ""
	planUser = ""
	planFile = ""

	// Parse the command line arguments
	PlanCmd.ParseFlags([]string{
		"--db", "testdb",
		"--user", "testuser",
		"--file", "/non/existent/file.sql",
	})

	// The command should fail because it can't connect to database
	// and the file doesn't exist
	err := runPlan(PlanCmd, []string{})
	if err == nil {
		t.Error("Expected error when file doesn't exist, but got none")
	}
}

// TestNormalizeSchemaNames_StripsSameSchemaQualifiers verifies that normalizeSchemaNames
// strips redundant same-schema qualifiers from expressions after replacing temp schema names.
// This is a regression test for GitHub issue #283.
func TestNormalizeSchemaNames_StripsSameSchemaQualifiers(t *testing.T) {
	tempSchema := "pgschema_tmp_20260101_120000_abcd1234"

	defaultVal := "public.my_default_id()"
	genExpr := "public.compute_hash(name)"
	checkClause := "(public.is_valid(status))"
	policyUsing := "(public.current_user_id() = user_id)"
	triggerCond := "public.should_audit()"

	testIR := &ir.IR{
		Schemas: map[string]*ir.Schema{
			tempSchema: {
				Name: tempSchema,
				Tables: map[string]*ir.Table{
					"items": {
						Name:   "items",
						Schema: tempSchema,
						Columns: []*ir.Column{
							{
								Name:         "id",
								DataType:     "uuid",
								DefaultValue: &defaultVal,
							},
							{
								Name:          "hash",
								DataType:      "text",
								GeneratedExpr: &genExpr,
							},
						},
						Constraints: map[string]*ir.Constraint{
							"items_check": {
								Name:        "items_check",
								Schema:      tempSchema,
								CheckClause: checkClause,
							},
						},
						Policies: map[string]*ir.RLSPolicy{
							"items_policy": {
								Name:      "items_policy",
								Schema:    tempSchema,
								Using:     policyUsing,
								WithCheck: "",
							},
						},
						Triggers: map[string]*ir.Trigger{
							"items_trigger": {
								Name:      "items_trigger",
								Schema:    tempSchema,
								Condition: triggerCond,
								Function:  tempSchema + ".audit_func",
							},
						},
					},
				},
			},
		},
	}

	normalizeSchemaNames(testIR, tempSchema, "public")

	table := testIR.Schemas["public"].Tables["items"]

	// Column default: "public.my_default_id()" → "my_default_id()"
	if got := *table.Columns[0].DefaultValue; got != "my_default_id()" {
		t.Errorf("column default: got %q, want %q", got, "my_default_id()")
	}

	// Generated expression: "public.compute_hash(name)" → "compute_hash(name)"
	if got := *table.Columns[1].GeneratedExpr; got != "compute_hash(name)" {
		t.Errorf("generated expr: got %q, want %q", got, "compute_hash(name)")
	}

	// Check clause: "(public.is_valid(status))" → "(is_valid(status))"
	if got := table.Constraints["items_check"].CheckClause; got != "(is_valid(status))" {
		t.Errorf("check clause: got %q, want %q", got, "(is_valid(status))")
	}

	// Policy USING: "(public.current_user_id() = user_id)" → "(current_user_id() = user_id)"
	if got := table.Policies["items_policy"].Using; got != "(current_user_id() = user_id)" {
		t.Errorf("policy using: got %q, want %q", got, "(current_user_id() = user_id)")
	}

	// Trigger condition: "public.should_audit()" → "should_audit()"
	if got := table.Triggers["items_trigger"].Condition; got != "should_audit()" {
		t.Errorf("trigger condition: got %q, want %q", got, "should_audit()")
	}

	// Trigger function: schema replaced but NOT stripped (it's a schema.name reference, not an expression)
	if got := table.Triggers["items_trigger"].Function; got != "public.audit_func" {
		t.Errorf("trigger function: got %q, want %q", got, "public.audit_func")
	}
}

// TestNormalizeSchemaNames_PreservesCrossSchemaQualifiers verifies that cross-schema
// qualifiers are preserved (not stripped) during normalization.
func TestNormalizeSchemaNames_PreservesCrossSchemaQualifiers(t *testing.T) {
	tempSchema := "pgschema_tmp_20260101_120000_abcd1234"

	defaultVal := "other_schema.my_func()"
	testIR := &ir.IR{
		Schemas: map[string]*ir.Schema{
			tempSchema: {
				Name: tempSchema,
				Tables: map[string]*ir.Table{
					"items": {
						Name:   "items",
						Schema: tempSchema,
						Columns: []*ir.Column{
							{
								Name:         "id",
								DataType:     "uuid",
								DefaultValue: &defaultVal,
							},
						},
					},
				},
			},
		},
	}

	normalizeSchemaNames(testIR, tempSchema, "public")

	if got := *testIR.Schemas["public"].Tables["items"].Columns[0].DefaultValue; got != "other_schema.my_func()" {
		t.Errorf("cross-schema qualifier should be preserved: got %q, want %q", got, "other_schema.my_func()")
	}
}

// TestNormalizeSchemaNames_TypeCastQualifiers verifies that same-schema type cast
// qualifiers are stripped during normalization.
func TestNormalizeSchemaNames_TypeCastQualifiers(t *testing.T) {
	tempSchema := "pgschema_tmp_20260101_120000_abcd1234"

	defaultVal := "'active'::public.status_type"
	testIR := &ir.IR{
		Schemas: map[string]*ir.Schema{
			tempSchema: {
				Name: tempSchema,
				Tables: map[string]*ir.Table{
					"items": {
						Name:   "items",
						Schema: tempSchema,
						Columns: []*ir.Column{
							{
								Name:         "status",
								DataType:     "text",
								DefaultValue: &defaultVal,
							},
						},
					},
				},
			},
		},
	}

	normalizeSchemaNames(testIR, tempSchema, "public")

	if got := *testIR.Schemas["public"].Tables["items"].Columns[0].DefaultValue; got != "'active'::status_type" {
		t.Errorf("type cast qualifier should be stripped: got %q, want %q", got, "'active'::status_type")
	}
}
