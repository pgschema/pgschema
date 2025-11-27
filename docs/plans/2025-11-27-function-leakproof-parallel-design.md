# Function LEAKPROOF and PARALLEL Support Design

**Date:** 2025-11-27
**Status:** Approved
**Scope:** Add full support for LEAKPROOF and PARALLEL attributes on PostgreSQL functions

## Overview

Extend pgschema to properly handle PostgreSQL function attributes LEAKPROOF and PARALLEL (SAFE/UNSAFE/RESTRICTED) throughout the dump/plan/apply workflow. This includes IR representation, database inspection, dump formatting, and migration generation.

## Requirements

### Functional Requirements

1. **Complete PARALLEL support**: All three PostgreSQL parallel safety levels
   - `PARALLEL SAFE` - Function can run in parallel workers
   - `PARALLEL UNSAFE` - Function cannot run in parallel (default)
   - `PARALLEL RESTRICTED` - Can run in parallel but restricted to leader

2. **LEAKPROOF support**: Boolean attribute indicating function won't leak argument information
   - Important for row-level security contexts
   - Defaults to false in PostgreSQL

3. **Migration detection**: Generate ALTER FUNCTION statements when attributes change
   - `ALTER FUNCTION ... LEAKPROOF` / `NOT LEAKPROOF`
   - `ALTER FUNCTION ... PARALLEL {SAFE|UNSAFE|RESTRICTED}`

4. **Hybrid output approach**:
   - Store explicit values in IR (no ambiguity in comparisons)
   - Output only non-default values in dumps (clean, readable)
   - Matches PostgreSQL conventions and existing pgschema patterns

## Design

### 1. IR Structure Changes

**File:** `ir/ir.go`

Add two new fields to the `Function` struct:

```go
type Function struct {
    Schema            string       `json:"schema"`
    Name              string       `json:"name"`
    Definition        string       `json:"definition"`
    ReturnType        string       `json:"return_type"`
    Language          string       `json:"language"`
    Parameters        []*Parameter `json:"parameters,omitempty"`
    Comment           string       `json:"comment,omitempty"`
    Volatility        string       `json:"volatility,omitempty"`
    IsStrict          bool         `json:"is_strict,omitempty"`
    IsSecurityDefiner bool         `json:"is_security_definer,omitempty"`
    IsLeakproof       bool         `json:"is_leakproof,omitempty"`        // NEW
    Parallel          string       `json:"parallel,omitempty"`            // NEW
}
```

**Field specifications:**
- `IsLeakproof`: Boolean, defaults to `false` (matches PostgreSQL default)
- `Parallel`: String with valid values `"SAFE"`, `"UNSAFE"`, `"RESTRICTED"`, defaults to `"UNSAFE"`
- Both use `omitempty` JSON tag for clean serialization

### 2. Database Inspector Changes

**File:** `ir/inspector.go`

Update `inspectFunctions()` to extract attributes from `pg_catalog.pg_proc`:

**System catalog columns:**
- `proleakproof` (boolean) → `IsLeakproof`
- `proparallel` (char) → `Parallel`
  - `'s'` → `"SAFE"`
  - `'u'` → `"UNSAFE"`
  - `'r'` → `"RESTRICTED"`

**Query addition:**
```sql
SELECT
    ...existing columns...,
    p.proleakproof,
    p.proparallel
FROM pg_catalog.pg_proc p
...
```

**Mapping logic:**
```go
func.IsLeakproof = proleakproof

switch proparallel {
case 's':
    func.Parallel = "SAFE"
case 'r':
    func.Parallel = "RESTRICTED"
case 'u':
    func.Parallel = "UNSAFE"
default:
    func.Parallel = "UNSAFE" // Defensive default
}
```

### 3. Dump Output Logic

**File:** `internal/dump/dump.go`

Update function formatting to output attributes only when non-default:

