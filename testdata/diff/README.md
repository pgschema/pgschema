# Diff Test Data

## Running Tests

```bash
# Run all diff tests
go test ./internal/diff -v

# Run only file-based tests
go test ./internal/diff -v -run TestDiffFromFiles

# Run CLI integration tests (plan and apply workflow)
go test ./cmd -v -run TestPlanAndApply

# Generate the plan file
go test ./cmd -v -run TestPlanAndApply --generate

# Run specific test cases with filter
PGSCHEMA_TEST_FILTER="create_table/" go test -v ./internal/diff -run TestDiffFromFiles
PGSCHEMA_TEST_FILTER="create_table/" go test -v ./cmd -run TestPlanAndApply
```
