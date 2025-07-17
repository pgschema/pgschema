package ir

import (
	"regexp"
	"sort"
	"strings"
)

// normalizeIR normalizes the IR representation from inspector to be compatible with parser
func normalizeIR(ir *IR) {
	if ir == nil {
		return
	}

	for _, schema := range ir.Schemas {
		normalizeSchema(schema)
	}
}

// normalizeSchema normalizes all objects within a schema
func normalizeSchema(schema *Schema) {
	if schema == nil {
		return
	}

	// Normalize tables
	for _, table := range schema.Tables {
		normalizeTable(table)
	}

	// Normalize views
	for _, view := range schema.Views {
		normalizeView(view)
	}

	// Normalize functions
	for _, function := range schema.Functions {
		normalizeFunction(function)
	}
}

// normalizeTable normalizes table-related objects
func normalizeTable(table *Table) {
	if table == nil {
		return
	}

	// Normalize columns
	for _, column := range table.Columns {
		normalizeColumn(column)
	}

	// Normalize policies
	for _, policy := range table.Policies {
		normalizePolicy(policy)
	}

	// Normalize triggers
	for _, trigger := range table.Triggers {
		normalizeTrigger(trigger)
	}
}

// normalizeColumn normalizes column default values
func normalizeColumn(column *Column) {
	if column == nil || column.DefaultValue == nil {
		return
	}

	normalized := normalizeDefaultValue(*column.DefaultValue)
	column.DefaultValue = &normalized
}

// normalizePolicy normalizes RLS policy representation
func normalizePolicy(policy *RLSPolicy) {
	if policy == nil {
		return
	}

	// Normalize roles - ensure consistent ordering and case
	policy.Roles = normalizePolicyRoles(policy.Roles)

	// Normalize expressions by removing extra whitespace
	policy.Using = normalizeExpression(policy.Using)
	policy.WithCheck = normalizeExpression(policy.WithCheck)
}

// normalizePolicyRoles normalizes policy roles for consistent comparison
func normalizePolicyRoles(roles []string) []string {
	if len(roles) == 0 {
		return roles
	}

	// Convert to lowercase and sort for consistent comparison
	normalized := make([]string, len(roles))
	for i, role := range roles {
		// Normalize role names (PUBLIC vs public)
		normalized[i] = strings.ToLower(strings.TrimSpace(role))
	}

	// Sort to ensure consistent ordering
	sort.Strings(normalized)
	return normalized
}

// normalizeExpression normalizes SQL expressions by removing extra whitespace
func normalizeExpression(expr string) string {
	if expr == "" {
		return expr
	}

	// Remove extra whitespace and normalize
	expr = strings.TrimSpace(expr)
	expr = regexp.MustCompile(`\s+`).ReplaceAllString(expr, " ")

	return expr
}

// normalizeView normalizes view definition
func normalizeView(view *View) {
	if view == nil {
		return
	}

	view.Definition = normalizeViewDefinition(view.Definition)
}

// normalizeViewDefinition normalizes view SQL definition for consistent comparison
func normalizeViewDefinition(definition string) string {
	if definition == "" {
		return definition
	}

	// Remove extra whitespace and normalize
	definition = strings.TrimSpace(definition)
	definition = regexp.MustCompile(`\s+`).ReplaceAllString(definition, " ")

	// Normalize common SQL formatting differences
	definition = regexp.MustCompile(`\(\s+`).ReplaceAllString(definition, "(")
	definition = regexp.MustCompile(`\s+\)`).ReplaceAllString(definition, ")")
	definition = regexp.MustCompile(`\s*,\s*`).ReplaceAllString(definition, ", ")

	// Normalize JOIN syntax differences
	definition = regexp.MustCompile(`\s+JOIN\s+`).ReplaceAllString(definition, " JOIN ")
	definition = regexp.MustCompile(`\s+ON\s+`).ReplaceAllString(definition, " ON ")

	return definition
}

// normalizeFunction normalizes function signature and definition
func normalizeFunction(function *Function) {
	if function == nil {
		return
	}

	function.Signature = normalizeFunctionSignature(function.Signature)
	// Temporarily disable function definition normalization to avoid SQL syntax issues
	// function.Definition = normalizeFunctionDefinition(function.Definition)
}

