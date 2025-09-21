package dump

// Permission Integration Tests
// These tests verify that pgschema handles permission errors properly when
// encountering database objects owned by inaccessible roles.
//
// This reproduces and verifies the fix for issue #32 where pgschema was outputting
// "%!s(<nil>)" instead of proper error handling for procedures/functions
// owned by roles the user doesn't have access to.

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/pgschema/pgschema/cmd/util"
	"github.com/pgschema/pgschema/testutil"
)

// TestDumpCommand_PermissionSuite verifies that pgschema handles permission errors properly
// when encountering database objects owned by inaccessible roles.
func TestDumpCommand_PermissionSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start single PostgreSQL container for all permission tests
	container := testutil.SetupPostgresContainerWithDB(ctx, t, "testdb", "postgres", "testpwd")
	defer container.Terminate(ctx, t)

	// Run each permission test with its own isolated database
	t.Run("ProcedurePermissionError", func(t *testing.T) {
		testProcedurePermission(t, ctx, container, "testdb_proc")
	})

	t.Run("FunctionPermissionError", func(t *testing.T) {
		testFunctionPermission(t, ctx, container, "testdb_func")
	})

	t.Run("MixedAccessibilityObjects", func(t *testing.T) {
		testMixedAccessibility(t, ctx, container, "testdb_mixed")
	})

	t.Run("IgnoredObjectsWithPermissionIssues", func(t *testing.T) {
		testIgnoredObjectsWithPermissions(t, ctx, container, "testdb_ignore")
	})
}

// setupTestDatabase creates a new database with permission test roles
func setupTestDatabase(ctx context.Context, t *testing.T, container *testutil.ContainerInfo, dbName string) *sql.DB {
	// Create the database
	_, err := container.Conn.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		t.Fatalf("Failed to create database %s: %v", dbName, err)
	}

	// Create unique role names for this test
	restrictedRole := fmt.Sprintf("restricted_owner_%s", dbName)
	regularUser := fmt.Sprintf("regular_user_%s", dbName)

	// Create roles in the main postgres connection
	_, err = container.Conn.ExecContext(ctx, fmt.Sprintf(`
		-- Create a restricted role that regular users can't access
		CREATE ROLE %s;

		-- Create a regular user without access to restricted_owner
		CREATE USER %s WITH PASSWORD 'userpass';
		GRANT CONNECT ON DATABASE %s TO %s;
	`, restrictedRole, regularUser, dbName, regularUser))
	if err != nil {
		t.Fatalf("Failed to setup permission test roles for %s: %v", dbName, err)
	}

	// Connect to the new database
	config := &util.ConnectionConfig{
		Host:            container.Host,
		Port:            container.Port,
		Database:        dbName,
		User:            "postgres",
		Password:        "testpwd",
		SSLMode:         "prefer",
		ApplicationName: "pgschema",
	}

	dbConn, err := util.Connect(config)
	if err != nil {
		t.Fatalf("Failed to connect to database %s: %v", dbName, err)
	}

	// Grant schema permissions within the database
	_, err = dbConn.ExecContext(ctx, fmt.Sprintf(`
		GRANT USAGE ON SCHEMA public TO %s;
	`, regularUser))
	if err != nil {
		t.Fatalf("Failed to setup schema permissions for %s: %v", dbName, err)
	}

	return dbConn
}

// getRoleNames returns the unique role names for a given database
func getRoleNames(dbName string) (restrictedRole string, regularUser string) {
	return fmt.Sprintf("restricted_owner_%s", dbName), fmt.Sprintf("regular_user_%s", dbName)
}

