package cmd

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestPlanCommand_DatabaseToDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start two PostgreSQL containers
	container1, err := postgres.Run(ctx,
		"postgres:17",
		postgres.WithDatabase("db1"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Fatalf("Failed to start container 1: %v", err)
	}
	defer func() {
		if err := container1.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container 1: %v", err)
		}
	}()

	container2, err := postgres.Run(ctx,
		"postgres:17",
		postgres.WithDatabase("db2"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Fatalf("Failed to start container 2: %v", err)
	}
	defer func() {
		if err := container2.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container 2: %v", err)
		}
	}()

	// Setup database 1 with initial schema
	db1DSN, err := container1.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string for db1: %v", err)
	}
	conn1, err := sql.Open("pgx", db1DSN)
	if err != nil {
		t.Fatalf("Failed to connect to db1: %v", err)
	}
	defer conn1.Close()

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
	db2DSN, err := container2.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string for db2: %v", err)
	}
	conn2, err := sql.Open("pgx", db2DSN)
	if err != nil {
		t.Fatalf("Failed to connect to db2: %v", err)
	}
	defer conn2.Close()

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
	containerHost1, err := container1.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container1 host: %v", err)
	}
	port1Mapped, err := container1.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get container1 port: %v", err)
	}

	containerHost2, err := container2.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container2 host: %v", err)
	}
	port2Mapped, err := container2.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get container2 port: %v", err)
	}

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
	port1 = port1Mapped.Int()
	db1 = "db1"
	user1 = "testuser"
	host2 = containerHost2
	port2 = port2Mapped.Int()
	db2 = "db2"
	user2 = "testuser"
	format = "text"

	// Set password via environment variable
	os.Setenv("PGPASSWORD", "testpass")

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

	// Start PostgreSQL container
	container, err := postgres.Run(ctx,
		"postgres:17",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Setup database with schema
	dbDSN, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}
	db, err := sql.Open("pgx", dbDSN)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

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
	hostContainer, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}
	portMapped, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

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
	port2 = portMapped.Int()
	db2 = "testdb"
	user2 = "testuser"
	format = "text"

	// Set password via environment variable
	os.Setenv("PGPASSWORD", "testpass")

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
	
	schema1File := filepath.Join(tmpDir, "schema1.sql")
	schema1SQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		);
	`
	err := os.WriteFile(schema1File, []byte(schema1SQL), 0644)
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
			err = runPlan(PlanCmd, []string{})
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

	// Start PostgreSQL container
	container, err := postgres.Run(ctx,
		"postgres:17",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Setup database with multiple schemas
	dbDSN, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}
	db, err := sql.Open("pgx", dbDSN)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

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
	hostContainer, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}
	portMapped, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

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
	port2 = portMapped.Int()
	db2 = "testdb"
	user2 = "testuser"
	schema2 = "public" // Filter to only public schema
	format = "text"

	// Set password via environment variable
	os.Setenv("PGPASSWORD", "testpass")

	// Run plan command
	err = runPlan(PlanCmd, []string{})
	if err != nil {
		t.Fatalf("Plan command failed with schema filtering: %v", err)
	}

	t.Log("Plan command executed successfully with schema filtering")
}