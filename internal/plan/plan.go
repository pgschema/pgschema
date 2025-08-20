package plan

import (
	"encoding/json"
	"fmt"
	"os"
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

// groupDiffs groups diffs into execution groups, respecting transaction boundaries
func groupDiffs(diffs []diff.Diff) []ExecutionGroup {
	if len(diffs) == 0 {
		return nil
	}

	var groups []ExecutionGroup
	var transactionalSteps []diff.Diff

	// Flatten all operations with rewrites into individual rewrite statements
	var flattenedOps []diff.Diff
	for _, d := range diffs {
		if d.Rewrite != nil && len(d.Rewrite.Statements) > 0 {
			// For operations with rewrites, create separate diff for each rewrite statement
			for _, rewriteStmt := range d.Rewrite.Statements {
				// Create a canonical statement equivalent for this rewrite statement
				canonicalStmt := diff.SQLStatement{
					SQL:                 d.Statements[0].SQL, // Use canonical SQL for display
					CanRunInTransaction: d.Statements[0].CanRunInTransaction,
				}

				// Create diff with single rewrite statement
				diffWithStmt := diff.Diff{
					Statements: []diff.SQLStatement{canonicalStmt},
					Type:       d.Type,
					Operation:  d.Operation,
					Path:       d.Path,
					Source:     d.Source,
					Rewrite: &diff.DiffRewrite{
						Statements: []diff.RewriteStatement{rewriteStmt},
					},
				}
				flattenedOps = append(flattenedOps, diffWithStmt)
			}
		} else {
			// Operations without rewrites are used as-is
			flattenedOps = append(flattenedOps, d)
		}
	}

	// Group flattened operations by transaction boundaries
	for _, op := range flattenedOps {
		var hasNonTransactional bool
		var hasDirective bool

		// Check if operation has non-transactional statements or directives
		if op.Rewrite != nil && len(op.Rewrite.Statements) > 0 {
			// Check the rewrite statement for transaction compatibility and directives
			rewriteStmt := op.Rewrite.Statements[0]
			hasNonTransactional = !rewriteStmt.CanRunInTransaction
			hasDirective = rewriteStmt.Directive != nil
		} else {
			// Check canonical statements
			for _, stmt := range op.Statements {
				if !stmt.CanRunInTransaction {
					hasNonTransactional = true
					break
				}
			}
		}

		if hasNonTransactional || hasDirective {
			// Flush any pending transactional operations
			if len(transactionalSteps) > 0 {
				groups = append(groups, ExecutionGroup{Steps: transactionalSteps})
				transactionalSteps = nil
			}

			// Add this non-transactional or directive operation in its own group
			groups = append(groups, ExecutionGroup{Steps: []diff.Diff{op}})
		} else {
			// Accumulate transactional operations
			transactionalSteps = append(transactionalSteps, op)
		}
	}

	// Flush remaining transactional operations
	if len(transactionalSteps) > 0 {
		groups = append(groups, ExecutionGroup{Steps: transactionalSteps})
	}

	return groups
}

// NewPlan creates a new plan from a list of diffs with online operations enabled
func NewPlan(diffs []diff.Diff) *Plan {
	return NewPlanWithOptions(diffs, true)
}

// NewPlanWithOptions creates a new plan from a list of diffs with configurable options
func NewPlanWithOptions(diffs []diff.Diff, enableOnlineOperations bool) *Plan {
	// Use environment variable for timestamp if provided, otherwise use current time
	createdAt := time.Now().Truncate(time.Second)
	if testTime := os.Getenv("PGSCHEMA_TEST_TIME"); testTime != "" {
		if parsedTime, err := time.Parse(time.RFC3339, testTime); err == nil {
			createdAt = parsedTime
		}
	}

	plan := &Plan{
		Version:         version.PlanFormat(),
		PgschemaVersion: version.App(),
		CreatedAt:       createdAt,
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

// NewPlanWithFingerprintAndOptions creates a new plan with fingerprint and configurable options
func NewPlanWithFingerprintAndOptions(diffs []diff.Diff, sourceFingerprint *fingerprint.SchemaFingerprint, enableOnlineOperations bool) *Plan {
	plan := NewPlanWithOptions(diffs, enableOnlineOperations)
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
		if format == SQLFormatHuman && len(p.Groups) > 1 {
			sqlOutput.WriteString(fmt.Sprintf("-- Transaction Group #%d\n", groupIdx+1))
		}

		for stepIdx, step := range group.Steps {
			// Use rewrite statements if available, otherwise canonical statements
			statementsToOutput := step.Statements
			var rewriteStatements []diff.RewriteStatement
			if step.Rewrite != nil {
				rewriteStatements = step.Rewrite.Statements
				// Convert rewrite statements to SQLStatements for output
				var convertedStatements []diff.SQLStatement
				for _, rewriteStmt := range step.Rewrite.Statements {
					convertedStatements = append(convertedStatements, diff.SQLStatement{
						SQL:                 rewriteStmt.SQL,
						CanRunInTransaction: rewriteStmt.CanRunInTransaction,
					})
				}
				statementsToOutput = convertedStatements
			}

			for stmtIdx, stmt := range statementsToOutput {
				// Find the corresponding directive for this statement
				var directive *diff.Directive
				if step.Rewrite != nil && len(rewriteStatements) > stmtIdx {
					directive = rewriteStatements[stmtIdx].Directive
				}
				
				if directive != nil {
					// Handle directive statements
					sqlOutput.WriteString(fmt.Sprintf("-- pgschema:%s\n", directive.Type))
					sqlOutput.WriteString(stmt.SQL)
					sqlOutput.WriteString("\n")
				} else {
					// Handle regular SQL statements
					sqlOutput.WriteString(stmt.SQL)
					sqlOutput.WriteString("\n")
				}

				// Add blank line between statements except for the last one in the last step
				if stmtIdx < len(statementsToOutput)-1 || stepIdx < len(group.Steps)-1 {
					sqlOutput.WriteString("\n")
				}
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
	return p.ToJSONWithDebug(false)
}

// ToJSONWithDebug returns the plan as structured JSON with optional source field inclusion
func (p *Plan) ToJSONWithDebug(includeSource bool) (string, error) {
	var buf strings.Builder
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	
	// Create a deep copy of the plan to avoid modifying the original
	planCopy := *p
	planCopy.Groups = make([]ExecutionGroup, len(p.Groups))
	
	for i, group := range p.Groups {
		planCopy.Groups[i] = ExecutionGroup{
			Steps: make([]diff.Diff, len(group.Steps)),
		}
		
		for j, step := range group.Steps {
			planCopy.Groups[i].Steps[j] = step // Copy the step
			if !includeSource {
				planCopy.Groups[i].Steps[j].Source = nil // Set to nil so omitempty excludes it
			}
		}
	}
	
	if err := encoder.Encode(&planCopy); err != nil {
		return "", fmt.Errorf("failed to marshal plan to JSON: %w", err)
	}
	
	// Remove the trailing newline that encoder.Encode adds
	result := buf.String()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}
	
	return result, nil
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
		stepObjTypeStr := step.Type.String()
		if !strings.HasSuffix(stepObjTypeStr, "s") {
			stepObjTypeStr += "s"
		}

		if stepObjTypeStr == "tables" {
			// For tables, track unique table paths and their primary operation
			tableOperations[step.Path] = step.Operation.String()
		} else if isSubResource(step.Type.String()) {
			// For sub-resources, track which tables have sub-resource changes
			tablePath := extractTablePathFromSubResource(step.Path, step.Type.String())
			if tablePath != "" {
				tablesWithSubResources[tablePath] = true
			}
		} else {
			// For non-table objects, track each operation
			nonTableOperations[stepObjTypeStr] = append(nonTableOperations[stepObjTypeStr], step.Operation.String())
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

	// Track all seen operations globally to avoid duplicates across groups
	seenOperations := make(map[string]bool) // "path.operation.subType" -> true

	// Flatten all steps from all groups
	var allSteps []diff.Diff
	for _, group := range p.Groups {
		allSteps = append(allSteps, group.Steps...)
	}

	for _, step := range allSteps {
		// Normalize object type
		stepObjTypeStr := step.Type.String()
		if !strings.HasSuffix(stepObjTypeStr, "s") {
			stepObjTypeStr += "s"
		}

		if stepObjTypeStr == "tables" {
			// This is a table-level change, record the operation
			tableOperations[step.Path] = step.Operation.String()
		} else if isSubResource(step.Type.String()) {
			// This is a sub-resource change
			tablePath := extractTablePathFromSubResource(step.Path, step.Type.String())
			if tablePath != "" {
				// Deduplicate all operations based on (type, operation, path) triplet
				operationKey := step.Path + "." + step.Operation.String() + "." + step.Type.String()
				if !seenOperations[operationKey] {
					seenOperations[operationKey] = true
					subResources[tablePath] = append(subResources[tablePath], struct {
						operation string
						path      string
						subType   string
					}{
						operation: step.Operation.String(),
						path:      step.Path,
						subType:   step.Type.String(),
					})
				}
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
				// Handle online index replacement display
				if subRes.subType == diff.DiffTypeTableIndex.String() && subRes.operation == diff.DiffOperationAlter.String() {
					subSymbol := c.PlanSymbol("change")
					displaySubType := strings.TrimPrefix(subRes.subType, "table.")
					fmt.Fprintf(summary, "    %s %s (%s - concurrent rebuild)\n", subSymbol, getLastPathComponent(subRes.path), displaySubType)
					continue
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
		stepObjTypeStr := step.Type.String()
		if !strings.HasSuffix(stepObjTypeStr, "s") {
			stepObjTypeStr += "s"
		}

		if stepObjTypeStr == objType {
			changes = append(changes, struct {
				operation string
				path      string
			}{
				operation: step.Operation.String(),
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
