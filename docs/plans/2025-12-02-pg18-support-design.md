# PostgreSQL 18 Support Design

## Overview

Upgrade `embedded-postgres` from v1.32.0 to v1.33.0 and add PostgreSQL 18 support throughout the codebase.

## Changes

### 1. Dependency Upgrade

```bash
go get github.com/fergusstrange/embedded-postgres@v1.33.0
```

### 2. Code Changes

#### 2.1 `internal/postgres/embedded.go`

Replace hardcoded version strings with embedded-postgres constants in `mapToEmbeddedPostgresVersion()`:

```go
func mapToEmbeddedPostgresVersion(majorVersion int) (PostgresVersion, error) {
    switch majorVersion {
    case 14:
        return embeddedpostgres.V14, nil
    case 15:
        return embeddedpostgres.V15, nil
    case 16:
        return embeddedpostgres.V16, nil
    case 17:
        return embeddedpostgres.V17, nil
    case 18:
        return embeddedpostgres.V18, nil
    default:
        return "", fmt.Errorf("unsupported PostgreSQL version %d (supported: 14-18)", majorVersion)
    }
}
```

#### 2.2 `testutil/postgres.go`

Update `getPostgresVersion()` to use constants and default to PG18:

```go
func getPostgresVersion() embeddedpostgres.PostgresVersion {
    versionStr := os.Getenv("PGSCHEMA_POSTGRES_VERSION")
    switch versionStr {
    case "14":
        return embeddedpostgres.V14
    case "15":
        return embeddedpostgres.V15
    case "16":
        return embeddedpostgres.V16
    case "17":
        return embeddedpostgres.V17
    case "18", "":
        return embeddedpostgres.V18
    default:
        return embeddedpostgres.V18
    }
}
```

Note: Return type changes from `postgres.PostgresVersion` to `embeddedpostgres.PostgresVersion`.

### 3. Documentation Updates

Update version references from "14-17" to "14-18" in:

| File | Lines |
|------|-------|
| `README.md` | Line 23 |
| `CLAUDE.md` | Lines 19, 187 |
| `ir/README.md` | Line 150 |
| `.claude/skills/pg_dump/SKILL.md` | Lines 16, 231, 245 |
| `.claude/skills/validate_db/SKILL.md` | Lines 19, 339, 645 |
| `.claude/skills/postgres_syntax/SKILL.md` | Lines 557, 575 |
| `.claude/skills/run_tests/SKILL.md` | Lines 151, 170, 489 |

### 4. Verification

Run full test suite on PG18:

```bash
PGSCHEMA_POSTGRES_VERSION=18 go test ./...
```

## Key Decisions

- **Default version**: PG18 (matching embedded-postgres v1.33.0 default)
- **Version constants**: Use `embeddedpostgres.V14` etc. instead of hardcoded strings like `"14.18.0"`
- **Scope**: Update all documentation including skills for consistency
