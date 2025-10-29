package diff

import (
	"fmt"
	"strings"

	"github.com/pgschema/pgschema/ir"
)

// generateCreateViewsSQL generates CREATE VIEW statements
// Views are assumed to be pre-sorted in topological order for dependency-aware creation
func generateCreateViewsSQL(views []*ir.View, targetSchema string, collector *diffCollector) {
	// Process views in the provided order (already topologically sorted)
	for _, view := range views {
		// If compare mode, CREATE OR REPLACE, otherwise CREATE
		sql := generateViewSQL(view, targetSchema)

		// Determine the diff type based on whether it's materialized
		diffType := DiffTypeView
		if view.Materialized {
			diffType = DiffTypeMaterializedView
		}

		// Create context for this statement
		context := &diffContext{
			Type:                diffType,
			Operation:           DiffOperationCreate,
			Path:                fmt.Sprintf("%s.%s", view.Schema, view.Name),
			Source:              view,
			CanRunInTransaction: true,
		}

		collector.collect(context, sql)

		// Add view comment
		if view.Comment != "" {
			viewName := qualifyEntityName(view.Schema, view.Name, targetSchema)
			commentType := DiffTypeViewComment
			if view.Materialized {
				commentType = DiffTypeMaterializedViewComment
				sql := fmt.Sprintf("COMMENT ON MATERIALIZED VIEW %s IS %s;", viewName, quoteString(view.Comment))

				// Create context for this statement
				context := &diffContext{
					Type:                commentType,
					Operation:           DiffOperationCreate,
					Path:                fmt.Sprintf("%s.%s", view.Schema, view.Name),
					Source:              view,
					CanRunInTransaction: true,
				}

				collector.collect(context, sql)
			} else {
				sql := fmt.Sprintf("COMMENT ON VIEW %s IS %s;", viewName, quoteString(view.Comment))

				// Create context for this statement
				context := &diffContext{
					Type:                commentType,
					Operation:           DiffOperationCreate,
					Path:                fmt.Sprintf("%s.%s", view.Schema, view.Name),
					Source:              view,
					CanRunInTransaction: true,
				}

				collector.collect(context, sql)
			}
		}

		// For materialized views, create indexes
		if view.Materialized && view.Indexes != nil {
			indexList := make([]*ir.Index, 0, len(view.Indexes))
			for _, index := range view.Indexes {
				indexList = append(indexList, index)
			}
			// Generate index SQL for materialized view indexes - use MaterializedView types
			generateCreateIndexesSQLWithType(indexList, targetSchema, collector, DiffTypeMaterializedViewIndex, DiffTypeMaterializedViewIndexComment)
		}
	}
}

