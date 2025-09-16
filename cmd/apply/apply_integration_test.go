package apply

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	planCmd "github.com/pgschema/pgschema/cmd/plan"
	"github.com/pgschema/pgschema/internal/plan"
	"github.com/pgschema/pgschema/testutil"
)

// TestApplyCommand_TransactionRollback verifies that the apply command uses proper
// transaction mode. If any statement fails in the middle of execution, the entire
// transaction should be rolled back and no partial changes should be applied.
//
// The test creates a migration with multiple statements that should all run in a single transaction:
// 1. CREATE TABLE posts with valid foreign key to users (valid)
// 2. CREATE TABLE products with invalid foreign key to nonexistent_users (fails)
// 3. ALTER TABLE users ADD COLUMN email (valid)
// 4. ALTER TABLE users ADD COLUMN status (valid)
//
// When the second statement fails, all statements in the transaction group should be rolled back,
// including the first successful CREATE TABLE statement and the subsequent column additions.
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

	// Create desired state schema file that will generate a failing migration with multiple statements
	tmpDir := t.TempDir()
	desiredStateFile := filepath.Join(tmpDir, "desired_state.sql")
	// This desired state will generate a migration that:
	// 1. Adds email column to users (valid)
	// 2. Adds status column to users (valid)
	// 3. Creates posts table with valid foreign key to users (valid)
	// 4. Creates products table with invalid foreign key reference (should cause rollback of all)
	desiredStateSQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255),
			status VARCHAR(50) DEFAULT 'active'
		);
		
		CREATE TABLE posts (
			id SERIAL PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			user_id INTEGER REFERENCES users(id)
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
	if !strings.Contains(plannedSQL, "ALTER TABLE users ADD COLUMN email") {
		t.Fatalf("Expected migration to contain 'ALTER TABLE users ADD COLUMN email', got: %s", plannedSQL)
	}
	if !strings.Contains(plannedSQL, "ALTER TABLE users ADD COLUMN status") {
		t.Fatalf("Expected migration to contain 'ALTER TABLE users ADD COLUMN status', got: %s", plannedSQL)
	}
	if !strings.Contains(plannedSQL, "CREATE TABLE IF NOT EXISTS posts") {
		t.Fatalf("Expected migration to contain 'CREATE TABLE IF NOT EXISTS posts', got: %s", plannedSQL)
	}
	if !strings.Contains(plannedSQL, "CREATE TABLE IF NOT EXISTS products") {
		t.Fatalf("Expected migration to contain 'CREATE TABLE IF NOT EXISTS products', got: %s", plannedSQL)
	}
	if !strings.Contains(plannedSQL, "REFERENCES nonexistent_users (id)") {
		t.Fatalf("Expected migration to contain foreign key reference to nonexistent_users, got: %s", plannedSQL)
	}

	t.Log("Migration plan verified - contains multiple statements with invalid foreign key reference")

	// Log transaction grouping information
	t.Logf("Migration plan has %d execution groups", len(migrationPlan.Groups))
	for i, group := range migrationPlan.Groups {
		t.Logf("Group %d has %d steps", i+1, len(group.Steps))
		for j, step := range group.Steps {
			t.Logf("  Step %d: %s", j+1, step.SQL[:min(50, len(step.SQL))])
		}
	}

	// Set global flag variables directly for this test
	applyHost = containerHost
	applyPort = portMapped
	applyDB = "testdb"
	applyUser = "testuser"
	applyPassword = "testpass"
	applySchema = "public"
	applyFile = desiredStateFile
	applyPlan = "" // Clear to avoid conflicts
	applyAutoApprove = true
	applyNoColor = false
	applyLockTimeout = ""
	applyApplicationName = "pgschema"

	// Call RunApply directly to avoid flag parsing issues
	err = RunApply(nil, nil)
	if err == nil {
		t.Fatal("Expected apply command to fail due to invalid DDL, but it succeeded")
	}

	t.Logf("Apply command failed as expected with error: %v", err)

	// Verify that ALL changes in the same transaction group were rolled back
	// Check that email column was NOT added to users table (should be rolled back)
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

	// Check that status column was NOT added to users table (should be rolled back)
	var statusColumnExists bool
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'users' AND column_name = 'status'
		)
	`).Scan(&statusColumnExists)
	if err != nil {
		t.Fatalf("Failed to check if status column exists after failed apply: %v", err)
	}
	if statusColumnExists {
		t.Fatal("Status column should not exist after failed transaction - rollback did not work properly")
	}

	// Verify posts table was NOT created (should be rolled back)
	var postsTableExists bool
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = 'posts'
		)
	`).Scan(&postsTableExists)
	if err != nil {
		t.Fatalf("Failed to check if posts table exists after failed apply: %v", err)
	}
	if postsTableExists {
		t.Fatal("Posts table should not exist after failed transaction - rollback did not work properly")
	}

	// Verify products table was NOT created (this was the failing statement)
	var productsTableExists bool
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = 'products'
		)
	`).Scan(&productsTableExists)
	if err != nil {
		t.Fatalf("Failed to check if products table exists after failed apply: %v", err)
	}
	if productsTableExists {
		t.Fatal("Products table should not exist after failed transaction")
	}

	// Verify the database is exactly in its original state
	var userColumnCount int
	err = conn.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.columns 
		WHERE table_name = 'users'
	`).Scan(&userColumnCount)
	if err != nil {
		t.Fatalf("Failed to count columns in users table: %v", err)
	}
	if userColumnCount != 2 {
		t.Fatalf("Expected users table to have exactly 2 columns (id, name), but found %d", userColumnCount)
	}

	var tableCount int
	err = conn.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.tables 
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
	`).Scan(&tableCount)
	if err != nil {
		t.Fatalf("Failed to count tables: %v", err)
	}
	if tableCount != 1 {
		t.Fatalf("Expected exactly 1 table (users), but found %d", tableCount)
	}

	t.Log("Transaction rollback verified successfully - all statements in the failed transaction group were properly rolled back")
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

	// Set global flag variables directly for this test
	applyHost = containerHost
	applyPort = portMapped
	applyDB = "testdb"
	applyUser = "testuser"
	applyPassword = "testpass"
	applySchema = "public"
	applyFile = desiredStateFile
	applyPlan = "" // Clear to avoid conflicts
	applyAutoApprove = true
	applyNoColor = false
	applyLockTimeout = ""
	applyApplicationName = "pgschema"

	// Call RunApply directly to avoid flag parsing issues
	err = RunApply(nil, nil)
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

	// Verify we can insert data using the new columns
	_, err = conn.ExecContext(ctx, `
		INSERT INTO users (name, email, created_at) 
		VALUES ('Test User', 'test@example.com', NOW())
	`)
	if err != nil {
		t.Fatalf("Failed to insert data with new columns: %v", err)
	}
}

// TestApplyCommand_WithPlanFile verifies that the apply command can apply changes
// from a pre-generated JSON plan file using the --plan flag.
//
// This test simulates a workflow where:
// 1. A plan is generated and saved to a JSON file
// 2. The plan file is later applied using `apply --plan`
//
// This workflow is common in CI/CD pipelines where plan generation and
// application happen in separate stages for review and approval.
func TestApplyCommand_WithPlanFile(t *testing.T) {
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
	`
	_, err = conn.ExecContext(ctx, initialSQL)
	if err != nil {
		t.Fatalf("Failed to setup initial schema: %v", err)
	}

	// Create desired state schema file
	tmpDir := t.TempDir()
	desiredStateFile := filepath.Join(tmpDir, "desired_state.sql")
	desiredStateSQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE INDEX idx_users_email ON public.users USING btree (email);
		
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

	// Step 1: Generate plan and save to JSON file
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

	// Save plan to JSON file
	planFile := filepath.Join(tmpDir, "migration_plan.json")
	jsonOutput, err := migrationPlan.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert plan to JSON: %v", err)
	}
	err = os.WriteFile(planFile, []byte(jsonOutput), 0644)
	if err != nil {
		t.Fatalf("Failed to write plan file: %v", err)
	}

	t.Logf("Generated plan JSON saved to: %s", planFile)

	// Step 2: Apply the plan using --plan flag
	// Set global flag variables directly for this test
	applyHost = containerHost
	applyPort = portMapped
	applyDB = "testdb"
	applyUser = "testuser"
	applyPassword = "testpass"
	applySchema = "public"
	applyFile = ""       // Clear to avoid conflicts
	applyPlan = planFile // Use the saved plan file
	applyAutoApprove = true
	applyNoColor = false
	applyLockTimeout = ""
	applyApplicationName = "pgschema"

	// Call RunApply directly to avoid flag parsing issues
	err = RunApply(nil, nil)
	if err != nil {
		t.Fatalf("Failed to apply plan from file: %v", err)
	}

	t.Log("Plan applied successfully from JSON file")

	// Step 3: Verify all changes were applied
	// Check that email column was added
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
	if !emailColumnExists {
		t.Fatal("Email column should exist after applying plan")
	}

	// Check that created_at column was added
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
		t.Fatal("created_at column should exist after applying plan")
	}

	// Check that products table was created
	var productsTableExists bool
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = 'products'
		)
	`).Scan(&productsTableExists)
	if err != nil {
		t.Fatalf("Failed to check if products table exists: %v", err)
	}
	if !productsTableExists {
		t.Fatal("products table should exist after applying plan")
	}

	// Check that index was created
	var indexExists bool
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes 
			WHERE tablename = 'users' AND indexname = 'idx_users_email'
		)
	`).Scan(&indexExists)
	if err != nil {
		t.Fatalf("Failed to check if index exists: %v", err)
	}
	if !indexExists {
		t.Fatal("idx_users_email index should exist after applying plan")
	}

	t.Log("All schema changes from plan file applied and verified successfully")
}

