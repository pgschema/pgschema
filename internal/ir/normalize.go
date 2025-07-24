package ir

import (
	"fmt"
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

	// Normalize procedures
	for _, procedure := range schema.Procedures {
		normalizeProcedure(procedure)
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

	// Normalize indexes
	for _, index := range table.Indexes {
		normalizeIndex(index)
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

	// Handle all parentheses normalization (adding required ones, removing unnecessary ones)
	expr = normalizeExpressionParentheses(expr)

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
	return definition
}

// normalizeFunction normalizes function signature and definition
func normalizeFunction(function *Function) {
	if function == nil {
		return
	}

	function.Signature = normalizeFunctionSignature(function.Signature)
	// lowercase LANGUAGE plpgsql is more common in modern usage
	function.Language = strings.ToLower(function.Language)
	// Normalize return type to handle PostgreSQL-specific formats
	function.ReturnType = normalizeFunctionReturnType(function.ReturnType)
	// Normalize parameter types
	for _, param := range function.Parameters {
		if param != nil {
			param.DataType = normalizePostgreSQLType(param.DataType)
		}
	}
}

// normalizeProcedure normalizes procedure representation
func normalizeProcedure(procedure *Procedure) {
	if procedure == nil {
		return
	}

	// Normalize language to lowercase (PLPGSQL → plpgsql)
	procedure.Language = strings.ToLower(procedure.Language)

	// Normalize arguments field when signature is present
	// Inspector provides: Arguments: "integer, text", Signature: "IN user_id integer, IN new_status text"
	// Parser provides: Arguments: "user_id integer, new_status text", no Signature
	// We need to make inspector match parser format
	if procedure.Signature != "" && procedure.Arguments != "" {
		// Extract parameter names and types from signature
		procedure.Arguments = normalizeProcedureArguments(procedure.Signature)
		// Clear signature as parser doesn't set it
		procedure.Signature = ""
	}
}

// normalizeProcedureArguments extracts parameter names and types from a procedure signature
func normalizeProcedureArguments(signature string) string {
	if signature == "" {
		return ""
	}

	// Parse signature like "IN user_id integer, IN new_status text"
	// to "user_id integer, new_status text"
	params := strings.Split(signature, ",")
	var normalizedParams []string

	for _, param := range params {
		param = strings.TrimSpace(param)
		if param == "" {
			continue
		}

		// Remove IN/OUT/INOUT modifiers
		param = regexp.MustCompile(`^(IN|OUT|INOUT)\s+`).ReplaceAllString(param, "")

		// Handle DEFAULT values - need to remove redundant type casts
		if strings.Contains(param, " DEFAULT ") {
			parts := strings.Split(param, " DEFAULT ")
			if len(parts) == 2 {
				// Parse the parameter name and type
				paramDef := strings.TrimSpace(parts[0])
				defaultValue := strings.TrimSpace(parts[1])

				// Remove redundant type casts from string literals
				// e.g., 'credit_card'::text -> 'credit_card'
				defaultValue = regexp.MustCompile(`'([^']+)'::text\b`).ReplaceAllString(defaultValue, "'$1'")

				param = paramDef + " DEFAULT " + defaultValue
			}
		}

		// Extract name and type
		fields := strings.Fields(param)
		if len(fields) >= 2 {
			// Check if this contains DEFAULT
			defaultIdx := -1
			for i, field := range fields {
				if field == "DEFAULT" {
					defaultIdx = i
					break
				}
			}

			if defaultIdx > 0 && defaultIdx >= 2 {
				// Format as "name type DEFAULT value"
				name := fields[0]
				typeStr := strings.Join(fields[1:defaultIdx], " ")
				defaultStr := strings.Join(fields[defaultIdx:], " ")
				normalizedParams = append(normalizedParams, name+" "+typeStr+" "+defaultStr)
			} else {
				// Format as "name type"
				normalizedParams = append(normalizedParams, fields[0]+" "+strings.Join(fields[1:], " "))
			}
		}
	}

	return strings.Join(normalizedParams, ", ")
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

// normalizeFunctionReturnType normalizes function return types, especially TABLE types
func normalizeFunctionReturnType(returnType string) string {
	if returnType == "" {
		return returnType
	}

	// Handle TABLE return types
	if strings.HasPrefix(returnType, "TABLE(") && strings.HasSuffix(returnType, ")") {
		// Extract the contents inside TABLE(...)
		inner := returnType[6 : len(returnType)-1] // Remove "TABLE(" and ")"

		// Split by comma to process each column definition
		parts := strings.Split(inner, ",")
		var normalizedParts []string

		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			// Normalize individual column definitions (name type)
			fields := strings.Fields(part)
			if len(fields) >= 2 {
				// Normalize the type part
				typePart := strings.Join(fields[1:], " ")
				normalizedType := normalizePostgreSQLType(typePart)
				normalizedParts = append(normalizedParts, fields[0]+" "+normalizedType)
			} else {
				// Just a type, normalize it
				normalizedParts = append(normalizedParts, normalizePostgreSQLType(part))
			}
		}

		return "TABLE(" + strings.Join(normalizedParts, ", ") + ")"
	}

	// For non-TABLE return types, apply regular type normalization
	return normalizePostgreSQLType(returnType)
}

// normalizeTrigger normalizes trigger representation
func normalizeTrigger(trigger *Trigger) {
	if trigger == nil {
		return
	}

	// Normalize trigger function call with the trigger's schema context
	trigger.Function = normalizeTriggerFunctionCall(trigger.Function, trigger.Schema)

	// Normalize trigger events to standard order: INSERT, UPDATE, DELETE, TRUNCATE
	trigger.Events = normalizeTriggerEvents(trigger.Events)

	// Normalize trigger condition (WHEN clause) for consistent comparison
	trigger.Condition = normalizeTriggerCondition(trigger.Condition)
}

// normalizeTriggerFunctionCall normalizes trigger function call syntax and removes same-schema qualifiers
func normalizeTriggerFunctionCall(functionCall string, triggerSchema string) string {
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

	// Strip schema qualifier if it matches the trigger's schema
	if triggerSchema != "" {
		schemaPrefix := triggerSchema + "."
		if strings.HasPrefix(functionCall, schemaPrefix) {
			functionCall = strings.TrimPrefix(functionCall, schemaPrefix)
		}
	}

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

// normalizeTriggerCondition normalizes trigger WHEN conditions for consistent comparison
func normalizeTriggerCondition(condition string) string {
	if condition == "" {
		return condition
	}

	// Remove extra whitespace
	condition = strings.TrimSpace(condition)
	condition = regexp.MustCompile(`\s+`).ReplaceAllString(condition, " ")

	// Normalize NEW and OLD identifiers to uppercase
	// Use word boundaries to ensure we only match the identifiers, not parts of other words
	condition = regexp.MustCompile(`\bnew\b`).ReplaceAllStringFunc(condition, func(match string) string {
		return strings.ToUpper(match)
	})
	condition = regexp.MustCompile(`\bold\b`).ReplaceAllStringFunc(condition, func(match string) string {
		return strings.ToUpper(match)
	})

	return condition
}

// normalizeIndex normalizes index WHERE clauses and other properties
func normalizeIndex(index *Index) {
	if index == nil {
		return
	}

	// Normalize WHERE clause for partial indexes
	if index.IsPartial && index.Where != "" {
		index.Where = normalizeIndexWhereClause(index.Where)
	}
}

// normalizeIndexWhereClause normalizes WHERE clauses in partial indexes
// It handles proper parentheses for different expression types
func normalizeIndexWhereClause(where string) string {
	if where == "" {
		return where
	}

	// Remove any existing outer parentheses to normalize the input
	normalizedWhere := strings.TrimSpace(where)
	if strings.HasPrefix(normalizedWhere, "(") && strings.HasSuffix(normalizedWhere, ")") {
		// Check if the parentheses wrap the entire expression
		inner := normalizedWhere[1 : len(normalizedWhere)-1]
		if isBalancedParentheses(inner) {
			normalizedWhere = inner
		}
	}

	// Determine if this expression needs outer parentheses based on its structure
	needsParentheses := shouldAddParenthesesForWhereClause(normalizedWhere)

	if needsParentheses {
		return fmt.Sprintf("(%s)", normalizedWhere)
	}

	return normalizedWhere
}

// shouldAddParenthesesForWhereClause determines if a WHERE clause needs outer parentheses
// Based on PostgreSQL's formatting expectations for pg_get_expr
func shouldAddParenthesesForWhereClause(expr string) bool {
	if expr == "" {
		return false
	}

	// Don't add parentheses for well-formed expressions that are self-contained:

	// 1. IN expressions: "column IN (value1, value2, value3)"
	if strings.Contains(expr, " IN (") {
		return false
	}

	// 2. Function calls: "function_name(args)"
	if matches, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*\s*\(.*\)$`, expr); matches {
		return false
	}

	// 3. Simple comparisons with parenthesized right side: "column = (value)"
	if matches, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*\s*[=<>!]+\s*\(.*\)$`, expr); matches {
		return false
	}

	// 4. Already fully parenthesized complex expressions
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		return false
	}

	// For other expressions (simple comparisons, AND/OR combinations, etc.), add parentheses
	return true
}

// normalizeExpressionParentheses handles parentheses normalization for policy expressions
// It ensures required parentheses for PostgreSQL DDL while removing unnecessary ones
func normalizeExpressionParentheses(expr string) string {
	if expr == "" {
		return expr
	}

	// Step 1: Ensure WITH CHECK/USING expressions are properly parenthesized
	// PostgreSQL requires parentheses around all policy expressions in DDL
	if !strings.HasPrefix(expr, "(") || !strings.HasSuffix(expr, ")") {
		expr = fmt.Sprintf("(%s)", expr)
	}

	// Step 2: Remove unnecessary parentheses around function calls within the expression
	// Specifically targets patterns like (function_name(...)) -> function_name(...)
	// This pattern looks for:
	// \( - opening parenthesis
	// ([a-zA-Z_][a-zA-Z0-9_]*) - function name (captured)
	// \( - opening parenthesis for function call
	// ([^)]*) - function arguments (captured, non-greedy to avoid matching nested parens)
	// \) - closing parenthesis for function call
	// \) - closing parenthesis around the whole function
	functionParensRegex := regexp.MustCompile(`\(([a-zA-Z_][a-zA-Z0-9_]*\([^)]*\))\)`)

	// Replace (function(...)) with function(...)
	// Keep applying until no more matches to handle nested cases
	for {
		original := expr
		expr = functionParensRegex.ReplaceAllString(expr, "$1")
		if expr == original {
			break
		}
	}

	// Step 3: Normalize redundant type casts in function arguments
	// Pattern: 'text'::text -> 'text' (removing redundant text cast from literals)
	redundantTextCastRegex := regexp.MustCompile(`'([^']+)'::text`)
	expr = redundantTextCastRegex.ReplaceAllString(expr, "'$1'")

	return expr
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
		"character varying":  "varchar", // Prefer short form
		"pg_catalog.text":    "text",
		"pg_catalog.varchar": "varchar", // Prefer short form
		"pg_catalog.bpchar":  "character",

		// Date/time types - convert verbose forms to canonical short forms
		"timestamp with time zone":    "timestamptz",
		"timestamp without time zone": "timestamp",
		"time with time zone":         "timetz",
		"timestamptz":                 "timestamptz",
		"timetz":                      "timetz",
		"pg_catalog.timestamptz":      "timestamptz",
		"pg_catalog.timestamp":        "timestamp",
		"pg_catalog.date":             "date",
		"pg_catalog.time":             "time",
		"pg_catalog.timetz":           "timetz",
		"pg_catalog.interval":         "interval",

		// Array types (internal PostgreSQL array notation)
		"_text":        "text[]",
		"_int2":        "smallint[]",
		"_int4":        "integer[]",
		"_int8":        "bigint[]",
		"_float4":      "real[]",
		"_float8":      "double precision[]",
		"_bool":        "boolean[]",
		"_varchar":     "varchar[]", // Prefer short form
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
