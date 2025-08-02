package apply

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	planCmd "github.com/pgschema/pgschema/cmd/plan"
	"github.com/pgschema/pgschema/internal/plan"
	"github.com/pgschema/pgschema/testutil"
	"github.com/spf13/cobra"
)

// TestApplyCommand_TransactionRollback verifies that the apply command uses proper
// transaction mode. If any statement fails in the middle of execution, the entire
// transaction should be rolled back and no partial changes should be applied.
//
// The test creates a migration that contains:
// 1. A valid DDL statement (ADD COLUMN email to users table)
// 2. An invalid DDL statement (CREATE TABLE with foreign key to nonexistent table)
//
// When the second statement fails, the first statement should also be rolled back.
func TestApplyCommand_TransactionRollback(t *testing.T) {
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
			name VARCHAR(255) NOT NULL
		);
		
		INSERT INTO users (name) VALUES ('Alice'), ('Bob');
	`
	_, err = conn.ExecContext(ctx, initialSQL)
	if err != nil {
		t.Fatalf("Failed to setup initial schema: %v", err)
	}

	// Verify initial state
	var count int
	err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query initial user count: %v", err)
	}
	if count != 2 {
		t.Fatalf("Expected 2 users initially, got %d", count)
	}

	// Verify no email column exists initially
	var emailColumnExists bool
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'users' AND column_name = 'email'
		)
	`).Scan(&emailColumnExists)
	if err != nil {
		t.Fatalf("Failed to check if email column exists: %v", err)
	}
	if emailColumnExists {
		t.Fatal("Email column should not exist initially")
	}

	// Create desired state schema file that will generate a failing migration
	tmpDir := t.TempDir()
	desiredStateFile := filepath.Join(tmpDir, "desired_state.sql")
	// This desired state will generate a migration that:
	// 1. Adds email column to users (valid)
	// 2. Creates a table with invalid SQL syntax (should cause rollback)
	desiredStateSQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255)
		);
		
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			user_id INTEGER REFERENCES nonexistent_users(id)
		);
	`
	err = os.WriteFile(desiredStateFile, []byte(desiredStateSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write desired state file: %v", err)
	}

	// Get container connection details
	containerHost := container.Host
	portMapped := container.Port

	// First, generate and verify the migration plan
	planConfig := &planCmd.PlanConfig{
		Host:            containerHost,
		Port:            portMapped,
		DB:              "testdb",
		User:            "testuser",
		Password:        "testpass",
		Schema:          "public",
		File:            desiredStateFile,
		ApplicationName: "pgschema",
	}

	migrationPlan, err := planCmd.GeneratePlan(planConfig)
	if err != nil {
		t.Fatalf("Failed to generate migration plan: %v", err)
	}

	// Verify the planned SQL contains the expected statements
	plannedSQL := migrationPlan.ToSQL(plan.SQLFormatRaw)
	t.Logf("Generated migration SQL:\n\n%s\n", plannedSQL)

	// Verify that the planned SQL contains our expected statements
	if !strings.Contains(plannedSQL, "CREATE TABLE IF NOT EXISTS products") {
		t.Fatalf("Expected migration to contain 'CREATE TABLE IF NOT EXISTS products', got: %s", plannedSQL)
	}
	if !strings.Contains(plannedSQL, "ALTER TABLE users ADD COLUMN email") {
		t.Fatalf("Expected migration to contain 'ALTER TABLE users ADD COLUMN email', got: %s", plannedSQL)
	}
	if !strings.Contains(plannedSQL, "REFERENCES nonexistent_users(id)") {
		t.Fatalf("Expected migration to contain foreign key reference to nonexistent_users, got: %s", plannedSQL)
	}

	t.Log("Migration plan verified - contains expected failing foreign key reference")

	// Create a new command instance to avoid flag conflicts
	cmd := &cobra.Command{}
	*cmd = *ApplyCmd

	// Set command arguments
	args := []string{
		"--host", containerHost,
		"--port", fmt.Sprintf("%d", portMapped),
		"--db", "testdb",
		"--user", "testuser",
		"--password", "testpass",
		"--file", desiredStateFile,
		"--auto-approve", // Skip interactive confirmation
	}
	cmd.SetArgs(args)

	// Run apply command - this should fail due to the invalid DDL
	err = cmd.Execute()
	if err == nil {
		t.Fatal("Expected apply command to fail due to invalid DDL, but it succeeded")
	}

	t.Logf("Apply command failed as expected with error: %v", err)

	// Verify that the database is still in the original state (transaction rolled back)
	// Check that email column was NOT added to users table
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'users' AND column_name = 'email'
		)
	`).Scan(&emailColumnExists)
	if err != nil {
		t.Fatalf("Failed to check if email column exists after failed apply: %v", err)
	}
	if emailColumnExists {
		t.Fatal("Email column should not exist after failed transaction - rollback did not work properly")
	}

	// Verify products table was not created
	var tableExists bool
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = 'products'
		)
	`).Scan(&tableExists)
	if err != nil {
		t.Fatalf("Failed to check if products table exists: %v", err)
	}
	if tableExists {
		t.Fatal("products table should not exist after failed transaction")
	}

	// Verify original data is still intact
	err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query user count after failed apply: %v", err)
	}
	if count != 2 {
		t.Fatalf("Expected 2 users after failed apply, got %d", count)
	}

	t.Log("Transaction rollback verified successfully - database remains in original state")
}

// TestApplyCommand_CreateIndexConcurrently verifies that CREATE INDEX CONCURRENTLY
// works correctly when mixed with other DDL statements.
//
// The plan detects non-transactional DDL (CREATE INDEX CONCURRENTLY) and executes
// all statements individually to avoid PostgreSQL's implicit transaction block.
//
// This test verifies that mixed transactional and non-transactional DDL can be
// applied successfully without the "cannot run inside a transaction block" error.
func TestApplyCommand_CreateIndexConcurrently(t *testing.T) {
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
			name VARCHAR(255) NOT NULL
		);
		
		INSERT INTO users (name) VALUES ('Alice'), ('Bob'), ('Charlie');
	`
	_, err = conn.ExecContext(ctx, initialSQL)
	if err != nil {
		t.Fatalf("Failed to setup initial schema: %v", err)
	}

	// Verify initial state
	var count int
	err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query initial user count: %v", err)
	}
	if count != 3 {
		t.Fatalf("Expected 3 users initially, got %d", count)
	}

	// Create desired state schema file that will generate a migration with mixed DDL
	tmpDir := t.TempDir()
	desiredStateFile := filepath.Join(tmpDir, "desired_state.sql")
	// This desired state will generate a migration that contains:
	// 1. ALTER TABLE to add email column (transactional)
	// 2. CREATE INDEX CONCURRENTLY on the email column (non-transactional)
	// 3. CREATE TABLE for products (transactional)
	desiredStateSQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE INDEX CONCURRENTLY idx_users_email ON public.users USING btree (email);
		CREATE INDEX CONCURRENTLY idx_users_created_at ON public.users USING btree (created_at);
		
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			price DECIMAL(10, 2)
		);
	`
	err = os.WriteFile(desiredStateFile, []byte(desiredStateSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write desired state file: %v", err)
	}

	// Get container connection details
	containerHost := container.Host
	portMapped := container.Port

	// First, generate and verify the migration plan
	planConfig := &planCmd.PlanConfig{
		Host:            containerHost,
		Port:            portMapped,
		DB:              "testdb",
		User:            "testuser",
		Password:        "testpass",
		Schema:          "public",
		File:            desiredStateFile,
		ApplicationName: "pgschema",
	}

	migrationPlan, err := planCmd.GeneratePlan(planConfig)
	if err != nil {
		t.Fatalf("Failed to generate migration plan: %v", err)
	}

	// Verify the planned SQL contains the expected statements
	plannedSQL := migrationPlan.ToSQL(plan.SQLFormatRaw)
	t.Logf("Generated migration SQL:\n%s", plannedSQL)

	// Verify that the planned SQL contains our expected statements
	if !strings.Contains(plannedSQL, "ALTER TABLE users ADD COLUMN email") {
		t.Fatalf("Expected migration to contain 'ALTER TABLE users ADD COLUMN email', got: %s", plannedSQL)
	}
	if !strings.Contains(plannedSQL, "ALTER TABLE users ADD COLUMN created_at") {
		t.Fatalf("Expected migration to contain 'ALTER TABLE users ADD COLUMN created_at', got: %s", plannedSQL)
	}
	if !strings.Contains(plannedSQL, "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email") {
		t.Fatalf("Expected migration to contain 'CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email', got: %s", plannedSQL)
	}
	if !strings.Contains(plannedSQL, "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_created_at") {
		t.Fatalf("Expected migration to contain 'CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_created_at', got: %s", plannedSQL)
	}
	if !strings.Contains(plannedSQL, "CREATE TABLE IF NOT EXISTS products") {
		t.Fatalf("Expected migration to contain 'CREATE TABLE IF NOT EXISTS products', got: %s", plannedSQL)
	}

	t.Log("Migration plan verified - contains mixed transactional and non-transactional DDL")

	// Create a new command instance to avoid flag conflicts
	cmd := &cobra.Command{}
	*cmd = *ApplyCmd

	// Set command arguments
	args := []string{
		"--host", containerHost,
		"--port", fmt.Sprintf("%d", portMapped),
		"--db", "testdb",
		"--user", "testuser",
		"--password", "testpass",
		"--file", desiredStateFile,
		"--auto-approve", // Skip interactive confirmation
	}
	cmd.SetArgs(args)

	// Run apply command - this should now succeed with individual statement execution
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Expected apply command to succeed, but it failed with error: %v", err)
	}

	t.Log("Apply command succeeded - CREATE INDEX CONCURRENTLY now works!")

	// Verify that all changes were applied successfully
	// Check that email column was added to users table
	var emailColumnExists bool
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'users' AND column_name = 'email'
		)
	`).Scan(&emailColumnExists)
	if err != nil {
		t.Fatalf("Failed to check if email column exists after apply: %v", err)
	}
	if !emailColumnExists {
		t.Fatal("Email column should exist after successful apply")
	}

	// Verify created_at column was added
	var createdAtColumnExists bool
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'users' AND column_name = 'created_at'
		)
	`).Scan(&createdAtColumnExists)
	if err != nil {
		t.Fatalf("Failed to check if created_at column exists: %v", err)
	}
	if !createdAtColumnExists {
		t.Fatal("created_at column should exist after successful apply")
	}

	// Verify products table was created
	var tableExists bool
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = 'products'
		)
	`).Scan(&tableExists)
	if err != nil {
		t.Fatalf("Failed to check if products table exists: %v", err)
	}
	if !tableExists {
		t.Fatal("products table should exist after successful apply")
	}

	// Verify indexes were created with CONCURRENTLY
	var indexCount int
	err = conn.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM pg_indexes 
		WHERE tablename = 'users' AND indexname LIKE 'idx_users_%'
	`).Scan(&indexCount)
	if err != nil {
		t.Fatalf("Failed to check index count: %v", err)
	}
	if indexCount != 2 {
		t.Fatalf("Expected 2 indexes to be created, but found %d", indexCount)
	}

	// Verify original data plus the new columns are intact
	err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query user count after apply: %v", err)
	}
	if count != 3 {
		t.Fatalf("Expected 3 users after apply, got %d", count)
	}

	// Verify we can insert data using the new columns
	_, err = conn.ExecContext(ctx, `
		INSERT INTO users (name, email, created_at) 
		VALUES ('Test User', 'test@example.com', NOW())
	`)
	if err != nil {
		t.Fatalf("Failed to insert data with new columns: %v", err)
	}
}
