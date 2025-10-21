package cmd

// Ignore Integration Tests
// These comprehensive integration tests verify the .pgschemaignore functionality
// across dump, plan, and apply commands by testing the complete workflow with
// various database object types and ignore patterns including wildcards and negation.

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/pgschema/pgschema/cmd/apply"
	"github.com/pgschema/pgschema/cmd/dump"
	planCmd "github.com/pgschema/pgschema/cmd/plan"
	"github.com/pgschema/pgschema/testutil"
	"github.com/spf13/cobra"
)

func TestIgnoreIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Setup PostgreSQL container
	containerInfo := testutil.SetupPostgresContainerWithDB(ctx, t, "testdb", "testuser", "testpass")
	defer containerInfo.Terminate(ctx, t)

	// Create the test schema with various object types
	createTestSchema(t, containerInfo.Conn)

	// Save current working directory and restore it at the end
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		os.Chdir(originalWd)
	}()

	// Create a temporary directory for our tests
	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Run sub-tests in isolated environments
	t.Run("dump", func(t *testing.T) {
		testIgnoreDump(t, containerInfo)
	})

	t.Run("plan", func(t *testing.T) {
		testIgnorePlan(t, containerInfo)
	})

	t.Run("plan_trigger_on_ignored_table", func(t *testing.T) {
		testIgnorePlanWithTriggerOnIgnoredTable(t, containerInfo)
	})

	t.Run("apply", func(t *testing.T) {
		// Create a fresh container for apply test to avoid fingerprint conflicts
		ctx := context.Background()
		applyContainerInfo := testutil.SetupPostgresContainerWithDB(ctx, t, "testdb", "testuser", "testpass")
		defer applyContainerInfo.Terminate(ctx, t)

		// Create the test schema in the fresh container
		createTestSchema(t, applyContainerInfo.Conn)

		testIgnoreApply(t, applyContainerInfo)
	})
}

// createTestSchema creates all test objects in the database
func createTestSchema(t *testing.T, conn *sql.DB) {
	testSQL := `
-- Create user status enum type
CREATE TYPE user_status AS ENUM ('active', 'inactive', 'suspended');

-- Create test enum type (to be ignored)
CREATE TYPE type_test_enum AS ENUM ('test1', 'test2');

-- Create regular tables
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    status user_status DEFAULT 'active'
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    total_amount DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    price DECIMAL(10,2) NOT NULL
);

-- Create temporary tables (to be ignored)
CREATE TABLE temp_backup (
    id SERIAL PRIMARY KEY,
    data TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE temp_cache (
    key TEXT PRIMARY KEY,
    value TEXT,
    expires_at TIMESTAMP
);

CREATE TABLE temp_session (
    session_id TEXT PRIMARY KEY,
    user_id INTEGER,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create test tables (to be ignored, except core)
CREATE TABLE test_data (
    id SERIAL PRIMARY KEY,
    test_value TEXT
);

CREATE TABLE test_results (
    id SERIAL PRIMARY KEY,
    result TEXT
);

-- Create test core table (NOT ignored due to negation pattern)
CREATE TABLE test_core_config (
    id SERIAL PRIMARY KEY,
    config_key TEXT NOT NULL,
    config_value TEXT NOT NULL
);

-- Create regular sequences
CREATE SEQUENCE user_id_seq;

-- Create temp sequence (to be ignored)
CREATE SEQUENCE seq_temp_counter;

-- Create regular views
CREATE VIEW user_orders_view AS
SELECT u.name, u.email, o.total_amount, o.created_at
FROM users u
JOIN orders o ON u.id = o.user_id;

CREATE VIEW product_summary AS
SELECT COUNT(*) as total_products, AVG(price) as avg_price
FROM products;

-- Create debug views (to be ignored)
CREATE VIEW debug_performance AS
SELECT 'debug_data' as info;

CREATE VIEW debug_stats AS
SELECT 'debug_stats' as stats;

-- Create temp view (to be ignored)
CREATE VIEW orders_view_tmp AS
SELECT * FROM orders WHERE created_at > NOW() - INTERVAL '1 hour';

-- Create regular functions
CREATE OR REPLACE FUNCTION get_user_count() RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION calculate_total(p_user_id INTEGER) RETURNS DECIMAL AS $$
BEGIN
    RETURN (SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE user_id = p_user_id);
END;
$$ LANGUAGE plpgsql;

-- Create test functions (to be ignored)
CREATE OR REPLACE FUNCTION fn_test_helper() RETURNS TEXT AS $$
BEGIN
    RETURN 'test helper';
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_debug_log(p_message TEXT) RETURNS VOID AS $$
BEGIN
    -- Debug function
    RETURN;
END;
$$ LANGUAGE plpgsql;

-- Create regular procedure
CREATE OR REPLACE PROCEDURE process_orders()
LANGUAGE plpgsql
AS $$
BEGIN
    -- Process orders logic
    UPDATE orders SET total_amount = total_amount * 1.1 WHERE total_amount > 100;
END;
$$;

-- Create temp procedure (to be ignored)
CREATE OR REPLACE PROCEDURE sp_temp_cleanup()
LANGUAGE plpgsql
AS $$
BEGIN
    DELETE FROM temp_cache WHERE expires_at < NOW();
END;
$$;

-- Create external table (to be ignored)
CREATE TABLE temp_external_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
`

	_, err := conn.Exec(testSQL)
	if err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}
}

