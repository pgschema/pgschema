package diff

import (
	"testing"

	"github.com/pgschema/pgschema/ir"
)

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
			want: `CONSTRAINT test_unique UNIQUE ("userId", "accountId")`,
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
			want: `CONSTRAINT test_pk PRIMARY KEY ("user", "order")`,
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
				IsValid: true,
			},
			want: `CONSTRAINT test_fk FOREIGN KEY ("userId") REFERENCES users (id) ON DELETE CASCADE`,
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
			want: `CONSTRAINT test_unique_lower UNIQUE (email, username)`,
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

func TestCheckConstraintQuoting(t *testing.T) {
	tests := []struct {
		name       string
		constraint *ir.Constraint
		want       string
	}{
		{
			name: "CHECK with camelCase column",
			constraint: &ir.Constraint{
				Name:        "positive_followers",
				Type:        ir.ConstraintTypeCheck,
				CheckClause: `CHECK ("followerCount" >= 0)`,
				IsValid:     true,
			},
			want: `CONSTRAINT positive_followers CHECK ("followerCount" >= 0)`,
		},
		{
			name: "CHECK with multiple camelCase columns and AND",
			constraint: &ir.Constraint{
				Name:        "valid_counts",
				Type:        ir.ConstraintTypeCheck,
				CheckClause: `CHECK ("likeCount" >= 0 AND "commentCount" >= 0)`,
				IsValid:     true,
			},
			want: `CONSTRAINT valid_counts CHECK ("likeCount" >= 0 AND "commentCount" >= 0)`,
		},
		{
			name: "CHECK with BETWEEN",
			constraint: &ir.Constraint{
				Name:        "stock_range",
				Type:        ir.ConstraintTypeCheck,
				CheckClause: `CHECK ("stockLevel" BETWEEN 0 AND 1000)`,
				IsValid:     true,
			},
			want: `CONSTRAINT stock_range CHECK ("stockLevel" BETWEEN 0 AND 1000)`,
		},
		{
			name: "CHECK with IN clause",
			constraint: &ir.Constraint{
				Name:        "valid_status",
				Type:        ir.ConstraintTypeCheck,
				CheckClause: `CHECK ("orderStatus" IN ('pending', 'shipped', 'delivered'))`,
				IsValid:     true,
			},
			want: `CONSTRAINT valid_status CHECK ("orderStatus" IN ('pending', 'shipped', 'delivered'))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For CHECK constraints, generateConstraintSQL returns the CheckClause as-is
			got := generateConstraintSQL(tt.constraint, "public")
			if got != tt.want {
				t.Errorf("generateConstraintSQL() for CHECK = %q, want %q", got, tt.want)
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
			quoted := ir.QuoteIdentifier(tt.columnName)
			hasQuotes := quoted[0] == '"' && quoted[len(quoted)-1] == '"'
			
			if hasQuotes != tt.wantQuoted {
				t.Errorf("ir.QuoteIdentifier(%q) = %q, want quoted: %v", 
					tt.columnName, quoted, tt.wantQuoted)
			}
		})
	}
}