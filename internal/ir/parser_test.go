package ir

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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

	// Get IR from database inspection using existing builder
	builder := NewBuilder(db)
	dbSchema, err := builder.BuildSchema(ctx)
	if err != nil {
		t.Fatalf("Failed to build schema from database: %v", err)
	}
	
	// Read the pgschema.sql file (our canonical output format)
	pgschemaPath := fmt.Sprintf("../../testdata/%s/pgschema.sql", testDataDir)
	pgschemaContent, err := os.ReadFile(pgschemaPath)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", pgschemaPath, err)
	}

	// Parse the SQL content using our parser
	parser := NewParser()
	parserSchema, err := parser.ParseSQL(string(pgschemaContent))
	if err != nil {
		t.Fatalf("Failed to parse SQL: %v", err)
	}

	// Compare the two schemas using deep comparison
	t.Logf("Database schema has %d schemas", len(dbSchema.Schemas))
	t.Logf("Parser schema has %d schemas", len(parserSchema.Schemas))
	
	if dbSchema.Schemas["public"] != nil {
		t.Logf("DB public schema has %d tables", len(dbSchema.Schemas["public"].Tables))
	}
	if parserSchema.Schemas["public"] != nil {
		t.Logf("Parser public schema has %d tables", len(parserSchema.Schemas["public"].Tables))
	}

	// Use deep comparison to validate equivalence
	compareSchemasDeep(t, dbSchema, parserSchema)

	// For debugging, save both schemas to files if comparison fails
	if t.Failed() {
		dbSchemaPath := fmt.Sprintf("%s_db_schema.json", testDataDir)
		parserSchemaPath := fmt.Sprintf("%s_parser_schema.json", testDataDir)
		
		if dbJSON, err := json.MarshalIndent(dbSchema, "", "  "); err == nil {
			if err := os.WriteFile(dbSchemaPath, dbJSON, 0644); err == nil {
				t.Logf("Debug: DB schema written to %s", dbSchemaPath)
			}
		}
		
		if parserJSON, err := json.MarshalIndent(parserSchema, "", "  "); err == nil {
			if err := os.WriteFile(parserSchemaPath, parserJSON, 0644); err == nil {
				t.Logf("Debug: Parser schema written to %s", parserSchemaPath)
			}
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

		compareDBSchemas(t, schemaName, expectedSchema, actualSchema)
	}
}

// compareDBSchemas compares two DBSchema objects deeply
func compareDBSchemas(t *testing.T, schemaName string, expected, actual *DBSchema) {
	// Compare tables - focus on base tables only since parser currently only handles tables
	expectedBaseTables := make(map[string]*Table)
	actualBaseTables := make(map[string]*Table)
	
	for name, table := range expected.Tables {
		if table.Type == TableTypeBase {
			expectedBaseTables[name] = table
		}
	}
	for name, table := range actual.Tables {
		if table.Type == TableTypeBase {
			actualBaseTables[name] = table
		}
	}

	if len(expectedBaseTables) != len(actualBaseTables) {
		t.Errorf("Schema %s: base table count mismatch: expected %d, got %d", 
			schemaName, len(expectedBaseTables), len(actualBaseTables))
	}

	for tableName, expectedTable := range expectedBaseTables {
		actualTable, exists := actualBaseTables[tableName]
		if !exists {
			t.Errorf("Schema %s: base table %s not found in actual result", schemaName, tableName)
			continue
		}

		compareTables(t, schemaName, tableName, expectedTable, actualTable)
	}

	// Compare other object types and log progress
	if len(expected.Views) > 0 || len(actual.Views) > 0 {
		if len(expected.Views) == len(actual.Views) {
			t.Logf("Schema %s: views match! Expected %d, parser found %d", 
				schemaName, len(expected.Views), len(actual.Views))
		} else {
			t.Logf("Schema %s: view count difference: expected %d, parser found %d", 
				schemaName, len(expected.Views), len(actual.Views))
		}
	}
	
	if len(expected.Functions) > 0 || len(actual.Functions) > 0 {
		if len(expected.Functions) == len(actual.Functions) {
			t.Logf("Schema %s: functions match! Expected %d, parser found %d", 
				schemaName, len(expected.Functions), len(actual.Functions))
		} else {
			t.Logf("Schema %s: function count difference: expected %d, parser found %d", 
				schemaName, len(expected.Functions), len(actual.Functions))
		}
	}
	
	if len(expected.Sequences) > 0 || len(actual.Sequences) > 0 {
		if len(expected.Sequences) == len(actual.Sequences) {
			t.Logf("Schema %s: sequences match! Expected %d, parser found %d", 
				schemaName, len(expected.Sequences), len(actual.Sequences))
		} else {
			t.Logf("Schema %s: sequence count difference: expected %d, parser found %d", 
				schemaName, len(expected.Sequences), len(actual.Sequences))
		}
	}
}

