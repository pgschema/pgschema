---
name: PostgreSQL Syntax Reference
description: Consult PostgreSQL's parser and grammar (gram.y) to understand SQL syntax, DDL statement structure, and parsing rules when implementing pgschema features
---

# PostgreSQL Syntax Reference

Use this skill when you need to understand PostgreSQL's SQL syntax, DDL statement structure, or how PostgreSQL parses specific SQL constructs. This is essential for correctly parsing SQL files and generating valid DDL in pgschema.

## When to Use This Skill

Invoke this skill when:
- Implementing new SQL statement parsing in `ir/parser.go`
- Debugging SQL parsing issues with pg_query_go
- Understanding complex SQL syntax (CREATE TABLE, CREATE TRIGGER, etc.)
- Generating DDL statements in `internal/diff/*.go`
- Validating SQL statement structure
- Understanding precedence and grammar rules
- Learning about PostgreSQL-specific syntax extensions

## Source Code Locations

**Main parser directory**: https://github.com/postgres/postgres/blob/master/src/backend/parser/

**Key files to reference**:

### Grammar and Lexer
- `gram.y` - **Main grammar file** - Yacc/Bison grammar defining PostgreSQL SQL syntax
- `scan.l` - Lexical scanner (Flex/Lex) - tokenization rules
- `keywords.c` - Reserved and non-reserved keywords

### Parser Implementation
- `parse_clause.c` - Parsing of clauses (WHERE, GROUP BY, ORDER BY, etc.)
- `parse_expr.c` - Expression parsing (operators, function calls, etc.)
- `parse_type.c` - Type name parsing and resolution
- `parse_relation.c` - Table and relation parsing
- `parse_target.c` - Target list parsing (SELECT list, etc.)
- `parse_func.c` - Function call parsing
- `parse_utilcmd.c` - **Utility commands** (DDL statements like CREATE, ALTER, DROP)

### Analysis and Transformation
- `analyze.c` - Post-parse analysis
- `parse_node.c` - Parse node creation utilities

## Step-by-Step Workflow

### 1. Identify the SQL Statement Type

Determine what kind of SQL you're working with:

| Statement Type | gram.y Section | parse_utilcmd.c Function |
|----------------|----------------|-------------------------|
| CREATE TABLE | `CreateStmt` | `transformCreateStmt()` |
| ALTER TABLE | `AlterTableStmt` | `transformAlterTableStmt()` |
| CREATE INDEX | `IndexStmt` | `transformIndexStmt()` |
| CREATE TRIGGER | `CreateTrigStmt` | `transformCreateTrigStmt()` |
| CREATE FUNCTION | `CreateFunctionStmt` | `transformCreateFunctionStmt()` |
| CREATE PROCEDURE | `CreateFunctionStmt` | (procedures are functions) |
| CREATE VIEW | `ViewStmt` | `transformViewStmt()` |
| CREATE MATERIALIZED VIEW | `CreateMatViewStmt` | - |
| CREATE SEQUENCE | `CreateSeqStmt` | `transformCreateSeqStmt()` |
| CREATE TYPE | `CreateEnumStmt`, `CreateDomainStmt`, `CompositeTypeStmt` | - |
| CREATE POLICY | `CreatePolicyStmt` | `transformCreatePolicyStmt()` |
| COMMENT ON | `CommentStmt` | - |

### 2. Locate the Grammar Rule in gram.y

Search gram.y for the statement's production rule:

**Example - Finding CREATE TRIGGER syntax**:
```bash
# In the postgres repository
grep -n "CreateTrigStmt:" src/backend/parser/gram.y
```

**What to look for**:
- The production rule name (e.g., `CreateTrigStmt:`)
- Alternative syntaxes (multiple `|` branches)
- Optional elements (`opt_*` rules)
- List constructs (`*_list` rules)
- Terminal tokens (keywords, literals)

### 3. Understand the Grammar Structure

**gram.y uses Yacc/Bison syntax**:

