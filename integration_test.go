package main

import (
	"bytes"
	"flag"
	"os/exec"
	"strings"
	"testing"
)

var testTempDbDSN = flag.String("test-temp-db-dsn", "", "Temporary database DSN for integration tests")

func TestIntegrationDiffDirectories(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This test requires a real PostgreSQL database for temp-db-dsn
	// Skip if -test-temp-db-dsn flag is not provided
	if *testTempDbDSN == "" {
		t.Skip("skipping integration test: -test-temp-db-dsn flag not provided")
	}

	cmd := exec.Command("go", "run", ".", "diff", "--source-dir", "testdata/schema1", "--target-dir", "testdata/schema2", "--temp-db-dsn", *testTempDbDSN)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	output := out.String()
	
	if strings.Contains(output, "No differences found") {
		t.Error("expected differences between schema1 and schema2, but none were found")
	}

	expectedChanges := []string{"phone", "updated_at", "products"}
	for _, change := range expectedChanges {
		if !strings.Contains(output, change) {
			t.Errorf("expected output to contain '%s', but it didn't. Output: %s", change, output)
		}
	}
}

func TestIntegrationSameDirectories(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This test requires a real PostgreSQL database for temp-db-dsn
	// Skip if -test-temp-db-dsn flag is not provided
	if *testTempDbDSN == "" {
		t.Skip("skipping integration test: -test-temp-db-dsn flag not provided")
	}

	cmd := exec.Command("go", "run", ".", "diff", "--source-dir", "testdata/schema1", "--target-dir", "testdata/schema1", "--temp-db-dsn", *testTempDbDSN)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err != nil {
		t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	output := out.String()
	
	if !strings.Contains(output, "No differences found") {
		t.Errorf("expected no differences when comparing same schema, but got: %s", output)
	}
}

func TestIntegrationInvalidFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cmd := exec.Command("go", "run", ".", "diff", "--source-dir", "testdata/schema1")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err == nil {
		t.Error("expected command to fail with incomplete flags, but it succeeded")
	}

	combinedOutput := out.String() + stderr.String()
	if !strings.Contains(combinedOutput, "must specify both source and target") {
		t.Errorf("expected error message about missing target, got: %s", combinedOutput)
	}
}

func TestIntegrationMissingTempDbDsn(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cmd := exec.Command("go", "run", ".", "diff", "--source-dir", "testdata/schema1", "--target-dir", "testdata/schema2")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err == nil {
		t.Error("expected command to fail with missing temp-db-dsn, but it succeeded")
	}

	combinedOutput := out.String() + stderr.String()
	if !strings.Contains(combinedOutput, "--temp-db-dsn is required when using directory-based schemas") {
		t.Errorf("expected error message about missing temp-db-dsn, got: %s", combinedOutput)
	}
}

func TestIntegrationNonExistentDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cmd := exec.Command("go", "run", ".", "diff", "--source-dir", "testdata/nonexistent", "--target-dir", "testdata/schema1", "--temp-db-dsn", "postgres://fake:fake@localhost:5432/fake")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err == nil {
		t.Error("expected command to fail with non-existent directory, but it succeeded")
	}

	combinedOutput := out.String() + stderr.String()
	if !strings.Contains(combinedOutput, "failed to load") && !strings.Contains(combinedOutput, "no such file or directory") && !strings.Contains(combinedOutput, "failed to ping temp database") {
		t.Errorf("expected error message about failed loading, file not found, or temp database connection, got: %s", combinedOutput)
	}
}