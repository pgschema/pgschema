# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`pgschema` is a CLI tool that brings Terraform-style declarative schema migration workflow to PostgreSQL. It provides a dump/edit/plan/apply workflow for database schema changes:

- **Dump**: Extract current schema in a developer-friendly format with support for all common objects
- **Edit**: Modify schema files to represent desired state
- **Plan**: Compare desired state with current database and generate migration plan
- **Apply**: Execute the migration with safety features like concurrent change detection, transaction-adaptive execution, and lock timeout control

The tool is written in Go 1.24+ (toolchain go1.24.7) and uses:
- Cobra for CLI commands
- embedded-postgres v1.29.0 for integration testing (no Docker required)
- pgx/v5 v5.7.5 for database connections
- pg_query_go/v6 v6.1.0 for SQL parsing
- Supports PostgreSQL versions 14-17

Key differentiators:
- Comprehensive Postgres support for virtually all schema-level objects
- State-based Terraform-like workflow (no migration history table)
- Schema-level focus for single-schema apps to multi-tenant architectures
- No shadow database required - works directly with schema files and target database

## Skills

This project includes specialized skills for development workflows. Use these skills when working on pgschema:

- **pg_dump Reference** - Consult PostgreSQL's pg_dump implementation for system catalog queries and schema extraction patterns
- **PostgreSQL Syntax Reference** - Understand PostgreSQL's parser and grammar (gram.y) for SQL syntax and DDL structure
- **Validate with Database** - Connect to live PostgreSQL to validate assumptions, compare pg_dump vs pgschema, and query system catalogs
- **Run Tests** - Execute automated Go tests to validate diff logic, plan generation, and dump functionality

Skills are located in `.claude/skills/` and provide detailed workflows for common development tasks.

## Quick Start Commands

### Build

```bash
# Standard build
go build -o pgschema .

# Build with version info (used in CI/Docker)
CGO_ENABLED=1 go build -ldflags="-w -s -X github.com/pgschema/pgschema/cmd.GitCommit=... -X 'github.com/pgschema/pgschema/cmd.BuildDate=...'" -o pgschema .
```

### Testing

For detailed testing workflows, see the **Run Tests** skill (`.claude/skills/run_tests/SKILL.md`).

```bash
# Quick validation - diff tests only (fast, no database)
PGSCHEMA_TEST_FILTER="create_trigger/" go test -v ./internal/diff -run TestDiffFromFiles

# Full integration test
PGSCHEMA_TEST_FILTER="create_trigger/" go test -v ./cmd -run TestPlanAndApply

# All tests
go test -v ./...
```

### Database Validation

For interactive database validation, see the **Validate with Database** skill (`.claude/skills/validate_db/SKILL.md`).

Connection details are in `.env`:
```
PGHOST=localhost
PGDATABASE=employee
PGUSER=postgres
PGPASSWORD=testpwd1
```

## Architecture

### Core Components

**CLI Commands** (`cmd/`):
- `dump/` - Schema extraction from live database
- `plan/` - Migration planning by comparing schemas
- `apply/` - Migration execution with safety checks
- `util/` - Shared utilities (connection, env, ignore file processing)
- `root.go` - Main CLI setup with Cobra

**Core Packages**:
- `ir/` - Intermediate Representation (IR) package - separate Go module
  - Schema objects (tables, indexes, functions, procedures, triggers, policies, etc.)
  - SQL parser using pg_query_go
  - Database inspector using pgx
  - Schema normalizer
  - Identifier quoting utilities

**Internal Packages** (`internal/`):
- `diff/` - Schema comparison and migration DDL generation
- `plan/` - Migration plan structures and execution
- `dump/` - Schema dump formatting and output
- `fingerprint/` - Schema fingerprinting for change detection
- `include/` - Include file processing for modular schemas
- `color/` - Terminal output colorization
- `logger/` - Structured logging
- `version/` - Version information

### Key Architecture Patterns

**Schema Representation**: Uses an Intermediate Representation (IR) to normalize schema objects from both parsed SQL files and live database introspection. This allows comparing schemas from different sources.

**Migration Planning**: The `diff` package compares IR representations to generate a sequence of migration steps with proper dependency ordering (topological sort).

**Database Integration**: Uses `pgx/v5` for database connections and `embedded-postgres` for integration testing against real PostgreSQL instances (no Docker required).

**SQL Parsing**: Leverages `pg_query_go/v6` (libpg_query bindings) for parsing PostgreSQL DDL statements. For understanding PostgreSQL syntax, see the **PostgreSQL Syntax Reference** skill.

**Modular Architecture**: The IR package is a separate Go module that can be versioned and used independently.

## Common Development Workflows

### Adding New Schema Object Support

1. Add IR representation in `ir/ir.go`
2. Add database introspection logic in `ir/inspector.go` (consult **pg_dump Reference** skill for system catalog queries)
3. Add parsing logic in `ir/parser.go` (consult **PostgreSQL Syntax Reference** skill for grammar)
4. Add diff logic in `internal/diff/`
5. Add test cases in `testdata/diff/create_[object_type]/` (see **Run Tests** skill)
6. Validate with live database (see **Validate with Database** skill)

