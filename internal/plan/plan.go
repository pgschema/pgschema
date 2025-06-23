package plan

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pgschema/pgschema/internal/diff"
)

// ActionType represents the type of action in a plan
type ActionType string

const (
	ActionCreate ActionType = "create"
	ActionModify ActionType = "modify"
	ActionDelete ActionType = "delete"
)

// ResourceType represents the type of database resource
type ResourceType string

const (
	ResourceSchema    ResourceType = "schema"
	ResourceTable     ResourceType = "table"
	ResourceColumn    ResourceType = "column"
	ResourceConstraint ResourceType = "constraint"
	ResourceFunction  ResourceType = "function"
	ResourceIndex     ResourceType = "index"
	ResourceExtension ResourceType = "extension"
)

// Action represents a single action in the migration plan
type Action struct {
	Type         ActionType   `json:"type"`
	ResourceType ResourceType `json:"resource_type"`
	ResourceName string       `json:"resource_name"`
	Description  string       `json:"description"`
	SQL          string       `json:"sql,omitempty"`
}

// Plan represents the migration plan between two DDL states
type Plan struct {
	// The underlying diff data
	Diff *diff.DDLDiff `json:"diff"`
	
	// Plan metadata
	CreatedAt time.Time `json:"created_at"`
	Actions   []*Action `json:"actions"`
}

// NewPlan creates a new plan from a DDLDiff
func NewPlan(ddlDiff *diff.DDLDiff) *Plan {
	plan := &Plan{
		Diff:      ddlDiff,
		CreatedAt: time.Now(),
		Actions:   []*Action{},
	}
	plan.generateActions()
	return plan
}

// generateActions creates plan actions from the diff data
func (p *Plan) generateActions() {
	p.Actions = []*Action{}
	
	// Schema actions
	for _, schema := range p.Diff.DroppedSchemas {
		p.Actions = append(p.Actions, &Action{
			Type:         ActionDelete,
			ResourceType: ResourceSchema,
			ResourceName: schema.Name,
			Description:  fmt.Sprintf("Drop schema %s", schema.Name),
		})
	}
	
	for _, schema := range p.Diff.AddedSchemas {
		p.Actions = append(p.Actions, &Action{
			Type:         ActionCreate,
			ResourceType: ResourceSchema,
			ResourceName: schema.Name,
			Description:  fmt.Sprintf("Create schema %s", schema.Name),
		})
	}
	
	for _, schemaDiff := range p.Diff.ModifiedSchemas {
		p.Actions = append(p.Actions, &Action{
			Type:         ActionModify,
			ResourceType: ResourceSchema,
			ResourceName: schemaDiff.New.Name,
			Description:  fmt.Sprintf("Modify schema %s (owner: %s → %s)", schemaDiff.New.Name, schemaDiff.Old.Owner, schemaDiff.New.Owner),
		})
	}
	
	// Table actions
	for _, table := range p.Diff.DroppedTables {
		p.Actions = append(p.Actions, &Action{
			Type:         ActionDelete,
			ResourceType: ResourceTable,
			ResourceName: fmt.Sprintf("%s.%s", table.Schema, table.Name),
			Description:  fmt.Sprintf("Drop table %s.%s", table.Schema, table.Name),
		})
	}
	
	for _, table := range p.Diff.AddedTables {
		p.Actions = append(p.Actions, &Action{
			Type:         ActionCreate,
			ResourceType: ResourceTable,
			ResourceName: fmt.Sprintf("%s.%s", table.Schema, table.Name),
			Description:  fmt.Sprintf("Create table %s.%s", table.Schema, table.Name),
		})
	}
	
	for _, tableDiff := range p.Diff.ModifiedTables {
		table := tableDiff.Table
		changes := []string{}
		if len(tableDiff.AddedColumns) > 0 {
			changes = append(changes, fmt.Sprintf("%d columns added", len(tableDiff.AddedColumns)))
		}
		if len(tableDiff.DroppedColumns) > 0 {
			changes = append(changes, fmt.Sprintf("%d columns dropped", len(tableDiff.DroppedColumns)))
		}
		if len(tableDiff.ModifiedColumns) > 0 {
			changes = append(changes, fmt.Sprintf("%d columns modified", len(tableDiff.ModifiedColumns)))
		}
		if len(tableDiff.AddedConstraints) > 0 {
			changes = append(changes, fmt.Sprintf("%d constraints added", len(tableDiff.AddedConstraints)))
		}
		if len(tableDiff.DroppedConstraints) > 0 {
			changes = append(changes, fmt.Sprintf("%d constraints dropped", len(tableDiff.DroppedConstraints)))
		}
		
		p.Actions = append(p.Actions, &Action{
			Type:         ActionModify,
			ResourceType: ResourceTable,
			ResourceName: fmt.Sprintf("%s.%s", table.Schema, table.Name),
			Description:  fmt.Sprintf("Modify table %s.%s (%s)", table.Schema, table.Name, strings.Join(changes, ", ")),
		})
	}
	
	// Function actions
	for _, function := range p.Diff.DroppedFunctions {
		p.Actions = append(p.Actions, &Action{
			Type:         ActionDelete,
			ResourceType: ResourceFunction,
			ResourceName: fmt.Sprintf("%s.%s", function.Schema, function.Name),
			Description:  fmt.Sprintf("Drop function %s.%s", function.Schema, function.Name),
		})
	}
	
	for _, function := range p.Diff.AddedFunctions {
		p.Actions = append(p.Actions, &Action{
			Type:         ActionCreate,
			ResourceType: ResourceFunction,
			ResourceName: fmt.Sprintf("%s.%s", function.Schema, function.Name),
			Description:  fmt.Sprintf("Create function %s.%s", function.Schema, function.Name),
		})
	}
	
	for _, functionDiff := range p.Diff.ModifiedFunctions {
		function := functionDiff.New
		p.Actions = append(p.Actions, &Action{
			Type:         ActionModify,
			ResourceType: ResourceFunction,
			ResourceName: fmt.Sprintf("%s.%s", function.Schema, function.Name),
			Description:  fmt.Sprintf("Modify function %s.%s", function.Schema, function.Name),
		})
	}
	
	// Extension actions
	for _, ext := range p.Diff.DroppedExtensions {
		p.Actions = append(p.Actions, &Action{
			Type:         ActionDelete,
			ResourceType: ResourceExtension,
			ResourceName: ext.Name,
			Description:  fmt.Sprintf("Drop extension %s", ext.Name),
		})
	}
	
	for _, ext := range p.Diff.AddedExtensions {
		p.Actions = append(p.Actions, &Action{
			Type:         ActionCreate,
			ResourceType: ResourceExtension,
			ResourceName: ext.Name,
			Description:  fmt.Sprintf("Create extension %s", ext.Name),
		})
	}
	
	// Index actions
	for _, index := range p.Diff.DroppedIndexes {
		p.Actions = append(p.Actions, &Action{
			Type:         ActionDelete,
			ResourceType: ResourceIndex,
			ResourceName: fmt.Sprintf("%s.%s", index.Schema, index.Name),
			Description:  fmt.Sprintf("Drop index %s.%s", index.Schema, index.Name),
		})
	}
	
	for _, index := range p.Diff.AddedIndexes {
		p.Actions = append(p.Actions, &Action{
			Type:         ActionCreate,
			ResourceType: ResourceIndex,
			ResourceName: fmt.Sprintf("%s.%s", index.Schema, index.Name),
			Description:  fmt.Sprintf("Create index %s.%s", index.Schema, index.Name),
		})
	}
}