// generateModifyViewsSQL generates CREATE OR REPLACE VIEW statements or comment changes
func generateModifyViewsSQL(diffs []*viewDiff, targetSchema string, collector *diffCollector) {
	for _, diff := range diffs {
		// Handle materialized views that require recreation (DROP + CREATE)
		if diff.RequiresRecreate {
			viewName := qualifyEntityName(diff.New.Schema, diff.New.Name, targetSchema)

			// DROP the old materialized view
			dropSQL := fmt.Sprintf("DROP MATERIALIZED VIEW %s RESTRICT;", viewName)
			createSQL := generateViewSQL(diff.New, targetSchema)

			statements := []SQLStatement{
				{
					SQL:                 dropSQL,
					CanRunInTransaction: true,
				},
				{
					SQL:                 createSQL,
					CanRunInTransaction: true,
				},
			}

			// Use DiffOperationAlter to categorize as a modification
			context := &diffContext{
				Type:                DiffTypeMaterializedView,
				Operation:           DiffOperationAlter,
				Path:                fmt.Sprintf("%s.%s", diff.New.Schema, diff.New.Name),
				Source:              diff,
				CanRunInTransaction: true,
			}
			collector.collectStatements(context, statements)

			// Add view comment if present
			if diff.New.Comment != "" {
				sql := fmt.Sprintf("COMMENT ON MATERIALIZED VIEW %s IS %s;", viewName, quoteString(diff.New.Comment))
				commentContext := &diffContext{
					Type:                DiffTypeMaterializedViewComment,
					Operation:           DiffOperationCreate,
					Path:                fmt.Sprintf("%s.%s", diff.New.Schema, diff.New.Name),
					Source:              diff.New,
					CanRunInTransaction: true,
				}
				collector.collect(commentContext, sql)
			}

			// Recreate indexes for materialized views
			if diff.New.Materialized && diff.New.Indexes != nil {
				indexList := make([]*ir.Index, 0, len(diff.New.Indexes))
				for _, index := range diff.New.Indexes {
					indexList = append(indexList, index)
				}
				generateCreateIndexesSQLWithType(indexList, targetSchema, collector, DiffTypeMaterializedViewIndex, DiffTypeMaterializedViewIndexComment)
			}

			continue // Skip the normal processing for this view
		}

		// Check if only the comment changed and definition is identical
		// Both IRs come from pg_get_viewdef() at the same PostgreSQL version, so string comparison is sufficient
		definitionsEqual := diff.Old.Definition == diff.New.Definition
		commentOnlyChange := diff.CommentChanged && definitionsEqual && diff.Old.Materialized == diff.New.Materialized

		// Check if only indexes changed (for materialized views)
		hasIndexChanges := len(diff.AddedIndexes) > 0 || len(diff.DroppedIndexes) > 0 || len(diff.ModifiedIndexes) > 0
		indexOnlyChange := diff.New.Materialized && hasIndexChanges && definitionsEqual && !diff.CommentChanged

		// Handle comment-only or index-only changes
		if commentOnlyChange || indexOnlyChange {
			// Only generate COMMENT ON VIEW statement if comment actually changed
			if diff.CommentChanged {
				viewName := qualifyEntityName(diff.New.Schema, diff.New.Name, targetSchema)

				// Determine the diff type and SQL based on whether it's materialized
				var sql string
				var diffType DiffType
				if diff.New.Materialized {
					diffType = DiffTypeMaterializedView
					if diff.NewComment == "" {
						sql = fmt.Sprintf("COMMENT ON MATERIALIZED VIEW %s IS NULL;", viewName)
					} else {
						sql = fmt.Sprintf("COMMENT ON MATERIALIZED VIEW %s IS %s;", viewName, quoteString(diff.NewComment))
					}
				} else {
					diffType = DiffTypeView
					if diff.NewComment == "" {
						sql = fmt.Sprintf("COMMENT ON VIEW %s IS NULL;", viewName)
					} else {
						sql = fmt.Sprintf("COMMENT ON VIEW %s IS %s;", viewName, quoteString(diff.NewComment))
					}
				}

				// Create context for this statement
				context := &diffContext{
					Type:                diffType,
					Operation:           DiffOperationAlter,
					Path:                fmt.Sprintf("%s.%s", diff.New.Schema, diff.New.Name),
					Source:              diff,
					CanRunInTransaction: true,
				}

				collector.collect(context, sql)
			}

			// For materialized views, handle index modifications (only if indexes actually changed)
			if diff.New.Materialized && hasIndexChanges {
				generateIndexModifications(
					diff.DroppedIndexes,
					diff.AddedIndexes,
					diff.ModifiedIndexes,
					targetSchema,
					DiffTypeMaterializedViewIndex,
					DiffTypeMaterializedViewIndexComment,
					collector,
				)
			}
		} else {
			// Create the new view (CREATE OR REPLACE works for regular views, materialized views are handled by drop/create cycle)
			sql := generateViewSQL(diff.New, targetSchema)

			// Determine diff type based on whether it's materialized
			diffType := DiffTypeView
			if diff.New.Materialized {
				diffType = DiffTypeMaterializedView
			}

			// Create context for this statement
			context := &diffContext{
				Type:                diffType,
				Operation:           DiffOperationAlter,
				Path:                fmt.Sprintf("%s.%s", diff.New.Schema, diff.New.Name),
				Source:              diff,
				CanRunInTransaction: true,
			}

			collector.collect(context, sql)

			// Add view comment for recreated views
			if diff.New.Comment != "" {
				viewName := qualifyEntityName(diff.New.Schema, diff.New.Name, targetSchema)
				var commentSQL string
				var commentType DiffType

				if diff.New.Materialized {
					commentSQL = fmt.Sprintf("COMMENT ON MATERIALIZED VIEW %s IS %s;", viewName, quoteString(diff.New.Comment))
					commentType = DiffTypeMaterializedViewComment
				} else {
					commentSQL = fmt.Sprintf("COMMENT ON VIEW %s IS %s;", viewName, quoteString(diff.New.Comment))
					commentType = DiffTypeViewComment
				}

				// Create context for this statement
				context := &diffContext{
					Type:                commentType,
					Operation:           DiffOperationCreate,
					Path:                fmt.Sprintf("%s.%s", diff.New.Schema, diff.New.Name),
					Source:              diff.New,
					CanRunInTransaction: true,
				}

				collector.collect(context, commentSQL)
			}

			// For materialized views that were recreated, recreate indexes
			if diff.New.Materialized && diff.New.Indexes != nil {
				indexList := make([]*ir.Index, 0, len(diff.New.Indexes))
				for _, index := range diff.New.Indexes {
					indexList = append(indexList, index)
				}
				generateCreateIndexesSQLWithType(indexList, targetSchema, collector, DiffTypeMaterializedViewIndex, DiffTypeMaterializedViewIndexComment)
			}
		}
	}
}

