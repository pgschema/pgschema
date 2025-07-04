# IR (Intermediate Representation) Tests

This directory contains tests for the Intermediate Representation (IR) package, which handles parsing and building PostgreSQL schema representations.

## Running All Tests

To run all tests in the ir package:

```bash
go test -v ./internal/ir
```

## Running Integration Tests Only

To run only the integration tests (ir_integration_test.go):

```bash
go test -v ./internal/ir -run "TestIRIntegration_"
```

## Running Specific Integration Tests

To run a particular test under ir_integration_test.go:

### Employee Database Integration Test

```bash
go test -v ./internal/ir -run "TestIRIntegration_Employee"
```

### Bytebase Database Integration Test

```bash
go test -v ./internal/ir -run "TestIRIntegration_Bytebase"
```

### Sakila Database Integration Test

```bash
go test -v ./internal/ir -run "TestIRIntegration_Sakila"
```

## Running Parser Tests

```bash
go test -v ./internal/ir -run "TestParseSQL_"
```
