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

	// Build event list
	var events []string
	for _, event := range tr.Events {
		events = append(events, string(event))
	}
	eventList := strings.Join(events, " OR ")

	stmt := fmt.Sprintf("CREATE TRIGGER %s %s %s ON %s.%s FOR EACH %s EXECUTE FUNCTION %s.%s();",
		tr.Name, tr.Timing, eventList, tr.Schema, tr.Table, tr.Level, tr.Schema, tr.Function)
	w.WriteStatementWithComment("TRIGGER", fmt.Sprintf("%s %s", tr.Table, tr.Name), tr.Schema, "", stmt)
	return w.String()
}