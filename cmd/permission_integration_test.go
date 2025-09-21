package cmd

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/pgschema/pgschema/cmd/dump"
	"github.com/pgschema/pgschema/testutil"
	"github.com/spf13/cobra"
)

// TestPermissionIssues verifies that pgschema handles permission errors properly
// when encountering database objects owned by inaccessible roles.
//
// This test reproduces the bug described in issue #32 where pgschema outputs
// "%!s(<nil>)" instead of proper error handling for procedures/functions
// owned by roles the user doesn't have access to.
func TestPermissionIssues(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start PostgreSQL container with superuser privileges
	container := testutil.SetupPostgresContainerWithDB(ctx, t, "testdb", "postgres", "testpwd")
	defer container.Terminate(ctx, t)

	conn := container.Conn

	// Shared setup: Create restricted role and regular user
	_, err := conn.ExecContext(ctx, `
		-- Create a restricted role that regular users can't access
		CREATE ROLE restricted_owner;

		-- Create a regular user without access to restricted_owner
		CREATE USER regular_user WITH PASSWORD 'userpass';
		GRANT CONNECT ON DATABASE testdb TO regular_user;
		GRANT USAGE ON SCHEMA public TO regular_user;
	`)
	if err != nil {
		t.Fatalf("Failed to setup shared permission test roles: %v", err)
	}

	// Test Case 1: Procedure owned by restricted role
	t.Run("procedure_owned_by_restricted_role", func(t *testing.T) {
		// Setup: Create procedure owned by restricted role
		_, err := conn.ExecContext(ctx, `
			-- Create a procedure in public schema owned by the restricted role
			CREATE OR REPLACE PROCEDURE public.test_procedure(param_name TEXT)
			LANGUAGE plpgsql
			AS $$
			BEGIN
				RAISE NOTICE 'This procedure is owned by restricted_owner: %', param_name;
			END;
			$$;

			-- Change ownership to the restricted role
			ALTER PROCEDURE public.test_procedure(TEXT) OWNER TO restricted_owner;
		`)
		if err != nil {
			t.Fatalf("Failed to setup permission test scenario: %v", err)
		}

		// Try to dump schema as regular_user (should fail or warn about permissions)
		output, err := executeDumpCommandAsUser(
			container.Host,
			container.Port,
			"testdb",
			"regular_user",
			"userpass",
			"public",
		)

		// Check that the bug is fixed: should NOT contain "%!s(<nil>)" anymore
		if strings.Contains(output, "%!s(<nil>)") {
			t.Errorf("BUG STILL EXISTS: Found '%s' in dump output instead of proper error handling", "%!s(<nil>)")
			t.Logf("Full output:\n%s", output)
		}

		// Expected behavior: Should return an error about permission denied
		if err == nil {
			t.Errorf("Expected permission-related error, but dump succeeded")
		} else if !strings.Contains(err.Error(), "permission denied") {
			t.Errorf("Expected 'permission denied' error, got: %v", err)
		}

		t.Logf("Dump output:\n%s", output)
		if err != nil {
			t.Logf("Dump error (this may be expected): %v", err)
		}
	})

	// Test Case 2: Function owned by restricted role
	t.Run("function_owned_by_restricted_role", func(t *testing.T) {
		// Setup: Create function owned by restricted role
		_, err := conn.ExecContext(ctx, `
			-- Create a function in public schema owned by the restricted role
			CREATE OR REPLACE FUNCTION public.test_function(input_text TEXT)
			RETURNS TEXT
			LANGUAGE plpgsql
			AS $$
			BEGIN
				RETURN 'Processed: ' || input_text;
			END;
			$$;

			-- Change ownership to the restricted role
			ALTER FUNCTION public.test_function(TEXT) OWNER TO restricted_owner;
		`)
		if err != nil {
			t.Fatalf("Failed to setup function permission test: %v", err)
		}

		// Try to dump schema as regular_user (should fail or warn about permissions)
		output, err := executeDumpCommandAsUser(
			container.Host,
			container.Port,
			"testdb",
			"regular_user",
			"userpass",
			"public",
		)

		// Check that the bug is fixed: should NOT contain "%!s(<nil>)" anymore
		if strings.Contains(output, "%!s(<nil>)") {
			t.Errorf("BUG STILL EXISTS: Found '%s' in function dump output instead of proper error handling", "%!s(<nil>)")
			t.Logf("Full output:\n%s", output)
		}

		// Expected behavior: Should return an error about permission denied
		if err == nil {
			t.Errorf("Expected permission-related error for inaccessible function, but dump succeeded")
		} else if !strings.Contains(err.Error(), "permission denied") {
			t.Errorf("Expected 'permission denied' error for function, got: %v", err)
		}

		t.Logf("Function dump output:\n%s", output)
		if err != nil {
			t.Logf("Function dump error (this may be expected): %v", err)
		}
	})

	// Test Case 3: Mixed accessible and inaccessible objects
	t.Run("mixed_accessible_and_inaccessible_objects", func(t *testing.T) {
		// Cleanup: Remove any existing procedures/functions from previous tests
		_, err := conn.ExecContext(ctx, `
			DROP PROCEDURE IF EXISTS public.test_procedure(TEXT);
			DROP FUNCTION IF EXISTS public.test_function(TEXT);
		`)
		if err != nil {
			t.Logf("Cleanup warning (non-fatal): %v", err)
		}

		// Setup: Create both accessible and inaccessible procedures
		_, err = conn.ExecContext(ctx, `
			-- Create accessible procedure owned by regular_user
			CREATE OR REPLACE PROCEDURE public.accessible_procedure()
			LANGUAGE plpgsql
			AS $$
			BEGIN
				RAISE NOTICE 'This procedure is accessible';
			END;
			$$;

			ALTER PROCEDURE accessible_procedure() OWNER TO regular_user;

			-- Create inaccessible procedure owned by restricted_owner
			CREATE OR REPLACE PROCEDURE public.inaccessible_procedure(test_param INTEGER)
			LANGUAGE plpgsql
			AS $$
			BEGIN
				RAISE NOTICE 'This procedure should not be accessible: %', test_param;
			END;
			$$;

			ALTER PROCEDURE inaccessible_procedure(INTEGER) OWNER TO restricted_owner;
		`)
		if err != nil {
			t.Fatalf("Failed to setup mixed permission test: %v", err)
		}

		// Try to dump schema as regular_user
		output, err := executeDumpCommandAsUser(
			container.Host,
			container.Port,
			"testdb",
			"regular_user",
			"userpass",
			"public",
		)

		// Should not contain the nil formatting bug
		if strings.Contains(output, "%!s(<nil>)") {
			t.Errorf("BUG STILL EXISTS: Found '%s' in mixed dump output", "%!s(<nil>)")
		}

		// Since we abort on permission errors, the dump should fail entirely
		// when encountering the inaccessible procedure
		if err == nil {
			t.Errorf("Expected permission-related error due to inaccessible procedure, but dump succeeded")
		} else if !strings.Contains(err.Error(), "permission denied") {
			t.Errorf("Expected 'permission denied' error, got: %v", err)
		}

		// The accessible procedure won't be in output because we abort on the first permission error
		// This is the expected behavior: fail fast rather than silently skip

		t.Logf("Mixed dump output:\n%s", output)
		if err != nil {
			t.Logf("Mixed dump error: %v", err)
		}
	})

	// Test Case 4: Procedure ignored via .pgschemaignore (future behavior)
	t.Run("ignored_procedure_with_permission_issues", func(t *testing.T) {
		// This test case will verify that when a procedure is explicitly ignored
		// via .pgschemaignore, permission issues should not cause the dump to fail

		// For now, this is a placeholder - the actual implementation will
		// need to respect .pgschemaignore patterns for procedures with permission issues

		t.Skip("Placeholder for future .pgschemaignore integration test")
	})
}

// executeDumpCommandAsUser executes the pgschema dump command as a specific user
func executeDumpCommandAsUser(host string, port int, database, user, password, schema string) (string, error) {
	// Create root command with dump as subcommand
	rootCmd := &cobra.Command{
		Use: "pgschema",
	}
	rootCmd.AddCommand(dump.DumpCmd)

	// Capture output
	var output strings.Builder

	// Set command arguments
	args := []string{
		"dump",
		"--host", host,
		"--port", fmt.Sprintf("%d", port),
		"--db", database,
		"--user", user,
		"--password", password,
		"--schema", schema,
	}
	rootCmd.SetArgs(args)
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)

	// Execute the command
	err := rootCmd.Execute()

	return output.String(), err
}