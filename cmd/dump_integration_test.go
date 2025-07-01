package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// normalizeVersionString replaces version strings with a normalized format for comparison
// This allows tests to pass when only the version info differs
func normalizeVersionString(content string) string {
	// Replace "Dumped by pgschema version X.Y.Z"
	re := regexp.MustCompile(`-- Dumped by pgschema version [^\n]+`)

	return re.ReplaceAllString(content, "-- Dumped by pgschema version NORMALIZED")
}

type containerInfo struct {
	container testcontainers.Container
	host      string
	port      int
	dsn       string
	conn      *sql.DB
}

func setupPostgresContainer(ctx context.Context, t *testing.T) *containerInfo {
	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:17",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}

	// Get connection string
	testDSN, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Connect to database
	conn, err := sql.Open("pgx", testDSN)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Get container connection details
	containerHost, err := postgresContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}
	containerPort, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

	return &containerInfo{
		container: postgresContainer,
		host:      containerHost,
		port:      containerPort.Int(),
		dsn:       testDSN,
		conn:      conn,
	}
}

func (ci *containerInfo) terminate(ctx context.Context, t *testing.T) {
	ci.conn.Close()
	if err := ci.container.Terminate(ctx); err != nil {
		t.Logf("Failed to terminate container: %v", err)
	}
}

func configureConnection(ci *containerInfo) {
	host = ci.host
	port = ci.port
	db = "testdb"
	user = "testuser"
	os.Setenv("PGPASSWORD", "testpass")
}

func restoreConnection(originalHost string, originalPort int, originalDb string, originalUser string, originalSchema string) {
	host = originalHost
	port = originalPort
	db = originalDb
	user = originalUser
	schema = originalSchema
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
	containerInfo := setupPostgresContainer(ctx, t)
	defer containerInfo.terminate(ctx, t)

	// Read and execute the pgdump.sql file
	pgdumpPath := fmt.Sprintf("../testdata/%s/pgdump.sql", testDataDir)
	pgdumpContent, err := os.ReadFile(pgdumpPath)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", pgdumpPath, err)
	}

	// Execute the SQL to create the schema
	_, err = containerInfo.conn.ExecContext(ctx, string(pgdumpContent))
	if err != nil {
		t.Fatalf("Failed to execute pgdump.sql: %v", err)
	}

	// Store original connection parameters and restore them later
	originalHost := host
	originalPort := port
	originalDb := db
	originalUser := user
	originalSchema := schema

	defer restoreConnection(originalHost, originalPort, originalDb, originalUser, originalSchema)

	// Configure connection parameters
	configureConnection(containerInfo)

	// Execute pgschema dump and capture output
	actualOutput := executePgSchemaDump(t, "")

	// Read expected output
	expectedPath := fmt.Sprintf("../testdata/%s/pgschema.sql", testDataDir)
	expectedContent, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", expectedPath, err)
	}
	expectedOutput := string(expectedContent)

	// Normalize version strings for comparison
	normalizedActual := normalizeVersionString(actualOutput)
	normalizedExpected := normalizeVersionString(expectedOutput)

	// Compare the normalized outputs
	if normalizedActual != normalizedExpected {
		t.Errorf("Output does not match %s", expectedPath)
		t.Logf("Total lines - Actual: %d, Expected: %d", len(strings.Split(actualOutput, "\n")), len(strings.Split(expectedOutput, "\n")))

		// Write actual output to file for debugging only when test fails
		actualFilename := fmt.Sprintf("%s_actual.sql", testDataDir)
		if err := os.WriteFile(actualFilename, []byte(actualOutput), 0644); err != nil {
			t.Logf("Failed to write actual output file for debugging: %v", err)
		} else {
			t.Logf("Actual output written to %s for debugging", actualFilename)
		}

		// Also write normalized versions for comparison
		normalizedActualFilename := fmt.Sprintf("%s_normalized_actual.sql", testDataDir)
		normalizedExpectedFilename := fmt.Sprintf("%s_normalized_expected.sql", testDataDir)
		os.WriteFile(normalizedActualFilename, []byte(normalizedActual), 0644)
		os.WriteFile(normalizedExpectedFilename, []byte(normalizedExpected), 0644)
		t.Logf("Normalized outputs written to %s and %s for debugging", normalizedActualFilename, normalizedExpectedFilename)
	} else {
		t.Logf("Success! Output matches %s (version differences ignored)", expectedPath)
	}
}

