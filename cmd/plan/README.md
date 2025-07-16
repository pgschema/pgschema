# Plan Command

This directory contains the implementation of the `pgschema plan` command, which generates migration plans to apply a desired schema state to a target database.

## Overview

The plan command compares a desired schema state (from a SQL file) with the current state of a database schema and generates a migration plan showing exactly what changes would be applied. This follows infrastructure-as-code principles similar to Terraform's plan command.

## Key Features

- **Unidirectional Planning**: Always from desired state (file) to current state (database)
- **Multiple Output Formats**: Text and JSON for easy integration
- **Dependency-Aware DDL**: Proper ordering for safe execution
- **Complete Change Detection**: Shows objects to add, modify, and drop
- **Schema Filtering**: Target specific schemas for comparison

## Running Tests

```bash
# All plan tests  
go test -v ./cmd/plan/

# Specific plan tests
go test -v ./cmd/plan/ -run "TestPlanCommand_FileToDatabase"
```

## Usage

### Basic Usage

```bash
# Generate plan to apply schema.sql to target database
pgschema plan --host localhost --db mydb --user myuser --file schema.sql

# Plan with specific schema
pgschema plan --host localhost --db mydb --user myuser --schema myschema --file desired-state.sql

# Plan with password
pgschema plan --host localhost --db mydb --user myuser --password mypass --file schema.sql
```

### Output Formats

```bash
# Text output (default)
pgschema plan --host localhost --db mydb --user myuser --file schema.sql

# JSON output for integration
pgschema plan --host localhost --db mydb --user myuser --file schema.sql --format json
```

## Architecture

The plan command uses:
- **Internal diff package**: For schema comparison logic
- **Internal plan package**: For migration plan generation
- **pgx/v5**: For PostgreSQL connectivity
- **Testcontainers**: For integration testing
