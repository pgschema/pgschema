package plan

import (
	"fmt"
	"os"
	"path/filepath"
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
	diffs := diff.GenerateMigration(oldIR, newIR, "public")

	plan := NewPlan(diffs, "public")
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

func TestPlanJSONRoundTrip(t *testing.T) {
	testDataDir := "../../testdata/migrate"
	versions := []string{"v1", "v2", "v3", "v4", "v5"}

	for _, version := range versions {
		t.Run(fmt.Sprintf("version_%s", version), func(t *testing.T) {
			planFilePath := filepath.Join(testDataDir, version, "plan.json")
			
			// Read the original plan.json file
			originalJSON, err := os.ReadFile(planFilePath)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", planFilePath, err)
			}

			// First FromJSON: Load plan from JSON
			plan1, err := FromJSON(originalJSON, "public")
			if err != nil {
				t.Fatalf("Failed to parse JSON from %s: %v", planFilePath, err)
			}

			// First ToJSON: Convert plan back to JSON
			json1, err := plan1.ToJSON()
			if err != nil {
				t.Fatalf("Failed to convert plan to JSON (first): %v", err)
			}

			// Compare original JSON with first round-trip JSON
			if string(originalJSON) != json1 {
				t.Errorf("JSON round-trip failed for %s: original and round-trip JSON differ", version)
				t.Logf("Original JSON length: %d", len(originalJSON))
				t.Logf("Round-trip JSON length: %d", len(json1))
				// For debugging, show first difference
				minLen := len(originalJSON)
				if len(json1) < minLen {
					minLen = len(json1)
				}
				for i := 0; i < minLen; i++ {
					if originalJSON[i] != json1[i] {
						t.Logf("First difference at position %d: original=%c, round-trip=%c", i, originalJSON[i], json1[i])
						break
					}
				}
			}
		})
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
	diffs := diff.GenerateMigration(oldIR, newIR, "public")

	plan := NewPlan(diffs, "public")
	jsonOutput, err := plan.ToJSON()

	if err != nil {
		t.Fatalf("Failed to generate JSON: %v", err)
	}

	if !strings.Contains(jsonOutput, `"diffs"`) {
		t.Error("JSON output should contain diffs")
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
	diffs := diff.GenerateMigration(oldIR, newIR, "public")

	plan := NewPlan(diffs, "public")
	summary := strings.TrimSpace(plan.HumanColored(false))

	if summary != "No changes detected." {
		t.Errorf("expected %q, got %q", "No changes detected.", summary)
	}
}
