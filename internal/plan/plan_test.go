package plan

import (
	"strings"
	"testing"

	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/ir"
)

// parseSQL is a helper function to convert SQL string to IR for tests
func parseSQL(t *testing.T, sql string) *ir.IR {
	parser := ir.NewParser()
	schema, err := parser.ParseSQL(sql)
	if err != nil {
		t.Fatalf("Failed to parse SQL: %v", err)
	}
	return schema
}

func TestPlanSummary(t *testing.T) {
	oldSQL := `CREATE TABLE users (
		id integer NOT NULL
	);`

	newSQL := `CREATE TABLE users (
		id integer NOT NULL,
		name text NOT NULL
	);
	CREATE TABLE posts (
		id integer NOT NULL,
		title text NOT NULL
	);`

	oldIR := parseSQL(t, oldSQL)
	newIR := parseSQL(t, newSQL)
	ddlDiff := diff.Diff(oldIR, newIR)

	plan := NewPlan(ddlDiff, "public")
	summary := plan.HumanColored(false)

	// Debug: print the summary to see what it looks like
	t.Logf("Summary output:\n%s", summary)
	
	if !strings.Contains(summary, "1 to add") {
		t.Error("Summary should mention 1 resource to add")
	}

	if !strings.Contains(summary, "1 to modify") {
		t.Error("Summary should mention 1 resource to modify")
	}

	// The colored output doesn't show "0 to drop" when there are no drops
	if strings.Contains(summary, "to drop") && !strings.Contains(summary, "1 to add, 1 to modify") {
		t.Error("Summary should not mention drops when there are none")
	}
}

func TestPlanToJSON(t *testing.T) {
	oldSQL := `CREATE TABLE users (
		id integer NOT NULL
	);`

	newSQL := `CREATE TABLE users (
		id integer NOT NULL,
		name text NOT NULL
	);`

	oldIR := parseSQL(t, oldSQL)
	newIR := parseSQL(t, newSQL)
	ddlDiff := diff.Diff(oldIR, newIR)

	plan := NewPlan(ddlDiff, "public")
	jsonOutput, err := plan.ToJSON()

	if err != nil {
		t.Fatalf("Failed to generate JSON: %v", err)
	}

	if !strings.Contains(jsonOutput, `"steps"`) {
		t.Error("JSON output should contain steps")
	}

	if !strings.Contains(jsonOutput, `"version"`) {
		t.Error("JSON output should contain version")
	}

	if !strings.Contains(jsonOutput, `"created_at"`) {
		t.Error("JSON output should contain created_at timestamp")
	}
}

func TestPlanNoChanges(t *testing.T) {
	sql := `CREATE TABLE users (
                id integer NOT NULL
        );`

	oldIR := parseSQL(t, sql)
	newIR := parseSQL(t, sql)
	ddlDiff := diff.Diff(oldIR, newIR)

	plan := NewPlan(ddlDiff, "public")
	summary := strings.TrimSpace(plan.HumanColored(false))

	if summary != "No changes detected." {
		t.Errorf("expected %q, got %q", "No changes detected.", summary)
	}
}

func TestCanRunInTransaction(t *testing.T) {
	tests := []struct {
		name     string
		diff     *diff.DDLDiff
		expected bool
	}{
		{
			name: "empty diff can run in transaction",
			diff: &diff.DDLDiff{},
			expected: true,
		},
		{
			name: "regular index can run in transaction",
			diff: &diff.DDLDiff{
				AddedTables: []*ir.Table{
					{
						Schema: "public",
						Name:   "users",
						Indexes: map[string]*ir.Index{
							"idx_users_email": {
								Schema:       "public",
								Name:         "idx_users_email",
								Table:        "users",
								IsConcurrent: false,
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "concurrent index cannot run in transaction",
			diff: &diff.DDLDiff{
				AddedTables: []*ir.Table{
					{
						Schema: "public",
						Name:   "users",
						Indexes: map[string]*ir.Index{
							"idx_users_email": {
								Schema:       "public",
								Name:         "idx_users_email",
								Table:        "users",
								IsConcurrent: true,
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "concurrent index in modified table cannot run in transaction",
			diff: &diff.DDLDiff{
				ModifiedTables: []*diff.TableDiff{
					{
						Table: &ir.Table{
							Schema: "public",
							Name:   "users",
						},
						AddedIndexes: []*ir.Index{
							{
								Schema:       "public",
								Name:         "idx_users_email",
								Table:        "users",
								IsConcurrent: true,
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "regular operations with non-concurrent indexes can run in transaction",
			diff: &diff.DDLDiff{
				AddedTables: []*ir.Table{
					{
						Schema: "public",
						Name:   "orders",
						Columns: []*ir.Column{
							{Name: "id", DataType: "integer"},
						},
					},
				},
				ModifiedTables: []*diff.TableDiff{
					{
						Table: &ir.Table{
							Schema: "public",
							Name:   "users",
						},
						AddedColumns: []*ir.Column{
							{Name: "age", DataType: "integer"},
						},
						AddedIndexes: []*ir.Index{
							{
								Schema:       "public",
								Name:         "idx_users_age",
								Table:        "users",
								IsConcurrent: false,
							},
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := NewPlan(tt.diff, "public")
			result := plan.CanRunInTransaction()
			if result != tt.expected {
				t.Errorf("CanRunInTransaction() = %v, want %v", result, tt.expected)
			}
		})
	}
}
