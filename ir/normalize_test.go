package ir

import (
	"testing"
)

// TestNormalizeDefaultValue_TempSchemaFunctionQualifier tests that column defaults
// referencing functions in the public schema are properly normalized when the table
// is in a temporary schema (pgschema_tmp_*).
//
// This is a regression test for GitHub issue #283:
// When pgschema plans a migration using a temp schema for desired state, pg_get_expr()
// with search_path=pg_catalog returns "public.func_name()" for functions in the public
// schema. normalizeDefaultValue with the temp schema name can't strip "public." because
// it doesn't match the temp schema name. On the target database, normalizeDefaultValue
// with tableSchema="public" DOES strip it. This causes a spurious diff.
//
// The fix: re-run normalizeIR after normalizeSchemaNames replaces temp → target schema.
func TestNormalizeDefaultValue_TempSchemaFunctionQualifier(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		tableSchema string
		expected    string
	}{
		{
			name:        "public function stripped when table in public schema",
			value:       "public.uuid_generate_v1mc()",
			tableSchema: "public",
			expected:    "uuid_generate_v1mc()",
		},
		{
			name:        "public function NOT stripped when table in temp schema (the bug)",
			value:       "public.uuid_generate_v1mc()",
			tableSchema: "pgschema_tmp_20260101_120000_abcd1234",
			expected:    "public.uuid_generate_v1mc()",
		},
		{
			name:        "temp schema function stripped when table in same temp schema",
			value:       "pgschema_tmp_20260101_120000_abcd1234.my_func()",
			tableSchema: "pgschema_tmp_20260101_120000_abcd1234",
			expected:    "my_func()",
		},
		{
			name:        "cross-schema function preserved",
			value:       "other_schema.my_func()",
			tableSchema: "public",
			expected:    "other_schema.my_func()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeDefaultValue(tt.value, tt.tableSchema)
			if result != tt.expected {
				t.Errorf("normalizeDefaultValue(%q, %q) = %q, want %q", tt.value, tt.tableSchema, result, tt.expected)
			}
		})
	}
}

// TestNormalizeIR_RerunAfterSchemaRename verifies that re-running normalizeIR
// after renaming temp schema to target schema produces correct results.
// This is the key behavior that fixes issue #283.
func TestNormalizeIR_RerunAfterSchemaRename(t *testing.T) {
	tempSchema := "pgschema_tmp_20260101_120000_abcd1234"

	// Simulate desired state IR from temp schema inspection
	defaultVal := "public.uuid_generate_v1mc()"
	testIR := &IR{
		Schemas: map[string]*Schema{
			tempSchema: {
				Name: tempSchema,
				Tables: map[string]*Table{
					"items": {
						Name:   "items",
						Schema: tempSchema,
						Columns: []*Column{
							{
								Name:         "id",
								DataType:     "uuid",
								DefaultValue: &defaultVal,
							},
						},
					},
				},
			},
		},
	}

	// First normalization pass (happens in inspector)
	normalizeIR(testIR)

	// After first normalization, "public." is NOT stripped (table schema is temp)
	col := testIR.Schemas[tempSchema].Tables["items"].Columns[0]
	if *col.DefaultValue != "public.uuid_generate_v1mc()" {
		t.Fatalf("After first normalizeIR, expected default to remain 'public.uuid_generate_v1mc()', got %q", *col.DefaultValue)
	}

	// Simulate normalizeSchemaNames: rename temp schema to public
	schema := testIR.Schemas[tempSchema]
	delete(testIR.Schemas, tempSchema)
	schema.Name = "public"
	testIR.Schemas["public"] = schema
	for _, table := range schema.Tables {
		table.Schema = "public"
	}

	// Second normalization pass (the fix: re-run after schema rename)
	normalizeIR(testIR)

	// After second normalization, "public." should be stripped
	col = testIR.Schemas["public"].Tables["items"].Columns[0]
	if *col.DefaultValue != "uuid_generate_v1mc()" {
		t.Errorf("After re-running normalizeIR with correct schema, expected 'uuid_generate_v1mc()', got %q", *col.DefaultValue)
	}
}

func TestNormalizeCheckClause(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "varchar IN with ::text cast - user form (has extra parens around column)",
			input:    "CHECK ((status)::text = ANY (ARRAY['pending'::character varying, 'shipped'::character varying, 'delivered'::character varying]::text[]))",
			expected: "CHECK (status::text IN ('pending'::character varying, 'shipped'::character varying, 'delivered'::character varying))",
		},
		{
			name:     "varchar IN without explicit cast - user form (no extra parens)",
			input:    "CHECK (status::text = ANY (ARRAY['pending'::character varying, 'shipped'::character varying, 'delivered'::character varying]::text[]))",
			expected: "CHECK (status::text IN ('pending'::character varying, 'shipped'::character varying, 'delivered'::character varying))",
		},
		{
			name:     "varchar IN with double cast - applied form (pgschema-generated SQL stored by PostgreSQL)",
			input:    "CHECK (status::text = ANY (ARRAY['pending'::character varying::text, 'shipped'::character varying::text, 'delivered'::character varying::text]))",
			expected: "CHECK (status::text IN ('pending'::character varying, 'shipped'::character varying, 'delivered'::character varying))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeCheckClause(tt.input)
			t.Logf("Input:    %s", tt.input)
			t.Logf("Output:   %s", result)
			t.Logf("Expected: %s", tt.expected)
			if result != tt.expected {
				t.Errorf("normalizeCheckClause() = %v, want %v", result, tt.expected)
			}
		})
	}
}
