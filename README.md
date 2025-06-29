# pgschema

A CLI tool to compare PostgreSQL schemas from directories or databases.

> **Note**: Starting from version X.X.X, the `inspect` command has switched from using a single `--dsn` flag to individual psql-style connection parameters (`--host`, `-p`, `-d`, `-U`). This change aligns with standard PostgreSQL tooling conventions.

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

#### Inspect Command

Inspect and output database schema information in pg_dump compatible format:

```bash
pgschema inspect --host hostname -p 5432 -d database -U username
```

For password authentication, use the `PGPASSWORD` environment variable:

```bash
PGPASSWORD=password pgschema inspect --host hostname -d database -U username
```

#### Diff Command

Compare two PostgreSQL schemas:

```bash
pgschema diff [flags]
```

### Flags

#### Inspect Command Flags

- `--host string`: Database server host (default: localhost)
- `-p, --port int`: Database server port (default: 5432)
- `-d, --dbname string`: Database name (required)
- `-U, --username string`: Database user name (required)

#### Diff Command Flags

- `--source-dir string`: Source schema directory containing SQL files
- `--source-dsn string`: Source database connection string
- `--target-dir string`: Target schema directory containing SQL files  
- `--target-dsn string`: Target database connection string
- `--temp-db-dsn string`: Temporary database connection string (required when using directory-based schemas)

### Examples

#### Inspect a database schema

```bash
# Inspect and output schema in pg_dump format
pgschema inspect --host localhost -p 5432 -d mydb -U myuser

# With password authentication
PGPASSWORD=mypassword pgschema inspect --host localhost -d mydb -U myuser

# Using custom host and port
pgschema inspect --host db.example.com -p 5433 -d mydb -U myuser

# Save schema to file
PGPASSWORD=mypassword pgschema inspect --host localhost -d mydb -U myuser > schema.sql
```

#### Compare two directories containing SQL schema files

```bash
pgschema diff \
  --source-dir ./schema/v1 \
  --target-dir ./schema/v2 \
  --temp-db-dsn "postgres://user:password@localhost:5432/postgres?sslmode=disable"
```

#### Compare a directory against a live database

```bash
pgschema diff \
  --source-dir ./schema \
  --target-dsn "postgres://user:password@localhost:5432/mydb?sslmode=disable" \
  --temp-db-dsn "postgres://user:password@localhost:5432/postgres?sslmode=disable"
```

#### Compare two databases

```bash
pgschema diff \
  --source-dsn "postgres://user:password@localhost:5432/db1?sslmode=disable" \
  --target-dsn "postgres://user:password@localhost:5432/db2?sslmode=disable"
```

#### Compare a database against a directory

```bash
pgschema diff \
  --source-dsn "postgres://user:password@localhost:5432/mydb?sslmode=disable" \
  --target-dir ./schema/latest \
  --temp-db-dsn "postgres://user:password@localhost:5432/postgres?sslmode=disable"
```

### Connection Options

#### For inspect command

The inspect command uses psql-style connection parameters:
- `--host`: Database server host
- `-p, --port`: Database server port
- `-d, --dbname`: Database name
- `-U, --username`: Database user name

Password authentication is handled via the `PGPASSWORD` environment variable.

#### For diff command

The diff command still uses PostgreSQL connection strings (DSN) in the format:

```
postgres://username:password@hostname:port/database?param1=value1&param2=value2
```

Common parameters:
- `sslmode=disable|require|verify-ca|verify-full`
- `connect_timeout=10`
- `application_name=pgschema`

### Directory Structure

When using `--source-dir` or `--target-dir`, the tool will recursively scan for `.sql` files in the specified directory and combine them to build the schema.

Example directory structure:
```
schema/
├── tables/
│   ├── users.sql
│   └── products.sql
├── indexes/
│   └── user_indexes.sql
└── functions/
    └── helpers.sql
```

### Temporary Database

When comparing directory-based schemas, pgschema requires a temporary database to create and analyze the schema structures. The `--temp-db-dsn` should point to a PostgreSQL instance where temporary databases can be created and dropped. The tool will:

1. Create temporary databases with unique names
2. Apply the schema files to these databases
3. Compare the resulting schemas
4. Clean up the temporary databases

**Important**: The user specified in the temp-db-dsn must have `CREATEDB` privileges.

## Output

The tool will output the SQL statements needed to transform the source schema into the target schema. If no differences are found, it will display "No differences found between schemas".

Example output:
```sql
ALTER TABLE users ADD COLUMN email VARCHAR(255);
CREATE INDEX idx_users_email ON users(email);
DROP TABLE old_table;
```

## Dependencies

This tool uses the [Stripe pg-schema-diff](https://github.com/stripe/pg-schema-diff) library for schema comparison and diff generation.

## Requirements

- Go 1.19 or later
- PostgreSQL 14, 15, 16, 17 (for runtime usage)
- Docker (for running integration tests with testcontainers) 