```yacc
CreateTrigStmt:
    CREATE opt_or_replace TRIGGER name TriggerActionTime TriggerEvents ON
    qualified_name TriggerReferencing TriggerForSpec TriggerWhen
    EXECUTE FUNCTION_or_PROCEDURE func_name '(' TriggerFuncArgs ')'
    {
        CreateTrigStmt *n = makeNode(CreateTrigStmt);
        n->trigname = $4;
        n->relation = $8;
        n->funcname = $13;
        /* ... */
        $$ = (Node *)n;
    }
```

**Key elements**:
- **Terminals** (uppercase): Keywords like `CREATE`, `TRIGGER`, `ON`
- **Non-terminals** (lowercase): Other grammar rules like `name`, `qualified_name`
- **Actions** (`{ ... }`): C code that builds the parse tree
- **Alternatives** (`|`): Different ways to write the same statement
- **Optional elements**: Rules prefixed with `opt_`

### 4. Trace Through Related Rules

Follow the grammar rules to understand the complete syntax:

**Example - Understanding trigger events**:
```yacc
TriggerEvents:
    TriggerOneEvent
    | TriggerEvents OR TriggerOneEvent

TriggerOneEvent:
    INSERT
    | DELETE
    | UPDATE
    | UPDATE OF columnList
    | TRUNCATE
```

This shows:
- Triggers can have multiple events combined with OR
- UPDATE can optionally specify columns with `OF columnList`

### 5. Cross-Reference with parse_utilcmd.c

After understanding the grammar, check how PostgreSQL transforms the parsed statement:

**Example - How CREATE TRIGGER is processed**:
```c
// In parse_utilcmd.c
static void
transformCreateTrigStmt(CreateTrigStmt *stmt, const char *queryString)
{
    // Validation and transformation logic
    // - Check trigger name conflicts
    // - Validate trigger function exists
    // - Process WHEN condition
    // - Handle constraint triggers
}
```

### 6. Apply to pgschema

Use this understanding in pgschema:

