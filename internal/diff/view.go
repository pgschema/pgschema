package diff

import (
	"fmt"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
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

	return normalized1 == normalized2
}

// normalizeViewDefinition normalizes a VIEW definition by parsing and deparsing it
func normalizeViewDefinition(definition string) string {
	// Wrap the definition in a SELECT statement for parsing
	// pg_query expects complete SQL statements
	selectStmt := definition
	if !strings.HasPrefix(strings.TrimSpace(strings.ToUpper(definition)), "SELECT") &&
		!strings.HasPrefix(strings.TrimSpace(strings.ToUpper(definition)), "WITH") {
		// If it doesn't start with SELECT or WITH, wrap it
		selectStmt = "SELECT " + definition
	}

	// Parse the definition
	result, err := pg_query.Parse(selectStmt)
	if err != nil {
		// If parsing fails, return the original for string comparison
		// This handles cases where the definition might have special syntax
		return definition
	}

	// Deparse to get normalized format
	deparsed, err := pg_query.Deparse(result)
	if err != nil {
		// If deparsing fails, return the original
		return definition
	}

	// Remove trailing semicolon for consistency
	deparsed = strings.TrimSuffix(strings.TrimSpace(deparsed), ";")

	return deparsed
}

// viewDependsOnView checks if viewA depends on viewB
func viewDependsOnView(viewA *ir.View, viewBName string) bool {
	// Simple heuristic: check if viewB name appears in viewA definition
	// This can be enhanced with proper dependency parsing later
	return strings.Contains(strings.ToLower(viewA.Definition), strings.ToLower(viewBName))
}
