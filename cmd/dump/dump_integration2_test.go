package dump

import (
	"bytes"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pgschema/pgschema/testutil"
)

var generateDump = flag.Bool("generate-dump", false, "generate expected test output files instead of comparing")

// TestDumpIntegration tests the complete dump CLI workflow using test cases
// from testdata/diff/. This test exercises the full end-to-end dump command that
// users will execute, providing comprehensive validation of the dump workflow.
//
// The test performs these actions for each test case:
// 1. Apply new.sql to database → initialize target state
// 2. Run dump command → generate schema output
// 3. Compare with dump.sql or generate it if --generate flag is used
//
// Test filtering can be controlled using the PGSCHEMA_TEST_FILTER environment variable:
//
// Examples:
//
//	# Run all tests under create_table/ (directory prefix with slash)
//	PGSCHEMA_TEST_FILTER="create_table/" go test -v ./cmd/dump -run TestDumpIntegration
//
//	# Run tests under create_table/ that start with "add_column"
//	PGSCHEMA_TEST_FILTER="create_table/add_column" go test -v ./cmd/dump -run TestDumpIntegration
//
//	# Run a specific test
//	PGSCHEMA_TEST_FILTER="create_table/add_column_identity" go test -v ./cmd/dump -run TestDumpIntegration
//
//	# Run all migrate tests
//	PGSCHEMA_TEST_FILTER="migrate/" go test -v ./cmd/dump -run TestDumpIntegration
func TestDumpIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDataRoot := "../../testdata/diff"

	// Start a single PostgreSQL container for all test cases
	container := testutil.SetupPostgresContainerWithDB(ctx, t, "postgres", "testuser", "testpass")
	defer container.Terminate(ctx, t)

	containerHost := container.Host
	portMapped := container.Port

	// Get test filter from environment variable
	testFilter := os.Getenv("PGSCHEMA_TEST_FILTER")

	// Collect all test cases first
	var testCases []dumpTestCase
	err := filepath.Walk(testDataRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory or if it's the root or category directories
		if !info.IsDir() || path == testDataRoot {
			return nil
		}

		// Skip category directories (e.g., create_table/, create_index/) - only process leaf directories
		relPath, _ := filepath.Rel(testDataRoot, path)
		pathDepth := len(strings.Split(relPath, string(filepath.Separator)))
		if pathDepth == 1 {
			return nil // Skip category directories
		}

		// Check if this directory contains the required test files
		newFile := filepath.Join(path, "new.sql")
		dumpFile := filepath.Join(path, "dump.sql")

		// Check for required input file (always required)
		if _, err := os.Stat(newFile); os.IsNotExist(err) {
			return fmt.Errorf("missing required file: %s", newFile)
		}

		// Check for output file when not generating
		if !*generateDump {
			if _, err := os.Stat(dumpFile); os.IsNotExist(err) {
				return fmt.Errorf("missing required file: %s (use --generate to create)", dumpFile)
			}
		}

		// Apply test filter if provided
		if testFilter != "" && !matchesFilter(relPath, testFilter) {
			return nil
		}

		// Get relative path for test name
		testName := strings.ReplaceAll(relPath, string(filepath.Separator), "_")

		testCases = append(testCases, dumpTestCase{
			name:     testName,
			newFile:  newFile,
			dumpFile: dumpFile,
		})

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to walk test data directory: %v", err)
	}

	// Check if filter was provided but no tests matched
	if testFilter != "" && len(testCases) == 0 {
		t.Fatalf("No test cases found matching filter: %s", testFilter)
	}

	// Run all test cases using the shared container
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runDumpIntegrationTest(t, ctx, containerHost, portMapped, tc)
		})
	}
}

type dumpTestCase struct {
	name     string
	newFile  string
	dumpFile string
}

// runDumpIntegrationTest executes a single dump integration test case with test-specific database
func runDumpIntegrationTest(t *testing.T, ctx context.Context, containerHost string, portMapped int, tc dumpTestCase) {
	// Create a unique database name for this test case (replace invalid chars)
	dbName := "test_" + strings.ReplaceAll(strings.ReplaceAll(tc.name, "/", "_"), "-", "_")
	// PostgreSQL identifiers are limited to 63 characters
	if len(dbName) > 63 {
		dbName = dbName[:63]
	}

	t.Logf("=== DUMP INTEGRATION TEST: %s → %s (DB: %s) ===", filepath.Base(tc.newFile), filepath.Base(tc.dumpFile), dbName)

	// Create test-specific database
	if err := createDumpDatabase(ctx, containerHost, portMapped, dbName); err != nil {
		t.Fatalf("Failed to create test database %s: %v", dbName, err)
	}

	// STEP 1: Apply new.sql to initialize database state
	t.Logf("--- Applying new.sql to initialize database state ---")
	newContent, err := os.ReadFile(tc.newFile)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", tc.newFile, err)
	}

	// Execute new.sql if it has content
	if len(strings.TrimSpace(string(newContent))) > 0 {
		if err := executeDumpSQL(ctx, containerHost, portMapped, dbName, string(newContent)); err != nil {
			t.Fatalf("Failed to execute new.sql: %v", err)
		}
		t.Logf("Applied new.sql to initialize database state")
	}

	// STEP 2: Test dump command
	t.Logf("--- Testing dump command ---")
	testDumpOutput(t, containerHost, portMapped, dbName, tc.dumpFile)

	t.Logf("=== DUMP INTEGRATION TEST COMPLETED ===")
}

