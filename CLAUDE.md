# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`pgschema` is a CLI tool that brings Terraform-style declarative schema migration workflow to PostgreSQL. It provides a dump/edit/plan/apply workflow for database schema changes:

- **Dump**: Extract current schema in a developer-friendly format
- **Edit**: Modify schema files to represent desired state
- **Plan**: Compare desired state with current database and generate migration plan
- **Apply**: Execute the migration with safety features like concurrent change detection

The tool is written in Go 1.23+ and uses Cobra for CLI commands, testcontainers for integration testing, and supports PostgreSQL versions 14-17.

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
# Unit tests only (fast, no Docker required)
go test -short -v ./...

# All tests including integration tests (requires Docker for testcontainers)
go test -v ./...

# Test specific packages
go test -v ./cmd/...
go test -v ./internal/diff/...

# Run specific test cases with pattern filtering
PGSCHEMA_TEST_FILTER="create_table/" go test -v ./cmd -run TestPlanAndApply
PGSCHEMA_TEST_FILTER="create_table/add_column" go test -v ./cmd -run TestPlanAndApply
```

### Code Generation

```bash
# Generate SQL queries using sqlc (requires PostgreSQL connection)
sqlc generate
```

## Architecture

### Core Components

**CLI Commands** (`cmd/`):

- `dump/` - Schema extraction from live database
- `plan/` - Migration planning by comparing schemas
- `apply/` - Migration execution with safety checks
- `root.go` - Main CLI setup with Cobra

**Internal Packages** (`internal/`):

- `ir/` - Intermediate Representation of schema objects (tables, indexes, functions, etc.)
- `diff/` - Schema comparison and migration plan generation
- `queries/` - Generated SQL queries using sqlc for database introspection
- `fingerprint/` - Schema fingerprinting for change detection
- `plan/` - Migration plan structures and execution logic
- `dump/` - Schema dump formatting and output
- `include/` - Include file processing for modular schemas
- `color/` - Terminal output colorization
- `logger/` - Structured logging utilities
- `version/` - Version information and compatibility checks

### Key Architecture Patterns

**Schema Representation**: Uses an Intermediate Representation (IR) to normalize schema objects from both parsed SQL files and live database introspection. This allows comparing schemas from different sources.

**Migration Planning**: The `diff` package compares IR representations to generate a sequence of migration steps with proper dependency ordering (topological sort).

**Database Integration**: Uses `pgx/v5` for database connections and `testcontainers` for integration testing against real PostgreSQL instances.

**SQL Parsing**: Leverages `pg_query_go` (libpg_query bindings) for parsing PostgreSQL DDL statements.

### Testing Structure

**Integration Tests**: Use testcontainers to spin up real PostgreSQL instances. Located in `*_integration_test.go` files.

**Diff Testing**: Extensive test cases in `testdata/diff/` with old/new schema pairs and expected migration SQL. Use `PGSCHEMA_TEST_FILTER` environment variable to run specific test cases.

**Test Data**:

- `testdata/dump/` - Sample database dumps (employee, sakila, bytebase, tenant)
- `testdata/diff/` - Schema comparison test cases organized by operation type
- `testdata/include/` - Include file processing test cases
- `testdata/migrate/` - Migration test cases and scenarios

## Common Development Patterns

### Adding New Schema Object Support

1. Add IR representation in `internal/ir/ir.go`
2. Add database introspection queries in `internal/queries/queries.sql`
3. Add parsing logic in `internal/ir/parser.go`
4. Add diff logic in `internal/diff/`
5. Add test cases in `testdata/diff/create_[object_type]/`

### Database Connection Patterns

Integration tests use connection helper in `cmd/util/connection.go`. Environment variables for testing:

- `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE` for database connections
- `PGSCHEMA_TEST_FILTER` for filtering integration test cases

### SQL Generation

The tool generates PostgreSQL DDL statements. Key considerations:

- Proper escaping and quoting of identifiers
- Transaction boundaries for safety
- Concurrent index creation support
- Row-level security (RLS) policy handling
