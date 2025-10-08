package ir

import (
	"strconv"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// postgreSQLFormatter implements PostgreSQL's pg_get_viewdef pretty-print algorithm
type postgreSQLFormatter struct {
	buffer      *strings.Builder
	indentLevel int
}

// newPostgreSQLFormatter creates a new PostgreSQL formatter
func newPostgreSQLFormatter() *postgreSQLFormatter {
	return &postgreSQLFormatter{
		buffer:      &strings.Builder{},
		indentLevel: 0,
	}
}

// formatQueryNode formats a query AST using PostgreSQL's formatting rules
func (f *postgreSQLFormatter) formatQueryNode(queryNode *pg_query.Node) string {
	if selectStmt := queryNode.GetSelectStmt(); selectStmt != nil {
		f.formatSelectStmt(selectStmt)
	} else {
		// Fallback to deparse if not a SelectStmt
		stmt := &pg_query.RawStmt{Stmt: queryNode}
		parseResult := &pg_query.ParseResult{Stmts: []*pg_query.RawStmt{stmt}}
		if deparseResult, err := pg_query.Deparse(parseResult); err == nil {
			return strings.TrimSpace(deparseResult)
		}
		return ""
	}

	return f.buffer.String()
}

// formatSelectStmt formats a SELECT statement according to PostgreSQL's rules
func (f *postgreSQLFormatter) formatSelectStmt(stmt *pg_query.SelectStmt) {
	// Add leading space and SELECT keyword (PostgreSQL adds a leading space)
	f.buffer.WriteString(" SELECT")

	// Format target list (columns)
	if len(stmt.TargetList) > 0 {
		f.formatTargetList(stmt.TargetList)
	}

	// Format FROM clause
	if len(stmt.FromClause) > 0 {
		f.buffer.WriteString("\n   FROM ")
		f.formatFromClause(stmt.FromClause)
	}

	// Format WHERE clause
	if stmt.WhereClause != nil {
		f.buffer.WriteString("\n  WHERE ")
		f.formatExpression(stmt.WhereClause)
	}

	// Format GROUP BY clause
	if len(stmt.GroupClause) > 0 {
		f.buffer.WriteString("\n  GROUP BY ")
		f.formatGroupByClause(stmt.GroupClause)
	}

	// Format HAVING clause
	if stmt.HavingClause != nil {
		f.buffer.WriteString("\n  HAVING ")
		f.formatExpression(stmt.HavingClause)
	}

	// Format ORDER BY clause
	if len(stmt.SortClause) > 0 {
		f.buffer.WriteString("\n  ORDER BY ")
		f.formatOrderByClause(stmt.SortClause)
	}
}

// formatTargetList formats the SELECT column list
func (f *postgreSQLFormatter) formatTargetList(targets []*pg_query.Node) {
	for i, target := range targets {
		if i == 0 {
			f.buffer.WriteString("\n    ") // First column indentation
		} else {
			f.buffer.WriteString(",\n    ") // Subsequent columns
		}

		if resTarget := target.GetResTarget(); resTarget != nil {
			f.formatResTarget(resTarget)
		}
	}
}

// formatResTarget formats a single SELECT target (column/expression)
func (f *postgreSQLFormatter) formatResTarget(target *pg_query.ResTarget) {
	// Format the expression
	if target.Val != nil {
		f.formatExpression(target.Val)
	}

	// Add alias if present
	if target.Name != "" {
		f.buffer.WriteString(" AS ")
		f.buffer.WriteString(target.Name)
	}
}

// formatFromClause formats the FROM clause
func (f *postgreSQLFormatter) formatFromClause(fromList []*pg_query.Node) {
	for i, fromItem := range fromList {
		if i > 0 {
			f.buffer.WriteString(", ")
		}
		f.formatFromItem(fromItem)
	}
}

// formatFromItem formats a single FROM item (table, join, subquery)
func (f *postgreSQLFormatter) formatFromItem(item *pg_query.Node) {
	switch {
	case item.GetRangeVar() != nil:
		f.formatRangeVar(item.GetRangeVar())
	case item.GetJoinExpr() != nil:
		f.formatJoinExpr(item.GetJoinExpr())
	case item.GetRangeSubselect() != nil:
		f.formatRangeSubselect(item.GetRangeSubselect())
	default:
		// Fallback to deparse for unknown node types
		if deparseResult, err := f.deparseNode(item); err == nil {
			f.buffer.WriteString(deparseResult)
		}
	}
}

// formatRangeVar formats a table reference
func (f *postgreSQLFormatter) formatRangeVar(rangeVar *pg_query.RangeVar) {
	if rangeVar.Schemaname != "" {
		f.buffer.WriteString(rangeVar.Schemaname)
		f.buffer.WriteString(".")
	}
	f.buffer.WriteString(rangeVar.Relname)

	if rangeVar.Alias != nil && rangeVar.Alias.Aliasname != "" {
		f.buffer.WriteString(" ")
		f.buffer.WriteString(rangeVar.Alias.Aliasname)
	}
}

// formatJoinExpr formats a JOIN expression
func (f *postgreSQLFormatter) formatJoinExpr(join *pg_query.JoinExpr) {
	// Format left side
	if join.Larg != nil {
		f.formatFromItem(join.Larg)
	}

	// Determine JOIN type keyword
	var joinKeyword string
	switch join.Jointype {
	case pg_query.JoinType_JOIN_LEFT:
		joinKeyword = "LEFT JOIN"
	case pg_query.JoinType_JOIN_RIGHT:
		joinKeyword = "RIGHT JOIN"
	case pg_query.JoinType_JOIN_FULL:
		joinKeyword = "FULL JOIN"
	case pg_query.JoinType_JOIN_INNER:
		// CROSS JOIN is represented as INNER JOIN with no quals (no ON condition)
		if join.Quals == nil {
			joinKeyword = "CROSS JOIN"
		} else {
			joinKeyword = "JOIN"
		}
	default:
		joinKeyword = "JOIN"
	}

	// Add JOIN keyword with proper indentation
	f.buffer.WriteString("\n     " + joinKeyword + " ")

	// Format right side
	if join.Rarg != nil {
		f.formatFromItem(join.Rarg)
	}

	// Add ON condition (only if present, CROSS JOIN has no ON condition)
	if join.Quals != nil {
		f.buffer.WriteString(" ON ")
		f.formatExpression(join.Quals)
	}
}

// formatRangeSubselect formats a subquery in FROM clause
func (f *postgreSQLFormatter) formatRangeSubselect(subselect *pg_query.RangeSubselect) {
	// Save the current buffer state
	savedBuffer := f.buffer.String()
	tempBuffer := &strings.Builder{}
	f.buffer = tempBuffer

	// Format the subquery
	if selectStmt := subselect.Subquery.GetSelectStmt(); selectStmt != nil {
		f.formatSelectStmt(selectStmt)
	}

	// Get the formatted subquery and trim leading space
	subqueryContent := strings.TrimPrefix(tempBuffer.String(), " ")

	// Restore original buffer and append formatted content
	f.buffer = &strings.Builder{}
	f.buffer.WriteString(savedBuffer)
	f.buffer.WriteString("(")
	f.buffer.WriteString(subqueryContent)
	f.buffer.WriteString(")")

	if subselect.Alias != nil && subselect.Alias.Aliasname != "" {
		f.buffer.WriteString(" ")
		f.buffer.WriteString(subselect.Alias.Aliasname)
	}
}

// formatExpression formats a general expression
func (f *postgreSQLFormatter) formatExpression(expr *pg_query.Node) {
	switch {
	case expr.GetColumnRef() != nil:
		f.formatColumnRef(expr.GetColumnRef())
	case expr.GetAConst() != nil:
		f.formatAConst(expr.GetAConst())
	case expr.GetAExpr() != nil:
		f.formatAExpr(expr.GetAExpr())
	case expr.GetFuncCall() != nil:
		f.formatFuncCall(expr.GetFuncCall())
	case expr.GetBoolExpr() != nil:
		f.formatBoolExpr(expr.GetBoolExpr())
	case expr.GetTypeCast() != nil:
		f.formatTypeCast(expr.GetTypeCast())
	case expr.GetCaseExpr() != nil:
		f.formatCaseExpr(expr.GetCaseExpr())
	case expr.GetSubLink() != nil:
		f.formatSubLink(expr.GetSubLink())
	case expr.GetCoalesceExpr() != nil:
		f.formatCoalesceExpr(expr.GetCoalesceExpr())
	default:
		// Fallback to deparse for complex expressions
		if deparseResult, err := f.deparseNode(expr); err == nil {
			f.buffer.WriteString(deparseResult)
		}
	}
}

// formatColumnRef formats a column reference
func (f *postgreSQLFormatter) formatColumnRef(col *pg_query.ColumnRef) {
	for i, field := range col.Fields {
		if i > 0 {
			f.buffer.WriteString(".")
		}
		if str := field.GetString_(); str != nil {
			f.buffer.WriteString(str.Sval)
		}
	}
}

// formatAConst formats a constant value
func (f *postgreSQLFormatter) formatAConst(constant *pg_query.A_Const) {
	// Check for NULL first
	if constant.Isnull {
		f.buffer.WriteString("NULL")
		return
	}

	switch val := constant.Val.(type) {
	case *pg_query.A_Const_Sval:
		f.buffer.WriteString("'")
		f.buffer.WriteString(val.Sval.Sval)
		f.buffer.WriteString("'")
	case *pg_query.A_Const_Ival:
		f.buffer.WriteString(strconv.FormatInt(int64(val.Ival.Ival), 10))
	case *pg_query.A_Const_Fval:
		f.buffer.WriteString(val.Fval.Fval)
	case *pg_query.A_Const_Boolval:
		if val.Boolval.Boolval {
			f.buffer.WriteString("true")
		} else {
			f.buffer.WriteString("false")
		}
	case *pg_query.A_Const_Bsval:
		f.buffer.WriteString(val.Bsval.Bsval)
	default:
		// Fallback to deparse
		if deparseResult, err := f.deparseNode(&pg_query.Node{Node: &pg_query.Node_AConst{AConst: constant}}); err == nil {
			f.buffer.WriteString(deparseResult)
		}
	}
}

// formatAExpr formats an A_Expr (binary/unary expressions)
func (f *postgreSQLFormatter) formatAExpr(expr *pg_query.A_Expr) {
	// Format left operand
	if expr.Lexpr != nil {
		f.formatExpression(expr.Lexpr)
	}

	// Format operator
	if len(expr.Name) > 0 {
		f.buffer.WriteString(" ")
		for i, nameNode := range expr.Name {
			if i > 0 {
				f.buffer.WriteString(".")
			}
			if str := nameNode.GetString_(); str != nil {
				f.buffer.WriteString(str.Sval)
			}
		}
		f.buffer.WriteString(" ")
	}

	// Format right operand
	if expr.Rexpr != nil {
		f.formatExpression(expr.Rexpr)
	}
}

// formatFuncCall formats a function call
func (f *postgreSQLFormatter) formatFuncCall(funcCall *pg_query.FuncCall) {
	// Format function name
	for i, nameNode := range funcCall.Funcname {
		if i > 0 {
			f.buffer.WriteString(".")
		}
		if str := nameNode.GetString_(); str != nil {
			f.buffer.WriteString(str.Sval)
		}
	}

	// Format arguments
	f.buffer.WriteString("(")

	// Handle aggregate functions with star (like COUNT(*))
	if funcCall.AggStar {
		f.buffer.WriteString("*")
	} else {
		// Regular arguments
		for i, arg := range funcCall.Args {
			if i > 0 {
				f.buffer.WriteString(", ")
			}
			f.formatExpression(arg)
		}
	}
	f.buffer.WriteString(")")

	// Handle window functions (OVER clause)
	if funcCall.Over != nil {
		f.buffer.WriteString(" OVER (")
		f.formatWindowDef(funcCall.Over)
		f.buffer.WriteString(")")
	}
}

// formatBoolExpr formats boolean expressions (AND, OR, NOT)
func (f *postgreSQLFormatter) formatBoolExpr(boolExpr *pg_query.BoolExpr) {
	switch boolExpr.Boolop {
	case pg_query.BoolExprType_AND_EXPR:
		for i, arg := range boolExpr.Args {
			if i > 0 {
				f.buffer.WriteString(" AND ")
			}
			f.formatExpression(arg)
		}
	case pg_query.BoolExprType_OR_EXPR:
		for i, arg := range boolExpr.Args {
			if i > 0 {
				f.buffer.WriteString(" OR ")
			}
			f.formatExpression(arg)
		}
	case pg_query.BoolExprType_NOT_EXPR:
		f.buffer.WriteString("NOT ")
		if len(boolExpr.Args) > 0 {
			f.formatExpression(boolExpr.Args[0])
		}
	}
}

// formatTypeCast formats a type cast expression
func (f *postgreSQLFormatter) formatTypeCast(typeCast *pg_query.TypeCast) {
	// Special handling for INTERVAL type casts
	if typeCast.TypeName != nil && len(typeCast.TypeName.Names) > 0 {
		// Get the type name (last element in the names array)
		typeName := ""
		if str := typeCast.TypeName.Names[len(typeCast.TypeName.Names)-1].GetString_(); str != nil {
			typeName = str.Sval
		}

		// Check if this is an interval cast with a string constant
		if typeName == "interval" && typeCast.Arg != nil {
			if aConst := typeCast.Arg.GetAConst(); aConst != nil {
				if sval := aConst.GetSval(); sval != nil {
					// Format as INTERVAL 'value' instead of 'value'::interval
					f.buffer.WriteString("INTERVAL '")
					f.buffer.WriteString(sval.Sval)
					f.buffer.WriteString("'")
					return
				}
			}
		}
	}

	// Default formatting for other type casts
	if typeCast.Arg != nil {
		f.formatExpression(typeCast.Arg)
	}

	f.buffer.WriteString("::")

	if typeCast.TypeName != nil {
		f.formatTypeName(typeCast.TypeName)
	}
}

// formatTypeName formats a type name
func (f *postgreSQLFormatter) formatTypeName(typeName *pg_query.TypeName) {
	for i, nameNode := range typeName.Names {
		if i > 0 {
			f.buffer.WriteString(".")
		}
		if str := nameNode.GetString_(); str != nil {
			f.buffer.WriteString(str.Sval)
		}
	}
}

// formatGroupByClause formats GROUP BY clause
func (f *postgreSQLFormatter) formatGroupByClause(groupBy []*pg_query.Node) {
	for i, item := range groupBy {
		if i > 0 {
			f.buffer.WriteString(", ")
		}
		f.formatExpression(item)
	}
}

// formatOrderByClause formats ORDER BY clause
func (f *postgreSQLFormatter) formatOrderByClause(orderBy []*pg_query.Node) {
	for i, item := range orderBy {
		if i > 0 {
			f.buffer.WriteString(", ")
		}
		if sortBy := item.GetSortBy(); sortBy != nil {
			f.formatExpression(sortBy.Node)
			if sortBy.SortbyDir == pg_query.SortByDir_SORTBY_DESC {
				f.buffer.WriteString(" DESC")
			}
		}
	}
}

// deparseNode is a helper to deparse individual nodes as fallback
func (f *postgreSQLFormatter) deparseNode(node *pg_query.Node) (string, error) {
	stmt := &pg_query.RawStmt{Stmt: node}
	parseResult := &pg_query.ParseResult{Stmts: []*pg_query.RawStmt{stmt}}
	return pg_query.Deparse(parseResult)
}

// formatCaseExpr formats CASE expressions
func (f *postgreSQLFormatter) formatCaseExpr(caseExpr *pg_query.CaseExpr) {
	f.buffer.WriteString("CASE")

	// CASE with an argument (CASE expr WHEN ...)
	if caseExpr.Arg != nil {
		f.buffer.WriteString(" ")
		f.formatExpression(caseExpr.Arg)
	}

	// Format WHEN clauses
	for _, whenClause := range caseExpr.Args {
		if when := whenClause.GetCaseWhen(); when != nil {
			f.buffer.WriteString(" WHEN ")
			f.formatExpression(when.Expr)
			f.buffer.WriteString(" THEN ")
			f.formatExpression(when.Result)
		}
	}

	// Format ELSE clause
	if caseExpr.Defresult != nil {
		f.buffer.WriteString(" ELSE ")
		f.formatExpression(caseExpr.Defresult)
	}

	f.buffer.WriteString(" END")
}

// formatCoalesceExpr formats COALESCE expressions
func (f *postgreSQLFormatter) formatCoalesceExpr(coalesceExpr *pg_query.CoalesceExpr) {
	f.buffer.WriteString("COALESCE(")
	
	// Format arguments
	for i, arg := range coalesceExpr.Args {
		if i > 0 {
			f.buffer.WriteString(", ")
		}
		f.formatExpression(arg)
	}
	
	f.buffer.WriteString(")")
}

// formatWindowDef formats window definition (OVER clause)
func (f *postgreSQLFormatter) formatWindowDef(windowDef *pg_query.WindowDef) {
	needsSpace := false

	// PARTITION BY clause
	if len(windowDef.PartitionClause) > 0 {
		f.buffer.WriteString("PARTITION BY ")
		for i, partExpr := range windowDef.PartitionClause {
			if i > 0 {
				f.buffer.WriteString(", ")
			}
			f.formatExpression(partExpr)
		}
		needsSpace = true
	}

	// ORDER BY clause
	if len(windowDef.OrderClause) > 0 {
		if needsSpace {
			f.buffer.WriteString(" ")
		}
		f.buffer.WriteString("ORDER BY ")
		for i, sortExpr := range windowDef.OrderClause {
			if i > 0 {
				f.buffer.WriteString(", ")
			}
			if sortBy := sortExpr.GetSortBy(); sortBy != nil {
				f.formatExpression(sortBy.Node)
				if sortBy.SortbyDir == pg_query.SortByDir_SORTBY_DESC {
					f.buffer.WriteString(" DESC")
				}
			}
		}
	}
}

// formatSubLink formats subquery expressions (IN, EXISTS, etc.)
func (f *postgreSQLFormatter) formatSubLink(subLink *pg_query.SubLink) {
	// For now, use deparse as fallback
	// This handles complex subquery expressions that need special formatting
	if deparseResult, err := f.deparseNode(&pg_query.Node{Node: &pg_query.Node_SubLink{SubLink: subLink}}); err == nil {
		f.buffer.WriteString(deparseResult)
	}
}
