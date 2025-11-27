# Function LEAKPROOF and PARALLEL Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add full support for PostgreSQL function LEAKPROOF and PARALLEL attributes (SAFE/UNSAFE/RESTRICTED) throughout the dump/plan/apply workflow.

**Architecture:** Extend Function IR with two new fields, update database inspector to query pg_catalog.pg_proc, modify dump formatter to output non-default attributes, and enhance diff logic to generate ALTER FUNCTION migrations for attribute changes.

**Tech Stack:** Go 1.24+, pgx/v5 for database queries, embedded-postgres for testing, PostgreSQL 14-17

---

## Task 1: Update IR Structure

**Files:**
- Modify: `ir/ir.go:124-136` (Function struct)

**Step 1: Add new fields to Function struct**

Add `IsLeakproof` and `Parallel` fields after `IsSecurityDefiner`:

```go
// Function represents a database function
type Function struct {
	Schema            string       `json:"schema"`
	Name              string       `json:"name"`
	Definition        string       `json:"definition"`
	ReturnType        string       `json:"return_type"`
	Language          string       `json:"language"`
	Parameters        []*Parameter `json:"parameters,omitempty"`
	Comment           string       `json:"comment,omitempty"`
	Volatility        string       `json:"volatility,omitempty"`          // IMMUTABLE, STABLE, VOLATILE
	IsStrict          bool         `json:"is_strict,omitempty"`           // STRICT or null behavior
	IsSecurityDefiner bool         `json:"is_security_definer,omitempty"` // SECURITY DEFINER
	IsLeakproof       bool         `json:"is_leakproof,omitempty"`        // LEAKPROOF
	Parallel          string       `json:"parallel,omitempty"`            // SAFE, UNSAFE, RESTRICTED
}
```

**Step 2: Verify code compiles**

