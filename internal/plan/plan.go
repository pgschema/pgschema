package plan

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/ir"
)

// Version constant for pgschema
const pgschemaVersion = "0.1.5"

// Plan represents the migration plan between two DDL states
type Plan struct {
	// The underlying diff data
	Diff *diff.DDLDiff `json:"diff"`

	// Plan metadata
	CreatedAt time.Time `json:"created_at"`
}

// typeCounts holds counts for each type of change
type typeCounts struct {
	added    int
	modified int
	dropped  int
}

// ObjectChange represents a single change to a database object
type ObjectChange struct {
	Address  string         `json:"address"`
	Mode     string         `json:"mode"`
	Type     string         `json:"type"`
	Name     string         `json:"name"`
	Schema   string         `json:"schema"`
	Table    string         `json:"table,omitempty"`
	Change   Change         `json:"change"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Change represents the actual change being made
type Change struct {
	Actions []string       `json:"actions"`
	Before  map[string]any `json:"before"`
	After   map[string]any `json:"after"`
}

// PlanJSON represents the structured JSON output format
type PlanJSON struct {
	Version         string         `json:"version"`
	PgschemaVersion string         `json:"pgschema_version"`
	CreatedAt       time.Time      `json:"created_at"`
	ObjectChanges   []ObjectChange `json:"object_changes"`
	Summary         PlanSummary    `json:"summary"`
}

// PlanSummary provides counts of changes by type
type PlanSummary struct {
	Add     int                    `json:"add"`
	Change  int                    `json:"change"`
	Destroy int                    `json:"destroy"`
	Total   int                    `json:"total"`
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
	ObjectTypeExtension ObjectType = "extensions"
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

// getObjectOrder returns the dependency order for database objects
func getObjectOrder() []ObjectType {
	return []ObjectType{
		ObjectTypeSchema,
		ObjectTypeExtension,
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
func NewPlan(ddlDiff *diff.DDLDiff) *Plan {
	return &Plan{
		Diff:      ddlDiff,
		CreatedAt: time.Now(),
	}
}

// Summary returns a human-readable summary of the plan
func (p *Plan) Summary() string {
	var summary strings.Builder

	// Count changes by type
	typeCounts := p.getTypeCountsDetailed()

	// Calculate totals
	totalAdd := 0
	totalModify := 0
	totalDrop := 0

	for _, counts := range typeCounts {
		totalAdd += counts.added
		totalModify += counts.modified
		totalDrop += counts.dropped
	}

	totalChanges := totalAdd + totalModify + totalDrop

	if totalChanges == 0 {
		summary.WriteString("No changes detected.\n")
		return summary.String()
	}

	// Write header with overall summary
	summary.WriteString(fmt.Sprintf("Plan: %d to add, %d to modify, %d to drop.\n\n", totalAdd, totalModify, totalDrop))

	// Write summary by type
	summary.WriteString("Summary by type:\n")
	for _, objType := range getObjectOrder() {
		objTypeStr := string(objType)
		if counts, exists := typeCounts[objTypeStr]; exists && (counts.added > 0 || counts.modified > 0 || counts.dropped > 0) {
			summary.WriteString(fmt.Sprintf("  %s: %d to add, %d to modify, %d to drop\n",
				objTypeStr, counts.added, counts.modified, counts.dropped))
		}
	}
	summary.WriteString("\n")

	// Detailed changes by type
	for _, objType := range getObjectOrder() {
		objTypeStr := string(objType)
		if counts, exists := typeCounts[objTypeStr]; exists {
			// Capitalize first letter for display
			displayName := strings.ToUpper(objTypeStr[:1]) + objTypeStr[1:]
			p.writeDetailedChanges(&summary, displayName, counts)
		}
	}

	// Add DDL section if there are changes
	if totalChanges > 0 {
		summary.WriteString("DDL to be executed:\n")
		summary.WriteString(strings.Repeat("-", 50) + "\n")
		migrationSQL := diff.GenerateMigrationSQL(p.Diff, "public")
		if migrationSQL != "" {
			summary.WriteString(migrationSQL)
			if !strings.HasSuffix(migrationSQL, "\n") {
				summary.WriteString("\n")
			}
		} else {
			summary.WriteString("-- No DDL statements generated\n")
		}
		summary.WriteString(strings.Repeat("-", 50) + "\n")
	}

	return summary.String()
}

// ToJSON returns the plan as structured JSON with only changed statements
func (p *Plan) ToJSON() (string, error) {
	planJSON := p.convertToStructuredJSON()

	data, err := json.MarshalIndent(planJSON, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal plan to JSON: %w", err)
	}
	return string(data), nil
}

// ToSQL returns only the SQL statements without any additional formatting
func (p *Plan) ToSQL() string {
	// Count total changes to check if there are any
	typeCounts := p.getTypeCountsDetailed()
	totalChanges := 0
	for _, counts := range typeCounts {
		totalChanges += counts.added + counts.modified + counts.dropped
	}

	if totalChanges == 0 {
		return "-- No changes detected\n"
	}

	// Generate migration SQL
	migrationSQL := diff.GenerateMigrationSQL(p.Diff, "public")
	if migrationSQL == "" {
		return "-- No DDL statements generated\n"
	}

	return migrationSQL
}

// ========== PRIVATE METHODS ==========

// getFullObjectName returns the full qualified name for sorting
func getFullObjectName(schema, name string) string {
	return fmt.Sprintf("%s.%s", schema, name)
}

// getTypeCountsDetailed returns detailed counts by object type
func (p *Plan) getTypeCountsDetailed() map[string]typeCounts {
	counts := make(map[string]typeCounts)

	// Schemas
	counts["schemas"] = typeCounts{
		added:    len(p.Diff.AddedSchemas),
		modified: len(p.Diff.ModifiedSchemas),
		dropped:  len(p.Diff.DroppedSchemas),
	}

	// Tables
	counts["tables"] = typeCounts{
		added:    len(p.Diff.AddedTables),
		modified: len(p.Diff.ModifiedTables),
		dropped:  len(p.Diff.DroppedTables),
	}

	// Views
	counts["views"] = typeCounts{
		added:    len(p.Diff.AddedViews),
		modified: len(p.Diff.ModifiedViews),
		dropped:  len(p.Diff.DroppedViews),
	}

	// Functions
	counts["functions"] = typeCounts{
		added:    len(p.Diff.AddedFunctions),
		modified: len(p.Diff.ModifiedFunctions),
		dropped:  len(p.Diff.DroppedFunctions),
	}

	// Procedures
	counts["procedures"] = typeCounts{
		added:    len(p.Diff.AddedProcedures),
		modified: len(p.Diff.ModifiedProcedures),
		dropped:  len(p.Diff.DroppedProcedures),
	}

	// Types
	counts["types"] = typeCounts{
		added:    len(p.Diff.AddedTypes),
		modified: len(p.Diff.ModifiedTypes),
		dropped:  len(p.Diff.DroppedTypes),
	}

	// Extensions
	counts["extensions"] = typeCounts{
		added:    len(p.Diff.AddedExtensions),
		modified: 0, // Extensions typically don't get modified
		dropped:  len(p.Diff.DroppedExtensions),
	}

	// Indexes, triggers, and policies are now co-located with tables
	// They are not counted separately in the summary
	indexCounts := typeCounts{0, 0, 0}
	triggerCounts := typeCounts{0, 0, 0}
	policyCounts := typeCounts{0, 0, 0}

	// Keep zero counts to avoid showing these sections
	counts["indexes"] = indexCounts
	counts["triggers"] = triggerCounts
	counts["policies"] = policyCounts

	// Sequences (placeholder for future implementation)
	counts["sequences"] = typeCounts{0, 0, 0}

	return counts
}

// writeDetailedChanges writes detailed changes for a specific object type
func (p *Plan) writeDetailedChanges(summary *strings.Builder, typeName string, counts typeCounts) {
	if counts.added == 0 && counts.modified == 0 && counts.dropped == 0 {
		return
	}

	fmt.Fprintf(summary, "%s:\n", typeName)

	switch typeName {
	case "Schemas":
		p.writeSchemaChanges(summary)
	case "Extensions":
		p.writeExtensionChanges(summary)
	case "Types":
		p.writeTypeChanges(summary)
	case "Functions":
		p.writeFunctionChanges(summary)
	case "Procedures":
		p.writeProcedureChanges(summary)
	case "Sequences":
		p.writeSequenceChanges(summary)
	case "Tables":
		p.writeTableChanges(summary)
	case "Views":
		p.writeViewChanges(summary)
	case "Indexes":
		// Indexes are co-located with tables
		// No separate output needed
	case "Triggers":
		// Triggers are co-located with tables
		// No separate output needed
	case "Policies":
		// Policies are co-located with tables
		// No separate output needed
	case "Columns":
		// Columns are co-located with tables
		// No separate output needed
	case "Rls":
		// RLS changes are handled as part of table modifications
		// No separate output needed
	}

	summary.WriteString("\n")
}

// writeSchemaChanges writes schema changes
func (p *Plan) writeSchemaChanges(summary *strings.Builder) {
	// Sort added schemas
	addedSchemas := make([]*ir.Schema, len(p.Diff.AddedSchemas))
	copy(addedSchemas, p.Diff.AddedSchemas)
	sort.Slice(addedSchemas, func(i, j int) bool {
		return addedSchemas[i].Name < addedSchemas[j].Name
	})
	for _, schema := range addedSchemas {
		fmt.Fprintf(summary, "  + %s\n", schema.Name)
	}

	// Sort modified schemas
	modifiedSchemas := make([]*diff.SchemaDiff, len(p.Diff.ModifiedSchemas))
	copy(modifiedSchemas, p.Diff.ModifiedSchemas)
	sort.Slice(modifiedSchemas, func(i, j int) bool {
		return modifiedSchemas[i].New.Name < modifiedSchemas[j].New.Name
	})
	for _, schemaDiff := range modifiedSchemas {
		fmt.Fprintf(summary, "  ~ %s\n", schemaDiff.New.Name)
	}

	// Sort dropped schemas
	droppedSchemas := make([]*ir.Schema, len(p.Diff.DroppedSchemas))
	copy(droppedSchemas, p.Diff.DroppedSchemas)
	sort.Slice(droppedSchemas, func(i, j int) bool {
		return droppedSchemas[i].Name < droppedSchemas[j].Name
	})
	for _, schema := range droppedSchemas {
		fmt.Fprintf(summary, "  - %s\n", schema.Name)
	}
}

// writeTableChanges writes table changes with co-located indexes, triggers, and policies
func (p *Plan) writeTableChanges(summary *strings.Builder) {
	// Sort added tables
	addedTables := make([]*ir.Table, len(p.Diff.AddedTables))
	copy(addedTables, p.Diff.AddedTables)
	sort.Slice(addedTables, func(i, j int) bool {
		return getFullObjectName(addedTables[i].Schema, addedTables[i].Name) <
			getFullObjectName(addedTables[j].Schema, addedTables[j].Name)
	})
	for _, table := range addedTables {
		fmt.Fprintf(summary, "  + %s.%s\n", table.Schema, table.Name)

		// Co-locate indexes for added tables
		var tableIndexes []*ir.Index
		for _, index := range table.Indexes {
			tableIndexes = append(tableIndexes, index)
		}
		sort.Slice(tableIndexes, func(i, j int) bool {
			return tableIndexes[i].Name < tableIndexes[j].Name
		})
		for _, index := range tableIndexes {
			fmt.Fprintf(summary, "    + index %s\n", index.Name)
		}

		// Co-locate triggers for added tables
		var tableTriggers []*ir.Trigger
		for _, trigger := range table.Triggers {
			tableTriggers = append(tableTriggers, trigger)
		}
		sort.Slice(tableTriggers, func(i, j int) bool {
			return tableTriggers[i].Name < tableTriggers[j].Name
		})
		for _, trigger := range tableTriggers {
			fmt.Fprintf(summary, "    + trigger %s\n", trigger.Name)
		}

		// Co-locate policies for added tables
		var tablePolicies []*ir.RLSPolicy
		for _, policy := range table.Policies {
			tablePolicies = append(tablePolicies, policy)
		}
		sort.Slice(tablePolicies, func(i, j int) bool {
			return tablePolicies[i].Name < tablePolicies[j].Name
		})
		for _, policy := range tablePolicies {
			fmt.Fprintf(summary, "    + policy %s\n", policy.Name)
		}

		// Co-locate constraints for added tables
		var tableConstraints []*ir.Constraint
		for _, constraint := range table.Constraints {
			tableConstraints = append(tableConstraints, constraint)
		}
		sort.Slice(tableConstraints, func(i, j int) bool {
			return tableConstraints[i].Name < tableConstraints[j].Name
		})
		for _, constraint := range tableConstraints {
			fmt.Fprintf(summary, "    + constraint %s\n", constraint.Name)
		}
	}

	// Sort modified tables with their related objects
	modifiedTables := make([]*diff.TableDiff, len(p.Diff.ModifiedTables))
	copy(modifiedTables, p.Diff.ModifiedTables)
	sort.Slice(modifiedTables, func(i, j int) bool {
		return getFullObjectName(modifiedTables[i].Table.Schema, modifiedTables[i].Table.Name) <
			getFullObjectName(modifiedTables[j].Table.Schema, modifiedTables[j].Table.Name)
	})
	for _, tableDiff := range modifiedTables {
		fmt.Fprintf(summary, "  ~ %s.%s\n", tableDiff.Table.Schema, tableDiff.Table.Name)

		// Co-locate added indexes
		addedIndexes := make([]*ir.Index, len(tableDiff.AddedIndexes))
		copy(addedIndexes, tableDiff.AddedIndexes)
		sort.Slice(addedIndexes, func(i, j int) bool {
			return addedIndexes[i].Name < addedIndexes[j].Name
		})
		for _, index := range addedIndexes {
			fmt.Fprintf(summary, "    + index %s\n", index.Name)
		}

		// Co-locate dropped indexes
		droppedIndexes := make([]*ir.Index, len(tableDiff.DroppedIndexes))
		copy(droppedIndexes, tableDiff.DroppedIndexes)
		sort.Slice(droppedIndexes, func(i, j int) bool {
			return droppedIndexes[i].Name < droppedIndexes[j].Name
		})
		for _, index := range droppedIndexes {
			fmt.Fprintf(summary, "    - index %s\n", index.Name)
		}

		// Co-locate added triggers
		addedTriggers := make([]*ir.Trigger, len(tableDiff.AddedTriggers))
		copy(addedTriggers, tableDiff.AddedTriggers)
		sort.Slice(addedTriggers, func(i, j int) bool {
			return addedTriggers[i].Name < addedTriggers[j].Name
		})
		for _, trigger := range addedTriggers {
			fmt.Fprintf(summary, "    + trigger %s\n", trigger.Name)
		}

		// Co-locate modified triggers
		modifiedTriggers := make([]*diff.TriggerDiff, len(tableDiff.ModifiedTriggers))
		copy(modifiedTriggers, tableDiff.ModifiedTriggers)
		sort.Slice(modifiedTriggers, func(i, j int) bool {
			return modifiedTriggers[i].New.Name < modifiedTriggers[j].New.Name
		})
		for _, triggerDiff := range modifiedTriggers {
			fmt.Fprintf(summary, "    ~ trigger %s\n", triggerDiff.New.Name)
		}

		// Co-locate dropped triggers
		droppedTriggers := make([]*ir.Trigger, len(tableDiff.DroppedTriggers))
		copy(droppedTriggers, tableDiff.DroppedTriggers)
		sort.Slice(droppedTriggers, func(i, j int) bool {
			return droppedTriggers[i].Name < droppedTriggers[j].Name
		})
		for _, trigger := range droppedTriggers {
			fmt.Fprintf(summary, "    - trigger %s\n", trigger.Name)
		}

		// Co-locate added policies
		addedPolicies := make([]*ir.RLSPolicy, len(tableDiff.AddedPolicies))
		copy(addedPolicies, tableDiff.AddedPolicies)
		sort.Slice(addedPolicies, func(i, j int) bool {
			return addedPolicies[i].Name < addedPolicies[j].Name
		})
		for _, policy := range addedPolicies {
			fmt.Fprintf(summary, "    + policy %s\n", policy.Name)
		}

		// Co-locate modified policies
		modifiedPolicies := make([]*diff.PolicyDiff, len(tableDiff.ModifiedPolicies))
		copy(modifiedPolicies, tableDiff.ModifiedPolicies)
		sort.Slice(modifiedPolicies, func(i, j int) bool {
			return modifiedPolicies[i].New.Name < modifiedPolicies[j].New.Name
		})
		for _, policyDiff := range modifiedPolicies {
			fmt.Fprintf(summary, "    ~ policy %s\n", policyDiff.New.Name)
		}

		// Co-locate dropped policies
		droppedPolicies := make([]*ir.RLSPolicy, len(tableDiff.DroppedPolicies))
		copy(droppedPolicies, tableDiff.DroppedPolicies)
		sort.Slice(droppedPolicies, func(i, j int) bool {
			return droppedPolicies[i].Name < droppedPolicies[j].Name
		})
		for _, policy := range droppedPolicies {
			fmt.Fprintf(summary, "    - policy %s\n", policy.Name)
		}

		// Co-locate added constraints
		addedConstraints := make([]*ir.Constraint, len(tableDiff.AddedConstraints))
		copy(addedConstraints, tableDiff.AddedConstraints)
		sort.Slice(addedConstraints, func(i, j int) bool {
			return addedConstraints[i].Name < addedConstraints[j].Name
		})
		for _, constraint := range addedConstraints {
			fmt.Fprintf(summary, "    + constraint %s\n", constraint.Name)
		}

		// Co-locate dropped constraints
		droppedConstraints := make([]*ir.Constraint, len(tableDiff.DroppedConstraints))
		copy(droppedConstraints, tableDiff.DroppedConstraints)
		sort.Slice(droppedConstraints, func(i, j int) bool {
			return droppedConstraints[i].Name < droppedConstraints[j].Name
		})
		for _, constraint := range droppedConstraints {
			fmt.Fprintf(summary, "    - constraint %s\n", constraint.Name)
		}

		// Co-locate added columns
		addedColumns := make([]*ir.Column, len(tableDiff.AddedColumns))
		copy(addedColumns, tableDiff.AddedColumns)
		sort.Slice(addedColumns, func(i, j int) bool {
			return addedColumns[i].Name < addedColumns[j].Name
		})
		for _, column := range addedColumns {
			fmt.Fprintf(summary, "    + column %s\n", column.Name)
		}

		// Co-locate modified columns
		modifiedColumns := make([]*diff.ColumnDiff, len(tableDiff.ModifiedColumns))
		copy(modifiedColumns, tableDiff.ModifiedColumns)
		sort.Slice(modifiedColumns, func(i, j int) bool {
			return modifiedColumns[i].New.Name < modifiedColumns[j].New.Name
		})
		for _, columnDiff := range modifiedColumns {
			fmt.Fprintf(summary, "    ~ column %s\n", columnDiff.New.Name)
		}

		// Co-locate dropped columns
		droppedColumns := make([]*ir.Column, len(tableDiff.DroppedColumns))
		copy(droppedColumns, tableDiff.DroppedColumns)
		sort.Slice(droppedColumns, func(i, j int) bool {
			return droppedColumns[i].Name < droppedColumns[j].Name
		})
		for _, column := range droppedColumns {
			fmt.Fprintf(summary, "    - column %s\n", column.Name)
		}
	}

	// Sort dropped tables
	droppedTables := make([]*ir.Table, len(p.Diff.DroppedTables))
	copy(droppedTables, p.Diff.DroppedTables)
	sort.Slice(droppedTables, func(i, j int) bool {
		return getFullObjectName(droppedTables[i].Schema, droppedTables[i].Name) <
			getFullObjectName(droppedTables[j].Schema, droppedTables[j].Name)
	})
	for _, table := range droppedTables {
		fmt.Fprintf(summary, "  - %s.%s\n", table.Schema, table.Name)
	}
}

// writeViewChanges writes view changes
func (p *Plan) writeViewChanges(summary *strings.Builder) {
	// Sort added views
	addedViews := make([]*ir.View, len(p.Diff.AddedViews))
	copy(addedViews, p.Diff.AddedViews)
	sort.Slice(addedViews, func(i, j int) bool {
		return getFullObjectName(addedViews[i].Schema, addedViews[i].Name) <
			getFullObjectName(addedViews[j].Schema, addedViews[j].Name)
	})
	for _, view := range addedViews {
		fmt.Fprintf(summary, "  + %s.%s\n", view.Schema, view.Name)
	}

	// Sort modified views
	modifiedViews := make([]*diff.ViewDiff, len(p.Diff.ModifiedViews))
	copy(modifiedViews, p.Diff.ModifiedViews)
	sort.Slice(modifiedViews, func(i, j int) bool {
		return getFullObjectName(modifiedViews[i].New.Schema, modifiedViews[i].New.Name) <
			getFullObjectName(modifiedViews[j].New.Schema, modifiedViews[j].New.Name)
	})
	for _, viewDiff := range modifiedViews {
		fmt.Fprintf(summary, "  ~ %s.%s\n", viewDiff.New.Schema, viewDiff.New.Name)
	}

	// Sort dropped views
	droppedViews := make([]*ir.View, len(p.Diff.DroppedViews))
	copy(droppedViews, p.Diff.DroppedViews)
	sort.Slice(droppedViews, func(i, j int) bool {
		return getFullObjectName(droppedViews[i].Schema, droppedViews[i].Name) <
			getFullObjectName(droppedViews[j].Schema, droppedViews[j].Name)
	})
	for _, view := range droppedViews {
		fmt.Fprintf(summary, "  - %s.%s\n", view.Schema, view.Name)
	}
}

// writeFunctionChanges writes function changes
func (p *Plan) writeFunctionChanges(summary *strings.Builder) {
	// Sort added functions
	addedFunctions := make([]*ir.Function, len(p.Diff.AddedFunctions))
	copy(addedFunctions, p.Diff.AddedFunctions)
	sort.Slice(addedFunctions, func(i, j int) bool {
		return getFullObjectName(addedFunctions[i].Schema, addedFunctions[i].Name) <
			getFullObjectName(addedFunctions[j].Schema, addedFunctions[j].Name)
	})
	for _, function := range addedFunctions {
		fmt.Fprintf(summary, "  + %s.%s\n", function.Schema, function.Name)
	}

	// Sort modified functions
	modifiedFunctions := make([]*diff.FunctionDiff, len(p.Diff.ModifiedFunctions))
	copy(modifiedFunctions, p.Diff.ModifiedFunctions)
	sort.Slice(modifiedFunctions, func(i, j int) bool {
		return getFullObjectName(modifiedFunctions[i].New.Schema, modifiedFunctions[i].New.Name) <
			getFullObjectName(modifiedFunctions[j].New.Schema, modifiedFunctions[j].New.Name)
	})
	for _, functionDiff := range modifiedFunctions {
		fmt.Fprintf(summary, "  ~ %s.%s\n", functionDiff.New.Schema, functionDiff.New.Name)
	}

	// Sort dropped functions
	droppedFunctions := make([]*ir.Function, len(p.Diff.DroppedFunctions))
	copy(droppedFunctions, p.Diff.DroppedFunctions)
	sort.Slice(droppedFunctions, func(i, j int) bool {
		return getFullObjectName(droppedFunctions[i].Schema, droppedFunctions[i].Name) <
			getFullObjectName(droppedFunctions[j].Schema, droppedFunctions[j].Name)
	})
	for _, function := range droppedFunctions {
		fmt.Fprintf(summary, "  - %s.%s\n", function.Schema, function.Name)
	}
}

// writeProcedureChanges writes procedure changes
func (p *Plan) writeProcedureChanges(summary *strings.Builder) {
	// Sort added procedures
	addedProcedures := make([]*ir.Procedure, len(p.Diff.AddedProcedures))
	copy(addedProcedures, p.Diff.AddedProcedures)
	sort.Slice(addedProcedures, func(i, j int) bool {
		return getFullObjectName(addedProcedures[i].Schema, addedProcedures[i].Name) <
			getFullObjectName(addedProcedures[j].Schema, addedProcedures[j].Name)
	})
	for _, procedure := range addedProcedures {
		fmt.Fprintf(summary, "  + %s.%s\n", procedure.Schema, procedure.Name)
	}

	// Sort modified procedures
	modifiedProcedures := make([]*diff.ProcedureDiff, len(p.Diff.ModifiedProcedures))
	copy(modifiedProcedures, p.Diff.ModifiedProcedures)
	sort.Slice(modifiedProcedures, func(i, j int) bool {
		return getFullObjectName(modifiedProcedures[i].New.Schema, modifiedProcedures[i].New.Name) <
			getFullObjectName(modifiedProcedures[j].New.Schema, modifiedProcedures[j].New.Name)
	})
	for _, procedureDiff := range modifiedProcedures {
		fmt.Fprintf(summary, "  ~ %s.%s\n", procedureDiff.New.Schema, procedureDiff.New.Name)
	}

	// Sort dropped procedures
	droppedProcedures := make([]*ir.Procedure, len(p.Diff.DroppedProcedures))
	copy(droppedProcedures, p.Diff.DroppedProcedures)
	sort.Slice(droppedProcedures, func(i, j int) bool {
		return getFullObjectName(droppedProcedures[i].Schema, droppedProcedures[i].Name) <
			getFullObjectName(droppedProcedures[j].Schema, droppedProcedures[j].Name)
	})
	for _, procedure := range droppedProcedures {
		fmt.Fprintf(summary, "  - %s.%s\n", procedure.Schema, procedure.Name)
	}
}

// writeSequenceChanges writes sequence changes (placeholder)
func (p *Plan) writeSequenceChanges(summary *strings.Builder) {
	// TODO: Implement when sequence support is added
}

// writeTypeChanges writes type changes
func (p *Plan) writeTypeChanges(summary *strings.Builder) {
	// Sort added types
	addedTypes := make([]*ir.Type, len(p.Diff.AddedTypes))
	copy(addedTypes, p.Diff.AddedTypes)
	sort.Slice(addedTypes, func(i, j int) bool {
		return getFullObjectName(addedTypes[i].Schema, addedTypes[i].Name) <
			getFullObjectName(addedTypes[j].Schema, addedTypes[j].Name)
	})
	for _, typeObj := range addedTypes {
		fmt.Fprintf(summary, "  + %s.%s\n", typeObj.Schema, typeObj.Name)
	}

	// Sort modified types
	modifiedTypes := make([]*diff.TypeDiff, len(p.Diff.ModifiedTypes))
	copy(modifiedTypes, p.Diff.ModifiedTypes)
	sort.Slice(modifiedTypes, func(i, j int) bool {
		return getFullObjectName(modifiedTypes[i].New.Schema, modifiedTypes[i].New.Name) <
			getFullObjectName(modifiedTypes[j].New.Schema, modifiedTypes[j].New.Name)
	})
	for _, typeDiff := range modifiedTypes {
		fmt.Fprintf(summary, "  ~ %s.%s\n", typeDiff.New.Schema, typeDiff.New.Name)
	}

	// Sort dropped types
	droppedTypes := make([]*ir.Type, len(p.Diff.DroppedTypes))
	copy(droppedTypes, p.Diff.DroppedTypes)
	sort.Slice(droppedTypes, func(i, j int) bool {
		return getFullObjectName(droppedTypes[i].Schema, droppedTypes[i].Name) <
			getFullObjectName(droppedTypes[j].Schema, droppedTypes[j].Name)
	})
	for _, typeObj := range droppedTypes {
		fmt.Fprintf(summary, "  - %s.%s\n", typeObj.Schema, typeObj.Name)
	}
}

// writeExtensionChanges writes extension changes
func (p *Plan) writeExtensionChanges(summary *strings.Builder) {
	// Sort added extensions
	addedExtensions := make([]*ir.Extension, len(p.Diff.AddedExtensions))
	copy(addedExtensions, p.Diff.AddedExtensions)
	sort.Slice(addedExtensions, func(i, j int) bool {
		return addedExtensions[i].Name < addedExtensions[j].Name
	})
	for _, ext := range addedExtensions {
		fmt.Fprintf(summary, "  + %s\n", ext.Name)
	}

	// Sort dropped extensions
	droppedExtensions := make([]*ir.Extension, len(p.Diff.DroppedExtensions))
	copy(droppedExtensions, p.Diff.DroppedExtensions)
	sort.Slice(droppedExtensions, func(i, j int) bool {
		return droppedExtensions[i].Name < droppedExtensions[j].Name
	})
	for _, ext := range droppedExtensions {
		fmt.Fprintf(summary, "  - %s\n", ext.Name)
	}
}

// convertToStructuredJSON converts the DDLDiff to a structured JSON format
func (p *Plan) convertToStructuredJSON() *PlanJSON {
	planJSON := &PlanJSON{
		Version:         "1.0",
		PgschemaVersion: pgschemaVersion,
		CreatedAt:       p.CreatedAt.Truncate(time.Second),
		Summary: PlanSummary{
			ByType: make(map[string]TypeSummary),
		},
		ObjectChanges: []ObjectChange{},
	}

	// Process added objects in dependency order
	p.addObjectChanges(planJSON, "schema", p.Diff.AddedSchemas, nil, []string{"create"})
	p.addObjectChanges(planJSON, "extension", p.Diff.AddedExtensions, nil, []string{"create"})
	p.addObjectChanges(planJSON, "type", p.Diff.AddedTypes, nil, []string{"create"})
	p.addObjectChanges(planJSON, "function", p.Diff.AddedFunctions, nil, []string{"create"})
	p.addObjectChanges(planJSON, "procedure", p.Diff.AddedProcedures, nil, []string{"create"})
	// Sequences placeholder
	p.addObjectChanges(planJSON, "table", p.Diff.AddedTables, nil, []string{"create"})
	p.addObjectChanges(planJSON, "view", p.Diff.AddedViews, nil, []string{"create"})
	// Indexes, triggers, and policies are handled as part of table modifications

	// Process dropped objects in reverse dependency order
	p.addObjectChanges(planJSON, "function", nil, p.Diff.DroppedFunctions, []string{"delete"})
	p.addObjectChanges(planJSON, "procedure", nil, p.Diff.DroppedProcedures, []string{"delete"})
	p.addObjectChanges(planJSON, "view", nil, p.Diff.DroppedViews, []string{"delete"})
	p.addObjectChanges(planJSON, "table", nil, p.Diff.DroppedTables, []string{"delete"})
	// Sequences placeholder
	p.addObjectChanges(planJSON, "type", nil, p.Diff.DroppedTypes, []string{"delete"})
	p.addObjectChanges(planJSON, "extension", nil, p.Diff.DroppedExtensions, []string{"delete"})
	p.addObjectChanges(planJSON, "schema", nil, p.Diff.DroppedSchemas, []string{"delete"})
	// Indexes, triggers, and policies are handled as part of table modifications

	// Process modified objects
	p.addModifiedObjectChanges(planJSON, "schema", p.Diff.ModifiedSchemas)
	p.addModifiedObjectChanges(planJSON, "type", p.Diff.ModifiedTypes)
	p.addModifiedObjectChanges(planJSON, "function", p.Diff.ModifiedFunctions)
	p.addModifiedObjectChanges(planJSON, "procedure", p.Diff.ModifiedProcedures)
	p.addModifiedObjectChanges(planJSON, "view", p.Diff.ModifiedViews)
	// Modified triggers and policies are handled as part of table modifications

	// Process modified tables (more complex)
	for _, tableDiff := range p.Diff.ModifiedTables {
		p.addTableChanges(planJSON, tableDiff)
	}

	// Sort all object changes alphabetically by address for JSON output
	sort.Slice(planJSON.ObjectChanges, func(i, j int) bool {
		return planJSON.ObjectChanges[i].Address < planJSON.ObjectChanges[j].Address
	})

	// Calculate summary
	p.calculateSummary(planJSON)

	return planJSON
}

// addObjectChanges adds object changes to the plan JSON
func (p *Plan) addObjectChanges(planJSON *PlanJSON, objType string, addedObjects, droppedObjects any, actions []string) {
	var objects []any

	if addedObjects != nil {
		switch v := addedObjects.(type) {
		case []*ir.Schema:
			for _, obj := range v {
				objects = append(objects, obj)
			}
		case []*ir.Table:
			for _, obj := range v {
				objects = append(objects, obj)
			}
		case []*ir.View:
			for _, obj := range v {
				objects = append(objects, obj)
			}
		case []*ir.Function:
			for _, obj := range v {
				objects = append(objects, obj)
			}
		case []*ir.Procedure:
			for _, obj := range v {
				objects = append(objects, obj)
			}
		case []*ir.Extension:
			for _, obj := range v {
				objects = append(objects, obj)
			}
		case []*ir.Index:
			for _, obj := range v {
				objects = append(objects, obj)
			}
		case []*ir.Trigger:
			for _, obj := range v {
				objects = append(objects, obj)
			}
		case []*ir.Type:
			for _, obj := range v {
				objects = append(objects, obj)
			}
		}
	}

	if droppedObjects != nil {
		switch v := droppedObjects.(type) {
		case []*ir.Schema:
			for _, obj := range v {
				objects = append(objects, obj)
			}
		case []*ir.Table:
			for _, obj := range v {
				objects = append(objects, obj)
			}
		case []*ir.View:
			for _, obj := range v {
				objects = append(objects, obj)
			}
		case []*ir.Function:
			for _, obj := range v {
				objects = append(objects, obj)
			}
		case []*ir.Procedure:
			for _, obj := range v {
				objects = append(objects, obj)
			}
		case []*ir.Extension:
			for _, obj := range v {
				objects = append(objects, obj)
			}
		case []*ir.Index:
			for _, obj := range v {
				objects = append(objects, obj)
			}
		case []*ir.Trigger:
			for _, obj := range v {
				objects = append(objects, obj)
			}
		case []*ir.Type:
			for _, obj := range v {
				objects = append(objects, obj)
			}
		}
	}

	for _, obj := range objects {
		change := p.createObjectChange(objType, obj, actions)
		if change != nil {
			planJSON.ObjectChanges = append(planJSON.ObjectChanges, *change)
		}
	}
}

// createObjectChange creates an ObjectChange from a database object
func (p *Plan) createObjectChange(objType string, obj any, actions []string) *ObjectChange {
	change := &ObjectChange{
		Mode:   objType,
		Type:   objType,
		Change: Change{Actions: actions},
	}

	// Set before/after based on action
	switch actions[0] {
	case "create":
		change.Change.Before = nil
		change.Change.After = p.objectToMap(obj)
	case "delete":
		change.Change.Before = p.objectToMap(obj)
		change.Change.After = nil
	}

	// Set address and other fields based on object type
	switch v := obj.(type) {
	case *ir.Schema:
		change.Address = v.Name
		change.Name = v.Name
		change.Schema = v.Name
	case *ir.Table:
		change.Address = fmt.Sprintf("%s.%s", v.Schema, v.Name)
		change.Name = v.Name
		change.Schema = v.Schema
	case *ir.View:
		change.Address = fmt.Sprintf("%s.%s", v.Schema, v.Name)
		change.Name = v.Name
		change.Schema = v.Schema
	case *ir.Function:
		change.Address = fmt.Sprintf("%s.%s", v.Schema, v.Name)
		change.Name = v.Name
		change.Schema = v.Schema
	case *ir.Procedure:
		change.Address = fmt.Sprintf("%s.%s", v.Schema, v.Name)
		change.Name = v.Name
		change.Schema = v.Schema
	case *ir.Extension:
		change.Address = v.Name
		change.Name = v.Name
	case *ir.Index:
		change.Address = fmt.Sprintf("%s.%s.%s", v.Schema, v.Table, v.Name)
		change.Name = v.Name
		change.Schema = v.Schema
		change.Table = v.Table
	case *ir.Trigger:
		change.Address = fmt.Sprintf("%s.%s.%s", v.Schema, v.Table, v.Name)
		change.Name = v.Name
		change.Schema = v.Schema
		change.Table = v.Table
	case *ir.Type:
		change.Address = fmt.Sprintf("%s.%s", v.Schema, v.Name)
		change.Name = v.Name
		change.Schema = v.Schema
	default:
		return nil
	}

	return change
}

// objectToMap converts a database object to a map for JSON serialization
func (p *Plan) objectToMap(obj any) map[string]any {
	result := make(map[string]any)

	switch v := obj.(type) {
	case *ir.Schema:
		result["name"] = v.Name
		if v.Owner != "" {
			result["owner"] = v.Owner
		}
	case *ir.Table:
		result["name"] = v.Name
		result["schema"] = v.Schema
		result["type"] = v.Type
		if len(v.Columns) > 0 {
			result["columns"] = v.Columns
		}
		if len(v.Constraints) > 0 {
			result["constraints"] = v.Constraints
		}
	case *ir.View:
		result["name"] = v.Name
		result["schema"] = v.Schema
		result["definition"] = v.Definition
	case *ir.Function:
		result["name"] = v.Name
		result["schema"] = v.Schema
		result["arguments"] = v.Arguments
		result["return_type"] = v.ReturnType
		result["language"] = v.Language
	case *ir.Procedure:
		result["name"] = v.Name
		result["schema"] = v.Schema
		result["arguments"] = v.Arguments
		result["language"] = v.Language
	case *ir.Extension:
		result["name"] = v.Name
		if v.Schema != "" {
			result["schema"] = v.Schema
		}
	case *ir.Index:
		result["name"] = v.Name
		result["schema"] = v.Schema
		result["table"] = v.Table
		result["columns"] = v.Columns
		result["is_unique"] = v.Type == ir.IndexTypeUnique
		result["is_primary"] = v.Type == ir.IndexTypePrimary
	case *ir.Trigger:
		result["name"] = v.Name
		result["schema"] = v.Schema
		result["table"] = v.Table
		result["timing"] = v.Timing
		result["events"] = v.Events
		result["function"] = v.Function
	case *ir.Type:
		result["name"] = v.Name
		result["schema"] = v.Schema
		result["kind"] = v.Kind
		if v.Kind == ir.TypeKindEnum {
			result["enum_values"] = v.EnumValues
		}
	case *ir.Column:
		result["name"] = v.Name
		result["position"] = v.Position
		result["data_type"] = v.DataType
		result["is_nullable"] = v.IsNullable
		if v.DefaultValue != nil {
			result["default_value"] = *v.DefaultValue
		}
		if v.MaxLength != nil {
			result["max_length"] = *v.MaxLength
		}
		result["is_identity"] = v.Identity != nil
		if v.Identity != nil && v.Identity.Generation != "" {
			result["identity_generation"] = v.Identity.Generation
		}
	case *ir.RLSPolicy:
		result["name"] = v.Name
		result["schema"] = v.Schema
		result["table"] = v.Table
		result["command"] = v.Command
		result["permissive"] = v.Permissive
		if v.Using != "" {
			result["using"] = v.Using
		}
		if v.WithCheck != "" {
			result["with_check"] = v.WithCheck
		}
		if len(v.Roles) > 0 {
			result["roles"] = v.Roles
		}
	}

	return result
}

// addModifiedObjectChanges adds modified object changes
func (p *Plan) addModifiedObjectChanges(planJSON *PlanJSON, objType string, modifiedObjects any) {
	switch v := modifiedObjects.(type) {
	case []*diff.SchemaDiff:
		for _, diff := range v {
			change := ObjectChange{
				Address: diff.New.Name,
				Mode:    objType,
				Type:    objType,
				Name:    diff.New.Name,
				Schema:  diff.New.Name,
				Change: Change{
					Actions: []string{"update"},
					Before:  p.objectToMap(diff.Old),
					After:   p.objectToMap(diff.New),
				},
			}
			planJSON.ObjectChanges = append(planJSON.ObjectChanges, change)
		}
	case []*diff.ViewDiff:
		for _, diff := range v {
			change := ObjectChange{
				Address: fmt.Sprintf("%s.%s", diff.New.Schema, diff.New.Name),
				Mode:    objType,
				Type:    objType,
				Name:    diff.New.Name,
				Schema:  diff.New.Schema,
				Change: Change{
					Actions: []string{"update"},
					Before:  p.objectToMap(diff.Old),
					After:   p.objectToMap(diff.New),
				},
			}
			planJSON.ObjectChanges = append(planJSON.ObjectChanges, change)
		}
	case []*diff.FunctionDiff:
		for _, diff := range v {
			change := ObjectChange{
				Address: fmt.Sprintf("%s.%s", diff.New.Schema, diff.New.Name),
				Mode:    objType,
				Type:    objType,
				Name:    diff.New.Name,
				Schema:  diff.New.Schema,
				Change: Change{
					Actions: []string{"update"},
					Before:  p.objectToMap(diff.Old),
					After:   p.objectToMap(diff.New),
				},
			}
			planJSON.ObjectChanges = append(planJSON.ObjectChanges, change)
		}
	case []*diff.ProcedureDiff:
		for _, diff := range v {
			change := ObjectChange{
				Address: fmt.Sprintf("%s.%s", diff.New.Schema, diff.New.Name),
				Mode:    objType,
				Type:    objType,
				Name:    diff.New.Name,
				Schema:  diff.New.Schema,
				Change: Change{
					Actions: []string{"update"},
					Before:  p.objectToMap(diff.Old),
					After:   p.objectToMap(diff.New),
				},
			}
			planJSON.ObjectChanges = append(planJSON.ObjectChanges, change)
		}
	case []*diff.TriggerDiff:
		for _, diff := range v {
			change := ObjectChange{
				Address: fmt.Sprintf("%s.%s.%s", diff.New.Schema, diff.New.Table, diff.New.Name),
				Mode:    objType,
				Type:    objType,
				Name:    diff.New.Name,
				Schema:  diff.New.Schema,
				Table:   diff.New.Table,
				Change: Change{
					Actions: []string{"update"},
					Before:  p.objectToMap(diff.Old),
					After:   p.objectToMap(diff.New),
				},
			}
			planJSON.ObjectChanges = append(planJSON.ObjectChanges, change)
		}
	case []*diff.TypeDiff:
		for _, diff := range v {
			change := ObjectChange{
				Address: fmt.Sprintf("%s.%s", diff.New.Schema, diff.New.Name),
				Mode:    objType,
				Type:    objType,
				Name:    diff.New.Name,
				Schema:  diff.New.Schema,
				Change: Change{
					Actions: []string{"update"},
					Before:  p.objectToMap(diff.Old),
					After:   p.objectToMap(diff.New),
				},
			}
			planJSON.ObjectChanges = append(planJSON.ObjectChanges, change)
		}
	}
}

// addTableChanges adds table-level changes with column, constraint, index, trigger, and policy details
func (p *Plan) addTableChanges(planJSON *PlanJSON, tableDiff *diff.TableDiff) {
	// Add table-level change if there are modifications
	if len(tableDiff.AddedColumns) > 0 || len(tableDiff.DroppedColumns) > 0 ||
		len(tableDiff.ModifiedColumns) > 0 || len(tableDiff.AddedConstraints) > 0 ||
		len(tableDiff.DroppedConstraints) > 0 || len(tableDiff.AddedIndexes) > 0 ||
		len(tableDiff.DroppedIndexes) > 0 || len(tableDiff.AddedTriggers) > 0 ||
		len(tableDiff.DroppedTriggers) > 0 || len(tableDiff.ModifiedTriggers) > 0 ||
		len(tableDiff.AddedPolicies) > 0 || len(tableDiff.DroppedPolicies) > 0 ||
		len(tableDiff.ModifiedPolicies) > 0 || len(tableDiff.RLSChanges) > 0 {

		change := ObjectChange{
			Address: fmt.Sprintf("%s.%s", tableDiff.Table.Schema, tableDiff.Table.Name),
			Mode:    "table",
			Type:    "table",
			Name:    tableDiff.Table.Name,
			Schema:  tableDiff.Table.Schema,
			Change: Change{
				Actions: []string{"update"},
				Before:  map[string]any{},
				After:   p.objectToMap(tableDiff.Table),
			},
			Metadata: map[string]any{
				"added_columns":       len(tableDiff.AddedColumns),
				"dropped_columns":     len(tableDiff.DroppedColumns),
				"modified_columns":    len(tableDiff.ModifiedColumns),
				"added_constraints":   len(tableDiff.AddedConstraints),
				"dropped_constraints": len(tableDiff.DroppedConstraints),
				"added_indexes":       len(tableDiff.AddedIndexes),
				"dropped_indexes":     len(tableDiff.DroppedIndexes),
				"added_triggers":      len(tableDiff.AddedTriggers),
				"dropped_triggers":    len(tableDiff.DroppedTriggers),
				"modified_triggers":   len(tableDiff.ModifiedTriggers),
				"added_policies":      len(tableDiff.AddedPolicies),
				"dropped_policies":    len(tableDiff.DroppedPolicies),
				"modified_policies":   len(tableDiff.ModifiedPolicies),
				"rls_changes":         len(tableDiff.RLSChanges),
			},
		}
		planJSON.ObjectChanges = append(planJSON.ObjectChanges, change)
	}

	// Add individual column changes
	for _, column := range tableDiff.AddedColumns {
		change := ObjectChange{
			Address: fmt.Sprintf("%s.%s.%s", tableDiff.Table.Schema, tableDiff.Table.Name, column.Name),
			Mode:    "column",
			Type:    "column",
			Name:    column.Name,
			Schema:  tableDiff.Table.Schema,
			Table:   tableDiff.Table.Name,
			Change: Change{
				Actions: []string{"create"},
				Before:  nil,
				After:   p.objectToMap(column),
			},
		}
		planJSON.ObjectChanges = append(planJSON.ObjectChanges, change)
	}

	for _, column := range tableDiff.DroppedColumns {
		change := ObjectChange{
			Address: fmt.Sprintf("%s.%s.%s", tableDiff.Table.Schema, tableDiff.Table.Name, column.Name),
			Mode:    "column",
			Type:    "column",
			Name:    column.Name,
			Schema:  tableDiff.Table.Schema,
			Table:   tableDiff.Table.Name,
			Change: Change{
				Actions: []string{"delete"},
				Before:  p.objectToMap(column),
				After:   nil,
			},
		}
		planJSON.ObjectChanges = append(planJSON.ObjectChanges, change)
	}

	for _, columnDiff := range tableDiff.ModifiedColumns {
		change := ObjectChange{
			Address: fmt.Sprintf("%s.%s.%s", tableDiff.Table.Schema, tableDiff.Table.Name, columnDiff.New.Name),
			Mode:    "column",
			Type:    "column",
			Name:    columnDiff.New.Name,
			Schema:  tableDiff.Table.Schema,
			Table:   tableDiff.Table.Name,
			Change: Change{
				Actions: []string{"update"},
				Before:  p.objectToMap(columnDiff.Old),
				After:   p.objectToMap(columnDiff.New),
			},
		}
		planJSON.ObjectChanges = append(planJSON.ObjectChanges, change)
	}

	// Add individual index changes
	for _, index := range tableDiff.AddedIndexes {
		change := ObjectChange{
			Address: fmt.Sprintf("%s.%s.%s", tableDiff.Table.Schema, tableDiff.Table.Name, index.Name),
			Mode:    "index",
			Type:    "index",
			Name:    index.Name,
			Schema:  tableDiff.Table.Schema,
			Table:   tableDiff.Table.Name,
			Change: Change{
				Actions: []string{"create"},
				Before:  nil,
				After:   p.objectToMap(index),
			},
		}
		planJSON.ObjectChanges = append(planJSON.ObjectChanges, change)
	}

	for _, index := range tableDiff.DroppedIndexes {
		change := ObjectChange{
			Address: fmt.Sprintf("%s.%s.%s", tableDiff.Table.Schema, tableDiff.Table.Name, index.Name),
			Mode:    "index",
			Type:    "index",
			Name:    index.Name,
			Schema:  tableDiff.Table.Schema,
			Table:   tableDiff.Table.Name,
			Change: Change{
				Actions: []string{"delete"},
				Before:  p.objectToMap(index),
				After:   nil,
			},
		}
		planJSON.ObjectChanges = append(planJSON.ObjectChanges, change)
	}

	// Add individual trigger changes
	for _, trigger := range tableDiff.AddedTriggers {
		change := ObjectChange{
			Address: fmt.Sprintf("%s.%s.%s", tableDiff.Table.Schema, tableDiff.Table.Name, trigger.Name),
			Mode:    "trigger",
			Type:    "trigger",
			Name:    trigger.Name,
			Schema:  tableDiff.Table.Schema,
			Table:   tableDiff.Table.Name,
			Change: Change{
				Actions: []string{"create"},
				Before:  nil,
				After:   p.objectToMap(trigger),
			},
		}
		planJSON.ObjectChanges = append(planJSON.ObjectChanges, change)
	}

	for _, trigger := range tableDiff.DroppedTriggers {
		change := ObjectChange{
			Address: fmt.Sprintf("%s.%s.%s", tableDiff.Table.Schema, tableDiff.Table.Name, trigger.Name),
			Mode:    "trigger",
			Type:    "trigger",
			Name:    trigger.Name,
			Schema:  tableDiff.Table.Schema,
			Table:   tableDiff.Table.Name,
			Change: Change{
				Actions: []string{"delete"},
				Before:  p.objectToMap(trigger),
				After:   nil,
			},
		}
		planJSON.ObjectChanges = append(planJSON.ObjectChanges, change)
	}

	for _, triggerDiff := range tableDiff.ModifiedTriggers {
		change := ObjectChange{
			Address: fmt.Sprintf("%s.%s.%s", tableDiff.Table.Schema, tableDiff.Table.Name, triggerDiff.New.Name),
			Mode:    "trigger",
			Type:    "trigger",
			Name:    triggerDiff.New.Name,
			Schema:  tableDiff.Table.Schema,
			Table:   tableDiff.Table.Name,
			Change: Change{
				Actions: []string{"update"},
				Before:  p.objectToMap(triggerDiff.Old),
				After:   p.objectToMap(triggerDiff.New),
			},
		}
		planJSON.ObjectChanges = append(planJSON.ObjectChanges, change)
	}

	// Add individual policy changes
	for _, policy := range tableDiff.AddedPolicies {
		change := ObjectChange{
			Address: fmt.Sprintf("%s.%s.%s", tableDiff.Table.Schema, tableDiff.Table.Name, policy.Name),
			Mode:    "policy",
			Type:    "policy",
			Name:    policy.Name,
			Schema:  tableDiff.Table.Schema,
			Table:   tableDiff.Table.Name,
			Change: Change{
				Actions: []string{"create"},
				Before:  nil,
				After:   p.objectToMap(policy),
			},
		}
		planJSON.ObjectChanges = append(planJSON.ObjectChanges, change)
	}

	for _, policy := range tableDiff.DroppedPolicies {
		change := ObjectChange{
			Address: fmt.Sprintf("%s.%s.%s", tableDiff.Table.Schema, tableDiff.Table.Name, policy.Name),
			Mode:    "policy",
			Type:    "policy",
			Name:    policy.Name,
			Schema:  tableDiff.Table.Schema,
			Table:   tableDiff.Table.Name,
			Change: Change{
				Actions: []string{"delete"},
				Before:  p.objectToMap(policy),
				After:   nil,
			},
		}
		planJSON.ObjectChanges = append(planJSON.ObjectChanges, change)
	}

	for _, policyDiff := range tableDiff.ModifiedPolicies {
		change := ObjectChange{
			Address: fmt.Sprintf("%s.%s.%s", tableDiff.Table.Schema, tableDiff.Table.Name, policyDiff.New.Name),
			Mode:    "policy",
			Type:    "policy",
			Name:    policyDiff.New.Name,
			Schema:  tableDiff.Table.Schema,
			Table:   tableDiff.Table.Name,
			Change: Change{
				Actions: []string{"update"},
				Before:  p.objectToMap(policyDiff.Old),
				After:   p.objectToMap(policyDiff.New),
			},
		}
		planJSON.ObjectChanges = append(planJSON.ObjectChanges, change)
	}

	// Add RLS changes
	for _, rlsChange := range tableDiff.RLSChanges {
		change := ObjectChange{
			Address: fmt.Sprintf("%s.%s", tableDiff.Table.Schema, tableDiff.Table.Name),
			Mode:    "rls",
			Type:    "rls",
			Name:    "row_level_security",
			Schema:  tableDiff.Table.Schema,
			Table:   tableDiff.Table.Name,
			Change: Change{
				Actions: []string{"update"},
				Before:  map[string]any{"enabled": !rlsChange.Enabled},
				After:   map[string]any{"enabled": rlsChange.Enabled},
			},
		}
		planJSON.ObjectChanges = append(planJSON.ObjectChanges, change)
	}
}

// calculateSummary calculates the summary statistics
func (p *Plan) calculateSummary(planJSON *PlanJSON) {
	typeStats := make(map[string]TypeSummary)

	for _, change := range planJSON.ObjectChanges {
		stats := typeStats[change.Type]

		if len(change.Change.Actions) > 0 {
			switch change.Change.Actions[0] {
			case "create":
				stats.Add++
				planJSON.Summary.Add++
			case "update":
				stats.Change++
				planJSON.Summary.Change++
			case "delete":
				stats.Destroy++
				planJSON.Summary.Destroy++
			}
		}

		typeStats[change.Type] = stats
	}

	planJSON.Summary.ByType = typeStats
	planJSON.Summary.Total = planJSON.Summary.Add + planJSON.Summary.Change + planJSON.Summary.Destroy
}