**For parsing** (`ir/parser.go`):
- pgschema uses `pg_query_go` which wraps libpg_query (based on PostgreSQL's parser)
- Parse tree structure matches gram.y production rules
- Access parsed nodes to extract information

**For DDL generation** (`internal/diff/*.go`):
- Follow gram.y syntax exactly
- Use proper keyword ordering
- Include all required elements
- Quote identifiers correctly

## Key Grammar Concepts

### Optional Elements

Grammar rules prefixed with `opt_` are optional:

```yacc
opt_or_replace:
    OR REPLACE     { $$ = true; }
    | /* EMPTY */  { $$ = false; }
```

This means `CREATE OR REPLACE TRIGGER ...` and `CREATE TRIGGER ...` are both valid.

### Lists

Lists are typically defined recursively:

```yacc
columnList:
    columnElem                     { $$ = list_make1($1); }
    | columnList ',' columnElem    { $$ = lappend($1, $3); }
```

### Alternatives

Use `|` to show different syntax options:

```yacc
TriggerActionTime:
    BEFORE     { $$ = TRIGGER_TYPE_BEFORE; }
    | AFTER    { $$ = TRIGGER_TYPE_AFTER; }
    | INSTEAD OF { $$ = TRIGGER_TYPE_INSTEAD; }
```

### Precedence

Operator precedence is defined at the top of gram.y:

```yacc
%left OR
%left AND
%right NOT
%nonassoc IS ISNULL NOTNULL
%nonassoc '<' '>' '=' LESS_EQUALS GREATER_EQUALS NOT_EQUALS
```

## Common Grammar Patterns

### CREATE Statement Pattern

Most CREATE statements follow this pattern:

```yacc
CreateSomethingStmt:
    CREATE opt_or_replace SOMETHING name definition_elements
```

### ALTER Statement Pattern

```yacc
AlterSomethingStmt:
    ALTER SOMETHING name alter_action
    | ALTER SOMETHING IF_P EXISTS name alter_action
```

### DROP Statement Pattern

```yacc
DropSomethingStmt:
    DROP SOMETHING name opt_drop_behavior
    | DROP SOMETHING IF_P EXISTS name opt_drop_behavior
```

## Important SQL Constructs for pgschema

### Table Columns with Constraints

```yacc
columnDef:
    ColId Typename opt_column_storage ColQualList
    | ColId Typename opt_column_storage GeneratedConstraintElem
    | ColId Typename opt_column_storage GENERATED generated_when AS IDENTITY_P OptParenthesizedSeqOptList
```

This covers:
- Regular columns: `column_name type`
- Generated columns: `column_name type GENERATED ALWAYS AS (expr) STORED`
- Identity columns: `column_name type GENERATED ALWAYS AS IDENTITY`

### Trigger WHEN Clause

```yacc
TriggerWhen:
    WHEN '(' a_expr ')'    { $$ = $3; }
    | /* EMPTY */          { $$ = NULL; }
```

### Index Elements

```yacc
index_elem:
    ColId opt_collate opt_class opt_asc_desc opt_nulls_order
    | func_expr_windowless opt_collate opt_class opt_asc_desc opt_nulls_order
    | '(' a_expr ')' opt_collate opt_class opt_asc_desc opt_nulls_order
```

This shows indexes can be on:
- Simple columns
- Function expressions (functional indexes)
- Arbitrary expressions (expression indexes)

### Foreign Key Options

```yacc
ConstraintAttributeSpec:
    ON DELETE key_action
    | ON UPDATE key_action
    | DEFERRABLE
    | NOT DEFERRABLE
    | INITIALLY DEFERRED
    | INITIALLY IMMEDIATE
```

## Keywords and Reserved Words

Check `keywords.c` for keyword classification:

**Reserved keywords**: Cannot be used as identifiers without quoting
- `SELECT`, `FROM`, `WHERE`, `CREATE`, `TABLE`, etc.

**Type function name keywords**: Can be used as function or type names
- `CHAR`, `CHARACTER`, `VARCHAR`, etc.

**Unreserved keywords**: Can be used as identifiers
- `ABORT`, `ABSOLUTE`, `ACCESS`, `ACTION`, etc.

**Impact on pgschema**: When generating DDL, quote identifiers that match reserved keywords.

## Examples

### Example 1: Understanding CREATE TABLE LIKE

**In gram.y**:
```yacc
TableLikeClause:
    LIKE qualified_name TableLikeOptionList
```

**TableLikeOptionList**:
```yacc
TableLikeOptionList:
    TableLikeOptionList INCLUDING TableLikeOption
    | TableLikeOptionList EXCLUDING TableLikeOption
    | /* EMPTY */
```

**TableLikeOption**:
```yacc
TableLikeOption:
    COMMENTS | CONSTRAINTS | DEFAULTS | IDENTITY_P | GENERATED | INDEXES | STATISTICS | STORAGE | ALL
```

**This tells us**:
- `LIKE table_name` is the basic syntax
- Can include/exclude specific features: `INCLUDING ALL`, `EXCLUDING INDEXES`, etc.
- Multiple options can be combined

**pgschema usage** (`ir/parser.go`):
```go
// Parse CREATE TABLE ... LIKE statements
if createTableStmt.Inherits != nil {
    for _, inherit := range createTableStmt.Inherits {
        if inherit.Relpersistence == "l" { // LIKE clause
            table.LikeClause = &LikeClause{
                Parent: inherit.Relname,
                Options: parseLikeOptions(inherit),
            }
        }
    }
}
```

### Example 2: Understanding Constraint Triggers

**In gram.y**:
```yacc
ConstraintAttributeSpec:
    DEFERRABLE           { $$ = CAS_DEFERRABLE; }
    | NOT DEFERRABLE     { $$ = CAS_NOT_DEFERRABLE; }
    | INITIALLY DEFERRED { $$ = CAS_INITIALLY_DEFERRED; }
    | INITIALLY IMMEDIATE { $$ = CAS_INITIALLY_IMMEDIATE; }
```

**For constraint triggers**:
```yacc
CreateTrigStmt:
    CREATE opt_or_replace CONSTRAINT TRIGGER name ...
```

**This tells us**:
- Constraint triggers use `CREATE CONSTRAINT TRIGGER`
- Can be `DEFERRABLE` or `NOT DEFERRABLE`
- Can be `INITIALLY DEFERRED` or `INITIALLY IMMEDIATE`

**pgschema DDL generation** (`internal/diff/trigger.go`):
```go
func generateCreateTrigger(trigger *ir.Trigger) string {
    var sql strings.Builder
    sql.WriteString("CREATE ")
    if trigger.IsConstraint {
        sql.WriteString("CONSTRAINT ")
    }
    sql.WriteString("TRIGGER ")
    sql.WriteString(quoteIdentifier(trigger.Name))
    // ...
    if trigger.Deferrable {
        sql.WriteString(" DEFERRABLE")
    }
    if trigger.InitiallyDeferred {
        sql.WriteString(" INITIALLY DEFERRED")
    }
    return sql.String()
}
```

### Example 3: Understanding Expression Indexes

**In gram.y**:
```yacc
index_elem:
    ColId opt_collate opt_class opt_asc_desc opt_nulls_order
    {
        $$ = makeIndexElem($1, NULL, NULL, $2, $3, $4, $5, NULL);
    }
    | func_expr_windowless opt_collate opt_class opt_asc_desc opt_nulls_order
    {
        $$ = makeIndexElem(NULL, $1, NULL, $2, $3, $4, $5, NULL);
    }
    | '(' a_expr ')' opt_collate opt_class opt_asc_desc opt_nulls_order
    {
        $$ = makeIndexElem(NULL, NULL, $2, $4, $5, $6, $7, NULL);
    }
```

**This tells us**:
- Index elements can be:
  1. Column names: `CREATE INDEX idx ON table (column)`
  2. Function calls: `CREATE INDEX idx ON table (lower(column))`
  3. Arbitrary expressions: `CREATE INDEX idx ON table ((column + 1))`
- Note the extra parentheses for arbitrary expressions: `(( ... ))`

**pgschema parsing consideration**:
```go
// When parsing index definitions, handle all three forms:
// 1. Simple column reference
// 2. Function expression
// 3. Arbitrary expression (needs extra parens in DDL)
```

### Example 4: Understanding GENERATED Columns

**In gram.y**:
```yacc
GeneratedConstraintElem:
    GENERATED generated_when AS '(' a_expr ')' STORED
    {
        Constraint *n = makeNode(Constraint);
        n->contype = CONSTR_GENERATED;
        n->generated_when = $2;
        n->raw_expr = $5;
        n->cooked_expr = NULL;
        n->location = @1;
        $$ = (Node *)n;
    }

generated_when:
    ALWAYS    { $$ = ATTRIBUTE_IDENTITY_ALWAYS; }
    | BY DEFAULT { $$ = ATTRIBUTE_IDENTITY_BY_DEFAULT; }
```

**This tells us**:
- Generated columns: `GENERATED ALWAYS AS (expression) STORED`
- Identity columns: `GENERATED ALWAYS AS IDENTITY` or `GENERATED BY DEFAULT AS IDENTITY`
- The expression must be in parentheses
- Must include `STORED` keyword for computed columns

## Working with pg_query_go

pgschema uses `pg_query_go/v6` which provides Go bindings to libpg_query (PostgreSQL parser):

### Parse Tree Structure

The parse tree from pg_query_go matches gram.y structure:

```go
import "github.com/pganalyze/pg_query_go/v6"

result, err := pg_query.Parse(sqlStatement)
if err != nil {
    return err
}

// result.Stmts contains parsed statement nodes
// Structure matches gram.y production rules
for _, stmt := range result.Stmts {
    switch node := stmt.Stmt.Node.(type) {
    case *pg_query.Node_CreateStmt:
        // Handle CREATE TABLE
    case *pg_query.Node_CreateTrigStmt:
        // Handle CREATE TRIGGER
    case *pg_query.Node_IndexStmt:
        // Handle CREATE INDEX
    }
}
```

### Accessing Grammar Elements

Map gram.y rules to pg_query_go node fields:

**gram.y**:
```yacc
CreateTrigStmt:
    CREATE TRIGGER name TriggerActionTime TriggerEvents ON qualified_name
```

**pg_query_go**:
```go
createTrigStmt := node.CreateTrigStmt
triggerName := createTrigStmt.Trigname  // maps to 'name'
timing := createTrigStmt.Timing          // maps to 'TriggerActionTime'
events := createTrigStmt.Events          // maps to 'TriggerEvents'
relation := createTrigStmt.Relation      // maps to 'qualified_name'
```

## Debugging Tips

### 1. Test Grammar Interactively

Clone postgres and build the parser:
```bash
git clone https://github.com/postgres/postgres.git
cd postgres
./configure
make -C src/backend/parser
```

### 2. Use pg_query_go for Validation

Test parsing in pgschema:
```go
import "github.com/pganalyze/pg_query_go/v6"

sql := "CREATE TRIGGER ..."
result, err := pg_query.Parse(sql)
if err != nil {
    // Invalid syntax
    fmt.Println("Parse error:", err)
}
// Valid syntax - examine result.Stmts
```

### 3. Compare with PostgreSQL Behavior

Test actual PostgreSQL behavior:
```bash
psql -c "CREATE TRIGGER ..."
# If PostgreSQL accepts it, the syntax is valid
# Use \d+ to see how PostgreSQL formats it
```

### 4. Check gram.y Comments

gram.y contains helpful comments explaining syntax choices and historical notes.

### 5. Search for Examples in Tests

PostgreSQL's test suite has extensive SQL examples:
```bash
# In postgres repo
find src/test/regress/sql -name "*.sql" -exec grep -l "CREATE TRIGGER" {} \;
```

## Version Differences

PostgreSQL syntax evolves across versions:

- **PostgreSQL 14**: Added `COMPRESSION` clause for tables
- **PostgreSQL 15**: Added `MERGE` statement, `UNIQUE NULLS NOT DISTINCT`
- **PostgreSQL 16**: Added SQL/JSON functions
- **PostgreSQL 17**: Added `MERGE` enhancements, incremental view maintenance

**For pgschema (supports 14-18)**:
- Check gram.y history to see when features were added
- Add version detection in parser if needed
- Test across all supported versions

## Verification Checklist

After consulting gram.y and implementing in pgschema:

- [ ] Grammar rule fully understood from gram.y
- [ ] All syntax alternatives identified
- [ ] Optional elements properly handled
- [ ] List constructs correctly parsed
- [ ] Keywords and quoting rules followed
- [ ] pg_query_go parse tree structure matches expectations
- [ ] DDL generation produces valid PostgreSQL syntax
- [ ] Test case added with sample SQL
- [ ] Tested against PostgreSQL (manually or via integration test)
- [ ] Works across PostgreSQL versions 14-18

## Quick Reference

**Finding syntax in gram.y**:
```bash
# Search for statement type
grep -n "CreateTrigStmt:" src/backend/parser/gram.y

# Find keyword definitions
grep -n "^TRIGGER" src/backend/parser/gram.y

# Understand an option
grep -A 10 "TriggerWhen:" src/backend/parser/gram.y
```

**Understanding precedence**:
```bash
# Look at top of gram.y
head -100 src/backend/parser/gram.y | grep -A 50 "%left\|%right\|%nonassoc"
```

**Find utility command handling**:
```bash
grep -n "transformCreateTrigStmt" src/backend/parser/parse_utilcmd.c
```
