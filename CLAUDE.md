# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Declarative schema migration for Postgres It provides comprehensive schema extraction with output compatible with `pg_dump`.

## Reference

- For 'dump' command, we want to generate the schema that's semantically equivalent to `pg_dump`, but be more developer friendly and terse. Please use https://github.com/postgres/postgres/tree/master/src/bin/pg_dump as a reference but do not blindly follow it and copy the exact format.
- `pgdump.sql` files under `./testdata/` folders are generated via `pg_dump`, you can use them to better understand the `pg_dump` output format.
- `raw.sql` files under `./testdata/` folders are the more developer friendly dump, that's the format we'd like to follow.

## Commands

### Build

```bash
# Install from GitHub
go install github.com/pgschema/pgschema@latest

# Build with verbose output
go build -v -o pgschema .
```

### Test

```bash
# Run all tests (requires Docker for integration tests)
go test -v ./...

# Run unit tests only (short mode)
go test -short -v ./...

# Run with coverage
go test -v -cover ./...

# Run specific test suites
go test -v ./internal/ir  # IR integration tests
go test -v ./cmd/dump     # Dump command tests
go test -v ./cmd/plan     # Plan command tests
go test -v ./cmd/apply    # Apply command tests

# Run tests with specific PostgreSQL version (default: 17)
PGSCHEMA_POSTGRES_VERSION=16 go test -v ./...
PGSCHEMA_POSTGRES_VERSION=14 go test -v ./...

# Test multiple PostgreSQL versions for release preparation
for version in 14 15 16 17; do
  echo "Testing with PostgreSQL $version"
  PGSCHEMA_POSTGRES_VERSION=$version go test -v ./...
done
```

#### Environment Variables for Testing

- `PGSCHEMA_POSTGRES_VERSION`: PostgreSQL version for test containers (default: "17"). Supported versions: 14, 15, 16, 17
- `PGSCHEMA_TEST_FILTER`: Filter for running specific diff tests (see internal/diff/diff_integration_test.go for examples)

### Code Generation

```bash
# Regenerate SQLC queries (from internal/queries/ directory)
cd internal/queries && sqlc generate
```

### Documentation

```bash
# Install Mintlify
npm i -g mint

# Run documentation server (from docs/ directory)
cd docs && mint dev
# View at http://localhost:3000
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

# Generate SQL migration script
pgschema plan --host hostname --db dbname --user username --file schema.sql --format sql
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

- `--format`: Output format: human, json, sql (default: human)

**Features:**

- Unidirectional planning: always from desired state (file) to current state (database)
- Shows what changes would be applied to make the database match the desired state
- Multiple output formats: human, JSON, SQL for easy integration
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

# Set custom lock timeout
pgschema apply --host hostname --db dbname --user username --file schema.sql --lock-timeout 5m

# Set custom application name for monitoring
pgschema apply --host hostname --db dbname --user username --file schema.sql --application-name "deployment-v1.2"
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
- `--lock-timeout`: Maximum time to wait for database locks (e.g., 30s, 5m, 1h)
- `--application-name`: Application name for database connection (default: pgschema)

**Features:**

- Shows migration plan before applying changes
- Interactive confirmation by default (can be bypassed with --auto-approve)
- Safe execution with detailed error reporting
- Same schema filtering and connection options as plan command
- Colored output for better readability

### Global Flags

- `--debug`: Enable debug logging

## Architecture

The application uses:

- **Cobra** for CLI structure
- **SQLC** for type-safe SQL query generation
- **pgx/v5** for PostgreSQL connectivity
- **Testcontainers** for integration testing
- **pg_query_go** for PostgreSQL query parsing

### Key Components

#### Core Structure

- `main.go`: Entry point that delegates to cmd package
- `cmd/`: Command implementations using Cobra
  - `cmd/root.go`: Root command and CLI setup with global flags
  - `cmd/dump/`: Dump command package with own README
  - `cmd/plan/`: Plan command package with own README
  - `cmd/apply/`: Apply command package with own README
  - `cmd/util/`: Shared utilities (connection handling)

#### Database Layer

- `internal/queries/`: SQLC-generated database operations
  - `sqlc.yaml`: SQLC configuration
  - `queries.sql`: SQL queries for schema dumping
  - `dml.sql.go`: Generated database operations
  - `models.sql.go`: Generated Go structs for database models
  - `queries.sql.go`: Generated query implementations

#### Core Packages

- `internal/ir/`: Intermediate Representation for schema parsing
  - Parser: Converts SQL files to IR
  - Inspector: Extracts schema from live database to IR
  - Normalizer: Ensures consistent representation
  - Comprehensive integration tests for multiple database types
- `internal/diff/`: Schema diffing and migration generation
  - Compares two IR schemas and generates differences
  - SQL writer for generating migration DDL
- `internal/plan/`: Migration planning logic
- `internal/color/`: Terminal color support
- `internal/version/`: Version management (VERSION file)

#### Testing

- `cmd/*_test.go`: Unit tests for each command
- `internal/ir/*_test.go`: IR package tests including integration tests
- `internal/diff/testdata/`: Schema change test scenarios
- `testdata/`: Sample databases for testing
  - `employee/`: Employee sample database
  - `sakila/`: DVD rental database (complex relationships)
  - `bytebase/`: Bytebase schema management
  - `tenant/`: Multi-tenant SaaS application
  - `gitlab/`: GitLab schema sample
  - `sourcegraph/`: Sourcegraph schema sample
  - Each contains `manifest.json`, `pgdump.sql`, and `pgschema.sql`

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
- `github.com/pganalyze/pg_query_go/v5 v5.1.0` - PostgreSQL query parser
- `github.com/lib/pq v1.10.9` - PostgreSQL driver (legacy compatibility)

**Testing Dependencies:**

- `github.com/testcontainers/testcontainers-go v0.37.0` - Integration testing
- `github.com/testcontainers/testcontainers-go/modules/postgres v0.37.0` - PostgreSQL test containers

**Requirements:**

- Go 1.23.0 or later for development
- PostgreSQL 14, 15, 16, 17 (for runtime usage)
- Docker (for running integration tests)

### Key Features

1. **Dependency Resolution**: Topological sorting ensures objects are created in the correct order
2. **Schema Qualification**: Automatic schema prefixing for cross-schema references
3. **Referential Actions**: Full support for ON DELETE/UPDATE clauses in foreign keys
4. **Structured Logging**: Debug logging throughout the dumping process
5. **pg_dump Compatibility**: Output format closely matches pg_dump style
6. **IR Pattern**: Two-path validation (Inspector and Parser) ensures semantic equivalence
7. **Infrastructure-as-Code**: Plan/apply workflow similar to Terraform

### Development Workflows

#### Version Management and Releases

- Version stored in `internal/version/VERSION`
- Updating version triggers automatic GitHub release
- Releases built with embedded version and commit information

#### SQLC Workflow

1. Edit queries in `internal/queries/queries.sql`
2. Run `cd internal/queries && sqlc generate`
3. Generated code appears as `*.sql.go` files

#### Testing Workflow

1. Unit tests validate command logic and basic functionality
2. Integration tests use testcontainers:
   - Spin up PostgreSQL container with specific version
   - Load test schema from `testdata/*/pgdump.sql`
   - Run pgschema commands
   - Compare output with expected results in `testdata/*/pgschema.sql`

#### IR (Intermediate Representation) Testing

The IR package has comprehensive tests ensuring:

- Parser can read pgschema output and convert to IR
- Inspector can extract schema from live database to IR
- Both paths produce equivalent IR structures
- Round-trip compatibility is maintained
