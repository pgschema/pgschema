package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestDumpIdenticalAcrossSchemas(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

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

	// Get connection details
	testDSN, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Connect to database
	conn, err := sql.Open("pgx", testDSN)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close()

	// Create two schemas
	schemas := []string{"schema1", "schema2"}
	for _, schemaName := range schemas {
		_, err = conn.Exec(fmt.Sprintf("CREATE SCHEMA %s", schemaName))
		if err != nil {
			t.Fatalf("Failed to create schema %s: %v", schemaName, err)
		}
	}

	// Read the employee raw SQL
	rawSQL, err := os.ReadFile("../testdata/employee/raw.sql")
	if err != nil {
		t.Fatalf("Failed to read raw.sql: %v", err)
	}

	// Load the SQL into both schemas
	for _, schemaName := range schemas {
		// Set search path to the target schema
		_, err = conn.Exec(fmt.Sprintf("SET search_path TO %s", schemaName))
		if err != nil {
			t.Fatalf("Failed to set search path to %s: %v", schemaName, err)
		}

		// Execute the SQL
		_, err = conn.Exec(string(rawSQL))
		if err != nil {
			t.Fatalf("Failed to load SQL into schema %s: %v", schemaName, err)
		}
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

	// Save original command variables
	originalHost := host
	originalPort := port
	originalDb := db
	originalUser := user
	originalSchema := schema

	defer func() {
		host = originalHost
		port = originalPort
		db = originalDb
		user = originalUser
		schema = originalSchema
	}()

	// Set connection parameters
	host = containerHost
	port = containerPort.Int()
	db = "testdb"
	user = "testuser"
	os.Setenv("PGPASSWORD", "testpass")

	// Dump both schemas
	var dumps []string
	for _, schemaName := range schemas {
		schema = schemaName

		// Capture output
		originalStdout := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("Failed to create pipe: %v", err)
		}
		os.Stdout = w

		// Run dump
		err = runDump(nil, nil)
		if err != nil {
			w.Close()
			os.Stdout = originalStdout
			t.Fatalf("Failed to dump schema %s: %v", schemaName, err)
		}

		// Restore stdout and read output
		w.Close()
		os.Stdout = originalStdout

		output, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("Failed to read output for schema %s: %v", schemaName, err)
		}

		dumps = append(dumps, string(output))
	}

	// Compare the two dumps
	if dumps[0] != dumps[1] {
		// Save dumps for debugging
		for i, dump := range dumps {
			debugFile := fmt.Sprintf("debug_schema%d_dump.sql", i+1)
			if err := os.WriteFile(debugFile, []byte(dump), 0644); err != nil {
				t.Logf("Failed to write debug file %s: %v", debugFile, err)
			} else {
				t.Logf("Saved dump to %s", debugFile)
			}
		}

		// Find first difference
		lines1 := strings.Split(dumps[0], "\n")
		lines2 := strings.Split(dumps[1], "\n")

		for i := 0; i < len(lines1) && i < len(lines2); i++ {
			if lines1[i] != lines2[i] {
				t.Errorf("First difference at line %d:\nSchema1: %s\nSchema2: %s",
					i+1, lines1[i], lines2[i])
				break
			}
		}

		t.Errorf("Dumps from schema1 and schema2 are not identical")
	}
}

func TestDumpSchemaQualifier(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

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

	// Get connection details
	testDSN, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Connect to database
	conn, err := sql.Open("pgx", testDSN)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close()

	// Create custom schema
	_, err = conn.Exec("CREATE SCHEMA myschema")
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Load test data into custom schema
	_, err = conn.Exec("SET search_path TO myschema")
	if err != nil {
		t.Fatalf("Failed to set search path: %v", err)
	}

	rawSQL, err := os.ReadFile("../testdata/employee/raw.sql")
	if err != nil {
		t.Fatalf("Failed to read raw.sql: %v", err)
	}

	_, err = conn.Exec(string(rawSQL))
	if err != nil {
		t.Fatalf("Failed to load SQL: %v", err)
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

	// Save original command variables
	originalHost := host
	originalPort := port
	originalDb := db
	originalUser := user
	originalSchema := schema

	defer func() {
		host = originalHost
		port = originalPort
		db = originalDb
		user = originalUser
		schema = originalSchema
	}()

	// Set connection parameters
	host = containerHost
	port = containerPort.Int()
	db = "testdb"
	user = "testuser"
	schema = "myschema"
	os.Setenv("PGPASSWORD", "testpass")

	// Capture output
	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	// Run dump
	err = runDump(nil, nil)
	if err != nil {
		w.Close()
		os.Stdout = originalStdout
		t.Fatalf("Failed to dump: %v", err)
	}

	// Restore stdout and read output
	w.Close()
	os.Stdout = originalStdout

	output, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	dumpContent := string(output)

	// Check that the dump does not contain schema qualifiers
	// This test will initially fail, showing that schema qualifiers are present
	if strings.Contains(dumpContent, "myschema.") {
		// Count occurrences for debugging
		count := strings.Count(dumpContent, "myschema.")
		t.Errorf("Dump contains %d schema qualifiers (myschema.), but should not contain any", count)

		// Show some examples
		lines := strings.Split(dumpContent, "\n")
		examples := 0
		for i, line := range lines {
			if strings.Contains(line, "myschema.") && examples < 5 {
				t.Logf("Line %d: %s", i+1, line)
				examples++
			}
		}

		// Save dump for debugging
		if err := os.WriteFile("debug_schema_qualifier_dump.sql", output, 0644); err != nil {
			t.Logf("Failed to write debug file: %v", err)
		} else {
			t.Logf("Saved dump to debug_schema_qualifier_dump.sql")
		}
	}
}
