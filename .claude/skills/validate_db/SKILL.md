---
name: Validate with Database
description: Connect to live PostgreSQL database to validate schema assumptions, compare pg_dump vs pgschema output, and query system catalogs interactively
---

# Validate with Database

Use this skill to connect to the test PostgreSQL database, validate assumptions about schema behavior, and cross-validate between pg_dump and pgschema implementations.

## When to Use This Skill

Invoke this skill when:
- Validating how PostgreSQL actually stores or represents schema objects
- Comparing pg_dump output with pgschema output
- Testing a new feature implementation against real database
- Debugging schema introspection issues
- Verifying system catalog query results
- Understanding how PostgreSQL formats specific DDL
- Checking version-specific behavior (PostgreSQL 14-18)
- Validating migration plans before implementing new features

## Database Connection Information

Connection details are stored in `.env` file at project root:

```
PGHOST=localhost
PGDATABASE=employee
PGUSER=postgres
PGPASSWORD=testpwd1
```

**Default connection**:
- Host: `localhost`
- Port: `5432` (default)
- Database: `employee`
- User: `postgres`
- Password: `testpwd1`

## Connection Methods

### Method 1: Using psql (Interactive Queries)

**Basic connection**:
```bash
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d employee
```

**One-off query**:
```bash
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d employee -c "SELECT version();"
```

**Execute multi-line query**:
```bash
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d postgres -c "
SELECT
    t.tgname,
    CASE
        WHEN t.tgqual IS NOT NULL
        THEN pg_get_expr(t.tgqual, t.tgrelid, false)
        ELSE 'NO WHEN CLAUSE'
    END as when_clause
FROM pg_catalog.pg_trigger t
JOIN pg_catalog.pg_class c ON t.tgrelid = c.oid
WHERE c.relname = 'test_table'
ORDER BY t.tgname;
"
```

### Method 2: Using pg_dump (Schema Export)

**Dump entire database schema**:
```bash
PGPASSWORD='testpwd1' pg_dump -h localhost -p 5432 -U postgres -d employee --schema-only --schema=public
```

**Dump specific table**:
```bash
PGPASSWORD='testpwd1' pg_dump -h localhost -p 5432 -U postgres -d employee --schema-only --table=employees
```

**Dump only specific object types**:
```bash
# Only triggers
PGPASSWORD='testpwd1' pg_dump -h localhost -p 5432 -U postgres -d employee --schema-only --schema=public | grep -A 20 "CREATE TRIGGER"

# Only indexes
PGPASSWORD='testpwd1' pg_dump -h localhost -p 5432 -U postgres -d employee --schema-only --schema=public | grep -A 10 "CREATE INDEX"
```

### Method 3: Using pgschema (Project Tool)

**Dump with pgschema**:
```bash
./pgschema dump --host localhost --port 5432 --db employee --user postgres --schema public
```

**Or using environment variables** (from .env):
```bash
# .env is automatically loaded by pgschema
./pgschema dump --schema public
```

**Dump to file**:
```bash
./pgschema dump --schema public -o /tmp/schema_dump.sql
```

### Method 4: Database Setup for Testing

**Create a test database**:
```bash
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -c "DROP DATABASE IF EXISTS test_validation;"
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -c "CREATE DATABASE test_validation;"
```

**Create test schema objects**:
```bash
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d test_validation -c "
CREATE TABLE test_table (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER test_trigger
    BEFORE INSERT ON test_table
    FOR EACH ROW
    WHEN (NEW.name IS NOT NULL)
    EXECUTE FUNCTION my_trigger_func();
"
```

## Common Validation Workflows

### Workflow 1: Compare pg_dump vs pgschema Output

**Purpose**: Verify pgschema produces comparable output to pg_dump

**Steps**:

1. **Dump with pg_dump**:
```bash
PGPASSWORD='testpwd1' pg_dump -h localhost -p 5432 -U postgres -d employee --schema-only --schema=public > /tmp/pg_dump_output.sql
```

2. **Dump with pgschema**:
```bash
./pgschema dump --schema public -o /tmp/pgschema_output.sql
```

