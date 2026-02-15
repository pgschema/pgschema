package ir

import (
	"testing"
)

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
