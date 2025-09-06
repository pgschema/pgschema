package diff

import (
	"testing"
	"github.com/pgschema/pgschema/internal/ir"
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