3. **Compare outputs**:
```bash
# Side-by-side comparison
diff -u /tmp/pg_dump_output.sql /tmp/pgschema_output.sql

# Or use a better diff tool
code --diff /tmp/pg_dump_output.sql /tmp/pgschema_output.sql
```

4. **Analyze differences**:
- Formatting differences (expected)
- Missing objects (bugs to fix)
- Different DDL structure (may need investigation)
- Comments handling
- Ordering differences

### Workflow 2: Validate System Catalog Queries

**Purpose**: Test system catalog queries return expected data

**Steps**:

1. **Create test object**:
```bash
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d postgres -c "
CREATE TABLE test_triggers (
    id INTEGER PRIMARY KEY,
    data TEXT
);

CREATE OR REPLACE FUNCTION trigger_func() RETURNS TRIGGER AS \$\$
BEGIN
    RETURN NEW;
END;
\$\$ LANGUAGE plpgsql;

CREATE TRIGGER test_when_trigger
    BEFORE INSERT ON test_triggers
    FOR EACH ROW
    WHEN (NEW.data <> '')
    EXECUTE FUNCTION trigger_func();
"
```

2. **Query system catalogs**:
```bash
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d postgres -c "
SELECT
    t.tgname,
    t.tgtype,
    CASE
        WHEN t.tgqual IS NOT NULL
        THEN pg_get_expr(t.tgqual, t.tgrelid, false)
        ELSE NULL
    END as when_clause,
    pg_get_triggerdef(t.oid) as full_definition
FROM pg_catalog.pg_trigger t
JOIN pg_catalog.pg_class c ON t.tgrelid = c.oid
WHERE c.relname = 'test_triggers'
  AND t.tgisinternal = false;
"
```

3. **Verify pgschema extracts same data**:
```bash
./pgschema dump --schema public | grep -A 20 "test_when_trigger"
```

4. **Cleanup**:
```bash
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d postgres -c "
DROP TRIGGER IF EXISTS test_when_trigger ON test_triggers;
DROP TABLE IF EXISTS test_triggers;
DROP FUNCTION IF EXISTS trigger_func();
"
```

### Workflow 3: Test Plan/Apply Workflow

**Purpose**: Validate pgschema plan and apply work correctly

**Steps**:

1. **Create initial schema**:
```bash
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d postgres -c "
DROP SCHEMA IF EXISTS test_workflow CASCADE;
CREATE SCHEMA test_workflow;
SET search_path TO test_workflow;

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE
);
"
```

2. **Dump current state**:
```bash
./pgschema dump --schema test_workflow -o /tmp/current_schema.sql
```

3. **Modify schema file** (edit /tmp/current_schema.sql):
```sql
-- Add a new column
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP  -- NEW
);
```

4. **Generate plan**:
```bash
./pgschema plan --schema test_workflow --file /tmp/current_schema.sql
```

5. **Review migration DDL** - should show:
```sql
ALTER TABLE users ADD COLUMN created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
```

6. **Apply migration**:
```bash
./pgschema apply --schema test_workflow --file /tmp/current_schema.sql --auto-approve
```

7. **Verify result**:
```bash
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d postgres -c "\d test_workflow.users"
```

8. **Cleanup**:
```bash
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d postgres -c "DROP SCHEMA IF EXISTS test_workflow CASCADE;"
```

### Workflow 4: Validate Specific DDL Formatting

**Purpose**: Understand how PostgreSQL formats specific constructs

**Steps**:

1. **Create object with specific feature**:
```bash
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d postgres -c "
CREATE TABLE test_pk_order (
    b INTEGER,
    a INTEGER,
    c INTEGER,
    PRIMARY KEY (a, b)  -- Note: different order than column definition
);
"
```

2. **Check how PostgreSQL stores it**:
```bash
# Use \d+ to see structure
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d postgres -c "\d+ test_pk_order"
```

3. **See pg_dump format**:
```bash
PGPASSWORD='testpwd1' pg_dump -h localhost -p 5432 -U postgres -d postgres --schema-only --table=test_pk_order
```

