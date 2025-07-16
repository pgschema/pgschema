# Dump Command

This directory contains the implementation of the `pgschema dump` command, which dumps PostgreSQL database schemas in a developer-friendly format.

## Key Features

- **Dependency Resolution**: Topological sorting ensures objects are created in the correct order
- **Schema Qualification**: Automatic schema prefixing for cross-schema references  
- **Referential Actions**: Full support for ON DELETE/UPDATE clauses in foreign keys
- **Developer Friendly**: More terse and readable than raw pg_dump output

## Running Tests

```bash
# All dump tests
go test -v ./cmd/dump/

# Specific dump tests
go test -v ./cmd/dump/ -run "TestDumpCommand_Employee"
```

## Usage

```bash
# Basic usage
pgschema dump --host localhost --db mydb --user myuser

# Dump specific schema
pgschema dump --host localhost --db mydb --user myuser --schema myschema

# With password
pgschema dump --host localhost --db mydb --user myuser --password mypass

# Using environment variable for password
PGPASSWORD=mypass pgschema dump --host localhost --db mydb --user myuser
```

## Architecture

The dump command uses:
- **SQLC** for type-safe SQL query generation
- **pgx/v5** for PostgreSQL connectivity
- **Testcontainers** for integration testing
