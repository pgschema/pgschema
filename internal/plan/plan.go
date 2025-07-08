package plan

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pgschema/pgschema/internal/diff"
	"github.com/pgschema/pgschema/internal/ir"
)

// Plan represents the migration plan between two DDL states
type Plan struct {
	// The underlying diff data
	Diff *diff.DDLDiff `json:"diff"`

	// Plan metadata
	CreatedAt time.Time `json:"created_at"`
}

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
	for _, objType := range []string{"schemas", "tables", "views", "sequences", "functions", "procedures", "types", "extensions", "indexes", "triggers"} {
		if counts, exists := typeCounts[objType]; exists && (counts.added > 0 || counts.modified > 0 || counts.dropped > 0) {
			summary.WriteString(fmt.Sprintf("  %s: %d to add, %d to modify, %d to drop\n", 
				objType, counts.added, counts.modified, counts.dropped))
		}
	}
	summary.WriteString("\n")

	// Detailed changes by type
	p.writeDetailedChanges(&summary, "Schemas", typeCounts["schemas"])
	p.writeDetailedChanges(&summary, "Tables", typeCounts["tables"])
	p.writeDetailedChanges(&summary, "Views", typeCounts["views"])
	p.writeDetailedChanges(&summary, "Sequences", typeCounts["sequences"])
	p.writeDetailedChanges(&summary, "Functions", typeCounts["functions"])
	p.writeDetailedChanges(&summary, "Procedures", typeCounts["procedures"])
	p.writeDetailedChanges(&summary, "Types", typeCounts["types"])
	p.writeDetailedChanges(&summary, "Extensions", typeCounts["extensions"])
	p.writeDetailedChanges(&summary, "Indexes", typeCounts["indexes"])
	p.writeDetailedChanges(&summary, "Triggers", typeCounts["triggers"])

	// Add DDL section if there are changes
	if totalChanges > 0 {
		summary.WriteString("DDL to be executed:\n")
		summary.WriteString(strings.Repeat("-", 50) + "\n")
		migrationSQL := p.Diff.GenerateMigrationSQL()
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

// typeCounts holds counts for each type of change
type typeCounts struct {
	added    int
	modified int
	dropped  int
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
	
	// Functions (including procedures)
	counts["functions"] = typeCounts{
		added:    len(p.Diff.AddedFunctions),
		modified: len(p.Diff.ModifiedFunctions),
		dropped:  len(p.Diff.DroppedFunctions),
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
	
	// Indexes
	counts["indexes"] = typeCounts{
		added:    len(p.Diff.AddedIndexes),
		modified: 0, // Indexes typically get dropped and recreated
		dropped:  len(p.Diff.DroppedIndexes),
	}
	
	// Triggers
	counts["triggers"] = typeCounts{
		added:    len(p.Diff.AddedTriggers),
		modified: len(p.Diff.ModifiedTriggers),
		dropped:  len(p.Diff.DroppedTriggers),
	}
	
	// Initialize empty counts for sequences and procedures
	counts["sequences"] = typeCounts{0, 0, 0}
	counts["procedures"] = typeCounts{0, 0, 0}
	
	return counts
}

// writeDetailedChanges writes detailed changes for a specific object type
func (p *Plan) writeDetailedChanges(summary *strings.Builder, typeName string, counts typeCounts) {
	if counts.added == 0 && counts.modified == 0 && counts.dropped == 0 {
		return
	}
	
	summary.WriteString(fmt.Sprintf("%s:\n", typeName))
	
	switch typeName {
	case "Schemas":
		p.writeSchemaChanges(summary)
	case "Tables":
		p.writeTableChanges(summary)
	case "Views":
		p.writeViewChanges(summary)
	case "Functions":
		p.writeFunctionChanges(summary)
	case "Types":
		p.writeTypeChanges(summary)
	case "Extensions":
		p.writeExtensionChanges(summary)
	case "Indexes":
		p.writeIndexChanges(summary)
	case "Triggers":
		p.writeTriggerChanges(summary)
	}
	
	summary.WriteString("\n")
}

// writeSchemaChanges writes schema changes
func (p *Plan) writeSchemaChanges(summary *strings.Builder) {
	for _, schema := range p.Diff.AddedSchemas {
		summary.WriteString(fmt.Sprintf("  + %s\n", schema.Name))
	}
	for _, schemaDiff := range p.Diff.ModifiedSchemas {
		summary.WriteString(fmt.Sprintf("  ~ %s\n", schemaDiff.New.Name))
	}
	for _, schema := range p.Diff.DroppedSchemas {
		summary.WriteString(fmt.Sprintf("  - %s\n", schema.Name))
	}
}

// writeTableChanges writes table changes
func (p *Plan) writeTableChanges(summary *strings.Builder) {
	for _, table := range p.Diff.AddedTables {
		summary.WriteString(fmt.Sprintf("  + %s.%s\n", table.Schema, table.Name))
	}
	for _, tableDiff := range p.Diff.ModifiedTables {
		summary.WriteString(fmt.Sprintf("  ~ %s.%s\n", tableDiff.Table.Schema, tableDiff.Table.Name))
	}
	for _, table := range p.Diff.DroppedTables {
		summary.WriteString(fmt.Sprintf("  - %s.%s\n", table.Schema, table.Name))
	}
}

// writeViewChanges writes view changes
func (p *Plan) writeViewChanges(summary *strings.Builder) {
	for _, view := range p.Diff.AddedViews {
		summary.WriteString(fmt.Sprintf("  + %s.%s\n", view.Schema, view.Name))
	}
	for _, viewDiff := range p.Diff.ModifiedViews {
		summary.WriteString(fmt.Sprintf("  ~ %s.%s\n", viewDiff.New.Schema, viewDiff.New.Name))
	}
	for _, view := range p.Diff.DroppedViews {
		summary.WriteString(fmt.Sprintf("  - %s.%s\n", view.Schema, view.Name))
	}
}

// writeFunctionChanges writes function changes
func (p *Plan) writeFunctionChanges(summary *strings.Builder) {
	for _, function := range p.Diff.AddedFunctions {
		summary.WriteString(fmt.Sprintf("  + %s.%s\n", function.Schema, function.Name))
	}
	for _, functionDiff := range p.Diff.ModifiedFunctions {
		summary.WriteString(fmt.Sprintf("  ~ %s.%s\n", functionDiff.New.Schema, functionDiff.New.Name))
	}
	for _, function := range p.Diff.DroppedFunctions {
		summary.WriteString(fmt.Sprintf("  - %s.%s\n", function.Schema, function.Name))
	}
}

// writeTypeChanges writes type changes
func (p *Plan) writeTypeChanges(summary *strings.Builder) {
	for _, typeObj := range p.Diff.AddedTypes {
		summary.WriteString(fmt.Sprintf("  + %s.%s\n", typeObj.Schema, typeObj.Name))
	}
	for _, typeDiff := range p.Diff.ModifiedTypes {
		summary.WriteString(fmt.Sprintf("  ~ %s.%s\n", typeDiff.New.Schema, typeDiff.New.Name))
	}
	for _, typeObj := range p.Diff.DroppedTypes {
		summary.WriteString(fmt.Sprintf("  - %s.%s\n", typeObj.Schema, typeObj.Name))
	}
}

// writeExtensionChanges writes extension changes
func (p *Plan) writeExtensionChanges(summary *strings.Builder) {
	for _, ext := range p.Diff.AddedExtensions {
		summary.WriteString(fmt.Sprintf("  + %s\n", ext.Name))
	}
	for _, ext := range p.Diff.DroppedExtensions {
		summary.WriteString(fmt.Sprintf("  - %s\n", ext.Name))
	}
}

// writeIndexChanges writes index changes
func (p *Plan) writeIndexChanges(summary *strings.Builder) {
	for _, index := range p.Diff.AddedIndexes {
		summary.WriteString(fmt.Sprintf("  + %s.%s\n", index.Schema, index.Name))
	}
	for _, index := range p.Diff.DroppedIndexes {
		summary.WriteString(fmt.Sprintf("  - %s.%s\n", index.Schema, index.Name))
	}
}

// writeTriggerChanges writes trigger changes
func (p *Plan) writeTriggerChanges(summary *strings.Builder) {
	for _, trigger := range p.Diff.AddedTriggers {
		summary.WriteString(fmt.Sprintf("  + %s.%s.%s\n", trigger.Schema, trigger.Table, trigger.Name))
	}
	for _, triggerDiff := range p.Diff.ModifiedTriggers {
		summary.WriteString(fmt.Sprintf("  ~ %s.%s.%s\n", triggerDiff.New.Schema, triggerDiff.New.Table, triggerDiff.New.Name))
	}
	for _, trigger := range p.Diff.DroppedTriggers {
		summary.WriteString(fmt.Sprintf("  - %s.%s.%s\n", trigger.Schema, trigger.Table, trigger.Name))
	}
}

// ObjectChange represents a single change to a database object
type ObjectChange struct {
	Address  string                 `json:"address"`
	Mode     string                 `json:"mode"`
	Type     string                 `json:"type"`
	Name     string                 `json:"name"`
	Schema   string                 `json:"schema"`
	Table    string                 `json:"table,omitempty"`
	Change   Change                 `json:"change"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Change represents the actual change being made
type Change struct {
	Actions []string               `json:"actions"`
	Before  map[string]interface{} `json:"before"`
	After   map[string]interface{} `json:"after"`
}

// PlanJSON represents the structured JSON output format
type PlanJSON struct {
	FormatVersion string          `json:"format_version"`
	CreatedAt     time.Time       `json:"created_at"`
	ObjectChanges []ObjectChange  `json:"object_changes"`
	Summary       PlanSummary     `json:"summary"`
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

// ToJSON returns the plan as structured JSON with only changed statements
func (p *Plan) ToJSON() (string, error) {
	planJSON := p.convertToStructuredJSON()
	
	data, err := json.MarshalIndent(planJSON, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal plan to JSON: %w", err)
	}
	return string(data), nil
}

// convertToStructuredJSON converts the DDLDiff to a structured JSON format
func (p *Plan) convertToStructuredJSON() *PlanJSON {
	planJSON := &PlanJSON{
		FormatVersion: "1.0",
		CreatedAt:     p.CreatedAt,
		ObjectChanges: []ObjectChange{},
		Summary: PlanSummary{
			ByType: make(map[string]TypeSummary),
		},
	}

	// Process added objects
	p.addObjectChanges(planJSON, "schema", p.Diff.AddedSchemas, nil, []string{"create"})
	p.addObjectChanges(planJSON, "table", p.Diff.AddedTables, nil, []string{"create"})
	p.addObjectChanges(planJSON, "view", p.Diff.AddedViews, nil, []string{"create"})
	p.addObjectChanges(planJSON, "function", p.Diff.AddedFunctions, nil, []string{"create"})
	p.addObjectChanges(planJSON, "extension", p.Diff.AddedExtensions, nil, []string{"create"})
	p.addObjectChanges(planJSON, "index", p.Diff.AddedIndexes, nil, []string{"create"})
	p.addObjectChanges(planJSON, "trigger", p.Diff.AddedTriggers, nil, []string{"create"})
	p.addObjectChanges(planJSON, "type", p.Diff.AddedTypes, nil, []string{"create"})

	// Process dropped objects
	p.addObjectChanges(planJSON, "schema", nil, p.Diff.DroppedSchemas, []string{"delete"})
	p.addObjectChanges(planJSON, "table", nil, p.Diff.DroppedTables, []string{"delete"})
	p.addObjectChanges(planJSON, "view", nil, p.Diff.DroppedViews, []string{"delete"})
	p.addObjectChanges(planJSON, "function", nil, p.Diff.DroppedFunctions, []string{"delete"})
	p.addObjectChanges(planJSON, "extension", nil, p.Diff.DroppedExtensions, []string{"delete"})
	p.addObjectChanges(planJSON, "index", nil, p.Diff.DroppedIndexes, []string{"delete"})
	p.addObjectChanges(planJSON, "trigger", nil, p.Diff.DroppedTriggers, []string{"delete"})
	p.addObjectChanges(planJSON, "type", nil, p.Diff.DroppedTypes, []string{"delete"})

	// Process modified objects
	p.addModifiedObjectChanges(planJSON, "schema", p.Diff.ModifiedSchemas)
	p.addModifiedObjectChanges(planJSON, "view", p.Diff.ModifiedViews)
	p.addModifiedObjectChanges(planJSON, "function", p.Diff.ModifiedFunctions)
	p.addModifiedObjectChanges(planJSON, "trigger", p.Diff.ModifiedTriggers)
	p.addModifiedObjectChanges(planJSON, "type", p.Diff.ModifiedTypes)

	// Process modified tables (more complex)
	for _, tableDiff := range p.Diff.ModifiedTables {
		p.addTableChanges(planJSON, tableDiff)
	}

	// Calculate summary
	p.calculateSummary(planJSON)

	return planJSON
}

// addObjectChanges adds object changes to the plan JSON
func (p *Plan) addObjectChanges(planJSON *PlanJSON, objType string, addedObjects, droppedObjects interface{}, actions []string) {
	var objects []interface{}
	
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
func (p *Plan) createObjectChange(objType string, obj interface{}, actions []string) *ObjectChange {
	change := &ObjectChange{
		Mode:   objType,
		Type:   objType,
		Change: Change{Actions: actions},
	}

	// Set before/after based on action
	if actions[0] == "create" {
		change.Change.Before = nil
		change.Change.After = p.objectToMap(obj)
	} else if actions[0] == "delete" {
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
func (p *Plan) objectToMap(obj interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
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
		result["is_unique"] = v.IsUnique
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
		result["is_identity"] = v.IsIdentity
		if v.IdentityGeneration != "" {
			result["identity_generation"] = v.IdentityGeneration
		}
	}
	
	return result
}

// addModifiedObjectChanges adds modified object changes
func (p *Plan) addModifiedObjectChanges(planJSON *PlanJSON, objType string, modifiedObjects interface{}) {
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

// addTableChanges adds table-level changes with column and constraint details
func (p *Plan) addTableChanges(planJSON *PlanJSON, tableDiff *diff.TableDiff) {
	// Add table-level change if there are modifications
	if len(tableDiff.AddedColumns) > 0 || len(tableDiff.DroppedColumns) > 0 ||
		len(tableDiff.ModifiedColumns) > 0 || len(tableDiff.AddedConstraints) > 0 ||
		len(tableDiff.DroppedConstraints) > 0 {
		
		change := ObjectChange{
			Address: fmt.Sprintf("%s.%s", tableDiff.Table.Schema, tableDiff.Table.Name),
			Mode:    "table",
			Type:    "table",
			Name:    tableDiff.Table.Name,
			Schema:  tableDiff.Table.Schema,
			Change: Change{
				Actions: []string{"update"},
				Before:  map[string]interface{}{},
				After:   p.objectToMap(tableDiff.Table),
			},
			Metadata: map[string]interface{}{
				"added_columns":       len(tableDiff.AddedColumns),
				"dropped_columns":     len(tableDiff.DroppedColumns),
				"modified_columns":    len(tableDiff.ModifiedColumns),
				"added_constraints":   len(tableDiff.AddedConstraints),
				"dropped_constraints": len(tableDiff.DroppedConstraints),
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

// Preview returns a detailed preview of all planned changes
func (p *Plan) Preview() string {
	var preview strings.Builder

	totalChanges := len(p.Diff.AddedSchemas) + len(p.Diff.AddedTables) + len(p.Diff.AddedViews) +
		len(p.Diff.AddedFunctions) + len(p.Diff.AddedExtensions) + len(p.Diff.AddedIndexes) +
		len(p.Diff.AddedTriggers) + len(p.Diff.AddedTypes) +
		len(p.Diff.ModifiedSchemas) + len(p.Diff.ModifiedTables) + len(p.Diff.ModifiedViews) +
		len(p.Diff.ModifiedFunctions) + len(p.Diff.ModifiedTriggers) + len(p.Diff.ModifiedTypes) +
		len(p.Diff.DroppedSchemas) + len(p.Diff.DroppedTables) + len(p.Diff.DroppedViews) +
		len(p.Diff.DroppedFunctions) + len(p.Diff.DroppedExtensions) + len(p.Diff.DroppedIndexes) +
		len(p.Diff.DroppedTriggers) + len(p.Diff.DroppedTypes)

	if totalChanges == 0 {
		preview.WriteString("No changes detected.\n")
		return preview.String()
	}

	preview.WriteString(fmt.Sprintf("Migration Plan (created at %s)\n", p.CreatedAt.Format(time.RFC3339)))
	preview.WriteString(strings.Repeat("=", 50) + "\n\n")

	preview.WriteString(p.Summary())

	return preview.String()
}

// GenerateMigrationSQL generates SQL statements for the migration
func (p *Plan) GenerateMigrationSQL() string {
	return p.Diff.GenerateMigrationSQL()
}
