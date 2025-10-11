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
  - `dump.go` - Main dump logic
  - `dump_integration_test.go` - Integration tests for dump command
  - `dump_permission_integration_test.go` - Permission-related tests
  - `multifile_test.go` - Tests for multi-file schema dumps
- `plan/` - Migration planning by comparing schemas
  - `plan.go` - Main plan logic
  - `plan_integration_test.go` - Integration tests for plan command
  - `output_test.go` - Tests for plan output formatting
- `apply/` - Migration execution with safety checks
  - `apply.go` - Main apply logic
  - `apply_integration_test.go` - Integration tests for apply command
  - `directive.go` - Handles pgschema directives (e.g., --pgschema-lock-timeout)
- `util/` - Shared utilities
  - `connection.go` - Database connection helpers
  - `env.go` - Environment variable handling
  - `ignoreloader.go` - Schema ignore file processing
- `root.go` - Main CLI setup with Cobra
- Integration test files at cmd level:
  - `migrate_integration_test.go` - Main end-to-end migration tests (TestPlanAndApply)
  - `schema_integration_test.go` - Schema-specific integration tests
  - `include_integration_test.go` - Include file processing tests
  - `ignore_integration_test.go` - Schema ignore functionality tests

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

- `comment/` - COMMENT ON statements for various objects (8 test cases)
  - add_column_comments, add_index_comment, add_table_comment, add_view_comment
  - alter_table_comment, drop_table_comment, mixed_comments, noop_column_comments
- `create_domain/` - Domain type creation and modification (3 test cases)
- `create_function/` - Function creation, modification, and deletion (4 test cases)
- `create_index/` - Index operations including concurrent creation, partial indexes (1 test case)
- `create_materialized_view/` - Materialized view operations (3 test cases)
- `create_policy/` - Row-level security policy operations (8 test cases)
- `create_procedure/` - Stored procedure operations (3 test cases)
- `create_sequence/` - Sequence operations (3 test cases)
- `create_table/` - Table operations (40 test cases for columns, constraints, etc.)
  - Columns: add_column_* (array, boolean, generated, identity, integer, jsonb, numeric, serial, text, timestamp, uuid, varchar)
  - Primary keys: add_pk_* (bigint, composite, identity, serial, single, text, uuid)
  - Unique constraints: add_uk_* (bigint, composite, identity, serial, single, text, uuid), add_unique_constraint
  - Other: add_check, add_table, add_table_composite_keys, add_table_like, add_table_like_forward_ref
  - Modifications: alter_column_types, alter_defaults, drop_column, remove_not_null
  - Special: add_table_no_online_rewrite, add_table_partitioned, add_table_serial_pk
- `create_trigger/` - Trigger creation and modification (7 test cases)
  - add_trigger, add_trigger_constraint, add_trigger_old_table, add_trigger_system_catalog
  - add_trigger_when_distinct, alter_trigger, drop_trigger
- `create_type/` - Custom type operations (3 test cases)
- `create_view/` - View creation and modification (6 test cases)
- `dependency/` - Cross-object dependency handling (3 test cases)
- `online/` - Online migration operations (12 test cases)
  - Indexes: add_composite_index, add_functional_index, add_materialized_view_index, add_partial_index, add_unique_multi_column_index
  - Constraints: add_constraint, add_fk, add_not_null
  - Alterations: alter_composite_index, alter_constraint, alter_fk, alter_materialized_view_index
- `migrate/` - Complex migration scenarios (6 test cases)

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

The tool currently supports (see `ir/ir.go` for complete data structures):

- **Tables** (`Table`):
  - Columns with data types, nullability, defaults, max_length, precision, scale
  - Identity columns (GENERATED ALWAYS/BY DEFAULT AS IDENTITY)
  - Generated columns (GENERATED ALWAYS AS ... STORED)
  - Partitioning (RANGE, LIST, HASH)
  - LIKE clauses with options (INCLUDING ALL, etc.)
  - Table dependencies tracking
- **Constraints** (`Constraint`):
  - Primary keys (single column, composite)
  - Foreign keys with ON DELETE/UPDATE rules, deferrable options
  - Unique constraints (single column, composite)
  - Check constraints with custom expressions
  - NOT VALID support for online migrations
- **Indexes** (`Index`):
  - Regular, unique, and primary key indexes
  - Multi-column indexes with column ordering
  - Partial indexes (with WHERE clauses)
  - Functional/expression indexes
  - Index methods (btree, hash, gin, gist, etc.)
  - Concurrent creation support
- **Views** (`View`):
  - Regular views
  - Materialized views with indexes
  - View definitions and comments
- **Functions** (`Function`):
  - Parameters with modes (IN, OUT, INOUT) and default values
  - Return types and language
  - Volatility (IMMUTABLE, STABLE, VOLATILE)
  - STRICT and SECURITY DEFINER options
  - Function signatures for overloading
- **Procedures** (`Procedure`):
  - Parameters with modes and types
  - Procedure signatures
  - Language and definition
- **Triggers** (`Trigger`):
  - Timing (BEFORE, AFTER, INSTEAD OF)
  - Events (INSERT, UPDATE, DELETE, TRUNCATE)
  - Level (ROW, STATEMENT)
  - WHEN conditions
  - Constraint triggers with deferrable options
  - REFERENCING OLD TABLE / NEW TABLE support
- **Sequences** (`Sequence`):
  - Start value, increment, min/max values
  - Cycle options and cache
  - Owned by table/column tracking
- **Types** (`Type`):
  - Enum types with values
  - Composite types with columns
  - Domain types with base type, NOT NULL, defaults, and constraints