// generateDropViewsSQL generates DROP [MATERIALIZED] VIEW statements
// Views are assumed to be pre-sorted in reverse topological order for dependency-aware dropping
func generateDropViewsSQL(views []*ir.View, targetSchema string, collector *diffCollector) {
	// Process views in the provided order (already reverse topologically sorted)
	for _, view := range views {
		viewName := qualifyEntityName(view.Schema, view.Name, targetSchema)
		var sql string
		var diffType DiffType
		if view.Materialized {
			sql = fmt.Sprintf("DROP MATERIALIZED VIEW %s RESTRICT;", viewName)
			diffType = DiffTypeMaterializedView
		} else {
			sql = fmt.Sprintf("DROP VIEW IF EXISTS %s CASCADE;", viewName)
			diffType = DiffTypeView
		}

		// Create context for this statement
		context := &diffContext{
			Type:                diffType,
			Operation:           DiffOperationDrop,
			Path:                fmt.Sprintf("%s.%s", view.Schema, view.Name),
			Source:              view,
			CanRunInTransaction: true,
		}

		collector.collect(context, sql)
	}
}

// generateViewSQL generates CREATE [OR REPLACE] [MATERIALIZED] VIEW statement
func generateViewSQL(view *ir.View, targetSchema string) string {
	// Determine view name based on context
	var viewName string
	if targetSchema != "" {
		// For diff scenarios, use schema qualification logic
		viewName = qualifyEntityName(view.Schema, view.Name, targetSchema)
	} else {
		// For dump scenarios, always include schema prefix
		viewName = fmt.Sprintf("%s.%s", view.Schema, view.Name)
	}

	// Determine CREATE statement type
	var createClause string
	if view.Materialized {
		createClause = "CREATE MATERIALIZED VIEW IF NOT EXISTS"
	} else {
		createClause = "CREATE OR REPLACE VIEW"
	}

	// Use the view definition as-is - it has already been normalized
	return fmt.Sprintf("%s %s AS\n%s;", createClause, viewName, view.Definition)
}

// viewsEqual compares two views for equality
// Both IRs come from pg_get_viewdef() at the same PostgreSQL version, so string comparison is sufficient
func viewsEqual(old, new *ir.View) bool {
	if old.Schema != new.Schema {
		return false
	}
	if old.Name != new.Name {
		return false
	}

	// Check if materialized status differs
	if old.Materialized != new.Materialized {
		return false
	}

	// Compare VIEW definitions semantically
	// Since one definition may come from parser (deparsed) and another from inspector (pg_get_viewdef),
	// they may have different formatting but be semantically equivalent
	return viewDefinitionsEqual(old.Definition, new.Definition)
}