func runTenantSchemaTest(t *testing.T, testDataDir string) {
	ctx := context.Background()

	// Setup PostgreSQL container
	containerInfo := setupPostgresContainer(ctx, t)
	defer containerInfo.terminate(ctx, t)

	// Load public schema types first
	publicSQL, err := os.ReadFile(fmt.Sprintf("../testdata/%s/public.sql", testDataDir))
	if err != nil {
		t.Fatalf("Failed to read public.sql: %v", err)
	}

	_, err = containerInfo.conn.Exec(string(publicSQL))
	if err != nil {
		t.Fatalf("Failed to load public types: %v", err)
	}

	// Create two tenant schemas
	tenants := []string{"tenant1", "tenant2"}
	for _, tenant := range tenants {
		_, err = containerInfo.conn.Exec(fmt.Sprintf("CREATE SCHEMA %s", tenant))
		if err != nil {
			t.Fatalf("Failed to create schema %s: %v", tenant, err)
		}
	}

	// Read the tenant SQL
	tenantSQL, err := os.ReadFile(fmt.Sprintf("../testdata/%s/tenant.sql", testDataDir))
	if err != nil {
		t.Fatalf("Failed to read tenant.sql: %v", err)
	}

	// Load the SQL into both tenant schemas
	for _, tenant := range tenants {
		// Set search path to include public for the types, but target schema first
		_, err = containerInfo.conn.Exec(fmt.Sprintf("SET search_path TO %s, public", tenant))
		if err != nil {
			t.Fatalf("Failed to set search path to %s: %v", tenant, err)
		}

		// Execute the SQL
		_, err = containerInfo.conn.Exec(string(tenantSQL))
		if err != nil {
			t.Fatalf("Failed to load SQL into schema %s: %v", tenant, err)
		}
	}

	// Save original command variables
	originalHost := host
	originalPort := port
	originalDb := db
	originalUser := user
	originalSchema := schema

	defer restoreConnection(originalHost, originalPort, originalDb, originalUser, originalSchema)

	// Dump both tenant schemas using pgschema dump command
	var dumps []string
	for _, tenantName := range tenants {
		// Set connection parameters for this specific tenant dump
		configureConnection(containerInfo)
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

		// Compare with expected output
		if dump != expected {
			// Save actual dump for debugging
			debugFile := fmt.Sprintf("debug_%s_dump.sql", tenantName)
			if err := os.WriteFile(debugFile, []byte(dump), 0644); err != nil {
				t.Logf("Failed to write debug file %s: %v", debugFile, err)
			} else {
				t.Logf("Saved %s dump to %s", tenantName, debugFile)
			}

			// Find first difference
			actualLines := strings.Split(dump, "\n")
			expectedLines := strings.Split(expected, "\n")

			for j := 0; j < len(actualLines) && j < len(expectedLines); j++ {
				if actualLines[j] != expectedLines[j] {
					t.Errorf("First difference at line %d in %s:\nActual:   %s\nExpected: %s",
						j+1, tenantName, actualLines[j], expectedLines[j])
					break
				}
			}

			if len(actualLines) != len(expectedLines) {
				t.Errorf("Different number of lines in %s - Actual: %d, Expected: %d",
					tenantName, len(actualLines), len(expectedLines))
			}

			t.Errorf("Dump from %s does not match expected output in testdata/%s/pgschema.sql", tenantName, testDataDir)
		}
	}
}