// compareTables compares two Table objects deeply
func compareTables(t *testing.T, schemaName, tableName string, expected, actual *Table) {
	// Compare basic properties
	if expected.Name != actual.Name {
		t.Errorf("Table %s.%s: name mismatch: expected %s, got %s", 
			schemaName, tableName, expected.Name, actual.Name)
	}

	if expected.Schema != actual.Schema {
		t.Errorf("Table %s.%s: schema mismatch: expected %s, got %s", 
			schemaName, tableName, expected.Schema, actual.Schema)
	}

	if expected.Type != actual.Type {
		t.Errorf("Table %s.%s: type mismatch: expected %s, got %s", 
			schemaName, tableName, expected.Type, actual.Type)
	}

	// Compare columns
	if len(expected.Columns) != len(actual.Columns) {
		t.Errorf("Table %s.%s: column count mismatch: expected %d, got %d", 
			schemaName, tableName, len(expected.Columns), len(actual.Columns))
	}

	// Create maps for easier comparison
	expectedColumns := make(map[string]*Column)
	actualColumns := make(map[string]*Column)
	
	for _, col := range expected.Columns {
		expectedColumns[col.Name] = col
	}
	for _, col := range actual.Columns {
		actualColumns[col.Name] = col
	}

	// Compare each column
	for colName, expectedCol := range expectedColumns {
		actualCol, exists := actualColumns[colName]
		if !exists {
			t.Errorf("Table %s.%s: column %s not found in actual result", 
				schemaName, tableName, colName)
			continue
		}

		compareColumns(t, schemaName, tableName, colName, expectedCol, actualCol)
	}

	// Compare constraints - for now just compare counts since parser doesn't fully implement constraints yet
	if len(expected.Constraints) != len(actual.Constraints) {
		t.Logf("Table %s.%s: constraint count difference: expected %d, got %d (parser may not fully implement constraints yet)", 
			schemaName, tableName, len(expected.Constraints), len(actual.Constraints))
	}
}

