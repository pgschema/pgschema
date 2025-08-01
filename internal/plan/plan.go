package plan

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pgschema/pgschema/internal/color"
	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/version"
)

// Plan represents the migration plan between two DDL states
type Plan struct {
	// The target schema for the migration
	TargetSchema string

	// Plan metadata
	createdAt time.Time

	// enableTransaction indicates whether DDL can run in a transaction (false for CREATE INDEX CONCURRENTLY)
	enableTransaction bool

	// Steps is the ordered list of SQL statements with their source changes
	Steps []diff.Diff
}

// PlanJSON represents the structured JSON output format
type PlanJSON struct {
	Version         string      `json:"version"`
	PgschemaVersion string      `json:"pgschema_version"`
	CreatedAt       time.Time   `json:"created_at"`
	Transaction     bool        `json:"transaction"`
	Steps           []diff.Diff `json:"diffs"`
}

// PlanSummary provides counts of changes by type
type PlanSummary struct {
	Total   int                    `json:"total"`
	Add     int                    `json:"add"`
	Change  int                    `json:"change"`
	Destroy int                    `json:"destroy"`
	ByType  map[string]TypeSummary `json:"by_type"`
}

// TypeSummary provides counts for a specific object type
type TypeSummary struct {
	Add     int `json:"add"`
	Change  int `json:"change"`
	Destroy int `json:"destroy"`
}

// Type represents the database object types in dependency order
type Type string

const (
	TypeSchema    Type = "schemas"
	TypeType      Type = "types"
	TypeFunction  Type = "functions"
	TypeProcedure Type = "procedures"
	TypeSequence  Type = "sequences"
	TypeTable     Type = "tables"
	TypeView      Type = "views"
	TypeIndex     Type = "indexes"
	TypeTrigger   Type = "triggers"
	TypePolicy    Type = "policies"
	TypeColumn    Type = "columns"
	TypeRLS       Type = "rls"
)

// SQLFormat represents the different output formats for SQL generation
type SQLFormat string

const (
	// SQLFormatRaw outputs just the raw SQL statements without additional formatting
	SQLFormatRaw SQLFormat = "raw"
)

// getObjectOrder returns the dependency order for database objects
func getObjectOrder() []Type {
	return []Type{
		TypeSchema,
		TypeType,
		TypeFunction,
		TypeProcedure,
		TypeSequence,
		TypeTable,
		TypeView,
		TypeIndex,
		TypeTrigger,
		TypePolicy,
		TypeColumn,
		TypeRLS,
	}
}

// ========== PUBLIC METHODS ==========

// NewPlan creates a new plan from a list of diffs
func NewPlan(diffs []diff.Diff, targetSchema string) *Plan {
	plan := &Plan{
		TargetSchema: targetSchema,
		createdAt:    time.Now(),
		Steps:        diffs,
	}
	// Enable transaction unless non-transactional DDL is present
	plan.enableTransaction = plan.CanRunInTransaction()

	return plan
}

// HasAnyChanges checks if the plan contains any changes by examining the diffs
func (p *Plan) HasAnyChanges() bool {
	return len(p.Steps) > 0
}

// CanRunInTransaction checks if all plan diffs can run in a transaction
func (p *Plan) CanRunInTransaction() bool {
	// Check each step to see if it can run in a transaction
	for _, step := range p.Steps {
		if !step.CanRunInTransaction {
			return false
		}
	}
	return true
}

// HumanColored returns a human-readable summary of the plan with color support
func (p *Plan) HumanColored(enableColor bool) string {
	c := color.New(enableColor)
	var summary strings.Builder

	// Calculate summary from diffs
	summaryData := p.calculateSummaryFromSteps()

	if summaryData.Total == 0 {
		summary.WriteString("No changes detected.\n")
		return summary.String()
	}

	// Write header with overall summary (colored like Terraform)
	summary.WriteString(c.FormatPlanHeader(summaryData.Add, summaryData.Change, summaryData.Destroy) + "\n\n")

	// Write summary by type with colors
	summary.WriteString(c.Bold("Summary by type:") + "\n")
	for _, objType := range getObjectOrder() {
		objTypeStr := string(objType)
		if typeSummary, exists := summaryData.ByType[objTypeStr]; exists && (typeSummary.Add > 0 || typeSummary.Change > 0 || typeSummary.Destroy > 0) {
			line := c.FormatSummaryLine(objTypeStr, typeSummary.Add, typeSummary.Change, typeSummary.Destroy)
			summary.WriteString(line + "\n")
		}
	}
	summary.WriteString("\n")

	// Detailed changes by type with symbols
	for _, objType := range getObjectOrder() {
		objTypeStr := string(objType)
		if typeSummary, exists := summaryData.ByType[objTypeStr]; exists && (typeSummary.Add > 0 || typeSummary.Change > 0 || typeSummary.Destroy > 0) {
			// Capitalize first letter for display
			displayName := strings.ToUpper(objTypeStr[:1]) + objTypeStr[1:]
			p.writeDetailedChangesFromSteps(&summary, displayName, objTypeStr, c)
		}
	}

	// Add transaction mode information
	if summaryData.Total > 0 {
		if p.enableTransaction {
			summary.WriteString("Transaction: true\n\n")
		} else {
			summary.WriteString("Transaction: false\n\n")
		}
	}

	// Add DDL section if there are changes
	if summaryData.Total > 0 {
		summary.WriteString(c.Bold("DDL to be executed:") + "\n")
		summary.WriteString(strings.Repeat("-", 50) + "\n\n")
		migrationSQL := p.ToSQL(SQLFormatRaw)
		if migrationSQL != "" {
			summary.WriteString(migrationSQL)
			if !strings.HasSuffix(migrationSQL, "\n") {
				summary.WriteString("\n")
			}
		} else {
			summary.WriteString("-- No DDL statements generated\n")
		}
	}

	return summary.String()
}

