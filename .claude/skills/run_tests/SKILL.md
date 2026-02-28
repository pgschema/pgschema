---
name: Run Tests
description: Run pgschema automated tests (go test) to validate diff logic, plan generation, and dump functionality using test fixtures
---

# Run Tests

Use this skill to run pgschema tests for validating implementation changes. Tests cover diff logic, plan generation, dump functionality, and end-to-end migration workflows.

## When to Use This Skill

Invoke this skill when:
- After implementing new schema object support
- After fixing bugs in diff or plan generation
- After modifying dump logic
- Before committing changes
- When debugging test failures
- When regenerating expected test outputs
- When adding new test cases
- When validating changes across PostgreSQL versions

## Test Categories

### 1. Diff Tests (Fast - No Database Required)

**Purpose**: Test schema comparison logic without needing a live database

**Command**:
```bash
go test -v ./internal/diff -run TestDiffFromFiles
```

**What it tests**:
- Compares `old.sql` vs `new.sql` from `testdata/diff/`
- Generates migration DDL
- Validates against `diff.sql`
- Pure logic testing - no database required

**Speed**: Very fast (~1-2 seconds)

### 2. Plan/Apply Integration Tests

**Purpose**: Test full workflow with embedded PostgreSQL

**Command**:
```bash
go test -v ./cmd -run TestPlanAndApply
```

**What it tests**:
- Creates test database with embedded-postgres
- Applies `old.sql` schema
- Generates plan by comparing `new.sql` with database
- Applies the plan
- Verifies final state matches expected schema

**Speed**: Slower (~30-60 seconds for all tests)

### 3. Dump Tests

**Purpose**: Test schema extraction from live databases

**Command**:
```bash
go test -v ./cmd/dump -run TestDumpCommand
```

**What it tests**:
- Dumps schema from test databases (employee, sakila, etc.)
- Validates output format
- Tests database introspection logic

**Speed**: Medium (~10-20 seconds)

## Common Test Workflows

### Workflow 1: Test Specific Feature (Scoped Testing)

Use `PGSCHEMA_TEST_FILTER` to run specific test cases:

**Pattern**: `PGSCHEMA_TEST_FILTER="path/to/test" go test ...`

**Examples**:

```bash
# Test specific diff case
PGSCHEMA_TEST_FILTER="create_view/add_view_array_operators" go test -v ./internal/diff -run TestDiffFromFiles

# Test all view-related diffs
PGSCHEMA_TEST_FILTER="create_view/" go test -v ./internal/diff -run TestDiffFromFiles

# Test all trigger-related integration tests
PGSCHEMA_TEST_FILTER="create_trigger/" go test -v ./cmd -run TestPlanAndApply

# Test specific trigger case
PGSCHEMA_TEST_FILTER="create_trigger/add_trigger_when_distinct" go test -v ./cmd -run TestPlanAndApply
```

**Test filter paths** (from `testdata/diff/`):
- `comment/` - Comment operations (11 test cases)
- `create_domain/` - Domain types (5 test cases)
- `create_function/` - Functions (8 test cases)
- `create_index/` - Indexes (2 test cases)
- `create_materialized_view/` - Materialized views (3 test cases)
- `create_policy/` - RLS policies (10 test cases)
- `create_procedure/` - Procedures (3 test cases)
- `create_sequence/` - Sequences (3 test cases)
- `create_table/` - Tables (37 test cases)
- `create_trigger/` - Triggers (7 test cases)
- `create_type/` - Custom types (3 test cases)
- `create_view/` - Views (4 test cases)
- `default_privilege/` - Default privileges (9 test cases)
- `privilege/` - Privileges/permissions (13 test cases)
- `dependency/` - Dependencies (13 test cases)
- `online/` - Online migrations (14 test cases)
- `migrate/` - Complex migrations (5 test cases)

### Workflow 2: Regenerate Expected Output

When implementation changes intentionally modify generated DDL:

**Command**:
```bash
PGSCHEMA_TEST_FILTER="path/to/test" go test -v ./cmd -run TestPlanAndApply --generate
```

**Example**:
```bash
# After fixing trigger DDL generation
PGSCHEMA_TEST_FILTER="create_trigger/add_trigger" go test -v ./cmd -run TestPlanAndApply --generate
```

**What `--generate` does**:
- Runs the test normally
- Overwrites `diff.sql`, `plan.json`, `plan.sql`, and `plan.txt` with actual output
- Use when you've intentionally changed how DDL is generated
- **Warning**: Only use when you're sure the new output is correct!

**Typical use cases**:
- Improved DDL formatting
- Added support for new syntax
- Fixed incorrect DDL generation
- Changed normalization logic

