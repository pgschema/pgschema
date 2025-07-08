package plan

import (
	"strings"
	"testing"

	"github.com/pgschema/pgschema/internal/diff"
)

func TestNewPlan(t *testing.T) {
	oldSQL := `CREATE TABLE users (
		id integer NOT NULL
	);`

	newSQL := `CREATE TABLE users (
		id integer NOT NULL,
		name text NOT NULL
	);`

	ddlDiff, err := diff.Diff(oldSQL, newSQL)
	if err != nil {
		t.Fatalf("Failed to generate diff: %v", err)
	}

	plan := NewPlan(ddlDiff)

	if plan.Diff != ddlDiff {
		t.Error("Plan should contain the original DDLDiff")
	}

	if plan.CreatedAt.IsZero() {
		t.Error("Plan should have a creation timestamp")
	}
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

	ddlDiff, err := diff.Diff(oldSQL, newSQL)
	if err != nil {
		t.Fatalf("Failed to generate diff: %v", err)
	}

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

	ddlDiff, err := diff.Diff(oldSQL, newSQL)
	if err != nil {
		t.Fatalf("Failed to generate diff: %v", err)
	}

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

func TestPlanPreview(t *testing.T) {
	oldSQL := ``

	newSQL := `CREATE TABLE users (
		id integer NOT NULL
	);`

	ddlDiff, err := diff.Diff(oldSQL, newSQL)
	if err != nil {
		t.Fatalf("Failed to generate diff: %v", err)
	}

	plan := NewPlan(ddlDiff)
	preview := plan.Preview()

	if !strings.Contains(preview, "Migration Plan") {
		t.Error("Preview should contain 'Migration Plan' header")
	}

	if !strings.Contains(preview, "1 to add") {
		t.Error("Preview should show resource count")
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

	ddlDiff, err := diff.Diff(oldSQL, newSQL)
	if err != nil {
		t.Fatalf("Failed to generate diff: %v", err)
	}

	plan := NewPlan(ddlDiff)
	sql := plan.GenerateMigrationSQL()

	if !strings.Contains(sql, "ALTER TABLE users ADD COLUMN name text NOT NULL") {
		t.Error("Generated SQL should contain the column addition")
	}
}
