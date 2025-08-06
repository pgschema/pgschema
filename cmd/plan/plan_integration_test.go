package plan

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pgschema/pgschema/testutil"
	"github.com/spf13/cobra"
)

func TestPlanCommand_DatabaseIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	var err error

	// Start PostgreSQL container
	container := testutil.SetupPostgresContainerWithDB(ctx, t, "testdb", "testuser", "testpass")
	defer container.Terminate(ctx, t)

	// Setup database with initial schema
	conn := container.Conn

	initialSQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) UNIQUE
		);
		
		CREATE TABLE posts (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id),
			title VARCHAR(255) NOT NULL
		);
	`
	_, err = conn.ExecContext(ctx, initialSQL)
	if err != nil {
		t.Fatalf("Failed to setup initial schema: %v", err)
	}

	// Create desired state schema file (with additional column and table)
	tmpDir := t.TempDir()
	desiredStateFile := filepath.Join(tmpDir, "desired_state.sql")
	desiredStateSQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) UNIQUE,
			created_at TIMESTAMP DEFAULT NOW()
		);
		
		CREATE TABLE posts (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id),
			title VARCHAR(255) NOT NULL,
			content TEXT
		);
		
		CREATE TABLE comments (
			id SERIAL PRIMARY KEY,
			post_id INTEGER REFERENCES posts(id),
			content TEXT NOT NULL
		);
	`
	err = os.WriteFile(desiredStateFile, []byte(desiredStateSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write desired state file: %v", err)
	}

	// Get container connection details
	containerHost := container.Host
	portMapped := container.Port

	// Reset global flag variables for clean test state
	outputHuman = ""
	outputJSON = ""
	outputSQL = ""

	// Create a new command instance to avoid flag conflicts
	cmd := &cobra.Command{}
	*cmd = *PlanCmd

	// Set command arguments
	args := []string{
		"--host", containerHost,
		"--port", fmt.Sprintf("%d", portMapped),
		"--db", "testdb",
		"--user", "testuser",
		"--password", "testpass",
		"--file", desiredStateFile,
		"--output-human", "stdout",
	}
	cmd.SetArgs(args)

	// Run plan command
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Plan command failed: %v", err)
	}

	// The plan should succeed and show the differences
	t.Log("Plan command executed successfully")
}

func TestPlanCommand_OutputFormats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	var err error

	// Start PostgreSQL container
	container := testutil.SetupPostgresContainerWithDB(ctx, t, "testdb", "testuser", "testpass")
	defer container.Terminate(ctx, t)

	// Setup simple database schema
	conn := container.Conn

	simpleSQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		);
	`
	_, err = conn.ExecContext(ctx, simpleSQL)
	if err != nil {
		t.Fatalf("Failed to setup database schema: %v", err)
	}

	// Create desired state schema file
	tmpDir := t.TempDir()
	desiredStateFile := filepath.Join(tmpDir, "desired.sql")
	desiredSQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) UNIQUE
		);
		
		CREATE TABLE posts (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id),
			title VARCHAR(255) NOT NULL
		);
	`
	err = os.WriteFile(desiredStateFile, []byte(desiredSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write desired state file: %v", err)
	}

	// Get container connection details
	containerHost := container.Host
	portMapped := container.Port

	// Test different output formats
	testCases := []struct {
		name       string
		outputFlag string
	}{
		{"human format", "--output-human"},
		{"json format", "--output-json"},
		{"sql format", "--output-sql"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset global flag variables for clean test state
			outputHuman = ""
			outputJSON = ""
			outputSQL = ""

			// Create a new command instance for each test
			cmd := &cobra.Command{}
			*cmd = *PlanCmd

			// Set command arguments
			args := []string{
				"--host", containerHost,
				"--port", fmt.Sprintf("%d", portMapped),
				"--db", "testdb",
				"--user", "testuser",
				"--password", "testpass",
				"--file", desiredStateFile,
				tc.outputFlag, "stdout",
			}
			cmd.SetArgs(args)

			// Run plan command
			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Plan command failed with %s: %v", tc.name, err)
			}

			t.Logf("Plan command executed successfully with %s", tc.name)
		})
	}
}

func TestPlanCommand_SchemaFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	var err error

	// Start PostgreSQL container
	container := testutil.SetupPostgresContainerWithDB(ctx, t, "testdb", "testuser", "testpass")
	defer container.Terminate(ctx, t)

	// Setup database with multiple schemas
	conn := container.Conn

	multiSchemaSQL := `
		CREATE SCHEMA app;
		CREATE SCHEMA analytics;
		
		CREATE TABLE public.users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		);
		
		CREATE TABLE app.products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		);
		
		CREATE TABLE analytics.reports (
			id SERIAL PRIMARY KEY,
			data TEXT
		);
	`
	_, err = conn.ExecContext(ctx, multiSchemaSQL)
	if err != nil {
		t.Fatalf("Failed to setup multi-schema database: %v", err)
	}

	// Create desired state file for public schema only
	tmpDir := t.TempDir()
	publicSchemaFile := filepath.Join(tmpDir, "public_schema.sql")
	publicSchemaSQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) UNIQUE
		);
		
		CREATE TABLE posts (
			id SERIAL PRIMARY KEY,
			title VARCHAR(255) NOT NULL
		);
	`
	err = os.WriteFile(publicSchemaFile, []byte(publicSchemaSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write public schema file: %v", err)
	}

	// Get container connection details
	containerHost := container.Host
	portMapped := container.Port

	// Reset global flag variables for clean test state
	outputHuman = ""
	outputJSON = ""
	outputSQL = ""

	// Create a new command instance
	cmd := &cobra.Command{}
	*cmd = *PlanCmd

	// Set command arguments with schema filtering
	args := []string{
		"--host", containerHost,
		"--port", fmt.Sprintf("%d", portMapped),
		"--db", "testdb",
		"--user", "testuser",
		"--password", "testpass",
		"--schema", "public", // Filter to only public schema
		"--file", publicSchemaFile,
		"--output-human", "stdout",
	}
	cmd.SetArgs(args)

	// Run plan command
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Plan command failed with schema filtering: %v", err)
	}

	t.Log("Plan command executed successfully with schema filtering")
}

func TestPlanCommand_EmptyDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	var err error

	// Start PostgreSQL container with empty database
	container := testutil.SetupPostgresContainerWithDB(ctx, t, "testdb", "testuser", "testpass")
	defer container.Terminate(ctx, t)

	// Create desired state schema file
	tmpDir := t.TempDir()
	desiredStateFile := filepath.Join(tmpDir, "initial_schema.sql")
	desiredStateSQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) UNIQUE
		);
		
		CREATE TABLE posts (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id),
			title VARCHAR(255) NOT NULL,
			content TEXT
		);
	`
	err = os.WriteFile(desiredStateFile, []byte(desiredStateSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write desired state file: %v", err)
	}

	// Get container connection details
	containerHost := container.Host
	portMapped := container.Port

	// Reset global flag variables for clean test state
	outputHuman = ""
	outputJSON = ""
	outputSQL = ""

	// Create a new command instance
	cmd := &cobra.Command{}
	*cmd = *PlanCmd

	// Set command arguments
	args := []string{
		"--host", containerHost,
		"--port", fmt.Sprintf("%d", portMapped),
		"--db", "testdb",
		"--user", "testuser",
		"--password", "testpass",
		"--file", desiredStateFile,
		"--output-human", "stdout",
	}
	cmd.SetArgs(args)

	// Run plan command
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Plan command failed on empty database: %v", err)
	}

	t.Log("Plan command executed successfully on empty database")
}
