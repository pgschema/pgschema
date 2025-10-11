# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`pgschema` is a CLI tool that brings Terraform-style declarative schema migration workflow to PostgreSQL. It provides a dump/edit/plan/apply workflow for database schema changes:

- **Dump**: Extract current schema in a developer-friendly format with support for all common objects
- **Edit**: Modify schema files to represent desired state
- **Plan**: Compare desired state with current database and generate migration plan
- **Apply**: Execute the migration with safety features like concurrent change detection, transaction-adaptive execution, and lock timeout control

The tool is written in Go 1.24+ and uses Cobra for CLI commands, embedded-postgres for integration testing, and supports PostgreSQL versions 14-17.

## Build and Development Commands

### Build

```bash
# Standard build
go build -o pgschema .

# Build with version info (used in CI/Docker)
CGO_ENABLED=1 go build -ldflags="-w -s -X github.com/pgschema/pgschema/cmd.GitCommit=... -X 'github.com/pgschema/pgschema/cmd.BuildDate=...'" -o pgschema .
```

### Testing

```bash
# Unit tests only (fast, no embedded Postgres required)
go test -short -v ./...

# All tests including integration tests (uses embedded Postgres, no Docker required)
go test -v ./...

# Test specific packages
go test -v ./cmd/...
go test -v ./internal/diff/...

# Run specific test cases with pattern filtering
PGSCHEMA_TEST_FILTER="create_table/" go test -v ./cmd -run TestPlanAndApply
PGSCHEMA_TEST_FILTER="create_table/add_column" go test -v ./cmd -run TestPlanAndApply
```

### Utilities

```bash
# Build for specific platforms (used in CI)
GOOS=linux GOARCH=amd64 go build -o pgschema-linux-amd64 .
GOOS=darwin GOARCH=arm64 go build -o pgschema-darwin-arm64 .

# Run with debug logging
./pgschema --debug dump --host localhost --db mydb --user postgres
```

## Architecture

### Core Components

**CLI Commands** (`cmd/`):

- `dump/` - Schema extraction from live database
- `plan/` - Migration planning by comparing schemas
- `apply/` - Migration execution with safety checks
- `root.go` - Main CLI setup with Cobra

**Core Packages**:

- `ir/` - Intermediate Representation (IR) package - now a separate module with its own versioning
  - Contains schema objects (tables, indexes, functions, procedures, triggers, policies, etc.)
  - Parser for SQL DDL statements using pg_query_go
  - Database inspector for extracting schema from live Postgres instances
  - Normalizer for consistent representation
  - Utilities for quoting and formatting

**Internal Packages** (`internal/`):

- `diff/` - Schema comparison and migration plan generation
  - Compares IR representations to generate migration steps
  - Handles topological sorting for dependency ordering
  - Generates DDL for various object types (tables, columns, constraints, indexes, triggers, etc.)
- `plan/` - Migration plan structures and execution logic
- `dump/` - Schema dump formatting and output
- `fingerprint/` - Schema fingerprinting for change detection
- `include/` - Include file processing for modular schemas
- `color/` - Terminal output colorization
- `logger/` - Structured logging utilities
- `version/` - Version information and compatibility checks

### Key Architecture Patterns

**Schema Representation**: Uses an Intermediate Representation (IR) to normalize schema objects from both parsed SQL files and live database introspection. This allows comparing schemas from different sources.

**Migration Planning**: The `diff` package compares IR representations to generate a sequence of migration steps with proper dependency ordering (topological sort).

**Database Integration**: Uses `pgx/v5` for database connections and `embedded-postgres` for integration testing against real PostgreSQL instances (no Docker required).

**SQL Parsing**: Leverages `pg_query_go/v6` (libpg_query bindings) for parsing PostgreSQL DDL statements.

**Modular Architecture**: The IR package is a separate Go module that can be versioned and used independently, allowing external tools to leverage the schema representation and parsing capabilities.

### Testing Structure

**Integration Tests**: Use `embedded-postgres` to spin up real PostgreSQL instances. Located in `*_integration_test.go` files. No Docker required.

**Diff Testing**: Extensive test cases in `testdata/diff/` with old/new schema pairs and expected migration SQL. Use `PGSCHEMA_TEST_FILTER` environment variable to run specific test cases.

**Test Data Categories** (`testdata/diff/`):

