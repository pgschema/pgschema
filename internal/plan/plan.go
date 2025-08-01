package plan

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pgschema/pgschema/internal/color"
	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/ir"
	"github.com/pgschema/pgschema/internal/version"
)

// Plan represents the migration plan between two DDL states
type Plan struct {
	// The underlying diff data
	Diff *diff.DDLDiff `json:"diff"`

	// The target schema for the migration
	TargetSchema string `json:"target_schema"`

	// Plan metadata
	CreatedAt time.Time `json:"created_at"`

	// EnableTransaction indicates whether DDL can run in a transaction (false for CREATE INDEX CONCURRENTLY)
	EnableTransaction bool `json:"enable_transaction"`
}

// ObjectChange represents a single change to a database object
type ObjectChange struct {
	Address  string         `json:"address"`
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
	Transaction     bool           `json:"transaction"`
	Summary         PlanSummary    `json:"summary"`
	ObjectChanges   []ObjectChange `json:"object_changes"`
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
		CreatedAt:    time.Now(),
	}
	// Enable transaction unless non-transactional DDL is present
	plan.EnableTransaction = !plan.hasNonTransactionalDDL()
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

	// Get JSON representation first for consistency
	planJSON := p.convertToStructuredJSON()

	if planJSON.Summary.Total == 0 {
		summary.WriteString("No changes detected.\n")
		return summary.String()
	}

	// Write header with overall summary (colored like Terraform)
	summary.WriteString(c.FormatPlanHeader(planJSON.Summary.Add, planJSON.Summary.Change, planJSON.Summary.Destroy) + "\n\n")

	// Write summary by type with colors
	summary.WriteString(c.Bold("Summary by type:") + "\n")
	for _, objType := range getObjectOrder() {
		objTypeStr := string(objType)
		if typeSummary, exists := planJSON.Summary.ByType[objTypeStr]; exists && (typeSummary.Add > 0 || typeSummary.Change > 0 || typeSummary.Destroy > 0) {
			line := c.FormatSummaryLine(objTypeStr, typeSummary.Add, typeSummary.Change, typeSummary.Destroy)
			summary.WriteString(line + "\n")
		}
	}
	summary.WriteString("\n")

	// Detailed changes by type with symbols
	for _, objType := range getObjectOrder() {
		objTypeStr := string(objType)
		if typeSummary, exists := planJSON.Summary.ByType[objTypeStr]; exists && (typeSummary.Add > 0 || typeSummary.Change > 0 || typeSummary.Destroy > 0) {
			// Capitalize first letter for display
			displayName := strings.ToUpper(objTypeStr[:1]) + objTypeStr[1:]
			p.writeDetailedChangesFromJSON(&summary, displayName, objTypeStr, planJSON.ObjectChanges, c)
		}
	}

	// Add transaction mode information
	if planJSON.Summary.Total > 0 {
		if planJSON.Transaction {
			summary.WriteString("Transaction: true\n\n")
		} else {
			summary.WriteString("Transaction: false\n\n")
		}
	}

	// Add DDL section if there are changes
	if planJSON.Summary.Total > 0 {
		summary.WriteString(c.Bold("DDL to be executed:") + "\n")
		summary.WriteString(strings.Repeat("-", 50) + "\n\n")
		migrationSQL := diff.GenerateMigrationSQL(p.Diff, p.TargetSchema)
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
	planJSON := p.convertToStructuredJSON()

	data, err := json.MarshalIndent(planJSON, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal plan to JSON: %w", err)
	}
	return string(data), nil
}

// ToSQL returns only the SQL statements without any additional formatting
func (p *Plan) ToSQL() string {
	// Get JSON representation first
	planJSON := p.convertToStructuredJSON()

	// Check if there are any changes
	if planJSON.Summary.Total == 0 {
		return ""
	}

	// Generate migration SQL
	return diff.GenerateMigrationSQL(p.Diff, p.TargetSchema)
}

// ========== PRIVATE METHODS ==========

// writeDetailedChangesFromJSON writes detailed changes using JSON representation for consistency
func (p *Plan) writeDetailedChangesFromJSON(summary *strings.Builder, displayName, objType string, objectChanges []ObjectChange, c *color.Color) {
	fmt.Fprintf(summary, "%s:\n", c.Bold(displayName))

	// Filter and sort changes for this object type
	var changes []ObjectChange
	for _, change := range objectChanges {
		if change.Type == objType {
			changes = append(changes, change)
		}
	}

	// Sort changes by address for consistent output
	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Address < changes[j].Address
	})

	// Write changes with appropriate symbols
	for _, change := range changes {
		if len(change.Change.Actions) > 0 {
			var symbol string
			switch change.Change.Actions[0] {
			case "create":
				symbol = c.PlanSymbol("add")
			case "update":
				symbol = c.PlanSymbol("change")
			case "delete":
				symbol = c.PlanSymbol("destroy")
			default:
				symbol = c.PlanSymbol("change")
			}

			// Format address for display - use full address for all types
			displayAddress := change.Address

			fmt.Fprintf(summary, "  %s %s\n", symbol, displayAddress)
		}
	}

	summary.WriteString("\n")
}

