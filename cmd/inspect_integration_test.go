package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestInspectCommand_ExactMatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testCases := []struct {
		name     string
		testData string
	}{
		{
			name:     "employee",
			testData: "employee",
		},
		{
			name:     "bytebase",
			testData: "bytebase",
		},
		// Add more test cases as needed:
		// {
		// 	name:     "sourcegraph",
		// 	testData: "sourcegraph",
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runExactMatchTest(t, tc.testData)
		})
	}
}

func runExactMatchTest(t *testing.T, testDataDir string) {
	ctx := context.Background()

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

	// Set DSN for inspect command
	originalDSN := dsn
	dsn = testDSN
	defer func() { dsn = originalDSN }()

	// Capture output by redirecting stdout
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the inspect command
	setupLogger()
	err = runInspect(nil, nil)

	// Restore stdout
	w.Close()
	os.Stdout = originalStdout

	if err != nil {
		t.Fatalf("Inspect command failed: %v", err)
	}

	// Read the captured output
	output := make([]byte, 100000)
	n, _ := r.Read(output)
	actualOutput := string(output[:n])

	// Read expected output
	expectedPath := fmt.Sprintf("../testdata/%s/pgschema.sql", testDataDir)
	expectedContent, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", expectedPath, err)
	}
	expectedOutput := string(expectedContent)

	// Compare the outputs
	if actualOutput != expectedOutput {
		t.Errorf("Output does not match %s", expectedPath)
		t.Logf("Total lines - Actual: %d, Expected: %d", len(strings.Split(actualOutput, "\n")), len(strings.Split(expectedOutput, "\n")))

		// Write actual output to file for debugging only when test fails
		actualFilename := fmt.Sprintf("%s_actual.sql", testDataDir)
		if err := os.WriteFile(actualFilename, []byte(actualOutput), 0644); err != nil {
			t.Logf("Failed to write actual output file for debugging: %v", err)
		} else {
			t.Logf("Actual output written to %s for debugging", actualFilename)
		}
	} else {
		t.Logf("Success! Output matches %s exactly", expectedPath)
	}
}
