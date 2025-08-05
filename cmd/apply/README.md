# Apply Command

The `apply` command applies a desired schema state to a target database schema. It compares the desired state (from a file) with the current state of a specific schema and applies the necessary changes.

## Features

- **Schema Application**: Applies changes to make the database match the desired state
- **Interactive Confirmation**: By default, prompts user to confirm before applying changes
- **Auto-approve Mode**: Option to apply changes without user confirmation
- **Connection Options**: Same connection options as plan and dump commands
- **Safe Execution**: Shows plan before applying changes
- **Schema Filtering**: Can target specific schemas (defaults to 'public')

## Usage

```bash
# Basic usage - will prompt for confirmation
pgschema apply --host hostname --db dbname --user username --file schema.sql

# Auto-approve changes without prompting
pgschema apply --host hostname --db dbname --user username --file schema.sql --auto-approve

# Apply to specific schema
pgschema apply --host hostname --db dbname --user username --schema myschema --file schema.sql

# With password
pgschema apply --host hostname --db dbname --user username --password mypassword --file schema.sql

# Disable colored output
pgschema apply --host hostname --db dbname --user username --file schema.sql --no-color

# Set custom lock timeout
pgschema apply --host hostname --db dbname --user username --file schema.sql --lock-timeout 5m

# Set custom application name
pgschema apply --host hostname --db dbname --user username --file schema.sql --application-name "myapp-migration"
```

## Flags

### Connection Options

- `--host`: Database server host (default: localhost)
- `--port`: Database server port (default: 5432)
- `--db`: Database name (required)
- `--user`: Database user name (required)
- `--password`: Database password (optional, can also use PGPASSWORD env var)
- `--schema`: Schema name to apply changes to (default: public)

### Apply Options

- `--file`: Path to desired state SQL schema file (required)
- `--auto-approve`: Apply changes without prompting for approval
- `--no-color`: Disable colored output
- `--lock-timeout`: Maximum time to wait for database locks (e.g., 30s, 5m, 1h)
- `--application-name`: Application name for database connection (default: pgschema)

### Global Options

- `--debug`: Enable debug logging

## Workflow

1. **Read Desired State**: Reads the schema definition from the specified file
2. **Analyze Current State**: Connects to the database and extracts the current schema
3. **Generate Plan**: Creates a migration plan showing what changes will be applied
4. **Display Plan**: Shows the plan in human-readable format with colored output
5. **Confirm Changes**: Prompts user to confirm (unless `--auto-approve` is used)
6. **Apply Changes**: Executes the SQL statements to update the database
7. **Report Results**: Confirms successful application of changes

## Examples

### Basic Apply with Confirmation

```bash
pgschema apply --host localhost --db myapp --user postgres --file desired_schema.sql
```

This will:

1. Show you the changes that will be applied
2. Prompt: "Do you want to apply these changes? (yes/no):"
3. Wait for your confirmation before proceeding

### Auto-approve for CI/CD

```bash
pgschema apply --host prod-db --db myapp --user deployer --file schema.sql --auto-approve
```

Perfect for automated deployments where manual confirmation is not possible.

### Apply to Specific Schema

```bash
pgschema apply --host localhost --db myapp --user postgres --schema tenant1 --file tenant_schema.sql
```

### Lock Timeout Control

```bash
pgschema apply --host localhost --db myapp --user postgres --file schema.sql --lock-timeout 5m
```

Controls how long the apply operation waits for database locks before timing out. If not specified, uses PostgreSQL's default lock timeout behavior. Useful for:

- **Production deployments**: Avoid hanging indefinitely on locked tables
- **Busy databases**: Set shorter timeouts to fail fast if tables are in use
- **Long operations**: Set longer timeouts for complex migrations

Common timeout values:

- `30s`: Good for most operations
- `5m`: For larger schema changes
- `10m`: For complex migrations with many dependencies
- `1h`: For very large database migrations

### Database Connection Monitoring

```bash
pgschema apply --host localhost --db myapp --user postgres --file schema.sql --application-name "deployment-v2.1"
```

Sets a custom application name that appears in PostgreSQL's `pg_stat_activity` view. Useful for:

- **Monitoring**: Identify pgschema connections in database monitoring tools
- **Debugging**: Track which deployment or process is running migrations
- **Auditing**: Log and trace database changes to specific operations

You can monitor active pgschema connections with:

```sql
SELECT application_name, state, query_start, query
FROM pg_stat_activity
WHERE application_name LIKE 'pgschema%';
```

## Safety Features

- **Plan Preview**: Always shows what changes will be applied before execution
- **No Changes Detection**: Skips execution if no changes are needed
- **Transaction Safety**: SQL statements are executed in a single transaction
- **Error Handling**: Detailed error messages for connection and execution failures

## Password Handling

You can provide the password using either:

1. **Command line flag**: `--password mypassword`
2. **Environment variable**: `PGPASSWORD=mypassword pgschema apply ...`

The environment variable method is recommended for security, especially in scripts.

## Exit Codes

- `0`: Success (changes applied or no changes needed)
- `1`: Error (connection failed, file not found, SQL execution failed, etc.)

## Related Commands

- [`plan`](../plan/README.md): Preview changes without applying them
- [`dump`](../dump/README.md): Extract current schema state

## Running Tests

```bash
# All plan tests
go test -v ./cmd/apply/

# Specific plan tests
go test -v ./cmd/apply/ -run "TestApplyCommand_TransactionRollback"
```
