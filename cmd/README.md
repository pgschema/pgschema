# CMD Package

## Running Tests

```bash
# Run all tests in cmd package
go test -v ./cmd/...

# Plan and apply tests (CLI integration tests)
go test -v ./cmd/ -run "TestPlanAndApply"

# If we change the plan generation logic, we may need to regenerate the test case plan.[json|sql|txt]
go test -v ./cmd/ -run "TestPlanAndApply" --generate

# Run a specific test case
PGSCHEMA_TEST_FILTER="create_table/add_column_identity" go test -v ./cmd -run TestPlanAndApply

# Include command tests
go test -v ./cmd/ -run "TestIncludeIntegration"

# Root command tests
go test -v ./cmd/ -run "TestRoot"
```
