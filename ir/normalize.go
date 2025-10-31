package ir

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// normalizeIR normalizes the IR representation from the inspector.
//
// Since both desired state (from embedded postgres) and current state (from target database)
// now come from the same PostgreSQL version via database inspection, most normalizations
// are no longer needed. The remaining normalizations handle:
//
// - Type name mappings (internal PostgreSQL types → standard SQL types, e.g., int4 → integer)
// - PostgreSQL internal representations (e.g., "~~ " → "LIKE", "= ANY (ARRAY[...])" → "IN (...)")
// - Minor formatting differences in default values, policies, triggers, etc.
// - View definition normalization for PostgreSQL 14-15 (table-qualified column names)
func normalizeIR(ir *IR) {
	if ir == nil {
		return
	}

	// Extract PostgreSQL major version from metadata
	pgMajorVersion := extractPostgreSQLMajorVersion(ir.Metadata.DatabaseVersion)

	for _, schema := range ir.Schemas {
		normalizeSchema(schema, pgMajorVersion)
	}
}

// extractPostgreSQLMajorVersion extracts the major version number from a PostgreSQL version string
// Examples: "PostgreSQL 14.18" -> 14, "PostgreSQL 15.13" -> 15, "PostgreSQL 17.5" -> 17
func extractPostgreSQLMajorVersion(versionString string) int {
	// Expected format: "PostgreSQL X.Y" or "PostgreSQL X.Y.Z"
	if !strings.Contains(versionString, "PostgreSQL") {
		return 0 // Unknown version
	}

	parts := strings.Fields(versionString)
	if len(parts) < 2 {
		return 0
	}

	versionPart := parts[1]
	dotIndex := strings.Index(versionPart, ".")
	if dotIndex == -1 {
		return 0
	}

	majorStr := versionPart[:dotIndex]
	major := 0
	fmt.Sscanf(majorStr, "%d", &major)
	return major
}

