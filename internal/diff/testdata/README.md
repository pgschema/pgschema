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
│   └── partitioned_table/ # Partitioned tables
├── alter_table/           # ALTER TABLE related tests (future)
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

### Not Yet Implemented
- ❌ Primary key constraint differences
- ❌ Foreign key constraint differences
- ❌ Unique constraint differences
- ❌ Check constraint differences
- ❌ Partitioning differences
- ❌ Index differences
- ❌ Trigger differences

Test cases for unimplemented features have empty `migration.sql` files to reflect the current behavior.