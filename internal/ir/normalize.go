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
	// For policy expressions, we want to preserve parentheses as they are part of the expected format
	policy.Using = normalizePolicyExpression(policy.Using)
	policy.WithCheck = normalizePolicyExpression(policy.WithCheck)
}

// normalizePolicyRoles normalizes policy roles for consistent comparison
func normalizePolicyRoles(roles []string) []string {
	if len(roles) == 0 {
		return roles
	}

	// Normalize role names with special handling for PUBLIC
	normalized := make([]string, len(roles))
	for i, role := range roles {
		trimmed := strings.TrimSpace(role)
		// Keep PUBLIC in uppercase, normalize others to lowercase
		if strings.ToUpper(trimmed) == "PUBLIC" {
			normalized[i] = "PUBLIC"
		} else {
			normalized[i] = strings.ToLower(trimmed)
		}
	}

	// Sort to ensure consistent ordering
	sort.Strings(normalized)
	return normalized
}

// normalizePolicyExpression normalizes policy expressions (USING/WITH CHECK clauses)
// It preserves parentheses as they are part of the expected format for policies
func normalizePolicyExpression(expr string) string {
	if expr == "" {
		return expr
	}

	// Remove extra whitespace and normalize
	expr = strings.TrimSpace(expr)
	expr = regexp.MustCompile(`\s+`).ReplaceAllString(expr, " ")

	// For policy expressions, we don't remove parentheses as they are expected

	// Normalize PostgreSQL internal type names to standard SQL types
	expr = normalizePostgreSQLType(expr)

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

// removeUnnecessaryParentheses removes outer parentheses only for complex expressions
// Simple expressions keep their parentheses to match PostgreSQL formatting expectations
func removeUnnecessaryParentheses(expr string) string {
	if expr == "" {
		return expr
	}

	// Only remove outer parentheses for complex expressions that contain
	// function calls, type casts, or other complex elements
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		// Check if parentheses are balanced and cover the entire expression
		inner := expr[1 : len(expr)-1]
		if isBalancedParentheses(inner) {
			// Only remove outer parentheses for complex expressions
			// (with function calls, type casts, etc.)
			if isComplexExpression(inner) {
				return inner
			}
		}
	}

	return expr
}

// isBalancedParentheses checks if parentheses are properly balanced in the expression
func isBalancedParentheses(expr string) bool {
	count := 0
	inQuotes := false
	var quoteChar rune

	for _, r := range expr {
		if !inQuotes {
			if r == '\'' || r == '"' {
				inQuotes = true
				quoteChar = r
			} else if r == '(' {
				count++
			} else if r == ')' {
				count--
				if count < 0 {
					return false
				}
			}
		} else {
			if r == quoteChar {
				inQuotes = false
			}
		}
	}

	return count == 0
}

// isComplexExpression checks if the expression contains complex elements
// like function calls, type casts, or nested operations that justify removing outer parentheses
func isComplexExpression(expr string) bool {
	// Check for function calls (contains parentheses)
	if strings.Contains(expr, "(") && strings.Contains(expr, ")") {
		return true
	}

	// Check for type casts (contains ::)
	if strings.Contains(expr, "::") {
		return true
	}

	// Check for complex operators or multiple operations
	complexPatterns := []string{
		" AND ", " OR ", " IN ", " NOT IN ", " LIKE ", " ILIKE ",
		" IS NULL", " IS NOT NULL", " BETWEEN ",
	}

	exprUpper := strings.ToUpper(expr)
	for _, pattern := range complexPatterns {
		if strings.Contains(exprUpper, pattern) {
			return true
		}
	}

	// For simple expressions like "tenant_id = 1", return false
	// to keep the outer parentheses
	return false
}