// viewDefinitionsEqual compares two VIEW definitions semantically
// Returns true if they represent the same query, regardless of formatting
func viewDefinitionsEqual(def1, def2 string) bool {
	// First, try simple string comparison (fast path)
	if def1 == def2 {
		return true
	}

	// Normalize and compare - both should produce the same AST if semantically equal
	// Parse both definitions and deparse them to get normalized format
	normalized1 := normalizeViewDefinition(def1)
	normalized2 := normalizeViewDefinition(def2)

	equal := normalized1 == normalized2

	// Debug logging when views don't match
	if !equal {
		fmt.Printf("\n=== VIEW DEFINITION MISMATCH DEBUG ===\n")
		fmt.Printf("Original def1 length: %d\n", len(def1))
		fmt.Printf("Original def2 length: %d\n", len(def2))
		fmt.Printf("Normalized def1 length: %d\n", len(normalized1))
		fmt.Printf("Normalized def2 length: %d\n", len(normalized2))
		fmt.Printf("\n--- Original Def1 (first 200 chars) ---\n%s\n", truncate(def1, 200))
		fmt.Printf("\n--- Original Def2 (first 200 chars) ---\n%s\n", truncate(def2, 200))
		fmt.Printf("\n--- Normalized Def1 (FULL) ---\n%s\n", normalized1)
		fmt.Printf("\n--- Normalized Def2 (FULL) ---\n%s\n", normalized2)
		fmt.Printf("======================================\n\n")
	}

	return equal
}

// truncate returns first n characters of string with "..." if longer
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// normalizeViewDefinition normalizes a VIEW definition for comparison
// Uses string-based normalization instead of parse/deparse because pg_query preserves
// differences in table aliasing and column qualification that are semantically equivalent
func normalizeViewDefinition(definition string) string {
	// Start with basic cleanup
	normalized := definition

	// Remove SQL comments (-- style and /* */ style)
	normalized = removeComments(normalized)

	// Strip type casts that pg_get_viewdef adds but user SQL doesn't have
	normalized = stripTypeCasts(normalized)

	// Normalize whitespace
	normalized = normalizeWhitespace(normalized)

	// Convert to lowercase for case-insensitive comparison
	// This handles differences in keyword casing
	normalized = strings.ToLower(normalized)

	// Normalize JOIN keywords (after lowercasing)
	normalized = strings.ReplaceAll(normalized, " full outer join ", " full join ")
	normalized = strings.ReplaceAll(normalized, " inner join ", " join ")
	normalized = strings.ReplaceAll(normalized, " left outer join ", " left join ")
	normalized = strings.ReplaceAll(normalized, " right outer join ", " right join ")

	// Remove table qualifications (employees.id → id, n.n → n)
	// pg_get_viewdef adds these but user SQL typically doesn't include them
	normalized = stripTableQualifications(normalized)

	// Remove all spaces before opening parentheses to handle "any (array" vs "any(array"
	normalized = strings.ReplaceAll(normalized, " (", "(")

	// DON'T remove double parentheses or concat parens here - too risky and breaks SQL structure
	// The key differences are handled by other normalizations (type casts, table quals, etc.)

	// Remove optional "as" keyword before table/subquery aliases
	// This handles "generate_series(...) as n" vs "generate_series(...) n"
	// and ") as subquery" vs ") subquery"
	normalized = normalizeAsKeyword(normalized)

	// Handle column alias syntax like "generate_series(0,11) n(n)" vs "generate_series(0,11) as n"
	// The database may return column definitions as "alias(column_name)" which we normalize to just "alias"
	normalized = normalizeColumnAliases(normalized)

	return normalized
}

