# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

pgschema is a CLI tool to dump and diff PostgreSQL schema. It provides comprehensive schema extraction with output compatible with `pg_dump`.

## Reference

- For 'dump' command, we want to generate the schema that's semantically equivalent to `pg_dump`, but be more developer friendly and terse. Please use https://github.com/postgres/postgres/tree/master/src/bin/pg_dump as a reference but do not blindly follow it and copy the exact format.
- `pgdump.sql` files under `./testdata/` folders are generated via `pg_dump`, you can use them to better understand the `pg_dump` output format.
- `raw.sql` files under `./testdata/` folders are the more developer friendly dump, that's the format we'd like to follow.

## Commands

### Build

```bash
# Install from GitHub
go install github.com/pgschema/pgschema@latest

# Build locally
go build -o pgschema .
```

### Test

```bash
# Run all tests
go test -v ./...

# Run with coverage
go test -v -cover ./...
```

### Dependencies

```bash
go mod tidy
```

### Available CLI Commands

#### `pgschema dump`

Primary command for database schema dumping.

**Usage:**

```bash
pgschema dump --host hostname --port 5432 --db dbname --user username

# Dump specific schema
pgschema dump --host hostname --db dbname --user username --schema myschema
```

**Connection Options:**

- `--host`: Database server host (default: localhost)
- `--port`: Database server port (default: 5432)
- `--db`: Database name (required)
- `--user`: Database user name (required)
- `--password`: Database password (optional, can also use PGPASSWORD env var)
- `--schema`: Schema name to dump (default: public)

**Password:**
You can provide the password using either the `--password` flag or the `PGPASSWORD` environment variable:

```bash
# Using password flag
pgschema dump --host hostname --db dbname --user username --password mypassword

# Using environment variable
PGPASSWORD=password pgschema dump --host hostname --db dbname --user username
```

**Features:**

- Comprehensive schema extraction (tables, views, sequences, functions, triggers, constraints)
- Topological sorting for dependency-aware object creation order
- pg_dump-style output format
- Schema qualification for cross-schema references
- Support for foreign key constraints with referential actions
- Schema filtering to dump only specific schemas

#### `pgschema plan`

