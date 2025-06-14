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
pgschema inspect --dsn "postgres://user:pass@host:port/dbname"
```

**Features:**
- Comprehensive schema extraction (tables, views, sequences, functions, triggers, constraints)
- Topological sorting for dependency-aware object creation order
- pg_dump-style output format
- Schema qualification for cross-schema references
- Support for foreign key constraints with referential actions

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
  - `cmd/inspect.go`: Main inspect command (693 lines)

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