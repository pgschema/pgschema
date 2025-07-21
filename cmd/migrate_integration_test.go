package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/pgschema/pgschema/cmd/apply"
	"github.com/pgschema/pgschema/cmd/util"
	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/ir"
	"github.com/pgschema/pgschema/internal/plan"
	"github.com/pgschema/pgschema/testutil"
	"github.com/spf13/cobra"
)

func TestPlanAndApply(t *testing.T) {
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

	testDir := "../testdata/migrate"
	// Discover available test data versions dynamically
	versions, err := discoverTestDataVersions(testDir)
	if err != nil {
		t.Fatalf("Failed to discover test data versions: %v", err)
	}

	for _, version := range versions {
		t.Run(fmt.Sprintf("Generate plan for %s", version), func(t *testing.T) {
			// Path to the schema file
			schemaFile := filepath.Join(testDir, version, "schema.sql")

			// Check if schema file exists
			if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
				t.Skipf("Schema file %s does not exist", schemaFile)
			}

			// Generate plan SQL directly without capturing output
			sqlOutput, err := generatePlanSQL(containerHost, portMapped, "testdb", "testuser", "testpass", "public", schemaFile)
			if err != nil {
				t.Fatalf("Failed to generate plan SQL for %s: %v", version, err)
			}

			// Apply the changes using pgschema apply command to test end-to-end functionality
			// The generated SQL must be executable - if it fails, that indicates a bug in our diff logic
			if sqlOutput != "" && !strings.Contains(sqlOutput, "No changes detected") && !strings.Contains(sqlOutput, "No DDL statements generated") {
				// Path A: Apply incremental migration (plan.sql) to public schema
				err = applySchemaChanges(containerHost, portMapped, "testdb", "testuser", "testpass", "public", schemaFile)
				if err != nil {
					t.Fatalf("Failed to apply schema changes for %s using pgschema apply: %v", version, err)
				}
				t.Logf("Applied %s schema changes using pgschema apply to public schema", version)

				// Create a separate database for the full schema application to avoid type conflicts
				fullDBName := fmt.Sprintf("testdb_full_%s", strings.ReplaceAll(version, "-", "_"))
				_, err = conn.Exec(fmt.Sprintf("CREATE DATABASE %s", fullDBName))
				if err != nil {
					t.Fatalf("Failed to create database %s: %v", fullDBName, err)
				}
				t.Logf("Created separate database %s for full schema application", fullDBName)

				// Apply the full schema.sql to the separate database
				err = applySchemaChanges(containerHost, portMapped, fullDBName, "testuser", "testpass", "public", schemaFile)
				if err != nil {
					t.Fatalf("Failed to apply full schema %s to separate database %s: %v", schemaFile, fullDBName, err)
				}
				t.Logf("Applied %s full schema to separate database %s", version, fullDBName)

				// Connect to the full schema database for IR comparison
				fullDBConn, err := util.Connect(&util.ConnectionConfig{
					Host:            containerHost,
					Port:            portMapped,
					Database:        fullDBName,
					User:            "testuser",
					Password:        "testpass",
					SSLMode:         "prefer",
					ApplicationName: "pgschema",
				})
				if err != nil {
					t.Fatalf("Failed to connect to full schema database %s: %v", fullDBName, err)
				}
				defer fullDBConn.Close()

				// Extract IR from both databases using Inspector
				inspector := ir.NewInspector(conn)
				fullDBInspector := ir.NewInspector(fullDBConn)

				// IR from public schema in main database (incremental migration result)
				publicSchemaIR, err := inspector.BuildIR(ctx, "public")
				if err != nil {
					t.Fatalf("Failed to build IR from public schema for %s: %v", version, err)
				}

				// IR from public schema in full database (complete schema application result)
				fullSchemaIR, err := fullDBInspector.BuildIR(ctx, "public")
				if err != nil {
					t.Fatalf("Failed to build IR from full database %s for %s: %v", fullDBName, version, err)
				}

				// Compare both IR representations for semantic equivalence
				publicInput := ir.IRComparisonInput{
					IR:          publicSchemaIR,
					Description: fmt.Sprintf("Incremental migration IR for %s (testdb.public)", version),
				}
				fullInput := ir.IRComparisonInput{
					IR:          fullSchemaIR,
					Description: fmt.Sprintf("Full schema IR for %s (%s.public)", version, fullDBName),
				}

				ir.CompareIRSemanticEquivalence(t, publicInput, fullInput)
				t.Logf("IR semantic equivalence verified between incremental and full schema application for %s", version)

				// Test idempotency: generate plan again to verify no changes are detected
				secondSqlOutput, err := generatePlanSQL(containerHost, portMapped, "testdb", "testuser", "testpass", "public", schemaFile)
				if err != nil {
					t.Fatalf("Failed to generate plan SQL for %s (idempotency check): %v", version, err)
				}

				// Verify that no changes are detected on second application
				if secondSqlOutput != "" && !strings.Contains(secondSqlOutput, "No changes detected") && !strings.Contains(secondSqlOutput, "No DDL statements generated") {
					t.Errorf("Expected no changes when applying %s schema twice, but got SQL output:\n%s", version, secondSqlOutput)
				} else {
					t.Logf("Idempotency verified for %s: no changes detected on second apply", version)
				}
			}

			t.Logf("Generated plans for %s", version)
		})
	}
}

