package util

import (
	"context"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

func TestStartEmbeddedPostgres(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping embedded postgres test in short mode")
	}

	config := &EmbeddedPostgresConfig{
		Version:  embeddedpostgres.PostgresVersion("17.5.0"),
		Database: "testdb",
		Username: "testuser",
		Password: "testpass",
	}

	ep, err := StartEmbeddedPostgres(config)
	if err != nil {
		t.Fatalf("Failed to start embedded postgres: %v", err)
	}
	defer ep.Stop()

	// Test connection
	ctx := context.Background()
	if err := ep.GetDB().PingContext(ctx); err != nil {
		t.Fatalf("Failed to ping embedded postgres: %v", err)
	}

	// Verify connection info
	host, port, database := ep.GetConnectionInfo()
	if host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", host)
	}
	if port == 0 {
		t.Error("Port should not be 0")
	}
	if database != "testdb" {
		t.Errorf("Expected database 'testdb', got '%s'", database)
	}
}

func TestApplySchemaSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping embedded postgres test in short mode")
	}

	config := &EmbeddedPostgresConfig{
		Version:  embeddedpostgres.PostgresVersion("17.5.0"),
		Database: "testdb",
		Username: "testuser",
		Password: "testpass",
	}

	ep, err := StartEmbeddedPostgres(config)
	if err != nil {
		t.Fatalf("Failed to start embedded postgres: %v", err)
	}
	defer ep.Stop()

	ctx := context.Background()

	// Test applying schema SQL
	schemaSQL := `
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT UNIQUE
		);

		CREATE INDEX idx_users_email ON users(email);
	`

	err = ep.ApplySchemaSQL(ctx, "public", schemaSQL)
	if err != nil {
		t.Fatalf("Failed to apply schema SQL: %v", err)
	}

	// Verify table was created
	var tableName string
	query := "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'users'"
	err = ep.GetDB().QueryRowContext(ctx, query).Scan(&tableName)
	if err != nil {
		t.Fatalf("Failed to query table: %v", err)
	}
	if tableName != "users" {
		t.Errorf("Expected table 'users', got '%s'", tableName)
	}

	// Verify index was created
	var indexName string
	query = "SELECT indexname FROM pg_indexes WHERE tablename = 'users' AND indexname = 'idx_users_email'"
	err = ep.GetDB().QueryRowContext(ctx, query).Scan(&indexName)
	if err != nil {
		t.Fatalf("Failed to query index: %v", err)
	}
	if indexName != "idx_users_email" {
		t.Errorf("Expected index 'idx_users_email', got '%s'", indexName)
	}
}

func TestApplySchemaSQL_CustomSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping embedded postgres test in short mode")
	}

	config := &EmbeddedPostgresConfig{
		Version:  embeddedpostgres.PostgresVersion("17.5.0"),
		Database: "testdb",
		Username: "testuser",
		Password: "testpass",
	}

	ep, err := StartEmbeddedPostgres(config)
	if err != nil {
		t.Fatalf("Failed to start embedded postgres: %v", err)
	}
	defer ep.Stop()

	ctx := context.Background()

	// Test applying schema SQL to custom schema
	schemaSQL := `
		CREATE TABLE products (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			price NUMERIC(10, 2)
		);
	`

	err = ep.ApplySchemaSQL(ctx, "myschema", schemaSQL)
	if err != nil {
		t.Fatalf("Failed to apply schema SQL: %v", err)
	}

	// Verify schema was created
	var schemaName string
	query := "SELECT schema_name FROM information_schema.schemata WHERE schema_name = 'myschema'"
	err = ep.GetDB().QueryRowContext(ctx, query).Scan(&schemaName)
	if err != nil {
		t.Fatalf("Failed to query schema: %v", err)
	}
	if schemaName != "myschema" {
		t.Errorf("Expected schema 'myschema', got '%s'", schemaName)
	}

	// Verify table was created in custom schema
	var tableName string
	query = "SELECT table_name FROM information_schema.tables WHERE table_schema = 'myschema' AND table_name = 'products'"
	err = ep.GetDB().QueryRowContext(ctx, query).Scan(&tableName)
	if err != nil {
		t.Fatalf("Failed to query table: %v", err)
	}
	if tableName != "products" {
		t.Errorf("Expected table 'products', got '%s'", tableName)
	}
}

func TestApplySchemaSQL_InvalidSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping embedded postgres test in short mode")
	}

	config := &EmbeddedPostgresConfig{
		Version:  embeddedpostgres.PostgresVersion("17.5.0"),
		Database: "testdb",
		Username: "testuser",
		Password: "testpass",
	}

	ep, err := StartEmbeddedPostgres(config)
	if err != nil {
		t.Fatalf("Failed to start embedded postgres: %v", err)
	}
	defer ep.Stop()

	ctx := context.Background()

	// Test with invalid SQL
	invalidSQL := "CREATE TABLE invalid syntax here"
	err = ep.ApplySchemaSQL(ctx, "public", invalidSQL)
	if err == nil {
		t.Error("Expected error for invalid SQL, got none")
	}
}

func TestStop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping embedded postgres test in short mode")
	}

	config := &EmbeddedPostgresConfig{
		Version:  embeddedpostgres.PostgresVersion("17.5.0"),
		Database: "testdb",
		Username: "testuser",
		Password: "testpass",
	}

	ep, err := StartEmbeddedPostgres(config)
	if err != nil {
		t.Fatalf("Failed to start embedded postgres: %v", err)
	}

	// Test stopping
	err = ep.Stop()
	if err != nil {
		t.Fatalf("Failed to stop embedded postgres: %v", err)
	}

	// Verify connection is closed
	ctx := context.Background()
	err = ep.GetDB().PingContext(ctx)
	if err == nil {
		t.Error("Expected error pinging stopped postgres, got none")
	}
}
