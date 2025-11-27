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
	"regexp"
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

func TestDumpCommand_Issue78ConstraintNotValidAndQuoting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "issue_78_constraint_not_valid_and_quoting")
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

func TestDumpCommand_Issue125FunctionDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "issue_125_function_default")
}

func TestDumpCommand_Issue133IndexSort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "issue_133_index_sort")
}

func runExactMatchTest(t *testing.T, testDataDir string) {
	runExactMatchTestWithContext(t, context.Background(), testDataDir)
}

func runExactMatchTestWithContext(t *testing.T, ctx context.Context, testDataDir string) {
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
	testutil.ShouldSkipTest(t, t.Name(), majorVersion)

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

// TestDumpCommand_QuoteAll validates the --quote-all flag behavior
func TestDumpCommand_QuoteAll(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	runQuoteAllTest(t, "quote_all_test")
}

// runQuoteAllTest validates that the --quote-all flag correctly quotes all identifiers
func runQuoteAllTest(t *testing.T, testDataDir string) {
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
	testutil.ShouldSkipTest(t, t.Name(), majorVersion)

	// Read and execute the pgdump.sql file
	pgdumpPath := fmt.Sprintf("../../testdata/dump/%s/pgdump.sql", testDataDir)
	pgdumpContent, err := os.ReadFile(pgdumpPath)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", pgdumpPath, err)
	}

	// Execute the SQL to create the schema
	_, err = conn.ExecContext(context.Background(), string(pgdumpContent))
	if err != nil {
		t.Fatalf("Failed to execute pgdump.sql: %v", err)
	}

	// Test 1: Dump without --quote-all (normal behavior)
	configNormal := &DumpConfig{
		Host:      host,
		Port:      port,
		DB:        dbname,
		User:      user,
		Password:  password,
		Schema:    "public",
		MultiFile: false,
		File:      "",
		QuoteAll:  false,
	}

	normalOutput, err := ExecuteDump(configNormal)
	if err != nil {
		t.Fatalf("Dump command failed without quote-all: %v", err)
	}

	// Test 2: Dump with --quote-all (all identifiers quoted)
	configQuoteAll := &DumpConfig{
		Host:      host,
		Port:      port,
		DB:        dbname,
		User:      user,
		Password:  password,
		Schema:    "public",
		MultiFile: false,
		File:      "",
		QuoteAll:  true,
	}

	quoteAllOutput, err := ExecuteDump(configQuoteAll)
	if err != nil {
		t.Fatalf("Dump command failed with quote-all: %v", err)
	}

	// Validate quote-all behavior
	validateQuoteAllBehavior(t, normalOutput, quoteAllOutput, testDataDir)
}

// validateQuoteAllBehavior verifies that --quote-all produces correctly quoted output
func validateQuoteAllBehavior(t *testing.T, normalOutput, quoteAllOutput, testName string) {
	// Split outputs into lines for analysis
	normalLines := strings.Split(normalOutput, "\n")
	quoteAllLines := strings.Split(quoteAllOutput, "\n")

	// Both outputs should have the same number of lines
	if len(normalLines) != len(quoteAllLines) {
		t.Fatalf("Different number of lines - Normal: %d, QuoteAll: %d", len(normalLines), len(quoteAllLines))
	}

	// Track identifiers that should be quoted in normal mode vs quote-all mode
	var normalQuotedIdentifiers []string
	var quoteAllQuotedIdentifiers []string

	// Regular expression to find quoted identifiers
	quotedIdentifierRegex := `"([^"]+)"`

	for i, normalLine := range normalLines {
		quoteAllLine := quoteAllLines[i]

		// Skip comment lines and empty lines
		if strings.HasPrefix(strings.TrimSpace(normalLine), "--") || strings.TrimSpace(normalLine) == "" {
			continue
		}

		// Extract quoted identifiers from both outputs
		normalMatches := regexp.MustCompile(quotedIdentifierRegex).FindAllStringSubmatch(normalLine, -1)
		quoteAllMatches := regexp.MustCompile(quotedIdentifierRegex).FindAllStringSubmatch(quoteAllLine, -1)

		for _, match := range normalMatches {
			normalQuotedIdentifiers = append(normalQuotedIdentifiers, match[1])
		}

		for _, match := range quoteAllMatches {
			quoteAllQuotedIdentifiers = append(quoteAllQuotedIdentifiers, match[1])
		}
	}

	// Validate expectations:
	// 1. Quote-all mode should have more quoted identifiers than normal mode
	if len(quoteAllQuotedIdentifiers) <= len(normalQuotedIdentifiers) {
		t.Errorf("Quote-all mode should have more quoted identifiers. Normal: %d, QuoteAll: %d",
			len(normalQuotedIdentifiers), len(quoteAllQuotedIdentifiers))
	}

	// 2. All identifiers that were quoted in normal mode should also be quoted in quote-all mode
	normalQuotedSet := make(map[string]bool)
	for _, id := range normalQuotedIdentifiers {
		normalQuotedSet[id] = true
	}

	quoteAllQuotedSet := make(map[string]bool)
	for _, id := range quoteAllQuotedIdentifiers {
		quoteAllQuotedSet[id] = true
	}

	for identifier := range normalQuotedSet {
		if !quoteAllQuotedSet[identifier] {
			t.Errorf("Identifier '%s' was quoted in normal mode but not in quote-all mode", identifier)
		}
	}

	// 3. Verify specific expected behaviors
	// Note: Currently only table and column names support quote-all. Other objects (indexes, sequences, views, functions) are not yet implemented
	expectedNormalQuoted := []string{"order", "MixedCase", "ID", "FirstName", "LastName", "SpecialColumn", "Index_Order_Status", "MixedCase_pkey"}
	expectedQuoteAllOnly := []string{"users", "id", "first_name", "last_name", "email", "created_at", "user_id", "total_amount", "status"}

	// Check that expected identifiers are quoted in normal mode
	for _, identifier := range expectedNormalQuoted {
		if !normalQuotedSet[identifier] {
			t.Errorf("Expected identifier '%s' to be quoted in normal mode, but it wasn't", identifier)
		}
	}

	// Check that additional identifiers are quoted only in quote-all mode
	for _, identifier := range expectedQuoteAllOnly {
		if normalQuotedSet[identifier] {
			t.Errorf("Identifier '%s' should not be quoted in normal mode", identifier)
		}
		if !quoteAllQuotedSet[identifier] {
			t.Errorf("Identifier '%s' should be quoted in quote-all mode", identifier)
		}
	}

	// Write outputs to files for debugging if test fails
	if t.Failed() {
		normalFilename := fmt.Sprintf("%s_normal.sql", testName)
		os.WriteFile(normalFilename, []byte(normalOutput), 0644)

		quoteAllFilename := fmt.Sprintf("%s_quote_all.sql", testName)
		os.WriteFile(quoteAllFilename, []byte(quoteAllOutput), 0644)

		t.Logf("Outputs written to %s and %s for debugging", normalFilename, quoteAllFilename)
		t.Logf("Normal quoted identifiers: %v", normalQuotedIdentifiers)
		t.Logf("Quote-all quoted identifiers: %v", quoteAllQuotedIdentifiers)
	}
}
