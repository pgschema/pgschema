# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

pgschema is a CLI tool to inspect and diff PostgreSQL schema. It provides comprehensive schema extraction with output compatible with `pg_dump`.

## Reference

- For 'inspect' command, we want to generate the schema file as close as `pg_dump`. Thus please use https://github.com/postgres/postgres/tree/master/src/bin/pg_dump as reference.
- `pgdump.sql` files under `./testdata/` folders are generated via `pg_dump`, you can use them to better understand the `pg_dump` output format.

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

#### `pgschema inspect`
Primary command for database schema inspection.

**Usage:**
```bash
pgschema inspect --host hostname --port 5432 --db dbname --user username
```

**Connection Options:**
- `--host`: Database server host (default: localhost)
- `--port`: Database server port (default: 5432)  
- `--db`: Database name (required)
- `--user`: Database user name (required)
- `--password`: Database password (optional, can also use PGPASSWORD env var)

**Password:**
You can provide the password using either the `--password` flag or the `PGPASSWORD` environment variable:
```bash
# Using password flag
pgschema inspect --host hostname --db dbname --user username --password mypassword

# Using environment variable
PGPASSWORD=password pgschema inspect --host hostname --db dbname --user username
```

**Features:**
- Comprehensive schema extraction (tables, views, sequences, functions, triggers, constraints)
- Topological sorting for dependency-aware object creation order
- pg_dump-style output format
- Schema qualification for cross-schema references
- Support for foreign key constraints with referential actions

#### `pgschema plan`
Generate migration plans by comparing two schema sources (databases or schema files).

**Usage:**
```bash
# Compare two schema files
pgschema plan --file1 schema1.sql --file2 schema2.sql

# Compare database to schema file
pgschema plan --db1 mydb --user1 myuser --file2 target.sql

# Compare two databases
pgschema plan --db1 prod_db --user1 user1 --db2 dev_db --user2 user2

# Compare specific schemas in databases
pgschema plan --db1 db1 --user1 user1 --schema1 public --db2 db2 --user2 user2 --schema2 staging
```

**Connection Options for Source 1:**
- `--host1`: Database server host for source 1 (default: localhost)
- `--port1`: Database server port for source 1 (default: 5432)  
- `--db1`: Database name for source 1
- `--user1`: Database user name for source 1
- `--password1`: Database password for source 1 (optional)
- `--schema1`: Schema name for source 1 (optional filter)
- `--file1`: Path to first SQL schema file

**Connection Options for Source 2:**
- `--host2`: Database server host for source 2 (default: localhost)
- `--port2`: Database server port for source 2 (default: 5432)
- `--db2`: Database name for source 2
- `--user2`: Database user name for source 2
- `--password2`: Database password for source 2 (optional)
- `--schema2`: Schema name for source 2 (optional filter)
- `--file2`: Path to second SQL schema file

**Output Options:**
- `--format`: Output format: text, json, preview (default: text)

**Password:**
You can provide passwords using the `--password1` and `--password2` flags:
```bash
# Using password flags for both sources
pgschema plan --db1 db1 --user1 user1 --password1 pass1 --db2 db2 --user2 user2 --password2 pass2

# Using password for one source (database to file comparison)
pgschema plan --db1 db1 --user1 user1 --password1 pass1 --file2 schema.sql
```

**Input Validation:**
- Each source must specify **either** a database connection **or** a schema file, but not both
- For database connections, both `--db` and `--user` are required
- Schema filtering is optional and only applies to database connections

**Features:**
- Flexible input sources: database connections or schema files
- Multiple output formats: text, JSON, preview
- Schema filtering for database sources
- Comprehensive diff analysis using existing diff/plan modules
- Consistent connection parameters (no shorthand flags to avoid conflicts)

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
  - `cmd/inspect.go`: Main inspect command (965 lines)
  - `cmd/plan.go`: Plan command for schema comparison (234 lines)

#### Database Layer
- `internal/queries/`: SQLC-generated database operations
  - `sqlc.yaml`: SQLC configuration
  - `queries.sql`: SQL queries for schema inspection (182 lines)
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

The inspect command handles:
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
4. **Structured Logging**: Debug logging throughout the inspection process
5. **pg_dump Compatibility**: Output format closely matches pg_dump style