// normalizeFunctionSignature normalizes function signatures for consistent comparison
func normalizeFunctionSignature(signature string) string {
	if signature == "" {
		return signature
	}

	// Remove extra whitespace
	signature = strings.TrimSpace(signature)
	signature = regexp.MustCompile(`\s+`).ReplaceAllString(signature, " ")

	// Normalize parameter formatting
	signature = regexp.MustCompile(`\(\s*`).ReplaceAllString(signature, "(")
	signature = regexp.MustCompile(`\s*\)`).ReplaceAllString(signature, ")")
	signature = regexp.MustCompile(`\s*,\s*`).ReplaceAllString(signature, ", ")

	return signature
}

// normalizeFunctionDefinition normalizes function body/definition
func normalizeFunctionDefinition(definition string) string {
	if definition == "" {
		return definition
	}

	// Remove leading/trailing whitespace
	definition = strings.TrimSpace(definition)

	// Normalize common SQL formatting patterns
	// Handle SQL block structure ($$...$$)
	if strings.Contains(definition, "$$") {
		definition = normalizeSQLBlock(definition)
	}

	// Normalize whitespace within the definition
	definition = regexp.MustCompile(`\s+`).ReplaceAllString(definition, " ")

	// Normalize common SQL keywords and patterns
	definition = strings.ReplaceAll(definition, " ( ", "(")
	definition = strings.ReplaceAll(definition, " ) ", ")")
	definition = regexp.MustCompile(`\(\s+`).ReplaceAllString(definition, "(")
	definition = regexp.MustCompile(`\s+\)`).ReplaceAllString(definition, ")")

	return definition
}

// normalizeSQLBlock normalizes SQL block definitions (e.g., function bodies)
func normalizeSQLBlock(definition string) string {
	// Split by $$ delimiters to handle function bodies
	parts := strings.Split(definition, "$$")
	if len(parts) >= 3 {
		// We have a function body between $$ delimiters
		// parts[0] is before first $$
		// parts[1] is the function body
		// parts[2] is after second $$

		functionBody := parts[1]

		// Normalize the function body by removing extra whitespace
		// but preserve essential structure for SQL validity
		lines := strings.Split(functionBody, "\n")
		var normalizedLines []string

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				normalizedLines = append(normalizedLines, trimmed)
			}
		}

		// Rejoin with newlines to preserve SQL structure - don't collapse to single line
		normalizedBody := strings.Join(normalizedLines, "\n")

		// Reconstruct the definition
		return parts[0] + "$$" + normalizedBody + "$$" + strings.Join(parts[2:], "$$")
	}

	return definition
}

// normalizeTrigger normalizes trigger representation
func normalizeTrigger(trigger *Trigger) {
	if trigger == nil {
		return
	}

	// Normalize trigger function call
	trigger.Function = normalizeTriggerFunctionCall(trigger.Function)

	// Normalize trigger events to standard order: INSERT, UPDATE, DELETE, TRUNCATE
	trigger.Events = normalizeTriggerEvents(trigger.Events)
}

// normalizeTriggerFunctionCall normalizes trigger function call syntax
func normalizeTriggerFunctionCall(functionCall string) string {
	if functionCall == "" {
		return functionCall
	}

	// Remove extra whitespace
	functionCall = strings.TrimSpace(functionCall)
	functionCall = regexp.MustCompile(`\s+`).ReplaceAllString(functionCall, " ")

	// Normalize function call formatting
	functionCall = regexp.MustCompile(`\(\s*`).ReplaceAllString(functionCall, "(")
	functionCall = regexp.MustCompile(`\s*\)`).ReplaceAllString(functionCall, ")")
	functionCall = regexp.MustCompile(`\s*,\s*`).ReplaceAllString(functionCall, ", ")

	return functionCall
}

// normalizeTriggerEvents normalizes trigger events to standard order
func normalizeTriggerEvents(events []TriggerEvent) []TriggerEvent {
	if len(events) == 0 {
		return events
	}

	// Define standard order: INSERT, UPDATE, DELETE, TRUNCATE
	standardOrder := []TriggerEvent{
		TriggerEventInsert,
		TriggerEventUpdate,
		TriggerEventDelete,
		TriggerEventTruncate,
	}

	// Create a set of events for quick lookup
	eventSet := make(map[TriggerEvent]bool)
	for _, event := range events {
		eventSet[event] = true
	}

	// Build normalized events in standard order
	var normalized []TriggerEvent
	for _, event := range standardOrder {
		if eventSet[event] {
			normalized = append(normalized, event)
		}
	}

	return normalized
}