### Debugging Schema Extraction

1. Consult **pg_dump Reference** skill to understand correct system catalog queries
2. Use **Validate with Database** skill to test queries against live PostgreSQL
3. Compare pg_dump output with pgschema output using workflows in **Validate with Database** skill

### Understanding SQL Syntax

1. Consult **PostgreSQL Syntax Reference** skill to find grammar rules in gram.y
2. Understand how pg_query_go parse tree maps to grammar
3. Test parsing with real SQL using **Validate with Database** skill

### Testing Changes

1. Use **Run Tests** skill for comprehensive testing workflows
2. Start with fast diff tests: `PGSCHEMA_TEST_FILTER="..." go test -v ./internal/diff -run TestDiffFromFiles`
3. Run integration tests: `PGSCHEMA_TEST_FILTER="..." go test -v ./cmd -run TestPlanAndApply`
4. Validate with live database using **Validate with Database** skill

## Supported Schema Objects

The tool supports comprehensive PostgreSQL schema objects (see `ir/ir.go` for complete data structures):

- **Tables**: Columns, identity columns, generated columns, partitioning, LIKE clauses
- **Constraints**: Primary keys, foreign keys, unique constraints, check constraints, NOT VALID
- **Indexes**: Regular, unique, partial, functional/expression indexes, concurrent creation
- **Views**: Regular views and materialized views with indexes
- **Functions**: Parameters, return types, volatility, STRICT, SECURITY DEFINER
- **Procedures**: Parameters, signatures, language
- **Triggers**: Timing, events, levels, WHEN conditions, constraint triggers, REFERENCING OLD/NEW TABLE
- **Sequences**: Start, increment, min/max, cycle, cache, owned by tracking
- **Types**: Enum, composite, domain types with constraints
- **Policies**: Row-level security with commands, roles, USING/WITH CHECK expressions
- **Aggregates**: Custom aggregates with transition and final functions
- **Comments**: On all supported object types

## Environment Variables

- **Database connection**: `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`
- **Environment files**: `.env` - automatically loaded by main.go
- **Test filtering**: `PGSCHEMA_TEST_FILTER` - run specific test cases (e.g., `"create_table/"` or `"create_table/add_column"`)
- **Postgres version**: `PGSCHEMA_POSTGRES_VERSION` - test against specific versions (14, 15, 16, 17)

## Important Implementation Notes

**Trigger Features**:
- Full support for WHEN conditions using `pg_get_expr(t.tgqual, t.tgrelid, false)` from `pg_catalog.pg_trigger`
- Constraint triggers with deferrable options
- REFERENCING OLD TABLE / NEW TABLE for statement-level triggers

**Online Migration Support**:
- CREATE INDEX CONCURRENTLY for non-blocking index creation
- ALTER TABLE ... ADD CONSTRAINT ... NOT VALID for online constraint addition
- Proper transaction handling - some operations must run outside transactions

**pgschema Directives**:
- Special SQL comments control behavior: `--pgschema-lock-timeout`, `--pgschema-no-transaction`
- Handled in `cmd/apply/directive.go`

**Reference Implementations**:
- PostgreSQL's pg_dump serves as reference for system catalog queries (see **pg_dump Reference** skill)
- PostgreSQL's gram.y defines canonical SQL syntax (see **PostgreSQL Syntax Reference** skill)

## Key Files Reference

**Entry Point & CLI**:
- `main.go` - Entry point, loads .env and calls cmd.Execute()
- `cmd/root.go` - Root CLI with global flags

**IR Package** (separate Go module at `./ir`):
- `ir/ir.go` - Core IR data structures for all schema objects
- `ir/parser.go` - SQL DDL parsing using pg_query_go
- `ir/inspector.go` - Database introspection using pgx
- `ir/normalizer.go` - Schema normalization
- `ir/quote.go` - Identifier quoting utilities

**Diff Package** (`internal/diff/`):
- `diff.go` - Main diff logic, topological sorting
- `table.go`, `index.go`, `trigger.go`, `view.go`, `function.go`, `procedure.go`, `sequence.go`, `type.go`, `policy.go`, `aggregate.go` - Object-specific diff operations

**Testing**:
- `cmd/migrate_integration_test.go` - Main integration test suite (TestPlanAndApply)
- `testdata/diff/` - 100+ test cases covering all schema object types
- See **Run Tests** skill for complete testing workflows

## Test Data Structure

Tests are organized in `testdata/diff/` by object type:
- `comment/` (8 tests), `create_domain/` (3), `create_function/` (4), `create_index/` (1)
- `create_materialized_view/` (3), `create_policy/` (8), `create_procedure/` (3), `create_sequence/` (3)
- `create_table/` (40 tests), `create_trigger/` (7), `create_type/` (3), `create_view/` (6)
- `dependency/` (3), `online/` (12), `migrate/` (6)

Each test case contains: `old.sql` (starting state), `new.sql` (desired state), `expected.sql` (expected migration DDL)

For detailed test workflows, filtering, and regeneration, see the **Run Tests** skill.