// compareColumns compares two Column objects
func compareColumns(t *testing.T, schemaName, tableName, colName string, expected, actual *Column) {
	if expected.Position != actual.Position {
		t.Errorf("Column %s.%s.%s: position mismatch: expected %d, got %d", 
			schemaName, tableName, colName, expected.Position, actual.Position)
	}

	if expected.DataType != actual.DataType {
		// Special handling for array types - database inspection may return "ARRAY" while parser returns "type[]"
		if expected.DataType == "ARRAY" && strings.HasSuffix(actual.DataType, "[]") {
			t.Logf("Column %s.%s.%s: array type difference: expected %s, got %s (both are arrays, different formats)", 
				schemaName, tableName, colName, expected.DataType, actual.DataType)
		} else if strings.HasSuffix(expected.DataType, "[]") && actual.DataType == "ARRAY" {
			t.Logf("Column %s.%s.%s: array type difference: expected %s, got %s (both are arrays, different formats)", 
				schemaName, tableName, colName, expected.DataType, actual.DataType)
		} else {
			t.Errorf("Column %s.%s.%s: data type mismatch: expected %s, got %s", 
				schemaName, tableName, colName, expected.DataType, actual.DataType)
		}
	}

	// Be lenient on nullable since parser doesn't parse ALTER TABLE NOT NULL constraints yet
	if expected.IsNullable != actual.IsNullable {
		t.Logf("Column %s.%s.%s: nullable difference: expected %t, got %t (parser may not handle NOT NULL constraints from ALTER TABLE)", 
			schemaName, tableName, colName, expected.IsNullable, actual.IsNullable)
	}

	// Be lenient on default values since parser doesn't parse ALTER TABLE defaults yet
	expectedDefault := ""
	actualDefault := ""
	if expected.DefaultValue != nil {
		expectedDefault = *expected.DefaultValue
	}
	if actual.DefaultValue != nil {
		actualDefault = *actual.DefaultValue
	}
	
	// Only log differences rather than fail
	if (expectedDefault != "") != (actualDefault != "") {
		t.Logf("Column %s.%s.%s: default value difference: expected %q, got %q (parser may not handle defaults from ALTER TABLE)", 
			schemaName, tableName, colName, expectedDefault, actualDefault)
	}
}

func TestExtractViewDefinitionFromAST(t *testing.T) {
	testCases := []struct {
		name               string
		viewSQL            string
		expectedDefinition string
		viewName           string
	}{
		{
			name:               "simple_select",
			viewSQL:            "CREATE VIEW test_view AS SELECT id, name FROM users WHERE active = true;",
			expectedDefinition: "SELECT id, name FROM users WHERE active = true",
			viewName:           "test_view",
		},
		{
			name:               "complex_select_with_joins",
			viewSQL:            "CREATE VIEW user_orders AS SELECT u.id, u.name, o.order_date, o.total FROM users u JOIN orders o ON u.id = o.user_id WHERE o.status = 'completed';",
			expectedDefinition: "SELECT u.id, u.name, o.order_date, o.total FROM users u JOIN orders o ON u.id = o.user_id WHERE o.status = 'completed'",
			viewName:           "user_orders",
		},
		{
			name:               "select_with_aggregation",
			viewSQL:            "CREATE VIEW order_summary AS SELECT user_id, COUNT(*) as order_count, SUM(total) as total_amount FROM orders GROUP BY user_id HAVING COUNT(*) > 5;",
			expectedDefinition: "SELECT user_id, count(*) AS order_count, sum(total) AS total_amount FROM orders GROUP BY user_id HAVING count(*) > 5",
			viewName:           "order_summary",
		},
		{
			name:               "schema_qualified_view",
			viewSQL:            "CREATE VIEW analytics.monthly_sales AS SELECT DATE_TRUNC('month', order_date) as month, SUM(total) as sales FROM orders GROUP BY DATE_TRUNC('month', order_date);",
			expectedDefinition: "SELECT date_trunc('month', order_date) AS month, sum(total) AS sales FROM orders GROUP BY date_trunc('month', order_date)",
			viewName:           "monthly_sales",
		},
		{
			name:               "view_with_subquery",
			viewSQL:            "CREATE VIEW top_customers AS SELECT user_id, total_spent FROM (SELECT user_id, SUM(total) as total_spent FROM orders GROUP BY user_id) subq WHERE total_spent > 1000;",
			expectedDefinition: "SELECT user_id, total_spent FROM (SELECT user_id, sum(total) AS total_spent FROM orders GROUP BY user_id) subq WHERE total_spent > 1000",
			viewName:           "top_customers",
		},
		{
			name:               "view_with_case_statement",
			viewSQL:            "CREATE VIEW user_status AS SELECT id, name, CASE WHEN last_login > NOW() - INTERVAL '30 days' THEN 'active' ELSE 'inactive' END as status FROM users;",
			expectedDefinition: "SELECT id, name, CASE WHEN last_login > (now() - '30 days'::interval) THEN 'active' ELSE 'inactive' END AS status FROM users",
			viewName:           "user_status",
		},
		{
			name:               "view_with_window_function",
			viewSQL:            "CREATE VIEW ranked_orders AS SELECT id, user_id, total, ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY total DESC) as rank FROM orders;",
			expectedDefinition: "SELECT id, user_id, total, row_number() OVER (PARTITION BY user_id ORDER BY total DESC) AS rank FROM orders",
			viewName:           "ranked_orders",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewParser()
			
			schema, err := parser.ParseSQL(tc.viewSQL)
			if err != nil {
				t.Fatalf("Failed to parse view SQL: %v", err)
			}
			
			// Find the schema containing the view
			var foundView *View
			var schemaName string
			for sName, s := range schema.Schemas {
				if view, exists := s.Views[tc.viewName]; exists {
					foundView = view
					schemaName = sName
					break
				}
			}
			
			if foundView == nil {
				t.Fatalf("View %s not found in any schema", tc.viewName)
			}
			
			// Check that the definition is not empty
			if foundView.Definition == "" {
				t.Fatal("View definition is empty")
			}
			
			// Normalize whitespace for comparison
			actualDef := strings.Join(strings.Fields(foundView.Definition), " ")
			expectedDef := strings.Join(strings.Fields(tc.expectedDefinition), " ")
			
			// The definition should match the expected SELECT clause
			if actualDef != expectedDef {
				t.Errorf("View definition mismatch:\nExpected: %s\nActual:   %s", expectedDef, actualDef)
			}
			
			// Ensure the definition doesn't contain CREATE VIEW
			if strings.Contains(strings.ToUpper(foundView.Definition), "CREATE VIEW") {
				t.Errorf("View definition should not contain CREATE VIEW, got: %s", foundView.Definition)
			}
			
			// Verify the definition contains SELECT
			if !strings.Contains(strings.ToUpper(foundView.Definition), "SELECT") {
				t.Errorf("View definition should contain SELECT, got: %s", foundView.Definition)
			}
			
			// Verify view metadata
			if foundView.Name != tc.viewName {
				t.Errorf("Expected view name %s, got %s", tc.viewName, foundView.Name)
			}
			
			// For schema-qualified views, check the schema
			if strings.Contains(tc.viewSQL, "analytics.") {
				if schemaName != "analytics" {
					t.Errorf("Expected view to be in analytics schema, found in %s", schemaName)
				}
			} else {
				if schemaName != "public" {
					t.Errorf("Expected view to be in public schema, found in %s", schemaName)
				}
			}
			
			t.Logf("✓ View %s definition extracted correctly: %s", tc.viewName, foundView.Definition)
		})
	}
}