// createIgnoreFile creates a .pgschemaignore file in the current directory
func createIgnoreFile(t *testing.T) func() {
	ignoreContent := `[tables]
patterns = ["temp_*", "test_*", "!test_core_*"]

[views]
patterns = ["debug_*", "*_view_tmp"]

[functions]
patterns = ["fn_test_*", "fn_debug_*"]

[procedures]
patterns = ["sp_temp_*"]

[types]
patterns = ["type_test_*"]

[sequences]
patterns = ["seq_temp_*"]
`

	err := os.WriteFile(".pgschemaignore", []byte(ignoreContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .pgschemaignore file: %v", err)
	}

	// Return cleanup function
	return func() {
		os.Remove(".pgschemaignore")
	}
}

// testIgnoreDump tests the dump command with ignore functionality
func testIgnoreDump(t *testing.T, containerInfo *testutil.ContainerInfo) {
	// Create .pgschemaignore file
	cleanup := createIgnoreFile(t)
	defer cleanup()

	// Execute dump command
	output := executeIgnoreDumpCommand(t, containerInfo)

	// Verify output contains expected objects and excludes ignored ones
	verifyDumpOutput(t, output)
}

// testIgnorePlanWithTriggerOnIgnoredTable tests that triggers can be defined on ignored tables
// This tests the scenario where users manage triggers on externally-managed tables
// without managing the table schema itself
func testIgnorePlanWithTriggerOnIgnoredTable(t *testing.T, containerInfo *testutil.ContainerInfo) {
	// Create .pgschemaignore file - temp_* pattern will ignore temp_external_users
	cleanup := createIgnoreFile(t)
	defer cleanup()

	// Create schema file that defines a trigger on the ignored external table
	schemaWithTrigger := `
-- Regular table managed by pgschema
CREATE TYPE user_status AS ENUM ('active', 'inactive', 'suspended');

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    status user_status DEFAULT 'active'
);

-- Trigger function for syncing external user profiles
CREATE OR REPLACE FUNCTION sync_external_user_profile()
RETURNS trigger AS $$
BEGIN
    -- Insert into user profiles when external user is created
    INSERT INTO users (name, email, status)
    VALUES ('External User', NEW.email, 'active');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Define trigger on ignored external table
CREATE TRIGGER on_external_user_created
    AFTER INSERT ON temp_external_users
    FOR EACH ROW
    EXECUTE FUNCTION sync_external_user_profile();
`

	schemaFile := "schema_with_trigger_on_ignored.sql"
	err := os.WriteFile(schemaFile, []byte(schemaWithTrigger), 0644)
	if err != nil {
		t.Fatalf("Failed to create schema file: %v", err)
	}
	defer os.Remove(schemaFile)

	// Execute plan command
	output := executeIgnorePlanCommand(t, containerInfo, schemaFile)

	// Verify that the plan doesn't attempt to manage the external table structure
	// The external table should not appear in CREATE/DROP TABLE statements
	if strings.Contains(output, "CREATE TABLE IF NOT EXISTS temp_external_users") ||
		strings.Contains(output, "DROP TABLE IF EXISTS temp_external_users") {
		t.Error("Plan should not create or drop external table temp_external_users")
	}

	// Verify that the trigger on the external table appears in the plan as being added
	// This proves we can manage triggers on external tables
	if !strings.Contains(output, "CREATE OR REPLACE TRIGGER on_external_user_created") {
		t.Error("Plan should include CREATE TRIGGER on_external_user_created for the external table")
	}
}

// testIgnorePlan tests the plan command with ignore functionality
func testIgnorePlan(t *testing.T, containerInfo *testutil.ContainerInfo) {
	// Create .pgschemaignore file
	cleanup := createIgnoreFile(t)
	defer cleanup()

	// Create a modified schema file with changes to both regular and ignored objects
	modifiedSchema := `
-- Modified regular table (should appear in plan)
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    status user_status DEFAULT 'active',
    phone TEXT -- NEW COLUMN
);

-- Modified ignored table (should NOT appear in plan)
CREATE TABLE temp_backup (
    id SERIAL PRIMARY KEY,
    data TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    backup_type TEXT -- NEW COLUMN - should be ignored
);

-- Keep test_core_config (should appear due to negation)
CREATE TABLE test_core_config (
    id SERIAL PRIMARY KEY,
    config_key TEXT NOT NULL,
    config_value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW() -- NEW COLUMN
);
`

	schemaFile := "modified_schema.sql"
	err := os.WriteFile(schemaFile, []byte(modifiedSchema), 0644)
	if err != nil {
		t.Fatalf("Failed to create modified schema file: %v", err)
	}
	defer os.Remove(schemaFile)

	// Execute plan command
	output := executeIgnorePlanCommand(t, containerInfo, schemaFile)

	// Verify plan output excludes ignored objects
	verifyPlanOutput(t, output)
}

// testIgnoreApply tests the apply command with ignore functionality
// This test verifies that ignored objects are excluded from fingerprint calculation
func testIgnoreApply(t *testing.T, containerInfo *testutil.ContainerInfo) {
	// Create .pgschemaignore file
	cleanup := createIgnoreFile(t)
	defer cleanup()

	// Verify that ignored objects exist before apply
	verifyIgnoredObjectsExist(t, containerInfo.Conn, "before apply")

	// Create a schema file with ONLY regular (non-ignored) objects
	// This schema does NOT include ignored objects like sp_temp_cleanup, temp_*, fn_test_*, etc.
	regularObjectsSchema := `
-- Regular enum type (not ignored)
CREATE TYPE user_status AS ENUM ('active', 'inactive', 'suspended');

-- Regular tables (not ignored)
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    status user_status DEFAULT 'active'
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    total_amount DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    price DECIMAL(10,2) NOT NULL
);

-- Keep test_core_config (not ignored due to negation pattern !test_core_*)
CREATE TABLE test_core_config (
    id SERIAL PRIMARY KEY,
    config_key TEXT NOT NULL,
    config_value TEXT NOT NULL
);

-- Regular sequence (not ignored)
CREATE SEQUENCE IF NOT EXISTS user_id_seq;

-- Regular views (not ignored)
CREATE VIEW user_orders_view AS
SELECT u.name, u.email, o.total_amount, o.created_at
FROM users u
JOIN orders o ON u.id = o.user_id;

CREATE VIEW product_summary AS
SELECT COUNT(*) as total_products, AVG(price) as avg_price
FROM products;

-- Regular functions (not ignored)
CREATE OR REPLACE FUNCTION get_user_count() RETURNS INTEGER AS $$
BEGIN
    RETURN (SELECT COUNT(*) FROM users);
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION calculate_total(p_user_id INTEGER) RETURNS DECIMAL AS $$
BEGIN
    RETURN (SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE user_id = p_user_id);
END;
$$ LANGUAGE plpgsql;

-- Regular procedure (not ignored)
CREATE OR REPLACE PROCEDURE process_orders()
LANGUAGE plpgsql
AS $$
BEGIN
    -- Process orders logic
    UPDATE orders SET total_amount = total_amount * 1.1 WHERE total_amount > 100;
END;
$$;
`

	schemaFile := "regular_objects_schema.sql"
	err := os.WriteFile(schemaFile, []byte(regularObjectsSchema), 0644)
	if err != nil {
		t.Fatalf("Failed to create schema file: %v", err)
	}
	defer os.Remove(schemaFile)

	// Execute apply command - should succeed because ignored objects are excluded from fingerprint
	err = executeIgnoreApplyCommandWithError(containerInfo, schemaFile)
	if err != nil {
		t.Fatalf("Apply command should succeed when ignored objects are excluded from fingerprint, but got error: %v", err)
	}

	// Verify that ignored objects still exist after apply (they should remain untouched)
	verifyIgnoredObjectsExist(t, containerInfo.Conn, "after apply")

	// Verify that the ignored procedure sp_temp_cleanup still exists
	verifyIgnoredProcedureExists(t, containerInfo.Conn, "after apply")
}

// executeIgnoreDumpCommand runs the dump command and returns the output
func executeIgnoreDumpCommand(t *testing.T, containerInfo *testutil.ContainerInfo) string {
	// Create a new root command with dump as subcommand
	rootCmd := &cobra.Command{
		Use: "pgschema",
	}
	rootCmd.AddCommand(dump.DumpCmd)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var output string
	done := make(chan bool)
	go func() {
		defer close(done)
		buf := make([]byte, 1024*1024) // 1MB buffer
		n, _ := r.Read(buf)
		output = string(buf[:n])
	}()

	// Set command arguments
	args := []string{
		"dump",
		"--host", containerInfo.Host,
		"--port", fmt.Sprintf("%d", containerInfo.Port),
		"--db", "testdb",
		"--user", "testuser",
		"--password", "testpass",
		"--schema", "public",
	}
	rootCmd.SetArgs(args)

	// Execute the command
	err := rootCmd.Execute()
	w.Close()
	os.Stdout = oldStdout
	<-done

	if err != nil {
		t.Fatalf("Failed to execute dump command: %v", err)
	}

	return output
}

// executeIgnorePlanCommand runs the plan command and returns the output
func executeIgnorePlanCommand(t *testing.T, containerInfo *testutil.ContainerInfo, schemaFile string) string {
	rootCmd := &cobra.Command{
		Use: "pgschema",
	}
	rootCmd.AddCommand(planCmd.PlanCmd)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var output string
	done := make(chan bool)
	go func() {
		defer close(done)
		buf := make([]byte, 1024*1024)
		n, _ := r.Read(buf)
		output = string(buf[:n])
	}()

	args := []string{
		"plan",
		"--host", containerInfo.Host,
		"--port", fmt.Sprintf("%d", containerInfo.Port),
		"--db", "testdb",
		"--user", "testuser",
		"--password", "testpass",
		"--schema", "public",
		"--file", schemaFile,
	}
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	w.Close()
	os.Stdout = oldStdout
	<-done

	if err != nil {
		t.Fatalf("Failed to execute plan command: %v", err)
	}

	return output
}

// executeIgnoreApplyCommandWithError runs the apply command and returns any error
func executeIgnoreApplyCommandWithError(containerInfo *testutil.ContainerInfo, schemaFile string) error {
	rootCmd := &cobra.Command{
		Use: "pgschema",
	}
	rootCmd.AddCommand(apply.ApplyCmd)

	args := []string{
		"apply",
		"--host", containerInfo.Host,
		"--port", fmt.Sprintf("%d", containerInfo.Port),
		"--db", "testdb",
		"--user", "testuser",
		"--password", "testpass",
		"--schema", "public",
		"--file", schemaFile,
		"--auto-approve",
	}
	rootCmd.SetArgs(args)

	return rootCmd.Execute()
}

// verifyIgnoredObjectsExist checks that ignored objects still exist in the database
func verifyIgnoredObjectsExist(t *testing.T, conn *sql.DB, phase string) {
	// Check that temp_backup table still exists (should be ignored)
	var tempTableExists bool
	err := conn.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_name = 'temp_backup'
			AND table_schema = 'public'
		)
	`).Scan(&tempTableExists)

	if err != nil {
		t.Fatalf("Failed to check temp_backup table existence %s: %v", phase, err)
	}

	if !tempTableExists {
		t.Errorf("temp_backup table should exist %s (ignored tables should remain unchanged)", phase)
	}

	// Check that test_data table still exists (should be ignored)
	var testTableExists bool
	err = conn.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_name = 'test_data'
			AND table_schema = 'public'
		)
	`).Scan(&testTableExists)

	if err != nil {
		t.Fatalf("Failed to check test_data table existence %s: %v", phase, err)
	}

	if !testTableExists {
		t.Errorf("test_data table should exist %s (ignored tables should remain unchanged)", phase)
	}
}