Generate a migration plan to apply a desired schema state to a target database (similar to Terraform's plan command).

**Usage:**

```bash
# Generate plan to apply schema.sql to the target database
pgschema plan --host hostname --db dbname --user username --file schema.sql

# Generate plan with specific schema
pgschema plan --host hostname --db dbname --user username --schema myschema --file desired-state.sql

# Generate plan with password
pgschema plan --host hostname --db dbname --user username --password mypassword --file schema.sql

# Generate plan with JSON output
pgschema plan --host hostname --db dbname --user username --file schema.sql --format json
```

**Target Database Connection Options:**

- `--host`: Database server host (default: localhost)
- `--port`: Database server port (default: 5432)
- `--db`: Database name (required)
- `--user`: Database user name (required)
- `--password`: Database password (optional)
- `--schema`: Schema name (default: public)

**Desired State Options:**

- `--file`: Path to desired state SQL schema file (required)

**Output Options:**

- `--format`: Output format: human, json (default: human)

**Password:**
You can provide the password using the `--password` flag:

```bash
pgschema plan --host hostname --db dbname --user username --password mypassword --file schema.sql
```

**Features:**

- Unidirectional planning: always from desired state (file) to current state (database)
- Shows what changes would be applied to make the database match the desired state
- Multiple output formats: human, JSON for easy integration
- Consistent with infrastructure-as-code principles
- Organized output by object types (tables, views, functions, sequences, etc.)
- Summary counts showing X objects to add, Y to modify, Z to drop for each type
- Complete DDL statements showing exactly what SQL will be executed
- Dependency-aware DDL ordering for safe execution

#### `pgschema apply`

Apply a desired schema state to a target database schema. Compares the desired state (from a file) with the current state and applies the necessary changes.

**Usage:**

```bash
# Apply changes with confirmation prompt
pgschema apply --host hostname --db dbname --user username --file schema.sql

# Apply changes without confirmation
pgschema apply --host hostname --db dbname --user username --file schema.sql --auto-approve

# Apply to specific schema
pgschema apply --host hostname --db dbname --user username --schema myschema --file schema.sql

# Apply with password
pgschema apply --host hostname --db dbname --user username --password mypassword --file schema.sql
```

**Target Database Connection Options:**

- `--host`: Database server host (default: localhost)
- `--port`: Database server port (default: 5432)
- `--db`: Database name (required)
- `--user`: Database user name (required)
- `--password`: Database password (optional)
- `--schema`: Schema name (default: public)

**Apply Options:**

- `--file`: Path to desired state SQL schema file (required)
- `--auto-approve`: Apply changes without prompting for approval
- `--no-color`: Disable colored output

**Password:**
You can provide the password using the `--password` flag:

```bash
pgschema apply --host hostname --db dbname --user username --password mypassword --file schema.sql
```

**Features:**

- Shows migration plan before applying changes
- Interactive confirmation by default (can be bypassed with --auto-approve)
- Safe execution with detailed error reporting
- Same schema filtering and connection options as plan command
- Colored output for better readability

#### `pgschema version`

Display version information.

**Usage:**

```bash
pgschema version
```

### Global Flags

- `--debug`: Enable debug logging

## Architecture

The application uses:

- **Cobra** for CLI structure
- **SQLC** for type-safe SQL query generation
- **pgx/v5** for PostgreSQL connectivity
- **Testcontainers** for integration testing

### Key Components

#### Core Structure

- `main.go`: Entry point that delegates to cmd package
- `cmd/`: Command implementations using Cobra
  - `cmd/root.go`: Root command and CLI setup with global flags
  - `cmd/version.go`: Version command implementation
  - `cmd/dump.go`: Main dump command (965 lines)
  - `cmd/plan.go`: Plan command for schema comparison (234 lines)
  - `cmd/apply.go`: Apply command for schema migration execution

#### Database Layer

- `internal/queries/`: SQLC-generated database operations
  - `sqlc.yaml`: SQLC configuration
  - `queries.sql`: SQL queries for schema dumping (182 lines)
  - `dml.sql.go`: Generated database operations
  - `models.sql.go`: Generated Go structs for database models
  - `queries.sql.go`: Generated query implementations

#### Testing

- `cmd/*_test.go`: Unit tests for each command
- `testdata/`: Sample databases for testing
  - `employee/`: Employee sample database
  - `bytebase/`: Bytebase sample
  - `gitlab/`: GitLab schema sample
  - `sourcegraph/`: Sourcegraph schema sample
  - Each contains `manifest.json`, `pgdump.sql`, and optionally `mine.sql`

### Database Objects Supported

The dump command handles:

- **Schemas**: User-defined schemas (excluding system schemas)
- **Tables**: Both BASE TABLE and VIEW types
- **Columns**: Full metadata including types, constraints, defaults
- **Constraints**: PRIMARY KEY, UNIQUE, FOREIGN KEY, CHECK constraints
- **Sequences**: Including ownership relationships
- **Functions**: User-defined functions with parameters and return types
- **Views**: With dependency resolution for proper ordering
- **Triggers**: Associated with tables
- **Indexes**: Basic index information
- **Extensions**: Placeholder support

### Dependencies

**Core Dependencies:**

- `github.com/jackc/pgx/v5 v5.7.5` - PostgreSQL driver
- `github.com/spf13/cobra v1.9.1` - CLI framework

**Testing Dependencies:**

- `github.com/testcontainers/testcontainers-go v0.37.0` - Integration testing
- `github.com/testcontainers/testcontainers-go/modules/postgres v0.37.0` - PostgreSQL test containers

### Key Features

1. **Dependency Resolution**: Topological sorting ensures objects are created in the correct order
2. **Schema Qualification**: Automatic schema prefixing for cross-schema references
3. **Referential Actions**: Full support for ON DELETE/UPDATE clauses in foreign keys
4. **Structured Logging**: Debug logging throughout the dumping process
5. **pg_dump Compatibility**: Output format closely matches pg_dump style
