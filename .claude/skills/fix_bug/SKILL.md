---
name: fix_bug
description: Fix a bug from a GitHub issue using TDD. Analyzes the issue, creates a reproducing test case, implements the fix, verifies it, runs refactor-pass, and creates a PR.
---

# Fix Bug

End-to-end workflow for fixing bugs reported as GitHub issues. Uses TDD: reproduce first, then fix, then verify.

## Prerequisites

- A GitHub issue URL or issue number (from the pgplex/pgschema repo)

## Workflow

### Phase 1: Analyze the Issue

1. **Fetch the issue** using `gh issue view <number>` to get the full description, labels, and any linked PRs.
2. **Understand the bug**: Identify what's broken, what the expected behavior is, and which area of code is affected.
3. **Classify the bug type**:
   - **Dump bug**: `pgschema dump` produces incorrect output (wrong SQL, missing objects, bad formatting). Test goes in `testdata/dump/`.
   - **Diff/Plan bug**: `pgschema dump` is correct but `pgschema plan` generates wrong migration DDL. Test goes in `testdata/diff/`.
   - **Both**: If unclear, start with dump. If dump output is correct, it's a diff bug.

### Phase 2: Create the Test Case (TDD - Red)

#### For Dump Bugs (`testdata/dump/`)

1. **Create test directory**:
   ```
   testdata/dump/issue_<NUMBER>_<short_description>/
   ```
   Use snake_case for the description. Example: `issue_250_enum_type_missing`

2. **Create test files**:
   - `manifest.json` - Metadata about the test case:
     ```json
     {
       "name": "issue_<NUMBER>_<short_description>",
       "description": "Test case for <bug description> (GitHub issue #<NUMBER>)",
       "source": "https://github.com/pgplex/pgschema/issues/<NUMBER>",
       "notes": [
         "Reproduces the bug where <specific behavior>",
         "Tests that <expected correct behavior>"
       ]
     }
     ```
   - `raw.sql` - The original DDL that creates the schema (what a user would write)
   - `pgdump.sql` - What `pg_dump --schema-only` produces for this schema (the input to the test). Generate this by applying `raw.sql` to an embedded-postgres and running pg_dump, or construct it manually based on pg_dump conventions.
   - `pgschema.sql` - The expected correct output from `pgschema dump`

3. **Register the test** in `cmd/dump/dump_integration_test.go`:
   ```go
   func TestDumpCommand_Issue<NUMBER><PascalDescription>(t *testing.T) {
       if testing.Short() {
           t.Skip("Skipping integration test in short mode")
       }
       runExactMatchTest(t, "issue_<NUMBER>_<short_description>")
   }
   ```

4. **Run the test to confirm it fails** (red):
   ```bash
   go test -v ./cmd/dump -run TestDumpCommand_Issue<NUMBER>
   ```

#### For Diff/Plan Bugs (`testdata/diff/`)

1. **Create test directory** under the appropriate category:
   ```
   testdata/diff/<category>/issue_<NUMBER>_<short_description>/
   ```
   Categories: `create_table`, `create_index`, `create_trigger`, `create_view`, `create_function`, `create_procedure`, `create_sequence`, `create_type`, `create_domain`, `create_policy`, `create_materialized_view`, `comment`, `privilege`, `default_privilege`, `dependency`, `online`, `migrate`.

   Example: `testdata/diff/create_index/issue_250_partial_index_diff`

2. **Create test files**:
   - `old.sql` - The starting schema state (current database)
   - `new.sql` - The desired schema state (user's SQL files)
   - Leave `diff.sql` empty or with expected content — it will be generated

3. **Run the diff test to confirm it fails** (red):
   ```bash
   PGSCHEMA_TEST_FILTER="<category>/issue_<NUMBER>_<short_description>" go test -v ./internal/diff -run TestDiffFromFiles
   ```

4. **Generate expected outputs** using `--generate` once you know the correct behavior:
   ```bash
   PGSCHEMA_TEST_FILTER="<category>/issue_<NUMBER>_<short_description>" go test -v ./cmd -run TestPlanAndApply --generate
   ```

### Phase 3: Implement the Fix (Green)

1. **Locate the relevant code**. Common locations:
   - Dump bugs: `ir/inspector.go` (database introspection), `ir/normalize.go` (normalization), `internal/dump/` (output formatting)
   - Diff bugs: `internal/diff/` (comparison logic — `table.go`, `column.go`, `index.go`, `trigger.go`, `view.go`, `function.go`, `procedure.go`, `sequence.go`, `type.go`, `policy.go`, `constraint.go`)
   - IR bugs: `ir/ir.go` (data structures), `ir/quote.go` (identifier quoting)

2. **Use reference skills as needed**:
   - Consult **pg_dump Reference** skill for correct system catalog queries
   - Consult **PostgreSQL Syntax Reference** skill for grammar questions
   - Use **Validate with Database** skill to test queries against live PostgreSQL

3. **Make the minimal fix**. Do not refactor surrounding code — focus on the bug.

4. **Run the test to confirm it passes** (green):
   ```bash
   # For dump bugs
   go test -v ./cmd/dump -run TestDumpCommand_Issue<NUMBER>

   # For diff bugs
   PGSCHEMA_TEST_FILTER="<category>/issue_<NUMBER>" go test -v ./internal/diff -run TestDiffFromFiles
   PGSCHEMA_TEST_FILTER="<category>/issue_<NUMBER>" go test -v ./cmd -run TestPlanAndApply
   ```

### Phase 4: Verify No Regressions

Run the full test suite to ensure nothing else broke:

```bash
go test -v ./...
```

If any tests fail, investigate and fix before proceeding.

### Phase 5: Refactor Pass

Invoke the **refactor-pass** skill to clean up:
- Remove any dead code introduced or exposed by the fix
- Simplify logic if the fix revealed unnecessary complexity
- Ensure the test case is clean and minimal
- Run build/tests to verify behavior after cleanup

### Phase 6: Create PR

1. **Create a feature branch**:
   ```bash
   git checkout -b fix/issue-<NUMBER>-<short-description>
   ```

2. **Commit the changes** with a descriptive message:
   ```
   fix: <concise description of what was fixed> (#<NUMBER>)
   ```

3. **Push and create PR**:
   ```bash
   git push -u origin fix/issue-<NUMBER>-<short-description>
   gh pr create --title "fix: <description> (#<NUMBER>)" --body "..."
   ```

   PR body should include:
   - `## Summary` — What was broken and how it was fixed
   - `Fixes #<NUMBER>` — Link to the issue for auto-close
   - `## Test plan` — What test case was added and how to run it

## Checklist

- [ ] Issue analyzed and bug type classified (dump vs diff)
- [ ] Test case created with proper naming (`issue_<NUMBER>_<description>`)
- [ ] Test fails before fix (red)
- [ ] Minimal fix implemented
- [ ] Test passes after fix (green)
- [ ] Full test suite passes (no regressions)
- [ ] Refactor pass completed
- [ ] PR created and linked to issue
