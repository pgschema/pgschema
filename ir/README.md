# IR (Intermediate Representation) Tests

## Running Tests

```bash
# Run all IR tests
go test -v ./ir

# Run integration tests only
go test -v ./ir -run "TestIRIntegration_"

# Run specific integration tests
go test -v ./ir -run "TestIRIntegration_Employee"
go test -v ./ir -run "TestIRIntegration_Bytebase"
go test -v ./ir -run "TestIRIntegration_Sakila"

# Run parser tests
go test -v ./ir -run "TestParser_"
```
