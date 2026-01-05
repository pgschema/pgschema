# COMMENT ON FUNCTION/PROCEDURE Support

## Overview

Add support for `COMMENT ON FUNCTION` and `COMMENT ON PROCEDURE` statements in pgschema diff generation.

## Current State

- `ir.Function` and `ir.Procedure` already have `Comment` fields
- Comments are inspected from the database via `ir/inspector.go`
- `functionsEqual()` and `proceduresEqual()` don't compare comments
- No code generates `COMMENT ON FUNCTION/PROCEDURE` statements

## Implementation

### Approach

Comment-only changes generate just the `COMMENT ON` statement without DROP/CREATE or CREATE OR REPLACE. This follows the existing pattern used for indexes (`generateIndexComment` in `internal/diff/index.go`).

### PostgreSQL Syntax

```sql
COMMENT ON FUNCTION schema.func(arg_types) IS 'description';
COMMENT ON FUNCTION schema.func(arg_types) IS NULL;  -- remove comment
COMMENT ON PROCEDURE schema.proc(arg_types) IS 'description';
COMMENT ON PROCEDURE schema.proc(arg_types) IS NULL;  -- remove comment
```

### Files to Modify

1. **`internal/diff/function.go`**
   - Add `generateFunctionComment()` helper
   - Call when creating new function with comment
   - Call when comment changes (with or without body changes)

2. **`internal/diff/procedure.go`**
   - Add `generateProcedureComment()` helper
   - Same pattern as functions

### Logic Flow

```
If function is new:
  → CREATE OR REPLACE FUNCTION ...
  → If has comment: COMMENT ON FUNCTION ... IS '...'

If function exists and changed:
  → If body/params changed: CREATE OR REPLACE FUNCTION ...
  → If comment changed: COMMENT ON FUNCTION ... IS '...' (or IS NULL)

If only comment changed (body identical):
  → Only generate COMMENT ON FUNCTION ...
```

### Test Cases

**`testdata/diff/comment/add_function_comment/`**

- old.sql: Function without comment
- new.sql: Same function with COMMENT ON FUNCTION statement
- diff.sql: Only the COMMENT ON FUNCTION statement

**`testdata/diff/comment/add_procedure_comment/`**

- old.sql: Procedure without comment
- new.sql: Same procedure with COMMENT ON PROCEDURE statement
- diff.sql: Only the COMMENT ON PROCEDURE statement
