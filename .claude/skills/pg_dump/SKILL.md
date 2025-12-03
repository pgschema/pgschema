---
name: pg_dump Reference
description: Consult PostgreSQL's pg_dump implementation for guidance on system catalog queries and schema extraction when implementing pgschema features
---

# pg_dump Reference

Use this skill when implementing or debugging pgschema features that involve extracting schema information from PostgreSQL databases. pg_dump is the canonical PostgreSQL schema dumping tool and serves as a reference implementation for how to query system catalogs correctly.

## When to Use This Skill

Invoke this skill when:
- Adding support for new PostgreSQL schema objects
- Debugging system catalog queries in `ir/inspector.go`
- Understanding how PostgreSQL represents objects internally
- Handling version-specific PostgreSQL features (versions 14-18)
- Learning correct DDL formatting patterns
- Understanding object dependency relationships

## Source Code Locations

**Main pg_dump repository**: https://github.com/postgres/postgres/blob/master/src/bin/pg_dump/

**Key files to reference**:
- `pg_dump.c` - Main implementation with system catalog queries
- `pg_dump.h` - Data structures and function declarations
- `pg_dump_sort.c` - Dependency sorting logic
- `pg_backup_archiver.c` - Output formatting
- `common.c` - Shared utility functions for querying system catalogs

## Step-by-Step Workflow

### 1. Identify the Schema Object

Determine which PostgreSQL object type you're working with:
- Tables and columns
- Constraints (PRIMARY KEY, FOREIGN KEY, UNIQUE, CHECK)
- Indexes (regular, unique, partial, functional)
- Triggers (including WHEN conditions, constraint triggers)
- Views and materialized views
- Functions and procedures
- Sequences
- Types (enum, composite, domain)
- Policies (row-level security)
- Aggregates
- Comments

### 2. Find the Relevant pg_dump Function

Search pg_dump.c for the function that handles your object type:

| Object Type | pg_dump Function | System Catalogs Used |
|-------------|------------------|---------------------|
| Tables & Columns | `getTables()` | `pg_class`, `pg_attribute`, `pg_type` |
| Indexes | `getIndexes()` | `pg_index`, `pg_class` |
| Triggers | `getTriggers()` | `pg_trigger`, `pg_proc` |
| Functions | `getFuncs()` | `pg_proc` |
| Procedures | `getProcs()` | `pg_proc` |
| Views | `getViews()` | `pg_class`, `pg_rewrite` |
| Materialized Views | `getMatViews()` | `pg_class` |
| Sequences | `getSequences()` | `pg_sequence`, `pg_class` |
| Constraints | `getConstraints()` | `pg_constraint` |
| Policies | `getPolicies()` | `pg_policy` |
| Aggregates | `getAggregates()` | `pg_aggregate`, `pg_proc` |
| Types | `getTypes()` | `pg_type` |
| Comments | `getComments()` | `pg_description` |

### 3. Analyze the System Catalog Query

Examine the SQL query used by pg_dump:
- Which system catalog tables are joined
- Which columns are selected
- How version-specific features are handled
- How PostgreSQL internal functions are used (`pg_get_expr`, `pg_get_constraintdef`, etc.)

**Example - Extracting trigger WHEN conditions**:

```sql
-- pg_dump's approach (from getTriggers):
SELECT t.tgname,
       pg_get_expr(t.tgqual, t.tgrelid, false) as when_clause
FROM pg_catalog.pg_trigger t
WHERE t.tgqual IS NOT NULL
```

Note: `information_schema.triggers.action_condition` is NOT reliable for WHEN clauses. Always use `pg_get_expr(t.tgqual, ...)` from `pg_catalog.pg_trigger`.

### 4. Check for Special Cases

Look for how pg_dump handles:
- **Version compatibility**: Different queries for different PostgreSQL versions
- **NULL handling**: How missing values are interpreted
- **Default values**: System vs. user-defined defaults
- **Internal objects**: Filtering out system-generated objects
- **Dependencies**: How object relationships are tracked

### 5. Adapt for pgschema

Apply the pattern to pgschema's codebase:

**For database introspection** (`ir/inspector.go`):
- Adapt the system catalog query for Go/pgx
- Use pgx parameter binding for safety
- Handle NULL values appropriately
- Add proper error handling

**For SQL parsing** (`ir/parser.go`):
- Understand how pg_dump formats DDL
- Use pg_query_go to parse SQL statements
- Extract relevant fields into IR structures

**For DDL generation** (`internal/diff/*.go`):
- Follow pg_dump's quoting rules
- Use PostgreSQL functions for formatting complex expressions
- Handle version-specific syntax

## Key System Catalog Tables

