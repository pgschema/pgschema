package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pgschema/pgschema/cmd/apply"
	planCmd "github.com/pgschema/pgschema/cmd/plan"
	"github.com/pgschema/pgschema/testutil"
	"github.com/spf13/cobra"
)

var generate = flag.Bool("generate", false, "generate expected test output files instead of comparing")

// TestPlanAndApply tests the complete CLI (plan and apply) workflow using test cases
// from testdata/diff/. This test exercises the full end-to-end CLI commands that
// users will execute, providing comprehensive validation of the plan and apply workflow.
//
// The test performs these actions for each test case:
// 1. Apply old.sql to database → initialize starting state
// 2. Run plan command with new.sql → generate and validate all output formats
// 3. Apply migration using apply command → execute the planned changes
// 4. Verify idempotency → re-running plan should produce no changes
//
// Test filtering can be controlled using the PGSCHEMA_TEST_FILTER environment variable:
//
// Examples:
//
//	# Run all tests under create_table/ (directory prefix with slash)
//	PGSCHEMA_TEST_FILTER="create_table/" go test -v ./cmd -run TestPlanAndApply
//
//	# Run tests under create_table/ that start with "add_column"
//	PGSCHEMA_TEST_FILTER="create_table/add_column" go test -v ./cmd -run TestPlanAndApply
//
//	# Run a specific test
//	PGSCHEMA_TEST_FILTER="create_table/add_column_identity" go test -v ./cmd -run TestPlanAndApply
//
//	# Run all migrate tests
//	PGSCHEMA_TEST_FILTER="migrate/" go test -v ./cmd -run TestPlanAndApply
func TestPlanAndApply(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDataRoot := "../testdata/diff"

	// Get test filter from environment variable
	testFilter := os.Getenv("PGSCHEMA_TEST_FILTER")

	// Walk through all test case directories in testdata/diff
	err := filepath.Walk(testDataRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory or if it's the root or category directories
		if !info.IsDir() || path == testDataRoot {
			return nil
		}

		// Skip category directories (e.g., create_table/, create_index/) - only process leaf directories
		relPath, _ := filepath.Rel(testDataRoot, path)
		pathDepth := len(strings.Split(relPath, string(filepath.Separator)))
		if pathDepth == 1 {
			return nil // Skip category directories
		}

		// Check if this directory contains the required test files
		oldFile := filepath.Join(path, "old.sql")
		newFile := filepath.Join(path, "new.sql")
		planSQLFile := filepath.Join(path, "plan.sql")
		planJSONFile := filepath.Join(path, "plan.json")
		planTXTFile := filepath.Join(path, "plan.txt")

		// Skip directories that don't contain the required test files
		if _, err := os.Stat(oldFile); os.IsNotExist(err) {
			return nil
		}
		if _, err := os.Stat(newFile); os.IsNotExist(err) {
			return nil
		}
		if _, err := os.Stat(planSQLFile); os.IsNotExist(err) {
			return nil
		}
		if _, err := os.Stat(planJSONFile); os.IsNotExist(err) {
			return nil
		}
		if _, err := os.Stat(planTXTFile); os.IsNotExist(err) {
			return nil
		}

		// Apply test filter if provided
		if testFilter != "" && !matchesFilter(relPath, testFilter) {
			return nil
		}

		// Get relative path for test name
		testName := strings.ReplaceAll(relPath, string(filepath.Separator), "_")

		t.Run(testName, func(t *testing.T) {
			runPlanAndApplyTest(t, ctx, oldFile, newFile, planSQLFile, planJSONFile, planTXTFile)
		})

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to walk test data directory: %v", err)
	}
}

// runPlanAndApplyTest executes a single plan and apply test case with fresh database
func runPlanAndApplyTest(t *testing.T, ctx context.Context, oldFile, newFile, planSQLFile, planJSONFile, planTXTFile string) {
	// Start a fresh PostgreSQL container for this test case
	container := testutil.SetupPostgresContainerWithDB(ctx, t, "testdb", "testuser", "testpass")
	defer container.Terminate(ctx, t)

	// Get container connection details
	containerHost := container.Host
	portMapped := container.Port

	t.Logf("=== PLAN AND APPLY TEST: %s → %s ===", filepath.Base(oldFile), filepath.Base(newFile))

	// STEP 1: Apply old.sql to initialize database state
	t.Logf("--- Applying old.sql to initialize database state ---")
	oldContent, err := os.ReadFile(oldFile)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", oldFile, err)
	}

	// Execute old.sql if it has content
	if len(strings.TrimSpace(string(oldContent))) > 0 {
		db := container.Conn
		_, err = db.ExecContext(ctx, string(oldContent))
		if err != nil {
			t.Fatalf("Failed to execute old.sql: %v", err)
		}
		t.Logf("Applied old.sql to initialize database state")
	}

	// STEP 2: Test plan command with new.sql as target
	t.Logf("--- Testing plan command outputs ---")
	testPlanOutputs(t, containerHost, portMapped, newFile, planSQLFile, planJSONFile, planTXTFile)

	// STEP 3: Apply the migration using apply command
	t.Logf("--- Applying migration using apply command ---")
	err = applySchemaChanges(containerHost, portMapped, "testdb", "testuser", "testpass", "public", newFile)
	if err != nil {
		t.Fatalf("Failed to apply schema changes using pgschema apply: %v", err)
	}
	t.Logf("Applied migration successfully")

	// STEP 4: Test idempotency - plan should produce no changes
	t.Logf("--- Testing idempotency ---")
	secondPlanOutput, err := generatePlanSQLFormatted(containerHost, portMapped, "testdb", "testuser", "testpass", "public", newFile)
	if err != nil {
		t.Fatalf("Failed to generate plan SQL for idempotency check: %v", err)
	}

	if secondPlanOutput != "" {
		t.Errorf("Expected no changes when applying schema twice, but got SQL output:\n%s", secondPlanOutput)
	} else {
		t.Logf("Idempotency verified: no changes detected on second apply")
	}

	t.Logf("=== PLAN AND APPLY TEST COMPLETED ===")
}