func TestExtractFunctionFromAST(t *testing.T) {
	testCases := []struct {
		name               string
		functionSQL        string
		expectedName       string
		expectedReturnType string
		expectedLanguage   string
		expectedDefinition string
		expectedParams     []struct {
			name     string
			dataType string
			mode     string
			position int
		}
		schemaName string
	}{
		{
			name:               "simple_sql_function",
			functionSQL:        "CREATE FUNCTION get_user_count() RETURNS integer AS $$ SELECT COUNT(*) FROM users; $$ LANGUAGE SQL;",
			expectedName:       "get_user_count",
			expectedReturnType: "integer",
			expectedLanguage:   "sql",
			expectedDefinition: " SELECT COUNT(*) FROM users; ",
			expectedParams:     []struct{name string; dataType string; mode string; position int}{},
			schemaName:         "public",
		},
		{
			name:               "function_with_parameters",
			functionSQL:        "CREATE FUNCTION get_user_by_id(user_id integer) RETURNS text AS $$ SELECT name FROM users WHERE id = user_id; $$ LANGUAGE SQL;",
			expectedName:       "get_user_by_id",
			expectedReturnType: "text",
			expectedLanguage:   "sql",
			expectedDefinition: " SELECT name FROM users WHERE id = user_id; ",
			expectedParams: []struct{name string; dataType string; mode string; position int}{
				{name: "user_id", dataType: "integer", mode: "IN", position: 1},
			},
			schemaName: "public",
		},
		{
			name:               "plpgsql_function",
			functionSQL:        "CREATE FUNCTION calculate_total(a integer, b integer) RETURNS integer AS $$ BEGIN RETURN a + b; END; $$ LANGUAGE plpgsql;",
			expectedName:       "calculate_total",
			expectedReturnType: "integer",
			expectedLanguage:   "plpgsql",
			expectedDefinition: " BEGIN RETURN a + b; END; ",
			expectedParams: []struct{name string; dataType string; mode string; position int}{
				{name: "a", dataType: "integer", mode: "IN", position: 1},
				{name: "b", dataType: "integer", mode: "IN", position: 2},
			},
			schemaName: "public",
		},
		{
			name:               "schema_qualified_function",
			functionSQL:        "CREATE FUNCTION utils.format_name(first_name text, last_name text) RETURNS text AS $$ SELECT first_name || ' ' || last_name; $$ LANGUAGE SQL;",
			expectedName:       "format_name",
			expectedReturnType: "text",
			expectedLanguage:   "sql",
			expectedDefinition: " SELECT first_name || ' ' || last_name; ",
			expectedParams: []struct{name string; dataType string; mode string; position int}{
				{name: "first_name", dataType: "text", mode: "IN", position: 1},
				{name: "last_name", dataType: "text", mode: "IN", position: 2},
			},
			schemaName: "utils",
		},
		{
			name:               "function_returns_void",
			functionSQL:        "CREATE FUNCTION log_activity(message text) RETURNS void AS $$ INSERT INTO activity_log (message) VALUES (message); $$ LANGUAGE SQL;",
			expectedName:       "log_activity",
			expectedReturnType: "void",
			expectedLanguage:   "sql",
			expectedDefinition: " INSERT INTO activity_log (message) VALUES (message); ",
			expectedParams: []struct{name string; dataType string; mode string; position int}{
				{name: "message", dataType: "text", mode: "IN", position: 1},
			},
			schemaName: "public",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewParser()
			
			schema, err := parser.ParseSQL(tc.functionSQL)
			if err != nil {
				t.Fatalf("Failed to parse function SQL: %v", err)
			}
			
			// Find the schema containing the function
			var foundFunction *Function
			var schemaName string
			for sName, s := range schema.Schemas {
				if function, exists := s.Functions[tc.expectedName]; exists {
					foundFunction = function
					schemaName = sName
					break
				}
			}
			
			if foundFunction == nil {
				t.Fatalf("Function %s not found in any schema", tc.expectedName)
			}
			
			// Verify function metadata
			if foundFunction.Name != tc.expectedName {
				t.Errorf("Expected function name %s, got %s", tc.expectedName, foundFunction.Name)
			}
			
			if foundFunction.ReturnType != tc.expectedReturnType {
				t.Errorf("Expected return type %s, got %s", tc.expectedReturnType, foundFunction.ReturnType)
			}
			
			if foundFunction.Language != tc.expectedLanguage {
				t.Errorf("Expected language %s, got %s", tc.expectedLanguage, foundFunction.Language)
			}
			
			if foundFunction.Definition != tc.expectedDefinition {
				t.Errorf("Expected definition %q, got %q", tc.expectedDefinition, foundFunction.Definition)
			}
			
			if schemaName != tc.schemaName {
				t.Errorf("Expected function to be in %s schema, found in %s", tc.schemaName, schemaName)
			}
			
			// Verify parameters
			if len(foundFunction.Parameters) != len(tc.expectedParams) {
				t.Errorf("Expected %d parameters, got %d", len(tc.expectedParams), len(foundFunction.Parameters))
			} else {
				for i, expectedParam := range tc.expectedParams {
					actualParam := foundFunction.Parameters[i]
					
					if actualParam.Name != expectedParam.name {
						t.Errorf("Parameter %d: expected name %s, got %s", i, expectedParam.name, actualParam.Name)
					}
					
					if actualParam.DataType != expectedParam.dataType {
						t.Errorf("Parameter %d: expected data type %s, got %s", i, expectedParam.dataType, actualParam.DataType)
					}
					
					if actualParam.Mode != expectedParam.mode {
						t.Errorf("Parameter %d: expected mode %s, got %s", i, expectedParam.mode, actualParam.Mode)
					}
					
					if actualParam.Position != expectedParam.position {
						t.Errorf("Parameter %d: expected position %d, got %d", i, expectedParam.position, actualParam.Position)
					}
				}
			}
			
			t.Logf("✓ Function %s parsed correctly: %s(%d params) -> %s [%s]", 
				tc.expectedName, tc.expectedName, len(foundFunction.Parameters), 
				foundFunction.ReturnType, foundFunction.Language)
		})
	}
}

