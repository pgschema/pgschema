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
    kcu.ordinal_position,
    ccu.table_schema AS foreign_table_schema,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name,
    cc.check_clause,
    rc.delete_rule,
    rc.update_rule
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
LEFT JOIN information_schema.referential_constraints rc
    ON tc.constraint_name = rc.constraint_name
    AND tc.table_schema = rc.constraint_schema
WHERE 
    tc.table_schema NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND tc.table_schema NOT LIKE 'pg_temp_%'
    AND tc.table_schema NOT LIKE 'pg_toast_temp_%'
ORDER BY tc.table_schema, tc.table_name, tc.constraint_type, tc.constraint_name, kcu.ordinal_position;

-- GetIndexes retrieves all indexes including regular indexes created with CREATE INDEX
-- name: GetIndexes :many
SELECT 
    n.nspname as schemaname,
    t.relname as tablename,
    i.relname as indexname,
    pg_get_indexdef(idx.indexrelid) as indexdef
FROM pg_index idx
JOIN pg_class i ON i.oid = idx.indexrelid
JOIN pg_class t ON t.oid = idx.indrelid
JOIN pg_namespace n ON n.oid = t.relnamespace
WHERE 
    NOT idx.indisprimary
    AND NOT idx.indisunique
    AND n.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND n.nspname NOT LIKE 'pg_temp_%'
    AND n.nspname NOT LIKE 'pg_toast_temp_%'
ORDER BY n.nspname, t.relname, i.relname;

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

-- GetFunctions retrieves all user-defined functions
-- name: GetFunctions :many
SELECT 
    routine_schema,
    routine_name,
    routine_definition,
    routine_type,
    data_type,
    external_language
FROM information_schema.routines
WHERE 
    routine_schema NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND routine_schema NOT LIKE 'pg_temp_%'
    AND routine_schema NOT LIKE 'pg_toast_temp_%'
    AND routine_type = 'FUNCTION'
ORDER BY routine_schema, routine_name;

-- GetViews retrieves all views
-- name: GetViews :many
SELECT 
    table_schema,
    table_name,
    view_definition
FROM information_schema.views
WHERE 
    table_schema NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND table_schema NOT LIKE 'pg_temp_%'
    AND table_schema NOT LIKE 'pg_toast_temp_%'
ORDER BY table_schema, table_name;

-- GetExtensions retrieves all extensions (placeholder for now)
-- name: GetExtensions :many
SELECT 
    'public' AS schema_name,
    'placeholder' AS extension_name,
    'placeholder' AS extension_version
WHERE false;

-- GetTriggers retrieves all triggers
-- name: GetTriggers :many
SELECT 
    trigger_schema,
    trigger_name,
    event_object_table,
    action_timing,
    event_manipulation,
    action_statement
FROM information_schema.triggers
WHERE 
    trigger_schema NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND trigger_schema NOT LIKE 'pg_temp_%'
    AND trigger_schema NOT LIKE 'pg_toast_temp_%'
ORDER BY trigger_schema, event_object_table, trigger_name;

-- GetViewDependencies retrieves view dependencies on tables and other views  
-- name: GetViewDependencies :many
SELECT DISTINCT
    vtu.view_schema AS dependent_schema,
    vtu.view_name AS dependent_name,
    vtu.table_schema AS source_schema,
    vtu.table_name AS source_name
FROM information_schema.view_table_usage vtu
WHERE 
    vtu.view_schema NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND vtu.view_schema NOT LIKE 'pg_temp_%'
    AND vtu.view_schema NOT LIKE 'pg_toast_temp_%'
    AND vtu.table_schema NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND vtu.table_schema NOT LIKE 'pg_temp_%'
    AND vtu.table_schema NOT LIKE 'pg_toast_temp_%'
ORDER BY vtu.view_schema, vtu.view_name, vtu.table_schema, vtu.table_name;

-- GetRLSTables retrieves tables with row level security enabled
-- name: GetRLSTables :many
SELECT 
    schemaname,
    tablename
FROM pg_tables
WHERE 
    schemaname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND schemaname NOT LIKE 'pg_temp_%'
    AND schemaname NOT LIKE 'pg_toast_temp_%'
    AND rowsecurity = true
ORDER BY schemaname, tablename;

-- GetRLSPolicies retrieves all row level security policies
-- name: GetRLSPolicies :many
SELECT 
    schemaname,
    tablename,
    policyname,
    permissive,
    cmd,
    qual,
    with_check
FROM pg_policies
WHERE 
    schemaname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND schemaname NOT LIKE 'pg_temp_%'
    AND schemaname NOT LIKE 'pg_toast_temp_%'
ORDER BY schemaname, tablename, policyname;