**Verification steps after `--generate`**:
1. Review the diff in git: `git diff testdata/diff/path/to/test/`
2. Ensure changes are intentional and correct
3. Run test again without `--generate` to verify it passes
4. Commit the updated `diff.sql` and plan files

### Workflow 3: Test Across PostgreSQL Versions

Test against different PostgreSQL versions (14-18):

**Command**:
```bash
PGSCHEMA_POSTGRES_VERSION=<version> go test -v ./cmd -run <test>
```

**Examples**:
```bash
# Test dump on PostgreSQL 14
PGSCHEMA_POSTGRES_VERSION=14 go test -v ./cmd/dump -run TestDumpCommand_Employee

# Test dump on PostgreSQL 17
PGSCHEMA_POSTGRES_VERSION=17 go test -v ./cmd/dump -run TestDumpCommand_Employee

# Test plan/apply on PostgreSQL 15
PGSCHEMA_POSTGRES_VERSION=15 PGSCHEMA_TEST_FILTER="create_trigger/" go test -v ./cmd -run TestPlanAndApply
```

**Supported versions**: 14, 15, 16, 17, 18

### Workflow 4: Run All Tests

**Before committing changes**:

```bash
# Quick check - diff tests only (fast)
go test -v ./internal/diff -run TestDiffFromFiles

# Full validation - all integration tests (slow)
go test -v ./...

# Skip integration tests - unit tests only (fast)
go test -short -v ./...
```

### Workflow 5: Continuous Testing During Development

**Watch mode** (requires external tool like `entr`):

```bash
# Install entr (macOS)
brew install entr

# Watch Go files and re-run tests on change
find . -name "*.go" | entr -c go test -v ./internal/diff -run TestDiffFromFiles

# Watch specific test case
find . -name "*.go" | entr -c sh -c 'PGSCHEMA_TEST_FILTER="create_trigger/add_trigger_when_distinct" go test -v ./internal/diff -run TestDiffFromFiles'
```

### Workflow 6: Debug Failing Test

**Steps**:

1. **Run failing test with verbose output**:
```bash
PGSCHEMA_TEST_FILTER="create_trigger/add_trigger" go test -v ./cmd -run TestPlanAndApply
```

2. **Check test data files**:
```bash
# View old schema
cat testdata/diff/create_trigger/add_trigger/old.sql

# View new schema
cat testdata/diff/create_trigger/add_trigger/new.sql

# View expected migration
cat testdata/diff/create_trigger/add_trigger/diff.sql
```

3. **Run with debugger** (optional):
```bash
# Using delve
dlv test ./internal/diff -- -test.run TestDiffFromFiles
```

4. **Isolate the issue**:
```bash
# Test just the diff logic (faster iteration)
PGSCHEMA_TEST_FILTER="create_trigger/add_trigger" go test -v ./internal/diff -run TestDiffFromFiles

# Test full integration if diff test passes
PGSCHEMA_TEST_FILTER="create_trigger/add_trigger" go test -v ./cmd -run TestPlanAndApply
```

5. **Compare actual vs expected**:
```bash
# The test will show the diff in output, or manually:
# Run test to generate actual output, then compare
# (Actual output is shown in test failure message)
```

## Test Structure

### Diff Test Structure

Located in `testdata/diff/<category>/<test_name>/`:

```
testdata/diff/create_trigger/add_trigger/
├── old.sql       # Starting schema state
├── new.sql       # Desired schema state
├── diff.sql      # Expected migration DDL
├── plan.json     # Expected plan in JSON format
├── plan.sql      # Expected plan as SQL statements
└── plan.txt      # Expected plan as human-readable text
```

**Test process**:
1. Apply `old.sql` to embedded PostgreSQL and inspect into IR
2. Apply `new.sql` to embedded PostgreSQL and inspect into IR
3. Diff the two IRs
4. Generate migration DDL
5. Compare with `diff.sql`

### Integration Test Structure

Same test data, different process:

1. Create test database with embedded-postgres
2. Apply `old.sql` to database and inspect into "current state" IR
3. Apply `new.sql` to separate embedded-postgres and inspect into "desired state" IR
4. Diff current state IR vs desired state IR
5. Generate plan (migration DDL)
6. Apply plan to database
7. Verify final state matches desired state

## Adding New Test Cases

### Step 1: Create Test Directory

```bash
mkdir -p testdata/diff/create_trigger/add_trigger_new_feature
```

### Step 2: Create old.sql

```bash
cat > testdata/diff/create_trigger/add_trigger_new_feature/old.sql << 'EOF'
CREATE TABLE test_table (
    id INTEGER PRIMARY KEY,
    data TEXT
);

CREATE FUNCTION trigger_func() RETURNS TRIGGER AS $$
BEGIN
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
EOF
```