// verifyIgnoredProcedureExists checks that the ignored procedure sp_temp_cleanup still exists
func verifyIgnoredProcedureExists(t *testing.T, conn *sql.DB, phase string) {
	var procedureExists bool
	err := conn.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.routines
			WHERE routine_name = 'sp_temp_cleanup'
			AND routine_schema = 'public'
			AND routine_type = 'PROCEDURE'
		)
	`).Scan(&procedureExists)

	if err != nil {
		t.Fatalf("Failed to check sp_temp_cleanup procedure existence %s: %v", phase, err)
	}

	if !procedureExists {
		t.Errorf("sp_temp_cleanup procedure should exist %s (ignored procedures should remain unchanged)", phase)
	}
}

// verifyDumpOutput checks that dump output contains expected objects and excludes ignored ones
func verifyDumpOutput(t *testing.T, output string) {
	// Objects that should be present (not ignored)
	expectedPresent := []string{
		"CREATE TABLE IF NOT EXISTS users",
		"CREATE TABLE IF NOT EXISTS orders",
		"CREATE TABLE IF NOT EXISTS products",
		"CREATE TABLE IF NOT EXISTS test_core_config", // Not ignored due to negation
		"CREATE OR REPLACE VIEW user_orders_view",
		"CREATE OR REPLACE VIEW product_summary",
		"CREATE OR REPLACE FUNCTION get_user_count",
		"CREATE OR REPLACE FUNCTION calculate_total",
		"CREATE OR REPLACE PROCEDURE process_orders",
		"CREATE TYPE user_status",
		"CREATE SEQUENCE IF NOT EXISTS user_id_seq",
	}

	// Objects that should be absent (ignored)
	expectedAbsent := []string{
		"temp_backup",
		"temp_cache",
		"temp_session",
		"test_data",
		"test_results",
		"debug_performance",
		"debug_stats",
		"orders_view_tmp",
		"fn_test_helper",
		"fn_debug_log",
		"sp_temp_cleanup",
		"type_test_enum",
		"seq_temp_counter",
	}

	// Check for expected present objects
	for _, expected := range expectedPresent {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected object not found in dump output: %s", expected)
		}
	}

	// Check for expected absent objects
	for _, unexpected := range expectedAbsent {
		if strings.Contains(output, unexpected) {
			t.Errorf("Ignored object found in dump output (should be excluded): %s", unexpected)
		}
	}
}

// verifyPlanOutput checks that plan output excludes ignored objects
func verifyPlanOutput(t *testing.T, output string) {
	// Changes that should appear in plan (regular objects)
	expectedInPlan := []string{
		"users",            // Should show column addition
		"test_core_config", // Not ignored due to negation
	}

	// Changes that should NOT appear in plan (ignored objects)
	expectedNotInPlan := []string{
		"temp_backup", // Should be ignored
	}

	// Check that regular objects appear in plan
	for _, expected := range expectedInPlan {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected object not found in plan output: %s", expected)
		}
	}

	// Check that ignored objects don't appear in plan
	for _, unexpected := range expectedNotInPlan {
		if strings.Contains(output, unexpected) {
			t.Errorf("Ignored object found in plan output (should be excluded): %s", unexpected)
		}
	}
}