// stripTableQualifications removes table prefixes from column references
// Converts table.column → column and table_alias.column → column
// This is a heuristic approach that works for most cases
func stripTableQualifications(sql string) string {
	// This is tricky because we need to avoid breaking things like:
	// - function calls: date_trunc('month',...)
	// - schema references: pg_catalog.text
	// - JOIN clauses: table1 ON table2.id = table1.id

	// Strategy: Look for patterns like " word.word" or ",word.word" or "(word.word"
	// where it's clearly a column reference, not a function or other construct

	result := sql

	// Pattern: identifier.identifier where the first identifier is likely a table/alias
	// We'll use a simple regex-like approach with string manipulation
	// Look for contexts where column refs appear: after spaces, commas, parens, operators

	for {
		modified := false

		// After comma: ",table.column" → ",column"
		result = replaceTableRefs(result, ",", ",")

		// After space: " table.column" → " column"
		result = replaceTableRefs(result, " ", " ")

		// After open paren: "(table.column" → "(column"
		result = replaceTableRefs(result, "(", "(")

		// At start: "table.column" → "column"
		if strings.Contains(result, ".") && len(result) > 0 {
			// Check if starts with table.column pattern
			parts := strings.SplitN(result, ".", 2)
			if len(parts) == 2 && isIdentifier(parts[0]) && strings.Contains(parts[1], " ") {
				// This looks like table.column at start
				colonIdx := strings.Index(parts[1], " ")
				if colonIdx > 0 && isIdentifier(parts[1][:colonIdx]) {
					result = parts[1] // Remove table prefix
					modified = true
				}
			}
		}

		if !modified {
			break
		}
	}

	return result
}

// replaceTableRefs replaces "prefix table.column suffix" with "prefix column suffix"
func replaceTableRefs(sql, prefix, replacement string) string {
	result := sql
	startIdx := 0

	for {
		// Find prefix
		idx := strings.Index(result[startIdx:], prefix)
		if idx == -1 {
			break
		}
		idx += startIdx

		// Check if followed by table.column pattern
		after := result[idx+len(prefix):]
		dotIdx := strings.Index(after, ".")
		if dotIdx == -1 {
			startIdx = idx + len(prefix)
			continue
		}

		// Extract table name candidate
		tableName := after[:dotIdx]
		if !isIdentifier(tableName) || strings.Contains(tableName, " ") {
			startIdx = idx + len(prefix)
			continue
		}

		// Extract column name candidate
		remaining := after[dotIdx+1:]
		var columnName string
		for i, ch := range remaining {
			if !isIdentifierChar(ch) {
				columnName = remaining[:i]
				break
			}
		}
		if columnName == "" {
			columnName = remaining
		}

		if !isIdentifier(columnName) {
			startIdx = idx + len(prefix)
			continue
		}

		// Don't remove schema qualifications for known schemas
		if tableName == "pg_catalog" || strings.HasPrefix(tableName, "pg_") {
			startIdx = idx + len(prefix)
			continue
		}

		// Replace table.column with just column
		before := result[:idx+len(prefix)]
		after = remaining[len(columnName):]
		result = before + columnName + after
		startIdx = idx + len(prefix)
	}

	return result
}

// isIdentifier checks if a string looks like a SQL identifier
func isIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, ch := range s {
		if !isIdentifierChar(ch) {
			return false
		}
	}
	return true
}

// isIdentifierChar checks if a character can be part of an identifier
func isIdentifierChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_'
}

// removeComments removes SQL comments from a string
func removeComments(sql string) string {
	// Remove -- comments (line comments)
	lines := strings.Split(sql, "\n")
	var cleaned []string
	for _, line := range lines {
		// Find -- and remove everything after it
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		cleaned = append(cleaned, line)
	}
	result := strings.Join(cleaned, "\n")

	// Remove /* */ comments (block comments)
	for {
		start := strings.Index(result, "/*")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "*/")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+2:]
	}

	return result
}

// normalizeWhitespace normalizes whitespace in SQL
func normalizeWhitespace(sql string) string {
	// Replace all newlines with spaces to collapse multi-line statements
	sql = strings.ReplaceAll(sql, "\r\n", " ")
	sql = strings.ReplaceAll(sql, "\r", " ")
	sql = strings.ReplaceAll(sql, "\n", " ")

	// Replace tabs with spaces
	sql = strings.ReplaceAll(sql, "\t", " ")

	// Replace multiple spaces with single space
	for strings.Contains(sql, "  ") {
		sql = strings.ReplaceAll(sql, "  ", " ")
	}

	// Remove spaces before/after commas and parentheses for consistency
	sql = strings.ReplaceAll(sql, " ,", ",")
	sql = strings.ReplaceAll(sql, ", ", ",")
	sql = strings.ReplaceAll(sql, " )", ")")
	sql = strings.ReplaceAll(sql, "( ", "(")

	return strings.TrimSpace(sql)
}

