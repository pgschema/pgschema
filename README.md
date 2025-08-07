![light-banner](https://raw.githubusercontent.com/pgschema/pgschema/main/docs/logo/light.png#gh-light-mode-only)
![dark-banner](https://raw.githubusercontent.com/pgschema/pgschema/main/docs/logo/dark.png#gh-dark-mode-only)

`pgschema` is a CLI tool that brings terraform-style declarative schema migration workflow to Postgres:

- **Dump** a Postgres schema in a developer-friendly format with support for all common objects
- **Edit** a schema to the desired state
- **Plan** a schema migration by comparing desired state with current database state
- **Apply** a schema migration with concurrent change detection, transaction-adaptive execution, and lock timeout control

Think of it as Terraform for your Postgres schemas - declare your desired state, generate plan, preview changes, and apply them with confidence.

## Supported Postgres version

[Tested with](https://github.com/pgschema/pgschema/blob/a06547f986adc0c7f42cb890420e79569a0e3ec9/.github/workflows/release.yml#L44-L46) Postgres 14, 15, 16, 17.

## Installation

Follow https://www.pgschema.com/installation

## Quick example

### Dump schema

```bash
# Dump current schema
$ PGPASSWORD=testpwd1 pgschema dump \
    --host localhost \
    --db testdb \
    --user postgres \
    --schema public > schema.sql
```

### Edit schema

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

### Generate plan

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

### Apply plan with confirmation

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

## Development

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

[Bytebase](https://www.bytebase.com?source=pgschema) - open source, web-based database DevSecOps platform.

<a href="https://www.bytebase.com?source=pgschema"><img src="https://raw.githubusercontent.com/pgschema/pgschema/main/docs/images/bytebase.webp" /></a>

## Star History

<a href="https://www.star-history.com/#pgschema/pgschema&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=pgschema/pgschema&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=pgschema/pgschema&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=pgschema/pgschema&type=Date" />
 </picture>
</a>