// ToJSON returns the plan as structured JSON with only changed statements
func (p *Plan) ToJSON() (string, error) {
	planJSON := &PlanJSON{
		Version:         version.PlanFormat(),
		PgschemaVersion: version.App(),
		CreatedAt:       p.createdAt.Truncate(time.Second),
		Transaction:     p.enableTransaction,
		Steps:           p.Steps,
	}

	data, err := json.MarshalIndent(planJSON, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal plan to JSON: %w", err)
	}
	return string(data), nil
}

// ToSQL returns the SQL statements with raw formatting
func (p *Plan) ToSQL(format SQLFormat) string {
	// Build SQL output from pre-generated diffs
	var sqlOutput strings.Builder

	for i, step := range p.Steps {
		// Add the SQL statement
		sqlOutput.WriteString(step.SQL)

		// Ensure statement ends with a newline
		if !strings.HasSuffix(step.SQL, "\n") {
			sqlOutput.WriteString("\n")
		}

		// Add separator between statements (but not after the last one)
		if i < len(p.Steps)-1 {
			sqlOutput.WriteString("\n")
		}
	}

	return sqlOutput.String()
}

// ========== PRIVATE METHODS ==========

// calculateSummaryFromSteps calculates summary statistics from the plan diffs
func (p *Plan) calculateSummaryFromSteps() PlanSummary {
	summary := PlanSummary{
		ByType: make(map[string]TypeSummary),
	}

	for _, step := range p.Steps {
		// Skip sub-objects that are co-located with tables per business logic
		// Indexes, triggers, policies, columns, and RLS are not counted separately in the summary
		if step.Type == "index" || step.Type == "trigger" ||
			step.Type == "policy" || step.Type == "column" ||
			step.Type == "rls" {
			continue
		}

		// Normalize object type to match the expected format (add 's' for plural)
		objType := step.Type
		if !strings.HasSuffix(objType, "s") {
			objType += "s"
		}

		stats := summary.ByType[objType]

		switch step.Operation {
		case "create":
			stats.Add++
			summary.Add++
		case "alter":
			stats.Change++
			summary.Change++
		case "drop":
			stats.Destroy++
			summary.Destroy++
		}

		summary.ByType[objType] = stats
	}

	summary.Total = summary.Add + summary.Change + summary.Destroy
	return summary
}

// writeDetailedChangesFromSteps writes detailed changes from plan diffs
func (p *Plan) writeDetailedChangesFromSteps(summary *strings.Builder, displayName, objType string, c *color.Color) {
	fmt.Fprintf(summary, "%s:\n", c.Bold(displayName))

	// Collect changes for this object type
	var changes []struct {
		operation string
		path      string
	}

	for _, step := range p.Steps {
		// Normalize object type
		stepObjType := step.Type
		if !strings.HasSuffix(stepObjType, "s") {
			stepObjType += "s"
		}

		if stepObjType == objType {
			changes = append(changes, struct {
				operation string
				path      string
			}{
				operation: step.Operation,
				path:      step.Path,
			})
		}
	}

	// Sort changes by path for consistent output
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].path < changes[j].path
	})

	// Write changes with appropriate symbols
	for _, change := range changes {
		var symbol string
		switch change.operation {
		case "create":
			symbol = c.PlanSymbol("add")
		case "alter":
			symbol = c.PlanSymbol("change")
		case "drop":
			symbol = c.PlanSymbol("destroy")
		default:
			symbol = c.PlanSymbol("change")
		}

		fmt.Fprintf(summary, "  %s %s\n", symbol, change.path)
	}

	summary.WriteString("\n")
}
