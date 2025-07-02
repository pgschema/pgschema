package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pgschema/pgschema/testutil"
)

func TestPlanCommand_DatabaseToDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	var err error

	// Start two PostgreSQL containers
	container1 := testutil.SetupPostgresContainerWithDB(ctx, t, "db1", "testuser", "testpass")
	defer container1.Terminate(ctx, t)

	container2 := testutil.SetupPostgresContainerWithDB(ctx, t, "db2", "testuser", "testpass")
	defer container2.Terminate(ctx, t)

	// Setup database 1 with initial schema
	conn1 := container1.Conn

	schema1SQL := `
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
	_, err = conn1.ExecContext(ctx, schema1SQL)
	if err != nil {
		t.Fatalf("Failed to setup schema in db1: %v", err)
	}

	// Setup database 2 with modified schema
	conn2 := container2.Conn

	schema2SQL := `
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
	_, err = conn2.ExecContext(ctx, schema2SQL)
	if err != nil {
		t.Fatalf("Failed to setup schema in db2: %v", err)
	}

	// Get container connection details
	containerHost1 := container1.Host
	port1Mapped := container1.Port

	containerHost2 := container2.Host
	port2Mapped := container2.Port

	// Save original values
	originalHost1 := host1
	originalPort1 := port1
	originalDb1 := db1
	originalUser1 := user1
	originalHost2 := host2
	originalPort2 := port2
	originalDb2 := db2
	originalUser2 := user2
	originalFormat := format

	defer func() {
		host1 = originalHost1
		port1 = originalPort1
		db1 = originalDb1
		user1 = originalUser1
		host2 = originalHost2
		port2 = originalPort2
		db2 = originalDb2
		user2 = originalUser2
		format = originalFormat
	}()

	// Set connection parameters for plan command
	host1 = containerHost1
	port1 = port1Mapped
	db1 = "db1"
	user1 = "testuser"
	host2 = containerHost2
	port2 = port2Mapped
	db2 = "db2"
	user2 = "testuser"
	format = "text"

	// Set password via environment variable
	testutil.SetEnvPassword("testpass")

	// Run plan command
	err = runPlan(PlanCmd, []string{})
	if err != nil {
		t.Fatalf("Plan command failed: %v", err)
	}

	// The plan should succeed - we don't check exact output since it's complex,
	// but we verify the command runs without error
	t.Log("Plan command executed successfully for database-to-database comparison")
}

func TestPlanCommand_FileToDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	var err error

	// Start PostgreSQL container
	container := testutil.SetupPostgresContainer(ctx, t)
	defer container.Terminate(ctx, t)

	// Setup database with schema
	db := container.Conn

	databaseSQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) UNIQUE,
			created_at TIMESTAMP DEFAULT NOW()
		);
	`
	_, err = db.ExecContext(ctx, databaseSQL)
	if err != nil {
		t.Fatalf("Failed to setup database schema: %v", err)
	}

	// Create temporary schema file
	tmpDir := t.TempDir()
	schemaFile := filepath.Join(tmpDir, "schema.sql")
	schemaFileSQL := `
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
	err = os.WriteFile(schemaFile, []byte(schemaFileSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}

	// Get container connection details
	hostContainer := container.Host
	portMapped := container.Port

	// Save original values
	originalFile1 := file1
	originalHost2 := host2
	originalPort2 := port2
	originalDb2 := db2
	originalUser2 := user2
	originalFormat := format

	defer func() {
		file1 = originalFile1
		host2 = originalHost2
		port2 = originalPort2
		db2 = originalDb2
		user2 = originalUser2
		format = originalFormat
	}()

	// Set parameters for plan command (file to database)
	file1 = schemaFile
	host2 = hostContainer
	port2 = portMapped
	db2 = "testdb"
	user2 = "testuser"
	format = "text"

	// Set password via environment variable
	testutil.SetEnvPassword("testpass")

	// Run plan command
	err = runPlan(PlanCmd, []string{})
	if err != nil {
		t.Fatalf("Plan command failed: %v", err)
	}

	t.Log("Plan command executed successfully for file-to-database comparison")
}

func TestPlanCommand_FileToFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary schema files
	tmpDir := t.TempDir()
	var err error
	
	schema1File := filepath.Join(tmpDir, "schema1.sql")
	schema1SQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		);
	`
	err = os.WriteFile(schema1File, []byte(schema1SQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write schema1 file: %v", err)
	}

	schema2File := filepath.Join(tmpDir, "schema2.sql")
	schema2SQL := `
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
	err = os.WriteFile(schema2File, []byte(schema2SQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write schema2 file: %v", err)
	}

	// Save original values
	originalFile1 := file1
	originalFile2 := file2
	originalFormat := format

	defer func() {
		file1 = originalFile1
		file2 = originalFile2
		format = originalFormat
	}()

	// Test different output formats
	testCases := []struct {
		name   string
		format string
	}{
		{"text format", "text"},
		{"json format", "json"},
		{"preview format", "preview"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set parameters for plan command
			file1 = schema1File
			file2 = schema2File
			format = tc.format

			// Run plan command
			err := runPlan(PlanCmd, []string{})
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
	container := testutil.SetupPostgresContainer(ctx, t)
	defer container.Terminate(ctx, t)

	// Setup database with multiple schemas
	db := container.Conn

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
	_, err = db.ExecContext(ctx, multiSchemaSQL)
	if err != nil {
		t.Fatalf("Failed to setup multi-schema database: %v", err)
	}

	// Create schema file with only public schema content
	tmpDir := t.TempDir()
	publicSchemaFile := filepath.Join(tmpDir, "public_schema.sql")
	publicSchemaSQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) UNIQUE
		);
	`
	err = os.WriteFile(publicSchemaFile, []byte(publicSchemaSQL), 0644)
	if err != nil {
		t.Fatalf("Failed to write public schema file: %v", err)
	}

	// Get container connection details
	hostContainer := container.Host
	portMapped := container.Port

	// Save original values
	originalFile1 := file1
	originalHost2 := host2
	originalPort2 := port2
	originalDb2 := db2
	originalUser2 := user2
	originalSchema2 := schema2
	originalFormat := format

	defer func() {
		file1 = originalFile1
		host2 = originalHost2
		port2 = originalPort2
		db2 = originalDb2
		user2 = originalUser2
		schema2 = originalSchema2
		format = originalFormat
	}()

	// Set parameters for plan command with schema filtering
	file1 = publicSchemaFile
	host2 = hostContainer
	port2 = portMapped
	db2 = "testdb"
	user2 = "testuser"
	schema2 = "public" // Filter to only public schema
	format = "text"

	// Set password via environment variable
	testutil.SetEnvPassword("testpass")

	// Run plan command
	err = runPlan(PlanCmd, []string{})
	if err != nil {
		t.Fatalf("Plan command failed with schema filtering: %v", err)
	}

	t.Log("Plan command executed successfully with schema filtering")
}