**Output rules:**
- Output `LEAKPROOF` only when `IsLeakproof == true`
- Output `PARALLEL SAFE` or `PARALLEL RESTRICTED` only when `Parallel != "UNSAFE"`
- Never output `NOT LEAKPROOF` or `PARALLEL UNSAFE` (they're defaults)

**Attribute ordering** (matching pg_dump):
1. `LANGUAGE`
2. Volatility (`IMMUTABLE`/`STABLE`/`VOLATILE`)
3. `PARALLEL {SAFE|RESTRICTED}` (if not UNSAFE)
4. `LEAKPROOF` (if true)
5. `STRICT` (if true)
6. `SECURITY DEFINER` (if true)

**Example output:**
```sql
CREATE FUNCTION safe_add(a integer, b integer)
RETURNS integer
LANGUAGE sql
IMMUTABLE
PARALLEL SAFE
LEAKPROOF
STRICT
AS $$
    SELECT a + b;
$$;
```

### 4. Diff and Migration Logic

**File:** `internal/diff/function.go`

Detect attribute changes and generate ALTER statements:

**Detection:**
- Compare `IsLeakproof` between old and new function
- Compare `Parallel` between old and new function
- Only applies when function signature is unchanged

**Migration SQL:**
```sql
-- LEAKPROOF changes
ALTER FUNCTION schema.function_name(arg_types) LEAKPROOF;
ALTER FUNCTION schema.function_name(arg_types) NOT LEAKPROOF;

-- PARALLEL changes
ALTER FUNCTION schema.function_name(arg_types) PARALLEL SAFE;
ALTER FUNCTION schema.function_name(arg_types) PARALLEL UNSAFE;
ALTER FUNCTION schema.function_name(arg_types) PARALLEL RESTRICTED;
```

**Multiple changes:**
- Generate separate ALTER statements (PostgreSQL doesn't support combining)
- Order: PARALLEL first, then LEAKPROOF (alphabetical)

**Edge cases:**
- If signature changes (parameters/return type), use DROP/CREATE (existing behavior)
- Attribute-only changes handled via ALTER (no dependencies broken)

### 5. Test Cases

#### Enhanced Test: `testdata/diff/create_function/add_function/`

**old.sql:** Empty (no functions)

**new.sql:** Functions with various attribute combinations:
1. `process_order()` - Add `LEAKPROOF` and `PARALLEL RESTRICTED`
2. `days_since_special_date()` - Keep `PARALLEL SAFE`, add `LEAKPROOF`
3. `safe_add()` - New simple function with `PARALLEL SAFE` and `LEAKPROOF`

**expected.sql:** CREATE statements matching new.sql with proper attribute ordering

#### New Test: `testdata/diff/create_function/alter_function_attributes/`

Tests attribute-only changes (ALTER FUNCTION path):

**old.sql:**
```sql
CREATE FUNCTION process_data(input text)
RETURNS text
LANGUAGE plpgsql
VOLATILE
AS $$ ... $$;
```

**new.sql:**
```sql
CREATE FUNCTION process_data(input text)
RETURNS text
LANGUAGE plpgsql
VOLATILE
PARALLEL SAFE
LEAKPROOF
AS $$ ... $$;
```

**expected.sql:**
```sql
ALTER FUNCTION process_data(text) PARALLEL SAFE;
ALTER FUNCTION process_data(text) LEAKPROOF;
```

## Implementation Checklist

1. ☐ Update `Function` struct in `ir/ir.go`
2. ☐ Update `inspectFunctions()` in `ir/inspector.go`
3. ☐ Update dump formatting in `internal/dump/dump.go`
4. ☐ Update diff logic in `internal/diff/function.go`
5. ☐ Enhance `add_function` test case
6. ☐ Create `alter_function_attributes` test case
7. ☐ Run diff tests: `PGSCHEMA_TEST_FILTER="create_function/" go test -v ./internal/diff -run TestDiffFromFiles`
8. ☐ Run integration tests: `PGSCHEMA_TEST_FILTER="create_function/" go test -v ./cmd -run TestPlanAndApply`
9. ☐ Validate with live PostgreSQL (compare pg_dump vs pgschema output)

## Success Criteria

- All existing function tests continue to pass
- New test cases pass for both diff and integration
- pg_dump output matches pgschema output for LEAKPROOF/PARALLEL attributes
- ALTER FUNCTION migrations execute successfully on live PostgreSQL
- No regression in function signature detection or DROP/CREATE logic

## References

- PostgreSQL Documentation: [Function Volatility and Parallel Safety](https://www.postgresql.org/docs/current/xfunc-volatility.html)
- System Catalog: `pg_catalog.pg_proc` columns `proleakproof` and `proparallel`
- SQL Syntax: `ALTER FUNCTION` for attribute changes
