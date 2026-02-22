package util

import (
	"strings"
	"testing"
)

func TestDetectSchemaFromReader(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "public schema",
			content: `--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.7
-- Dumped by pgschema version 1.7.1
-- Dumped from schema: public

`,
			expected: "public",
		},
		{
			name: "non-public schema",
			content: `--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.7
-- Dumped by pgschema version 1.7.1
-- Dumped from schema: vehicle

`,
			expected: "vehicle",
		},
		{
			name: "quoted schema name",
			content: `--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.7
-- Dumped by pgschema version 1.7.1
-- Dumped from schema: my_schema

`,
			expected: "my_schema",
		},
		{
			name: "no schema header (old dump format)",
			content: `--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.7
-- Dumped by pgschema version 1.7.1


CREATE TABLE IF NOT EXISTS users (
    id integer NOT NULL
);
`,
			expected: "",
		},
		{
			name:     "empty file",
			content:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := detectSchemaFromReader(strings.NewReader(tt.content))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}
