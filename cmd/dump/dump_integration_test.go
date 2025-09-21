package dump

// Dump Integration Tests
// These comprehensive integration tests verify the entire dump workflow by comparing
// schema representations from two different sources:
// 1. Database inspection (pgdump.sql → database → dump command → schema output)
// 2. Expected output verification (comparing actual vs expected schema dumps)
// This ensures our pgschema output accurately represents the original database schema

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/pgschema/pgschema/internal/util"
	"github.com/pgschema/pgschema/testutil"
)

func TestDumpCommand_Employee(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "employee")
}

func TestDumpCommand_Sakila(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "sakila")
}

func TestDumpCommand_Bytebase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "bytebase")
}

func TestDumpCommand_TenantSchemas(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runTenantSchemaTest(t, "tenant")
}

// TestDumpCommand_PermissionErrors verifies that pgschema handles permission errors properly
// when encountering database objects owned by inaccessible roles.
//
// This test reproduces and verifies the fix for issue #32 where pgschema was outputting
// "%!s(<nil>)" instead of proper error handling for procedures/functions
// owned by roles the user doesn't have access to.
func TestDumpCommand_PermissionErrors(t *testing.T) {
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

		// Try to dump schema as regular_user (should fail with permission error)
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

		// Try to dump schema as regular_user (should fail with permission error)
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

func runExactMatchTest(t *testing.T, testDataDir string) {
	runExactMatchTestWithContext(t, context.Background(), testDataDir)
}

func runExactMatchTestWithContext(t *testing.T, ctx context.Context, testDataDir string) {
	// Setup PostgreSQL container
	containerInfo := testutil.SetupPostgresContainer(ctx, t)
	defer containerInfo.Terminate(ctx, t)

	// Read and execute the pgdump.sql file
	pgdumpPath := fmt.Sprintf("../../testdata/dump/%s/pgdump.sql", testDataDir)
	pgdumpContent, err := os.ReadFile(pgdumpPath)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", pgdumpPath, err)
	}

	// Execute the SQL to create the schema
	_, err = containerInfo.Conn.ExecContext(ctx, string(pgdumpContent))
	if err != nil {
		t.Fatalf("Failed to execute pgdump.sql: %v", err)
	}

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

	// Configure connection parameters
	host = containerInfo.Host
	port = containerInfo.Port
	db = "testdb"
	user = "testuser"
	testutil.SetEnvPassword("testpass")

	// Execute pgschema dump and capture output
	actualOutput := executePgSchemaDump(t, "")

	// Read expected output
	expectedPath := fmt.Sprintf("../../testdata/dump/%s/pgschema.sql", testDataDir)
	expectedContent, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", expectedPath, err)
	}
	expectedOutput := string(expectedContent)

	// Use shared comparison function
	compareSchemaOutputs(t, actualOutput, expectedOutput, testDataDir)
}

func runTenantSchemaTest(t *testing.T, testDataDir string) {
	ctx := context.Background()

	// Setup PostgreSQL container
	containerInfo := testutil.SetupPostgresContainer(ctx, t)
	defer containerInfo.Terminate(ctx, t)

	// Load public schema types first
	publicSQL, err := os.ReadFile(fmt.Sprintf("../../testdata/dump/%s/public.sql", testDataDir))
	if err != nil {
		t.Fatalf("Failed to read public.sql: %v", err)
	}

	_, err = containerInfo.Conn.Exec(string(publicSQL))
	if err != nil {
		t.Fatalf("Failed to load public types: %v", err)
	}

	// Load utility functions (if util.sql exists)
	utilPath := fmt.Sprintf("../../testdata/dump/%s/util.sql", testDataDir)
	if utilSQL, err := os.ReadFile(utilPath); err == nil {
		_, err = containerInfo.Conn.Exec(string(utilSQL))
		if err != nil {
			t.Fatalf("Failed to load utility functions from util.sql: %v", err)
		}
	} else if !os.IsNotExist(err) {
		t.Fatalf("Failed to read util.sql: %v", err)
	}

	// Create two tenant schemas
	tenants := []string{"tenant1", "tenant2"}
	for _, tenant := range tenants {
		_, err = containerInfo.Conn.Exec(fmt.Sprintf("CREATE SCHEMA %s", tenant))
		if err != nil {
			t.Fatalf("Failed to create schema %s: %v", tenant, err)
		}
	}

	// Read the tenant SQL
	tenantSQL, err := os.ReadFile(fmt.Sprintf("../../testdata/dump/%s/pgschema.sql", testDataDir))
	if err != nil {
		t.Fatalf("Failed to read tenant.sql: %v", err)
	}

	// Load the SQL into both tenant schemas
	for _, tenant := range tenants {
		// Set search path to include public for the types, but target schema first
		quotedTenant := util.QuoteIdentifier(tenant)
		_, err = containerInfo.Conn.Exec(fmt.Sprintf("SET search_path TO %s, public", quotedTenant))
		if err != nil {
			t.Fatalf("Failed to set search path to %s: %v", tenant, err)
		}

		// Execute the SQL
		_, err = containerInfo.Conn.Exec(string(tenantSQL))
		if err != nil {
			t.Fatalf("Failed to load SQL into schema %s: %v", tenant, err)
		}
	}

	// Save original command variables
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

	// Dump both tenant schemas using pgschema dump command
	var dumps []string
	for _, tenantName := range tenants {
		// Set connection parameters for this specific tenant dump
		host = containerInfo.Host
		port = containerInfo.Port
		db = "testdb"
		user = "testuser"
		testutil.SetEnvPassword("testpass")
		schema = tenantName

		// Execute pgschema dump and capture output
		actualOutput := executePgSchemaDump(t, fmt.Sprintf("tenant %s", tenantName))
		dumps = append(dumps, actualOutput)
	}

	// Read expected output
	expectedBytes, err := os.ReadFile(fmt.Sprintf("../../testdata/dump/%s/pgschema.sql", testDataDir))
	if err != nil {
		t.Fatalf("Failed to read expected output: %v", err)
	}
	expected := string(expectedBytes)

	// Compare both dumps against expected output and between each other
	for i, dump := range dumps {
		tenantName := tenants[i]

		// Use shared comparison function
		compareSchemaOutputs(t, dump, expected, fmt.Sprintf("%s_%s", testDataDir, tenantName))
	}

	// Also compare the two tenant dumps with each other
	if len(dumps) == 2 {
		compareSchemaOutputs(t, dumps[0], dumps[1], fmt.Sprintf("%s_tenant1_vs_tenant2", testDataDir))
	}
}