### Step 3: Create new.sql

```bash
cat > testdata/diff/create_trigger/add_trigger_new_feature/new.sql << 'EOF'
CREATE TABLE test_table (
    id INTEGER PRIMARY KEY,
    data TEXT
);

CREATE FUNCTION trigger_func() RETURNS TRIGGER AS $$
BEGIN
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER my_trigger
    BEFORE INSERT ON test_table
    FOR EACH ROW
    WHEN (NEW.data IS NOT NULL)
    EXECUTE FUNCTION trigger_func();
EOF
```

### Step 4: Generate diff.sql and plan files

**Option A: Use --generate flag** (recommended):
```bash
PGSCHEMA_TEST_FILTER="create_trigger/add_trigger_new_feature" go test -v ./cmd -run TestPlanAndApply --generate
```
This generates `diff.sql`, `plan.json`, `plan.sql`, and `plan.txt`.

**Option B: Manually create diff.sql**:
```bash
cat > testdata/diff/create_trigger/add_trigger_new_feature/diff.sql << 'EOF'
CREATE TRIGGER my_trigger
    BEFORE INSERT ON test_table
    FOR EACH ROW
    WHEN ((NEW.data IS NOT NULL))
    EXECUTE FUNCTION trigger_func();
EOF
```

### Step 5: Run Test

```bash
# Test diff logic
PGSCHEMA_TEST_FILTER="create_trigger/add_trigger_new_feature" go test -v ./internal/diff -run TestDiffFromFiles

# Test full integration
PGSCHEMA_TEST_FILTER="create_trigger/add_trigger_new_feature" go test -v ./cmd -run TestPlanAndApply
```

### Step 6: Verify and Commit

```bash
git add testdata/diff/create_trigger/add_trigger_new_feature/
git commit -m "test: add test case for trigger with new feature"
```

## Common Test Commands Reference

### Diff Tests

```bash
# All diff tests
go test -v ./internal/diff -run TestDiffFromFiles

# Specific category
PGSCHEMA_TEST_FILTER="create_table/" go test -v ./internal/diff -run TestDiffFromFiles

# Specific test
PGSCHEMA_TEST_FILTER="create_table/add_column_generated" go test -v ./internal/diff -run TestDiffFromFiles
```

### Integration Tests

```bash
# All integration tests
go test -v ./cmd -run TestPlanAndApply

# Specific category
PGSCHEMA_TEST_FILTER="create_trigger/" go test -v ./cmd -run TestPlanAndApply

# Specific test with timeout (for slow tests)
PGSCHEMA_TEST_FILTER="create_trigger/add_trigger" go test -v ./cmd -run TestPlanAndApply -timeout 2m

# With regeneration
PGSCHEMA_TEST_FILTER="create_trigger/add_trigger" go test -v ./cmd -run TestPlanAndApply --generate
```

### Dump Tests

```bash
# All dump tests
go test -v ./cmd/dump -run TestDumpCommand

# Specific database
go test -v ./cmd/dump -run TestDumpCommand_Employee

# With specific PostgreSQL version
PGSCHEMA_POSTGRES_VERSION=17 go test -v ./cmd/dump -run TestDumpCommand_Employee
```

### All Tests

```bash
# Everything (slow)
go test -v ./...

# Unit tests only (fast - no embedded-postgres)
go test -short -v ./...

# Specific package
go test -v ./internal/diff/...
go test -v ./cmd/...
go test -v ./ir/...
```

## Test Timeouts

Some integration tests may take longer, especially with embedded-postgres:

```bash
# Default timeout: 2 minutes
go test -v ./cmd -run TestPlanAndApply

# Extended timeout: 5 minutes
go test -v ./cmd -run TestPlanAndApply -timeout 5m

# Specific slow test
PGSCHEMA_TEST_FILTER="create_trigger/" go test -v ./cmd -run TestPlanAndApply -timeout 5m
```

## Understanding Test Failures

### Diff Test Failure

```
--- FAIL: TestDiffFromFiles/create_trigger/add_trigger (0.00s)
    Expected:
    CREATE TRIGGER my_trigger BEFORE INSERT ON test_table

    Actual:
    CREATE TRIGGER my_trigger AFTER INSERT ON test_table
```

**What this means**: The generated migration DDL doesn't match diff.sql

**How to fix**:
1. Check if the actual output is correct
2. If correct: Update diff.sql (or use `--generate`)
3. If incorrect: Fix the diff logic in `internal/diff/trigger.go`

### Integration Test Failure