- `comment/` - COMMENT ON statements for various objects
- `create_domain/` - Domain type creation and modification
- `create_function/` - Function creation, modification, and deletion
- `create_index/` - Index operations including concurrent creation, partial indexes
- `create_materialized_view/` - Materialized view operations
- `create_policy/` - Row-level security policy operations
- `create_procedure/` - Stored procedure operations
- `create_sequence/` - Sequence operations
- `create_table/` - Table operations (40+ test cases for columns, constraints, etc.)
- `create_trigger/` - Trigger creation and modification
- `create_type/` - Custom type operations
- `create_view/` - View creation and modification
- `dependency/` - Cross-object dependency handling
- `online/` - Online migration operations (concurrent indexes, NOT VALID constraints, etc.)
- `migrate/` - Complex migration scenarios

**Other Test Data**:

- `testdata/dump/` - Sample database dumps (employee, sakila, bytebase, tenant)
- `testdata/include/` - Include file processing test cases

## Common Development Patterns

### Adding New Schema Object Support

1. Add IR representation in `ir/ir.go`
2. Add database introspection logic in `ir/inspector.go`
3. Add parsing logic in `ir/parser.go`
4. Add diff logic in `internal/diff/`
5. Add test cases in `testdata/diff/create_[object_type]/`

### Supported Schema Objects

The tool currently supports:

- **Tables**: columns, constraints (PK, FK, UNIQUE, CHECK), identity columns, partitioning, LIKE clauses
- **Indexes**: regular, unique, partial, concurrent creation, multi-column
- **Views**: regular and materialized views
- **Functions**: including arguments, return types, volatility, security definer
- **Procedures**: stored procedures with parameters
- **Triggers**: table triggers with timing and events
- **Sequences**: with full configuration options
- **Types**: composite types, enums, domains
- **Policies**: row-level security policies
- **Comments**: on all supported object types
- **Aggregates**: custom aggregate functions

### Database Connection Patterns

Integration tests use connection helper in `cmd/util/connection.go`. Environment variables:

- **Database connection**: `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`
- **Environment files**: `.env` support via `godotenv` - automatically loaded by main.go
- **Test filtering**: `PGSCHEMA_TEST_FILTER` for running specific test cases (supports directory paths like `create_table/` or specific tests like `create_table/add_column`)
- **Postgres version**: `PGSCHEMA_POSTGRES_VERSION` to test against specific Postgres versions (14, 15, 16, 17)

### SQL Generation

The tool generates PostgreSQL DDL statements. Key considerations:

- **Identifier quoting**: Proper escaping and quoting of identifiers (see `ir/quote.go`)
- **Transaction handling**: Transaction-adaptive execution (some operations like CREATE INDEX CONCURRENTLY require non-transactional execution)
- **Online operations**: Support for concurrent index creation, NOT VALID constraints
- **Dependency ordering**: Topological sort ensures objects are created/dropped in correct order
- **Safety features**:
  - Concurrent change detection via fingerprinting
  - Lock timeout control
  - Optional confirmation prompts

### Testing Best Practices

When adding new features or fixing bugs:

1. **Use test filtering** to run relevant tests quickly:
   ```bash
   PGSCHEMA_TEST_FILTER="create_table/add_column" go test -v ./cmd -run TestPlanAndApply
   ```

2. **Generate expected outputs** with `--generate` flag (test files only):
   ```bash
   PGSCHEMA_TEST_FILTER="online/add_index" go test -v ./cmd -run TestPlanAndApply --generate
   ```

3. **Verify against multiple Postgres versions**:
   ```bash
   PGSCHEMA_POSTGRES_VERSION=14 go test -v ./cmd/dump -run TestDumpCommand_Employee
   PGSCHEMA_POSTGRES_VERSION=17 go test -v ./cmd/dump -run TestDumpCommand_Employee
   ```

4. **Test end-to-end**: The `TestPlanAndApply` integration test in `cmd/migrate_integration_test.go` tests the full workflow

### Key Files to Know

- `main.go` - Entry point, loads .env file and calls cmd.Execute()
- `cmd/root.go` - Root CLI command setup with global flags
- `ir/ir.go` - Core IR data structures for all schema objects
- `ir/parser.go` - SQL DDL parsing using pg_query_go
- `ir/inspector.go` - Database introspection queries
- `internal/diff/diff.go` - Main diff logic and orchestration
- `internal/diff/table.go` - Table-specific diff operations (largest file)
- `internal/plan/plan.go` - Migration plan execution
- `cmd/migrate_integration_test.go` - Main integration test suite
