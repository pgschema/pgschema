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

func TestDumpCommand_Issue78ConstraintNotValid(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "issue_78_constraint_not_valid")
}

func TestDumpCommand_Issue80IndexNameQuote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "issue_80_index_name_quote")
}

func TestDumpCommand_Issue82ViewLogicExpr(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "issue_82_view_logic_expr")
}

func TestDumpCommand_Issue83ExplicitConstraintName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "issue_83_explicit_constraint_name")
}

func runExactMatchTest(t *testing.T, testDataDir string) {
	runExactMatchTestWithContext(t, context.Background(), testDataDir)
}

func runExactMatchTestWithContext(t *testing.T, ctx context.Context, testDataDir string) {
	// Setup PostgreSQL container
	containerInfo := testutil.SetupTestPostgres(ctx, t)
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

	// Create dump configuration
	config := &DumpConfig{
		Host:      containerInfo.Host,
		Port:      containerInfo.Port,
		DB:        containerInfo.DBName,
		User:      containerInfo.User,
		Password:  containerInfo.Password,
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
	ctx := context.Background()

	// Setup PostgreSQL container
	containerInfo := testutil.SetupTestPostgres(ctx, t)
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
		quotedTenant := ir.QuoteIdentifier(tenant)
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

	// Dump both tenant schemas using pgschema dump command
	var dumps []string
	for _, tenantName := range tenants {
		// Create dump configuration for this tenant
		config := &DumpConfig{
			Host:      containerInfo.Host,
			Port:      containerInfo.Port,
			DB:        containerInfo.DBName,
			User:      containerInfo.User,
			Password:  containerInfo.Password,
			Schema:    tenantName,
			MultiFile: false,
			File:      "",
		}

		// Execute pgschema dump
		actualOutput, err := ExecuteDump(config)
		if err != nil {
			t.Fatalf("Dump command failed for tenant %s: %v", tenantName, err)
		}
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
