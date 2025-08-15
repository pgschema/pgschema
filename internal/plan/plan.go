package plan

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pgschema/pgschema/internal/color"
	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/fingerprint"
	"github.com/pgschema/pgschema/internal/version"
)

// ExecutionGroup represents a group of diffs that should be executed together
type ExecutionGroup struct {
	Steps []diff.Diff `json:"steps"`
}

// Plan represents the migration plan between two DDL states
type Plan struct {
	// Version information
	Version         string `json:"version"`
	PgschemaVersion string `json:"pgschema_version"`

	// When the plan was created
	CreatedAt time.Time `json:"created_at"`

	// Source database fingerprint when plan was created
	SourceFingerprint *fingerprint.SchemaFingerprint `json:"source_fingerprint,omitempty"`

	// Groups is the ordered list of execution groups
	Groups []ExecutionGroup `json:"groups"`
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
	// Human-readable format with comments
	SQLFormatHuman SQLFormat = "human"
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

// isConcurrentIndex checks if a diff represents a concurrent index operation
func isConcurrentIndex(d diff.Diff) bool {
	return d.Type == "table.index" && !d.CanRunInTransaction
}

// groupDiffs groups diffs into execution groups, with concurrent indexes at the end
func groupDiffs(diffs []diff.Diff) []ExecutionGroup {
	if len(diffs) == 0 {
		return nil
	}

	// Separate transactional and concurrent index operations
	var transactional []diff.Diff
	var concurrentIndexes []diff.Diff

	for _, d := range diffs {
		if isConcurrentIndex(d) {
			concurrentIndexes = append(concurrentIndexes, d)
		} else {
			transactional = append(transactional, d)
		}
	}

	var groups []ExecutionGroup

	// Add transactional group first if there are any
	if len(transactional) > 0 {
		groups = append(groups, ExecutionGroup{
			Steps: transactional,
		})
	}

	// Add concurrent indexes at the end - each in its own group
	for _, ci := range concurrentIndexes {
		groups = append(groups, ExecutionGroup{
			Steps: []diff.Diff{ci},
		})
	}

	return groups
}

// NewPlan creates a new plan from a list of diffs
func NewPlan(diffs []diff.Diff) *Plan {
	plan := &Plan{
		Version:         version.PlanFormat(),
		PgschemaVersion: version.App(),
		CreatedAt:       time.Now().Truncate(time.Second),
		Groups:          groupDiffs(diffs),
	}

	return plan
}

// NewPlanWithFingerprint creates a new plan from diffs and includes source fingerprint
func NewPlanWithFingerprint(diffs []diff.Diff, sourceFingerprint *fingerprint.SchemaFingerprint) *Plan {
	plan := NewPlan(diffs)
	plan.SourceFingerprint = sourceFingerprint
	return plan
}

// HasAnyChanges checks if the plan contains any changes by examining the groups
func (p *Plan) HasAnyChanges() bool {
	for _, g := range p.Groups {
		if len(g.Steps) > 0 {
			return true
		}
	}
	return false
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

	// Add DDL section if there are changes
	if summaryData.Total > 0 {
		summary.WriteString(c.Bold("DDL to be executed:") + "\n")
		summary.WriteString(strings.Repeat("-", 50) + "\n\n")
		migrationSQL := p.ToSQL(SQLFormatHuman)
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

// ToSQL returns the SQL statements with formatting based on the specified format
func (p *Plan) ToSQL(format SQLFormat) string {
	// Build SQL output from groups
	var sqlOutput strings.Builder

	for groupIdx, group := range p.Groups {
		// Add transaction group comment for human-readable format
		if format == SQLFormatHuman {
			sqlOutput.WriteString(fmt.Sprintf("-- Transaction Group #%d\n", groupIdx+1))
		}
		
		for i, step := range group.Steps {
			// Add the SQL statement
			sqlOutput.WriteString(step.SQL)

			// Ensure statement ends with a newline
			if !strings.HasSuffix(step.SQL, "\n") {
				sqlOutput.WriteString("\n")
			}

			// Add separator between statements within group
			if i < len(group.Steps)-1 {
				sqlOutput.WriteString("\n")
			}
		}

		// Add separator between groups
		if groupIdx < len(p.Groups)-1 {
			sqlOutput.WriteString("\n")
		}
	}

	return sqlOutput.String()
}

// ToJSON returns the plan as structured JSON with only changed statements
func (p *Plan) ToJSON() (string, error) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal plan to JSON: %w", err)
	}
	return string(data), nil
}

// FromJSON creates a Plan from JSON data
func FromJSON(jsonData []byte) (*Plan, error) {
	var plan Plan
	if err := json.Unmarshal(jsonData, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan JSON: %w", err)
	}
	return &plan, nil
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

	// Flatten all steps from all groups
	var allSteps []diff.Diff
	for _, group := range p.Groups {
		allSteps = append(allSteps, group.Steps...)
	}

	for _, step := range allSteps {
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

	// Flatten all steps from all groups
	var allSteps []diff.Diff
	for _, group := range p.Groups {
		allSteps = append(allSteps, group.Steps...)
	}

	for _, step := range allSteps {
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

			// Group online index replacements
			onlineIndexChanges := make(map[string]bool)
			tempIndexNames := make(map[string]string) // temp_name -> original_name
			
			if len(subResourceList) > 0 {
				// First pass: identify all index operations
				indexOps := make(map[string][]string) // index_name -> [operations]
				
				for _, subRes := range subResourceList {
					if subRes.subType == "table.index" {
						indexName := getLastPathComponent(subRes.path)
						indexOps[indexName] = append(indexOps[indexName], subRes.operation)
					}
				}
				
				// Second pass: detect online replacement pattern
				// Look for indexes that have multiple operations AND have a corresponding _new index
				for indexName := range indexOps {
					if strings.HasSuffix(indexName, "_new") {
						// This is a temp index, find its original
						originalName := strings.TrimSuffix(indexName, "_new")
						if originalOps, exists := indexOps[originalName]; exists && len(originalOps) > 1 {
							// Check if original index has drop and rename operations
							hasDrop := false
							hasRename := false
							for _, op := range originalOps {
								if op == "drop" {
									hasDrop = true
								}
								if op == "rename" {
									hasRename = true
								}
							}
							
							// If original has drop+rename and we have a temp create, it's online replacement
							if hasDrop && hasRename {
								onlineIndexChanges[originalName] = true
								tempIndexNames[indexName] = originalName
							}
						}
					}
				}
			}

			// Track which online index changes we've already displayed
			displayedOnlineIndexes := make(map[string]bool)
			
			for _, subRes := range subResourceList {
				indexName := getLastPathComponent(subRes.path)
				
				// Handle online index replacement display
				if subRes.subType == "table.index" {
					// Skip temp index operations entirely
					if _, isTempIndex := tempIndexNames[indexName]; isTempIndex {
						continue
					}
					
					// For original indexes in online replacement, show as single concurrent change
					if onlineIndexChanges[indexName] && !displayedOnlineIndexes[indexName] {
						subSymbol := c.PlanSymbol("change")
						displaySubType := strings.TrimPrefix(subRes.subType, "table.")
						fmt.Fprintf(summary, "    %s %s (%s - concurrent rebuild)\n", subSymbol, indexName, displaySubType)
						displayedOnlineIndexes[indexName] = true
						continue
					}
					
					// Skip other operations for indexes that are being replaced online
					if onlineIndexChanges[indexName] {
						continue
					}
				}
				
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

	// Flatten all steps from all groups
	var allSteps []diff.Diff
	for _, group := range p.Groups {
		allSteps = append(allSteps, group.Steps...)
	}

	for _, step := range allSteps {
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