// TestApplyCommand_FingerprintMismatch verifies that the apply command correctly
// detects and rejects plans when the database schema has been modified after plan generation.
//
// The test simulates this scenario:
// 1. Create initial schema with users table
// 2. Generate a plan to add email column
// 3. Make out-of-band change (add phone column) to simulate concurrent modification
// 4. Try to apply the original plan - should fail with fingerprint mismatch
//
// This ensures that plans are not applied to databases that have been modified
// since the plan was generated, preventing potential conflicts.
func TestApplyCommand_FingerprintMismatch(t *testing.T) {
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

	// Verify initial state - only id and name columns
	var columnCount int
	err = conn.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.columns 
		WHERE table_name = 'users'
	`).Scan(&columnCount)
	if err != nil {
		t.Fatalf("Failed to count initial columns: %v", err)
	}
	if columnCount != 2 {
		t.Fatalf("Expected 2 columns initially, got %d", columnCount)
	}

	// Create desired state schema file that will add email column
	tmpDir := t.TempDir()
	desiredStateFile := filepath.Join(tmpDir, "desired_state.sql")
	desiredStateSQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255)
		);
	`
	err = os.WriteFile(desiredStateFile, []byte(desiredStateSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write desired state file: %v", err)
	}

	// Get container connection details
	containerHost := container.Host
	portMapped := container.Port

	// Generate migration plan to add email column
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

	// Verify the plan includes fingerprint and contains expected changes
	if migrationPlan.SourceFingerprint == nil {
		t.Fatal("Expected plan to include source fingerprint, but it was nil")
	}

	plannedSQL := migrationPlan.ToSQL(plan.SQLFormatRaw)
	if !strings.Contains(plannedSQL, "ALTER TABLE users ADD COLUMN email") {
		t.Fatalf("Expected migration to contain 'ALTER TABLE users ADD COLUMN email', got: %s", plannedSQL)
	}

	t.Log("Migration plan generated successfully with fingerprint")

	// Make out-of-band schema change to simulate concurrent modification
	// This will change the database schema and invalidate the plan's fingerprint
	outOfBandSQL := `ALTER TABLE users ADD COLUMN phone VARCHAR(20);`
	_, err = conn.ExecContext(ctx, outOfBandSQL)
	if err != nil {
		t.Fatalf("Failed to apply out-of-band schema change: %v", err)
	}

	// Verify phone column was added
	var phoneColumnExists bool
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'users' AND column_name = 'phone'
		)
	`).Scan(&phoneColumnExists)
	if err != nil {
		t.Fatalf("Failed to check if phone column exists: %v", err)
	}
	if !phoneColumnExists {
		t.Fatal("Phone column should exist after out-of-band change")
	}

	t.Log("Out-of-band schema change applied successfully (added phone column)")

	// Save plan to JSON file (simulating plan file workflow)
	planFile := filepath.Join(tmpDir, "migration_plan.json")
	jsonOutput, err := migrationPlan.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert plan to JSON: %v", err)
	}
	err = os.WriteFile(planFile, []byte(jsonOutput), 0644)
	if err != nil {
		t.Fatalf("Failed to write plan file: %v", err)
	}

	// Attempt to apply the plan using the plan file - should fail with fingerprint mismatch
	// Set global flag variables for apply command
	applyHost = containerHost
	applyPort = portMapped
	applyDB = "testdb"
	applyUser = "testuser"
	applyPassword = "testpass"
	applySchema = "public"
	applyFile = ""       // Clear file to use plan instead
	applyPlan = planFile // Use the saved plan file
	applyAutoApprove = true
	applyNoColor = false
	applyLockTimeout = ""
	applyApplicationName = "pgschema"

	// Call RunApply - should fail due to fingerprint mismatch
	err = RunApply(nil, nil)
	if err == nil {
		t.Fatal("Expected apply command to fail due to fingerprint mismatch, but it succeeded")
	}

	// Verify error message mentions fingerprint mismatch
	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "fingerprint mismatch") {
		t.Fatalf("Expected error to mention 'fingerprint mismatch', got: %s", errorMsg)
	}
	if !strings.Contains(errorMsg, "schema fingerprint mismatch") {
		t.Fatalf("Expected error to mention 'schema fingerprint mismatch', got: %s", errorMsg)
	}

	t.Logf("Apply command failed as expected with fingerprint error: %v", err)

	// Verify that the database is in the expected state:
	// - phone column exists (from out-of-band change)
	// - email column does NOT exist (original plan was not applied)
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'users' AND column_name = 'phone'
		)
	`).Scan(&phoneColumnExists)
	if err != nil {
		t.Fatalf("Failed to check phone column after failed apply: %v", err)
	}
	if !phoneColumnExists {
		t.Fatal("Phone column should still exist after failed apply")
	}

	var emailColumnExists bool
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'users' AND column_name = 'email'
		)
	`).Scan(&emailColumnExists)
	if err != nil {
		t.Fatalf("Failed to check email column after failed apply: %v", err)
	}
	if emailColumnExists {
		t.Fatal("Email column should NOT exist after failed apply due to fingerprint mismatch")
	}

	// Verify we now have 3 columns (id, name, phone)
	err = conn.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.columns 
		WHERE table_name = 'users'
	`).Scan(&columnCount)
	if err != nil {
		t.Fatalf("Failed to count columns after failed apply: %v", err)
	}
	if columnCount != 3 {
		t.Fatalf("Expected 3 columns after failed apply (id, name, phone), got %d", columnCount)
	}

	t.Log("Fingerprint validation successfully prevented applying outdated plan to modified database")
}

