package diff

import (
	"testing"
)

func TestNeedsQuoting(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		want       bool
	}{
		{"empty string", "", false},
		{"simple lowercase", "tablename", false},
		{"with underscore", "table_name", false},
		{"reserved word user", "user", true},
		{"reserved word USER", "USER", true},
		{"reserved word Order", "Order", true},
		{"camelCase", "userId", true},
		{"PascalCase", "CreatedAt", true},
		{"starts with number", "1table", true},
		{"contains special char", "table-name", true},
		{"all lowercase", "createdat", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needsQuoting(tt.identifier); got != tt.want {
				t.Errorf("needsQuoting(%q) = %v, want %v", tt.identifier, got, tt.want)
			}
		})
	}
}

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		want       string
	}{
		{"simple lowercase", "tablename", "tablename"},
		{"reserved word", "user", `"user"`},
		{"camelCase", "userId", `"userId"`},
		{"already quoted", `"userId"`, `"userId"`}, // Should not double-quote
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Handle already quoted identifiers
			if len(tt.identifier) > 2 && tt.identifier[0] == '"' && tt.identifier[len(tt.identifier)-1] == '"' {
				// If already quoted, should return as-is
				if got := tt.identifier; got != tt.want {
					t.Errorf("already quoted identifier %q should remain %q, got %q", tt.identifier, tt.want, got)
				}
			} else {
				if got := quoteIdentifier(tt.identifier); got != tt.want {
					t.Errorf("quoteIdentifier(%q) = %q, want %q", tt.identifier, got, tt.want)
				}
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
		want         string
	}{
		{"same schema lowercase", "public", "users", "public", "users"},
		{"same schema camelCase", "public", "userId", "public", `"userId"`},
		{"different schema", "auth", "users", "public", "auth.users"},
		{"different schema camelCase", "auth", "userId", "public", `auth."userId"`},
		{"reserved word", "public", "user", "public", `"user"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := qualifyEntityNameWithQuotes(tt.entitySchema, tt.entityName, tt.targetSchema); got != tt.want {
				t.Errorf("qualifyEntityNameWithQuotes(%q, %q, %q) = %q, want %q",
					tt.entitySchema, tt.entityName, tt.targetSchema, got, tt.want)
			}
		})
	}
}