//go:build integration
// +build integration

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

func TestDumpTenantSchemasIdentical(t *testing.T) {
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

	// Load public schema types first
	publicSQL, err := os.ReadFile("../testdata/tenant/public.sql")
	if err != nil {
		t.Fatalf("Failed to read public.sql: %v", err)
	}

	_, err = conn.Exec(string(publicSQL))
	if err != nil {
		t.Fatalf("Failed to load public types: %v", err)
	}

	// Create two tenant schemas
	tenants := []string{"tenant1", "tenant2"}
	for _, tenant := range tenants {
		_, err = conn.Exec(fmt.Sprintf("CREATE SCHEMA %s", tenant))
		if err != nil {
			t.Fatalf("Failed to create schema %s: %v", tenant, err)
		}
	}

	// Read the tenant SQL
	tenantSQL, err := os.ReadFile("../testdata/tenant/tenant.sql")
	if err != nil {
		t.Fatalf("Failed to read tenant.sql: %v", err)
	}

	// Load the SQL into both tenant schemas
	for _, tenant := range tenants {
		// Set search path to include public for the types, but target schema first
		_, err = conn.Exec(fmt.Sprintf("SET search_path TO %s, public", tenant))
		if err != nil {
			t.Fatalf("Failed to set search path to %s: %v", tenant, err)
		}

		// Execute the SQL
		_, err = conn.Exec(string(tenantSQL))
		if err != nil {
			t.Fatalf("Failed to load SQL into schema %s: %v", tenant, err)
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

	// Dump both tenant schemas
	var dumps []string
	for _, tenantName := range tenants {
		schema = tenantName

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
			t.Fatalf("Failed to dump schema %s: %v", tenantName, err)
		}

		// Restore stdout and read output
		w.Close()
		os.Stdout = originalStdout

		output, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("Failed to read output for schema %s: %v", tenantName, err)
		}

		dumps = append(dumps, string(output))
	}

	// Compare the two dumps
	if dumps[0] != dumps[1] {
		// Save dumps for debugging
		for i, dump := range dumps {
			debugFile := fmt.Sprintf("debug_tenant%d_dump.sql", i+1)
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
				t.Errorf("First difference at line %d:\nTenant1: %s\nTenant2: %s", 
					i+1, lines1[i], lines2[i])
				break
			}
		}

		t.Errorf("Dumps from tenant1 and tenant2 are not identical")
	}
}

func TestDumpTenantSchemaQualifiers(t *testing.T) {
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

	// Load public schema types
	publicSQL, err := os.ReadFile("../testdata/tenant/public.sql")
	if err != nil {
		t.Fatalf("Failed to read public.sql: %v", err)
	}

	_, err = conn.Exec(string(publicSQL))
	if err != nil {
		t.Fatalf("Failed to load public types: %v", err)
	}

	// Create tenant schema
	_, err = conn.Exec("CREATE SCHEMA company1")
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Load test data into tenant schema
	_, err = conn.Exec("SET search_path TO company1, public")
	if err != nil {
		t.Fatalf("Failed to set search path: %v", err)
	}

	tenantSQL, err := os.ReadFile("../testdata/tenant/tenant.sql")
	if err != nil {
		t.Fatalf("Failed to read tenant.sql: %v", err)
	}

	_, err = conn.Exec(string(tenantSQL))
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
	schema = "company1"
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

	// Check that the dump does not contain tenant schema qualifiers
	// but may contain public schema qualifiers for the types
	if strings.Contains(dumpContent, "company1.") {
		// Count occurrences for debugging
		count := strings.Count(dumpContent, "company1.")
		t.Errorf("Dump contains %d tenant schema qualifiers (company1.), but should not contain any", count)
		
		// Show some examples
		lines := strings.Split(dumpContent, "\n")
		examples := 0
		for i, line := range lines {
			if strings.Contains(line, "company1.") && examples < 5 {
				t.Logf("Line %d: %s", i+1, line)
				examples++
			}
		}
		
		// Save dump for debugging
		if err := os.WriteFile("debug_tenant_qualifier_dump.sql", output, 0644); err != nil {
			t.Logf("Failed to write debug file: %v", err)
		} else {
			t.Logf("Saved dump to debug_tenant_qualifier_dump.sql")
		}
	}

	// Check if public schema qualifiers are preserved (they should be)
	if !strings.Contains(dumpContent, "public.") {
		t.Logf("Note: dump does not contain public schema qualifiers for types, which may be expected depending on implementation")
	}
}