```
--- FAIL: TestPlanAndApply/create_trigger/add_trigger (2.34s)
    Error: trigger 'my_trigger' not found in final schema
```

**What this means**: The migration was applied but final state doesn't match expected

**How to fix**:
1. Check if the plan SQL is correct
2. Verify the SQL is valid PostgreSQL
3. Check if the apply logic executed properly
4. Inspect database state manually using test_db skill

### Timeout Failure

```
panic: test timed out after 2m0s
```

**What this means**: Test took too long (usually embedded-postgres startup)

**How to fix**:
```bash
# Increase timeout
PGSCHEMA_TEST_FILTER="slow_test" go test -v ./cmd -run TestPlanAndApply -timeout 5m
```

## Test Environment Variables

| Variable | Purpose | Example |
|----------|---------|---------|
| `PGSCHEMA_TEST_FILTER` | Run specific test cases | `"create_trigger/"` |
| `PGSCHEMA_POSTGRES_VERSION` | Test specific PG version | `14`, `15`, `16`, `17`, `18` |
| `PGHOST`, `PGPORT`, `PGUSER`, etc. | Database connection (if not using embedded) | See `.env` |

## Best Practices

### Before Committing

1. **Run relevant tests**:
```bash
# If you modified trigger logic
PGSCHEMA_TEST_FILTER="create_trigger/" go test -v ./internal/diff -run TestDiffFromFiles
PGSCHEMA_TEST_FILTER="create_trigger/" go test -v ./cmd -run TestPlanAndApply
```

2. **Run full test suite**:
```bash
go test -v ./...
```

3. **Check for unintended changes**:
```bash
git status
# Ensure only intended test files changed
```

### When Adding Features

1. **Start with diff test** (faster iteration):
```bash
# Create test case
mkdir -p testdata/diff/create_feature/test_name

# Add old.sql, new.sql, then use --generate for diff.sql

# Test
PGSCHEMA_TEST_FILTER="create_feature/test_name" go test -v ./internal/diff -run TestDiffFromFiles
```

2. **Then integration test**:
```bash
PGSCHEMA_TEST_FILTER="create_feature/test_name" go test -v ./cmd -run TestPlanAndApply
```

3. **Test across versions**:
```bash
PGSCHEMA_POSTGRES_VERSION=14 PGSCHEMA_TEST_FILTER="create_feature/" go test -v ./cmd -run TestPlanAndApply
PGSCHEMA_POSTGRES_VERSION=17 PGSCHEMA_TEST_FILTER="create_feature/" go test -v ./cmd -run TestPlanAndApply
```

### When Fixing Bugs

1. **Create failing test first**:
```bash
# Add test case that reproduces bug
mkdir -p testdata/diff/category/bug_reproduction
# Add old.sql, new.sql, then use --generate for diff.sql

# Verify it fails
PGSCHEMA_TEST_FILTER="category/bug_reproduction" go test -v ./internal/diff -run TestDiffFromFiles
```

2. **Fix the bug**:
```bash
# Modify code in internal/diff/ or ir/
```

3. **Verify test passes**:
```bash
PGSCHEMA_TEST_FILTER="category/bug_reproduction" go test -v ./internal/diff -run TestDiffFromFiles
PGSCHEMA_TEST_FILTER="category/bug_reproduction" go test -v ./cmd -run TestPlanAndApply
```

4. **Run related tests**:
```bash
PGSCHEMA_TEST_FILTER="category/" go test -v ./...
```

## Quick Reference

**Most common commands**:

```bash
# Fast diff test for specific feature
PGSCHEMA_TEST_FILTER="create_trigger/add_trigger" go test -v ./internal/diff -run TestDiffFromFiles

# Full integration test for specific feature
PGSCHEMA_TEST_FILTER="create_trigger/add_trigger" go test -v ./cmd -run TestPlanAndApply

# Regenerate expected output (after intentional change)
PGSCHEMA_TEST_FILTER="create_trigger/add_trigger" go test -v ./cmd -run TestPlanAndApply --generate

# Test all triggers
PGSCHEMA_TEST_FILTER="create_trigger/" go test -v ./cmd -run TestPlanAndApply

# Test everything (before commit)
go test -v ./...

# Dump tests
go test -v ./cmd/dump -run TestDumpCommand
```

## Verification Checklist

Before committing changes:

- [ ] Ran diff tests for affected areas
- [ ] Ran integration tests for affected areas
- [ ] Tests pass on at least one PostgreSQL version
- [ ] If intentionally changed DDL, updated diff.sql and plan files
- [ ] New features have test coverage
- [ ] Bug fixes have regression tests
- [ ] No unintended test file modifications
- [ ] All tests pass: `go test -v ./...`
