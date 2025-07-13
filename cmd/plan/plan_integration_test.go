package plan

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
		"--format", "text",
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
		name   string
		format string
	}{
		{"text format", "text"},
		{"json format", "json"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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
				"--format", tc.format,
			}
			cmd.SetArgs(args)

			// Run plan command
			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Plan command failed with %s format: %v", tc.format, err)
			}

			t.Logf("Plan command executed successfully with %s format", tc.format)
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
		"--format", "text",
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
		"--format", "text",
	}
	cmd.SetArgs(args)

	// Run plan command
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Plan command failed on empty database: %v", err)
	}

	t.Log("Plan command executed successfully on empty database")
}

func TestPlanCommand_GenerateTestdataPlans(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start a single PostgreSQL container for the entire test
	container := testutil.SetupPostgresContainerWithDB(ctx, t, "testdb", "testuser", "testpass")
	defer container.Terminate(ctx, t)

	// Get container connection details
	containerHost := container.Host
	portMapped := container.Port
	conn := container.Conn

	// Test data versions
	versions := []string{"v1", "v2", "v3", "v4", "v5"}

	for _, version := range versions {
		t.Run(fmt.Sprintf("Generate plan for %s", version), func(t *testing.T) {
			// Path to the schema file
			schemaFile := filepath.Join("testdata", version, "schema.sql")
			
			// Check if schema file exists
			if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
				t.Skipf("Schema file %s does not exist", schemaFile)
			}

			// Create a new command instance for JSON output
			cmdJSON := &cobra.Command{}
			*cmdJSON = *PlanCmd
			
			// Set command arguments for JSON
			argsJSON := []string{
				"--host", containerHost,
				"--port", fmt.Sprintf("%d", portMapped),
				"--db", "testdb",
				"--user", "testuser",
				"--password", "testpass",
				"--file", schemaFile,
				"--format", "json",
			}
			cmdJSON.SetArgs(argsJSON)

			// Capture JSON output
			jsonOutput := captureOutput(t, func() error {
				return cmdJSON.Execute()
			})

			// Write JSON output to file
			jsonFile := filepath.Join("testdata", version, "plan.json")
			err := os.WriteFile(jsonFile, []byte(jsonOutput), 0644)
			if err != nil {
				t.Fatalf("Failed to write JSON plan for %s: %v", version, err)
			}

			// Create a new command instance for text output
			cmdText := &cobra.Command{}
			*cmdText = *PlanCmd
			
			// Set command arguments for text
			argsText := []string{
				"--host", containerHost,
				"--port", fmt.Sprintf("%d", portMapped),
				"--db", "testdb",
				"--user", "testuser",
				"--password", "testpass",
				"--file", schemaFile,
				"--format", "text",
			}
			cmdText.SetArgs(argsText)

			// Capture text output
			textOutput := captureOutput(t, func() error {
				return cmdText.Execute()
			})

			// Write text output to file
			textFile := filepath.Join("testdata", version, "plan.txt")
			err = os.WriteFile(textFile, []byte(textOutput), 0644)
			if err != nil {
				t.Fatalf("Failed to write text plan for %s: %v", version, err)
			}

			// Create a new command instance for SQL output
			cmdSQL := &cobra.Command{}
			*cmdSQL = *PlanCmd
			
			// Set command arguments for SQL
			argsSQL := []string{
				"--host", containerHost,
				"--port", fmt.Sprintf("%d", portMapped),
				"--db", "testdb",
				"--user", "testuser",
				"--password", "testpass",
				"--file", schemaFile,
				"--format", "sql",
			}
			cmdSQL.SetArgs(argsSQL)

			// Capture SQL output
			sqlOutput := captureOutput(t, func() error {
				return cmdSQL.Execute()
			})

			// Write SQL output to file
			sqlFile := filepath.Join("testdata", version, "plan.sql")
			err = os.WriteFile(sqlFile, []byte(sqlOutput), 0644)
			if err != nil {
				t.Fatalf("Failed to write SQL plan for %s: %v", version, err)
			}

			// Apply the generated plan.sql to the database to prepare for the next version
			// The generated SQL must be executable - if it fails, that indicates a bug in our diff logic
			if sqlOutput != "" && !strings.Contains(sqlOutput, "No changes detected") && !strings.Contains(sqlOutput, "No DDL statements generated") {
				_, err = conn.ExecContext(ctx, sqlOutput)
				if err != nil {
					t.Fatalf("Failed to apply plan SQL for %s - this indicates a bug in our diff generation logic. SQL:\n%s\nError: %v", version, sqlOutput, err)
				}
				t.Logf("Applied %s plan to database", version)
			}

			t.Logf("Generated plans for %s", version)
		})
	}
}

// captureOutput captures stdout during function execution
func captureOutput(t *testing.T, fn func() error) string {
	// Backup original stdout
	oldStdout := os.Stdout
	
	// Create a pipe to capture output
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	
	// Replace stdout with the write end of the pipe
	os.Stdout = w
	
	// Channel to capture the output
	outputChan := make(chan string)
	
	// Start a goroutine to read from the pipe
	go func() {
		output := make([]byte, 0, 1024)
		buf := make([]byte, 1024)
		for {
			n, err := r.Read(buf)
			if err != nil {
				break
			}
			output = append(output, buf[:n]...)
		}
		outputChan <- string(output)
	}()
	
	// Execute the function
	err = fn()
	
	// Close the write end of the pipe
	w.Close()
	
	// Restore original stdout
	os.Stdout = oldStdout
	
	// Get the captured output
	output := <-outputChan
	
	if err != nil {
		t.Fatalf("Function execution failed: %v", err)
	}
	
	return output
}