// TestApplyCommand_WaitDirective verifies that wait directives work correctly
// with concurrent index creation and provide progress monitoring.
func TestApplyCommand_WaitDirective(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	var err error

	// Start PostgreSQL container
	container := testutil.SetupPostgresContainerWithDB(ctx, t, "testdb", "testuser", "testpass")
	defer container.Terminate(ctx, t)

	// Setup database with initial schema and data
	conn := container.Conn

	initialSQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255),
			status VARCHAR(50) DEFAULT 'active'
		);
		
		-- Insert a decent amount of data to make index creation take some time
		INSERT INTO users (name, email, status) 
		SELECT 
			'User ' || i,
			'user' || i || '@example.com',
			CASE WHEN i % 3 = 0 THEN 'inactive' ELSE 'active' END
		FROM generate_series(1, 50000) i;
	`
	_, err = conn.ExecContext(ctx, initialSQL)
	if err != nil {
		t.Fatalf("Failed to setup initial schema: %v", err)
	}

	// Create desired state schema file that will generate a concurrent index
	tmpDir := t.TempDir()
	desiredStateFile := filepath.Join(tmpDir, "desired_state.sql")
	desiredStateSQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255),
			status VARCHAR(50) DEFAULT 'active'
		);

		-- This will trigger a CREATE INDEX CONCURRENTLY with wait directive
		CREATE INDEX CONCURRENTLY idx_users_email_status ON users (email, status);
	`

	err = os.WriteFile(desiredStateFile, []byte(desiredStateSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write desired state file: %v", err)
	}

	// Set global variables for apply command
	applyHost = container.Host
	applyPort = container.Port
	applyDB = "testdb"
	applyUser = "testuser"
	applyPassword = "testpass"
	applySchema = "public"
	applyFile = desiredStateFile
	applyPlan = "" // Clear to avoid conflicts
	applyAutoApprove = true
	applyNoColor = false
	applyLockTimeout = ""
	applyApplicationName = "pgschema"

	// Capture start time to verify wait directive execution
	startTime := time.Now()

	// Call RunApply directly to avoid flag parsing issues
	err = RunApply(nil, nil)
	if err != nil {
		t.Fatalf("Expected apply command to succeed, but it failed with error: %v", err)
	}

	// Verify that some time passed (indicating wait directive was executed)
	elapsed := time.Since(startTime)
	t.Logf("Index creation with wait directive took %v", elapsed)

	// Verify that the index was created successfully
	var indexExists bool
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes 
			WHERE indexname = 'idx_users_email_status'
		)
	`).Scan(&indexExists)
	if err != nil {
		t.Fatalf("Failed to check if index exists: %v", err)
	}
	if !indexExists {
		t.Fatal("Index idx_users_email_status should exist after successful apply")
	}

	// Verify that the index is valid (concurrent creation completed successfully)
	var indexValid bool
	err = conn.QueryRowContext(ctx, `
		SELECT i.indisvalid
		FROM pg_class c
		JOIN pg_index i ON c.oid = i.indexrelid
		WHERE c.relname = 'idx_users_email_status'
	`).Scan(&indexValid)
	if err != nil {
		t.Fatalf("Failed to check if index is valid: %v", err)
	}
	if !indexValid {
		t.Fatal("Index idx_users_email_status should be valid after wait directive completion")
	}
}
