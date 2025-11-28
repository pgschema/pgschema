package ir

import (
	"testing"
)

func TestNeedsQuoting(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		expected   bool
	}{
		{"simple lowercase", "users", false},
		{"reserved word", "user", true},
		{"limit keyword", "limit", true},
		{"bigint type", "bigint", true},
		{"boolean type", "boolean", true},
		{"update command", "update", true},
		{"delete command", "delete", true},
		{"insert command", "insert", true},
		{"returning clause", "returning", true},
		{"camelCase", "firstName", true},
		{"UPPERCASE", "USERS", true},
		{"MixedCase", "MyApp", true},
		{"with underscore", "user_name", false},
		{"starts with underscore", "_private", false},
		{"starts with number", "1table", true},
		{"contains dash", "user-table", true},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NeedsQuoting(tt.identifier)
			if result != tt.expected {
				t.Errorf("NeedsQuoting(%q) = %v; want %v", tt.identifier, result, tt.expected)
			}
		})
	}
}

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		expected   string
	}{
		{"simple lowercase", "users", "users"},
		{"reserved word", "user", `"user"`},
		{"limit keyword", "limit", `"limit"`},
		{"bigint type", "bigint", `"bigint"`},
		{"boolean type", "boolean", `"boolean"`},
		{"update command", "update", `"update"`},
		{"delete command", "delete", `"delete"`},
		{"insert command", "insert", `"insert"`},
		{"returning clause", "returning", `"returning"`},
		{"camelCase", "firstName", `"firstName"`},
		{"UPPERCASE", "USERS", `"USERS"`},
		{"MixedCase", "MyApp", `"MyApp"`},
		{"with underscore", "user_name", "user_name"},
		{"starts with underscore", "_private", "_private"},
		{"starts with number", "1table", `"1table"`},
		{"contains dash", "user-table", `"user-table"`},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := QuoteIdentifier(tt.identifier)
			if result != tt.expected {
				t.Errorf("QuoteIdentifier(%q) = %q; want %q", tt.identifier, result, tt.expected)
			}
		})
	}
}

func TestQualifyEntityNameWithQuotes(t *testing.T) {
	tests := []struct {
		name         string
		entitySchema string
		entityName   string
		targetSchema string
		expected     string
	}{
		{"same schema lowercase", "public", "users", "public", "users"},
		{"same schema mixed case", "MyApp", "Orders", "MyApp", `"Orders"`},
		{"different schema lowercase", "tenant", "users", "public", "tenant.users"},
		{"different schema mixed case", "MyApp", "Orders", "public", `"MyApp"."Orders"`},
		{"reserved word schema", "user", "table", "public", `"user"."table"`},
		{"mixed case schema with lowercase target", "MyApp", "users", "public", `"MyApp".users`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := QualifyEntityNameWithQuotes(tt.entitySchema, tt.entityName, tt.targetSchema)
			if result != tt.expected {
				t.Errorf("QualifyEntityNameWithQuotes(%q, %q, %q) = %q; want %q", tt.entitySchema, tt.entityName, tt.targetSchema, result, tt.expected)
			}
		})
	}
}
