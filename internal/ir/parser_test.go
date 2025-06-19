package ir

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestParseSQL_Employee(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test parser with employee dataset
	runParserIntegrationTest(t, "employee")
}

func TestParseSQL_Bytebase(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test parser with bytebase dataset
	runParserIntegrationTest(t, "bytebase")
}

func runParserIntegrationTest(t *testing.T, testDataDir string) {
	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
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
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Get connection string
	testDSN, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Connect to database and load schema
	db, err := sql.Open("pgx", testDSN)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Read and execute the pgdump.sql file to populate the database
	pgdumpPath := fmt.Sprintf("../../testdata/%s/pgdump.sql", testDataDir)
	pgdumpContent, err := os.ReadFile(pgdumpPath)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", pgdumpPath, err)
	}

	// Execute the SQL to create the schema
	_, err = db.ExecContext(ctx, string(pgdumpContent))
	if err != nil {
		t.Fatalf("Failed to execute pgdump.sql: %v", err)
	}

	// TODO: Use the existing inspect command to get the database IR
	// For now, we'll just test that the parser can parse the pgschema.sql file

	// Read the pgschema.sql file (our canonical output format)
	pgschemaPath := fmt.Sprintf("../../testdata/%s/pgschema.sql", testDataDir)
	pgschemaContent, err := os.ReadFile(pgschemaPath)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", pgschemaPath, err)
	}

	// Parse the SQL content using our parser
	parser := NewParser()
	schema, err := parser.ParseSQL(string(pgschemaContent))
	if err != nil {
		t.Fatalf("Failed to parse SQL: %v", err)
	}

	// Basic validation - ensure we have some schemas and tables
	if len(schema.Schemas) == 0 {
		t.Error("Expected to parse at least one schema")
	}

	// Check if we have the public schema
	publicSchema, exists := schema.Schemas["public"]
	if !exists {
		t.Error("Expected to find public schema")
	} else {
		if len(publicSchema.Tables) == 0 {
			t.Error("Expected to find at least one table in public schema")
		}

		t.Logf("Successfully parsed %d schemas", len(schema.Schemas))
		t.Logf("Public schema has %d tables", len(publicSchema.Tables))

		// Log table names for debugging
		for tableName := range publicSchema.Tables {
			t.Logf("Found table: %s", tableName)
		}
	}

	// For debugging, save the parsed schema to a file
	debugSchemaPath := fmt.Sprintf("%s_parsed_schema.json", testDataDir)
	if schemaJSON, err := json.MarshalIndent(schema, "", "  "); err == nil {
		if err := os.WriteFile(debugSchemaPath, schemaJSON, 0644); err == nil {
			t.Logf("Debug: Parsed schema written to %s", debugSchemaPath)
		}
	}
}

func TestParseSQL_BasicTable(t *testing.T) {
	// Test basic table parsing
	sql := `
CREATE TABLE public.test_table (
    id integer NOT NULL,
    name text NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE ONLY public.test_table
    ADD CONSTRAINT test_table_pkey PRIMARY KEY (id);
`

	parser := NewParser()
	schema, err := parser.ParseSQL(sql)
	if err != nil {
		t.Fatalf("Failed to parse basic table SQL: %v", err)
	}

	// Validate schema
	if len(schema.Schemas) != 1 {
		t.Errorf("Expected 1 schema, got %d", len(schema.Schemas))
	}

	publicSchema, exists := schema.Schemas["public"]
	if !exists {
		t.Fatal("Expected public schema to exist")
	}

	if len(publicSchema.Tables) != 1 {
		t.Errorf("Expected 1 table, got %d", len(publicSchema.Tables))
	}

	table, exists := publicSchema.Tables["test_table"]
	if !exists {
		t.Fatal("Expected test_table to exist")
	}

	// Validate table structure
	if table.Schema != "public" {
		t.Errorf("Expected schema 'public', got '%s'", table.Schema)
	}

	if table.Name != "test_table" {
		t.Errorf("Expected name 'test_table', got '%s'", table.Name)
	}

	if len(table.Columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(table.Columns))
	}

	// Check specific columns
	expectedColumns := map[string]struct {
		position int
		dataType string
		nullable bool
	}{
		"id":         {1, "integer", false},
		"name":       {2, "text", false},
		"created_at": {3, "timestamp with time zone", true}, // DEFAULT makes it nullable unless NOT NULL is explicit
	}

	for _, col := range table.Columns {
		expected, exists := expectedColumns[col.Name]
		if !exists {
			t.Errorf("Unexpected column: %s", col.Name)
			continue
		}

		if col.Position != expected.position {
			t.Errorf("Column %s: expected position %d, got %d", col.Name, expected.position, col.Position)
		}

		if col.DataType != expected.dataType {
			t.Errorf("Column %s: expected type %s, got %s", col.Name, expected.dataType, col.DataType)
		}
	}

	t.Logf("Successfully parsed basic table with %d columns", len(table.Columns))
}

// Helper function to compare two schemas deeply
func compareSchemasDeep(t *testing.T, expected, actual *Schema) {
	// Compare schema count
	if len(expected.Schemas) != len(actual.Schemas) {
		t.Errorf("Schema count mismatch: expected %d, got %d", len(expected.Schemas), len(actual.Schemas))
		return
	}

	// Compare each schema
	for schemaName, expectedSchema := range expected.Schemas {
		actualSchema, exists := actual.Schemas[schemaName]
		if !exists {
			t.Errorf("Schema %s not found in actual result", schemaName)
			continue
		}

		// Compare tables
		if len(expectedSchema.Tables) != len(actualSchema.Tables) {
			t.Errorf("Schema %s: table count mismatch: expected %d, got %d",
				schemaName, len(expectedSchema.Tables), len(actualSchema.Tables))
		}

		for tableName, expectedTable := range expectedSchema.Tables {
			actualTable, exists := actualSchema.Tables[tableName]
			if !exists {
				t.Errorf("Schema %s: table %s not found in actual result", schemaName, tableName)
				continue
			}

			// Compare table properties
			if expectedTable.Name != actualTable.Name {
				t.Errorf("Table name mismatch: expected %s, got %s", expectedTable.Name, actualTable.Name)
			}

			if expectedTable.Schema != actualTable.Schema {
				t.Errorf("Table schema mismatch: expected %s, got %s", expectedTable.Schema, actualTable.Schema)
			}

			// Compare columns
			if len(expectedTable.Columns) != len(actualTable.Columns) {
				t.Errorf("Table %s: column count mismatch: expected %d, got %d",
					tableName, len(expectedTable.Columns), len(actualTable.Columns))
			}

			// Deep comparison of columns would go here...
		}
	}
}