Run: `go build -o pgschema .`
Expected: Successful compilation (new fields don't break anything)

**Step 3: Commit IR changes**

```bash
git add ir/ir.go
git commit -m "feat: add IsLeakproof and Parallel fields to Function IR"
```

---

## Task 2: Update Database Inspector

**Files:**
- Modify: `ir/inspector.go` (inspectFunctions method)

**Step 1: Locate the inspectFunctions query**

Find the SELECT query in `inspectFunctions()` that queries `pg_catalog.pg_proc`. It should be around line 400-500.

**Step 2: Add proleakproof and proparallel to SELECT**

Add these columns to the existing SELECT statement:

```go
p.proleakproof,
p.proparallel
```

The query should look like:
```go
query := `
SELECT
	n.nspname AS schema_name,
	p.proname AS function_name,
	pg_get_functiondef(p.oid) AS definition,
	pg_get_function_result(p.oid) AS return_type,
	l.lanname AS language,
	p.provolatile,
	p.proisstrict,
	p.prosecdef,
	p.proleakproof,
	p.proparallel,
	obj_description(p.oid, 'pg_proc') AS comment
FROM pg_catalog.pg_proc p
...
`
```

**Step 3: Add variables to scan into**

In the scan section, add variables:

```go
var (
	// ... existing variables ...
	proleakproof bool
	proparallel  string
)
```

**Step 4: Add to Scan() call**

Add to the existing `rows.Scan()`:

```go
&proleakproof,
&proparallel,
```

**Step 5: Map proparallel to Parallel field**

After the existing volatility mapping, add:

```go
// Map LEAKPROOF
fn.IsLeakproof = proleakproof

// Map PARALLEL
switch proparallel {
case "s":
	fn.Parallel = "SAFE"
case "r":
	fn.Parallel = "RESTRICTED"
case "u":
	fn.Parallel = "UNSAFE"
default:
	fn.Parallel = "UNSAFE" // Defensive default
}
```

**Step 6: Verify code compiles**

Run: `go build -o pgschema .`
Expected: Successful compilation

**Step 7: Commit inspector changes**

```bash
git add ir/inspector.go
git commit -m "feat: extract LEAKPROOF and PARALLEL from pg_catalog.pg_proc"
```

---

## Task 3: Update Dump Formatter

**Files:**
- Modify: `internal/dump/dump.go` (function formatting)

**Step 1: Locate function dump logic**

Find the function that formats function definitions for dump output. Look for code that builds the CREATE FUNCTION statement. Should be in a method like `dumpFunction()` or similar.

**Step 2: Add PARALLEL output logic**

After the LANGUAGE and volatility output, add:

```go
// Add PARALLEL if not default (UNSAFE)
if fn.Parallel == "SAFE" {
	fmt.Fprintf(buf, "PARALLEL SAFE\n")
} else if fn.Parallel == "RESTRICTED" {
	fmt.Fprintf(buf, "PARALLEL RESTRICTED\n")
}
// Note: Don't output PARALLEL UNSAFE (it's the default)
```

**Step 3: Add LEAKPROOF output logic**

After PARALLEL, add:

```go
// Add LEAKPROOF if true
if fn.IsLeakproof {
	fmt.Fprintf(buf, "LEAKPROOF\n")
}
// Note: Don't output NOT LEAKPROOF (it's the default)
```

The attribute order should be:
1. LANGUAGE
2. Volatility (IMMUTABLE/STABLE/VOLATILE)
3. PARALLEL (if not UNSAFE)
4. LEAKPROOF (if true)
5. STRICT (if true)
6. SECURITY DEFINER (if true)

**Step 4: Test dump output manually**

Run:
```bash
go build -o pgschema .
PGPASSWORD='testpwd1' ./pgschema dump -h localhost -p 5432 -U postgres -d postgres --schema public
```

Expected: Should compile and run (may not show new attributes yet if test DB doesn't have them)

**Step 5: Commit dump formatter changes**

```bash
git add internal/dump/dump.go
git commit -m "feat: output LEAKPROOF and PARALLEL in function dumps"
```

---

## Task 4: Create Test Case - add_function Enhancement

**Files:**
- Modify: `testdata/diff/create_function/add_function/new.sql`
- Modify: `testdata/diff/create_function/add_function/plan.sql`

**Step 1: Update new.sql with LEAKPROOF and PARALLEL**

Modify the existing functions to add attributes:

```sql
CREATE FUNCTION process_order(
    order_id integer,
    -- Simple numeric defaults
    discount_percent numeric DEFAULT 0,
    priority_level integer DEFAULT 1,
    -- String defaults
    note varchar DEFAULT '',
    status text DEFAULT 'pending',
    -- Boolean defaults
    apply_tax boolean DEFAULT true,
    is_priority boolean DEFAULT false
)
RETURNS numeric
LANGUAGE plpgsql
VOLATILE
PARALLEL RESTRICTED
LEAKPROOF
SECURITY DEFINER
STRICT
AS $$
DECLARE
    total numeric;
BEGIN
    SELECT amount INTO total FROM orders WHERE id = order_id;
    RETURN total - (total * discount_percent / 100);
END;
$$;

-- Table function with RETURN clause (bug report test case)
CREATE FUNCTION days_since_special_date() RETURNS SETOF timestamptz
    LANGUAGE sql
    STABLE
    PARALLEL SAFE
    LEAKPROOF
    RETURN generate_series(date_trunc('day', '2025-01-01'::timestamp), date_trunc('day', NOW()), '1 day'::interval);

-- Simple pure function demonstrating PARALLEL SAFE + LEAKPROOF
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

**Step 2: Update plan.sql to match expected output**

Update `plan.sql` to show the normalized CREATE FUNCTION statements that pgschema will generate (with proper attribute ordering):

```sql
CREATE OR REPLACE FUNCTION days_since_special_date()
RETURNS SETOF timestamp with time zone
LANGUAGE sql
STABLE
PARALLEL SAFE
LEAKPROOF
RETURN generate_series((date_trunc('day'::text, '2025-01-01 00:00:00'::timestamp without time zone))::timestamp with time zone, date_trunc('day'::text, now()), '1 day'::interval);

CREATE OR REPLACE FUNCTION process_order(
    order_id integer,
    discount_percent numeric DEFAULT 0,
    priority_level integer DEFAULT 1,
    note varchar DEFAULT '',
    status text DEFAULT 'pending',
    apply_tax boolean DEFAULT true,
    is_priority boolean DEFAULT false
)
RETURNS numeric
LANGUAGE plpgsql
VOLATILE
PARALLEL RESTRICTED
LEAKPROOF
SECURITY DEFINER
STRICT
AS $$
DECLARE
    total numeric;
BEGIN
    SELECT amount INTO total FROM orders WHERE id = order_id;
    RETURN total - (total * discount_percent / 100);
END;
$$;

CREATE OR REPLACE FUNCTION safe_add(a integer, b integer)
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

**Step 3: Run diff test to see current output**

Run: `PGSCHEMA_TEST_FILTER="create_function/add_function" go test -v ./internal/diff -run TestDiffFromFiles`
Expected: Test may fail initially (dump output may not match yet due to formatting details)

**Step 4: Commit test case updates**

```bash
git add testdata/diff/create_function/add_function/new.sql testdata/diff/create_function/add_function/plan.sql
git commit -m "test: add LEAKPROOF and PARALLEL to add_function test case"
```

---

## Task 5: Update Diff Logic for ALTER FUNCTION

**Files:**
- Modify: `internal/diff/function.go`

**Step 1: Locate function comparison logic**

Find the code that compares old vs new functions with the same signature. Look for logic that checks if functions differ.

**Step 2: Add LEAKPROOF comparison**

Add after existing attribute comparisons (like volatility, strict, security definer):

```go
// Check LEAKPROOF changes
if oldFunc.IsLeakproof != newFunc.IsLeakproof {
	var stmt string
	if newFunc.IsLeakproof {
		stmt = fmt.Sprintf("ALTER FUNCTION %s.%s(%s) LEAKPROOF;",
			QuoteIdentifier(newFunc.Schema),
			QuoteIdentifier(newFunc.Name),
			newFunc.GetArguments())
	} else {
		stmt = fmt.Sprintf("ALTER FUNCTION %s.%s(%s) NOT LEAKPROOF;",
			QuoteIdentifier(newFunc.Schema),
			QuoteIdentifier(newFunc.Name),
			newFunc.GetArguments())
	}
	steps = append(steps, MigrationStep{
		Type:        "ALTER_FUNCTION_LEAKPROOF",
		Schema:      newFunc.Schema,
		Name:        newFunc.Name,
		SQL:         stmt,
		Description: fmt.Sprintf("Alter function %s.%s LEAKPROOF", newFunc.Schema, newFunc.Name),
	})
}
```

**Step 3: Add PARALLEL comparison**

Add after LEAKPROOF:

```go
// Check PARALLEL changes
if oldFunc.Parallel != newFunc.Parallel {
	stmt := fmt.Sprintf("ALTER FUNCTION %s.%s(%s) PARALLEL %s;",
		QuoteIdentifier(newFunc.Schema),
		QuoteIdentifier(newFunc.Name),
		newFunc.GetArguments(),
		newFunc.Parallel)
	steps = append(steps, MigrationStep{
		Type:        "ALTER_FUNCTION_PARALLEL",
		Schema:      newFunc.Schema,
		Name:        newFunc.Name,
		SQL:         stmt,
		Description: fmt.Sprintf("Alter function %s.%s PARALLEL %s", newFunc.Schema, newFunc.Name, newFunc.Parallel),
	})
}
```

**Step 4: Verify code compiles**

Run: `go build -o pgschema .`
Expected: Successful compilation

**Step 5: Commit diff logic**

```bash
git add internal/diff/function.go
git commit -m "feat: generate ALTER FUNCTION for LEAKPROOF and PARALLEL changes"
```

---

## Task 6: Create Test Case - alter_function_attributes

**Files:**
- Create: `testdata/diff/create_function/alter_function_attributes/old.sql`
- Create: `testdata/diff/create_function/alter_function_attributes/new.sql`
- Create: `testdata/diff/create_function/alter_function_attributes/plan.sql`

**Step 1: Create test directory**

Run: `mkdir -p testdata/diff/create_function/alter_function_attributes`

**Step 2: Write old.sql (function without attributes)**

Create `testdata/diff/create_function/alter_function_attributes/old.sql`:

```sql
CREATE FUNCTION process_data(input text)
RETURNS text
LANGUAGE plpgsql
VOLATILE
AS $$
BEGIN
    RETURN upper(input);
END;
$$;

CREATE FUNCTION calculate_total(amount numeric, tax_rate numeric)
RETURNS numeric
LANGUAGE sql
STABLE
AS $$
    SELECT amount * (1 + tax_rate);
$$;
```

**Step 3: Write new.sql (add LEAKPROOF and PARALLEL)**

Create `testdata/diff/create_function/alter_function_attributes/new.sql`:

```sql
CREATE FUNCTION process_data(input text)
RETURNS text
LANGUAGE plpgsql
VOLATILE
PARALLEL SAFE
LEAKPROOF
AS $$
BEGIN
    RETURN upper(input);
END;
$$;

CREATE FUNCTION calculate_total(amount numeric, tax_rate numeric)
RETURNS numeric
LANGUAGE sql
STABLE
PARALLEL SAFE
LEAKPROOF
AS $$
    SELECT amount * (1 + tax_rate);
$$;
```

**Step 4: Write plan.sql (expected ALTER statements)**

Create `testdata/diff/create_function/alter_function_attributes/plan.sql`:

```sql
ALTER FUNCTION process_data(text) PARALLEL SAFE;

ALTER FUNCTION process_data(text) LEAKPROOF;

ALTER FUNCTION calculate_total(numeric, numeric) PARALLEL SAFE;

ALTER FUNCTION calculate_total(numeric, numeric) LEAKPROOF;
```

**Step 5: Run diff test**

Run: `PGSCHEMA_TEST_FILTER="create_function/alter_function_attributes" go test -v ./internal/diff -run TestDiffFromFiles`
Expected: Test should pass if diff logic is correct

**Step 6: Commit new test case**

```bash
git add testdata/diff/create_function/alter_function_attributes/
git commit -m "test: add alter_function_attributes test case"
```

---

## Task 7: Run All Function Diff Tests

**Files:**
- N/A (testing only)

**Step 1: Run all function diff tests**

Run: `PGSCHEMA_TEST_FILTER="create_function/" go test -v ./internal/diff -run TestDiffFromFiles`
Expected: All tests pass

**Step 2: If tests fail, check test fixtures**

Common issues:
- Attribute ordering doesn't match expected output
- Missing quote identifiers
- Incorrect normalization of function signatures

Fix by adjusting dump formatter or test fixture files to match actual pg_dump behavior.

**Step 3: Regenerate expected files if needed**

If the logic is correct but expected files need updating, use the test framework's regeneration option (if available) or manually update `plan.sql` files to match actual output.

---

## Task 8: Run Integration Tests

**Files:**
- N/A (testing only)

**Step 1: Run function integration tests**

Run: `PGSCHEMA_TEST_FILTER="create_function/" go test -v ./cmd -run TestPlanAndApply -timeout 5m`
Expected: All tests pass (creates embedded postgres, applies migrations, validates)

**Step 2: Check for transaction handling**

Ensure ALTER FUNCTION statements execute correctly within transactions (they should - it's a DDL but doesn't require --pgschema-no-transaction).

**Step 3: If tests fail, debug**

Use @superpowers:systematic-debugging skill:
- Check embedded postgres logs
- Verify SQL syntax
- Confirm pg_catalog queries work on test postgres version

**Step 4: Verify across postgres versions**

Run tests against all supported versions:

```bash
PGSCHEMA_POSTGRES_VERSION=14 go test -v ./cmd -run TestPlanAndApply/create_function
PGSCHEMA_POSTGRES_VERSION=15 go test -v ./cmd -run TestPlanAndApply/create_function
PGSCHEMA_POSTGRES_VERSION=16 go test -v ./cmd -run TestPlanAndApply/create_function
PGSCHEMA_POSTGRES_VERSION=17 go test -v ./cmd -run TestPlanAndApply/create_function
```

Expected: All pass (LEAKPROOF and PARALLEL supported in all versions 14+)

---

## Task 9: Validate Against Live Database

**Files:**
- N/A (manual validation)

**Step 1: Use validate_db skill**

Run @validate_db skill to connect to live PostgreSQL and compare outputs.

**Step 2: Create test function in live database**

Connect to test database and create a function with attributes:

```sql
CREATE FUNCTION test_leakproof_parallel(x integer)
RETURNS integer
LANGUAGE sql
IMMUTABLE
PARALLEL SAFE
LEAKPROOF
STRICT
AS $$
    SELECT x * 2;
$$;
```

**Step 3: Compare pg_dump vs pgschema dump**

Run both:
```bash
PGPASSWORD='testpwd1' pg_dump -h localhost -p 5432 -U postgres -d postgres --schema-only --schema=public | grep -A 10 "test_leakproof_parallel"

PGPASSWORD='testpwd1' ./pgschema dump -h localhost -p 5432 -U postgres -d postgres --schema public | grep -A 10 "test_leakproof_parallel"
```

Expected: Output should be identical (attribute order and formatting match)

**Step 4: Test ALTER FUNCTION migration**

1. Create function without attributes
2. Run pgschema plan with new.sql that adds attributes
3. Verify plan shows ALTER FUNCTION statements
4. Run pgschema apply
5. Verify function has new attributes in database

```bash
# Setup
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d postgres -c "
CREATE OR REPLACE FUNCTION test_alter(x int) RETURNS int LANGUAGE sql AS 'SELECT x';
"

# Create desired state file
echo "CREATE FUNCTION test_alter(x int) RETURNS int LANGUAGE sql PARALLEL SAFE LEAKPROOF AS 'SELECT x';" > /tmp/test.sql

# Plan (using embedded postgres by default)
./pgschema plan --schema public /tmp/test.sql

# Should show:
# ALTER FUNCTION test_alter(integer) PARALLEL SAFE;
# ALTER FUNCTION test_alter(integer) LEAKPROOF;

# Apply
./pgschema apply --schema public /tmp/test.sql -h localhost -p 5432 -U postgres -d postgres

# Verify
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d postgres -c "
SELECT proname, proparallel, proleakproof
FROM pg_proc
WHERE proname = 'test_alter';
"
```

Expected: proparallel='s', proleakproof=true

---

## Task 10: Run Full Test Suite

**Files:**
- N/A (testing only)

**Step 1: Run all tests**

Run: `go test -v ./...`
Expected: All tests pass (no regressions)

**Step 2: Check test coverage**

Run: `go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out`
Expected: New code paths are covered by tests

**Step 3: Fix any regressions**

If any existing tests fail, use @superpowers:systematic-debugging to identify and fix issues.

---

## Task 11: Final Commit and Cleanup

**Files:**
- All modified files

**Step 1: Review all changes**

Run: `git status && git diff`
Expected: Only intended changes present

**Step 2: Run final verification**

```bash
go build -o pgschema .
go test -v ./...
```

Expected: Clean build, all tests pass

**Step 3: Create final commit if needed**

If there are uncommitted changes:

```bash
git add -A
git commit -m "feat: complete LEAKPROOF and PARALLEL function support

- Add IsLeakproof and Parallel fields to Function IR
- Extract attributes from pg_catalog.pg_proc
- Output attributes in dump (non-defaults only)
- Generate ALTER FUNCTION for attribute changes
- Add comprehensive test cases

Supports all three parallel modes: SAFE, UNSAFE, RESTRICTED
Validated against PostgreSQL 14-17"
```

**Step 4: Verify git log**

Run: `git log --oneline -10`
Expected: Clean commit history with descriptive messages

---

## Success Criteria

- ✅ All function diff tests pass
- ✅ All function integration tests pass
- ✅ Tests pass on PostgreSQL 14, 15, 16, 17
- ✅ pg_dump output matches pgschema output for LEAKPROOF/PARALLEL
- ✅ ALTER FUNCTION migrations execute successfully
- ✅ No regressions in existing function tests
- ✅ Manual validation against live database successful

## Skills Referenced

- @superpowers:systematic-debugging - For debugging test failures
- @validate_db - For live database validation
- @run_tests - For running pgschema test suite
- @pg_dump - For comparing output with pg_dump reference

## Notes

- LEAKPROOF and PARALLEL are supported in PostgreSQL 9.2+ and 9.6+ respectively (well within our 14-17 support range)
- ALTER FUNCTION for these attributes is transactional (no special handling needed)
- The hybrid approach (store explicit, output non-defaults) ensures clean diffs and readable dumps
