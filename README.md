![light-banner](https://raw.githubusercontent.com/pgschema/pgschema/main/docs/logo/light.png#gh-light-mode-only)
![dark-banner](https://raw.githubusercontent.com/pgschema/pgschema/main/docs/logo/dark.png#gh-dark-mode-only)

<a href="https://www.star-history.com/#pgschema/pgschema&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=pgschema/pgschema&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=pgschema/pgschema&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=pgschema/pgschema&type=Date" />
 </picture>
</a>

`pgschema` is a CLI tool that brings terraform-style declarative schema migration workflow to Postgres:

- **Dump** a Postgres schema in a developer-friendly format with support for all common objects
- **Edit** a schema to the desired state
- **Plan** a schema migration by comparing desired state with current database state
- **Apply** a schema migration with concurrent change detection, transaction-adaptive execution, and lock timeout control

Think of it as Terraform for your Postgres schemas - declare your desired state, generate plan, preview changes, and apply them with confidence.

## Key differentiators from other tools

1. **Comprehensive Postgres Support**: Handles virtually all schema-level database objects across Postgres versions 14 through 17
1. **State-Based Terraform-Like Workflow**: No separate migration table needed to track migration history - determines changes by comparing your schema files with actual database state
1. **Schema-Level Focus**: Designed for real-world Postgres usage patterns, from single-schema applications to multi-tenant architectures
1. **No Shadow Database Required**: Works directly with your schema files and target database - no temporary databases needed for validation

See more details in the [introduction blog post](https://www.pgschema.com/blog/pgschema-postgres-declarative-schema-migration-like-terraform).

Watch in action:

[![asciicast](https://asciinema.org/a/vXHygDMUkGYsF6nmz2h0ONEQC.svg)](https://asciinema.org/a/vXHygDMUkGYsF6nmz2h0ONEQC)


## Installation

> **Note**: Windows is not supported. Please use WSL (Windows Subsystem for Linux) or a Linux VM.

- **Pre-built binaries**: Download from the [releases page](https://github.com/pgschema/pgschema/releases)
- **Build from source**: See [Development](#development) section below
- **Detailed instructions**: Visit https://www.pgschema.com/installation

**macOS (Apple Silicon)**

```bash
# Download and install the latest release
curl -L \
  https://github.com/pgschema/pgschema/releases/latest/download/pgschema-darwin-arm64 \
  -o pgschema
chmod +x pgschema
sudo mv pgschema /usr/local/bin/
# Verify installation
pgschema --help
```

**Linux (AMD64)**

```bash
# Download and install the latest release
curl -L \
  https://github.com/pgschema/pgschema/releases/latest/download/pgschema-linux-amd64 \
  -o pgschema
chmod +x pgschema
sudo mv pgschema /usr/local/bin/
# Verify installation
pgschema --help
```

## Getting help

- [Docs](https://www.pgschema.com)
- [Discord channel](https://discord.gg/rvgZCYuJG4)
- [GitHub issues](https://github.com/pgschema/pgschema/issues)

## Quick example

### Step 1: Dump schema

```bash
# Dump current schema
$ PGPASSWORD=testpwd1 pgschema dump \
    --host localhost \
    --db testdb \
    --user postgres \
    --schema public > schema.sql
```

### Step 2: Edit schema

```bash
# Edit schema file declaratively
--- a/schema.sql
+++ b/schema.sql
@@ -12,5 +12,6 @@

 CREATE TABLE IF NOT EXISTS users (
     id SERIAL PRIMARY KEY,
-    username varchar(50) NOT NULL UNIQUE
+    username varchar(50) NOT NULL UNIQUE,
+    age INT NOT NULL
 );
```

### Step 3: Generate plan

```bash
$ PGPASSWORD=testpwd1 pgschema plan \
    --host localhost \
    --db testdb \
    --user postgres \
    --schema public \
    --file schema.sql \
    --output-human stdout \
    --output-json plan.json

Plan: 1 to modify.

Summary by type:
  tables: 1 to modify

Tables:
  ~ users
    + age (column)

Transaction: true

DDL to be executed:
--------------------------------------------------

ALTER TABLE users ADD COLUMN age integer NOT NULL;
```

### Step 4: Apply plan with confirmation

```bash
# Or use --auto-approve to skip confirmation
$ PGPASSWORD=testpwd1 pgschema apply \
    --host localhost \
    --db testdb \
    --user postgres \
    --schema public \
    --plan plan.json

Plan: 1 to modify.

Summary by type:
  tables: 1 to modify

Tables:
  ~ users
    + age (column)

Transaction: true

DDL to be executed:
--------------------------------------------------

ALTER TABLE users ADD COLUMN age integer NOT NULL;

Do you want to apply these changes? (yes/no): yes

Applying changes...
Changes applied successfully!
```

## LLM Readiness

- https://www.pgschema.com/llms.txt
- https://www.pgschema.com/llms-full.txt

![_](https://raw.githubusercontent.com/pgschema/pgschema/main/docs/images/copy-page.webp)

## Development

> [!NOTE]
> **For external contributors**: If you require any features, please create a GitHub issue to discuss first instead of creating a PR directly.

### Build

```bash
git clone https://github.com/pgschema/pgschema.git
cd pgschema
go mod tidy
go build -o pgschema .
```

### Run tests

```bash
# Run unit tests only
go test -short -v ./...

# Run all tests including integration tests (uses Postgres testcontainers with Docker)
go test -v ./...
```

## Sponsor

[Bytebase](https://www.bytebase.com?utm_sourcepgschema) - open source, web-based database DevSecOps platform.

<a href="https://www.bytebase.com?utm_sourcepgschema"><img src="https://raw.githubusercontent.com/pgschema/pgschema/main/docs/images/bytebase.webp" /></a>
