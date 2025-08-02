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
	// The underlying diff data
	Diff *diff.DDLDiff `json:"diff"`

	// The target schema for the migration
	TargetSchema string `json:"target_schema"`

	// Plan metadata
	createdAt time.Time `json:"created_at"`

	// EnableTransaction indicates whether DDL can run in a transaction (false for CREATE INDEX CONCURRENTLY)
	EnableTransaction bool `json:"enable_transaction"`

	// Steps is the ordered list of SQL statements with their source changes
	Steps []diff.PlanStep `json:"steps,omitempty"`

	// SQLCollector is used to collect SQL statements with context
	sqlCollector *diff.SQLCollector
}

// PlanJSON represents the structured JSON output format
type PlanJSON struct {
	Version         string          `json:"version"`
	PgschemaVersion string          `json:"pgschema_version"`
	CreatedAt       time.Time       `json:"created_at"`
	Transaction     bool            `json:"transaction"`
	Steps           []diff.PlanStep `json:"steps"`
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

// ObjectType represents the database object types in dependency order
type ObjectType string

const (
	ObjectTypeSchema    ObjectType = "schemas"
	ObjectTypeType      ObjectType = "types"
	ObjectTypeFunction  ObjectType = "functions"
	ObjectTypeProcedure ObjectType = "procedures"
	ObjectTypeSequence  ObjectType = "sequences"
	ObjectTypeTable     ObjectType = "tables"
	ObjectTypeView      ObjectType = "views"
	ObjectTypeIndex     ObjectType = "indexes"
	ObjectTypeTrigger   ObjectType = "triggers"
	ObjectTypePolicy    ObjectType = "policies"
	ObjectTypeColumn    ObjectType = "columns"
	ObjectTypeRLS       ObjectType = "rls"
)

// SQLFormat represents the different output formats for SQL generation
type SQLFormat string

const (
	// SQLFormatRaw outputs just the raw SQL statements without additional formatting
	SQLFormatRaw SQLFormat = "raw"
	// SQLFormatDump outputs SQL with pg_dump-style DDL headers and separators
	SQLFormatDump SQLFormat = "dump"
)

// getObjectOrder returns the dependency order for database objects
func getObjectOrder() []ObjectType {
	return []ObjectType{
		ObjectTypeSchema,
		ObjectTypeType,
		ObjectTypeFunction,
		ObjectTypeProcedure,
		ObjectTypeSequence,
		ObjectTypeTable,
		ObjectTypeView,
		ObjectTypeIndex,
		ObjectTypeTrigger,
		ObjectTypePolicy,
		ObjectTypeColumn,
		ObjectTypeRLS,
	}
}

// ========== PUBLIC METHODS ==========

// NewPlan creates a new plan from a DDLDiff
func NewPlan(ddlDiff *diff.DDLDiff, targetSchema string) *Plan {
	plan := &Plan{
		Diff:         ddlDiff,
		TargetSchema: targetSchema,
		createdAt:    time.Now(),
		sqlCollector: diff.NewSQLCollector(),
	}
	// Enable transaction unless non-transactional DDL is present
	plan.EnableTransaction = !plan.hasNonTransactionalDDL()

	// Generate SQL and populate steps
	diff.CollectMigrationSQL(plan.Diff, plan.TargetSchema, plan.sqlCollector)
	plan.Steps = plan.sqlCollector.GetSteps()

	return plan
}

// hasNonTransactionalDDL checks if the diff contains any DDL that cannot run in a transaction
func (p *Plan) hasNonTransactionalDDL() bool {
	// Check indexes in added tables
	for _, table := range p.Diff.AddedTables {
		for _, index := range table.Indexes {
			if index.IsConcurrent {
				return true
			}
		}
	}

	// Check indexes in modified tables
	for _, table := range p.Diff.ModifiedTables {
		for _, index := range table.AddedIndexes {
			if index.IsConcurrent {
				return true
			}
		}
		// Also check modified indexes
		for _, indexDiff := range table.ModifiedIndexes {
			if indexDiff.New != nil && indexDiff.New.IsConcurrent {
				return true
			}
		}
	}
	return false
}

// HumanColored returns a human-readable summary of the plan with color support
func (p *Plan) HumanColored(enableColor bool) string {
	c := color.New(enableColor)
	var summary strings.Builder

	// Calculate summary from steps
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
		if p.EnableTransaction {
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
		Transaction:     p.EnableTransaction,
		Steps:           p.Steps,
	}

	data, err := json.MarshalIndent(planJSON, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal plan to JSON: %w", err)
	}
	return string(data), nil
}

// ToSQL returns the SQL statements with optional formatting
// format can be SQLFormatRaw (just SQL) or SQLFormatDump (with pg_dump-style DDL headers)
func (p *Plan) ToSQL(format SQLFormat) string {
	// Build SQL output from pre-generated steps
	var sqlOutput strings.Builder

	for i, step := range p.Steps {
		if format == SQLFormatDump {
			// Check if this is a comment statement
			if strings.ToUpper(step.ObjectType) == "COMMENT" {
				// For comments, just write the raw SQL without DDL header
				if i > 0 {
					sqlOutput.WriteString("\n") // Add separator from previous statement
				}
				sqlOutput.WriteString(step.SQL)
				if !strings.HasSuffix(step.SQL, "\n") {
					sqlOutput.WriteString("\n")
				}
			} else {
				// Add DDL separator with comment header for non-comment statements
				sqlOutput.WriteString("--\n")

				// Determine schema name for comment (use "-" for target schema)
				commentSchemaName := step.ObjectPath
				if strings.Contains(step.ObjectPath, ".") {
					parts := strings.Split(step.ObjectPath, ".")
					if len(parts) >= 2 && parts[0] == p.TargetSchema {
						commentSchemaName = "-"
					} else {
						commentSchemaName = parts[0]
					}
				}

				// Print object comment header
				objectName := step.ObjectPath
				if strings.Contains(step.ObjectPath, ".") {
					parts := strings.Split(step.ObjectPath, ".")
					if len(parts) >= 2 {
						objectName = parts[1]
					}
				}

				sqlOutput.WriteString(fmt.Sprintf("-- Name: %s; Type: %s; Schema: %s; Owner: -\n", objectName, strings.ToUpper(step.ObjectType), commentSchemaName))
				sqlOutput.WriteString("--\n")
				sqlOutput.WriteString("\n")

				// Add the SQL statement
				sqlOutput.WriteString(step.SQL)
			}

			// Add newline after SQL, and extra newline only if not last item
			if i < len(p.Steps)-1 {
				sqlOutput.WriteString("\n\n")
			}
		} else {
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
	}

	return sqlOutput.String()
}

// ========== PRIVATE METHODS ==========

// calculateSummaryFromSteps calculates summary statistics from the plan steps
func (p *Plan) calculateSummaryFromSteps() PlanSummary {
	summary := PlanSummary{
		ByType: make(map[string]TypeSummary),
	}

	for _, step := range p.Steps {
		// Skip sub-objects that are co-located with tables per business logic
		// Indexes, triggers, policies, columns, and RLS are not counted separately in the summary
		if step.ObjectType == "index" || step.ObjectType == "trigger" ||
			step.ObjectType == "policy" || step.ObjectType == "column" ||
			step.ObjectType == "rls" {
			continue
		}

		// Normalize object type to match the expected format (add 's' for plural)
		objType := step.ObjectType
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

// writeDetailedChangesFromSteps writes detailed changes from plan steps
func (p *Plan) writeDetailedChangesFromSteps(summary *strings.Builder, displayName, objType string, c *color.Color) {
	fmt.Fprintf(summary, "%s:\n", c.Bold(displayName))

	// Collect changes for this object type
	var changes []struct {
		operation string
		path      string
	}

	for _, step := range p.Steps {
		// Normalize object type
		stepObjType := step.ObjectType
		if !strings.HasSuffix(stepObjType, "s") {
			stepObjType += "s"
		}

		if stepObjType == objType {
			changes = append(changes, struct {
				operation string
				path      string
			}{
				operation: step.Operation,
				path:      step.ObjectPath,
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
