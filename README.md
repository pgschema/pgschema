# pgschema

A CLI tool to dump and diff PostgreSQL schemas. It provides comprehensive schema extraction with output compatible with `pg_dump`.

## Installation

### Production

Download the latest binary from the releases page or build from source:

```bash
go install github.com/pgschema/pgschema@latest
```

Or build locally:

```bash
git clone https://github.com/pgschema/pgschema.git
cd pgschema
go build -o pgschema .
```

### Development

1. Clone the repository:
```bash
git clone https://github.com/pgschema/pgschema.git
cd pgschema
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build the binary:
```bash
go build -o pgschema .
```

4. Run tests:
```bash
# Run unit tests only
go test -short -v ./...

# Run all tests including integration tests (uses testcontainers with Docker)
go test -v ./...
```

## Usage

The `pgschema` tool provides commands to work with PostgreSQL schemas.

### Commands

#### Dump Command

Dump and output database schema information in pg_dump compatible format:

```bash
pgschema dump --host hostname -p 5432 -d database -U user
```

For password authentication, use the `PGPASSWORD` environment variable:

```bash
PGPASSWORD=password pgschema dump --host hostname -d database -U user
```

#### Plan Command

Generate migration plans by comparing two schema sources (databases or schema files):

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

#### Version Command

Display version information:

```bash
pgschema version
```

### Flags

#### Inspect Command Flags

- `--host string`: Database server host (default: localhost)
- `-p, --port int`: Database server port (default: 5432)
- `-d, --db string`: Database name (required)
- `-U, --user string`: Database user name (required)

#### Plan Command Flags

**Source 1 (Database Connection):**
- `--host1 string`: Database server host for source 1 (default: localhost)
- `--port1 int`: Database server port for source 1 (default: 5432)
- `--db1 string`: Database name for source 1
- `--user1 string`: Database user name for source 1
- `--schema1 string`: Schema name for source 1 (optional filter)

**Source 1 (Schema File):**
- `--file1 string`: Path to first SQL schema file

**Source 2 (Database Connection):**
- `--host2 string`: Database server host for source 2 (default: localhost)
- `--port2 int`: Database server port for source 2 (default: 5432)
- `--db2 string`: Database name for source 2
- `--user2 string`: Database user name for source 2
- `--schema2 string`: Schema name for source 2 (optional filter)

**Source 2 (Schema File):**
- `--file2 string`: Path to second SQL schema file

**Output Options:**
- `--format string`: Output format: text, json, preview (default: text)

**Global Flags:**
- `--debug`: Enable debug logging

### Examples

#### Dump a database schema

```bash
# Dump and output schema in pg_dump format
pgschema dump --host localhost -p 5432 -d mydb -U myuser

# With password authentication
PGPASSWORD=mypassword pgschema dump --host localhost -d mydb -U myuser

# Using custom host and port
pgschema dump --host db.example.com -p 5433 -d mydb -U myuser

# Save schema to file
PGPASSWORD=mypassword pgschema dump --host localhost -d mydb -U myuser > schema.sql
```

#### Generate migration plans

```bash
# Compare two schema files
pgschema plan --file1 current_schema.sql --file2 target_schema.sql

# Compare two schema files with JSON output
pgschema plan --file1 v1.sql --file2 v2.sql --format json

# Compare database to schema file
pgschema plan --db1 production_db --user1 readonly_user --file2 target_schema.sql

# Compare two databases
pgschema plan --db1 staging_db --user1 user1 --db2 production_db --user2 user2

# Compare specific schemas in databases
pgschema plan --db1 db1 --user1 user1 --schema1 public --db2 db2 --user2 user2 --schema2 app_schema

# Compare databases with different hosts and ports
pgschema plan \
  --host1 staging.example.com --port1 5432 --db1 myapp --user1 user1 \
  --host2 prod.example.com --port2 5433 --db2 myapp --user2 user2

# Preview format for detailed migration plan
pgschema plan --file1 old.sql --file2 new.sql --format preview
```

### Connection Options

#### Database Connections

Both `dump` and `plan` commands use psql-style connection parameters:
- `--host`: Database server host (default: localhost)
- `-p, --port`: Database server port (default: 5432) 
- `-d, --db`: Database name
- `-U, --user`: Database user name

Password authentication is handled via the `PGPASSWORD` environment variable:

```bash
export PGPASSWORD=your_password
pgschema dump --host hostname -d database -U user
```

#### Plan Command Input Validation

The plan command enforces strict input validation:
- Each source (1 and 2) must specify **either** a database connection **or** a schema file, but not both
- For database connections, both `--db` and `--user` are required
- Schema filtering (`--schema1`, `--schema2`) is optional and only applies to database connections

## Output

### Dump Command

The `dump` command outputs PostgreSQL schema in pg_dump compatible format:

```sql
--
-- PostgreSQL database dump
--

-- Dumped from database version 17.2
-- Dumped by pgschema

CREATE SCHEMA analytics;

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE
);

-- More DDL statements...
```

### Plan Command

The `plan` command shows migration plans in different formats:

**Text format (default):**
```
Plan: 1 to add, 1 to change, 0 to destroy.

Resources to be created:
  + table public.posts

Resources to be modified:
  ~ table public.users
```

**JSON format:**
```json
{
  "diff": {
    "AddedTables": [...],
    "ModifiedTables": [...],
    "DroppedTables": [...]
  },
  "created_at": "2024-01-01T12:00:00Z"
}
```

**Preview format:**
```
Migration Plan (created at 2024-01-01T12:00:00Z)
==================================================

Plan: 1 to add, 1 to change, 0 to destroy.

Resources to be created:
  + table public.posts

Resources to be modified:
  ~ table public.users
```

## Requirements

- Go 1.19 or later
- PostgreSQL 14, 15, 16, 17 (for runtime usage)
- Docker (for running integration tests with testcontainers) 