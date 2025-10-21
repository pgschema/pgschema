package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pgschema/pgschema/cmd/apply"
	planCmd "github.com/pgschema/pgschema/cmd/plan"
	"github.com/pgschema/pgschema/testutil"
	"github.com/spf13/cobra"
)

// TestNonPublicSchemaOperations verifies that pgschema works correctly with non-public schemas.
// This test uses the actual CLI commands to ensure the --schema flag works properly.
func TestNonPublicSchemaOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start PostgreSQL container
	container := testutil.SetupPostgresContainerWithDB(ctx, t, "testdb", "testuser", "testpass")
	defer container.Terminate(ctx, t)

	conn := container.Conn

	// Test Case 1: Plan and Apply to tenant schema using CLI
	t.Run("cli_plan_and_apply_tenant_schema", func(t *testing.T) {
		// Setup: Create tenant schema with initial table
		_, err := conn.ExecContext(ctx, `
			CREATE SCHEMA IF NOT EXISTS tenant;
			CREATE TABLE tenant.users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL
			);
		`)
		if err != nil {
			t.Fatalf("Failed to setup tenant schema: %v", err)
		}

		// Create desired state file to add email column
		tmpDir := t.TempDir()
		desiredStateFile := filepath.Join(tmpDir, "tenant_desired.sql")
		desiredStateSQL := `
			CREATE TABLE IF NOT EXISTS users (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				email VARCHAR(255) UNIQUE
			);
		`
		err = os.WriteFile(desiredStateFile, []byte(desiredStateSQL), 0644)
		if err != nil {
			t.Fatalf("Failed to write desired state file: %v", err)
		}

		// Step 1: Generate plan using CLI
		planOutput, err := executePlanCommand(
			container.Host, 
			container.Port, 
			"testdb", 
			"testuser", 
			"testpass", 
			"tenant", // Non-public schema
			desiredStateFile,
		)
		if err != nil {
			t.Fatalf("Failed to generate plan via CLI: %v", err)
		}

		t.Logf("Plan output for tenant schema:\n%s", planOutput)

		// Verify plan contains expected changes
		if !strings.Contains(planOutput, "ALTER TABLE") || !strings.Contains(planOutput, "email") {
			t.Logf("WARNING: Expected plan to contain ALTER TABLE for email column, got:\n%s", planOutput)
		}

		// Step 2: Apply changes using CLI
		err = executeApplyCommand(
			container.Host,
			container.Port,
			"testdb",
			"testuser",
			"testpass",
			"tenant", // Non-public schema
			desiredStateFile,
		)
		if err != nil {
			t.Fatalf("Failed to apply changes via CLI: %v", err)
		}

		// Step 3: Verify changes were applied to the correct schema
		var emailInTenant bool
		err = conn.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_schema = 'tenant' 
				AND table_name = 'users' 
				AND column_name = 'email'
			)
		`).Scan(&emailInTenant)
		if err != nil {
			t.Fatalf("Failed to check if email column exists in tenant.users: %v", err)
		}

		if !emailInTenant {
			t.Fatal("CRITICAL BUG: Email column should exist in tenant.users after apply, but it doesn't!")
		}

		// Also verify public schema wasn't affected
		var tableInPublic bool
		err = conn.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'users'
			)
		`).Scan(&tableInPublic)
		if err != nil {
			t.Fatalf("Failed to check if users table exists in public: %v", err)
		}

		if tableInPublic {
			t.Fatal("BUG: Users table should NOT exist in public schema - changes leaked to wrong schema!")
		}

		t.Log("✓ Successfully applied changes to tenant schema via CLI")
	})

	// Test Case 2: Test schema isolation with multiple schemas
	t.Run("cli_schema_isolation", func(t *testing.T) {
		// Setup: Create two separate schemas with identical tables
		_, err := conn.ExecContext(ctx, `
			CREATE SCHEMA IF NOT EXISTS app_a;
			CREATE SCHEMA IF NOT EXISTS app_b;
			
			CREATE TABLE app_a.products (
				id SERIAL PRIMARY KEY, 
				name VARCHAR(255) NOT NULL
			);
			CREATE TABLE app_b.products (
				id SERIAL PRIMARY KEY, 
				name VARCHAR(255) NOT NULL
			);
		`)
		if err != nil {
			t.Fatalf("Failed to setup test schemas: %v", err)
		}

		// Create desired state file to add price column
		tmpDir := t.TempDir()
		desiredStateFile := filepath.Join(tmpDir, "products_with_price.sql")
		desiredStateSQL := `
			CREATE TABLE IF NOT EXISTS products (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				price DECIMAL(10, 2)
			);
		`
		err = os.WriteFile(desiredStateFile, []byte(desiredStateSQL), 0644)
		if err != nil {
			t.Fatalf("Failed to write desired state file: %v", err)
		}

		// Apply changes ONLY to app_a schema
		err = executeApplyCommand(
			container.Host,
			container.Port,
			"testdb",
			"testuser",
			"testpass",
			"app_a", // Target only app_a
			desiredStateFile,
		)
		if err != nil {
			t.Fatalf("Failed to apply changes to app_a: %v", err)
		}

		// Verify app_a has the new column
		var priceInAppA bool
		err = conn.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_schema = 'app_a' 
				AND table_name = 'products' 
				AND column_name = 'price'
			)
		`).Scan(&priceInAppA)
		if err != nil {
			t.Fatalf("Failed to check price column in app_a: %v", err)
		}

		// Verify app_b does NOT have the new column
		var priceInAppB bool
		err = conn.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_schema = 'app_b' 
				AND table_name = 'products' 
				AND column_name = 'price'
			)
		`).Scan(&priceInAppB)
		if err != nil {
			t.Fatalf("Failed to check price column in app_b: %v", err)
		}

		if !priceInAppA {
			t.Fatal("CRITICAL BUG: price column should exist in app_a.products after apply")
		}
		if priceInAppB {
			t.Fatal("CRITICAL BUG: price column should NOT exist in app_b.products - schema isolation violated!")
		}

		t.Log("✓ Schema isolation verified - changes properly isolated between app_a and app_b")
	})

	// Test Case 3: Test mixed-case schema name handling
	t.Run("cli_mixed_case_schema", func(t *testing.T) {
		// Setup: Create mixed-case schema with initial table
		_, err := conn.ExecContext(ctx, `
			CREATE SCHEMA IF NOT EXISTS "MyApp";
			CREATE TABLE "MyApp".orders (
				id SERIAL PRIMARY KEY,
				customer_name VARCHAR(255) NOT NULL
			);
		`)
		if err != nil {
			t.Fatalf("Failed to setup mixed-case schema: %v", err)
		}

		// Create desired state file to add status column
		tmpDir := t.TempDir()
		desiredStateFile := filepath.Join(tmpDir, "orders_with_status.sql")
		desiredStateSQL := `
			CREATE TABLE IF NOT EXISTS orders (
				id SERIAL PRIMARY KEY,
				customer_name VARCHAR(255) NOT NULL,
				status VARCHAR(50) DEFAULT 'pending'
			);
		`
		err = os.WriteFile(desiredStateFile, []byte(desiredStateSQL), 0644)
		if err != nil {
			t.Fatalf("Failed to write desired state file: %v", err)
		}

		// Step 1: Generate plan using CLI for mixed-case schema
		planOutput, err := executePlanCommand(
			container.Host, 
			container.Port, 
			"testdb", 
			"testuser", 
			"testpass", 
			"MyApp", // Mixed-case schema
			desiredStateFile,
		)
		if err != nil {
			t.Fatalf("Failed to generate plan for mixed-case schema: %v", err)
		}

		t.Logf("Plan output for mixed-case schema:\\n%s", planOutput)

		// Verify plan contains expected changes
		if !strings.Contains(planOutput, "ALTER TABLE") || !strings.Contains(planOutput, "status") {
			t.Logf("WARNING: Expected plan to contain ALTER TABLE for status column, got:\\n%s", planOutput)
		}

		// Step 2: Apply changes using CLI for mixed-case schema
		err = executeApplyCommand(
			container.Host,
			container.Port,
			"testdb",
			"testuser",
			"testpass",
			"MyApp", // Mixed-case schema
			desiredStateFile,
		)
		if err != nil {
			t.Fatalf("Failed to apply changes to mixed-case schema: %v", err)
		}

		// Step 3: Verify changes were applied to the correct mixed-case schema
		var statusInMixedCase bool
		err = conn.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_schema = 'MyApp' 
				AND table_name = 'orders' 
				AND column_name = 'status'
			)
		`).Scan(&statusInMixedCase)
		if err != nil {
			t.Fatalf("Failed to check if status column exists in MyApp.orders: %v", err)
		}

		if !statusInMixedCase {
			t.Fatal("CRITICAL BUG: Status column should exist in MyApp.orders after apply, but it doesn't!")
		}

		// Also verify lowercase version doesn't exist (ensuring no case folding occurred)
		var statusInLowercase bool
		err = conn.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_schema = 'myapp' 
				AND table_name = 'orders' 
				AND column_name = 'status'
			)
		`).Scan(&statusInLowercase)
		if err != nil {
			t.Fatalf("Failed to check if status column exists in lowercase myapp.orders: %v", err)
		}

		if statusInLowercase {
			t.Fatal("BUG: Status column should NOT exist in lowercase myapp schema - schema name was incorrectly case-folded!")
		}

		t.Log("✓ Successfully applied changes to mixed-case schema via CLI")
	})

	// Test Case 4: Test schema-qualified function in DEFAULT values (Bug #12 reproduction)
	// TODO: need to dump the target database schema and apply to the tmp database first
	// to get the utils schema
	// t.Run("schema_qualified_function_in_default", func(t *testing.T) {
	// 	// Setup: Create utils schema with function (pre-existing)
	// 	_, err := conn.ExecContext(ctx, `
	// 		CREATE SCHEMA IF NOT EXISTS utils;

	// 		CREATE FUNCTION utils.generate_something()
	// 		  RETURNS text
	// 		  LANGUAGE plpgsql
	// 		  STABLE
	// 		  PARALLEL SAFE
	// 		AS $$
	// 		BEGIN
	// 		  RETURN 'Something';
	// 		END;
	// 		$$;
	// 	`)
	// 	if err != nil {
	// 		t.Fatalf("Failed to setup utils schema and function: %v", err)
	// 	}

	// 	// Create desired state file with table that references utils function
	// 	tmpDir := t.TempDir()
	// 	desiredStateFile := filepath.Join(tmpDir, "table_with_utils_function.sql")
	// 	desiredStateSQL := `
	// 		CREATE TABLE IF NOT EXISTS something_table (
	// 		   column_one text DEFAULT utils.generate_something()
	// 		);
	// 	`
	// 	err = os.WriteFile(desiredStateFile, []byte(desiredStateSQL), 0644)
	// 	if err != nil {
	// 		t.Fatalf("Failed to write desired state file: %v", err)
	// 	}

	// 	// Step 1: Generate plan using CLI
	// 	planOutput, err := executePlanCommand(
	// 		container.Host,
	// 		container.Port,
	// 		"testdb",
	// 		"testuser",
	// 		"testpass",
	// 		"public", // Target schema
	// 		desiredStateFile,
	// 	)
	// 	if err != nil {
	// 		t.Fatalf("Failed to generate plan via CLI: %v", err)
	// 	}

	// 	t.Logf("Plan output:\n%s", planOutput)

	// 	// Verify the plan contains the full schema-qualified function name
	// 	if !strings.Contains(planOutput, "utils.generate_something()") {
	// 		t.Errorf("Expected 'utils.generate_something()' in plan output, but not found")

	// 		// Check if it contains the truncated version
	// 		if strings.Contains(planOutput, "utils()") {
	// 			t.Errorf("Found 'utils()' instead of 'utils.generate_something()' - function name was truncated in plan")
	// 		}
	// 	}

	// 	// Verify plan doesn't contain the truncated version
	// 	if strings.Contains(planOutput, "DEFAULT utils()") {
	// 		t.Errorf("Found 'DEFAULT utils()' in plan - function name was truncated, expected 'DEFAULT utils.generate_something()'")
	// 	}

	// 	// Step 2: Apply changes using CLI
	// 	err = executeApplyCommand(
	// 		container.Host,
	// 		container.Port,
	// 		"testdb",
	// 		"testuser",
	// 		"testpass",
	// 		"public",
	// 		desiredStateFile,
	// 	)
	// 	if err != nil {
	// 		t.Fatalf("Failed to apply changes via CLI: %v", err)
	// 	}

	// 	// Step 3: Verify the table was created correctly with proper DEFAULT
	// 	var columnDefault string
	// 	err = conn.QueryRowContext(ctx, `
	// 		SELECT column_default
	// 		FROM information_schema.columns
	// 		WHERE table_schema = 'public'
	// 		AND table_name = 'something_table'
	// 		AND column_name = 'column_one'
	// 	`).Scan(&columnDefault)
	// 	if err != nil {
	// 		t.Fatalf("Failed to check column default: %v", err)
	// 	}

	// 	// Verify the actual column default contains the full function name
	// 	if !strings.Contains(columnDefault, "utils.generate_something()") {
	// 		t.Errorf("Column default in database: %s", columnDefault)
	// 		t.Errorf("Expected column default to contain 'utils.generate_something()'")
	// 	}

	// 	// Verify the function actually works by testing the default
	// 	var testValue string
	// 	err = conn.QueryRowContext(ctx, `
	// 		INSERT INTO something_table DEFAULT VALUES RETURNING column_one
	// 	`).Scan(&testValue)
	// 	if err != nil {
	// 		t.Fatalf("Failed to test default value: %v", err)
	// 	}

	// 	if testValue != "Something" {
	// 		t.Errorf("Expected default value 'Something', got '%s'", testValue)
	// 	}

	// 	t.Log("✓ Schema-qualified function in DEFAULT preserved correctly through plan and apply")
	// })
}