func executePgSchemaDump(t *testing.T, contextInfo string) string {
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

	if err != nil {
		if contextInfo != "" {
			t.Fatalf("Dump command failed for %s: %v", contextInfo, err)
		} else {
			t.Fatalf("Dump command failed: %v", err)
		}
	}

	// Wait for reading to complete
	<-done
	if readErr != nil {
		if contextInfo != "" {
			t.Fatalf("Failed to read captured output for %s: %v", contextInfo, readErr)
		} else {
			t.Fatalf("Failed to read captured output: %v", readErr)
		}
	}

	return actualOutput
}

// normalizeSchemaOutput removes version-specific lines for comparison
func normalizeSchemaOutput(output string) string {
	lines := strings.Split(output, "\n")
	var normalizedLines []string

	for _, line := range lines {
		// Skip version-related lines
		if strings.Contains(line, "-- Dumped by pgschema version") ||
			strings.Contains(line, "-- Dumped from database version") {
			continue
		}
		normalizedLines = append(normalizedLines, line)
	}

	return strings.Join(normalizedLines, "\n")
}

// compareSchemaOutputs compares actual and expected schema outputs
func compareSchemaOutputs(t *testing.T, actualOutput, expectedOutput string, testName string) {
	// Normalize both outputs to ignore version differences
	normalizedActual := normalizeSchemaOutput(actualOutput)
	normalizedExpected := normalizeSchemaOutput(expectedOutput)

	// Compare the normalized outputs
	if normalizedActual != normalizedExpected {
		t.Errorf("Output does not match for %s", testName)
		t.Logf("Total lines - Actual: %d, Expected: %d", len(strings.Split(actualOutput, "\n")), len(strings.Split(expectedOutput, "\n")))

		// Write actual output to file for debugging only when test fails
		actualFilename := fmt.Sprintf("%s_actual.sql", testName)
		os.WriteFile(actualFilename, []byte(actualOutput), 0644)

		expectedFilename := fmt.Sprintf("%s_expected.sql", testName)
		os.WriteFile(expectedFilename, []byte(expectedOutput), 0644)
		t.Logf("Outputs written to %s and %s for debugging", actualFilename, expectedFilename)

		// Find and show first difference
		actualLines := strings.Split(normalizedActual, "\n")
		expectedLines := strings.Split(normalizedExpected, "\n")

		for i := 0; i < len(actualLines) && i < len(expectedLines); i++ {
			if actualLines[i] != expectedLines[i] {
				t.Errorf("First difference at line %d:\nActual:   %s\nExpected: %s",
					i+1, actualLines[i], expectedLines[i])
				break
			}
		}

		if len(actualLines) != len(expectedLines) {
			t.Errorf("Different number of lines - Actual: %d, Expected: %d",
				len(actualLines), len(expectedLines))
		}
	} else {
		t.Logf("Success! Output matches for %s", testName)
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
