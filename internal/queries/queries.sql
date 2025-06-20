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
    n.nspname AS table_schema,
    cl.relname AS table_name,
    c.conname AS constraint_name,
    CASE c.contype
        WHEN 'c' THEN 'CHECK'
        WHEN 'f' THEN 'FOREIGN KEY'
        WHEN 'p' THEN 'PRIMARY KEY'
        WHEN 'u' THEN 'UNIQUE'
        WHEN 'x' THEN 'EXCLUDE'
        ELSE 'UNKNOWN'
    END AS constraint_type,
    a.attname AS column_name,
    a.attnum AS ordinal_position,
    fn.nspname AS foreign_table_schema,
    fcl.relname AS foreign_table_name,
    fa.attname AS foreign_column_name,
    fa.attnum AS foreign_ordinal_position,
    CASE WHEN c.contype = 'c' THEN pg_get_constraintdef(c.oid) ELSE NULL END AS check_clause,
    CASE c.confdeltype
        WHEN 'a' THEN 'NO ACTION'
        WHEN 'r' THEN 'RESTRICT'
        WHEN 'c' THEN 'CASCADE'
        WHEN 'n' THEN 'SET NULL'
        WHEN 'd' THEN 'SET DEFAULT'
        ELSE NULL
    END AS delete_rule,
    CASE c.confupdtype
        WHEN 'a' THEN 'NO ACTION'
        WHEN 'r' THEN 'RESTRICT'
        WHEN 'c' THEN 'CASCADE'
        WHEN 'n' THEN 'SET NULL'
        WHEN 'd' THEN 'SET DEFAULT'
        ELSE NULL
    END AS update_rule
FROM pg_constraint c
JOIN pg_class cl ON c.conrelid = cl.oid
JOIN pg_namespace n ON cl.relnamespace = n.oid
LEFT JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY(c.conkey)
LEFT JOIN pg_class fcl ON c.confrelid = fcl.oid
LEFT JOIN pg_namespace fn ON fcl.relnamespace = fn.oid
LEFT JOIN pg_attribute fa ON fa.attrelid = c.confrelid AND fa.attnum = c.confkey[array_position(c.conkey, a.attnum)]
WHERE n.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND n.nspname NOT LIKE 'pg_temp_%'
    AND n.nspname NOT LIKE 'pg_toast_temp_%'
ORDER BY n.nspname, cl.relname, c.contype, c.conname, a.attnum;

-- GetIndexes retrieves all indexes including regular and unique indexes created with CREATE INDEX
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

-- GetFunctions retrieves all user-defined functions (excluding extension members)
-- name: GetFunctions :many
SELECT 
    r.routine_schema,
    r.routine_name,
    r.routine_definition,
    r.routine_type,
    r.data_type,
    r.external_language
FROM information_schema.routines r
LEFT JOIN pg_proc p ON p.proname = r.routine_name 
    AND p.pronamespace = (SELECT oid FROM pg_namespace WHERE nspname = r.routine_schema)
LEFT JOIN pg_depend d ON d.objid = p.oid AND d.deptype = 'e'
WHERE 
    r.routine_schema NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND r.routine_schema NOT LIKE 'pg_temp_%'
    AND r.routine_schema NOT LIKE 'pg_toast_temp_%'
    AND r.routine_type = 'FUNCTION'
    AND d.objid IS NULL  -- Exclude functions that are extension members
ORDER BY r.routine_schema, r.routine_name;

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

-- GetExtensions retrieves all extensions
-- name: GetExtensions :many
SELECT 
    n.nspname AS schema_name,
    e.extname AS extension_name,
    e.extversion AS extension_version,
    d.description AS extension_comment
FROM pg_extension e
JOIN pg_namespace n ON e.extnamespace = n.oid
LEFT JOIN pg_description d ON d.objoid = e.oid AND d.classoid = 'pg_extension'::regclass
WHERE n.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND n.nspname NOT LIKE 'pg_temp_%'
    AND n.nspname NOT LIKE 'pg_toast_temp_%'
ORDER BY e.extname;

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