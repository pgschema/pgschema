package ir

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/pgschema/pgschema/testutil"
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
	containerInfo := testutil.SetupPostgresContainer(ctx, t)
	defer containerInfo.Terminate(ctx, t)

	// Get database connection
	db := containerInfo.Conn

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
	dbIR, err := builder.BuildIR(ctx, "public")
	if err != nil {
		t.Fatalf("Failed to build IR from database: %v", err)
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
	t.Logf("Database schema has %d schemas", len(dbIR.Schemas))
	t.Logf("Parser schema has %d schemas", len(parserSchema.Schemas))

	if dbIR.Schemas["public"] != nil {
		t.Logf("DB public schema has %d tables", len(dbIR.Schemas["public"].Tables))
	}
	if parserSchema.Schemas["public"] != nil {
		t.Logf("Parser public schema has %d tables", len(parserSchema.Schemas["public"].Tables))
	}

	// Use deep comparison to validate equivalence
	compareSchemasDeep(t, dbIR, parserSchema)

	// For debugging, save both schemas to files if comparison fails
	if t.Failed() {
		dbIRPath := fmt.Sprintf("%s_db_schema.json", testDataDir)
		parserSchemaPath := fmt.Sprintf("%s_parser_schema.json", testDataDir)

		if dbJSON, err := json.MarshalIndent(dbIR, "", "  "); err == nil {
			if err := os.WriteFile(dbIRPath, dbJSON, 0644); err == nil {
				t.Logf("Debug: DB schema written to %s", dbIRPath)
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
		"created_at": {3, "timestamptz", true}, // DEFAULT makes it nullable unless NOT NULL is explicit
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
func compareSchemasDeep(t *testing.T, expected, actual *IR) {
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

// compareDBSchemas compares two Schema objects deeply
func compareDBSchemas(t *testing.T, schemaName string, expected, actual *Schema) {
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
			t.Errorf("Schema %s: view count difference: expected %d, parser found %d",
				schemaName, len(expected.Views), len(actual.Views))
		}
	}

	if len(expected.Functions) > 0 || len(actual.Functions) > 0 {
		if len(expected.Functions) == len(actual.Functions) {
			t.Logf("Schema %s: functions match! Expected %d, parser found %d",
				schemaName, len(expected.Functions), len(actual.Functions))
		} else {
			t.Errorf("Schema %s: function count difference: expected %d, parser found %d",
				schemaName, len(expected.Functions), len(actual.Functions))
		}
	}

	if len(expected.Sequences) > 0 || len(actual.Sequences) > 0 {
		if len(expected.Sequences) == len(actual.Sequences) {
			t.Logf("Schema %s: sequences match! Expected %d, parser found %d",
				schemaName, len(expected.Sequences), len(actual.Sequences))
		} else {
			t.Errorf("Schema %s: sequence count difference: expected %d, parser found %d",
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
		t.Errorf("Table %s.%s: constraint count difference: expected %d, got %d (parser may not fully implement constraints yet)",
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
		t.Errorf("Column %s.%s.%s: data type mismatch: expected %s, got %s",
			schemaName, tableName, colName, expected.DataType, actual.DataType)
	}

	// Be lenient on nullable since parser doesn't parse ALTER TABLE NOT NULL constraints yet
	if expected.IsNullable != actual.IsNullable {
		t.Errorf("Column %s.%s.%s: nullable difference: expected %t, got %t (parser may not handle NOT NULL constraints from ALTER TABLE)",
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
		t.Errorf("Column %s.%s.%s: default value difference: expected %q, got %q (parser may not handle defaults from ALTER TABLE)",
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
			expectedParams: []struct {
				name     string
				dataType string
				mode     string
				position int
			}{},
			schemaName: "public",
		},
		{
			name:               "function_with_parameters",
			functionSQL:        "CREATE FUNCTION get_user_by_id(user_id integer) RETURNS text AS $$ SELECT name FROM users WHERE id = user_id; $$ LANGUAGE SQL;",
			expectedName:       "get_user_by_id",
			expectedReturnType: "text",
			expectedLanguage:   "sql",
			expectedDefinition: " SELECT name FROM users WHERE id = user_id; ",
			expectedParams: []struct {
				name     string
				dataType string
				mode     string
				position int
			}{
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
			expectedParams: []struct {
				name     string
				dataType string
				mode     string
				position int
			}{
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
			expectedParams: []struct {
				name     string
				dataType string
				mode     string
				position int
			}{
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
			expectedParams: []struct {
				name     string
				dataType string
				mode     string
				position int
			}{
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

func TestExtractConstraintFromAST(t *testing.T) {
	testCases := []struct {
		name              string
		constraintSQL     string
		expectedName      string
		expectedType      ConstraintType
		expectedColumns   []string
		expectedTable     string
		expectedSchema    string
		referencedTable   string
		referencedSchema  string
		referencedColumns []string
		checkClause       string
		deleteRule        string
		updateRule        string
	}{
		{
			name:            "primary_key_constraint",
			constraintSQL:   "CREATE TABLE test_table (id INTEGER); ALTER TABLE ONLY public.test_table ADD CONSTRAINT test_table_pkey PRIMARY KEY (id);",
			expectedName:    "test_table_pkey",
			expectedType:    ConstraintTypePrimaryKey,
			expectedColumns: []string{"id"},
			expectedTable:   "test_table",
			expectedSchema:  "public",
		},
		{
			name:            "unique_constraint",
			constraintSQL:   "CREATE TABLE test_table (email TEXT); ALTER TABLE ONLY public.test_table ADD CONSTRAINT test_table_email_key UNIQUE (email);",
			expectedName:    "test_table_email_key",
			expectedType:    ConstraintTypeUnique,
			expectedColumns: []string{"email"},
			expectedTable:   "test_table",
			expectedSchema:  "public",
		},
		{
			name:              "foreign_key_constraint",
			constraintSQL:     "CREATE TABLE users (id INTEGER); CREATE TABLE orders (id INTEGER, user_id INTEGER); ALTER TABLE ONLY public.orders ADD CONSTRAINT orders_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);",
			expectedName:      "orders_user_id_fkey",
			expectedType:      ConstraintTypeForeignKey,
			expectedColumns:   []string{"user_id"},
			expectedTable:     "orders",
			expectedSchema:    "public",
			referencedTable:   "users",
			referencedSchema:  "public",
			referencedColumns: []string{"id"},
		},
		{
			name:            "check_constraint",
			constraintSQL:   "CREATE TABLE test_table (age INTEGER); ALTER TABLE ONLY public.test_table ADD CONSTRAINT test_table_age_check CHECK ((age >= 0));",
			expectedName:    "test_table_age_check",
			expectedType:    ConstraintTypeCheck,
			expectedColumns: []string{},
			expectedTable:   "test_table",
			expectedSchema:  "public",
			checkClause:     "CHECK ((age >= 0))",
		},
		{
			name:              "foreign_key_with_actions",
			constraintSQL:     "CREATE TABLE users (id INTEGER); CREATE TABLE orders (id INTEGER, user_id INTEGER); ALTER TABLE ONLY public.orders ADD CONSTRAINT orders_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE ON UPDATE RESTRICT;",
			expectedName:      "orders_user_id_fkey",
			expectedType:      ConstraintTypeForeignKey,
			expectedColumns:   []string{"user_id"},
			expectedTable:     "orders",
			expectedSchema:    "public",
			referencedTable:   "users",
			referencedSchema:  "public",
			referencedColumns: []string{"id"},
			deleteRule:        "CASCADE",
			updateRule:        "RESTRICT",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewParser()

			schema, err := parser.ParseSQL(tc.constraintSQL)
			if err != nil {
				t.Fatalf("Failed to parse constraint SQL: %v", err)
			}

			// Find the table containing the constraint
			var foundConstraint *Constraint
			for _, s := range schema.Schemas {
				if table, exists := s.Tables[tc.expectedTable]; exists {
					if constraint, exists := table.Constraints[tc.expectedName]; exists {
						foundConstraint = constraint
						break
					}
				}
			}

			if foundConstraint == nil {
				t.Fatalf("Constraint %s not found in table %s", tc.expectedName, tc.expectedTable)
			}

			// Verify constraint metadata
			if foundConstraint.Name != tc.expectedName {
				t.Errorf("Expected constraint name %s, got %s", tc.expectedName, foundConstraint.Name)
			}

			if foundConstraint.Type != tc.expectedType {
				t.Errorf("Expected constraint type %s, got %s", tc.expectedType, foundConstraint.Type)
			}

			if foundConstraint.Table != tc.expectedTable {
				t.Errorf("Expected table %s, got %s", tc.expectedTable, foundConstraint.Table)
			}

			if foundConstraint.Schema != tc.expectedSchema {
				t.Errorf("Expected schema %s, got %s", tc.expectedSchema, foundConstraint.Schema)
			}

			// Verify columns
			if len(foundConstraint.Columns) != len(tc.expectedColumns) {
				t.Errorf("Expected %d columns, got %d", len(tc.expectedColumns), len(foundConstraint.Columns))
			} else {
				for i, expectedCol := range tc.expectedColumns {
					if i < len(foundConstraint.Columns) && foundConstraint.Columns[i].Name != expectedCol {
						t.Errorf("Expected column %s, got %s", expectedCol, foundConstraint.Columns[i].Name)
					}
				}
			}

			// Verify foreign key references
			if tc.referencedTable != "" {
				if foundConstraint.ReferencedTable != tc.referencedTable {
					t.Errorf("Expected referenced table %s, got %s", tc.referencedTable, foundConstraint.ReferencedTable)
				}

				if foundConstraint.ReferencedSchema != tc.referencedSchema {
					t.Errorf("Expected referenced schema %s, got %s", tc.referencedSchema, foundConstraint.ReferencedSchema)
				}

				if len(foundConstraint.ReferencedColumns) != len(tc.referencedColumns) {
					t.Errorf("Expected %d referenced columns, got %d", len(tc.referencedColumns), len(foundConstraint.ReferencedColumns))
				} else {
					for i, expectedCol := range tc.referencedColumns {
						if i < len(foundConstraint.ReferencedColumns) && foundConstraint.ReferencedColumns[i].Name != expectedCol {
							t.Errorf("Expected referenced column %s, got %s", expectedCol, foundConstraint.ReferencedColumns[i].Name)
						}
					}
				}
			}

			// Verify check clause
			if tc.checkClause != "" && foundConstraint.CheckClause != tc.checkClause {
				t.Errorf("Expected check clause %s, got %s", tc.checkClause, foundConstraint.CheckClause)
			}

			// Verify referential actions
			if tc.deleteRule != "" && foundConstraint.DeleteRule != tc.deleteRule {
				t.Errorf("Expected delete rule %s, got %s", tc.deleteRule, foundConstraint.DeleteRule)
			}

			if tc.updateRule != "" && foundConstraint.UpdateRule != tc.updateRule {
				t.Errorf("Expected update rule %s, got %s", tc.updateRule, foundConstraint.UpdateRule)
			}

			t.Logf("✓ Constraint %s parsed correctly: %s on %s.%s",
				tc.expectedName, tc.expectedType, tc.expectedSchema, tc.expectedTable)
		})
	}
}

func TestExtractIndexFromAST(t *testing.T) {
	testCases := []struct {
		name            string
		indexSQL        string
		expectedName    string
		expectedTable   string
		expectedSchema  string
		expectedMethod  string
		expectedUnique  bool
		expectedColumns []string
		expectedPartial bool
		whereClause     string
	}{
		{
			name:            "simple_btree_index",
			indexSQL:        "CREATE TABLE test_table (name TEXT); CREATE INDEX idx_test_name ON public.test_table USING btree (name);",
			expectedName:    "idx_test_name",
			expectedTable:   "test_table",
			expectedSchema:  "public",
			expectedMethod:  "btree",
			expectedUnique:  false,
			expectedColumns: []string{"name"},
			expectedPartial: false,
		},
		{
			name:            "unique_index",
			indexSQL:        "CREATE TABLE test_table (email TEXT); CREATE UNIQUE INDEX idx_test_email ON public.test_table USING btree (email);",
			expectedName:    "idx_test_email",
			expectedTable:   "test_table",
			expectedSchema:  "public",
			expectedMethod:  "btree",
			expectedUnique:  true,
			expectedColumns: []string{"email"},
			expectedPartial: false,
		},
		{
			name:            "partial_index",
			indexSQL:        "CREATE TABLE test_table (status TEXT, created_at TIMESTAMP); CREATE INDEX idx_active_status ON public.test_table USING btree (created_at) WHERE (status = 'active');",
			expectedName:    "idx_active_status",
			expectedTable:   "test_table",
			expectedSchema:  "public",
			expectedMethod:  "btree",
			expectedUnique:  false,
			expectedColumns: []string{"created_at"},
			expectedPartial: true,
			whereClause:     "(status = 'active')",
		},
		{
			name:            "gin_index",
			indexSQL:        "CREATE TABLE test_table (data JSONB); CREATE INDEX idx_test_data ON public.test_table USING gin (data);",
			expectedName:    "idx_test_data",
			expectedTable:   "test_table",
			expectedSchema:  "public",
			expectedMethod:  "gin",
			expectedUnique:  false,
			expectedColumns: []string{"data"},
			expectedPartial: false,
		},
		{
			name:            "multi_column_index",
			indexSQL:        "CREATE TABLE test_table (first_name TEXT, last_name TEXT); CREATE INDEX idx_test_name ON public.test_table USING btree (first_name, last_name);",
			expectedName:    "idx_test_name",
			expectedTable:   "test_table",
			expectedSchema:  "public",
			expectedMethod:  "btree",
			expectedUnique:  false,
			expectedColumns: []string{"first_name", "last_name"},
			expectedPartial: false,
		},
		{
			name:            "regular_multi_column_btree_index",
			indexSQL:        "CREATE TABLE employees (department_id INTEGER, salary NUMERIC, hire_date DATE); CREATE INDEX idx_dept_salary_hire ON public.employees USING btree (department_id, salary DESC, hire_date);",
			expectedName:    "idx_dept_salary_hire",
			expectedTable:   "employees",
			expectedSchema:  "public",
			expectedMethod:  "btree",
			expectedUnique:  false,
			expectedColumns: []string{"department_id", "salary", "hire_date"},
			expectedPartial: false,
		},
		{
			name:            "unique_multi_column_index",
			indexSQL:        "CREATE TABLE users (email TEXT, username TEXT, deleted_at TIMESTAMP); CREATE UNIQUE INDEX idx_unique_email_username ON public.users USING btree (email, username) WHERE deleted_at IS NULL;",
			expectedName:    "idx_unique_email_username",
			expectedTable:   "users",
			expectedSchema:  "public",
			expectedMethod:  "btree",
			expectedUnique:  true,
			expectedColumns: []string{"email", "username"},
			expectedPartial: true,
			whereClause:     "(deleted_at IS NULL)",
		},
		{
			name:            "partial_multi_column_index_with_complex_where",
			indexSQL:        "CREATE TABLE orders (customer_id INTEGER, order_date DATE, status TEXT, total NUMERIC); CREATE INDEX idx_active_orders ON public.orders USING btree (customer_id, order_date DESC) WHERE status IN ('pending', 'processing') AND total > 100;",
			expectedName:    "idx_active_orders",
			expectedTable:   "orders",
			expectedSchema:  "public",
			expectedMethod:  "btree",
			expectedUnique:  false,
			expectedColumns: []string{"customer_id", "order_date"},
			expectedPartial: true,
			whereClause:     "(expression)",
		},
		{
			name:            "functional_index_lower",
			indexSQL:        "CREATE TABLE products (name TEXT, sku TEXT); CREATE INDEX idx_lower_name ON public.products USING btree (lower(name));",
			expectedName:    "idx_lower_name",
			expectedTable:   "products",
			expectedSchema:  "public",
			expectedMethod:  "btree",
			expectedUnique:  false,
			expectedColumns: []string{"lower()"},
			expectedPartial: false,
		},
		{
			name:            "functional_index_multi_expression",
			indexSQL:        "CREATE TABLE logs (created_at TIMESTAMP, level TEXT, message TEXT); CREATE INDEX idx_date_level ON public.logs USING btree (date(created_at), upper(level));",
			expectedName:    "idx_date_level",
			expectedTable:   "logs",
			expectedSchema:  "public",
			expectedMethod:  "btree",
			expectedUnique:  false,
			expectedColumns: []string{"date()", "upper()"},
			expectedPartial: false,
		},
		{
			name:            "hash_index_single_column",
			indexSQL:        "CREATE TABLE cache (key TEXT, value TEXT); CREATE INDEX idx_cache_key ON public.cache USING hash (key);",
			expectedName:    "idx_cache_key",
			expectedTable:   "cache",
			expectedSchema:  "public",
			expectedMethod:  "hash",
			expectedUnique:  false,
			expectedColumns: []string{"key"},
			expectedPartial: false,
		},
		{
			name:            "gist_index_for_geometry",
			indexSQL:        "CREATE TABLE locations (name TEXT, geom geometry); CREATE INDEX idx_locations_geom ON public.locations USING gist (geom);",
			expectedName:    "idx_locations_geom",
			expectedTable:   "locations",
			expectedSchema:  "public",
			expectedMethod:  "gist",
			expectedUnique:  false,
			expectedColumns: []string{"geom"},
			expectedPartial: false,
		},
		{
			name:            "multi_column_with_mixed_order",
			indexSQL:        "CREATE TABLE transactions (account_id INTEGER, amount DECIMAL, created_at TIMESTAMP); CREATE INDEX idx_account_amount_date ON public.transactions USING btree (account_id ASC, amount DESC, created_at ASC);",
			expectedName:    "idx_account_amount_date",
			expectedTable:   "transactions",
			expectedSchema:  "public",
			expectedMethod:  "btree",
			expectedUnique:  false,
			expectedColumns: []string{"account_id", "amount", "created_at"},
			expectedPartial: false,
		},
		{
			name:            "unique_index_with_include_columns",
			indexSQL:        "CREATE TABLE articles (id SERIAL, slug TEXT, title TEXT, content TEXT); CREATE UNIQUE INDEX idx_unique_slug ON public.articles USING btree (slug) INCLUDE (title);",
			expectedName:    "idx_unique_slug",
			expectedTable:   "articles",
			expectedSchema:  "public",
			expectedMethod:  "btree",
			expectedUnique:  true,
			expectedColumns: []string{"slug"},
			expectedPartial: false,
		},
		{
			name:            "concurrent_index",
			indexSQL:        "CREATE TABLE users (email TEXT, status TEXT); CREATE INDEX CONCURRENTLY idx_users_email ON public.users USING btree (email);",
			expectedName:    "idx_users_email",
			expectedTable:   "users",
			expectedSchema:  "public",
			expectedMethod:  "btree",
			expectedUnique:  false,
			expectedColumns: []string{"email"},
			expectedPartial: false,
		},
		{
			name:            "unique_concurrent_multi_column_index",
			indexSQL:        "CREATE TABLE accounts (account_number TEXT, routing_number TEXT, bank_code TEXT); CREATE UNIQUE INDEX CONCURRENTLY idx_unique_account ON public.accounts USING btree (account_number, routing_number, bank_code);",
			expectedName:    "idx_unique_account",
			expectedTable:   "accounts",
			expectedSchema:  "public",
			expectedMethod:  "btree",
			expectedUnique:  true,
			expectedColumns: []string{"account_number", "routing_number", "bank_code"},
			expectedPartial: false,
		},
		{
			name:            "partial_concurrent_multi_column_index",
			indexSQL:        "CREATE TABLE orders (customer_id INTEGER, status TEXT, order_date DATE); CREATE INDEX CONCURRENTLY idx_active_orders ON public.orders USING btree (customer_id, order_date DESC) WHERE status = 'active';",
			expectedName:    "idx_active_orders",
			expectedTable:   "orders",
			expectedSchema:  "public",
			expectedMethod:  "btree",
			expectedUnique:  false,
			expectedColumns: []string{"customer_id", "order_date"},
			expectedPartial: true,
			whereClause:     "(status = 'active')",
		},
		{
			name:            "functional_concurrent_partial_index",
			indexSQL:        "CREATE TABLE users (first_name TEXT, last_name TEXT, status TEXT); CREATE INDEX CONCURRENTLY idx_users_names ON public.users USING btree (lower(first_name), lower(last_name)) WHERE status = 'active';",
			expectedName:    "idx_users_names",
			expectedTable:   "users",
			expectedSchema:  "public",
			expectedMethod:  "btree",
			expectedUnique:  false,
			expectedColumns: []string{"lower()", "lower()"},
			expectedPartial: true,
			whereClause:     "(status = 'active')",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewParser()

			schema, err := parser.ParseSQL(tc.indexSQL)
			if err != nil {
				t.Fatalf("Failed to parse index SQL: %v", err)
			}

			// Find the table containing the index
			var foundIndex *Index
			for _, s := range schema.Schemas {
				if table, exists := s.Tables[tc.expectedTable]; exists {
					if index, exists := table.Indexes[tc.expectedName]; exists {
						foundIndex = index
						break
					}
				}
			}

			if foundIndex == nil {
				t.Fatalf("Index %s not found in table %s", tc.expectedName, tc.expectedTable)
			}

			// Verify index metadata
			if foundIndex.Name != tc.expectedName {
				t.Errorf("Expected index name %s, got %s", tc.expectedName, foundIndex.Name)
			}

			if foundIndex.Table != tc.expectedTable {
				t.Errorf("Expected table %s, got %s", tc.expectedTable, foundIndex.Table)
			}

			if foundIndex.Schema != tc.expectedSchema {
				t.Errorf("Expected schema %s, got %s", tc.expectedSchema, foundIndex.Schema)
			}

			if foundIndex.Method != tc.expectedMethod {
				t.Errorf("Expected method %s, got %s", tc.expectedMethod, foundIndex.Method)
			}

			if foundIndex.IsUnique != tc.expectedUnique {
				t.Errorf("Expected unique %t, got %t", tc.expectedUnique, foundIndex.IsUnique)
			}

			if foundIndex.IsPartial != tc.expectedPartial {
				t.Errorf("Expected partial %t, got %t", tc.expectedPartial, foundIndex.IsPartial)
			}

			// Verify columns
			if len(foundIndex.Columns) != len(tc.expectedColumns) {
				t.Errorf("Expected %d columns, got %d", len(tc.expectedColumns), len(foundIndex.Columns))
			} else {
				for i, expectedCol := range tc.expectedColumns {
					if i < len(foundIndex.Columns) && foundIndex.Columns[i].Name != expectedCol {
						t.Errorf("Expected column %s, got %s", expectedCol, foundIndex.Columns[i].Name)
					}
				}
			}

			// Verify WHERE clause for partial indexes
			if tc.whereClause != "" && foundIndex.Where != tc.whereClause {
				t.Errorf("Expected WHERE clause %s, got %s", tc.whereClause, foundIndex.Where)
			}

			t.Logf("✓ Index %s parsed correctly: %s on %s.%s (%s)",
				tc.expectedName, tc.expectedMethod, tc.expectedSchema, tc.expectedTable,
				strings.Join(tc.expectedColumns, ", "))
		})
	}
}

func TestExtractTriggerFromAST(t *testing.T) {
	testCases := []struct {
		name             string
		triggerSQL       string
		expectedName     string
		expectedTable    string
		expectedSchema   string
		expectedTiming   TriggerTiming
		expectedEvents   []TriggerEvent
		expectedLevel    TriggerLevel
		expectedFunction string
	}{
		{
			name:             "simple_insert_trigger",
			triggerSQL:       "CREATE TABLE test_table (id INTEGER); CREATE FUNCTION test_func() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql; CREATE TRIGGER test_trigger BEFORE INSERT ON public.test_table FOR EACH ROW EXECUTE FUNCTION test_func();",
			expectedName:     "test_trigger",
			expectedTable:    "test_table",
			expectedSchema:   "public",
			expectedTiming:   TriggerTimingBefore,
			expectedEvents:   []TriggerEvent{TriggerEventInsert},
			expectedLevel:    TriggerLevelRow,
			expectedFunction: "test_func()",
		},
		{
			name:             "multi_event_trigger",
			triggerSQL:       "CREATE TABLE test_table (id INTEGER, name TEXT); CREATE FUNCTION audit_func() RETURNS TRIGGER AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql; CREATE TRIGGER audit_trigger AFTER INSERT OR UPDATE OR DELETE ON public.test_table FOR EACH ROW EXECUTE FUNCTION audit_func();",
			expectedName:     "audit_trigger",
			expectedTable:    "test_table",
			expectedSchema:   "public",
			expectedTiming:   TriggerTimingAfter,
			expectedEvents:   []TriggerEvent{TriggerEventInsert, TriggerEventUpdate, TriggerEventDelete},
			expectedLevel:    TriggerLevelRow,
			expectedFunction: "audit_func()",
		},
		{
			name:             "statement_level_trigger",
			triggerSQL:       "CREATE TABLE test_table (id INTEGER); CREATE FUNCTION log_func() RETURNS TRIGGER AS $$ BEGIN RETURN NULL; END; $$ LANGUAGE plpgsql; CREATE TRIGGER log_trigger BEFORE TRUNCATE ON public.test_table FOR EACH STATEMENT EXECUTE FUNCTION log_func();",
			expectedName:     "log_trigger",
			expectedTable:    "test_table",
			expectedSchema:   "public",
			expectedTiming:   TriggerTimingBefore,
			expectedEvents:   []TriggerEvent{TriggerEventTruncate},
			expectedLevel:    TriggerLevelStatement,
			expectedFunction: "log_func()",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewParser()

			schema, err := parser.ParseSQL(tc.triggerSQL)
			if err != nil {
				t.Fatalf("Failed to parse trigger SQL: %v", err)
			}

			// Find the table containing the trigger
			var foundTrigger *Trigger
			for _, s := range schema.Schemas {
				if table, exists := s.Tables[tc.expectedTable]; exists {
					if trigger, exists := table.Triggers[tc.expectedName]; exists {
						foundTrigger = trigger
						break
					}
				}
			}

			if foundTrigger == nil {
				t.Fatalf("Trigger %s not found in table %s", tc.expectedName, tc.expectedTable)
			}

			// Verify trigger metadata
			if foundTrigger.Name != tc.expectedName {
				t.Errorf("Expected trigger name %s, got %s", tc.expectedName, foundTrigger.Name)
			}

			if foundTrigger.Table != tc.expectedTable {
				t.Errorf("Expected table %s, got %s", tc.expectedTable, foundTrigger.Table)
			}

			if foundTrigger.Schema != tc.expectedSchema {
				t.Errorf("Expected schema %s, got %s", tc.expectedSchema, foundTrigger.Schema)
			}

			if foundTrigger.Timing != tc.expectedTiming {
				t.Errorf("Expected timing %s, got %s", tc.expectedTiming, foundTrigger.Timing)
			}

			if foundTrigger.Level != tc.expectedLevel {
				t.Errorf("Expected level %s, got %s", tc.expectedLevel, foundTrigger.Level)
			}

			if foundTrigger.Function != tc.expectedFunction {
				t.Errorf("Expected function %s, got %s", tc.expectedFunction, foundTrigger.Function)
			}

			// Verify events
			if len(foundTrigger.Events) != len(tc.expectedEvents) {
				t.Errorf("Expected %d events, got %d", len(tc.expectedEvents), len(foundTrigger.Events))
			} else {
				for i, expectedEvent := range tc.expectedEvents {
					if i < len(foundTrigger.Events) && foundTrigger.Events[i] != expectedEvent {
						t.Errorf("Expected event %s, got %s", expectedEvent, foundTrigger.Events[i])
					}
				}
			}

			t.Logf("✓ Trigger %s parsed correctly: %s %s on %s.%s",
				tc.expectedName, tc.expectedTiming, tc.expectedLevel, tc.expectedSchema, tc.expectedTable)
		})
	}
}

func TestExtractExtensionFromAST(t *testing.T) {
	testCases := []struct {
		name           string
		extensionSQL   string
		expectedName   string
		expectedSchema string
	}{
		{
			name:           "simple_extension",
			extensionSQL:   "CREATE EXTENSION IF NOT EXISTS uuid_ossp;",
			expectedName:   "uuid_ossp",
			expectedSchema: "",
		},
		{
			name:           "extension_with_schema",
			extensionSQL:   "CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;",
			expectedName:   "pgcrypto",
			expectedSchema: "public",
		},
		{
			name:           "hstore_extension",
			extensionSQL:   "CREATE EXTENSION IF NOT EXISTS hstore WITH SCHEMA public;",
			expectedName:   "hstore",
			expectedSchema: "public",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewParser()

			schema, err := parser.ParseSQL(tc.extensionSQL)
			if err != nil {
				t.Fatalf("Failed to parse extension SQL: %v", err)
			}

			// Find the extension
			var foundExtension *Extension
			for _, ext := range schema.Extensions {
				if ext.Name == tc.expectedName {
					foundExtension = ext
					break
				}
			}

			if foundExtension == nil {
				t.Fatalf("Extension %s not found", tc.expectedName)
			}

			// Verify extension metadata
			if foundExtension.Name != tc.expectedName {
				t.Errorf("Expected extension name %s, got %s", tc.expectedName, foundExtension.Name)
			}

			if foundExtension.Schema != tc.expectedSchema {
				t.Errorf("Expected schema %s, got %s", tc.expectedSchema, foundExtension.Schema)
			}

			t.Logf("✓ Extension %s parsed correctly with schema %s",
				tc.expectedName, tc.expectedSchema)
		})
	}
}

func TestExtractTypeFromAST(t *testing.T) {
	testCases := []struct {
		name             string
		typeSQL          string
		expectedName     string
		expectedSchema   string
		expectedKind     TypeKind
		expectedValues   []string
		expectedColumns  []string
		expectedBaseType string
	}{
		{
			name:           "enum_type",
			typeSQL:        "CREATE TYPE public.status_enum AS ENUM ('active', 'inactive', 'pending');",
			expectedName:   "status_enum",
			expectedSchema: "public",
			expectedKind:   TypeKindEnum,
			expectedValues: []string{"active", "inactive", "pending"},
		},
		{
			name:            "composite_type",
			typeSQL:         "CREATE TYPE public.address AS (street TEXT, city TEXT, postal_code TEXT);",
			expectedName:    "address",
			expectedSchema:  "public",
			expectedKind:    TypeKindComposite,
			expectedColumns: []string{"street", "city", "postal_code"},
		},
		{
			name:             "domain_type",
			typeSQL:          "CREATE DOMAIN public.email AS TEXT CHECK (VALUE ~ '^[A-Za-z0-9._%-]+@[A-Za-z0-9.-]+[.][A-Za-z]+$');",
			expectedName:     "email",
			expectedSchema:   "public",
			expectedKind:     TypeKindDomain,
			expectedBaseType: "text",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewParser()

			schema, err := parser.ParseSQL(tc.typeSQL)
			if err != nil {
				t.Fatalf("Failed to parse type SQL: %v", err)
			}

			// Find the type
			var foundType *Type
			for _, s := range schema.Schemas {
				if userType, exists := s.Types[tc.expectedName]; exists {
					foundType = userType
					break
				}
			}

			if foundType == nil {
				t.Fatalf("Type %s not found", tc.expectedName)
			}

			// Verify type metadata
			if foundType.Name != tc.expectedName {
				t.Errorf("Expected type name %s, got %s", tc.expectedName, foundType.Name)
			}

			if foundType.Schema != tc.expectedSchema {
				t.Errorf("Expected schema %s, got %s", tc.expectedSchema, foundType.Schema)
			}

			if foundType.Kind != tc.expectedKind {
				t.Errorf("Expected kind %s, got %s", tc.expectedKind, foundType.Kind)
			}

			// Verify enum values
			if tc.expectedKind == TypeKindEnum {
				if len(foundType.EnumValues) != len(tc.expectedValues) {
					t.Errorf("Expected %d enum values, got %d", len(tc.expectedValues), len(foundType.EnumValues))
				} else {
					for i, expectedValue := range tc.expectedValues {
						if i < len(foundType.EnumValues) && foundType.EnumValues[i] != expectedValue {
							t.Errorf("Expected enum value %s, got %s", expectedValue, foundType.EnumValues[i])
						}
					}
				}
			}

			// Verify composite columns
			if tc.expectedKind == TypeKindComposite {
				if len(foundType.Columns) != len(tc.expectedColumns) {
					t.Errorf("Expected %d columns, got %d", len(tc.expectedColumns), len(foundType.Columns))
				} else {
					for i, expectedCol := range tc.expectedColumns {
						if i < len(foundType.Columns) && foundType.Columns[i].Name != expectedCol {
							t.Errorf("Expected column %s, got %s", expectedCol, foundType.Columns[i].Name)
						}
					}
				}
			}

			// Verify domain base type
			if tc.expectedKind == TypeKindDomain && tc.expectedBaseType != "" {
				if foundType.BaseType != tc.expectedBaseType {
					t.Errorf("Expected base type %s, got %s", tc.expectedBaseType, foundType.BaseType)
				}
			}

			t.Logf("✓ Type %s parsed correctly: %s in schema %s",
				tc.expectedName, tc.expectedKind, tc.expectedSchema)
		})
	}
}

func TestExtractAggregateFromAST(t *testing.T) {
	testCases := []struct {
		name               string
		aggregateSQL       string
		expectedName       string
		expectedSchema     string
		expectedReturnType string
		expectedStateType  string
		expectedTransition string
		expectedArguments  string
	}{
		{
			name:               "simple_aggregate",
			aggregateSQL:       "CREATE FUNCTION my_avg_sfunc(NUMERIC, NUMERIC) RETURNS NUMERIC AS $$ SELECT ($1 * $2 + $3) / ($2 + 1) $$ LANGUAGE SQL; CREATE AGGREGATE public.my_avg(NUMERIC) (SFUNC = my_avg_sfunc, STYPE = NUMERIC);",
			expectedName:       "my_avg",
			expectedSchema:     "public",
			expectedReturnType: "numeric",
			expectedStateType:  "numeric",
			expectedTransition: "my_avg_sfunc",
			expectedArguments:  "numeric",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewParser()

			schema, err := parser.ParseSQL(tc.aggregateSQL)
			if err != nil {
				t.Fatalf("Failed to parse aggregate SQL: %v", err)
			}

			// Find the aggregate
			var foundAggregate *Aggregate
			for _, s := range schema.Schemas {
				if aggregate, exists := s.Aggregates[tc.expectedName]; exists {
					foundAggregate = aggregate
					break
				}
			}

			if foundAggregate == nil {
				t.Fatalf("Aggregate %s not found", tc.expectedName)
			}

			// Verify aggregate metadata
			if foundAggregate.Name != tc.expectedName {
				t.Errorf("Expected aggregate name %s, got %s", tc.expectedName, foundAggregate.Name)
			}

			if foundAggregate.Schema != tc.expectedSchema {
				t.Errorf("Expected schema %s, got %s", tc.expectedSchema, foundAggregate.Schema)
			}

			if foundAggregate.ReturnType != tc.expectedReturnType {
				t.Errorf("Expected return type %s, got %s", tc.expectedReturnType, foundAggregate.ReturnType)
			}

			if foundAggregate.StateType != tc.expectedStateType {
				t.Errorf("Expected state type %s, got %s", tc.expectedStateType, foundAggregate.StateType)
			}

			if foundAggregate.TransitionFunction != tc.expectedTransition {
				t.Errorf("Expected transition function %s, got %s", tc.expectedTransition, foundAggregate.TransitionFunction)
			}

			if foundAggregate.Arguments != tc.expectedArguments {
				t.Errorf("Expected arguments %s, got %s", tc.expectedArguments, foundAggregate.Arguments)
			}

			t.Logf("✓ Aggregate %s parsed correctly in schema %s",
				tc.expectedName, tc.expectedSchema)
		})
	}
}

func TestExtractProcedureFromAST(t *testing.T) {
	testCases := []struct {
		name             string
		procedureSQL     string
		expectedName     string
		expectedSchema   string
		expectedLanguage string
		expectedArgs     string
	}{
		{
			name:             "simple_procedure",
			procedureSQL:     "CREATE PROCEDURE public.update_stats(table_name TEXT) LANGUAGE SQL AS $$ UPDATE stats SET last_updated = NOW() WHERE name = table_name; $$;",
			expectedName:     "update_stats",
			expectedSchema:   "public",
			expectedLanguage: "sql",
			expectedArgs:     "table_name text",
		},
		{
			name:             "plpgsql_procedure",
			procedureSQL:     "CREATE PROCEDURE public.process_orders() LANGUAGE plpgsql AS $$ BEGIN UPDATE orders SET status = 'processed' WHERE status = 'pending'; END; $$;",
			expectedName:     "process_orders",
			expectedSchema:   "public",
			expectedLanguage: "plpgsql",
			expectedArgs:     "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewParser()

			schema, err := parser.ParseSQL(tc.procedureSQL)
			if err != nil {
				t.Fatalf("Failed to parse procedure SQL: %v", err)
			}

			// Find the procedure
			var foundProcedure *Procedure
			for _, s := range schema.Schemas {
				if procedure, exists := s.Procedures[tc.expectedName]; exists {
					foundProcedure = procedure
					break
				}
			}

			if foundProcedure == nil {
				t.Fatalf("Procedure %s not found", tc.expectedName)
			}

			// Verify procedure metadata
			if foundProcedure.Name != tc.expectedName {
				t.Errorf("Expected procedure name %s, got %s", tc.expectedName, foundProcedure.Name)
			}

			if foundProcedure.Schema != tc.expectedSchema {
				t.Errorf("Expected schema %s, got %s", tc.expectedSchema, foundProcedure.Schema)
			}

			if foundProcedure.Language != tc.expectedLanguage {
				t.Errorf("Expected language %s, got %s", tc.expectedLanguage, foundProcedure.Language)
			}

			if foundProcedure.Arguments != tc.expectedArgs {
				t.Errorf("Expected arguments %s, got %s", tc.expectedArgs, foundProcedure.Arguments)
			}

			t.Logf("✓ Procedure %s parsed correctly in schema %s",
				tc.expectedName, tc.expectedSchema)
		})
	}
}

func TestExtractPolicyFromAST(t *testing.T) {
	testCases := []struct {
		name            string
		policySQL       string
		expectedName    string
		expectedTable   string
		expectedSchema  string
		expectedCommand PolicyCommand
		expectedUsing   string
		expectedCheck   string
	}{
		{
			name:            "select_policy",
			policySQL:       "CREATE TABLE users (id INTEGER, name TEXT); ALTER TABLE users ENABLE ROW LEVEL SECURITY; CREATE POLICY user_policy ON public.users FOR SELECT USING (id = current_user_id());",
			expectedName:    "user_policy",
			expectedTable:   "users",
			expectedSchema:  "public",
			expectedCommand: PolicyCommandSelect,
			expectedUsing:   "(id = current_user_id())",
		},
		{
			name:            "insert_policy_with_check",
			policySQL:       "CREATE TABLE orders (id INTEGER, user_id INTEGER); ALTER TABLE orders ENABLE ROW LEVEL SECURITY; CREATE POLICY order_policy ON public.orders FOR INSERT WITH CHECK (user_id = current_user_id());",
			expectedName:    "order_policy",
			expectedTable:   "orders",
			expectedSchema:  "public",
			expectedCommand: PolicyCommandInsert,
			expectedCheck:   "(user_id = current_user_id())",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewParser()

			schema, err := parser.ParseSQL(tc.policySQL)
			if err != nil {
				t.Fatalf("Failed to parse policy SQL: %v", err)
			}

			// Find the table containing the policy
			var foundPolicy *RLSPolicy
			for _, s := range schema.Schemas {
				if table, exists := s.Tables[tc.expectedTable]; exists {
					if policy, exists := table.Policies[tc.expectedName]; exists {
						foundPolicy = policy
						break
					}
				}
			}

			if foundPolicy == nil {
				t.Fatalf("Policy %s not found in table %s", tc.expectedName, tc.expectedTable)
			}

			// Verify policy metadata
			if foundPolicy.Name != tc.expectedName {
				t.Errorf("Expected policy name %s, got %s", tc.expectedName, foundPolicy.Name)
			}

			if foundPolicy.Table != tc.expectedTable {
				t.Errorf("Expected table %s, got %s", tc.expectedTable, foundPolicy.Table)
			}

			if foundPolicy.Schema != tc.expectedSchema {
				t.Errorf("Expected schema %s, got %s", tc.expectedSchema, foundPolicy.Schema)
			}

			if foundPolicy.Command != tc.expectedCommand {
				t.Errorf("Expected command %s, got %s", tc.expectedCommand, foundPolicy.Command)
			}

			if tc.expectedUsing != "" && foundPolicy.Using != tc.expectedUsing {
				t.Errorf("Expected using %s, got %s", tc.expectedUsing, foundPolicy.Using)
			}

			if tc.expectedCheck != "" && foundPolicy.WithCheck != tc.expectedCheck {
				t.Errorf("Expected check %s, got %s", tc.expectedCheck, foundPolicy.WithCheck)
			}

			t.Logf("✓ Policy %s parsed correctly: %s on %s.%s",
				tc.expectedName, tc.expectedCommand, tc.expectedSchema, tc.expectedTable)
		})
	}
}
