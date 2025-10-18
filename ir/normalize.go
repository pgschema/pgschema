package ir

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
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

	// Normalize types (including domains)
	for _, typeObj := range schema.Types {
		normalizeType(typeObj)
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

	// Normalize constraints
	for _, constraint := range table.Constraints {
		normalizeConstraint(constraint)
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

// normalizeDefaultValue normalizes default values for semantic comparison
func normalizeDefaultValue(value string) string {
	// Remove unnecessary whitespace
	value = strings.TrimSpace(value)

	// Handle nextval sequence references - remove schema qualification
	if strings.Contains(value, "nextval(") {
		// Pattern: nextval('schema_name.seq_name'::regclass) -> nextval('seq_name'::regclass)
		re := regexp.MustCompile(`nextval\('([^.]+)\.([^']+)'::regclass\)`)
		if re.MatchString(value) {
			// Replace with unqualified sequence name
			value = re.ReplaceAllString(value, "nextval('$2'::regclass)")
		}
		// Early return for nextval - don't apply type casting normalization
		return value
	}

	// Handle type casting - remove explicit type casts that are semantically equivalent
	// Use regex to properly handle type casts within complex expressions
	// Pattern: 'literal'::type -> 'literal' (removes redundant casts from string literals)
	if strings.Contains(value, "::") {
		// Use regex to match and remove type casts from string literals
		// This handles: 'text'::text, 'utc'::text, '{}'::jsonb, '{}'::text[], etc.
		// Also handles multi-word types like 'value'::character varying
		// Pattern explanation:
		// '([^']*)' - matches a quoted string literal (capturing the content)
		// ::[a-zA-Z_][\w\s.]* - matches ::typename
		//   [a-zA-Z_] - type name must start with letter or underscore
		//   [\w\s.]* - followed by word chars, spaces, or dots (for "character varying" or "pg_catalog.text")
		// (?:\[\])? - optionally followed by [] for array types (non-capturing group)
		// (?:\b|(?=\[)|$) - followed by word boundary, opening bracket, or end of string
		re := regexp.MustCompile(`'([^']*)'::(?:[a-zA-Z_][\w\s.]*)(?:\[\])?`)
		value = re.ReplaceAllString(value, "'$1'")
	}

	return value
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
		// Keep PUBLIC in uppercase, normalize others to lowercase
		if strings.ToUpper(role) == "PUBLIC" {
			normalized[i] = "PUBLIC"
		} else {
			normalized[i] = strings.ToLower(role)
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
// across different PostgreSQL versions.
//
// PostgreSQL versions produce different pg_get_viewdef() output:
// - PostgreSQL 15: Includes table qualifiers → "dept_emp.emp_no, max(dept_emp.from_date)"
// - PostgreSQL 16+: Omits unnecessary qualifiers → "emp_no, max(from_date)"
//
// This function removes unnecessary table qualifiers from column references when unambiguous
// to ensure consistent comparison between Inspector (database) and Parser (SQL files).
func normalizeViewDefinition(definition string) string {
	if definition == "" {
		return definition
	}

	// Parse the view definition to get AST and remove unnecessary table qualifiers
	normalized, err := removeUnnecessaryTableQualifiers(definition)
	if err != nil {
		// If parsing fails, use the original definition
		normalized = definition
	}

	// Apply all AST-based normalizations in one pass to avoid re-parsing
	// This includes:
	// 1. Converting PostgreSQL's "= ANY (ARRAY[...])" to "IN (...)"
	// 2. Normalizing ORDER BY clauses to use aliases
	normalized = normalizeViewWithAST(normalized)

	return normalized
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
		functionCall = strings.TrimPrefix(functionCall, schemaPrefix)
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

	// Normalize whitespace
	condition = strings.TrimSpace(condition)
	condition = regexp.MustCompile(`\s+`).ReplaceAllString(condition, " ")

	// Normalize NEW and OLD identifiers to uppercase
	condition = regexp.MustCompile(`\bnew\b`).ReplaceAllStringFunc(condition, func(match string) string {
		return strings.ToUpper(match)
	})
	condition = regexp.MustCompile(`\bold\b`).ReplaceAllStringFunc(condition, func(match string) string {
		return strings.ToUpper(match)
	})

	// PostgreSQL stores "IS NOT DISTINCT FROM" as "NOT (... IS DISTINCT FROM ...)"
	// Convert the internal form to the SQL standard form for consistency
	// Pattern: NOT (expr IS DISTINCT FROM expr) -> expr IS NOT DISTINCT FROM expr
	re := regexp.MustCompile(`NOT \((.+?)\s+IS\s+DISTINCT\s+FROM\s+(.+?)\)`)
	condition = re.ReplaceAllString(condition, "$1 IS NOT DISTINCT FROM $2")

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
	if strings.HasPrefix(where, "(") && strings.HasSuffix(where, ")") {
		// Check if the parentheses wrap the entire expression
		inner := where[1 : len(where)-1]
		if isBalancedParentheses(inner) {
			where = inner
		}
	}

	// Convert PostgreSQL's "= ANY (ARRAY[...])" format to "IN (...)" format
	where = convertAnyArrayToIn(where)

	// Determine if this expression needs outer parentheses based on its structure
	needsParentheses := shouldAddParenthesesForWhereClause(where)

	if needsParentheses {
		return fmt.Sprintf("(%s)", where)
	}

	return where
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

// isBalancedParentheses checks if parentheses are properly balanced in the expression
func isBalancedParentheses(expr string) bool {
	count := 0
	inQuotes := false
	var quoteChar rune

	for _, r := range expr {
		if !inQuotes {
			switch r {
			case '\'', '"':
				inQuotes = true
				quoteChar = r
			case '(':
				count++
			case ')':
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

// normalizeType normalizes type-related objects, including domain constraints
func normalizeType(typeObj *Type) {
	if typeObj == nil || typeObj.Kind != TypeKindDomain {
		return
	}

	// Normalize domain default value
	if typeObj.Default != "" {
		typeObj.Default = normalizeDomainDefault(typeObj.Default)
	}

	// Normalize domain constraints
	for _, constraint := range typeObj.Constraints {
		normalizeDomainConstraint(constraint)
	}
}

// normalizeDomainDefault normalizes domain default values
func normalizeDomainDefault(defaultValue string) string {
	if defaultValue == "" {
		return defaultValue
	}

	// Remove redundant type casts from string literals
	// e.g., 'example@acme.com'::text -> 'example@acme.com'
	defaultValue = regexp.MustCompile(`'([^']+)'::text\b`).ReplaceAllString(defaultValue, "'$1'")

	return defaultValue
}

// normalizeDomainConstraint normalizes domain constraint definitions
func normalizeDomainConstraint(constraint *DomainConstraint) {
	if constraint == nil || constraint.Definition == "" {
		return
	}

	def := constraint.Definition

	// Normalize VALUE keyword to uppercase in domain constraints
	// Use word boundaries to ensure we only match the identifier, not parts of other words
	def = regexp.MustCompile(`\bvalue\b`).ReplaceAllStringFunc(def, func(match string) string {
		return strings.ToUpper(match)
	})

	// Handle CHECK constraints
	if strings.HasPrefix(def, "CHECK ") {
		// Extract the expression inside CHECK (...)
		checkMatch := regexp.MustCompile(`^CHECK\s*\((.*)\)$`).FindStringSubmatch(def)
		if len(checkMatch) > 1 {
			expr := checkMatch[1]

			// Remove outer parentheses if they wrap the entire expression
			expr = strings.TrimSpace(expr)
			if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
				inner := expr[1 : len(expr)-1]
				if isBalancedParentheses(inner) {
					expr = inner
				}
			}

			// Remove redundant type casts
			// e.g., '...'::text -> '...'
			expr = regexp.MustCompile(`'([^']+)'::text\b`).ReplaceAllString(expr, "'$1'")

			// Reconstruct the CHECK constraint
			def = fmt.Sprintf("CHECK (%s)", expr)
		}
	}

	constraint.Definition = def
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

// normalizeConstraint normalizes constraint definitions from inspector format to parser format
func normalizeConstraint(constraint *Constraint) {
	if constraint == nil {
		return
	}

	// Only normalize CHECK constraints - other constraint types are already consistent
	if constraint.Type == ConstraintTypeCheck && constraint.CheckClause != "" {
		constraint.CheckClause = normalizeCheckClause(constraint.CheckClause)
	}
}

// normalizeCheckClause converts PostgreSQL's normalized CHECK expressions to parser format
// Uses pg_query to parse and deparse for consistent normalization
func normalizeCheckClause(checkClause string) string {
	// Strip " NOT VALID" suffix if present (mimicking pg_dump behavior)
	// PostgreSQL's pg_get_constraintdef may include NOT VALID at the end,
	// but we want to control its placement via the IsValid field
	clause := strings.TrimSpace(checkClause)
	if strings.HasSuffix(clause, " NOT VALID") {
		clause = strings.TrimSuffix(clause, " NOT VALID")
		clause = strings.TrimSpace(clause)
	}

	// Remove "CHECK " prefix if present
	if after, found := strings.CutPrefix(clause, "CHECK "); found {
		clause = after
	}

	// Remove outer parentheses - pg_get_constraintdef wraps in parentheses
	clause = strings.TrimSpace(clause)
	if len(clause) > 0 && clause[0] == '(' && clause[len(clause)-1] == ')' {
		if isBalancedParentheses(clause[1 : len(clause)-1]) {
			clause = clause[1 : len(clause)-1]
			clause = strings.TrimSpace(clause)
		}
	}

	// Apply legacy normalizations for PostgreSQL-specific patterns
	normalizedClause := applyLegacyCheckNormalizations(clause)

	// Try to normalize using pg_query parse/deparse for consistent formatting
	pgNormalizedClause := normalizeExpressionWithPgQuery(normalizedClause)
	if pgNormalizedClause != "" {
		return fmt.Sprintf("CHECK (%s)", pgNormalizedClause)
	}

	// Fallback to legacy normalization result if pg_query fails
	return fmt.Sprintf("CHECK (%s)", normalizedClause)
}

// normalizeExpressionWithPgQuery normalizes an expression using PostgreSQL's parser
func normalizeExpressionWithPgQuery(expr string) string {
	// Create a dummy SELECT statement with the expression to parse it
	dummySQL := fmt.Sprintf("SELECT %s", expr)

	parseResult, err := pg_query.Parse(dummySQL)
	if err != nil {
		// If parsing fails, return empty string to trigger fallback
		return ""
	}

	// Deparse to get normalized form
	deparsed, err := pg_query.Deparse(parseResult)
	if err != nil {
		return ""
	}

	// Extract the expression from "SELECT expr" format
	if after, found := strings.CutPrefix(deparsed, "SELECT "); found {
		normalized := strings.TrimSpace(after)
		// Remove redundant numeric type casts from literals
		normalized = removeRedundantNumericCasts(normalized)
		return normalized
	}

	return ""
}

// removeRedundantNumericCasts removes type casts from numeric literals
// e.g., "0::numeric" -> "0", "123::integer" -> "123"
func removeRedundantNumericCasts(expr string) string {
	// Pattern: number::numeric_type -> number
	// This handles: 0::numeric, 123::integer, 45.67::numeric, etc.
	patterns := []string{
		`(\d+(?:\.\d+)?)::numeric\b`,
		`(\d+)::integer\b`,
		`(\d+)::bigint\b`,
		`(\d+)::smallint\b`,
		`(\d+(?:\.\d+)?)::decimal\b`,
		`(\d+(?:\.\d+)?)::real\b`,
		`(\d+(?:\.\d+)?)::double\s+precision\b`,
	}

	result := expr
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, "$1")
	}

	return result
}

// applyLegacyCheckNormalizations applies the existing normalization patterns
func applyLegacyCheckNormalizations(clause string) string {
	// Convert PostgreSQL's "= ANY (ARRAY[...])" format to "IN (...)" format
	if strings.Contains(clause, "= ANY (ARRAY[") {
		return convertAnyArrayToIn(clause)
	}

	// Convert "column ~~ 'pattern'::text" to "column LIKE 'pattern'"
	if strings.Contains(clause, " ~~ ") {
		parts := strings.Split(clause, " ~~ ")
		if len(parts) == 2 {
			columnName := strings.TrimSpace(parts[0])
			pattern := strings.TrimSpace(parts[1])
			// Remove type cast
			if idx := strings.Index(pattern, "::"); idx != -1 {
				pattern = pattern[:idx]
			}
			return fmt.Sprintf("%s LIKE %s", columnName, pattern)
		}
	}

	return clause
}

// removeUnnecessaryTableQualifiers removes table qualifiers from column references
// when they are unambiguous (i.e., when there's only one table in the FROM clause)
func removeUnnecessaryTableQualifiers(definition string) (string, error) {
	// Parse the SQL definition to validate and extract table information
	parseResult, err := pg_query.Parse(definition)
	if err != nil {
		return definition, err
	}

	if len(parseResult.Stmts) == 0 {
		return definition, fmt.Errorf("no statements found")
	}

	// Get the first statement (should be a SELECT)
	stmt := parseResult.Stmts[0]
	selectStmt := stmt.Stmt.GetSelectStmt()
	if selectStmt == nil {
		return definition, fmt.Errorf("not a SELECT statement")
	}

	// Extract table names from FROM clause
	tables := extractTablesFromFromClause(selectStmt.FromClause)

	// If there's more than one table, keep qualifiers as they might be necessary
	if len(tables) != 1 {
		return definition, fmt.Errorf("multiple tables found, keeping original")
	}

	tableName := tables[0]

	// Use regex-based replacement to preserve formatting while removing qualifiers
	// This approach maintains the original PostgreSQL pretty-printing format
	qualifierRegex := regexp.MustCompile(`\b` + regexp.QuoteMeta(tableName) + `\.([a-zA-Z_][a-zA-Z0-9_]*)\b`)
	normalized := qualifierRegex.ReplaceAllString(definition, "$1")

	return normalized, nil
}

// extractTablesFromFromClause extracts table names or aliases from the FROM clause
func extractTablesFromFromClause(fromClause []*pg_query.Node) []string {
	var tables []string

	for _, fromItem := range fromClause {
		if rangeVar := fromItem.GetRangeVar(); rangeVar != nil {
			if rangeVar.Relname != "" {
				// Use alias if present, otherwise use the table name
				if rangeVar.Alias != nil && rangeVar.Alias.Aliasname != "" {
					tables = append(tables, rangeVar.Alias.Aliasname)
				} else {
					tables = append(tables, rangeVar.Relname)
				}
			}
		}
		// TODO: Handle other FROM clause types like JOINs, subqueries, etc.
		// For now, we only handle simple table references
	}

	return tables
}

// convertAnyArrayToIn converts PostgreSQL's "column = ANY (ARRAY[...])" format
// to the more readable "column IN (...)" format
func convertAnyArrayToIn(expr string) string {
	if !strings.Contains(expr, "= ANY (ARRAY[") {
		return expr
	}

	// Extract the column name and values
	parts := strings.Split(expr, " = ANY (ARRAY[")
	if len(parts) != 2 {
		return expr
	}

	columnName := strings.TrimSpace(parts[0])

	// Remove the closing parentheses and brackets
	valuesPart := parts[1]
	valuesPart = strings.TrimSuffix(valuesPart, "])")
	valuesPart = strings.TrimSuffix(valuesPart, "])) ")
	valuesPart = strings.TrimSuffix(valuesPart, "]))")
	valuesPart = strings.TrimSuffix(valuesPart, "])")

	// Split the values and clean them up
	values := strings.Split(valuesPart, ", ")
	var cleanValues []string
	for _, val := range values {
		val = strings.TrimSpace(val)
		// Remove type casts like ::text, ::varchar, etc.
		if idx := strings.Index(val, "::"); idx != -1 {
			val = val[:idx]
		}
		cleanValues = append(cleanValues, val)
	}

	// Return converted format: "column IN ('val1', 'val2')"
	return fmt.Sprintf("%s IN (%s)", columnName, strings.Join(cleanValues, ", "))
}

// normalizeViewWithAST applies all AST-based normalizations in a single pass
// This includes converting "= ANY (ARRAY[...])" to "IN (...)" and normalizing ORDER BY
func normalizeViewWithAST(definition string) string {
	if definition == "" {
		return definition
	}

	// Parse the view definition
	parseResult, err := pg_query.Parse(definition)
	if err != nil {
		return definition
	}

	if len(parseResult.Stmts) == 0 {
		return definition
	}

	stmt := parseResult.Stmts[0]
	selectStmt := stmt.Stmt.GetSelectStmt()
	if selectStmt == nil {
		return definition
	}

	// Step 1: Normalize ORDER BY clauses (modify AST if needed)
	orderByModified := false
	if len(selectStmt.SortClause) > 0 {
		// Build reverse alias map (expression -> alias) from target list
		exprToAliasMap := buildExpressionToAliasMap(selectStmt.TargetList)

		// Transform ORDER BY clauses: replace complex expressions with aliases when possible
		for _, sortItem := range selectStmt.SortClause {
			if sortBy := sortItem.GetSortBy(); sortBy != nil {
				if wasModified := normalizeOrderByExpressionToAlias(sortBy, exprToAliasMap); wasModified {
					orderByModified = true
				}
			}
		}
	}

	// Step 2: Check if we need to use custom formatter
	// Use custom formatter if:
	// a) The view definition contains "= ANY" (needs conversion to IN)
	// b) ORDER BY was modified
	needsCustomFormatter := strings.Contains(definition, "= ANY") || orderByModified

	if needsCustomFormatter {
		// Use custom formatter to format the entire query
		// The formatter will handle:
		// - Converting "= ANY (ARRAY[...])" to "IN (...)"
		// - Proper formatting of all expressions
		formatter := newPostgreSQLFormatter()
		formatted := formatter.formatQueryNode(stmt.Stmt)
		if formatted != "" {
			return formatted
		}
	}

	return definition
}

// normalizeOrderByInView normalizes ORDER BY clauses in view definitions
// This converts PostgreSQL's pg_get_viewdef format (with parentheses and expressions)
// back to parser format (using column aliases) for consistent comparison
// Uses AST manipulation for robustness
func normalizeOrderByInView(definition string) string {
	if definition == "" {
		return definition
	}

	// Parse the view definition
	parseResult, err := pg_query.Parse(definition)
	if err != nil {
		return definition
	}

	if len(parseResult.Stmts) == 0 {
		return definition
	}

	stmt := parseResult.Stmts[0]
	selectStmt := stmt.Stmt.GetSelectStmt()
	if selectStmt == nil || len(selectStmt.SortClause) == 0 {
		return definition
	}

	// Build reverse alias map (expression -> alias) from target list
	// This helps us convert ORDER BY expressions back to aliases
	exprToAliasMap := buildExpressionToAliasMap(selectStmt.TargetList)

	// Transform ORDER BY clauses: replace complex expressions with aliases when possible
	modified := false
	for _, sortItem := range selectStmt.SortClause {
		if sortBy := sortItem.GetSortBy(); sortBy != nil {
			if wasModified := normalizeOrderByExpressionToAlias(sortBy, exprToAliasMap); wasModified {
				modified = true
			}
		}
	}

	// If we made modifications, use PostgreSQL formatter to maintain formatting
	// IMPORTANT: Use the custom formatter to preserve ANY->IN conversions done earlier
	if modified {
		formatter := newPostgreSQLFormatter()
		formatted := formatter.formatQueryNode(stmt.Stmt)
		if formatted != "" {
			return formatted
		}
	}

	return definition
}

// buildExpressionToAliasMap creates a map from expression fingerprints to their aliases
// This helps convert ORDER BY expressions back to column aliases
func buildExpressionToAliasMap(targetList []*pg_query.Node) map[string]string {
	exprToAlias := make(map[string]string)

	for _, target := range targetList {
		if resTarget := target.GetResTarget(); resTarget != nil && resTarget.Name != "" && resTarget.Val != nil {
			// Create a fingerprint of the expression by deparsing it
			if fingerprint := getExpressionFingerprint(resTarget.Val); fingerprint != "" {
				exprToAlias[fingerprint] = resTarget.Name
			}
		}
	}

	return exprToAlias
}

// normalizeOrderByExpressionToAlias converts ORDER BY expressions back to aliases when possible
// Returns true if the expression was modified
func normalizeOrderByExpressionToAlias(sortBy *pg_query.SortBy, exprToAliasMap map[string]string) bool {
	if sortBy.Node == nil {
		return false
	}

	// Get the fingerprint of the current ORDER BY expression
	fingerprint := getExpressionFingerprint(sortBy.Node)
	if fingerprint == "" {
		return false
	}

	// Check if this expression matches one of our aliased expressions
	if alias, exists := exprToAliasMap[fingerprint]; exists {
		// Replace the complex expression with a simple ColumnRef to the alias
		sortBy.Node = &pg_query.Node{
			Node: &pg_query.Node_ColumnRef{
				ColumnRef: &pg_query.ColumnRef{
					Fields: []*pg_query.Node{{
						Node: &pg_query.Node_String_{
							String_: &pg_query.String{Sval: alias},
						},
					}},
				},
			},
		}
		return true
	}

	return false
}

// getExpressionFingerprint creates a normalized fingerprint of an expression
// This is used to match expressions between SELECT list and ORDER BY
func getExpressionFingerprint(expr *pg_query.Node) string {
	if expr == nil {
		return ""
	}

	// Create a temporary SELECT statement with just this expression to deparse it
	tempSelect := &pg_query.SelectStmt{
		TargetList: []*pg_query.Node{{
			Node: &pg_query.Node_ResTarget{
				ResTarget: &pg_query.ResTarget{Val: expr},
			},
		}},
	}
	tempResult := &pg_query.ParseResult{
		Stmts: []*pg_query.RawStmt{{
			Stmt: &pg_query.Node{
				Node: &pg_query.Node_SelectStmt{SelectStmt: tempSelect},
			},
		}},
	}

	if deparsed, err := pg_query.Deparse(tempResult); err == nil {
		// Extract just the expression part from "SELECT expression"
		if expr, found := strings.CutPrefix(deparsed, "SELECT "); found {
			// Normalize the fingerprint by removing extra whitespace and lowercasing
			return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(expr), " ", ""))
		}
	}

	return ""
}