4. **Query system catalogs directly**:
```bash
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d postgres -c "
SELECT
    c.relname as table_name,
    con.conname as constraint_name,
    pg_get_constraintdef(con.oid) as constraint_def
FROM pg_constraint con
JOIN pg_class c ON con.conrelid = c.oid
WHERE c.relname = 'test_pk_order';
"
```

5. **Compare with pgschema**:
```bash
./pgschema dump --schema public | grep -A 10 "test_pk_order"
```

### Workflow 5: Cross-Version Testing

**Purpose**: Validate behavior across PostgreSQL versions 14-18

**Steps**:

1. **Check current version**:
```bash
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d postgres -c "SELECT version();"
```

2. **Run version-specific integration tests**:
```bash
# Test against specific version
PGSCHEMA_POSTGRES_VERSION=14 go test -v ./cmd/dump -run TestDumpCommand_Employee
PGSCHEMA_POSTGRES_VERSION=17 go test -v ./cmd/dump -run TestDumpCommand_Employee
```

3. **Check for version-specific features**:
```bash
# PostgreSQL 15+ feature: UNIQUE NULLS NOT DISTINCT
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d postgres -c "
SELECT version();
CREATE TABLE test_nulls (
    id INTEGER,
    email TEXT UNIQUE NULLS NOT DISTINCT
);
"
```

## Useful System Catalog Queries

### Inspect Tables and Columns

```sql
-- All tables in schema
SELECT schemaname, tablename
FROM pg_tables
WHERE schemaname = 'public';

-- Columns with types
SELECT
    a.attname as column_name,
    pg_catalog.format_type(a.atttypid, a.atttypmod) as data_type,
    a.attnotnull as not_null,
    pg_get_expr(ad.adbin, ad.adrelid) as default_value,
    a.attgenerated as generated
FROM pg_attribute a
LEFT JOIN pg_attrdef ad ON (a.attrelid = ad.adrelid AND a.attnum = ad.adnum)
WHERE a.attrelid = 'public.employees'::regclass
  AND a.attnum > 0
  AND NOT a.attisdropped
ORDER BY a.attnum;
```

### Inspect Constraints

```sql
-- All constraints on a table
SELECT
    con.conname as constraint_name,
    con.contype as constraint_type,
    pg_get_constraintdef(con.oid) as definition
FROM pg_constraint con
WHERE con.conrelid = 'public.employees'::regclass;
```

### Inspect Indexes

```sql
-- All indexes on a table
SELECT
    i.relname as index_name,
    am.amname as index_type,
    pg_get_indexdef(idx.indexrelid) as definition,
    CASE
        WHEN idx.indpred IS NOT NULL
        THEN pg_get_expr(idx.indpred, idx.indrelid, true)
        ELSE NULL
    END as where_clause
FROM pg_index idx
JOIN pg_class i ON i.oid = idx.indexrelid
JOIN pg_class t ON t.oid = idx.indrelid
JOIN pg_am am ON i.relam = am.oid
WHERE t.relname = 'employees'
  AND t.relnamespace = 'public'::regnamespace;
```

### Inspect Triggers

```sql
-- All triggers on a table
SELECT
    t.tgname as trigger_name,
    t.tgenabled as enabled,
    CASE t.tgtype::integer & 66
        WHEN 2 THEN 'BEFORE'
        WHEN 64 THEN 'INSTEAD OF'
        ELSE 'AFTER'
    END as timing,
    pg_get_triggerdef(t.oid) as full_definition,
    CASE
        WHEN t.tgqual IS NOT NULL
        THEN pg_get_expr(t.tgqual, t.tgrelid, false)
        ELSE NULL
    END as when_condition
FROM pg_trigger t
JOIN pg_class c ON t.tgrelid = c.oid
WHERE c.relname = 'employees'
  AND c.relnamespace = 'public'::regnamespace
  AND NOT t.tgisinternal;
```

### Inspect Views and Materialized Views

```sql
-- Views
SELECT
    schemaname,
    viewname,
    definition
FROM pg_views
WHERE schemaname = 'public';

-- Materialized views
SELECT
    schemaname,
    matviewname,
    definition
FROM pg_matviews
WHERE schemaname = 'public';
```

### Inspect Comments