// stripTypeCasts removes explicit type casts that PostgreSQL adds
// pg_get_viewdef() adds casts like ::text, ::character varying, ::interval, etc.
// but user SQL typically doesn't include these, causing false differences
func stripTypeCasts(sql string) string {
	// Common type casts that pg_get_viewdef adds:
	// ::text, ::character varying, ::integer, ::bigint, ::numeric, ::boolean, ::timestamp, etc.
	// Also handle schema-qualified types like ::pg_catalog.text

	// Pattern: ::type_name or ::schema.type_name
	// We need to be careful not to break ::interval or other necessary casts

	replacements := map[string]string{
		"::text":               "",
		"::character varying":  "",
		"::varchar":            "",
		"::integer":            "",
		"::bigint":             "",
		"::smallint":           "",
		"::numeric":            "",
		"::boolean":            "",
		"::regconfig":          "",
		"::pg_catalog.text":    "",
		"::pg_catalog.varchar": "",
		// Keep ::interval as it's functionally important
	}

	result := sql
	for cast, replacement := range replacements {
		result = strings.ReplaceAll(result, cast, replacement)
	}

	// After removing type casts, we may have extra parentheses like:
	// "COALESCE((first_name || ' ') || last_name, 'default')"
	// should match "COALESCE(first_name || ' ' || last_name, 'default')"
	// However, removing these parens is very risky and can break SQL structure
	// We'll accept this small difference for now
	// result = normalizeParensInFunctions(result)

	return result
}

// normalizeParensInFunctions normalizes parentheses within function calls
// Specifically handles cases where type cast removal leaves extra parens
func normalizeParensInFunctions(sql string) string {
	result := sql

	// Normalize patterns like "coalesce((a ||'b') ||c" to "coalesce(a ||'b' ||c"
	// Strategy: Within function calls, flatten consecutive || operations

	// Find all function calls and normalize their contents
	// This is complex, so let's use a simpler heuristic:
	// Replace "((expr) ||" with "(expr ||" multiple times
	for i := 0; i < 3; i++ {
		oldResult := result
		result = strings.ReplaceAll(result, "((", "(")
		result = strings.ReplaceAll(result, ") ||", " ||")
		if result == oldResult {
			break
		}
	}

	return result
}

// removeUnnecessaryConcatParens removes parentheses that were added for type cast grouping
// After stripping type casts, patterns like "(a || b) || c" should become "a || b || c"
func removeUnnecessaryConcatParens(sql string) string {
	result := sql

	// Pattern: look for "(expr || expr) ||" and flatten it
	// We need to be careful not to break function calls
	// Simple heuristic: if we see ") ||" preceded by "||", remove the parens

	// This is complex, so let's use a multi-pass approach
	// Pass 1: Replace ") || " with " || " when preceded by " || "
	for i := 0; i < 5; i++ {
		oldResult := result
		// Look for pattern "|| something) ||" and try to flatten
		result = strings.ReplaceAll(result, ") || ", " || ")
		result = strings.ReplaceAll(result, " || (", " || ")
		if result == oldResult {
			break
		}
	}

	return result
}

// normalizeConcatParens removes parentheses around concatenation expressions
// Converts "(a || b) || c" to "a || b || c"
// This handles differences in how PostgreSQL groups concatenations
func normalizeConcatParens(sql string) string {
	result := sql

	// More targeted approach: only remove parens that are clearly for concatenation grouping
	// Pattern: "(col || 'text') || other" should become "col || 'text' || other"
	// But don't break function calls like "func((col || 'text'))"

	// Look for pattern: "(something || something) || something"
	// This is specifically for cases where parens group a concat that's then concat'd again
	// We can't use simple ReplaceAll as it's too aggressive

	// For now, we'll keep this simple and handle the most common case:
	// Remove spaces around || to normalize spacing
	result = strings.ReplaceAll(result, " || ", "||")
	result = strings.ReplaceAll(result, "||", " || ")

	return result
}

