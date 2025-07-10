package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pgschema/pgschema/internal/ir"
	"github.com/pgschema/pgschema/internal/utils"
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

// generateDropTriggersSQL generates DROP TRIGGER statements
func (d *DDLDiff) generateDropTriggersSQL(w *SQLWriter, triggers []*ir.Trigger, targetSchema string) {
	// Sort triggers by name for consistent ordering
	sortedTriggers := make([]*ir.Trigger, len(triggers))
	copy(sortedTriggers, triggers)
	sort.Slice(sortedTriggers, func(i, j int) bool {
		return sortedTriggers[i].Name < sortedTriggers[j].Name
	})

	for _, trigger := range sortedTriggers {
		w.WriteDDLSeparator()
		sql := fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s;", trigger.Name, trigger.Table)
		w.WriteStatementWithComment("TRIGGER", trigger.Name, trigger.Schema, "", sql, targetSchema)
	}
}

// generateCreateTriggersSQL generates CREATE OR REPLACE TRIGGER statements
func (d *DDLDiff) generateCreateTriggersSQL(w *SQLWriter, triggers []*ir.Trigger, targetSchema string) {
	// Sort triggers by name for consistent ordering
	sortedTriggers := make([]*ir.Trigger, len(triggers))
	copy(sortedTriggers, triggers)
	sort.Slice(sortedTriggers, func(i, j int) bool {
		return sortedTriggers[i].Name < sortedTriggers[j].Name
	})

	for _, trigger := range sortedTriggers {
		w.WriteDDLSeparator()
		sql := d.generateTriggerSQLWithMode(trigger, targetSchema, true) // Use OR REPLACE for added triggers
		w.WriteStatementWithComment("TRIGGER", trigger.Name, trigger.Schema, "", sql, targetSchema)
	}
}

// generateModifyTriggersSQL generates CREATE OR REPLACE TRIGGER statements for modified triggers
func (d *DDLDiff) generateModifyTriggersSQL(w *SQLWriter, diffs []*TriggerDiff, targetSchema string) {
	for _, diff := range diffs {
		w.WriteDDLSeparator()
		sql := d.generateTriggerSQLWithMode(diff.New, targetSchema, true) // Use OR REPLACE for modified triggers
		w.WriteStatementWithComment("TRIGGER", diff.New.Name, diff.New.Schema, "", sql, targetSchema)
	}
}

// generateTriggerSQL generates CREATE TRIGGER statement
func (d *DDLDiff) generateTriggerSQL(trigger *ir.Trigger, targetSchema string) string {
	return d.generateTriggerSQLWithMode(trigger, targetSchema, false)
}

// generateTriggerSQLWithMode generates CREATE [OR REPLACE] TRIGGER statement
func (d *DDLDiff) generateTriggerSQLWithMode(trigger *ir.Trigger, targetSchema string, useReplace bool) string {
	// Build event list in standard order: INSERT, UPDATE, DELETE
	var events []string
	eventOrder := []ir.TriggerEvent{ir.TriggerEventInsert, ir.TriggerEventUpdate, ir.TriggerEventDelete}
	for _, orderEvent := range eventOrder {
		for _, triggerEvent := range trigger.Events {
			if triggerEvent == orderEvent {
				events = append(events, string(triggerEvent))
				break
			}
		}
	}
	eventList := strings.Join(events, " OR ")

	// Only include table name without schema if it's in the target schema
	tableName := utils.QualifyEntityName(trigger.Schema, trigger.Table, targetSchema)

	// Determine CREATE statement type
	createClause := "CREATE TRIGGER"
	if useReplace {
		createClause = "CREATE OR REPLACE TRIGGER"
	}

	// Build the trigger statement with proper formatting
	stmt := fmt.Sprintf("%s %s\n    %s %s ON %s\n    FOR EACH %s",
		createClause, trigger.Name, trigger.Timing, eventList, tableName, trigger.Level)
	
	// Add WHEN clause if present
	if trigger.Condition != "" {
		stmt += fmt.Sprintf("\n    WHEN (%s)", trigger.Condition)
	}
	
	// Add EXECUTE FUNCTION clause
	stmt += fmt.Sprintf("\n    EXECUTE FUNCTION %s;", trigger.Function)

	return stmt
}

// generateTableTriggers generates SQL for triggers belonging to a specific table
func (d *DDLDiff) generateTableTriggers(w *SQLWriter, table *ir.Table, targetSchema string) {
	isDumpScenario := len(d.AddedTables) > 0 && len(d.DroppedTables) == 0 && len(d.ModifiedTables) == 0
	
	// Get sorted trigger names for consistent output
	triggerNames := make([]string, 0, len(table.Triggers))
	for triggerName := range table.Triggers {
		triggerNames = append(triggerNames, triggerName)
	}
	sort.Strings(triggerNames)

	for _, triggerName := range triggerNames {
		trigger := table.Triggers[triggerName]
		// Include all triggers for this table (for dump scenarios) or only added triggers (for diff scenarios)
		shouldInclude := isDumpScenario || d.isTriggerInAddedList(trigger)
		if shouldInclude {
			w.WriteDDLSeparator()
			// Use CREATE TRIGGER for dump scenarios, CREATE OR REPLACE for diff scenarios
			sql := d.generateTriggerSQL(trigger, targetSchema) // Always use CREATE TRIGGER for table-level generation
			w.WriteStatementWithComment("TRIGGER", triggerName, table.Schema, "", sql, targetSchema)
		}
	}
}

// isTriggerInAddedList checks if a trigger is in the added triggers list
func (d *DDLDiff) isTriggerInAddedList(trigger *ir.Trigger) bool {
	for _, addedTrigger := range d.AddedTriggers {
		if addedTrigger.Name == trigger.Name && addedTrigger.Schema == trigger.Schema && addedTrigger.Table == trigger.Table {
			return true
		}
	}
	return false
}