// Summary returns a human-readable summary of the plan
func (p *Plan) Summary() string {
	var summary strings.Builder
	
	createCount := 0
	modifyCount := 0
	deleteCount := 0
	
	for _, action := range p.Actions {
		switch action.Type {
		case ActionCreate:
			createCount++
		case ActionModify:
			modifyCount++
		case ActionDelete:
			deleteCount++
		}
	}
	
	totalActions := createCount + modifyCount + deleteCount
	
	if totalActions == 0 {
		summary.WriteString("No changes detected.\n")
		return summary.String()
	}
	
	summary.WriteString(fmt.Sprintf("Plan: %d to add, %d to change, %d to destroy.\n\n", createCount, modifyCount, deleteCount))
	
	// Group actions by type for better readability
	if createCount > 0 {
		summary.WriteString("Resources to be created:\n")
		for _, action := range p.Actions {
			if action.Type == ActionCreate {
				summary.WriteString(fmt.Sprintf("  + %s %s\n", action.ResourceType, action.ResourceName))
			}
		}
		summary.WriteString("\n")
	}
	
	if modifyCount > 0 {
		summary.WriteString("Resources to be modified:\n")
		for _, action := range p.Actions {
			if action.Type == ActionModify {
				summary.WriteString(fmt.Sprintf("  ~ %s %s\n", action.ResourceType, action.ResourceName))
			}
		}
		summary.WriteString("\n")
	}
	
	if deleteCount > 0 {
		summary.WriteString("Resources to be destroyed:\n")
		for _, action := range p.Actions {
			if action.Type == ActionDelete {
				summary.WriteString(fmt.Sprintf("  - %s %s\n", action.ResourceType, action.ResourceName))
			}
		}
		summary.WriteString("\n")
	}
	
	return summary.String()
}

// ToJSON returns the plan as JSON
func (p *Plan) ToJSON() (string, error) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal plan to JSON: %w", err)
	}
	return string(data), nil
}

// Preview returns a detailed preview of all planned actions
func (p *Plan) Preview() string {
	var preview strings.Builder
	
	if len(p.Actions) == 0 {
		preview.WriteString("No changes detected.\n")
		return preview.String()
	}
	
	preview.WriteString(fmt.Sprintf("Migration Plan (created at %s)\n", p.CreatedAt.Format(time.RFC3339)))
	preview.WriteString(strings.Repeat("=", 50) + "\n\n")
	
	for i, action := range p.Actions {
		var symbol string
		switch action.Type {
		case ActionCreate:
			symbol = "+"
		case ActionModify:
			symbol = "~"
		case ActionDelete:
			symbol = "-"
		}
		
		preview.WriteString(fmt.Sprintf("%s [%d] %s\n", symbol, i+1, action.Description))
	}
	
	preview.WriteString("\n" + p.Summary())
	
	return preview.String()
}

// GenerateMigrationSQL generates SQL statements for the migration
func (p *Plan) GenerateMigrationSQL() string {
	return p.Diff.GenerateMigrationSQL()
}