// convertToStructuredJSON converts the DDLDiff to a structured JSON format
func (p *Plan) convertToStructuredJSON() *PlanJSON {
	planJSON := &PlanJSON{
		Version:         version.PlanFormat(),
		PgschemaVersion: version.App(),
		CreatedAt:       p.CreatedAt.Truncate(time.Second),
		Transaction:     p.EnableTransaction,
		Summary: PlanSummary{
			ByType: make(map[string]TypeSummary),
		},
		ObjectChanges: []ObjectChange{},
	}

	// Process added objects in dependency order
	p.addObjectChanges(planJSON, string(ObjectTypeSchema), p.Diff.AddedSchemas, nil, []string{"create"})
	p.addObjectChanges(planJSON, string(ObjectTypeType), p.Diff.AddedTypes, nil, []string{"create"})
	p.addObjectChanges(planJSON, string(ObjectTypeFunction), p.Diff.AddedFunctions, nil, []string{"create"})
	p.addObjectChanges(planJSON, string(ObjectTypeProcedure), p.Diff.AddedProcedures, nil, []string{"create"})
	// Sequences placeholder
	p.addObjectChanges(planJSON, string(ObjectTypeTable), p.Diff.AddedTables, nil, []string{"create"})
	p.addObjectChanges(planJSON, string(ObjectTypeView), p.Diff.AddedViews, nil, []string{"create"})
	// Indexes, triggers, and policies are handled as part of table modifications

	// Process dropped objects in reverse dependency order
	p.addObjectChanges(planJSON, string(ObjectTypeFunction), nil, p.Diff.DroppedFunctions, []string{"delete"})
	p.addObjectChanges(planJSON, string(ObjectTypeProcedure), nil, p.Diff.DroppedProcedures, []string{"delete"})
	p.addObjectChanges(planJSON, string(ObjectTypeView), nil, p.Diff.DroppedViews, []string{"delete"})
	p.addObjectChanges(planJSON, string(ObjectTypeTable), nil, p.Diff.DroppedTables, []string{"delete"})
	// Sequences placeholder
	p.addObjectChanges(planJSON, string(ObjectTypeType), nil, p.Diff.DroppedTypes, []string{"delete"})
	p.addObjectChanges(planJSON, string(ObjectTypeSchema), nil, p.Diff.DroppedSchemas, []string{"delete"})
	// Indexes, triggers, and policies are handled as part of table modifications

	// Process modified objects
	p.addModifiedObjectChanges(planJSON, string(ObjectTypeSchema), p.Diff.ModifiedSchemas)
	p.addModifiedObjectChanges(planJSON, string(ObjectTypeType), p.Diff.ModifiedTypes)
	p.addModifiedObjectChanges(planJSON, string(ObjectTypeFunction), p.Diff.ModifiedFunctions)
	p.addModifiedObjectChanges(planJSON, string(ObjectTypeProcedure), p.Diff.ModifiedProcedures)
	p.addModifiedObjectChanges(planJSON, string(ObjectTypeView), p.Diff.ModifiedViews)
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
			Type:    string(ObjectTypeTable),
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
			Type:    string(ObjectTypeColumn),
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
			Type:    string(ObjectTypeColumn),
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
			Type:    string(ObjectTypeColumn),
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
			Type:    string(ObjectTypeIndex),
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
			Type:    string(ObjectTypeIndex),
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
			Type:    string(ObjectTypeTrigger),
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
			Type:    string(ObjectTypeTrigger),
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
			Type:    string(ObjectTypeTrigger),
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
			Type:    string(ObjectTypePolicy),
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
			Type:    string(ObjectTypePolicy),
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
			Type:    string(ObjectTypePolicy),
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
			Type:    string(ObjectTypeRLS),
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
// This matches the business logic in getTypeCountsDetailed() where indexes, triggers,
// and policies are co-located with tables and not counted separately in the summary
func (p *Plan) calculateSummary(planJSON *PlanJSON) {
	typeStats := make(map[string]TypeSummary)

	for _, change := range planJSON.ObjectChanges {
		// Skip sub-objects that are co-located with tables per business logic
		// Indexes, triggers, policies, columns, and RLS are not counted separately in the summary
		if change.Type == string(ObjectTypeIndex) || change.Type == string(ObjectTypeTrigger) ||
			change.Type == string(ObjectTypePolicy) || change.Type == string(ObjectTypeColumn) ||
			change.Type == string(ObjectTypeRLS) {
			continue
		}

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
