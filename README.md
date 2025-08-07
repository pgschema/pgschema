> [!NOTE]  
> Brought to you by [Bytebase](https://www.bytebase.com/), open-source database DevSecOps platform.

![light-banner](https://raw.githubusercontent.com/pgschema/pgschema/main/docs/logo/light.png#gh-light-mode-only)
![dark-banner](https://raw.githubusercontent.com/pgschema/pgschema/main/docs/logo/dark.png#gh-dark-mode-only)

# pgschema

A CLI tool that brings terraform-style declarative schema migration workflow to Postgres:

- **Dump** a Postgres schema in a developer-friendly format with support for all common objects
- **Plan** a schema migration by comparing desired state with current database state
- **Apply** a schema migration with concurrent change detection, transaction-adaptive execution, and lock timeout control

Think of it as Terraform for your Postgres schemas - declare your desired state, generate plan, preview changes, and apply them with confidence.

## Installation

Follow https://www.pgschema.com/installation

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

# Run all tests including integration tests (uses testcontainers with Docker)
go test -v ./...
```

## Sponsor

[Bytebase](https://www.bytebase.com?source=pgschema) is an open source, web-based database DevSecOps platform.

<a href="https://www.bytebase.com?source=pgschema"><img src="https://raw.githubusercontent.com/pgschema/pgschema/main/docs/images/bytebase.webp" /></a>

## Star History

<a href="https://www.star-history.com/#pgschema/pgschema&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=pgschema/pgschema&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=pgschema/pgschema&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=pgschema/pgschema&type=Date" />
 </picture>
</a>