// testDumpOutput tests dump output against expected file
func testDumpOutput(t *testing.T, containerHost string, portMapped int, dbName, dumpFile string) {
	// Generate dump output
	dumpOutput, err := generateDumpOutput(containerHost, portMapped, dbName, "testuser", "testpass", "public")
	if err != nil {
		t.Fatalf("Failed to generate dump output: %v", err)
	}

	if *generateDump {
		// Generate mode: write actual output to expected file
		actualDumpStr := strings.ReplaceAll(dumpOutput, "\r\n", "\n")
		err := os.WriteFile(dumpFile, []byte(actualDumpStr), 0644)
		if err != nil {
			t.Fatalf("Failed to write expected dump output file %s: %v", dumpFile, err)
		}
		t.Logf("Generated expected dump output file %s", dumpFile)
	} else {
		// Compare mode: compare with expected file
		expectedDump, err := os.ReadFile(dumpFile)
		if err != nil {
			t.Fatalf("Failed to read expected dump output file %s: %v", dumpFile, err)
		}

		// Compare dump output
		expectedDumpStr := strings.ReplaceAll(string(expectedDump), "\r\n", "\n")
		actualDumpStr := strings.ReplaceAll(dumpOutput, "\r\n", "\n")

		// Normalize both outputs to ignore version differences
		normalizedActual := normalizeDumpOutput(actualDumpStr)
		normalizedExpected := normalizeDumpOutput(expectedDumpStr)

		if normalizedActual != normalizedExpected {
			t.Errorf("Dump output mismatch.\nExpected:\n%s\n\nActual:\n%s", normalizedExpected, normalizedActual)
			// Write actual output to file for easier comparison
			actualFile := strings.Replace(dumpFile, ".sql", "_actual.sql", 1)
			os.WriteFile(actualFile, []byte(actualDumpStr), 0644)
			t.Logf("Actual dump output written to %s", actualFile)
		}
	}
}

// generateDumpOutput generates dump output using the CLI dump command
func generateDumpOutput(hostVal string, portVal int, database, userVal, passwordVal, schemaVal string) (string, error) {
	// Store original command variables and restore them later
	originalHost := host
	originalPort := port
	originalDB := db
	originalUser := user
	originalPassword := password
	originalSchema := schema
	defer func() {
		host = originalHost
		port = originalPort
		db = originalDB
		user = originalUser
		password = originalPassword
		schema = originalSchema
	}()

	// Set command variables
	host = hostVal
	port = portVal
	db = database
	user = userVal
	password = passwordVal
	schema = schemaVal

	// Capture stdout by redirecting it temporarily
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w

	// Execute dump command in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- runDump(nil, nil)
	}()

	// Copy the output from the pipe to our buffer in a goroutine
	copyDone := make(chan struct{})
	go func() {
		defer close(copyDone)
		defer r.Close()
		buf.ReadFrom(r)
	}()

	// Wait for command to complete
	cmdErr := <-done

	// Close the writer to signal EOF to the reader
	w.Close()

	// Wait for the copy operation to complete
	<-copyDone

	// Restore stdout
	os.Stdout = oldStdout

	if cmdErr != nil {
		return "", cmdErr
	}

	return buf.String(), nil
}

// createDumpDatabase creates a test-specific database using the shared container
func createDumpDatabase(ctx context.Context, hostVal string, portVal int, dbName string) error {
	// Connect to postgres database to create the test database
	dsn := fmt.Sprintf("postgres://testuser:testpass@%s:%d/postgres?sslmode=disable", hostVal, portVal)
	dbConn, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres database: %v", err)
	}
	defer dbConn.Close()

	// Create the test database
	_, err = dbConn.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		return fmt.Errorf("failed to create database %s: %v", dbName, err)
	}

	return nil
}

// executeDumpSQL executes SQL statements in the specified database
func executeDumpSQL(ctx context.Context, hostVal string, portVal int, dbName string, sqlContent string) error {
	// Connect to the specific test database
	dsn := fmt.Sprintf("postgres://testuser:testpass@%s:%d/%s?sslmode=disable", hostVal, portVal, dbName)
	dbConn, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database %s: %v", dbName, err)
	}
	defer dbConn.Close()

	// Execute the SQL
	_, err = dbConn.ExecContext(ctx, sqlContent)
	if err != nil {
		return fmt.Errorf("failed to execute SQL in database %s: %v", dbName, err)
	}

	return nil
}

// normalizeDumpOutput removes version-specific lines for comparison
func normalizeDumpOutput(output string) string {
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

// matchesFilter checks if a relative path matches the given filter pattern
func matchesFilter(relPath, filter string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}

	// Handle directory prefix patterns with trailing slash
	if strings.HasSuffix(filter, "/") {
		// "create_table/" matches "create_table/add_column_identity"
		return strings.HasPrefix(relPath+"/", filter)
	}

	// Handle patterns with slash (both specific tests and prefix patterns)
	if strings.Contains(filter, "/") {
		// "create_table/add_column_identity" matches "create_table/add_column_identity"
		// "create_table/add_column" matches "create_table/add_column_identity"
		return strings.HasPrefix(relPath, filter)
	}

	// Fallback: check if filter is a substring of the path
	return strings.Contains(relPath, filter)
}
