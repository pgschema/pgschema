package plan

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/ir"
)

// discoverTestDataVersions discovers available test data versions in the testdata directory
func discoverTestDataVersions(testdataDir string) ([]string, error) {
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read testdata directory: %w", err)
	}
	var versions []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if the directory contains a plan.json file
			planFile := filepath.Join(testdataDir, entry.Name(), "plan.json")
			if _, err := os.Stat(planFile); err == nil {
				versions = append(versions, entry.Name())
			}
		}
	}
	// Sort versions to ensure deterministic test execution order
	sort.Strings(versions)
	return versions, nil
}

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

	plan := NewPlan(diffs)
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
	testDataDir := "../../testdata/diff/migrate"

	// Discover available test data versions dynamically
	versions, err := discoverTestDataVersions(testDataDir)
	if err != nil {
		t.Fatalf("Failed to discover test data versions: %v", err)
	}

	if len(versions) == 0 {
		t.Skip("No test data versions found")
	}

	for _, version := range versions {
		t.Run(fmt.Sprintf("version_%s", version), func(t *testing.T) {
			planFilePath := filepath.Join(testDataDir, version, "plan.json")

			// Read the original plan.json file
			originalJSON, err := os.ReadFile(planFilePath)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", planFilePath, err)
			}

			// First FromJSON: Load plan from JSON
			plan1, err := FromJSON(originalJSON)
			if err != nil {
				t.Fatalf("Failed to parse JSON from %s: %v", planFilePath, err)
			}

			// Check if original JSON has source fields to determine debug mode
			hasSourceFields := strings.Contains(string(originalJSON), `"source":`)
			
			// First ToJSON: Convert plan back to JSON with same debug mode as original
			json1, err := plan1.ToJSONWithDebug(hasSourceFields)
			if err != nil {
				t.Fatalf("Failed to convert plan to JSON (first): %v", err)
			}

			// Compare original JSON with first round-trip JSON
			// Parse both JSON strings into maps to compare structure
			var originalMap, roundTripMap map[string]interface{}
			if err := json.Unmarshal(originalJSON, &originalMap); err != nil {
				t.Fatalf("Failed to unmarshal original JSON: %v", err)
			}
			if err := json.Unmarshal([]byte(json1), &roundTripMap); err != nil {
				t.Fatalf("Failed to unmarshal round-trip JSON: %v", err)
			}

			// Use go-cmp to show detailed differences
			if diff := cmp.Diff(originalMap, roundTripMap); diff != "" {
				t.Errorf("JSON round-trip failed for %s: mismatch (-original +roundtrip):\n%s", version, diff)
			}

			// Second round-trip: FromJSON -> ToJSON again
			// This should produce identical string output
			plan2, err := FromJSON([]byte(json1))
			if err != nil {
				t.Fatalf("Failed to parse JSON from round-trip: %v", err)
			}

			json2, err := plan2.ToJSONWithDebug(hasSourceFields)
			if err != nil {
				t.Fatalf("Failed to convert plan to JSON (second): %v", err)
			}

			// After first round-trip, subsequent round-trips should produce identical strings
			if json1 != json2 {
				t.Errorf("JSON not stable after first round-trip for %s", version)
				t.Logf("First round-trip length: %d", len(json1))
				t.Logf("Second round-trip length: %d", len(json2))

				// Show structural differences if any
				var map1, map2 map[string]interface{}
				json.Unmarshal([]byte(json1), &map1)
				json.Unmarshal([]byte(json2), &map2)
				if diff := cmp.Diff(map1, map2); diff != "" {
					t.Errorf("Structural difference in second round-trip (-first +second):\n%s", diff)
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

	plan := NewPlan(diffs)
	
	// Test non-debug version (default behavior) - should NOT contain source field
	jsonOutput, err := plan.ToJSON()
	if err != nil {
		t.Fatalf("Failed to generate JSON: %v", err)
	}

	if !strings.Contains(jsonOutput, `"groups"`) {
		t.Error("JSON output should contain groups")
	}
	
	// Non-debug version should NOT contain source field
	if strings.Contains(jsonOutput, `"source"`) {
		t.Error("JSON output should NOT contain source field when debug is disabled")
	}
	
	// Test debug version - should contain source field
	jsonDebugOutput, err := plan.ToJSONWithDebug(true)
	if err != nil {
		t.Fatalf("Failed to generate debug JSON: %v", err)
	}

	if !strings.Contains(jsonDebugOutput, `"groups"`) {
		t.Error("Debug JSON output should contain groups")
	}
	
	// Debug version should contain source field
	if !strings.Contains(jsonDebugOutput, `"source"`) {
		t.Error("Debug JSON output should contain source field when debug is enabled")
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

	plan := NewPlan(diffs)
	summary := strings.TrimSpace(plan.HumanColored(false))

	if summary != "No changes detected." {
		t.Errorf("expected %q, got %q", "No changes detected.", summary)
	}
}
