package diff

import (
	"fmt"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/pgschema/pgschema/internal/ir"
)

// generateCreateViewsSQL generates CREATE VIEW statements
// Views are assumed to be pre-sorted in topological order for dependency-aware creation
func generateCreateViewsSQL(views []*ir.View, targetSchema string, collector *diffCollector) {
	// Process views in the provided order (already topologically sorted)
	for _, view := range views {
		// If compare mode, CREATE OR REPLACE, otherwise CREATE
		sql := generateViewSQL(view, targetSchema)

		// Create context for this statement
		context := &diffContext{
			Type:      "view",
			Operation: "create",
			Path:      fmt.Sprintf("%s.%s", view.Schema, view.Name),
			Source:    view,
		}

		collector.collect(context, sql)

		// Add view comment
		if view.Comment != "" {
			viewName := qualifyEntityName(view.Schema, view.Name, targetSchema)
			sql := fmt.Sprintf("COMMENT ON VIEW %s IS %s;", viewName, quoteString(view.Comment))

			// Create context for this statement
			context := &diffContext{
				Type:                "view.comment",
				Operation:           "create",
				Path:                fmt.Sprintf("%s.%s", view.Schema, view.Name),
				Source:              view,
				CanRunInTransaction: true,
			}

			collector.collect(context, sql)
		}
	}
}