### Core Tables
- `pg_class` - Tables, indexes, views, sequences
- `pg_attribute` - Table columns
- `pg_type` - Data types
- `pg_constraint` - Constraints (PK, FK, UNIQUE, CHECK)
- `pg_index` - Index definitions

### Functions & Triggers
- `pg_proc` - Functions, procedures, trigger functions
- `pg_trigger` - Trigger definitions
- `pg_aggregate` - Aggregate function definitions

### Access Control
- `pg_policy` - Row-level security policies

### Metadata
- `pg_description` - Comments on database objects
- `pg_depend` - Object dependencies

### Helper Functions
- `pg_get_expr(expr, relation, pretty)` - Deparse expressions
- `pg_get_constraintdef(constraint_oid, pretty)` - Get constraint definition
- `pg_get_indexdef(index_oid, column, pretty)` - Get index definition
- `pg_get_triggerdef(trigger_oid, pretty)` - Get trigger definition

## Important Considerations

### pgschema is NOT pg_dump

**Key differences**:
- **Format**: pgschema outputs declarative schema files for editing, pg_dump creates archive dumps for restore
- **Scope**: pgschema focuses on single-schema objects, pg_dump handles entire databases
- **Workflow**: pgschema supports plan/apply (Terraform-style), pg_dump is dump/restore only
- **Normalization**: pgschema normalizes for comparison, pg_dump preserves exact format

### When NOT to Copy pg_dump

Don't blindly copy pg_dump for:
- Output formatting (pgschema has different conventions)
- Archive/restore logic (not applicable to pgschema)
- Full database dumps (pgschema is schema-focused)
- Ancient version support (pgschema supports PostgreSQL 14+)

### When pg_dump is Authoritative

Always reference pg_dump for:
- System catalog query patterns
- Understanding PostgreSQL internals
- Correct use of `pg_get_*` functions
- Version-specific feature detection
- Object dependency tracking

## Examples

### Example 1: Extracting Generated Column Information

**pg_dump approach**:
```sql
SELECT a.attname,
       a.attgenerated,
       pg_get_expr(ad.adbin, ad.adrelid) as generation_expr
FROM pg_attribute a
LEFT JOIN pg_attrdef ad ON (a.attrelid = ad.adrelid AND a.attnum = ad.adnum)
WHERE a.attgenerated != ''
```

**pgschema adaptation** (in `ir/inspector.go`):
```go
query := `
SELECT a.attname,
       a.attgenerated,
       pg_get_expr(ad.adbin, ad.adrelid) as generation_expr
FROM pg_attribute a
LEFT JOIN pg_attrdef ad ON (a.attrelid = ad.adrelid AND a.attnum = ad.adnum)
WHERE a.attrelid = $1 AND a.attgenerated != ''
`
rows, err := conn.Query(ctx, query, tableOID)
```

### Example 2: Handling Partial Indexes

**pg_dump extracts WHERE clauses**:
```sql
SELECT pg_get_expr(i.indpred, i.indrelid, true) as index_predicate
FROM pg_index i
WHERE i.indpred IS NOT NULL
```

**pgschema stores in IR** (`ir/ir.go`):
```go
type Index struct {
    Name      string
    Columns   []string
    Predicate string  // WHERE clause for partial indexes
    // ...
}
```

## Tips for Success

1. **Search strategically**: Clone postgres repo and use grep/ag to search for specific system catalog columns or keywords

2. **Check git history**: Use `git log -p` or GitHub blame to see when features were added and understand the evolution

3. **Read comments carefully**: pg_dump.c contains valuable comments explaining PostgreSQL internals and edge cases

4. **Cross-reference documentation**: Always combine pg_dump source with official PostgreSQL documentation:
   - System catalogs: https://www.postgresql.org/docs/current/catalogs.html
   - Functions: https://www.postgresql.org/docs/current/functions-info.html

5. **Test incrementally**: After adapting from pg_dump, test against real PostgreSQL instances using pgschema's embedded-postgres integration tests

6. **Version awareness**: Check how pg_dump handles version differences - pgschema supports PostgreSQL 14-18, so you may need conditional logic

## Verification Checklist

After consulting pg_dump and implementing in pgschema:

- [ ] System catalog query correctly extracts all necessary fields
- [ ] NULL values are handled appropriately
- [ ] Version-specific features are detected and handled
- [ ] Internal/system objects are filtered out
- [ ] Dependencies are tracked correctly
- [ ] Integration test added in `testdata/diff/`
- [ ] Test passes with `go test -v ./internal/diff -run TestDiffFromFiles`
- [ ] Test passes with `go test -v ./cmd -run TestPlanAndApply`
- [ ] Tested against multiple PostgreSQL versions (14-18)
