package ir

import (
	"fmt"
	"strings"
)

// Trigger represents a database trigger
type Trigger struct {
	Schema    string         `json:"schema"`
	Table     string         `json:"table"`
	Name      string         `json:"name"`
	Timing    TriggerTiming  `json:"timing"` // BEFORE, AFTER, INSTEAD OF
	Events    []TriggerEvent `json:"events"` // INSERT, UPDATE, DELETE
	Level     TriggerLevel   `json:"level"`  // ROW, STATEMENT
	Function  string         `json:"function"`
	Condition string         `json:"condition,omitempty"` // WHEN condition
	Comment   string         `json:"comment,omitempty"`
}

// GenerateSQL for Trigger
func (tr *Trigger) GenerateSQL() string {
	return tr.GenerateSQLWithSchema(tr.Schema)
}

// GenerateSQLWithSchema generates SQL for a trigger with target schema context
func (tr *Trigger) GenerateSQLWithSchema(targetSchema string) string {
	w := NewSQLWriter()

	// Build event list in standard order: INSERT, UPDATE, DELETE
	var events []string
	eventOrder := []TriggerEvent{TriggerEventInsert, TriggerEventUpdate, TriggerEventDelete}
	for _, orderEvent := range eventOrder {
		for _, triggerEvent := range tr.Events {
			if triggerEvent == orderEvent {
				events = append(events, string(triggerEvent))
				break
			}
		}
	}
	eventList := strings.Join(events, " OR ")

	// Only include table name without schema if it's in the target schema
	var tableName string
	if tr.Schema == targetSchema {
		tableName = tr.Table
	} else {
		tableName = fmt.Sprintf("%s.%s", tr.Schema, tr.Table)
	}

	// Function field should contain the complete function call including parameters
	stmt := fmt.Sprintf("CREATE TRIGGER %s %s %s ON %s FOR EACH %s EXECUTE FUNCTION %s;",
		tr.Name, tr.Timing, eventList, tableName, tr.Level, tr.Function)
	
	// For comment header, use "-" if in target schema
	commentSchema := tr.Schema
	if tr.Schema == targetSchema {
		commentSchema = "-"
	}
	w.WriteStatementWithComment("TRIGGER", fmt.Sprintf("%s %s", tr.Table, tr.Name), commentSchema, "", stmt, "")
	return w.String()
}

// GenerateMigrationSQL for Trigger (without comments for migration)
func (tr *Trigger) GenerateMigrationSQL() string {
	// Build event list in standard order: INSERT, UPDATE, DELETE
	var events []string
	eventOrder := []TriggerEvent{TriggerEventInsert, TriggerEventUpdate, TriggerEventDelete}
	for _, orderEvent := range eventOrder {
		for _, triggerEvent := range tr.Events {
			if triggerEvent == orderEvent {
				events = append(events, string(triggerEvent))
				break
			}
		}
	}
	eventList := strings.Join(events, " OR ")

	// Build the CREATE TRIGGER statement
	stmt := fmt.Sprintf("CREATE OR REPLACE TRIGGER %s\n    %s %s ON %s\n    FOR EACH %s",
		tr.Name, tr.Timing, eventList, tr.Table, tr.Level)

	// Add WHEN condition if present
	if tr.Condition != "" {
		stmt += fmt.Sprintf("\n    WHEN (%s)", tr.Condition)
	}

	// Add EXECUTE FUNCTION
	stmt += fmt.Sprintf("\n    EXECUTE FUNCTION %s;", tr.Function)

	return stmt
}
