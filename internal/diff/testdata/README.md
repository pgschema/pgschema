# Diff Test Data

This directory contains file-based test cases for the diff functionality. Each test case is organized by SQL statement type and variation.

## Structure

```
testdata/
├── create_table/          # CREATE TABLE related tests
│   ├── basic_table/       # Simple table creation
│   ├── add_column/        # Adding columns to existing table
│   ├── drop_column/       # Dropping columns from table
│   ├── modify_column/     # Changing column types/properties
│   ├── with_primary_key/  # Tables with PRIMARY KEY constraints
│   ├── with_foreign_key/  # Tables with FOREIGN KEY constraints
│   ├── with_unique_constraint/  # Tables with UNIQUE constraints
│   ├── with_check_constraint/   # Tables with CHECK constraints
│   ├── with_defaults/     # Tables with DEFAULT values
│   ├── with_not_null/     # Tables with NOT NULL constraints
│   ├── partitioned_table/ # Partitioned tables
│   ├── multi_tenancy/     # Multi-tenant schema patterns with tenant_id
│   ├── array_columns/     # Tables with array column types
│   └── complex_constraints/ # Complex CHECK constraints with regex validation
├── alter_table/           # ALTER TABLE related tests
│   ├── add_column_with_fk/ # Adding columns with foreign key constraints
│   ├── change_column_type/ # Type evolution (int→bigint, etc.)
│   └── add_constraint/    # Adding UNIQUE and CHECK constraints to existing tables
├── create_extension/      # CREATE EXTENSION related tests
│   └── add_extension/     # Adding PostgreSQL extensions (citext, hstore, pg_trgm)
├── create_type/           # CREATE TYPE related tests
│   └── enum_type/         # Creating and using ENUM types
├── create_function/       # CREATE FUNCTION related tests
│   └── plpgsql_function/  # PL/pgSQL functions with dynamic SQL
├── create_view/           # CREATE VIEW related tests
│   └── complex_view/      # Views with JOINs and COALESCE patterns
├── drop_objects/          # DROP related tests
│   ├── cascade_drop/      # DROP TABLE CASCADE scenarios
│   └── conditional_drop/  # IF EXISTS patterns for safe dropping
├── create_index/          # CREATE INDEX related tests (future)
└── drop_table/            # DROP TABLE related tests (future)
```

## Test Case Format

Each test case directory contains exactly three files:

- **old.sql**: The initial DDL state (can be empty for new objects)
- **new.sql**: The target DDL state  
- **migration.sql**: The expected migration output from `Diff(old.sql, new.sql).GenerateMigrationSQL()`

## Running Tests

The tests are automatically discovered and run by the `TestDiffFromFiles` function in `diff_test.go`.

```bash
# Run all diff tests
go test ./internal/diff -v

# Run only file-based tests
go test ./internal/diff -v -run TestDiffFromFiles
```

## Adding New Test Cases

1. Create a new directory under the appropriate statement type (e.g., `create_table/new_feature/`)
2. Add the three required files: `old.sql`, `new.sql`, `migration.sql`
3. Run the tests to verify they pass

## Current Status

### Implemented Features
- ✅ Table creation/deletion
- ✅ Column addition/deletion/modification
- ✅ DEFAULT value changes
- ✅ NOT NULL constraint changes
- ✅ Data type changes

### Test Coverage (Sourcegraph-Inspired Patterns)
- ✅ Multi-tenancy patterns (tenant_id column addition)
- ✅ Type evolution migrations (integer → bigint)
- ✅ Array column type modifications
- ✅ Complex CHECK constraints with regex validation
- ✅ Foreign key constraints with referential actions
- ✅ UNIQUE constraints with multi-column support
- ✅ CASCADE drops for dependency cleanup
- ✅ Conditional drops with IF EXISTS

### Test Cases Created (Framework Ready)
- 📋 PL/pgSQL functions with dynamic SQL
- 📋 Complex views with JOINs and COALESCE
- 📋 PostgreSQL extensions (citext, hstore, pg_trgm)
- 📋 ENUM type creation and usage

### Not Yet Implemented
- ❌ Primary key constraint differences
- ❌ Partitioning differences
- ❌ Index differences
- ❌ Trigger differences
- ❌ Sequence differences
- ❌ Policy differences (RLS)
- ❌ View differences
- ❌ Function differences
- ❌ Type differences
- ❌ Extension differences

Test cases for unimplemented features have empty `migration.sql` files to reflect the current behavior.