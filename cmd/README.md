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
go test -v ./cmd -run "TestDumpCommand_"
```

## Running Specific Integration Tests

To run a particular test under inspect_integration_test.go:

### Employee Database Test
```bash
go test -v ./cmd -run "TestDumpCommand_Employee"
```

### Sakila Database Test
```bash
go test -v ./cmd -run "TestDumpCommand_Sakila"
```

### Bytebase Database Test
```bash
go test -v ./cmd -run "TestDumpCommand_Bytebase"
```

## Running Plan Command Tests

To run all plan command tests:

```bash
go test -v ./cmd -run "TestPlan"
```

### Specific Plan Tests

To run specific plan command tests:

```bash
# Test basic plan command functionality
go test -v ./cmd -run "TestPlanCommand$"

# Test plan command execution with different formats  
go test -v ./cmd -run "TestPlanCommandExecution"

# Test plan command input validation
go test -v ./cmd -run "TestPlanValidation"

# Test plan command error handling
go test -v ./cmd -run "TestPlanCommandErrors"
```

### Plan Integration Tests

To run plan command integration tests (requires Docker):

```bash
# Run all plan integration tests
go test -v ./cmd -run "TestPlanCommand_" -timeout 10m

# Test file-to-file comparison
go test -v ./cmd -run "TestPlanCommand_FileToFile"

# Test file-to-database comparison  
go test -v ./cmd -run "TestPlanCommand_FileToDatabase"

# Test database-to-database comparison
go test -v ./cmd -run "TestPlanCommand_DatabaseToDatabase"

# Test schema filtering functionality
go test -v ./cmd -run "TestPlanCommand_SchemaFiltering"
```