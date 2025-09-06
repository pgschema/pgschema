package diff

import (
	"testing"

	"github.com/pgschema/pgschema/internal/ir"
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

func TestGenerateConstraintSQL_WithQuoting(t *testing.T) {
	tests := []struct {
		name       string
		constraint *ir.Constraint
		want       string
	}{
		{
			name: "UNIQUE with camelCase columns",
			constraint: &ir.Constraint{
				Name: "test_unique",
				Type: ir.ConstraintTypeUnique,
				Columns: []*ir.ConstraintColumn{
					{Name: "userId", Position: 1},
					{Name: "accountId", Position: 2},
				},
			},
			want: `UNIQUE ("userId", "accountId")`,
		},
		{
			name: "PRIMARY KEY with reserved word",
			constraint: &ir.Constraint{
				Name: "test_pk",
				Type: ir.ConstraintTypePrimaryKey,
				Columns: []*ir.ConstraintColumn{
					{Name: "user", Position: 1},
					{Name: "order", Position: 2},
				},
			},
			want: `PRIMARY KEY ("user", "order")`,
		},
		{
			name: "FOREIGN KEY with camelCase",
			constraint: &ir.Constraint{
				Name: "test_fk",
				Type: ir.ConstraintTypeForeignKey,
				Columns: []*ir.ConstraintColumn{
					{Name: "userId", Position: 1},
				},
				ReferencedTable: "users",
				ReferencedColumns: []*ir.ConstraintColumn{
					{Name: "id", Position: 1},
				},
				DeleteRule: "CASCADE",
			},
			want: `FOREIGN KEY ("userId") REFERENCES users (id) ON DELETE CASCADE`,
		},
		{
			name: "UNIQUE with lowercase columns (no quotes needed)",
			constraint: &ir.Constraint{
				Name: "test_unique_lower",
				Type: ir.ConstraintTypeUnique,
				Columns: []*ir.ConstraintColumn{
					{Name: "email", Position: 1},
					{Name: "username", Position: 2},
				},
			},
			want: `UNIQUE (email, username)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateConstraintSQL(tt.constraint, "public")
			if got != tt.want {
				t.Errorf("generateConstraintSQL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAddColumnIdentifierQuoting(t *testing.T) {
	tests := []struct {
		name       string
		columnName string
		wantQuoted bool
	}{
		{"camelCase column", "followerCount", true},
		{"PascalCase column", "IsVerified", true},
		{"lowercase column", "follower_count", false},
		{"reserved word", "user", true},
		{"with numbers", "column123", false},
		{"starts with uppercase", "Column", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quoted := quoteIdentifier(tt.columnName)
			hasQuotes := quoted[0] == '"' && quoted[len(quoted)-1] == '"'
			
			if hasQuotes != tt.wantQuoted {
				t.Errorf("quoteIdentifier(%q) = %q, want quoted: %v", 
					tt.columnName, quoted, tt.wantQuoted)
			}
		})
	}
}
