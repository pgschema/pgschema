# Diff Test Data

## Running Tests

```bash
# Run all diff tests
go test ./internal/diff -v

# Run only file-based tests
go test ./internal/diff -v -run TestDiffFromFiles

# Run only inspector and parser integration tests
go test ./internal/diff -v -run TestDiffInspectorAndParser

# Run specific test cases with filter
PGSCHEMA_TEST_FILTER="create_table/" go test -v ./internal/diff -run TestDiffFromFiles
```
