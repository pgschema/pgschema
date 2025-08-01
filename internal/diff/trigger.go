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

// generateCreateTriggersFromTables collects and creates all triggers from added tables
func generateCreateTriggersFromTables(tables []*ir.Table, targetSchema string, collector *diffCollector) {
	var allTriggers []*ir.Trigger

	// Collect all triggers from added tables in deterministic order
	for _, table := range tables {
		// Sort trigger names for deterministic ordering
		triggerNames := sortedKeys(table.Triggers)

		// Add triggers in sorted order
		for _, triggerName := range triggerNames {
			trigger := table.Triggers[triggerName]
			allTriggers = append(allTriggers, trigger)
		}
	}

	// Generate CREATE TRIGGER statements for all collected triggers
	if len(allTriggers) > 0 {
		generateCreateTriggersSQL(allTriggers, targetSchema, collector)
	}
}

// generateCreateTriggersSQL generates CREATE OR REPLACE TRIGGER statements
func generateCreateTriggersSQL(triggers []*ir.Trigger, targetSchema string, collector *diffCollector) {
	// Sort triggers by name for consistent ordering
	sortedTriggers := make([]*ir.Trigger, len(triggers))
	copy(sortedTriggers, triggers)
	sort.Slice(sortedTriggers, func(i, j int) bool {
		return sortedTriggers[i].Name < sortedTriggers[j].Name
	})

	for _, trigger := range sortedTriggers {
		sql := generateTriggerSQLWithMode(trigger, targetSchema)

		// Create context for this statement
		context := &diffContext{
			Type:                "trigger",
			Operation:           "create",
			Path:                fmt.Sprintf("%s.%s", trigger.Schema, trigger.Name),
			Source:              trigger,
			CanRunInTransaction: true,
		}

		collector.collect(context, sql)
	}
}

// generateTriggerSQLWithMode generates CREATE [OR REPLACE] TRIGGER statement
func generateTriggerSQLWithMode(trigger *ir.Trigger, targetSchema string) string {
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

	// Build the trigger statement with proper formatting
	stmt := fmt.Sprintf("CREATE OR REPLACE TRIGGER %s\n    %s %s ON %s\n    FOR EACH %s",
		trigger.Name, trigger.Timing, eventList, tableName, trigger.Level)

	// Add WHEN clause if present
	if trigger.Condition != "" {
		stmt += fmt.Sprintf("\n    WHEN (%s)", trigger.Condition)
	}

	// Add EXECUTE FUNCTION clause
	stmt += fmt.Sprintf("\n    EXECUTE FUNCTION %s;", trigger.Function)

	return stmt
}