// executePlanCommand executes the pgschema plan command using the CLI interface
func executePlanCommand(host string, port int, database, user, password, schema, schemaFile string) (string, error) {
	// Reset plan flags for clean state
	planCmd.ResetFlags()

	// Create root command with plan as subcommand
	rootCmd := &cobra.Command{
		Use: "pgschema",
	}
	rootCmd.AddCommand(planCmd.PlanCmd)

	// Capture stdout
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w

	// Set command arguments
	args := []string{
		"plan",
		"--host", host,
		"--port", fmt.Sprintf("%d", port),
		"--db", database,
		"--user", user,
		"--password", password,
		"--schema", schema,
		"--file", schemaFile,
		"--output-sql", "stdout",
	}
	rootCmd.SetArgs(args)

	// Execute command in goroutine
	done := make(chan error, 1)
	go func() {
		done <- rootCmd.Execute()
	}()

	// Copy output
	copyDone := make(chan struct{})
	go func() {
		defer close(copyDone)
		defer r.Close()
		buf.ReadFrom(r)
	}()

	// Wait for command
	cmdErr := <-done
	w.Close()
	<-copyDone

	// Restore stdout
	os.Stdout = oldStdout

	if cmdErr != nil {
		return buf.String(), fmt.Errorf("plan command failed: %w (output: %s)", cmdErr, buf.String())
	}

	return buf.String(), nil
}

// executeApplyCommand executes the pgschema apply command using the CLI interface
func executeApplyCommand(host string, port int, database, user, password, schema, schemaFile string) error {
	// Create root command with apply as subcommand
	rootCmd := &cobra.Command{
		Use: "pgschema",
	}
	rootCmd.AddCommand(apply.ApplyCmd)

	// Set command arguments
	args := []string{
		"apply",
		"--host", host,
		"--port", fmt.Sprintf("%d", port),
		"--db", database,
		"--user", user,
		"--password", password,
		"--schema", schema,
		"--file", schemaFile,
		"--auto-approve", // Auto-approve for testing
	}
	rootCmd.SetArgs(args)

	// Execute the command
	return rootCmd.Execute()
}