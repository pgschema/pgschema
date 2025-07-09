package diff

import (
	"fmt"
	"sort"

	"github.com/pgschema/pgschema/internal/ir"
)

// triggersEqual compares two triggers for equality
func triggersEqual(old, new *ir.Trigger) bool {
	if old.Schema != new.Schema {
		return false
	}
	if old.Table != new.Table {
		return false
	}
	if old.Name != new.Name {
		return false
	}
	if old.Timing != new.Timing {
		return false
	}
	if old.Level != new.Level {
		return false
	}
	if old.Function != new.Function {
		return false
	}
	if old.Condition != new.Condition {
		return false
	}

	// Compare events
	if len(old.Events) != len(new.Events) {
		return false
	}
	for i, event := range old.Events {
		if event != new.Events[i] {
			return false
		}
	}

	return true
}

// GenerateDropTriggerSQL generates SQL for dropping triggers
func GenerateDropTriggerSQL(triggers []*ir.Trigger) []string {
	var statements []string
	
	// Sort triggers by schema.table.name for consistent ordering
	sortedTriggers := make([]*ir.Trigger, len(triggers))
	copy(sortedTriggers, triggers)
	sort.Slice(sortedTriggers, func(i, j int) bool {
		keyI := sortedTriggers[i].Schema + "." + sortedTriggers[i].Table + "." + sortedTriggers[i].Name
		keyJ := sortedTriggers[j].Schema + "." + sortedTriggers[j].Table + "." + sortedTriggers[j].Name
		return keyI < keyJ
	})
	
	for _, trigger := range sortedTriggers {
		statements = append(statements, fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s.%s;", trigger.Name, trigger.Schema, trigger.Table))
	}
	
	return statements
}

// GenerateCreateTriggerSQL generates SQL for creating triggers
func (d *DDLDiff) GenerateCreateTriggerSQL(triggers []*ir.Trigger) []string {
	var statements []string
	
	// Sort triggers by schema.table.name for consistent ordering
	sortedTriggers := make([]*ir.Trigger, len(triggers))
	copy(sortedTriggers, triggers)
	sort.Slice(sortedTriggers, func(i, j int) bool {
		keyI := sortedTriggers[i].Schema + "." + sortedTriggers[i].Table + "." + sortedTriggers[i].Name
		keyJ := sortedTriggers[j].Schema + "." + sortedTriggers[j].Table + "." + sortedTriggers[j].Name
		return keyI < keyJ
	})
	
	for _, trigger := range sortedTriggers {
		statements = append(statements, d.generateTriggerSQL(trigger, ""))
	}
	
	return statements
}

// GenerateAlterTriggerSQL generates SQL for modifying triggers
func (d *DDLDiff) GenerateAlterTriggerSQL(triggerDiffs []*TriggerDiff) []string {
	var statements []string
	
	// Sort modified triggers by schema.table.name for consistent ordering
	sortedTriggerDiffs := make([]*TriggerDiff, len(triggerDiffs))
	copy(sortedTriggerDiffs, triggerDiffs)
	sort.Slice(sortedTriggerDiffs, func(i, j int) bool {
		keyI := sortedTriggerDiffs[i].New.Schema + "." + sortedTriggerDiffs[i].New.Table + "." + sortedTriggerDiffs[i].New.Name
		keyJ := sortedTriggerDiffs[j].New.Schema + "." + sortedTriggerDiffs[j].New.Table + "." + sortedTriggerDiffs[j].New.Name
		return keyI < keyJ
	})
	
	for _, triggerDiff := range sortedTriggerDiffs {
		// Use CREATE OR REPLACE for trigger modifications
		statements = append(statements, d.generateTriggerSQL(triggerDiff.New, ""))
	}
	
	return statements
}