# IR (Intermediate Representation) Package

[![Go Reference](https://pkg.go.dev/badge/github.com/pgschema/pgschema/ir.svg)](https://pkg.go.dev/github.com/pgschema/pgschema/ir)
[![Go Report Card](https://goreportcard.com/badge/github.com/pgschema/pgschema/ir)](https://goreportcard.com/report/github.com/pgschema/pgschema/ir)

The `ir` package provides an Intermediate Representation for PostgreSQL database schemas. It can be used by external projects to parse SQL files, introspect live databases, and work with normalized schema representations.

## Installation

### Latest Version
```bash
go get github.com/pgschema/pgschema/ir
```

### Specific Version
```bash
go get github.com/pgschema/pgschema/ir@ir/v0.1.0
```

## Usage

### Introspecting Live Database

```go
import (
    "context"
    "database/sql"
    "github.com/pgschema/pgschema/ir"
    _ "github.com/lib/pq"
)

// Connect to database
db, err := sql.Open("postgres", "postgresql://user:pass@localhost/dbname?sslmode=disable")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Create inspector with no ignore config
inspector := ir.NewInspector(db, nil)

// Build IR from database for "public" schema
ctx := context.Background()
schema, err := inspector.BuildIR(ctx, "public")
if err != nil {
    log.Fatal(err)
}

// Access normalized schema data
if publicSchema, ok := schema.GetSchema("public"); ok {
    fmt.Printf("Found %d tables\n", len(publicSchema.Tables))
}
```

### Working with Schema Objects

The IR package provides strongly-typed representations of PostgreSQL objects:

- **Tables**: Columns, constraints, indexes, triggers, RLS policies
- **Views**: View definitions and dependencies
- **Functions**: Parameters, return types, language
- **Procedures**: Parameters and language
- **Types**: Enums, composites, domains
- **Sequences**: Start, increment, min/max values

### Comparing Schemas

```go
// Compare two schemas to identify differences
oldSchema := // ... parse or introspect old schema
newSchema := // ... parse or introspect new schema

// The main pgschema tool provides diff functionality
// See github.com/pgschema/pgschema/internal/diff for implementation
```

## Key Features

- **Database Introspection**: Query live databases using optimized SQL queries
- **Normalization**: Consistent representation from PostgreSQL system catalogs
- **Rich Type System**: Full support for PostgreSQL data types and constraints
- **Concurrent Safe**: Thread-safe access to schema data structures
- **Embedded Testing**: Use embedded PostgreSQL for testing without Docker (see `ParseSQLForTest` in testutil.go)

## Schema Object Types

### Tables
```go
type Table struct {
    Schema       string
    Name         string
    Type         TableType // BASE_TABLE, VIEW, etc.
    Columns      []*Column
    Constraints  map[string]*Constraint
    Indexes      map[string]*Index
    Triggers     map[string]*Trigger
    RLSEnabled   bool
    Policies     map[string]*RLSPolicy
    // ...
}
```

### Functions
```go
type Function struct {
    Schema     string
    Name       string
    Arguments  []*Parameter
    Returns    string
    Language   string
    Body       string
    // ...
}
```

### Views
```go
type View struct {
    Schema     string
    Name       string
    Definition string
    Columns    []*Column
    // ...
}
```

## Generated SQL Queries

The package includes pre-generated SQL queries in `queries/` for database introspection:

```go
import "github.com/pgschema/pgschema/ir/queries"

q := queries.New(db)
tables, err := q.GetTables(ctx, "public")
```

## Testing

```bash
# Run all tests (uses embedded PostgreSQL, no Docker required)
go test -v ./...

# Skip integration tests (faster)
go test -short -v ./...
```

## Versioning

This package follows semantic versioning. Releases are tagged with the pattern `ir/v<major>.<minor>.<patch>`.

### Release Process

1. Update the version in `ir/VERSION`
2. Commit the change
3. Create and push a tag: `git tag ir/v0.1.0 && git push origin ir/v0.1.0`
4. Or trigger a manual release from the GitHub Actions workflow

### Changelog

#### v0.1.0
- Initial standalone release of the IR package
- Support for PostgreSQL schema parsing and introspection
- Complete type system for tables, views, functions, procedures, types, and sequences

## Version Compatibility

- **Go**: 1.24.0+
- **PostgreSQL**: 14, 15, 16, 17

## License

Same as the parent pgschema project.