package diff

import (
	"strings"
	"testing"

	"github.com/pgschema/pgschema/internal/ir"
)

func TestGenerateIndexSQL_CamelCaseColumns(t *testing.T) {
	tests := []struct {
		name     string
		index    *ir.Index
		expected string
	}{
		{
			name: "single camelCase column",
			index: &ir.Index{
				Name:   "idx_invite_assignedTo",
				Schema: "public",
				Table:  "invite",
				Columns: []*ir.IndexColumn{
					{Name: "assignedTo"},
				},
			},
			expected: `CREATE INDEX IF NOT EXISTS "idx_invite_assignedTo" ON invite ("assignedTo");`,
		},
		{
			name: "multiple camelCase columns",
			index: &ir.Index{
				Name:   "idx_invite_composite",
				Schema: "public",
				Table:  "invite",
				Columns: []*ir.IndexColumn{
					{Name: "createdAt"},
					{Name: "invitedBy"},
				},
			},
			expected: `CREATE INDEX IF NOT EXISTS idx_invite_composite ON invite ("createdAt", "invitedBy");`,
		},
		{
			name: "mixed case and lowercase columns",
			index: &ir.Index{
				Name:   "idx_mixed",
				Schema: "public",
				Table:  "users",
				Columns: []*ir.IndexColumn{
					{Name: "firstName"},
					{Name: "email"},
					{Name: "lastName"},
				},
			},
			expected: `CREATE INDEX IF NOT EXISTS idx_mixed ON users ("firstName", email, "lastName");`,
		},
		{
			name: "unique index with camelCase",
			index: &ir.Index{
				Name:   "uk_user_email",
				Schema: "public",
				Table:  "user",
				Type:   ir.IndexTypeUnique,
				Columns: []*ir.IndexColumn{
					{Name: "emailAddress"},
				},
			},
			expected: `CREATE UNIQUE INDEX IF NOT EXISTS uk_user_email ON "user" ("emailAddress");`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateIndexSQL(tt.index, "public", false)
			if result != tt.expected {
				t.Errorf("generateIndexSQL() = %q; want %q", result, tt.expected)
			}
		})
	}
}

func TestGenerateIndexSQL_Concurrent_CamelCase(t *testing.T) {
	index := &ir.Index{
		Name:   "idx_invite_assignedTo",
		Schema: "public",
		Table:  "invite",
		Columns: []*ir.IndexColumn{
			{Name: "assignedTo"},
		},
	}

	result := generateIndexSQL(index, "public", true)
	expected := `CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invite_assignedTo" ON invite ("assignedTo");`
	
	if result != expected {
		t.Errorf("generateIndexSQL(concurrent=true) = %q; want %q", result, expected)
	}
}

func TestGenerateIndexSQL_ReservedWords(t *testing.T) {
	index := &ir.Index{
		Name:   "idx_user_order",
		Schema: "public",
		Table:  "user",  // reserved word
		Columns: []*ir.IndexColumn{
			{Name: "order"},  // reserved word
			{Name: "userId"},
		},
	}

	result := generateIndexSQL(index, "public", false)
	
	// Both "user" and "order" should be quoted as reserved words
	if !strings.Contains(result, `"user"`) {
		t.Errorf("Table name 'user' should be quoted")
	}
	if !strings.Contains(result, `"order"`) {
		t.Errorf("Column name 'order' should be quoted")
	}
}

func TestGenerateIndexSQL_FunctionalIndex(t *testing.T) {
	tests := []struct {
		name     string
		index    *ir.Index
		expected string
	}{
		{
			name: "functional index with lower",
			index: &ir.Index{
				Name:   "idx_users_lower_email",
				Schema: "public",
				Table:  "users",
				Columns: []*ir.IndexColumn{
					{Name: "lower(email)"},
				},
			},
			expected: `CREATE INDEX IF NOT EXISTS idx_users_lower_email ON users (lower(email));`,
		},
		{
			name: "functional index with upper on camelCase column",
			index: &ir.Index{
				Name:   "idx_users_upper_firstname",
				Schema: "public",
				Table:  "users",
				Columns: []*ir.IndexColumn{
					{Name: "upper(firstName)"},  // firstName inside function should not be quoted
				},
			},
			expected: `CREATE INDEX IF NOT EXISTS idx_users_upper_firstname ON users (upper(firstName));`,
		},
		{
			name: "mixed functional and regular columns",
			index: &ir.Index{
				Name:   "idx_mixed_func",
				Schema: "public",
				Table:  "users",
				Columns: []*ir.IndexColumn{
					{Name: "lower(email)"},
					{Name: "lastName"},  // regular camelCase should be quoted
					{Name: "age"},       // regular lowercase should not be quoted
				},
			},
			expected: `CREATE INDEX IF NOT EXISTS idx_mixed_func ON users (lower(email), "lastName", age);`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateIndexSQL(tt.index, "public", false)
			if result != tt.expected {
				t.Errorf("generateIndexSQL() = %q; want %q", result, tt.expected)
			}
		})
	}
}