package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pgschema/pgschema/cmd/apply"
	planCmd "github.com/pgschema/pgschema/cmd/plan"
	"github.com/pgschema/pgschema/testutil"
	"github.com/spf13/cobra"
)

var generate = flag.Bool("generate", false, "generate expected test output files instead of comparing")

func TestPlanAndApply(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start a single PostgreSQL container for the entire test
	// This container will be used for sequential migrations (v1 -> v2 -> v3 -> ...)
	container := testutil.SetupPostgresContainerWithDB(ctx, t, "testdb", "testuser", "testpass")
	defer container.Terminate(ctx, t)

	// Get container connection details
	containerHost := container.Host
	portMapped := container.Port

	testDir := "../testdata/migrate"
	// Discover available test data versions dynamically
	versions, err := discoverTestDataVersions(testDir)
	if err != nil {
		t.Fatalf("Failed to discover test data versions: %v", err)
	}

	// Run versions sequentially to build incremental changes
	for _, version := range versions {
		t.Run(fmt.Sprintf("Generate plan for %s", version), func(t *testing.T) {
			// Path to the schema file
			schemaFile := filepath.Join(testDir, version, "schema.sql")

			// Check if schema file exists
			if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
				t.Skipf("Schema file %s does not exist", schemaFile)
			}

			// Generate and validate SQL plan output
			sqlFormattedOutput, err := generatePlanSQLFormatted(containerHost, portMapped, "testdb", "testuser", "testpass", "public", schemaFile)
			if err != nil {
				t.Fatalf("Failed to generate plan SQL formatted output for %s: %v", version, err)
			}

			// Handle SQL output - either generate or compare
			expectedSQLFile := filepath.Join(testDir, version, "plan.sql")
			if *generate {
				// Generate mode: write actual output to expected file
				actualSQLStr := strings.ReplaceAll(sqlFormattedOutput, "\r\n", "\n")
				err := os.WriteFile(expectedSQLFile, []byte(actualSQLStr), 0644)
				if err != nil {
					t.Fatalf("Failed to write expected SQL output file %s: %v", expectedSQLFile, err)
				}
				t.Logf("Generated expected SQL output file %s", expectedSQLFile)
			} else if _, err := os.Stat(expectedSQLFile); err == nil {
				// Compare mode: compare with expected file
				expectedSQL, err := os.ReadFile(expectedSQLFile)
				if err != nil {
					t.Fatalf("Failed to read expected SQL output file %s: %v", expectedSQLFile, err)
				}

				// Compare SQL output (normalize line endings)
				expectedSQLStr := strings.ReplaceAll(string(expectedSQL), "\r\n", "\n")
				actualSQLStr := strings.ReplaceAll(sqlFormattedOutput, "\r\n", "\n")
				if actualSQLStr != expectedSQLStr {
					t.Errorf("SQL output mismatch for %s.\nExpected:\n%s\n\nActual:\n%s", version, expectedSQLStr, actualSQLStr)
					// Write actual output to file for easier comparison
					actualFile := filepath.Join(testDir, version, "plan_actual.sql")
					os.WriteFile(actualFile, []byte(actualSQLStr), 0644)
					t.Logf("Actual SQL output written to %s", actualFile)
				}
			}

			// Generate and validate human-readable plan output
			humanOutput, err := generatePlanHuman(containerHost, portMapped, "testdb", "testuser", "testpass", "public", schemaFile)
			if err != nil {
				t.Fatalf("Failed to generate plan human output for %s: %v", version, err)
			}

			// Handle human output - either generate or compare
			expectedHumanFile := filepath.Join(testDir, version, "plan.txt")
			if *generate {
				// Generate mode: write actual output to expected file
				actualHumanStr := strings.ReplaceAll(humanOutput, "\r\n", "\n")
				err := os.WriteFile(expectedHumanFile, []byte(actualHumanStr), 0644)
				if err != nil {
					t.Fatalf("Failed to write expected human output file %s: %v", expectedHumanFile, err)
				}
				t.Logf("Generated expected human output file %s", expectedHumanFile)
			} else if _, err := os.Stat(expectedHumanFile); err == nil {
				// Compare mode: compare with expected file
				expectedHuman, err := os.ReadFile(expectedHumanFile)
				if err != nil {
					t.Fatalf("Failed to read expected human output file %s: %v", expectedHumanFile, err)
				}

				// Compare human output (normalize line endings)
				expectedHumanStr := strings.ReplaceAll(string(expectedHuman), "\r\n", "\n")
				actualHumanStr := strings.ReplaceAll(humanOutput, "\r\n", "\n")
				if actualHumanStr != expectedHumanStr {
					t.Errorf("Human output mismatch for %s.\nExpected:\n%s\n\nActual:\n%s", version, expectedHumanStr, actualHumanStr)
					// Write actual output to file for easier comparison
					actualFile := filepath.Join(testDir, version, "plan_actual.txt")
					os.WriteFile(actualFile, []byte(actualHumanStr), 0644)
					t.Logf("Actual human output written to %s", actualFile)
				}
			}

			// Generate and validate JSON plan output
			jsonOutput, err := generatePlanJSON(containerHost, portMapped, "testdb", "testuser", "testpass", "public", schemaFile)
			if err != nil {
				t.Fatalf("Failed to generate plan JSON output for %s: %v", version, err)
			}

			// Handle JSON output - either generate or compare
			expectedJSONFile := filepath.Join(testDir, version, "plan.json")
			if *generate {
				// Generate mode: write actual output to expected file
				err := os.WriteFile(expectedJSONFile, []byte(jsonOutput), 0644)
				if err != nil {
					t.Fatalf("Failed to write expected JSON output file %s: %v", expectedJSONFile, err)
				}
				t.Logf("Generated expected JSON output file %s", expectedJSONFile)
			} else if _, err := os.Stat(expectedJSONFile); err == nil {
				// Compare mode: compare with expected file
				expectedJSONBytes, err := os.ReadFile(expectedJSONFile)
				if err != nil {
					t.Fatalf("Failed to read expected JSON output file %s: %v", expectedJSONFile, err)
				}

				// Parse both JSON structures
				var expectedJSON, actualJSON map[string]interface{}

				if err := json.Unmarshal(expectedJSONBytes, &expectedJSON); err != nil {
					t.Fatalf("Failed to parse expected JSON for %s: %v", version, err)
				}

				if err := json.Unmarshal([]byte(jsonOutput), &actualJSON); err != nil {
					t.Fatalf("Failed to parse actual JSON for %s: %v. JSON output length: %d, content: %q", version, err, len(jsonOutput), jsonOutput)
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
					t.Errorf("JSON plan mismatch for %s (-want +got):\n%s", version, diff)
				}
			}

			// Apply incremental migration to main testdb for the next iteration
			err = applySchemaChanges(containerHost, portMapped, "testdb", "testuser", "testpass", "public", schemaFile)
			if err != nil {
				t.Fatalf("Failed to apply schema changes for %s using pgschema apply: %v", version, err)
			}
			t.Logf("Applied %s schema changes using pgschema apply to testdb.public", version)

			// Test idempotency: generate plan again to verify no changes are detected
			secondSqlOutput, err := generatePlanSQLFormatted(containerHost, portMapped, "testdb", "testuser", "testpass", "public", schemaFile)
			if err != nil {
				t.Fatalf("Failed to generate plan SQL for %s (idempotency check): %v", version, err)
			}

			// Verify that no changes are detected on second application
			if secondSqlOutput != "" {
				t.Errorf("Expected no changes when applying %s schema twice, but got SQL output:\n%s", version, secondSqlOutput)
			} else {
				t.Logf("Idempotency verified for %s: no changes detected on second apply", version)
			}
		})
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
