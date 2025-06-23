package plan

import (
	"strings"
	"testing"

	"github.com/pgschema/pgschema/internal/diff"
)

func TestNewPlan(t *testing.T) {
	oldSQL := `CREATE TABLE public.users (
		id integer NOT NULL
	);`

	newSQL := `CREATE TABLE public.users (
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
	
	if len(plan.Actions) == 0 {
		t.Error("Plan should have generated actions")
	}
	
	if plan.CreatedAt.IsZero() {
		t.Error("Plan should have a creation timestamp")
	}
}

func TestPlanSummary(t *testing.T) {
	oldSQL := `CREATE TABLE public.users (
		id integer NOT NULL
	);`

	newSQL := `CREATE TABLE public.users (
		id integer NOT NULL,
		name text NOT NULL
	);
	CREATE TABLE public.posts (
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
	
	if !strings.Contains(summary, "1 to change") {
		t.Error("Summary should mention 1 resource to change")
	}
	
	if !strings.Contains(summary, "0 to destroy") {
		t.Error("Summary should mention 0 resources to destroy")
	}
}

func TestPlanToJSON(t *testing.T) {
	oldSQL := `CREATE TABLE public.users (
		id integer NOT NULL
	);`

	newSQL := `CREATE TABLE public.users (
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
	
	if !strings.Contains(jsonOutput, `"type"`) {
		t.Error("JSON output should contain action types")
	}
	
	if !strings.Contains(jsonOutput, `"resource_type"`) {
		t.Error("JSON output should contain resource types")
	}
}

func TestPlanPreview(t *testing.T) {
	oldSQL := ``

	newSQL := `CREATE TABLE public.users (
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
	
	if !strings.Contains(preview, "+ [1] Create table public.users") {
		t.Error("Preview should show the table creation action")
	}
}

func TestGenerateMigrationSQL(t *testing.T) {
	oldSQL := `CREATE TABLE public.users (
		id integer NOT NULL
	);`

	newSQL := `CREATE TABLE public.users (
		id integer NOT NULL,
		name text NOT NULL
	);`

	ddlDiff, err := diff.Diff(oldSQL, newSQL)
	if err != nil {
		t.Fatalf("Failed to generate diff: %v", err)
	}

	plan := NewPlan(ddlDiff)
	sql := plan.GenerateMigrationSQL()
	
	if !strings.Contains(sql, "ALTER TABLE public.users ADD COLUMN name text NOT NULL") {
		t.Error("Generated SQL should contain the column addition")
	}
}