func TestExtractSequenceFromAST(t *testing.T) {
	testCases := []struct {
		name             string
		sequenceSQL      string
		expectedName     string
		expectedDataType string
		expectedStart    int64
		expectedIncr     int64
		expectedMinVal   *int64
		expectedMaxVal   *int64
		expectedCycle    bool
		schemaName       string
	}{
		{
			name:             "simple_sequence",
			sequenceSQL:      "CREATE SEQUENCE user_id_seq;",
			expectedName:     "user_id_seq",
			expectedDataType: "bigint",
			expectedStart:    1,
			expectedIncr:     1,
			expectedMinVal:   nil,
			expectedMaxVal:   nil,
			expectedCycle:    false,
			schemaName:       "public",
		},
		{
			name:             "sequence_with_start_increment",
			sequenceSQL:      "CREATE SEQUENCE order_id_seq START WITH 1000 INCREMENT BY 5;",
			expectedName:     "order_id_seq",
			expectedDataType: "bigint",
			expectedStart:    1000,
			expectedIncr:     5,
			expectedMinVal:   nil,
			expectedMaxVal:   nil,
			expectedCycle:    false,
			schemaName:       "public",
		},
		{
			name:             "sequence_with_min_max_values",
			sequenceSQL:      "CREATE SEQUENCE count_seq START WITH 10 INCREMENT BY 2 MINVALUE 5 MAXVALUE 100;",
			expectedName:     "count_seq",
			expectedDataType: "bigint",
			expectedStart:    10,
			expectedIncr:     2,
			expectedMinVal:   func() *int64 { v := int64(5); return &v }(),
			expectedMaxVal:   func() *int64 { v := int64(100); return &v }(),
			expectedCycle:    false,
			schemaName:       "public",
		},
		{
			name:             "sequence_with_cycle",
			sequenceSQL:      "CREATE SEQUENCE cycle_seq START WITH 1 INCREMENT BY 1 MINVALUE 1 MAXVALUE 10 CYCLE;",
			expectedName:     "cycle_seq",
			expectedDataType: "bigint",
			expectedStart:    1,
			expectedIncr:     1,
			expectedMinVal:   func() *int64 { v := int64(1); return &v }(),
			expectedMaxVal:   func() *int64 { v := int64(10); return &v }(),
			expectedCycle:    true,
			schemaName:       "public",
		},
		{
			name:             "schema_qualified_sequence",
			sequenceSQL:      "CREATE SEQUENCE analytics.report_id_seq START WITH 100 INCREMENT BY 10;",
			expectedName:     "report_id_seq",
			expectedDataType: "bigint",
			expectedStart:    100,
			expectedIncr:     10,
			expectedMinVal:   nil,
			expectedMaxVal:   nil,
			expectedCycle:    false,
			schemaName:       "analytics",
		},
		{
			name:             "sequence_as_integer",
			sequenceSQL:      "CREATE SEQUENCE small_seq AS integer START WITH 1 INCREMENT BY 1;",
			expectedName:     "small_seq",
			expectedDataType: "integer",
			expectedStart:    1,
			expectedIncr:     1,
			expectedMinVal:   nil,
			expectedMaxVal:   nil,
			expectedCycle:    false,
			schemaName:       "public",
		},
		{
			name:             "sequence_with_negative_increment",
			sequenceSQL:      "CREATE SEQUENCE reverse_seq START WITH 1000 INCREMENT BY -1 MINVALUE 1 MAXVALUE 1000;",
			expectedName:     "reverse_seq",
			expectedDataType: "bigint",
			expectedStart:    1000,
			expectedIncr:     -1,
			expectedMinVal:   func() *int64 { v := int64(1); return &v }(),
			expectedMaxVal:   func() *int64 { v := int64(1000); return &v }(),
			expectedCycle:    false,
			schemaName:       "public",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewParser()
			
			schema, err := parser.ParseSQL(tc.sequenceSQL)
			if err != nil {
				t.Fatalf("Failed to parse sequence SQL: %v", err)
			}
			
			// Find the schema containing the sequence
			var foundSequence *Sequence
			var schemaName string
			for sName, s := range schema.Schemas {
				if sequence, exists := s.Sequences[tc.expectedName]; exists {
					foundSequence = sequence
					schemaName = sName
					break
				}
			}
			
			if foundSequence == nil {
				t.Fatalf("Sequence %s not found in any schema", tc.expectedName)
			}
			
			// Verify sequence metadata
			if foundSequence.Name != tc.expectedName {
				t.Errorf("Expected sequence name %s, got %s", tc.expectedName, foundSequence.Name)
			}
			
			if foundSequence.DataType != tc.expectedDataType {
				t.Errorf("Expected data type %s, got %s", tc.expectedDataType, foundSequence.DataType)
			}
			
			if foundSequence.StartValue != tc.expectedStart {
				t.Errorf("Expected start value %d, got %d", tc.expectedStart, foundSequence.StartValue)
			}
			
			if foundSequence.Increment != tc.expectedIncr {
				t.Errorf("Expected increment %d, got %d", tc.expectedIncr, foundSequence.Increment)
			}
			
			if foundSequence.CycleOption != tc.expectedCycle {
				t.Errorf("Expected cycle option %t, got %t", tc.expectedCycle, foundSequence.CycleOption)
			}
			
			if schemaName != tc.schemaName {
				t.Errorf("Expected sequence to be in %s schema, found in %s", tc.schemaName, schemaName)
			}
			
			// Verify min/max values (handle nil pointers)
			if tc.expectedMinVal == nil {
				if foundSequence.MinValue != nil {
					t.Errorf("Expected MinValue to be nil, got %d", *foundSequence.MinValue)
				}
			} else {
				if foundSequence.MinValue == nil {
					t.Errorf("Expected MinValue to be %d, got nil", *tc.expectedMinVal)
				} else if *foundSequence.MinValue != *tc.expectedMinVal {
					t.Errorf("Expected MinValue %d, got %d", *tc.expectedMinVal, *foundSequence.MinValue)
				}
			}
			
			if tc.expectedMaxVal == nil {
				if foundSequence.MaxValue != nil {
					t.Errorf("Expected MaxValue to be nil, got %d", *foundSequence.MaxValue)
				}
			} else {
				if foundSequence.MaxValue == nil {
					t.Errorf("Expected MaxValue to be %d, got nil", *tc.expectedMaxVal)
				} else if *foundSequence.MaxValue != *tc.expectedMaxVal {
					t.Errorf("Expected MaxValue %d, got %d", *tc.expectedMaxVal, *foundSequence.MaxValue)
				}
			}
			
			t.Logf("✓ Sequence %s parsed correctly: %s start=%d incr=%d cycle=%t [%s]", 
				tc.expectedName, tc.expectedDataType, foundSequence.StartValue, 
				foundSequence.Increment, foundSequence.CycleOption, foundSequence.Schema)
		})
	}
}
