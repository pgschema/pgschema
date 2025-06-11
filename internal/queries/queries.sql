-- GetSchemas retrieves all user-defined schemas
-- name: GetSchemas :many
SELECT 
    schema_name
FROM information_schema.schemata
WHERE 
    schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND schema_name NOT LIKE 'pg_temp_%'
    AND schema_name NOT LIKE 'pg_toast_temp_%'
ORDER BY schema_name;

-- GetTables retrieves all tables in the database with metadata
-- name: GetTables :many
SELECT 
    t.table_schema,
    t.table_name,
    t.table_type
FROM information_schema.tables t
WHERE 
    t.table_schema NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND t.table_schema NOT LIKE 'pg_temp_%'
    AND t.table_schema NOT LIKE 'pg_toast_temp_%'
    AND t.table_type IN ('BASE TABLE', 'VIEW')
ORDER BY t.table_schema, t.table_name;

-- GetColumns retrieves all columns for all tables
-- name: GetColumns :many
SELECT 
    c.table_schema,
    c.table_name,
    c.column_name,
    c.ordinal_position,
    c.column_default,
    c.is_nullable,
    c.data_type,
    c.character_maximum_length,
    c.numeric_precision,
    c.numeric_scale,
    c.udt_name
FROM information_schema.columns c
WHERE 
    c.table_schema NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND c.table_schema NOT LIKE 'pg_temp_%'
    AND c.table_schema NOT LIKE 'pg_toast_temp_%'
ORDER BY c.table_schema, c.table_name, c.ordinal_position;

-- GetConstraints retrieves all table constraints
-- name: GetConstraints :many
SELECT 
    tc.table_schema,
    tc.table_name,
    tc.constraint_name,
    tc.constraint_type,
    kcu.column_name,
    ccu.table_schema AS foreign_table_schema,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name,
    cc.check_clause
FROM information_schema.table_constraints tc
LEFT JOIN information_schema.key_column_usage kcu 
    ON tc.constraint_name = kcu.constraint_name 
    AND tc.table_schema = kcu.table_schema
LEFT JOIN information_schema.constraint_column_usage ccu 
    ON tc.constraint_name = ccu.constraint_name 
    AND tc.table_schema = ccu.table_schema
LEFT JOIN information_schema.check_constraints cc 
    ON tc.constraint_name = cc.constraint_name 
    AND tc.table_schema = cc.constraint_schema
WHERE 
    tc.table_schema NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND tc.table_schema NOT LIKE 'pg_temp_%'
    AND tc.table_schema NOT LIKE 'pg_toast_temp_%'
ORDER BY tc.table_schema, tc.table_name, tc.constraint_type, tc.constraint_name;

-- GetIndexes retrieves all indexes (simplified for sqlc compatibility)
-- name: GetIndexes :many
SELECT 
    tc.table_schema as schemaname,
    tc.table_name as tablename,
    tc.constraint_name as indexname,
    'INDEX' as indextype
FROM information_schema.table_constraints tc
WHERE 
    tc.constraint_type IN ('PRIMARY KEY', 'UNIQUE')
    AND tc.table_schema NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND tc.table_schema NOT LIKE 'pg_temp_%'
    AND tc.table_schema NOT LIKE 'pg_toast_temp_%'
ORDER BY tc.table_schema, tc.table_name, tc.constraint_name;

-- GetSequences retrieves all sequences
-- name: GetSequences :many
SELECT 
    sequence_schema,
    sequence_name,
    data_type,
    start_value,
    minimum_value,
    maximum_value,
    increment,
    cycle_option
FROM information_schema.sequences
WHERE 
    sequence_schema NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND sequence_schema NOT LIKE 'pg_temp_%'
    AND sequence_schema NOT LIKE 'pg_toast_temp_%'
ORDER BY sequence_schema, sequence_name;