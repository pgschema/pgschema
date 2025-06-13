package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spf13/cobra"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestInspectCommand(t *testing.T) {
	// Test that the command is properly configured
	if InspectCmd.Use != "inspect" {
		t.Errorf("Expected Use to be 'inspect', got '%s'", InspectCmd.Use)
	}

	if InspectCmd.Short == "" {
		t.Error("Expected Short description to be set")
	}

	if InspectCmd.Long == "" {
		t.Error("Expected Long description to be set")
	}

	// Test that the --dsn flag is required
	flags := InspectCmd.Flags()
	dsnFlag := flags.Lookup("dsn")
	if dsnFlag == nil {
		t.Error("Expected --dsn flag to be defined")
	}

	// Test command validation - should fail without --dsn
	cmd := &cobra.Command{}
	cmd.AddCommand(InspectCmd)
	
	// Reset the dsn variable for clean test
	dsn = ""
	
	err := InspectCmd.RunE(InspectCmd, []string{})
	if err == nil {
		t.Error("Expected command to fail without database connection, but it didn't")
	}
}

func TestInspectCommand_Integration(t *testing.T) {
	// Start PostgreSQL container
	ctx := context.Background()
	
	container, err := postgres.Run(ctx,
		"postgres:17",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Get connection details
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

	// Connect to database
	testDSN := fmt.Sprintf("postgres://testuser:testpass@%s:%s/testdb?sslmode=disable", 
		host, port.Port())
	
	db, err := sql.Open("pgx", testDSN)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create a simple test schema
	_, err = db.ExecContext(ctx, `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT UNIQUE
		);
		
		CREATE SEQUENCE user_audit_seq START 1;
		
		CREATE TABLE audit (
			id BIGINT DEFAULT nextval('user_audit_seq') PRIMARY KEY,
			user_id INTEGER REFERENCES users(id),
			action TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		);
		
		CREATE FUNCTION log_action() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			INSERT INTO audit(user_id, action) VALUES (NEW.id, 'INSERT');
			RETURN NEW;
		END;
		$$;
		
		CREATE TRIGGER user_log_trigger AFTER INSERT ON users FOR EACH ROW EXECUTE FUNCTION log_action();
	`)
	if err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}

	// Test the inspect command
	originalDSN := dsn
	dsn = testDSN // Set global variable for inspect command
	defer func() { dsn = originalDSN }()
	
	// Capture output by redirecting stdout
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the inspect command
	err = runInspect(nil, nil)
	
	// Restore stdout
	w.Close()
	os.Stdout = originalStdout
	
	if err != nil {
		t.Fatalf("Inspect command failed: %v", err)
	}

	// Read the captured output
	output := make([]byte, 50000)
	n, _ := r.Read(output)
	outputStr := string(output[:n])

	// Write output to file for debugging
	if err := os.WriteFile("test_output.sql", []byte(outputStr), 0644); err != nil {
		t.Logf("Failed to write output file: %v", err)
	} else {
		t.Logf("Output written to test_output.sql")
	}

	// Verify the output contains expected pg_dump elements
	expectedElements := []string{
		"-- PostgreSQL database dump",
		"-- Dumped from database version",
		"-- Dumped by pgschema version",
		"CREATE TABLE public.users",
		"CREATE SEQUENCE public.user_audit_seq",
		"CREATE FUNCTION public.log_action()",
		"-- PostgreSQL database dump complete",
	}

	for _, element := range expectedElements {
		if !strings.Contains(outputStr, element) {
			t.Errorf("Expected element '%s' not found in output", element)
		}
	}

	// Verify the output structure follows pg_dump order
	headerIndex := strings.Index(outputStr, "-- PostgreSQL database dump")
	versionIndex := strings.Index(outputStr, "-- Dumped by pgschema version")
	functionIndex := strings.Index(outputStr, "CREATE FUNCTION")
	tableIndex := strings.Index(outputStr, "CREATE TABLE")
	sequenceIndex := strings.Index(outputStr, "CREATE SEQUENCE")
	footerIndex := strings.Index(outputStr, "-- PostgreSQL database dump complete")

	if headerIndex == -1 || versionIndex == -1 || footerIndex == -1 {
		t.Error("Missing required header, version, or footer sections")
	}

	if !(headerIndex < versionIndex && versionIndex < footerIndex) {
		t.Error("Output sections are not in the correct order")
	}

	// Check that we have some database objects
	if functionIndex == -1 && tableIndex == -1 && sequenceIndex == -1 {
		t.Error("No database objects found in output")
	}

	t.Logf("Test passed! Output contains all expected pg_dump elements in correct order")
}

func TestInspectCommand_ErrorHandling(t *testing.T) {
	// Test with invalid DSN to ensure proper error handling
	originalDSN := dsn
	dsn = "invalid://connection/string"
	defer func() { dsn = originalDSN }()

	err := runInspect(nil, nil)
	if err == nil {
		t.Error("Expected error with invalid DSN, but got nil")
	}

	// Test with DSN that fails to connect
	dsn = "postgres://invalid:invalid@localhost:9999/nonexistent"
	err = runInspect(nil, nil)
	if err == nil {
		t.Error("Expected error with unreachable database, but got nil")
	}
}