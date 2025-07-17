package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/pgschema/pgschema/cmd/util"
	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/ir"
	"github.com/pgschema/pgschema/internal/plan"
	"github.com/pgschema/pgschema/testutil"
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

	// Discover available test data versions dynamically
	versions, err := discoverTestDataVersions("testdata")
	if err != nil {
		t.Fatalf("Failed to discover test data versions: %v", err)
	}

	for _, version := range versions {
		t.Run(fmt.Sprintf("Generate plan for %s", version), func(t *testing.T) {
			// Path to the schema file
			schemaFile := filepath.Join("testdata", version, "schema.sql")

			// Check if schema file exists
			if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
				t.Skipf("Schema file %s does not exist", schemaFile)
			}

			// Generate plan SQL directly without capturing output
			sqlOutput, err := generatePlanSQL(containerHost, portMapped, "testdb", "testuser", "testpass", "public", schemaFile)
			if err != nil {
				t.Fatalf("Failed to generate plan SQL for %s: %v", version, err)
			}

			// Apply the generated plan.sql to the database to prepare for the next version
			// The generated SQL must be executable - if it fails, that indicates a bug in our diff logic
			if sqlOutput != "" && !strings.Contains(sqlOutput, "No changes detected") && !strings.Contains(sqlOutput, "No DDL statements generated") {
				_, err = conn.ExecContext(ctx, sqlOutput)
				if err != nil {
					t.Fatalf("Failed to apply plan SQL for %s - this indicates a bug in our diff generation logic. SQL:\n%s\nError: %v", version, sqlOutput, err)
				}
				t.Logf("Applied %s plan to database", version)

				// After applying plan.sql, verify semantic equivalence between database and schema.sql
				// Parse schema.sql to IR
				schemaContent, err := os.ReadFile(schemaFile)
				if err != nil {
					t.Fatalf("Failed to read schema file %s: %v", schemaFile, err)
				}

				parser := ir.NewParser()
				parserIR, err := parser.ParseSQL(string(schemaContent))
				if err != nil {
					t.Fatalf("Failed to parse schema.sql into IR for %s: %v", version, err)
				}

				// Use inspector to convert database schema to IR
				inspector := ir.NewInspector(conn)
				dbIR, err := inspector.BuildIR(ctx, "public")
				if err != nil {
					t.Fatalf("Failed to build IR from database for %s: %v", version, err)
				}

				// Compare both IR formats for semantic equivalence
				dbInput := ir.IRComparisonInput{
					IR:          dbIR,
					Description: fmt.Sprintf("Database IR after applying %s plan.sql", version),
				}
				parserInput := ir.IRComparisonInput{
					IR:          parserIR,
					Description: fmt.Sprintf("Parser IR from %s schema.sql", version),
				}

				ir.CompareIRSemanticEquivalence(t, dbInput, parserInput)
				t.Logf("IR semantic equivalence verified for %s", version)
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
