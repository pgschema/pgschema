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
	"os"
	"strings"
	"testing"

	"github.com/pgschema/pgschema/ir"
	"github.com/pgschema/pgschema/testutil"
)

func TestDumpCommand_Employee(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "employee", "TestDumpCommand_Employee")
}

func TestDumpCommand_Sakila(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "sakila", "TestDumpCommand_Sakila")
}

func TestDumpCommand_Bytebase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "bytebase", "TestDumpCommand_Bytebase")
}

func TestDumpCommand_TenantSchemas(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runTenantSchemaTest(t, "tenant")
}

func TestDumpCommand_Issue78ConstraintNotValid(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "issue_78_constraint_not_valid", "TestDumpCommand_Issue78ConstraintNotValid")
}

func TestDumpCommand_Issue80IndexNameQuote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "issue_80_index_name_quote", "TestDumpCommand_Issue80IndexNameQuote")
}

func TestDumpCommand_Issue82ViewLogicExpr(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "issue_82_view_logic_expr", "TestDumpCommand_Issue82ViewLogicExpr")
}

func TestDumpCommand_Issue83ExplicitConstraintName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "issue_83_explicit_constraint_name", "TestDumpCommand_Issue83ExplicitConstraintName")
}

func TestDumpCommand_Issue125FunctionDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "issue_125_function_default", "TestDumpCommand_Issue125FunctionDefault")
}

func TestDumpCommand_Issue133IndexSort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "issue_133_index_sort", "TestDumpCommand_Issue133IndexSort")
}

func runExactMatchTest(t *testing.T, testDataDir string, testName string) {
	runExactMatchTestWithContext(t, context.Background(), testDataDir, testName)
}

func runExactMatchTestWithContext(t *testing.T, ctx context.Context, testDataDir string, testName string) {
	// Setup PostgreSQL
	embeddedPG := testutil.SetupPostgres(t)
	defer embeddedPG.Stop()

	// Connect to database
	conn, host, port, dbname, user, password := testutil.ConnectToPostgres(t, embeddedPG)
	defer conn.Close()

	// Detect PostgreSQL version and skip tests if needed
	majorVersion, err := testutil.GetMajorVersion(conn)
	if err != nil {
		t.Fatalf("Failed to detect PostgreSQL version: %v", err)
	}

	// Check if this test should be skipped for this PostgreSQL version
	// If skipped, ShouldSkipTest will call t.Skipf() and stop execution
	testutil.ShouldSkipTest(t, testName, majorVersion)

	// Read and execute the pgdump.sql file
	pgdumpPath := fmt.Sprintf("../../testdata/dump/%s/pgdump.sql", testDataDir)
	pgdumpContent, err := os.ReadFile(pgdumpPath)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", pgdumpPath, err)
	}

	// Execute the SQL to create the schema
	_, err = conn.ExecContext(ctx, string(pgdumpContent))
	if err != nil {
		t.Fatalf("Failed to execute pgdump.sql: %v", err)
	}

	// Create dump configuration
	config := &DumpConfig{
		Host:      host,
		Port:      port,
		DB:        dbname,
		User:      user,
		Password:  password,
		Schema:    "public",
		MultiFile: false,
		File:      "",
	}

	// Execute pgschema dump
	actualOutput, err := ExecuteDump(config)
	if err != nil {
		t.Fatalf("Dump command failed: %v", err)
	}

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
	// Setup PostgreSQL
	embeddedPG := testutil.SetupPostgres(t)
	defer embeddedPG.Stop()

	// Connect to database
	conn, host, port, dbname, user, password := testutil.ConnectToPostgres(t, embeddedPG)
	defer conn.Close()

	// Read the tenant SQL that will be loaded into all schemas
	tenantSQL, err := os.ReadFile(fmt.Sprintf("../../testdata/dump/%s/tenant.sql", testDataDir))
	if err != nil {
		t.Fatalf("Failed to read tenant.sql: %v", err)
	}

	// Load utility functions (if util.sql exists)
	utilPath := fmt.Sprintf("../../testdata/dump/%s/util.sql", testDataDir)
	if utilSQL, err := os.ReadFile(utilPath); err == nil {
		_, err = conn.Exec(string(utilSQL))
		if err != nil {
			t.Fatalf("Failed to load utility functions from util.sql: %v", err)
		}
	} else if !os.IsNotExist(err) {
		t.Fatalf("Failed to read util.sql: %v", err)
	}

	// Create two tenant schemas (public already exists)
	schemas := []string{"public", "tenant1", "tenant2"}
	for _, schema := range schemas[1:] { // Skip public as it already exists
		_, err = conn.Exec(fmt.Sprintf("CREATE SCHEMA %s", schema))
		if err != nil {
			t.Fatalf("Failed to create schema %s: %v", schema, err)
		}
	}

	// Load the tenant SQL into all three schemas
	for _, schema := range schemas {
		// Set search path to target schema only
		quotedSchema := ir.QuoteIdentifier(schema)
		_, err = conn.Exec(fmt.Sprintf("SET search_path TO %s", quotedSchema))
		if err != nil {
			t.Fatalf("Failed to set search path to %s: %v", schema, err)
		}

		// Execute the SQL - objects will be created in the target schema
		_, err = conn.Exec(string(tenantSQL))
		if err != nil {
			t.Fatalf("Failed to load SQL into schema %s: %v", schema, err)
		}
	}

	// Dump all three schemas using pgschema dump command
	var dumps []string
	for _, schemaName := range schemas {
		// Create dump configuration for this schema
		config := &DumpConfig{
			Host:      host,
			Port:      port,
			DB:        dbname,
			User:      user,
			Password:  password,
			Schema:    schemaName,
			MultiFile: false,
			File:      "",
		}

		// Execute pgschema dump
		actualOutput, err := ExecuteDump(config)
		if err != nil {
			t.Fatalf("Dump command failed for schema %s: %v", schemaName, err)
		}
		dumps = append(dumps, actualOutput)
	}

	// Read expected output
	expectedBytes, err := os.ReadFile(fmt.Sprintf("../../testdata/dump/%s/pgschema.sql", testDataDir))
	if err != nil {
		t.Fatalf("Failed to read expected output: %v", err)
	}
	expected := string(expectedBytes)

	// Compare all dumps against expected output
	for i, dump := range dumps {
		schemaName := schemas[i]

		// Use shared comparison function
		compareSchemaOutputs(t, dump, expected, fmt.Sprintf("%s_%s", testDataDir, schemaName))
	}

	// Also compare all dumps with each other - they should be identical
	for i := 1; i < len(dumps); i++ {
		compareSchemaOutputs(t, dumps[0], dumps[i], fmt.Sprintf("%s_%s_vs_%s", testDataDir, schemas[0], schemas[i]))
	}
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
	}
}
