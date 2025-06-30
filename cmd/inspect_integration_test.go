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

func TestInspectCommand_Employee(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "employee")
}

func TestInspectCommand_Sakila(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "sakila")
}

func TestInspectCommand_Bytebase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	runExactMatchTest(t, "bytebase")
}

func runExactMatchTest(t *testing.T, testDataDir string) {
	runExactMatchTestWithContext(t, context.Background(), testDataDir)
}

func runExactMatchTestWithContext(t *testing.T, ctx context.Context, testDataDir string) {

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
	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Get connection string
	testDSN, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Connect to database and load schema
	db, err := sql.Open("pgx", testDSN)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Read and execute the pgdump.sql file
	pgdumpPath := fmt.Sprintf("../testdata/%s/pgdump.sql", testDataDir)
	pgdumpContent, err := os.ReadFile(pgdumpPath)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", pgdumpPath, err)
	}

	// Execute the SQL to create the schema
	_, err = db.ExecContext(ctx, string(pgdumpContent))
	if err != nil {
		t.Fatalf("Failed to execute pgdump.sql: %v", err)
	}

	// Parse the connection string to extract individual components
	// testDSN format: "postgres://testuser:testpass@host:port/testdb?sslmode=disable"
	// We need to set the individual flag variables instead of using dsn
	originalHost := host
	originalPort := port
	originalDbname := dbname
	originalUsername := username

	defer func() {
		host = originalHost
		port = originalPort
		dbname = originalDbname
		username = originalUsername
	}()

	// Extract connection details from container
	containerHost, err := postgresContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}
	containerPort, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

	// Set connection parameters
	host = containerHost
	port = containerPort.Int()
	dbname = "testdb"
	username = "testuser"

	// Set password via environment variable
	os.Setenv("PGPASSWORD", "testpass")

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

	// Run the inspect command
	setupLogger()
	err = runInspect(nil, nil)

	// Close write end and restore stdout
	w.Close()
	os.Stdout = originalStdout

	if err != nil {
		t.Fatalf("Inspect command failed: %v", err)
	}

	// Wait for reading to complete
	<-done
	if readErr != nil {
		t.Fatalf("Failed to read captured output: %v", readErr)
	}

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
