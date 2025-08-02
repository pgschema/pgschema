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
	Summary         PlanSummary `json:"summary"`
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
		Summary:         p.calculateSummaryFromSteps(),
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

	// For tables, we need to group by table path to avoid counting duplicates
	// For other object types, count each operation individually
	
	// Track table operations by table path
	tableOperations := make(map[string]string) // table_path -> operation
	
	// Track tables that have sub-resource changes (these should be counted as modified)
	tablesWithSubResources := make(map[string]bool) // table_path -> true
	
	// Track non-table operations
	nonTableOperations := make(map[string][]string) // objType -> []operations

	for _, step := range p.Steps {
		// Normalize object type to match the expected format (add 's' for plural)
		stepObjType := step.Type
		if !strings.HasSuffix(stepObjType, "s") {
			stepObjType += "s"
		}

		if stepObjType == "tables" {
			// For tables, track unique table paths and their primary operation
			tableOperations[step.Path] = step.Operation
		} else if isSubResource(step.Type) {
			// For sub-resources, track which tables have sub-resource changes
			tablePath := extractTablePathFromSubResource(step.Path, step.Type)
			if tablePath != "" {
				tablesWithSubResources[tablePath] = true
			}
		} else {
			// For non-table objects, track each operation
			nonTableOperations[stepObjType] = append(nonTableOperations[stepObjType], step.Operation)
		}
	}

	// Count table operations (one per unique table)
	// Include both direct table operations and tables with sub-resource changes
	allAffectedTables := make(map[string]string)
	
	// First, add direct table operations
	for tablePath, operation := range tableOperations {
		allAffectedTables[tablePath] = operation
	}
	
	// Then, add tables that only have sub-resource changes (count as "alter")
	for tablePath := range tablesWithSubResources {
		if _, alreadyCounted := allAffectedTables[tablePath]; !alreadyCounted {
			allAffectedTables[tablePath] = "alter" // Sub-resource changes = table modification
		}
	}

	if len(allAffectedTables) > 0 {
		stats := summary.ByType["tables"]
		for _, operation := range allAffectedTables {
			switch operation {
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
		}
		summary.ByType["tables"] = stats
	}

	// Count non-table operations (each operation counted individually)
	for objType, operations := range nonTableOperations {
		stats := summary.ByType[objType]
		for _, operation := range operations {
			switch operation {
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
		}
		summary.ByType[objType] = stats
	}

	summary.Total = summary.Add + summary.Change + summary.Destroy
	return summary
}

// writeDetailedChangesFromSteps writes detailed changes from plan diffs
func (p *Plan) writeDetailedChangesFromSteps(summary *strings.Builder, displayName, objType string, c *color.Color) {
	fmt.Fprintf(summary, "%s:\n", c.Bold(displayName))

	if objType == "tables" {
		// For tables, group all changes by table path to avoid duplicates
		p.writeTableChanges(summary, c)
	} else {
		// For non-table objects, use the original logic
		p.writeNonTableChanges(summary, objType, c)
	}

	summary.WriteString("\n")
}

// writeTableChanges handles table-specific output with proper grouping
func (p *Plan) writeTableChanges(summary *strings.Builder, c *color.Color) {
	// Group all changes by table path and track operations
	tableOperations := make(map[string]string) // table_path -> operation
	subResources := make(map[string][]struct {
		operation string
		path      string
		subType   string
	})

	for _, step := range p.Steps {
		// Normalize object type
		stepObjType := step.Type
		if !strings.HasSuffix(stepObjType, "s") {
			stepObjType += "s"
		}

		if stepObjType == "tables" {
			// This is a table-level change, record the operation
			tableOperations[step.Path] = step.Operation
		} else if isSubResource(step.Type) {
			// This is a sub-resource change
			tablePath := extractTablePathFromSubResource(step.Path, step.Type)
			if tablePath != "" {
				subResources[tablePath] = append(subResources[tablePath], struct {
					operation string
					path      string
					subType   string
				}{
					operation: step.Operation,
					path:      step.Path,
					subType:   step.Type,
				})
			}
		}
	}

	// Get all unique table paths (from both direct table changes and sub-resources)
	allTables := make(map[string]bool)
	for tablePath := range tableOperations {
		allTables[tablePath] = true
	}
	for tablePath := range subResources {
		allTables[tablePath] = true
	}

	// Sort table paths for consistent output
	var sortedTables []string
	for tablePath := range allTables {
		sortedTables = append(sortedTables, tablePath)
	}
	sort.Strings(sortedTables)

	// Display each table once with all its changes
	for _, tablePath := range sortedTables {
		var symbol string
		if operation, hasDirectChange := tableOperations[tablePath]; hasDirectChange {
			// Table has direct changes, use the operation to determine symbol
			switch operation {
			case "create":
				symbol = c.PlanSymbol("add")
			case "alter":
				symbol = c.PlanSymbol("change")
			case "drop":
				symbol = c.PlanSymbol("destroy")
			default:
				symbol = c.PlanSymbol("change")
			}
		} else {
			// Table has no direct changes, only sub-resource changes
			// Sub-resource changes to existing tables should always be considered modifications
			symbol = c.PlanSymbol("change")
		}

		fmt.Fprintf(summary, "  %s %s\n", symbol, getLastPathComponent(tablePath))

		// Show sub-resources for this table
		if subResourceList, exists := subResources[tablePath]; exists {
			// Sort sub-resources by type then path
			sort.Slice(subResourceList, func(i, j int) bool {
				if subResourceList[i].subType != subResourceList[j].subType {
					return subResourceList[i].subType < subResourceList[j].subType
				}
				return subResourceList[i].path < subResourceList[j].path
			})

			for _, subRes := range subResourceList {
				var subSymbol string
				switch subRes.operation {
				case "create":
					subSymbol = c.PlanSymbol("add")
				case "alter":
					subSymbol = c.PlanSymbol("change")
				case "drop":
					subSymbol = c.PlanSymbol("destroy")
				default:
					subSymbol = c.PlanSymbol("change")
				}
				// Clean up sub-resource type for display (remove "table." prefix)
				displaySubType := strings.TrimPrefix(subRes.subType, "table.")
				fmt.Fprintf(summary, "    %s %s (%s)\n", subSymbol, getLastPathComponent(subRes.path), displaySubType)
			}
		}
	}
}

// writeNonTableChanges handles non-table objects with the original logic
func (p *Plan) writeNonTableChanges(summary *strings.Builder, objType string, c *color.Color) {
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

		fmt.Fprintf(summary, "  %s %s\n", symbol, getLastPathComponent(change.path))
	}
}

// isSubResource checks if the given type is a sub-resource of tables
func isSubResource(objType string) bool {
	return strings.HasPrefix(objType, "table.") && objType != "table"
}

// getLastPathComponent extracts the last component from a dot-separated path
func getLastPathComponent(path string) string {
	parts := strings.Split(path, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}

// extractTablePathFromSubResource extracts the parent table path from a sub-resource path
func extractTablePathFromSubResource(subResourcePath, subResourceType string) string {
	if strings.HasPrefix(subResourceType, "table.") {
		// For sub-resources, the path format depends on the sub-resource type:
		// - "schema.table.resource_name" -> "schema.table" (indexes, policies, columns)
		// - "schema.table" -> "schema.table" (RLS, table comments)
		parts := strings.Split(subResourcePath, ".")
		
		// Special handling for RLS and table-level changes
		if subResourceType == "table.rls" || subResourceType == "table.comment" {
			// For RLS and table comments, the path is already the table path
			return subResourcePath
		}
		
		if len(parts) >= 2 {
			// For other sub-resources, return the first two parts as table path
			if len(parts) >= 3 {
				return parts[0] + "." + parts[1]
			}
			// If only 2 parts, it's likely "schema.table" already
			return subResourcePath
		}
	}
	return ""
}