// testPlanOutputs tests all plan output formats against expected files
func testPlanOutputs(t *testing.T, containerHost string, portMapped int, schemaFile, planSQLFile, planJSONFile, planTXTFile string) {
	// Test SQL format
	sqlFormattedOutput, err := generatePlanSQLFormatted(containerHost, portMapped, "testdb", "testuser", "testpass", "public", schemaFile)
	if err != nil {
		t.Fatalf("Failed to generate plan SQL formatted output: %v", err)
	}

	if *generate {
		// Generate mode: write actual output to expected file
		actualSQLStr := strings.ReplaceAll(sqlFormattedOutput, "\r\n", "\n")
		err := os.WriteFile(planSQLFile, []byte(actualSQLStr), 0644)
		if err != nil {
			t.Fatalf("Failed to write expected SQL output file %s: %v", planSQLFile, err)
		}
		t.Logf("Generated expected SQL output file %s", planSQLFile)
	} else {
		// Compare mode: compare with expected file (always required)
		expectedSQL, err := os.ReadFile(planSQLFile)
		if err != nil {
			t.Fatalf("Failed to read expected SQL output file %s: %v", planSQLFile, err)
		}

		// Compare SQL output
		expectedSQLStr := strings.ReplaceAll(string(expectedSQL), "\r\n", "\n")
		actualSQLStr := strings.ReplaceAll(sqlFormattedOutput, "\r\n", "\n")
		if actualSQLStr != expectedSQLStr {
			t.Errorf("SQL output mismatch.\nExpected:\n%s\n\nActual:\n%s", expectedSQLStr, actualSQLStr)
			// Write actual output to file for easier comparison
			actualFile := strings.Replace(planSQLFile, ".sql", "_actual.sql", 1)
			os.WriteFile(actualFile, []byte(actualSQLStr), 0644)
			t.Logf("Actual SQL output written to %s", actualFile)
		}
	}

	// Test human-readable format
	humanOutput, err := generatePlanHuman(containerHost, portMapped, "testdb", "testuser", "testpass", "public", schemaFile)
	if err != nil {
		t.Fatalf("Failed to generate plan human output: %v", err)
	}

	if *generate {
		// Generate mode: write actual output to expected file
		actualHumanStr := strings.ReplaceAll(humanOutput, "\r\n", "\n")
		err := os.WriteFile(planTXTFile, []byte(actualHumanStr), 0644)
		if err != nil {
			t.Fatalf("Failed to write expected human output file %s: %v", planTXTFile, err)
		}
		t.Logf("Generated expected human output file %s", planTXTFile)
	} else {
		// Compare mode: compare with expected file (always required)
		expectedHuman, err := os.ReadFile(planTXTFile)
		if err != nil {
			t.Fatalf("Failed to read expected human output file %s: %v", planTXTFile, err)
		}

		// Compare human output (normalize line endings and trim)
		expectedHumanStr := strings.TrimSpace(strings.ReplaceAll(string(expectedHuman), "\r\n", "\n"))
		actualHumanStr := strings.TrimSpace(strings.ReplaceAll(humanOutput, "\r\n", "\n"))
		if actualHumanStr != expectedHumanStr {
			t.Errorf("Human output mismatch.\nExpected:\n%s\n\nActual:\n%s", expectedHumanStr, actualHumanStr)
			// Write actual output to file for easier comparison
			actualFile := strings.Replace(planTXTFile, ".txt", "_actual.txt", 1)
			os.WriteFile(actualFile, []byte(actualHumanStr), 0644)
			t.Logf("Actual human output written to %s", actualFile)
		}
	}

	// Test JSON format
	jsonOutput, err := generatePlanJSON(containerHost, portMapped, "testdb", "testuser", "testpass", "public", schemaFile)
	if err != nil {
		t.Fatalf("Failed to generate plan JSON output: %v", err)
	}

	if *generate {
		// Generate mode: write actual output to expected file
		err := os.WriteFile(planJSONFile, []byte(jsonOutput), 0644)
		if err != nil {
			t.Fatalf("Failed to write expected JSON output file %s: %v", planJSONFile, err)
		}
		t.Logf("Generated expected JSON output file %s", planJSONFile)
	} else {
		// Compare mode: compare with expected file (always required)
		expectedJSONBytes, err := os.ReadFile(planJSONFile)
		if err != nil {
			t.Fatalf("Failed to read expected JSON output file %s: %v", planJSONFile, err)
		}

		// Parse both JSON structures
		var expectedJSON, actualJSON map[string]interface{}

		if err := json.Unmarshal(expectedJSONBytes, &expectedJSON); err != nil {
			t.Fatalf("Failed to parse expected JSON: %v", err)
		}

		if err := json.Unmarshal([]byte(jsonOutput), &actualJSON); err != nil {
			t.Fatalf("Failed to parse actual JSON: %v. JSON output length: %d, content: %q", err, len(jsonOutput), jsonOutput)
		}

		// Compare JSON using go-cmp, ignoring dynamic fields
		ignoreFields := cmp.FilterPath(func(p cmp.Path) bool {
			// Get the last element of the path
			if len(p) == 0 {
				return false
			}
			last := p[len(p)-1]
			if mf, ok := last.(cmp.MapIndex); ok {
				key := fmt.Sprintf("%v", mf.Key().Interface())
				// Match field names
				return key == "created_at" || key == "pgschema_version"
			}
			return false
		}, cmp.Ignore())

		if diff := cmp.Diff(expectedJSON, actualJSON, ignoreFields); diff != "" {
			t.Errorf("JSON plan mismatch (-want +got):\n%s", diff)
		}
	}
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

// resetPlanFlags resets the plan command global flag variables for testing
func resetPlanFlags() {
	planCmd.ResetFlags()
}

// generatePlanOutput generates plan output using the CLI plan command with the specified format
func generatePlanOutput(host string, port int, database, user, password, schema, schemaFile, outputFlag string, extraArgs ...string) (string, error) {
	// Reset global flag variables for clean state
	resetPlanFlags()

	// Create a new root command with plan as subcommand
	rootCmd := &cobra.Command{
		Use: "pgschema",
	}

	// Add the plan command as a subcommand
	rootCmd.AddCommand(planCmd.PlanCmd)

	// Capture stdout by redirecting it temporarily
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w

	// Set command arguments for plan
	args := []string{
		"plan",
		"--host", host,
		"--port", fmt.Sprintf("%d", port),
		"--db", database,
		"--user", user,
		"--password", password,
		"--schema", schema,
		"--file", schemaFile,
		outputFlag, "stdout",
	}

	// Add any extra arguments
	args = append(args, extraArgs...)
	rootCmd.SetArgs(args)

	// Execute the root command with plan subcommand in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- rootCmd.Execute()
	}()

	// Copy the output from the pipe to our buffer in a goroutine
	copyDone := make(chan struct{})
	go func() {
		defer close(copyDone)
		defer r.Close()
		buf.ReadFrom(r)
	}()

	// Wait for command to complete
	cmdErr := <-done

	// Close the writer to signal EOF to the reader
	w.Close()

	// Wait for the copy operation to complete
	<-copyDone

	// Restore stdout
	os.Stdout = oldStdout

	if cmdErr != nil {
		return "", cmdErr
	}

	return buf.String(), nil
}

