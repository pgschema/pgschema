# CMD Package

## Running Tests

```bash
# Run all tests in cmd package
go test -v ./cmd/...

# Plan and apply tests
go test -v ./cmd/ -run "TestPlanAndApply"

# Include command tests
go test -v ./cmd/ -run "TestIncludeIntegration"

# Root command tests
go test -v ./cmd/ -run "TestRoot"
```
