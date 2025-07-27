# Include Functionality Test Data

Test data for pgschema's `\i` include functionality with simplified SQL files.

## Files

- **`main.sql`** - Complete schema with all includes
- **`schema.sql`** - Pre-generated concatenated output 
- **`simple_test.sql`** - Basic subset for quick testing
- **`nested_main.sql`** - Tests nested includes

## Directory Structure

```
types/           # Enums and composite types
domains/         # Domain types with constraints  
sequences/       # Basic sequences
tables/          # Tables with indexes, constraints, policies, comments
functions/       # Simple SQL functions
procedures/      # Simple procedures
views/           # Views with comments
triggers/        # Trigger functions and triggers
```

## Usage

```sql
-- Include files
\i tables/users.sql
\i views/user_views.sql

-- Use with pgschema
pgschema plan --file main.sql --db mydb --user myuser
```

## Security

- Path restricted to current directory and subdirectories
- No `../` directory traversal allowed
- Circular dependency detection
- File existence validation

## Testing

Tests cover:
- Basic and nested includes
- Security (path traversal, cycles)
- Error handling (missing files)
- Output validation against `schema.sql`