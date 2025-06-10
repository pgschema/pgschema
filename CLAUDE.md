# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

pgschema is a PostgreSQL schema comparison tool that identifies differences between database schemas and generates SQL migration statements. It supports comparing directories (containing SQL files), live databases, or a mix of both.

## Commands

### Build
```bash
go build -o pgschema .
```

### Test
```bash
# Unit tests only (no PostgreSQL required)
go test -short -v ./...

# All tests including integration tests (requires PostgreSQL)
go test -v ./... -test-temp-db-dsn="postgres://user:password@localhost:5432/postgres?sslmode=disable"
```

### Dependencies
```bash
go mod tidy
```

## Architecture

The application uses:
- **Cobra** for CLI structure with `diff` and `version` subcommands
- **Stripe's pg-schema-diff** library as the core comparison engine
- **Temporary databases** for analyzing directory-based schemas

Key components:
- `main.go`: CLI logic, schema loading, and temporary database management
- `main_test.go`: Unit tests for CLI validation and schema loading
- `integration_test.go`: End-to-end tests requiring PostgreSQL
- `testdata/`: Sample schemas demonstrating evolution patterns

When comparing directories, the tool creates temporary databases to load SQL files, performs the comparison, then cleans up. This approach allows unified comparison logic regardless of source type.