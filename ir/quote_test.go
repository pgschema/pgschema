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

func TestQuoteIdentifierWithForce(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		forceQuote bool
		expected   string
	}{
		{"simple without force", "users", false, "users"},
		{"simple with force", "users", true, `"users"`},
		{"reserved word without force", "user", false, `"user"`},
		{"reserved word with force", "user", true, `"user"`},
		{"camelCase without force", "firstName", false, `"firstName"`},
		{"camelCase with force", "firstName", true, `"firstName"`},
		{"empty string without force", "", false, ""},
		{"empty string with force", "", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := QuoteIdentifierWithForce(tt.identifier, tt.forceQuote)
			if result != tt.expected {
				t.Errorf("QuoteIdentifierWithForce(%q, %v) = %q; want %q", tt.identifier, tt.forceQuote, result, tt.expected)
			}
		})
	}
}

func TestQualifyEntityNameWithQuotesAndForce(t *testing.T) {
	tests := []struct {
		name         string
		entitySchema string
		entityName   string
		targetSchema string
		forceQuote   bool
		expected     string
	}{
		{"same schema without force", "public", "users", "public", false, "users"},
		{"same schema with force", "public", "users", "public", true, `"users"`},
		{"different schema without force", "tenant", "users", "public", false, "tenant.users"},
		{"different schema with force", "tenant", "users", "public", true, `"tenant"."users"`},
		{"reserved word schema with force", "user", "table", "public", true, `"user"."table"`},
		{"mixed case schema with force", "MyApp", "Orders", "public", true, `"MyApp"."Orders"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := QualifyEntityNameWithQuotesAndForce(tt.entitySchema, tt.entityName, tt.targetSchema, tt.forceQuote)
			if result != tt.expected {
				t.Errorf("QualifyEntityNameWithQuotesAndForce(%q, %q, %q, %v) = %q; want %q", tt.entitySchema, tt.entityName, tt.targetSchema, tt.forceQuote, result, tt.expected)
			}
		})
	}
}