- **Policies** (`RLSPolicy`):
  - Row-level security policies
  - Commands (SELECT, INSERT, UPDATE, DELETE, ALL)
  - Permissive/restrictive policies
  - Role-based policies
  - USING and WITH CHECK expressions
- **Comments**: On all supported object types (tables, columns, indexes, views, functions, procedures, triggers, sequences, types, policies)
- **Aggregates** (`Aggregate`):
  - Custom aggregate functions
  - Transition and final functions
  - State type and initial conditions

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
   # Run all tests in a category
   env PGSCHEMA_TEST_FILTER="create_table/" go test -v ./cmd -run TestPlanAndApply

   # Run specific test case
   env PGSCHEMA_TEST_FILTER="create_table/add_column" go test -v ./cmd -run TestPlanAndApply

   # Run with timeout for longer tests
   env PGSCHEMA_TEST_FILTER="create_trigger/add_trigger" go test -v ./cmd -run TestPlanAndApply -timeout 2m

   # Run diff tests (faster, no database required)
   env PGSCHEMA_TEST_FILTER="create_view/add_view" go test -v ./internal/diff -run TestDiffFromFiles
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

4. **Test end-to-end**: The `TestPlanAndApply` integration test in `cmd/migrate_integration_test.go` tests the full workflow:
   - Creates a test database with embedded-postgres
   - Applies old schema from testdata
   - Generates plan comparing new schema with database
   - Applies the plan
   - Verifies final state matches expected schema

5. **Test structure**: Each test case in `testdata/diff/` contains:
   - `old.sql` - Starting schema state
   - `new.sql` - Desired schema state
   - `expected.sql` - Expected migration DDL
   - Tests verify generated DDL matches expected output

### Important Implementation Notes

**Trigger Features**:
- Full support for trigger WHEN conditions (see `testdata/diff/create_trigger/add_trigger_when_distinct`)
- Constraint trigger support with deferrable options
- REFERENCING OLD TABLE / NEW TABLE support for statement-level triggers
- System catalog queries correctly extract WHEN clauses using `pg_get_expr(t.tgqual, t.tgrelid, false)`

**Procedure Mode Classification**:
- Procedures are correctly classified by their mode (see recent commit "feat: correct classify modify procedure")
- Changes to procedure parameters or definition trigger proper DROP/CREATE sequences

**Comment Handling**:
- Comments now use the `comment/` directory structure (refactored from `comment_on/`)
- Support for comments on all object types: tables, columns, indexes, views, functions, procedures, triggers, sequences, types, policies

**Online Migration Support**:
- CREATE INDEX CONCURRENTLY for non-blocking index creation
- ALTER TABLE ... ADD CONSTRAINT ... NOT VALID for online constraint addition
- ALTER TABLE ... VALIDATE CONSTRAINT for validating constraints after addition
- Proper transaction handling - some operations must run outside transactions

**pgschema Directives**:
- Special SQL comments control behavior: `--pgschema-lock-timeout`, `--pgschema-no-transaction`
- Handled in `cmd/apply/directive.go`

**Test Filtering**:
- `PGSCHEMA_TEST_FILTER` environment variable supports both directory paths (`create_table/`) and specific tests (`create_table/add_column`)
- Tests fail if filter pattern doesn't match any tests (prevents silent no-ops)

### Key Files to Know

**Entry Point & CLI**:
- `main.go` - Entry point, loads .env file and calls cmd.Execute()
- `cmd/root.go` - Root CLI command setup with global flags (--debug, --host, --port, --user, --db, --schema)

**IR Package** (separate Go module at `./ir`):
- `ir/ir.go` - Core IR data structures for all schema objects (Table, Column, Index, Trigger, View, Function, Procedure, Sequence, Type, Policy, Aggregate, Constraint)
- `ir/parser.go` - SQL DDL parsing using pg_query_go
- `ir/inspector.go` - Database introspection queries using pgx
- `ir/normalizer.go` - Schema normalization for consistent comparison
- `ir/quote.go` - Identifier quoting and formatting utilities
- `ir/ir_integration_test.go` - Integration tests for IR package

**Diff Package**:
- `internal/diff/diff.go` - Main diff logic and orchestration, topological sorting
- `internal/diff/table.go` - Table-specific diff operations (columns, constraints, modifications)
- `internal/diff/index.go` - Index diff operations
- `internal/diff/trigger.go` - Trigger diff operations
- `internal/diff/view.go` - View and materialized view diff operations
- `internal/diff/function.go` - Function diff operations
- `internal/diff/procedure.go` - Procedure diff operations
- `internal/diff/sequence.go` - Sequence diff operations
- `internal/diff/type.go` - Type diff operations
- `internal/diff/policy.go` - RLS policy diff operations
- `internal/diff/aggregate.go` - Aggregate diff operations

**Other Key Internal Packages**:
- `internal/plan/plan.go` - Migration plan structures and execution logic
- `internal/dump/dump.go` - Schema dump formatting and output
- `internal/fingerprint/fingerprint.go` - Schema fingerprinting for concurrent change detection
- `internal/include/include.go` - Include file processing for modular schemas

**Testing**:
- `cmd/migrate_integration_test.go` - Main integration test suite (TestPlanAndApply)
- `cmd/dump/dump_integration_test.go` - Dump command integration tests
- `cmd/plan/plan_integration_test.go` - Plan command integration tests
- `cmd/apply/apply_integration_test.go` - Apply command integration tests
- `testdata/diff/` - Extensive test cases for all schema object types (100+ test cases)