// normalizeSchema normalizes all objects within a schema
func normalizeSchema(schema *Schema, pgMajorVersion int) {
	if schema == nil {
		return
	}

	// Normalize tables
	for _, table := range schema.Tables {
		normalizeTable(table)
	}

	// Normalize views
	for _, view := range schema.Views {
		normalizeView(view, pgMajorVersion)
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
	if strings.Contains(value, "::") {
		// Handle NULL::type -> NULL
		// Example: NULL::text -> NULL
		re := regexp.MustCompile(`\bNULL::(?:[a-zA-Z_][\w\s.]*)(?:\[\])?`)
		value = re.ReplaceAllString(value, "NULL")

		// Handle numeric literals with type casts
		// Example: '-1'::integer -> -1
		// Example: '100'::bigint -> 100
		// Note: PostgreSQL sometimes casts numeric literals to different types, e.g., -1::integer stored as numeric
		re = regexp.MustCompile(`'(-?\d+(?:\.\d+)?)'::(?:integer|bigint|smallint|numeric|decimal|real|double precision|int2|int4|int8|float4|float8)`)
		value = re.ReplaceAllString(value, "$1")

		// Handle string literals with type casts (including escaped quotes)
		// Example: 'text'::text -> 'text'
		// Example: 'O''Brien'::text -> 'O''Brien'
		// Example: '{}'::jsonb -> '{}'
		// Example: '{1,2,3}'::integer[] -> '{1,2,3}'
		// Pattern explanation:
		// '(?:[^']|'')*' - matches a quoted string literal, handling escaped quotes ''
		// ::[a-zA-Z_][\w\s.]* - matches ::typename
		// (?:\[\])? - optionally followed by [] for array types
		re = regexp.MustCompile(`('(?:[^']|'')*')::(?:[a-zA-Z_][\w\s.]*)(?:\[\])?`)
		value = re.ReplaceAllString(value, "$1")

		// Handle date/timestamp literals with type casts
		// Example: '2024-01-01'::date -> '2024-01-01'
		// Already handled by the string literal pattern above

		// Handle parenthesized expressions with type casts - remove outer parentheses
		// Example: (100)::bigint -> 100::bigint
		// Pattern captures the number and the type cast separately
		re = regexp.MustCompile(`\((\d+)\)(::(?:bigint|integer|smallint|numeric|decimal))`)
		value = re.ReplaceAllString(value, "$1$2")
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

// normalizeView normalizes view definition.
//
// For PostgreSQL 14-15: pg_get_viewdef() returns table-qualified column names when views
// are created with schema-qualified table references. We normalize by removing these
// table qualifications to match the simplified output from PostgreSQL 16+.
//
// For PostgreSQL 16+: No normalization needed as pg_get_viewdef() already returns
// simplified column references.
func normalizeView(view *View, pgMajorVersion int) {
	if view == nil {
		return
	}

	// Only normalize for PostgreSQL 14 and 15
	if pgMajorVersion == 14 || pgMajorVersion == 15 {
		view.Definition = normalizeViewDefinitionPG14_15(view.Definition)
	}
}

// normalizeViewDefinitionPG14_15 removes table qualifications from column references
// in PostgreSQL 14-15 view definitions.
//
// In PG 14-15, when views are created with schema-qualified table references
// (e.g., FROM public.dept_emp), pg_get_viewdef() returns table-qualified column names
// (e.g., dept_emp.emp_no). In PG 16+, these are simplified to just the column name.
//
// This function normalizes PG 14-15 output to match PG 16+ format:
//   Before: SELECT dept_emp.emp_no, max(dept_emp.from_date) ... GROUP BY dept_emp.emp_no
//   After:  SELECT emp_no, max(from_date) ... GROUP BY emp_no
//
// Important: Alias qualifications (e.g., l.emp_no where l is an alias) are preserved.
func normalizeViewDefinitionPG14_15(definition string) string {
	// Extract table names from FROM/JOIN clauses to identify which qualifiers to remove
	// Pattern matches: FROM table_name or JOIN table_name or FROM schema.table_name
	tableNames := extractTableNamesFromView(definition)

	// For each table name, remove qualifications like table_name.column_name
	result := definition
	for _, tableName := range tableNames {
		// Match table_name.column_name but not preceded by a dot (to avoid schema.table_name.column)
		// Pattern: (non-dot or start) + tableName + dot + identifier
		pattern := regexp.MustCompile(`([^.]|^)(` + regexp.QuoteMeta(tableName) + `)\.([a-zA-Z_][a-zA-Z0-9_]*|"[^"]+")`)
		result = pattern.ReplaceAllString(result, "${1}${3}")
	}

	return result
}

// extractTableNamesFromView extracts table names (without aliases) from FROM and JOIN clauses
func extractTableNamesFromView(definition string) []string {
	var tableNames []string

	// Pattern for FROM/JOIN clauses: FROM|JOIN [schema.]table_name [alias]
	// Captures the table name (without schema prefix, without alias)
	// Examples:
	//   FROM dept_emp -> dept_emp
	//   FROM public.dept_emp -> dept_emp
	//   FROM dept_emp d -> dept_emp
	//   JOIN dept_emp_latest_date l -> dept_emp_latest_date
	//
	// Updated pattern to make schema part truly optional with (?:...)?
	fromPattern := regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+(?:(?:[a-zA-Z_][a-zA-Z0-9_]*|"[^"]+")\.)?([a-zA-Z_][a-zA-Z0-9_]*|"[^"]+")\b`)

	matches := fromPattern.FindAllStringSubmatch(definition, -1)
	for _, match := range matches {
		if len(match) > 1 {
			tableName := match[1]
			// Remove quotes if present
			tableName = strings.Trim(tableName, "\"")
			// Only add if not already in the list
			found := false
			for _, existing := range tableNames {
				if existing == tableName {
					found = true
					break
				}
			}
			if !found {
				tableNames = append(tableNames, tableName)
			}
		}
	}

	return tableNames
}

// normalizeFunction normalizes function signature and definition
func normalizeFunction(function *Function) {
	if function == nil {
		return
	}

	// lowercase LANGUAGE plpgsql is more common in modern usage
	function.Language = strings.ToLower(function.Language)
	// Normalize return type to handle PostgreSQL-specific formats
	function.ReturnType = normalizeFunctionReturnType(function.ReturnType)
	// Normalize parameter types, modes, and default values
	for _, param := range function.Parameters {
		if param != nil {
			param.DataType = normalizePostgreSQLType(param.DataType)
			// Normalize mode: empty string → "IN" for functions (PostgreSQL default)
			// Functions: IN is default, only OUT/INOUT/VARIADIC need explicit mode
			// But for consistent comparison, normalize empty to "IN"
			if param.Mode == "" {
				param.Mode = "IN"
			}
			// Normalize default values
			if param.DefaultValue != nil {
				normalized := normalizeDefaultValue(*param.DefaultValue)
				param.DefaultValue = &normalized
			}
		}
	}
	// Normalize function body to handle whitespace differences
	function.Definition = normalizeFunctionDefinition(function.Definition)
}

// normalizeFunctionDefinition normalizes function body whitespace
// PostgreSQL stores function bodies with specific whitespace that may differ from source
func normalizeFunctionDefinition(def string) string {
	if def == "" {
		return def
	}

	// Only trim trailing whitespace from each line, preserving the line structure
	// This ensures leading/trailing blank lines are preserved (matching PostgreSQL storage)
	lines := strings.Split(def, "\n")
	var normalized []string
	for _, line := range lines {
		// Trim all trailing whitespace (spaces, tabs, CR) but preserve leading whitespace for indentation
		normalized = append(normalized, strings.TrimRightFunc(line, unicode.IsSpace))
	}

	return strings.Join(normalized, "\n")
}

// normalizeProcedure normalizes procedure representation
func normalizeProcedure(procedure *Procedure) {
	if procedure == nil {
		return
	}

	// Normalize language to lowercase (PLPGSQL → plpgsql)
	procedure.Language = strings.ToLower(procedure.Language)

	// Normalize parameter types, modes, and default values
	for _, param := range procedure.Parameters {
		if param != nil {
			param.DataType = normalizePostgreSQLType(param.DataType)
			// Normalize mode: empty string → "IN" for procedures (PostgreSQL default)
			if param.Mode == "" {
				param.Mode = "IN"
			}
			// Normalize default values
			if param.DefaultValue != nil {
				normalized := normalizeDefaultValue(*param.DefaultValue)
				param.DefaultValue = &normalized
			}
		}
	}
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

// normalizeCheckClause normalizes CHECK constraint expressions.
//
// Since both desired state (from embedded postgres) and current state (from target database)
// now come from the same PostgreSQL version via pg_get_constraintdef(), they produce identical
// output. We only need basic cleanup for PostgreSQL internal representations.
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

	// Apply basic normalizations for PostgreSQL internal representations
	// (e.g., "~~ " to "LIKE", "= ANY (ARRAY[...])" to "IN (...)")
	normalizedClause := applyLegacyCheckNormalizations(clause)

	return fmt.Sprintf("CHECK (%s)", normalizedClause)
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