```sql
-- Comments on tables
SELECT
    c.relname as table_name,
    d.description as comment
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
LEFT JOIN pg_description d ON d.objoid = c.oid AND d.objsubid = 0
WHERE n.nspname = 'public'
  AND c.relkind = 'r'
  AND d.description IS NOT NULL;

-- Comments on columns
SELECT
    c.relname as table_name,
    a.attname as column_name,
    d.description as comment
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
JOIN pg_attribute a ON a.attrelid = c.oid
LEFT JOIN pg_description d ON d.objoid = c.oid AND d.objsubid = a.attnum
WHERE n.nspname = 'public'
  AND c.relkind = 'r'
  AND a.attnum > 0
  AND NOT a.attisdropped
  AND d.description IS NOT NULL;
```

## Troubleshooting Common Issues

### Issue: Connection Refused

```bash
# Check if PostgreSQL is running
pg_isready -h localhost -p 5432

# Check if port is open
nc -zv localhost 5432

# Check PostgreSQL logs
tail -f /usr/local/var/postgresql@14/server.log  # macOS Homebrew
```

### Issue: Authentication Failed

```bash
# Verify credentials
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -c "SELECT current_user;"

# Check pg_hba.conf settings
cat /usr/local/var/postgresql@14/pg_hba.conf
```

### Issue: Database Doesn't Exist

```bash
# List all databases
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -c "\l"

# Create if missing
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -c "CREATE DATABASE employee;"
```

### Issue: Schema Not Found

```bash
# List all schemas
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d employee -c "\dn"

# Create if missing
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d employee -c "CREATE SCHEMA public;"
```

## Test Data Setup

### Load Sample Database

The project includes test data in `testdata/dump/`:

```bash
# Employee database
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -c "DROP DATABASE IF EXISTS employee;"
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -c "CREATE DATABASE employee;"
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d employee < testdata/dump/employee/employee.sql

# Sakila database (if available)
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -c "DROP DATABASE IF EXISTS sakila;"
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -c "CREATE DATABASE sakila;"
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d sakila < testdata/dump/sakila/sakila.sql
```

### Create Minimal Test Database

```bash
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres <<EOF
DROP DATABASE IF EXISTS test_minimal;
CREATE DATABASE test_minimal;
\c test_minimal

CREATE SCHEMA app;

CREATE TABLE app.users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE app.posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES app.users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    content TEXT,
    published_at TIMESTAMP
);

CREATE INDEX idx_posts_user ON app.posts(user_id);
CREATE INDEX idx_posts_published ON app.posts(published_at) WHERE published_at IS NOT NULL;

COMMENT ON TABLE app.users IS 'Application users';
COMMENT ON COLUMN app.users.email IS 'User email address (must be unique)';
EOF
```

## Quick Reference Commands

**Connect to database**:
```bash
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d employee
```

**Run query**:
```bash
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d employee -c "SELECT * FROM employees LIMIT 5;"
```

**Describe table**:
```bash
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -d employee -c "\d+ employees"
```

**pg_dump schema only**:
```bash
PGPASSWORD='testpwd1' pg_dump -h localhost -p 5432 -U postgres -d employee --schema-only --schema=public
```

**pgschema dump**:
```bash
./pgschema dump --schema public
```

**pgschema plan**:
```bash
./pgschema plan --schema public --file schema.sql
```

**Drop and recreate test database**:
```bash
PGPASSWORD='testpwd1' psql -h localhost -p 5432 -U postgres -c "DROP DATABASE IF EXISTS test_db; CREATE DATABASE test_db;"
```

## Validation Checklist

When validating implementation:

- [ ] Test database is running and accessible
- [ ] Connection credentials from .env work
- [ ] pg_dump produces expected output
- [ ] pgschema produces comparable output
- [ ] System catalog queries return expected data
- [ ] DDL formatting matches PostgreSQL conventions
- [ ] Plan generates correct migration DDL
- [ ] Apply successfully executes migration
- [ ] Final state matches expected schema
- [ ] Tested across PostgreSQL versions 14-18 (if version-specific)
- [ ] Test database cleaned up after validation