// generatePlanHuman generates plan human-readable output using the CLI plan command
func generatePlanHuman(host string, port int, database, user, password, schema, schemaFile string) (string, error) {
	return generatePlanOutput(host, port, database, user, password, schema, schemaFile, "--output-human", "stdout", "--no-color")
}

// generatePlanJSON generates plan JSON output using the CLI plan command
func generatePlanJSON(host string, port int, database, user, password, schema, schemaFile string) (string, error) {
	return generatePlanOutput(host, port, database, user, password, schema, schemaFile, "--output-json", "stdout")
}

// generatePlanSQLFormatted generates plan SQL output using the CLI plan command
func generatePlanSQLFormatted(host string, port int, database, user, password, schema, schemaFile string) (string, error) {
	return generatePlanOutput(host, port, database, user, password, schema, schemaFile, "--output-sql", "stdout")
}

// matchesFilter checks if a relative path matches the given filter pattern
func matchesFilter(relPath, filter string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}

	// Handle directory prefix patterns with trailing slash
	if strings.HasSuffix(filter, "/") {
		// "create_table/" matches "create_table/add_column_identity"
		return strings.HasPrefix(relPath+"/", filter)
	}

	// Handle patterns with slash (both specific tests and prefix patterns)
	if strings.Contains(filter, "/") {
		// "create_table/add_column_identity" matches "create_table/add_column_identity"
		// "create_table/add_column" matches "create_table/add_column_identity"
		return strings.HasPrefix(relPath, filter)
	}

	// Fallback: check if filter is a substring of the path
	return strings.Contains(relPath, filter)
}