// testProcedurePermission tests procedure owned by restricted role
func testProcedurePermission(t *testing.T, ctx context.Context, container *testutil.ContainerInfo, dbName string) {
	// Setup isolated database
	dbConn := setupTestDatabase(ctx, t, container, dbName)
	defer dbConn.Close()

	// Get role names for this database
	restrictedRole, regularUser := getRoleNames(dbName)

	// Create procedure owned by restricted role
	_, err := dbConn.ExecContext(ctx, fmt.Sprintf(`
		-- Create a procedure in public schema owned by the restricted role
		CREATE OR REPLACE PROCEDURE public.test_procedure(param_name TEXT)
		LANGUAGE plpgsql
		AS $$
		BEGIN
			RAISE NOTICE 'This procedure is owned by restricted_owner: %', param_name;
		END;
		$$;

		-- Change ownership to the restricted role
		ALTER PROCEDURE public.test_procedure(TEXT) OWNER TO %s;
	`, restrictedRole))
	if err != nil {
		t.Fatalf("Failed to setup permission test scenario: %v", err)
	}

	// Try to dump schema as regular_user (should fail with permission error)
	output, err := executeDumpCommandAsUser(
		container.Host,
		container.Port,
		dbName,
		regularUser,
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
}

// testFunctionPermission tests function owned by restricted role
func testFunctionPermission(t *testing.T, ctx context.Context, container *testutil.ContainerInfo, dbName string) {
	// Setup isolated database
	dbConn := setupTestDatabase(ctx, t, container, dbName)
	defer dbConn.Close()

	// Get role names for this database
	restrictedRole, regularUser := getRoleNames(dbName)

	// Create function owned by restricted role
	_, err := dbConn.ExecContext(ctx, fmt.Sprintf(`
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
		ALTER FUNCTION public.test_function(TEXT) OWNER TO %s;
	`, restrictedRole))
	if err != nil {
		t.Fatalf("Failed to setup function permission test: %v", err)
	}

	// Try to dump schema as regular_user (should fail with permission error)
	output, err := executeDumpCommandAsUser(
		container.Host,
		container.Port,
		dbName,
		regularUser,
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
}

// testMixedAccessibility tests mixed accessible and inaccessible objects
func testMixedAccessibility(t *testing.T, ctx context.Context, container *testutil.ContainerInfo, dbName string) {
	// Setup isolated database
	dbConn := setupTestDatabase(ctx, t, container, dbName)
	defer dbConn.Close()

	// Get role names for this database
	restrictedRole, regularUser := getRoleNames(dbName)

	// Create both accessible and inaccessible procedures
	_, err := dbConn.ExecContext(ctx, fmt.Sprintf(`
		-- Create accessible procedure owned by regular_user
		CREATE OR REPLACE PROCEDURE public.accessible_procedure()
		LANGUAGE plpgsql
		AS $$
		BEGIN
			RAISE NOTICE 'This procedure is accessible';
		END;
		$$;

		ALTER PROCEDURE accessible_procedure() OWNER TO %s;

		-- Create inaccessible procedure owned by restricted_owner
		CREATE OR REPLACE PROCEDURE public.inaccessible_procedure(test_param INTEGER)
		LANGUAGE plpgsql
		AS $$
		BEGIN
			RAISE NOTICE 'This procedure should not be accessible: %', test_param;
		END;
		$$;

		ALTER PROCEDURE inaccessible_procedure(INTEGER) OWNER TO %s;
	`, regularUser, restrictedRole))
	if err != nil {
		t.Fatalf("Failed to setup mixed permission test: %v", err)
	}

	// Try to dump schema as regular_user
	output, err := executeDumpCommandAsUser(
		container.Host,
		container.Port,
		dbName,
		regularUser,
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
}

// testIgnoredObjectsWithPermissions tests procedures ignored via .pgschemaignore
func testIgnoredObjectsWithPermissions(t *testing.T, ctx context.Context, container *testutil.ContainerInfo, dbName string) {
	// This test verifies that when procedures/functions are explicitly ignored
	// via .pgschemaignore, permission issues should not cause the dump to fail

	// Setup isolated database
	dbConn := setupTestDatabase(ctx, t, container, dbName)
	defer dbConn.Close()

	// Get role names for this database
	restrictedRole, regularUser := getRoleNames(dbName)

	// Create both ignored procedures/functions with permission issues
	// and non-ignored ones that should still cause errors
	_, err := dbConn.ExecContext(ctx, fmt.Sprintf(`
		-- Create ignored procedure owned by restricted role
		CREATE OR REPLACE PROCEDURE public.skip_proc_restricted(param_text TEXT)
		LANGUAGE plpgsql
		AS $$
		BEGIN
			RAISE NOTICE 'This procedure is ignored and restricted: %%', param_text;
		END;
		$$;

		ALTER PROCEDURE skip_proc_restricted(TEXT) OWNER TO %s;

		-- Create ignored function owned by restricted role
		CREATE OR REPLACE FUNCTION public.skip_func_restricted(input_text TEXT)
		RETURNS TEXT
		LANGUAGE plpgsql
		AS $$
		BEGIN
			RETURN 'Ignored function result: ' || input_text;
		END;
		$$;

		ALTER FUNCTION skip_func_restricted(TEXT) OWNER TO %s;

		-- Create non-ignored procedure owned by restricted role (should still cause error)
		CREATE OR REPLACE PROCEDURE public.check_proc_restricted(param_int INTEGER)
		LANGUAGE plpgsql
		AS $$
		BEGIN
			RAISE NOTICE 'This procedure is not ignored and restricted: %', param_int;
		END;
		$$;

		ALTER PROCEDURE check_proc_restricted(INTEGER) OWNER TO %s;
	`, restrictedRole, restrictedRole, restrictedRole))
	if err != nil {
		t.Fatalf("Failed to setup ignore permission test: %v", err)
	}

	// Create .pgschemaignore file that ignores the specific procedures/functions
	ignoreContent := `[procedures]
patterns = ["skip_proc_restricted"]

[functions]
patterns = ["skip_func_restricted"]
`

	err = os.WriteFile(".pgschemaignore", []byte(ignoreContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .pgschemaignore file: %v", err)
	}

	// Cleanup function to remove .pgschemaignore file
	defer func() {
		os.Remove(".pgschemaignore")
	}()

	// Try to dump schema as regular_user
	// Should fail due to check_proc_restricted, but ignored objects should not cause issues
	output, err := executeDumpCommandAsUser(
		container.Host,
		container.Port,
		dbName,
		regularUser,
		"userpass",
		"public",
	)

	// Should not contain the nil formatting bug
	if strings.Contains(output, "%!s(<nil>)") {
		t.Errorf("BUG STILL EXISTS: Found '%s' in dump output with ignored objects", "%!s(<nil>)")
	}

	// The dump should still fail because of the non-ignored restricted procedure
	if err == nil {
		t.Errorf("Expected permission-related error due to non-ignored restricted procedure, but dump succeeded")
	} else if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("Expected 'permission denied' error, got: %v", err)
	}

	// Verify the error is specifically about the non-ignored procedure
	if err != nil && !strings.Contains(err.Error(), "check_proc_restricted") {
		t.Errorf("Expected error about 'check_proc_restricted', got: %v", err)
	}

	// The ignored procedures/functions should not appear in the error message
	if err != nil {
		if strings.Contains(err.Error(), "skip_proc_restricted") {
			t.Errorf("Error should not mention skip_proc_restricted (it should be skipped), got: %v", err)
		}
		if strings.Contains(err.Error(), "skip_func_restricted") {
			t.Errorf("Error should not mention skip_func_restricted (it should be skipped), got: %v", err)
		}
	}

	// Verify that the ignored objects were indeed skipped successfully
	// The error should only be about the non-ignored procedure
	if err != nil && strings.Contains(err.Error(), "check_proc_restricted") {
		t.Logf("✓ Correctly failed only on non-ignored restricted procedure")
	}

	t.Logf("Ignore test dump output:\n%s", output)
	if err != nil {
		t.Logf("Ignore test dump error (expected due to non-ignored object): %v", err)
	}

	// Additional test: Create .pgschemaignore that ignores ALL restricted objects
	ignoreAllContent := `[procedures]
patterns = ["*_restricted"]

[functions]
patterns = ["*_restricted"]
`

	err = os.WriteFile(".pgschemaignore", []byte(ignoreAllContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create comprehensive .pgschemaignore file: %v", err)
	}

	// Now the dump should succeed because all restricted objects are ignored
	output2, err2 := executeDumpCommandAsUser(
		container.Host,
		container.Port,
		dbName,
		regularUser,
		"userpass",
		"public",
	)

	if err2 != nil {
		t.Errorf("Expected dump to succeed when all restricted objects are ignored, got error: %v", err2)
	}

	// Should not contain the nil formatting bug
	if strings.Contains(output2, "%!s(<nil>)") {
		t.Errorf("BUG STILL EXISTS: Found '%s' in comprehensive ignore dump output", "%!s(<nil>)")
	}

	t.Logf("Comprehensive ignore test dump output:\n%s", output2)
	if err2 != nil {
		t.Logf("Comprehensive ignore test dump error (should be nil): %v", err2)
	} else {
		t.Logf("✓ Dump succeeded when all restricted objects were ignored")
	}
}

// executeDumpCommandAsUser executes the pgschema dump command as a specific user
// This helper is used specifically for permission testing where we need to run
// the dump command with restricted database user credentials.
func executeDumpCommandAsUser(hostArg string, portArg int, database, userArg, password, schemaArg string) (string, error) {
	// Store original connection parameters and restore them later
	originalConfig := testutil.TestConnectionConfig{
		Host:   host,
		Port:   port,
		DB:     db,
		User:   user,
		Schema: schema,
	}
	defer func() {
		host = originalConfig.Host
		port = originalConfig.Port
		db = originalConfig.DB
		user = originalConfig.User
		schema = originalConfig.Schema
	}()

	// Set connection parameters for this specific dump
	host = hostArg
	port = portArg
	db = database
	user = userArg
	schema = schemaArg
	testutil.SetEnvPassword(password)

	// Capture output by redirecting stdout
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Read from pipe in a goroutine to avoid deadlock
	var actualOutput string
	var readErr error
	done := make(chan bool)

	go func() {
		defer close(done)
		output, err := io.ReadAll(r)
		if err != nil {
			readErr = err
			return
		}
		actualOutput = string(output)
	}()

	// Run the dump command
	// Logger setup handled by root command
	err := runDump(nil, nil)

	// Close write end and restore stdout
	w.Close()
	os.Stdout = originalStdout

	// Wait for reading to complete
	<-done
	if readErr != nil {
		return "", fmt.Errorf("failed to read captured output: %w", readErr)
	}

	return actualOutput, err
}