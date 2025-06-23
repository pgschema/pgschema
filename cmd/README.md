# CMD Tests

This directory contains tests for the pgschema command-line interface.

## Running All Tests

To run all tests in the cmd package:

```bash
go test -v ./cmd
```

## Running Integration Tests Only

To run only the integration tests (inspect_integration_test.go):

```bash
go test -v ./cmd -run "TestInspectCommand_"
```

## Running Specific Integration Tests

To run a particular test under inspect_integration_test.go:

### Employee Database Test
```bash
go test -v ./cmd -run "TestInspectCommand_Employee"
```

### Sakila Database Test
```bash
go test -v ./cmd -run "TestInspectCommand_Sakila"
```

### Bytebase Database Test
```bash
go test -v ./cmd -run "TestInspectCommand_Bytebase"
```