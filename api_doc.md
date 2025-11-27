# pgschema Public API

This package provides a programmatic API for pgschema's PostgreSQL schema management functionality. It offers Terraform-style declarative schema migration workflows with dump/plan/apply operations that can be used directly from Go code.

## Quick Start

```go
package main

import (
    "context"
    "log"

    "github.com/pgschema/pgschema/pgschema"
)

func main() {
    ctx := context.Background()

    dbConfig := pgschema.DatabaseConfig{
        Host:     "localhost",
        Port:     5432,
        Database: "myapp",
        User:     "postgres",
        Password: "password",
        Schema:   "public",
    }

    // Dump current schema
    schema, err := pgschema.DumpSchema(ctx, dbConfig)
    if err != nil {
        log.Fatal(err)
    }

    // Generate migration plan
    plan, err := pgschema.GeneratePlan(ctx, dbConfig, "desired_schema.sql")
    if err != nil {
        log.Fatal(err)
    }

    // Apply migration
    err = pgschema.ApplyPlan(ctx, dbConfig, plan, true)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Core Operations

### Dump

Extract current database schema as SQL:

```go
// Simple dump to string
schema, err := pgschema.DumpSchema(ctx, dbConfig)

// Dump to file
err := pgschema.DumpSchemaToFile(ctx, dbConfig, "schema.sql")

// Dump to multiple files organized by object type
err := pgschema.DumpSchemaMultiFile(ctx, dbConfig, "schemas/")
```

### Plan

Generate migration plans by comparing current and desired states:

```go
// Generate plan from desired state file
plan, err := pgschema.GeneratePlan(ctx, dbConfig, "desired_schema.sql")

// Generate plan with custom options
client := pgschema.NewClient(dbConfig)
plan, err := client.Plan(ctx, pgschema.PlanOptions{
    File:            "desired_schema.sql",
    ApplicationName: "my-migration-tool",
})
```

### Apply

Apply migration plans to update database schema:

```go
// Apply schema file directly
err := pgschema.ApplySchemaFile(ctx, dbConfig, "desired_schema.sql", false)

// Apply with auto-approval (no prompts)
err := pgschema.ApplySchemaFile(ctx, dbConfig, "desired_schema.sql", true)

// Apply silently (no output)
err := pgschema.QuietApplySchemaFile(ctx, dbConfig, "desired_schema.sql")
```

## Relationship to CLI

This API provides programmatic access to the same functionality available through the pgschema CLI commands:

- `pgschema dump` → `pgschema.DumpSchema()` / `Client.Dump()`
- `pgschema plan` → `pgschema.GeneratePlan()` / `Client.Plan()`
- `pgschema apply` → `pgschema.ApplySchemaFile()` / `Client.Apply()`

The CLI commands and public API are independent - using the API doesn't affect CLI behavior.