// generateModifyViewsSQL generates CREATE OR REPLACE VIEW statements or comment changes
func generateModifyViewsSQL(diffs []*viewDiff, targetSchema string, collector *diffCollector) {
	for _, diff := range diffs {
		// Check if only the comment changed and definition is semantically identical
		definitionsEqual := diff.Old.Definition == diff.New.Definition || compareViewDefinitionsSemantically(diff.Old.Definition, diff.New.Definition)
		commentOnlyChange := diff.CommentChanged && definitionsEqual && diff.Old.Materialized == diff.New.Materialized
		if commentOnlyChange {
			// Only generate COMMENT ON VIEW statement for comment-only changes
			viewName := qualifyEntityName(diff.New.Schema, diff.New.Name, targetSchema)
			if diff.NewComment == "" {
				sql := fmt.Sprintf("COMMENT ON VIEW %s IS NULL;", viewName)

				// Create context for this statement
				context := &diffContext{
					Type:                "view",
					Operation:           "alter",
					Path:                fmt.Sprintf("%s.%s", diff.New.Schema, diff.New.Name),
					Source:              diff,
					CanRunInTransaction: true,
				}

				collector.collect(context, sql)
			} else {
				sql := fmt.Sprintf("COMMENT ON VIEW %s IS %s;", viewName, quoteString(diff.NewComment))

				// Create context for this statement
				context := &diffContext{
					Type:                "view",
					Operation:           "alter",
					Path:                fmt.Sprintf("%s.%s", diff.New.Schema, diff.New.Name),
					Source:              diff,
					CanRunInTransaction: true,
				}

				collector.collect(context, sql)
			}
		} else {
			// Create the new view (CREATE OR REPLACE works for regular views, materialized views are handled by drop/create cycle)
			sql := generateViewSQL(diff.New, targetSchema)

			// Create context for this statement
			context := &diffContext{
				Type:                "view",
				Operation:           "alter",
				Path:                fmt.Sprintf("%s.%s", diff.New.Schema, diff.New.Name),
				Source:              diff,
				CanRunInTransaction: true,
			}

			collector.collect(context, sql)

			// Add view comment for recreated views
			if diff.New.Comment != "" {
				viewName := qualifyEntityName(diff.New.Schema, diff.New.Name, targetSchema)
				sql := fmt.Sprintf("COMMENT ON VIEW %s IS %s;", viewName, quoteString(diff.New.Comment))

				// Create context for this statement
				context := &diffContext{
					Type:                "comment",
					Operation:           "create",
					Path:                fmt.Sprintf("%s.%s", diff.New.Schema, diff.New.Name),
					Source:              diff.New,
					CanRunInTransaction: true,
				}

				collector.collect(context, sql)
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
		if view.Materialized {
			sql = fmt.Sprintf("DROP MATERIALIZED VIEW %s RESTRICT;", viewName)
		} else {
			sql = fmt.Sprintf("DROP VIEW IF EXISTS %s CASCADE;", viewName)
		}

		// Create context for this statement
		context := &diffContext{
			Type:                "view",
			Operation:           "drop",
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

	return fmt.Sprintf("%s %s AS\n%s;", createClause, viewName, view.Definition)
}

// normalizeViewDefinition normalizes SQL view definitions for semantic comparison
func normalizeViewDefinition(definition string) string {
	// Remove trailing semicolon and whitespace
	definition = strings.TrimSpace(definition)
	definition = strings.TrimSuffix(definition, ";")
	definition = strings.TrimSpace(definition)

	return definition
}

// viewsEqual compares two views for equality using semantic comparison
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

	// Quick path: if string definitions are identical, they're equal
	if old.Definition == new.Definition {
		return true
	}

	// Use semantic comparison using AST analysis (assumes valid SQL)
	return compareViewDefinitionsSemantically(old.Definition, new.Definition)
}

// viewDependsOnView checks if viewA depends on viewB
func viewDependsOnView(viewA *ir.View, viewBName string) bool {
	// Simple heuristic: check if viewB name appears in viewA definition
	// This can be enhanced with proper dependency parsing later
	return strings.Contains(strings.ToLower(viewA.Definition), strings.ToLower(viewBName))
}

// compareViewDefinitionsSemantically compares two SQL view definitions semantically
// using AST comparison rather than string comparison to handle formatting differences
// Assumes valid SQL syntax is always passed
func compareViewDefinitionsSemantically(def1, def2 string) bool {
	if def1 == def2 {
		return true // Quick path for identical strings
	}

	// Normalize the SQL definitions before parsing
	def1 = normalizeViewDefinition(def1)
	def2 = normalizeViewDefinition(def2)

	// Quick check after normalization
	if def1 == def2 {
		return true
	}

	// Parse both definitions into ASTs (assuming valid SQL)
	result1, err1 := pg_query.Parse(def1)
	result2, err2 := pg_query.Parse(def2)

	if err1 != nil || err2 != nil {
		return false
	}

	// Both should have exactly one statement (the SELECT for the view)
	if len(result1.Stmts) != 1 || len(result2.Stmts) != 1 {
		return false
	}

	// Compare the SELECT statements semantically
	equal, _ := compareSelectStatements(result1.Stmts[0], result2.Stmts[0])
	return equal
}

// compareSelectStatements compares two SELECT statement ASTs for semantic equivalence
func compareSelectStatements(stmt1, stmt2 *pg_query.RawStmt) (bool, error) {
	// Extract SelectStmt from RawStmt
	selectStmt1 := stmt1.Stmt.GetSelectStmt()
	selectStmt2 := stmt2.Stmt.GetSelectStmt()

	if selectStmt1 == nil || selectStmt2 == nil {
		return false, fmt.Errorf("expected SELECT statements")
	}

	// Compare key components of SELECT statements
	if !compareTargetLists(selectStmt1.TargetList, selectStmt2.TargetList) {
		return false, nil
	}

	if !compareFromClauses(selectStmt1.FromClause, selectStmt2.FromClause) {
		return false, nil
	}

	if !compareWhereClauses(selectStmt1.WhereClause, selectStmt2.WhereClause) {
		return false, nil
	}

	// Compare GROUP BY clause
	if !compareGroupByClauses(selectStmt1.GroupClause, selectStmt2.GroupClause) {
		return false, nil
	}

	// TODO: Add comparison for HAVING, ORDER BY, etc. as needed

	return true, nil
}

// compareTargetLists compares SELECT target lists (column expressions)
func compareTargetLists(list1, list2 []*pg_query.Node) bool {
	if len(list1) != len(list2) {
		return false
	}

	for i, target1 := range list1 {
		target2 := list2[i]
		if !compareResTargets(target1.GetResTarget(), target2.GetResTarget()) {
			return false
		}
	}

	return true
}

// compareResTargets compares individual SELECT targets (columns/expressions)
func compareResTargets(target1, target2 *pg_query.ResTarget) bool {
	if target1 == nil || target2 == nil {
		return target1 == target2
	}

	// Compare target names (aliases)
	if target1.Name != target2.Name {
		return false
	}

	// Compare target expressions
	return compareExpressions(target1.Val, target2.Val)
}

// compareFromClauses compares FROM clauses including JOINs
func compareFromClauses(from1, from2 []*pg_query.Node) bool {
	if len(from1) != len(from2) {
		return false
	}

	for i, node1 := range from1 {
		node2 := from2[i]
		if !compareFromClauseNode(node1, node2) {
			return false
		}
	}

	return true
}

// compareFromClauseNode compares individual FROM clause nodes (tables, JOINs, etc.)
func compareFromClauseNode(node1, node2 *pg_query.Node) bool {
	// Handle JoinExpr (the main case we're fixing)
	if join1 := node1.GetJoinExpr(); join1 != nil {
		join2 := node2.GetJoinExpr()
		if join2 == nil {
			return false
		}
		return compareJoinExprs(join1, join2)
	}

	// Handle RangeVar (simple table references)
	if rangeVar1 := node1.GetRangeVar(); rangeVar1 != nil {
		rangeVar2 := node2.GetRangeVar()
		if rangeVar2 == nil {
			return false
		}
		return compareRangeVars(rangeVar1, rangeVar2)
	}

	// TODO: Add other FROM clause node types as needed

	return false
}

// compareJoinExprs compares JOIN expressions - this is the key function for our issue
func compareJoinExprs(join1, join2 *pg_query.JoinExpr) bool {
	if join1 == nil || join2 == nil {
		return join1 == join2
	}

	// Compare join type
	if join1.Jointype != join2.Jointype {
		return false
	}

	// Compare left and right operands
	if !compareFromClauseNode(join1.Larg, join2.Larg) {
		return false
	}

	if !compareFromClauseNode(join1.Rarg, join2.Rarg) {
		return false
	}

	// Compare join conditions - this is where the parentheses differences occur
	return compareExpressions(join1.Quals, join2.Quals)
}

// compareRangeVars compares table references
func compareRangeVars(rv1, rv2 *pg_query.RangeVar) bool {
	if rv1 == nil || rv2 == nil {
		return rv1 == rv2
	}

	// Normalize schema names - empty string should be treated as "public"
	schema1 := rv1.Schemaname
	schema2 := rv2.Schemaname
	if schema1 == "" {
		schema1 = "public"
	}
	if schema2 == "" {
		schema2 = "public"
	}

	// Compare normalized schema and table names
	return schema1 == schema2 &&
		rv1.Relname == rv2.Relname &&
		rv1.Alias.GetAliasname() == rv2.Alias.GetAliasname()
}

// compareExpressions compares SQL expressions semantically
func compareExpressions(expr1, expr2 *pg_query.Node) bool {
	if expr1 == nil || expr2 == nil {
		return expr1 == expr2
	}

	// Handle TypeCast expressions using normalized comparison
	if expr1.GetTypeCast() != nil || expr2.GetTypeCast() != nil {
		return compareExpressionsWithTypeCast(expr1, expr2)
	}

	// Handle BoolExpr (AND, OR, NOT)
	if boolExpr1 := expr1.GetBoolExpr(); boolExpr1 != nil {
		boolExpr2 := expr2.GetBoolExpr()
		if boolExpr2 == nil {
			return false
		}
		return compareBoolExprs(boolExpr1, boolExpr2)
	}

	// Handle A_Expr (comparison operators like =, <, >, etc.)
	if aExpr1 := expr1.GetAExpr(); aExpr1 != nil {
		aExpr2 := expr2.GetAExpr()
		if aExpr2 == nil {
			return false
		}
		return compareAExprs(aExpr1, aExpr2)
	}

	// Handle ColumnRef (column references)
	if colRef1 := expr1.GetColumnRef(); colRef1 != nil {
		colRef2 := expr2.GetColumnRef()
		if colRef2 == nil {
			return false
		}
		return compareColumnRefs(colRef1, colRef2)
	}

	// Handle A_Const (constants)
	if const1 := expr1.GetAConst(); const1 != nil {
		const2 := expr2.GetAConst()
		if const2 == nil {
			return false
		}
		return compareAConsts(const1, const2)
	}

	// Handle FuncCall (function calls)
	if funcCall1 := expr1.GetFuncCall(); funcCall1 != nil {
		funcCall2 := expr2.GetFuncCall()
		if funcCall2 == nil {
			return false
		}
		return compareFuncCalls(funcCall1, funcCall2)
	}

	// Handle CaseExpr (CASE expressions)
	if caseExpr1 := expr1.GetCaseExpr(); caseExpr1 != nil {
		caseExpr2 := expr2.GetCaseExpr()
		if caseExpr2 == nil {
			return false
		}
		return compareCaseExprs(caseExpr1, caseExpr2)
	}

	// TODO: Add other expression types as needed

	return false
}

// compareBoolExprs compares boolean expressions (AND, OR, NOT)
func compareBoolExprs(bool1, bool2 *pg_query.BoolExpr) bool {
	if bool1 == nil || bool2 == nil {
		return bool1 == bool2
	}

	// Must have same boolean operation type
	if bool1.Boolop != bool2.Boolop {
		return false
	}

	// Must have same number of arguments
	if len(bool1.Args) != len(bool2.Args) {
		return false
	}

	// Compare each argument
	for i, arg1 := range bool1.Args {
		arg2 := bool2.Args[i]
		if !compareExpressions(arg1, arg2) {
			return false
		}
	}

	return true
}

// compareAExprs compares A_Expr nodes (comparison operators)
func compareAExprs(expr1, expr2 *pg_query.A_Expr) bool {
	if expr1 == nil || expr2 == nil {
		return expr1 == expr2
	}

	// Compare operator names
	if !compareOperatorNames(expr1.Name, expr2.Name) {
		return false
	}

	// Compare left and right operands
	return compareExpressions(expr1.Lexpr, expr2.Lexpr) &&
		compareExpressions(expr1.Rexpr, expr2.Rexpr)
}

// compareOperatorNames compares operator names
func compareOperatorNames(names1, names2 []*pg_query.Node) bool {
	if len(names1) != len(names2) {
		return false
	}

	for i, name1 := range names1 {
		name2 := names2[i]
		str1 := name1.GetString_()
		str2 := name2.GetString_()
		if str1 == nil || str2 == nil || str1.Sval != str2.Sval {
			return false
		}
	}

	return true
}

// compareColumnRefs compares column references
func compareColumnRefs(col1, col2 *pg_query.ColumnRef) bool {
	if col1 == nil || col2 == nil {
		return col1 == col2
	}

	// Quick path: if they have same structure, compare directly
	if len(col1.Fields) == len(col2.Fields) {
		allMatch := true
		for i, field1 := range col1.Fields {
			field2 := col2.Fields[i]
			str1 := field1.GetString_()
			str2 := field2.GetString_()
			if str1 == nil || str2 == nil || str1.Sval != str2.Sval {
				allMatch = false
				break
			}
		}
		if allMatch {
			return true
		}
	}

	// Handle alias expansion: compare "alias.column" vs "column"
	// Extract the final column name from each reference
	colName1 := getColumnName(col1)
	colName2 := getColumnName(col2)

	// If the column names match, consider them equivalent
	// This handles cases like "e.id" vs "id"
	return colName1 == colName2
}

// getColumnName extracts the final column name from a ColumnRef
func getColumnName(colRef *pg_query.ColumnRef) string {
	if colRef == nil || len(colRef.Fields) == 0 {
		return ""
	}

	// Get the last field (the actual column name)
	lastField := colRef.Fields[len(colRef.Fields)-1]
	if str := lastField.GetString_(); str != nil {
		return str.Sval
	}

	return ""
}

// compareAConsts compares constant values
func compareAConsts(const1, const2 *pg_query.A_Const) bool {
	if const1 == nil || const2 == nil {
		return const1 == const2
	}

	// Compare the actual values, not the string representation (which includes location info)
	switch val1 := const1.Val.(type) {
	case *pg_query.A_Const_Sval:
		if val2, ok := const2.Val.(*pg_query.A_Const_Sval); ok {
			return val1.Sval.Sval == val2.Sval.Sval
		}
	case *pg_query.A_Const_Ival:
		if val2, ok := const2.Val.(*pg_query.A_Const_Ival); ok {
			return val1.Ival.Ival == val2.Ival.Ival
		}
	case *pg_query.A_Const_Fval:
		if val2, ok := const2.Val.(*pg_query.A_Const_Fval); ok {
			return val1.Fval.Fval == val2.Fval.Fval
		}
	case *pg_query.A_Const_Boolval:
		if val2, ok := const2.Val.(*pg_query.A_Const_Boolval); ok {
			return val1.Boolval.Boolval == val2.Boolval.Boolval
		}
	case *pg_query.A_Const_Bsval:
		if val2, ok := const2.Val.(*pg_query.A_Const_Bsval); ok {
			return val1.Bsval.Bsval == val2.Bsval.Bsval
		}
	}

	// Fallback to string comparison if types don't match or are unknown
	return const1.String() == const2.String()
}

// compareWhereClauses compares WHERE clauses
func compareWhereClauses(where1, where2 *pg_query.Node) bool {
	return compareExpressions(where1, where2)
}

// compareGroupByClauses compares GROUP BY clauses
func compareGroupByClauses(group1, group2 []*pg_query.Node) bool {
	if len(group1) != len(group2) {
		return false
	}

	for i, expr1 := range group1 {
		expr2 := group2[i]
		if !compareExpressions(expr1, expr2) {
			return false
		}
	}

	return true
}

// compareFuncCalls compares function call expressions
func compareFuncCalls(func1, func2 *pg_query.FuncCall) bool {
	if func1 == nil || func2 == nil {
		return func1 == func2
	}

	// Compare function names
	if !compareFuncNames(func1.Funcname, func2.Funcname) {
		return false
	}

	// Compare arguments
	if len(func1.Args) != len(func2.Args) {
		return false
	}

	for i, arg1 := range func1.Args {
		arg2 := func2.Args[i]
		if !compareExpressions(arg1, arg2) {
			return false
		}
	}

	// Ignore other function properties like location, agg_star for now
	// We can add them later if needed

	return true
}

// compareFuncNames compares function name lists
func compareFuncNames(names1, names2 []*pg_query.Node) bool {
	if len(names1) != len(names2) {
		return false
	}

	for i, name1 := range names1 {
		name2 := names2[i]
		str1 := name1.GetString_()
		str2 := name2.GetString_()
		if str1 == nil || str2 == nil || str1.Sval != str2.Sval {
			return false
		}
	}

	return true
}

// compareCaseExprs compares CASE expressions
func compareCaseExprs(case1, case2 *pg_query.CaseExpr) bool {
	if case1 == nil || case2 == nil {
		return case1 == case2
	}

	// Compare the case expression argument (the expression after CASE, if any)
	if !compareExpressions(case1.Arg, case2.Arg) {
		return false
	}

	// Compare WHEN clauses
	if len(case1.Args) != len(case2.Args) {
		return false
	}

	for i, when1 := range case1.Args {
		when2 := case2.Args[i]
		if !compareCaseWhenClauses(when1.GetCaseWhen(), when2.GetCaseWhen()) {
			return false
		}
	}

	// Compare ELSE clause (default result)
	return compareExpressions(case1.Defresult, case2.Defresult)
}

// compareCaseWhenClauses compares individual WHEN clauses in CASE expressions
func compareCaseWhenClauses(when1, when2 *pg_query.CaseWhen) bool {
	if when1 == nil || when2 == nil {
		return when1 == when2
	}

	// Compare the WHEN condition
	if !compareExpressions(when1.Expr, when2.Expr) {
		return false
	}

	// Compare the THEN result
	return compareExpressions(when1.Result, when2.Result)
}

// compareExpressionsWithTypeCast compares expressions where at least one has a type cast
// This handles PostgreSQL's automatic type casting behavior in a normalized way
func compareExpressionsWithTypeCast(expr1, expr2 *pg_query.Node) bool {
	typeCast1 := expr1.GetTypeCast()
	typeCast2 := expr2.GetTypeCast()

	// Case 1: Both expressions are TypeCasts
	if typeCast1 != nil && typeCast2 != nil {
		return compareTypeCasts(typeCast1, typeCast2)
	}

	// Case 2: Only one expression is a TypeCast
	if typeCast1 != nil {
		// expr1 is TypeCast, expr2 is not
		argCompare := compareExpressions(typeCast1.Arg, expr2)
		if argCompare {
			return isImplicitCast(typeCast1)
		}
		return false
	}

	if typeCast2 != nil {
		// expr2 is TypeCast, expr1 is not
		argCompare := compareExpressions(expr1, typeCast2.Arg)
		if argCompare {
			return isImplicitCast(typeCast2)
		}
		return false
	}

	// This should never happen as we check for TypeCast existence before calling this function
	return false
}

// compareTypeCasts compares two TypeCast expressions
func compareTypeCasts(cast1, cast2 *pg_query.TypeCast) bool {
	if cast1 == nil || cast2 == nil {
		return cast1 == cast2
	}

	// Compare the arguments being cast
	if !compareExpressions(cast1.Arg, cast2.Arg) {
		return false
	}

	// Compare the target types - consider compatible types as equivalent
	return areCompatibleTypes(cast1.TypeName, cast2.TypeName)
}

// isImplicitCast checks if a type cast is likely an implicit cast added by PostgreSQL
func isImplicitCast(typeCast *pg_query.TypeCast) bool {
	if typeCast.TypeName == nil || len(typeCast.TypeName.Names) == 0 {
		return false
	}

	// Get the target type name
	var typeName string
	if str := typeCast.TypeName.Names[len(typeCast.TypeName.Names)-1].GetString_(); str != nil {
		typeName = str.Sval
	}

	// PostgreSQL commonly adds these implicit casts
	implicitCastTypes := map[string]bool{
		"text":    true,
		"varchar": true,
		"char":    true,
		"int4":    true,
		"int8":    true,
		"numeric": true,
		"bool":    true,
	}

	return implicitCastTypes[typeName]
}

// areCompatibleTypes checks if two type names are compatible for comparison
func areCompatibleTypes(type1, type2 *pg_query.TypeName) bool {
	if type1 == nil || type2 == nil {
		return type1 == type2
	}

	// Extract type names
	typeName1 := getTypeName(type1)
	typeName2 := getTypeName(type2)

	// Exact match
	if typeName1 == typeName2 {
		return true
	}

	// Check for compatible text types
	textTypes := map[string]bool{
		"text": true, "varchar": true, "char": true, "character varying": true,
	}
	if textTypes[typeName1] && textTypes[typeName2] {
		return true
	}

	// Check for compatible integer types
	intTypes := map[string]bool{
		"int4": true, "integer": true, "int": true,
		"int8": true, "bigint": true,
	}
	if intTypes[typeName1] && intTypes[typeName2] {
		return true
	}

	return false
}

// getTypeName extracts the type name from a TypeName node
func getTypeName(typeName *pg_query.TypeName) string {
	if typeName == nil || len(typeName.Names) == 0 {
		return ""
	}

	// Get the last name in the list (the actual type name)
	if str := typeName.Names[len(typeName.Names)-1].GetString_(); str != nil {
		return str.Sval
	}

	return ""
}