// normalizePostgreSQLType normalizes PostgreSQL internal type names to standard SQL types.
// This function handles both expressions (with type casts) and direct type names.
func normalizePostgreSQLType(input string) string {
	if input == "" {
		return input
	}

	// Map of PostgreSQL internal types to standard SQL types
	typeMap := map[string]string{
		// Numeric types
		"int2":               "smallint",
		"int4":               "integer",
		"int8":               "bigint",
		"float4":             "real",
		"float8":             "double precision",
		"bool":               "boolean",
		"pg_catalog.int2":    "smallint",
		"pg_catalog.int4":    "integer",
		"pg_catalog.int8":    "bigint",
		"pg_catalog.float4":  "real",
		"pg_catalog.float8":  "double precision",
		"pg_catalog.bool":    "boolean",
		"pg_catalog.numeric": "numeric",

		// Character types
		"bpchar":             "character",
		"varchar":            "character varying",
		"pg_catalog.text":    "text",
		"pg_catalog.varchar": "character varying",
		"pg_catalog.bpchar":  "character",

		// Date/time types - convert verbose forms to canonical short forms
		"timestamp with time zone": "timestamptz",
		"time with time zone":      "timetz",
		"timestamptz":              "timestamptz",
		"timetz":                   "timetz",
		"pg_catalog.timestamptz":   "timestamptz",
		"pg_catalog.timestamp":     "timestamp",
		"pg_catalog.date":          "date",
		"pg_catalog.time":          "time",
		"pg_catalog.timetz":        "timetz",
		"pg_catalog.interval":      "interval",

		// Array types (internal PostgreSQL array notation)
		"_text":        "text[]",
		"_int2":        "smallint[]",
		"_int4":        "integer[]",
		"_int8":        "bigint[]",
		"_float4":      "real[]",
		"_float8":      "double precision[]",
		"_bool":        "boolean[]",
		"_varchar":     "character varying[]",
		"_char":        "character[]",
		"_bpchar":      "character[]",
		"_numeric":     "numeric[]",
		"_uuid":        "uuid[]",
		"_json":        "json[]",
		"_jsonb":       "jsonb[]",
		"_bytea":       "bytea[]",
		"_inet":        "inet[]",
		"_cidr":        "cidr[]",
		"_macaddr":     "macaddr[]",
		"_macaddr8":    "macaddr8[]",
		"_date":        "date[]",
		"_time":        "time[]",
		"_timetz":      "timetz[]",
		"_timestamp":   "timestamp[]",
		"_timestamptz": "timestamptz[]",
		"_interval":    "interval[]",

		// Other common types
		"pg_catalog.uuid":    "uuid",
		"pg_catalog.json":    "json",
		"pg_catalog.jsonb":   "jsonb",
		"pg_catalog.bytea":   "bytea",
		"pg_catalog.inet":    "inet",
		"pg_catalog.cidr":    "cidr",
		"pg_catalog.macaddr": "macaddr",

		// Serial types
		"serial":      "serial",
		"smallserial": "smallserial",
		"bigserial":   "bigserial",
	}

	// Check if this is an expression with type casts (contains "::")
	if strings.Contains(input, "::") {
		// Handle expressions with type casts
		expr := input

		// Replace PostgreSQL internal type names with standard SQL types in type casts
		for pgType, sqlType := range typeMap {
			expr = strings.ReplaceAll(expr, "::"+pgType, "::"+sqlType)
		}

		// Handle pg_catalog prefix removal for unmapped types in type casts
		// Look for patterns like "::pg_catalog.sometype"
		if strings.Contains(expr, "::pg_catalog.") {
			expr = regexp.MustCompile(`::pg_catalog\.(\w+)`).ReplaceAllString(expr, "::$1")
		}

		return expr
	}

	// Handle direct type names
	typeName := input

	// Check if we have a direct mapping
	if normalized, exists := typeMap[typeName]; exists {
		return normalized
	}

	// Remove pg_catalog prefix for unmapped types
	if after, found := strings.CutPrefix(typeName, "pg_catalog."); found {
		return after
	}

	// Return as-is if no mapping found
	return typeName
}
