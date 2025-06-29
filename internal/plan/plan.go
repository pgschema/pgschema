package plan

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pgschema/pgschema/internal/diff"
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
	
	// Count changes from DDLDiff
	createCount := len(p.Diff.AddedSchemas) + len(p.Diff.AddedTables) + len(p.Diff.AddedViews) + 
		len(p.Diff.AddedFunctions) + len(p.Diff.AddedExtensions) + len(p.Diff.AddedIndexes)
	
	modifyCount := len(p.Diff.ModifiedSchemas) + len(p.Diff.ModifiedTables) + len(p.Diff.ModifiedViews) + 
		len(p.Diff.ModifiedFunctions)
	
	deleteCount := len(p.Diff.DroppedSchemas) + len(p.Diff.DroppedTables) + len(p.Diff.DroppedViews) + 
		len(p.Diff.DroppedFunctions) + len(p.Diff.DroppedExtensions) + len(p.Diff.DroppedIndexes)
	
	totalChanges := createCount + modifyCount + deleteCount
	
	if totalChanges == 0 {
		summary.WriteString("No changes detected.\n")
		return summary.String()
	}
	
	summary.WriteString(fmt.Sprintf("Plan: %d to add, %d to change, %d to destroy.\n\n", createCount, modifyCount, deleteCount))
	
	// Group changes by type for better readability
	if createCount > 0 {
		summary.WriteString("Resources to be created:\n")
		for _, schema := range p.Diff.AddedSchemas {
			summary.WriteString(fmt.Sprintf("  + schema %s\n", schema.Name))
		}
		for _, table := range p.Diff.AddedTables {
			summary.WriteString(fmt.Sprintf("  + table %s.%s\n", table.Schema, table.Name))
		}
		for _, view := range p.Diff.AddedViews {
			summary.WriteString(fmt.Sprintf("  + view %s.%s\n", view.Schema, view.Name))
		}
		for _, function := range p.Diff.AddedFunctions {
			summary.WriteString(fmt.Sprintf("  + function %s.%s\n", function.Schema, function.Name))
		}
		for _, ext := range p.Diff.AddedExtensions {
			summary.WriteString(fmt.Sprintf("  + extension %s\n", ext.Name))
		}
		for _, index := range p.Diff.AddedIndexes {
			summary.WriteString(fmt.Sprintf("  + index %s.%s\n", index.Schema, index.Name))
		}
		summary.WriteString("\n")
	}
	
	if modifyCount > 0 {
		summary.WriteString("Resources to be modified:\n")
		for _, schemaDiff := range p.Diff.ModifiedSchemas {
			summary.WriteString(fmt.Sprintf("  ~ schema %s\n", schemaDiff.New.Name))
		}
		for _, tableDiff := range p.Diff.ModifiedTables {
			summary.WriteString(fmt.Sprintf("  ~ table %s.%s\n", tableDiff.Table.Schema, tableDiff.Table.Name))
		}
		for _, viewDiff := range p.Diff.ModifiedViews {
			summary.WriteString(fmt.Sprintf("  ~ view %s.%s\n", viewDiff.New.Schema, viewDiff.New.Name))
		}
		for _, functionDiff := range p.Diff.ModifiedFunctions {
			summary.WriteString(fmt.Sprintf("  ~ function %s.%s\n", functionDiff.New.Schema, functionDiff.New.Name))
		}
		summary.WriteString("\n")
	}
	
	if deleteCount > 0 {
		summary.WriteString("Resources to be destroyed:\n")
		for _, schema := range p.Diff.DroppedSchemas {
			summary.WriteString(fmt.Sprintf("  - schema %s\n", schema.Name))
		}
		for _, table := range p.Diff.DroppedTables {
			summary.WriteString(fmt.Sprintf("  - table %s.%s\n", table.Schema, table.Name))
		}
		for _, view := range p.Diff.DroppedViews {
			summary.WriteString(fmt.Sprintf("  - view %s.%s\n", view.Schema, view.Name))
		}
		for _, function := range p.Diff.DroppedFunctions {
			summary.WriteString(fmt.Sprintf("  - function %s.%s\n", function.Schema, function.Name))
		}
		for _, ext := range p.Diff.DroppedExtensions {
			summary.WriteString(fmt.Sprintf("  - extension %s\n", ext.Name))
		}
		for _, index := range p.Diff.DroppedIndexes {
			summary.WriteString(fmt.Sprintf("  - index %s.%s\n", index.Schema, index.Name))
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

// Preview returns a detailed preview of all planned changes
func (p *Plan) Preview() string {
	var preview strings.Builder
	
	totalChanges := len(p.Diff.AddedSchemas) + len(p.Diff.AddedTables) + len(p.Diff.AddedViews) + 
		len(p.Diff.AddedFunctions) + len(p.Diff.AddedExtensions) + len(p.Diff.AddedIndexes) +
		len(p.Diff.ModifiedSchemas) + len(p.Diff.ModifiedTables) + len(p.Diff.ModifiedViews) + 
		len(p.Diff.ModifiedFunctions) +
		len(p.Diff.DroppedSchemas) + len(p.Diff.DroppedTables) + len(p.Diff.DroppedViews) + 
		len(p.Diff.DroppedFunctions) + len(p.Diff.DroppedExtensions) + len(p.Diff.DroppedIndexes)
	
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