// generatePlanSQL generates plan SQL using the internal plan logic without capturing stdout
func generatePlanSQL(host string, port int, database, user, password, schema, schemaFile string) (string, error) {
	// Read desired state file
	desiredStateData, err := os.ReadFile(schemaFile)
	if err != nil {
		return "", fmt.Errorf("failed to read desired state schema file: %w", err)
	}
	desiredState := string(desiredStateData)

	// Get current state from target database
	config := &util.ConnectionConfig{
		Host:            host,
		Port:            port,
		Database:        database,
		User:            user,
		Password:        password,
		SSLMode:         "prefer",
		ApplicationName: "pgschema",
	}

	conn, err := util.Connect(config)
	if err != nil {
		return "", fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close()

	ctx := context.Background()

	// Build IR using the IR system
	inspector := ir.NewInspector(conn)
	currentStateIR, err := inspector.BuildIR(ctx, schema)
	if err != nil {
		return "", fmt.Errorf("failed to build IR: %w", err)
	}

	// Parse desired state to IR
	desiredParser := ir.NewParser()
	desiredStateIR, err := desiredParser.ParseSQL(desiredState)
	if err != nil {
		return "", fmt.Errorf("failed to parse desired state schema file: %w", err)
	}

	// Generate diff (current -> desired) using IR directly
	ddlDiff := diff.Diff(currentStateIR, desiredStateIR)

	// Create plan from diff
	migrationPlan := plan.NewPlan(ddlDiff, schema)

	// Return SQL output
	return migrationPlan.ToSQL(), nil
}

// applySchemaChanges applies schema changes using the pgschema apply command
func applySchemaChanges(host string, port int, database, user, password, schema, schemaFile string) error {
	// Create a new root command with apply as subcommand
	rootCmd := &cobra.Command{
		Use: "pgschema",
	}

	// Add the apply command as a subcommand
	rootCmd.AddCommand(apply.ApplyCmd)

	// Set command arguments for apply
	args := []string{
		"apply",
		"--host", host,
		"--port", fmt.Sprintf("%d", port),
		"--db", database,
		"--user", user,
		"--password", password,
		"--schema", schema,
		"--file", schemaFile,
		"--auto-approve", // Auto-approve to avoid prompting during tests
	}
	rootCmd.SetArgs(args)

	// Execute the root command with apply subcommand
	return rootCmd.Execute()
}

// discoverTestDataVersions reads the testdata directory and returns a sorted list of version directories
func discoverTestDataVersions(testdataDir string) ([]string, error) {
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read testdata directory: %w", err)
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if the directory contains a schema.sql file
			schemaFile := filepath.Join(testdataDir, entry.Name(), "schema.sql")
			if _, err := os.Stat(schemaFile); err == nil {
				versions = append(versions, entry.Name())
			}
		}
	}

	// Sort versions to ensure deterministic test execution order
	sort.Strings(versions)
	return versions, nil
}
