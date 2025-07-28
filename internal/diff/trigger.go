package diff

import (
	"fmt"
	"sort"
	"strings"

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

// generateCreateTriggersSQL generates CREATE OR REPLACE TRIGGER statements
func generateCreateTriggersSQL(w Writer, triggers []*ir.Trigger, targetSchema string, compare bool) {
	// Sort triggers by name for consistent ordering
	sortedTriggers := make([]*ir.Trigger, len(triggers))
	copy(sortedTriggers, triggers)
	sort.Slice(sortedTriggers, func(i, j int) bool {
		return sortedTriggers[i].Name < sortedTriggers[j].Name
	})

	for _, trigger := range sortedTriggers {
		w.WriteDDLSeparator()
		sql := generateTriggerSQLWithMode(trigger, targetSchema, compare)
		w.WriteStatementWithComment("TRIGGER", trigger.Name, trigger.Schema, "", sql, targetSchema)
	}
}

// generateModifyTriggersSQL generates CREATE OR REPLACE TRIGGER statements for modified triggers
func generateModifyTriggersSQL(w Writer, diffs []*TriggerDiff, targetSchema string) {
	for _, diff := range diffs {
		w.WriteDDLSeparator()
		sql := generateTriggerSQLWithMode(diff.New, targetSchema, true) // Use OR REPLACE for modified triggers
		w.WriteStatementWithComment("TRIGGER", diff.New.Name, diff.New.Schema, "", sql, targetSchema)
	}
}

// generateDropTriggersSQL generates DROP TRIGGER statements
func generateDropTriggersSQL(w Writer, triggers []*ir.Trigger, targetSchema string) {
	// Sort triggers by name for consistent ordering
	sortedTriggers := make([]*ir.Trigger, len(triggers))
	copy(sortedTriggers, triggers)
	sort.Slice(sortedTriggers, func(i, j int) bool {
		return sortedTriggers[i].Name < sortedTriggers[j].Name
	})

	for _, trigger := range sortedTriggers {
		w.WriteDDLSeparator()
		tableName := qualifyEntityName(trigger.Schema, trigger.Table, targetSchema)
		sql := fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s;", trigger.Name, tableName)
		w.WriteStatementWithComment("TRIGGER", trigger.Name, trigger.Schema, "", sql, targetSchema)
	}
}

// generateTriggerSQLWithMode generates CREATE [OR REPLACE] TRIGGER statement
func generateTriggerSQLWithMode(trigger *ir.Trigger, targetSchema string, useReplace bool) string {
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
	tableName := qualifyEntityName(trigger.Schema, trigger.Table, targetSchema)

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
