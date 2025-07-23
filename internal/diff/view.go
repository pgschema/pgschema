package diff

import (
	"fmt"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v5"
	"github.com/pgschema/pgschema/internal/ir"
)

// generateCreateViewsSQL generates CREATE VIEW statements
// Views are assumed to be pre-sorted in topological order for dependency-aware creation
func generateCreateViewsSQL(w *SQLWriter, views []*ir.View, targetSchema string, compare bool) {
	// Process views in the provided order (already topologically sorted)
	for _, view := range views {
		w.WriteDDLSeparator()
		// If compare mode, CREATE OR REPLACE, otherwise CREATE
		sql := generateViewSQL(view, targetSchema, compare)
		w.WriteStatementWithComment("VIEW", view.Name, view.Schema, "", sql, targetSchema)
	}
}

// generateModifyViewsSQL generates CREATE OR REPLACE VIEW statements
func generateModifyViewsSQL(w *SQLWriter, diffs []*ViewDiff, targetSchema string) {
	for _, diff := range diffs {
		w.WriteDDLSeparator()
		sql := generateViewSQL(diff.New, targetSchema, true) // Use OR REPLACE for modified views
		w.WriteStatementWithComment("VIEW", diff.New.Name, diff.New.Schema, "", sql, targetSchema)
	}
}

// generateDropViewsSQL generates DROP VIEW statements
// Views are assumed to be pre-sorted in reverse topological order for dependency-aware dropping
func generateDropViewsSQL(w *SQLWriter, views []*ir.View, targetSchema string) {
	// Process views in the provided order (already reverse topologically sorted)
	for _, view := range views {
		w.WriteDDLSeparator()
		viewName := qualifyEntityName(view.Schema, view.Name, targetSchema)
		sql := fmt.Sprintf("DROP VIEW IF EXISTS %s CASCADE;", viewName)
		w.WriteStatementWithComment("VIEW", view.Name, view.Schema, "", sql, targetSchema)
	}
}

// generateViewSQL generates CREATE [OR REPLACE] VIEW statement
func generateViewSQL(view *ir.View, targetSchema string, useReplace bool) string {
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
	createClause := "CREATE VIEW"
	if useReplace {
		createClause = "CREATE OR REPLACE VIEW"
	}

	return fmt.Sprintf("%s %s AS\n%s;", createClause, viewName, view.Definition)
}

// viewsEqual compares two views for equality using semantic comparison
func viewsEqual(old, new *ir.View) bool {
	if old.Schema != new.Schema {
		return false
	}
	if old.Name != new.Name {
		return false
	}

	// Quick path: if string definitions are identical, they're equal
	if old.Definition == new.Definition {
		return true
	}

	// Use semantic comparison using AST analysis (assumes valid SQL)
	return compareViewDefinitionsSemanticially(old.Definition, new.Definition)
}

// viewDependsOnView checks if viewA depends on viewB
func viewDependsOnView(viewA *ir.View, viewBName string) bool {
	// Simple heuristic: check if viewB name appears in viewA definition
	// This can be enhanced with proper dependency parsing later
	return strings.Contains(strings.ToLower(viewA.Definition), strings.ToLower(viewBName))
}

// compareViewDefinitionsSemanticially compares two SQL view definitions semantically
// using AST comparison rather than string comparison to handle formatting differences
// Assumes valid SQL syntax is always passed
func compareViewDefinitionsSemanticially(def1, def2 string) bool {
	if def1 == def2 {
		return true // Quick path for identical strings
	}

	// Parse both definitions into ASTs (assuming valid SQL)
	result1, _ := pg_query.Parse(def1)
	result2, _ := pg_query.Parse(def2)

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

	// TODO: Add comparison for GROUP BY, HAVING, ORDER BY, etc. as needed

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

	// Compare schema and table names
	return rv1.Schemaname == rv2.Schemaname &&
		rv1.Relname == rv2.Relname &&
		rv1.Alias.GetAliasname() == rv2.Alias.GetAliasname()
}

// compareExpressions compares SQL expressions semantically
func compareExpressions(expr1, expr2 *pg_query.Node) bool {
	if expr1 == nil || expr2 == nil {
		return expr1 == expr2
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

	// Compare field names
	if len(col1.Fields) != len(col2.Fields) {
		return false
	}

	for i, field1 := range col1.Fields {
		field2 := col2.Fields[i]
		str1 := field1.GetString_()
		str2 := field2.GetString_()
		if str1 == nil || str2 == nil || str1.Sval != str2.Sval {
			return false
		}
	}

	return true
}

// compareAConsts compares constant values
func compareAConsts(const1, const2 *pg_query.A_Const) bool {
	if const1 == nil || const2 == nil {
		return const1 == const2
	}

	// Compare the constant values - this is simplified and may need expansion
	return const1.String() == const2.String()
}

// compareWhereClauses compares WHERE clauses
func compareWhereClauses(where1, where2 *pg_query.Node) bool {
	return compareExpressions(where1, where2)
}
