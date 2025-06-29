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

	// Function field should contain the complete function call including parameters
	stmt := fmt.Sprintf("CREATE TRIGGER %s %s %s ON %s.%s FOR EACH %s EXECUTE FUNCTION %s;",
		tr.Name, tr.Timing, eventList, tr.Schema, tr.Table, tr.Level, tr.Function)
	w.WriteStatementWithComment("TRIGGER", fmt.Sprintf("%s %s", tr.Table, tr.Name), tr.Schema, "", stmt)
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
	stmt := fmt.Sprintf("CREATE OR REPLACE TRIGGER %s\n    %s %s ON %s.%s\n    FOR EACH %s",
		tr.Name, tr.Timing, eventList, tr.Schema, tr.Table, tr.Level)

	// Add WHEN condition if present
	if tr.Condition != "" {
		stmt += fmt.Sprintf("\n    WHEN (%s)", tr.Condition)
	}

	// Add EXECUTE FUNCTION
	stmt += fmt.Sprintf("\n    EXECUTE FUNCTION %s;", tr.Function)

	return stmt
}