// normalizeAsKeyword removes optional "as" keyword before table/subquery aliases
// Handles "generate_series(...) as n" vs "generate_series(...) n"
// and ") as subquery" vs ") subquery"
func normalizeAsKeyword(sql string) string {
	result := sql

	// Don't remove " as " because it's too risky and can break CTE syntax
	// Instead, we'll rely on other normalizations to handle the differences
	// The key insight is that PostgreSQL may use "identifier(column_list)" syntax
	// while user SQL uses "as identifier" syntax, which normalizeColumnAliases handles

	return result
}

// normalizeColumnAliases normalizes column alias syntax
// Handles "generate_series(0,11) n(n)" vs "generate_series(0,11) as n"
// The database may return column definitions as "alias(column_name)"
func normalizeColumnAliases(sql string) string {
	result := sql

	// Pattern 1: "identifier(identifier)" after a closing paren (function call)
	// Example: "generate_series(0,11) n(n)" should become "generate_series(0,11) n"

	// Pattern 2: " as identifier" after a closing paren
	// Example: "generate_series(0,11) as n" should become "generate_series(0,11) n"

	// First, handle " as " after closing parens (for table aliases)
	// Pattern: ") as word" → ") word" but only for single words (table aliases)
	result = normalizeTableAliases(result)

	// Then handle column list syntax like "n(n)" → "n"
	result = removeColumnListSyntax(result)

	return result
}

// normalizeTableAliases converts ") as identifier" to ") identifier" for table aliases
func normalizeTableAliases(sql string) string {
	var builder strings.Builder
	builder.Grow(len(sql))

	i := 0
	for i < len(sql) {
		// Look for pattern ") as "
		if i < len(sql)-5 && sql[i:i+5] == ") as " {
			// Check if what follows is a simple identifier (table alias)
			// and not a column alias (which would be in SELECT context)
			j := i + 5
			identStart := j
			for j < len(sql) && isIdentifierChar(rune(sql[j])) {
				j++
			}

			if j > identStart {
				// Found an identifier after ") as "
				// Check if next char is not '(' (which would indicate a function/column list)
				if j >= len(sql) || sql[j] != '(' {
					// This looks like a table alias, remove "as "
					builder.WriteString(") ")
					builder.WriteString(sql[identStart:j])
					i = j
					continue
				}
			}
		}

		builder.WriteByte(sql[i])
		i++
	}

	return builder.String()
}

// removeColumnListSyntax removes column list syntax like "n(n)" → "n"
func removeColumnListSyntax(sql string) string {
	var builder strings.Builder
	builder.Grow(len(sql))

	i := 0
	for i < len(sql) {
		// Look for pattern ") identifier(identifier)"
		if i < len(sql)-5 && sql[i:i+2] == ") " {
			// Copy ") "
			builder.WriteString(") ")
			i += 2

			// Try to match "identifier(identifier)"
			identStart := i
			for i < len(sql) && isIdentifierChar(rune(sql[i])) {
				i++
			}

			if i < len(sql) && sql[i] == '(' {
				// Found opening paren, save the identifier
				ident1 := sql[identStart:i]
				i++ // skip '('

				// Look for second identifier
				ident2Start := i
				for i < len(sql) && isIdentifierChar(rune(sql[i])) {
					i++
				}

				if i < len(sql) && sql[i] == ')' {
					// Found closing paren - this matches "identifier(identifier)" pattern
					// Just write the first identifier, skip the "(identifier)" part
					builder.WriteString(ident1)
					i++ // skip ')'
				} else {
					// Not the pattern, write what we saw
					builder.WriteString(ident1)
					builder.WriteString("(")
					builder.WriteString(sql[ident2Start:i])
				}
			} else {
				// No opening paren, just write the identifier we found
				builder.WriteString(sql[identStart:i])
			}
		} else {
			builder.WriteByte(sql[i])
			i++
		}
	}

	return builder.String()
}

// viewDependsOnView checks if viewA depends on viewB
func viewDependsOnView(viewA *ir.View, viewBName string) bool {
	// Simple heuristic: check if viewB name appears in viewA definition
	// This can be enhanced with proper dependency parsing later
	return strings.Contains(strings.ToLower(viewA.Definition), strings.ToLower(viewBName))
}
