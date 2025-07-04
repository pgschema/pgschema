package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/pgschema/pgschema/testutil"
)

// normalizeVersionString replaces version strings with a normalized format for comparison
// This allows tests to pass when only the version info differs
func normalizeVersionString(content string) string {
	// Replace "Dumped by pgschema version X.Y.Z"
	re := regexp.MustCompile(`-- Dumped by pgschema version [^\n]+`)

	return re.ReplaceAllString(content, "-- Dumped by pgschema version NORMALIZED")
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
	setupLogger()
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

// compareSchemaOutputs compares actual and expected schema outputs, ignoring version differences
func compareSchemaOutputs(t *testing.T, actualOutput, expectedOutput string, testName string) {
	// Normalize version strings for comparison
	normalizedActual := normalizeVersionString(actualOutput)
	normalizedExpected := normalizeVersionString(expectedOutput)

	// Compare the normalized outputs
	if normalizedActual != normalizedExpected {
		t.Errorf("Output does not match for %s", testName)
		t.Logf("Total lines - Actual: %d, Expected: %d", len(strings.Split(actualOutput, "\n")), len(strings.Split(expectedOutput, "\n")))

		// Write actual output to file for debugging only when test fails
		actualFilename := fmt.Sprintf("%s_actual.sql", testName)
		if err := os.WriteFile(actualFilename, []byte(actualOutput), 0644); err != nil {
			t.Logf("Failed to write actual output file for debugging: %v", err)
		} else {
			t.Logf("Actual output written to %s for debugging", actualFilename)
		}

		// Also write normalized versions for comparison
		normalizedActualFilename := fmt.Sprintf("%s_normalized_actual.sql", testName)
		normalizedExpectedFilename := fmt.Sprintf("%s_normalized_expected.sql", testName)
		os.WriteFile(normalizedActualFilename, []byte(normalizedActual), 0644)
		os.WriteFile(normalizedExpectedFilename, []byte(normalizedExpected), 0644)
		t.Logf("Normalized outputs written to %s and %s for debugging", normalizedActualFilename, normalizedExpectedFilename)

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
		t.Logf("Success! Output matches for %s (version differences ignored)", testName)
	}
}

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

func runExactMatchTest(t *testing.T, testDataDir string) {
	runExactMatchTestWithContext(t, context.Background(), testDataDir)
}

func runExactMatchTestWithContext(t *testing.T, ctx context.Context, testDataDir string) {
	// Setup PostgreSQL container
	containerInfo := testutil.SetupPostgresContainer(ctx, t)
	defer containerInfo.Terminate(ctx, t)

	// Read and execute the pgdump.sql file
	pgdumpPath := fmt.Sprintf("../testdata/%s/pgdump.sql", testDataDir)
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
	expectedPath := fmt.Sprintf("../testdata/%s/pgschema.sql", testDataDir)
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
	publicSQL, err := os.ReadFile(fmt.Sprintf("../testdata/%s/public.sql", testDataDir))
	if err != nil {
		t.Fatalf("Failed to read public.sql: %v", err)
	}

	_, err = containerInfo.Conn.Exec(string(publicSQL))
	if err != nil {
		t.Fatalf("Failed to load public types: %v", err)
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
	tenantSQL, err := os.ReadFile(fmt.Sprintf("../testdata/%s/pgschema.sql", testDataDir))
	if err != nil {
		t.Fatalf("Failed to read tenant.sql: %v", err)
	}

	// Load the SQL into both tenant schemas
	for _, tenant := range tenants {
		// Set search path to include public for the types, but target schema first
		_, err = containerInfo.Conn.Exec(fmt.Sprintf("SET search_path TO %s, public", tenant))
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
	expectedBytes, err := os.ReadFile(fmt.Sprintf("../testdata/%s/pgschema.sql", testDataDir))
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
