# CMD Package

This directory contains the command-line interface implementation for pgschema.

## Running All Tests

To run all tests in the cmd package and its subpackages:

```bash
go test -v ./cmd/...
```

## Running Tests by Command

### Plan and apply Tests

```bash
go test -v ./cmd/ -run "TestPlanAndApply"
```

### Include Command Tests

```bash
go test -v ./cmd/ -run "TestIncludeIntegration"
```

### Migrate Command Tests

```bash
go test -v ./cmd/ -run "TestIncludeIntegration"

# Regenerate the expected plan files
go test -v ./cmd/ -run "TestIncludeIntegration" --regenerate
```

### Dump Command Tests

```bash
# All dump tests
go test -v ./cmd/dump/

# Specific dump tests
go test -v ./cmd/dump/ -run "TestDumpCommand_Employee"
```

### Plan Command Tests

```bash
# All plan tests
go test -v ./cmd/plan/

# Specific plan tests
go test -v ./cmd/plan/ -run "TestPlanCommand_FileToDatabase"
```

### Root Command Tests

```bash
# Root command tests
go test -v ./cmd/ -run "TestRoot"
```

## Command Documentation

For detailed documentation about each command, see:

- [`dump/README.md`](./dump/README.md) - Dump command documentation
- [`plan/README.md`](./plan/README.md) - Plan command documentation

### Global Flags

- `--debug`: Enable debug logging across all commands
- `--help`: Show help information for any command
