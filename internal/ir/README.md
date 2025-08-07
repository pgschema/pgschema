# IR (Intermediate Representation) Tests

## Running Tests

```bash
# Run all IR tests
go test -v ./internal/ir

# Run integration tests only
go test -v ./internal/ir -run "TestIRIntegration_"

# Run specific integration tests
go test -v ./internal/ir -run "TestIRIntegration_Employee"
go test -v ./internal/ir -run "TestIRIntegration_Bytebase"
go test -v ./internal/ir -run "TestIRIntegration_Sakila"

# Run parser tests
go test -v ./internal/ir -run "TestParser_"
```
