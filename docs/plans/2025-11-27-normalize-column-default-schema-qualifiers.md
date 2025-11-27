# Normalize Column Default Schema Qualifiers

**Date:** 2025-11-27
**Status:** Approved for implementation

## Problem Statement

pgschema generates DDL with schema-qualified function calls in column defaults (e.g., `DEFAULT public.get_default_status()`), but PostgreSQL's `pg_get_expr()` strips schema qualifiers for functions in the same schema as the table. This causes drift detection - the second plan/apply detects false differences and attempts to re-apply the same default expression.

### Reproduction

```bash
PGSCHEMA_TEST_FILTER="dependency/function_to_table" go test -v ./cmd -run TestPlanAndApply
```

**Expected:** No changes detected on second apply (idempotent)
**Actual:** Generates `ALTER TABLE users ALTER COLUMN status SET DEFAULT public.get_default_status();` on second apply

## Root Cause

1. User writes: `DEFAULT get_default_status()` (no qualifier) in `new.sql`
2. pgschema applies it to embedded postgres, which stores it as-is
3. pgschema dumps it using `pg_get_expr()` → returns `get_default_status()` (no qualifier)
4. pgschema generates plan SQL with `DEFAULT public.get_default_status()` (WITH qualifier)
5. That gets applied to target database, which stores it WITH the qualifier
6. Second run: pgschema reads back `get_default_status()` (no qualifier) via `pg_get_expr()`
7. Compares with plan which has `public.get_default_status()` → MISMATCH

**Key insight:** PostgreSQL's `pg_get_expr()` automatically strips schema qualifiers for functions in the same schema as the table, but pgschema's DDL generation doesn't account for this normalization.

## Solution

Add a normalization step in the diff/DDL generation phase that strips schema qualifiers from function calls in column defaults when the function is in the same schema as the table.

### Design Decisions

**Why normalize during DDL generation instead of extraction?**
- The IR should preserve the original expression from `pg_get_expr()` without modification
- Normalization at DDL generation time provides context (target schema, comparison state)
- Consistent with existing type qualification logic in codebase

**Why string-based approach?**
- Sufficient for this specific case (function calls in column defaults)
- Low risk of false matches (column defaults are typically simple expressions)
- Simple, performant, and maintainable
- Falls back gracefully if pattern doesn't match

**Scope:**
- Only normalize function calls in the same schema as the table
- Preserve cross-schema references (e.g., `other_schema.func()`)
- Preserve pg_catalog references (though `pg_get_expr()` already handles these)

## Implementation

### New Helper Function

**File:** `internal/diff/table.go`

```go
// normalizeDefaultExpr removes schema qualifiers from function calls
// when the function is in the same schema as the table.
// This matches PostgreSQL's behavior where pg_get_expr() returns
// unqualified function names for functions in the same schema.
func normalizeDefaultExpr(defaultExpr string, tableSchema string) string {
    if defaultExpr == "" || tableSchema == "" {
        return defaultExpr
    }

    // Pattern: schema.function_name(
    // Replace "tableSchema." with "" when followed by identifier and (
    prefix := tableSchema + "."

    if strings.Contains(defaultExpr, prefix) {
        // Use regex to match schema.identifier( pattern
        // Example: public.get_status() -> get_status()
        pattern := regexp.MustCompile(regexp.QuoteMeta(prefix) + `([a-zA-Z_][a-zA-Z0-9_]*)\(`)
        defaultExpr = pattern.ReplaceAllString(defaultExpr, `${1}(`)
    }

    return defaultExpr
}
```

### Integration Points

Apply normalization in DDL generation functions in `internal/diff/table.go`:

1. **generateAddColumnDDL()** - when adding new columns with defaults
2. **generateAlterColumnDefaultDDL()** - when changing column defaults
3. Any location where `column.DefaultValue` is used to generate DDL

**Example usage:**
```go
if column.DefaultValue != nil {
    normalizedDefault := normalizeDefaultExpr(*column.DefaultValue, tableSchema)
    ddl := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;",
        quotedTableName,
        ir.QuoteIdentifier(column.Name),
        normalizedDefault)
}
```

## Testing

### Existing Test

`testdata/diff/dependency/function_to_table/` already captures this scenario.

**After fix:**
- `plan.sql` should generate: `DEFAULT get_default_status()` (no qualifier)
- `plan.txt` should show the same
- Integration test should pass: applying twice produces no changes

### Validation Commands

```bash
# 1. Run diff test (fast, no database)
env PGSCHEMA_TEST_FILTER="dependency/function_to_table" go test -v ./internal/diff -run TestDiffFromFiles

# 2. Run integration test (applies to real DB)
env PGSCHEMA_TEST_FILTER="dependency/function_to_table" go test -v ./cmd -run TestPlanAndApply

# 3. Regenerate expected outputs
env PGSCHEMA_TEST_FILTER="dependency/function_to_table" go test -v ./cmd -run TestPlanAndApply --generate
```

### Additional Test Coverage (Future)

Consider adding test cases for:
- Cross-schema function references: `DEFAULT other_schema.func()` (should keep qualifier)
- Multiple functions: `DEFAULT COALESCE(public.func1(), public.func2())`
- Nested functions: `DEFAULT public.outer(public.inner())`

## Edge Cases

| Case | Behavior | Notes |
|------|----------|-------|
| Nested function calls | Both qualifiers stripped | `public.outer(public.inner())` → `outer(inner())` |
| Cross-schema references | Qualifier preserved | `other_schema.func()` → unchanged |
| Schema names with special chars | Handled | `regexp.QuoteMeta()` escapes properly |
| String literals with schema names | Low risk | Pattern requires `(` after identifier |
| Case sensitivity | Assumes lowercase | Acceptable - PostgreSQL folds to lowercase |
| Multiple spaces | Not handled | `public.  func()` - rare in practice |

## Known Limitations

- String-based approach, not a full SQL parser
- May have false positives in exotic formatting cases
- Acceptable trade-off for simplicity and performance

## Future Improvements

- If issues arise, use pg_query_go to parse expressions properly
- Add flag to disable normalization if exact qualification is desired
- Extend to other expression contexts if similar issues found

## References

- PostgreSQL function `pg_get_expr(expr, relation, pretty)` documentation
- Existing type qualification logic in `ir/queries/queries.sql` lines 76-96
- Test case: `testdata/diff/dependency/function_to_table/`
