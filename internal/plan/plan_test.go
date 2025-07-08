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

	plan := NewPlan(ddlDiff)
	summary := plan.Summary()

	if !strings.Contains(summary, "1 to add") {
		t.Error("Summary should mention 1 resource to add")
	}

	if !strings.Contains(summary, "1 to modify") {
		t.Error("Summary should mention 1 resource to modify")
	}

	if !strings.Contains(summary, "0 to drop") {
		t.Error("Summary should mention 0 resources to drop")
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

	plan := NewPlan(ddlDiff)
	jsonOutput, err := plan.ToJSON()

	if err != nil {
		t.Fatalf("Failed to generate JSON: %v", err)
	}

	if !strings.Contains(jsonOutput, `"object_changes"`) {
		t.Error("JSON output should contain object_changes")
	}

	if !strings.Contains(jsonOutput, `"summary"`) {
		t.Error("JSON output should contain summary")
	}

	if !strings.Contains(jsonOutput, `"format_version"`) {
		t.Error("JSON output should contain format_version")
	}

	if !strings.Contains(jsonOutput, `"created_at"`) {
		t.Error("JSON output should contain created_at timestamp")
	}
}

func TestGenerateMigrationSQL(t *testing.T) {
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

	plan := NewPlan(ddlDiff)
	sql := plan.GenerateMigrationSQL()

	if !strings.Contains(sql, "ALTER TABLE users ADD COLUMN name text NOT NULL") {
		t.Error("Generated SQL should contain the column addition")
	}
}
