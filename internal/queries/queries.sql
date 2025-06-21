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
    t.table_type,
    d.description AS table_comment
FROM information_schema.tables t
LEFT JOIN pg_class c ON c.relname = t.table_name
LEFT JOIN pg_namespace n ON c.relnamespace = n.oid AND n.nspname = t.table_schema
LEFT JOIN pg_description d ON d.objoid = c.oid AND d.classoid = 'pg_class'::regclass AND d.objsubid = 0
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
    COALESCE(pg_get_expr(ad.adbin, ad.adrelid), c.column_default) AS column_default,
    c.is_nullable,
    c.data_type,
    c.character_maximum_length,
    c.numeric_precision,
    c.numeric_scale,
    c.udt_name,
    d.description AS column_comment,
    CASE 
        WHEN dt.typtype = 'd' THEN dn.nspname || '.' || dt.typname
        ELSE c.udt_name
    END AS resolved_type
FROM information_schema.columns c
LEFT JOIN pg_class cl ON cl.relname = c.table_name
LEFT JOIN pg_namespace n ON cl.relnamespace = n.oid AND n.nspname = c.table_schema
LEFT JOIN pg_description d ON d.objoid = cl.oid AND d.classoid = 'pg_class'::regclass AND d.objsubid = c.ordinal_position
LEFT JOIN pg_attribute a ON a.attrelid = cl.oid AND a.attname = c.column_name
LEFT JOIN pg_attrdef ad ON ad.adrelid = a.attrelid AND ad.adnum = a.attnum
LEFT JOIN pg_type dt ON dt.oid = a.atttypid
LEFT JOIN pg_namespace dn ON dt.typnamespace = dn.oid
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
    END AS update_rule,
    c.condeferrable AS deferrable,
    c.condeferred AS initially_deferred
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
    COALESCE(pg_get_function_result(p.oid), r.data_type) AS data_type,
    r.external_language,
    desc_func.description AS function_comment,
    oidvectortypes(p.proargtypes) AS function_arguments,
    pg_get_function_arguments(p.oid) AS function_signature,
    CASE p.provolatile
        WHEN 'i' THEN 'IMMUTABLE'
        WHEN 's' THEN 'STABLE'
        WHEN 'v' THEN 'VOLATILE'
        ELSE NULL
    END AS volatility,
    p.proisstrict AS is_strict,
    p.prosecdef AS is_security_definer
FROM information_schema.routines r
LEFT JOIN pg_proc p ON p.proname = r.routine_name 
    AND p.pronamespace = (SELECT oid FROM pg_namespace WHERE nspname = r.routine_schema)
LEFT JOIN pg_depend d ON d.objid = p.oid AND d.deptype = 'e'
LEFT JOIN pg_description desc_func ON desc_func.objoid = p.oid AND desc_func.classoid = 'pg_proc'::regclass
WHERE 
    r.routine_schema NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND r.routine_schema NOT LIKE 'pg_temp_%'
    AND r.routine_schema NOT LIKE 'pg_toast_temp_%'
    AND r.routine_type = 'FUNCTION'
    AND d.objid IS NULL  -- Exclude functions that are extension members
ORDER BY r.routine_schema, r.routine_name;

-- GetProcedures retrieves all user-defined procedures (excluding extension members)
-- name: GetProcedures :many
SELECT 
    r.routine_schema,
    r.routine_name,
    r.routine_definition,
    r.routine_type,
    r.external_language,
    desc_proc.description AS procedure_comment,
    oidvectortypes(p.proargtypes) AS procedure_arguments,
    pg_get_function_arguments(p.oid) AS procedure_signature
FROM information_schema.routines r
LEFT JOIN pg_proc p ON p.proname = r.routine_name 
    AND p.pronamespace = (SELECT oid FROM pg_namespace WHERE nspname = r.routine_schema)
LEFT JOIN pg_depend d ON d.objid = p.oid AND d.deptype = 'e'
LEFT JOIN pg_description desc_proc ON desc_proc.objoid = p.oid AND desc_proc.classoid = 'pg_proc'::regclass
WHERE 
    r.routine_schema NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND r.routine_schema NOT LIKE 'pg_temp_%'
    AND r.routine_schema NOT LIKE 'pg_toast_temp_%'
    AND r.routine_type = 'PROCEDURE'
    AND d.objid IS NULL  -- Exclude procedures that are extension members
ORDER BY r.routine_schema, r.routine_name;

-- GetAggregates retrieves all user-defined aggregates
-- name: GetAggregates :many
SELECT 
    n.nspname AS aggregate_schema,
    p.proname AS aggregate_name,
    pg_get_function_arguments(p.oid) AS aggregate_signature,
    oidvectortypes(p.proargtypes) AS aggregate_arguments,
    format_type(p.prorettype, NULL) AS aggregate_return_type,
    -- Get transition function
    tf.proname AS transition_function,
    tfn.nspname AS transition_function_schema,
    -- Get state type
    format_type(a.aggtranstype, NULL) AS state_type,
    -- Get initial condition
    a.agginitval AS initial_condition,
    -- Get final function if exists
    ff.proname AS final_function,
    ffn.nspname AS final_function_schema,
    -- Comment
    d.description AS aggregate_comment
FROM pg_proc p
JOIN pg_namespace n ON p.pronamespace = n.oid
JOIN pg_aggregate a ON a.aggfnoid = p.oid
LEFT JOIN pg_proc tf ON a.aggtransfn = tf.oid
LEFT JOIN pg_namespace tfn ON tf.pronamespace = tfn.oid
LEFT JOIN pg_proc ff ON a.aggfinalfn = ff.oid
LEFT JOIN pg_namespace ffn ON ff.pronamespace = ffn.oid
LEFT JOIN pg_description d ON d.objoid = p.oid AND d.classoid = 'pg_proc'::regclass
WHERE p.prokind = 'a'  -- Only aggregates
    AND n.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND n.nspname NOT LIKE 'pg_temp_%'
    AND n.nspname NOT LIKE 'pg_toast_temp_%'
    AND NOT EXISTS (
        SELECT 1 FROM pg_depend dep 
        WHERE dep.objid = p.oid AND dep.deptype = 'e'
    )  -- Exclude extension members
ORDER BY n.nspname, p.proname;

-- GetViews retrieves all views
-- name: GetViews :many
SELECT 
    v.table_schema,
    v.table_name,
    v.view_definition,
    d.description AS view_comment
FROM information_schema.views v
LEFT JOIN pg_class c ON c.relname = v.table_name
LEFT JOIN pg_namespace n ON c.relnamespace = n.oid AND n.nspname = v.table_schema
LEFT JOIN pg_description d ON d.objoid = c.oid AND d.classoid = 'pg_class'::regclass AND d.objsubid = 0
WHERE 
    v.table_schema NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND v.table_schema NOT LIKE 'pg_temp_%'
    AND v.table_schema NOT LIKE 'pg_toast_temp_%'
ORDER BY v.table_schema, v.table_name;

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

-- GetTypes retrieves all user-defined types (ENUM and composite types)
-- name: GetTypes :many
SELECT 
    n.nspname AS type_schema,
    t.typname AS type_name,
    CASE t.typtype
        WHEN 'e' THEN 'ENUM'
        WHEN 'c' THEN 'COMPOSITE'
        ELSE 'OTHER'
    END AS type_kind,
    d.description AS type_comment
FROM pg_type t
JOIN pg_namespace n ON t.typnamespace = n.oid
LEFT JOIN pg_description d ON d.objoid = t.oid AND d.classoid = 'pg_type'::regclass
LEFT JOIN pg_class c ON t.typrelid = c.oid
WHERE t.typtype IN ('e', 'c')  -- ENUM and composite types only
    AND n.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND n.nspname NOT LIKE 'pg_temp_%'
    AND n.nspname NOT LIKE 'pg_toast_temp_%'
    AND (t.typtype = 'e' OR (t.typtype = 'c' AND c.relkind = 'c'))  -- For composite types, only include true composite types (not table types)
ORDER BY n.nspname, t.typname;

-- GetEnumValues retrieves enum values for ENUM types
-- name: GetEnumValues :many
SELECT 
    n.nspname AS type_schema,
    t.typname AS type_name,
    e.enumlabel AS enum_value,
    e.enumsortorder AS enum_order
FROM pg_enum e
JOIN pg_type t ON e.enumtypid = t.oid
JOIN pg_namespace n ON t.typnamespace = n.oid
WHERE n.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND n.nspname NOT LIKE 'pg_temp_%'
    AND n.nspname NOT LIKE 'pg_toast_temp_%'
ORDER BY n.nspname, t.typname, e.enumsortorder;

-- GetCompositeTypeColumns retrieves columns for composite types
-- name: GetCompositeTypeColumns :many
SELECT 
    n.nspname AS type_schema,
    t.typname AS type_name,
    a.attname AS column_name,
    a.attnum AS column_position,
    format_type(a.atttypid, a.atttypmod) AS column_type
FROM pg_type t
JOIN pg_namespace n ON t.typnamespace = n.oid
JOIN pg_class c ON t.typrelid = c.oid
JOIN pg_attribute a ON c.oid = a.attrelid
WHERE t.typtype = 'c'  -- composite types only
    AND c.relkind = 'c'  -- only true composite types, not table types
    AND a.attnum > 0  -- exclude system columns
    AND NOT a.attisdropped  -- exclude dropped columns
    AND n.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND n.nspname NOT LIKE 'pg_temp_%'
    AND n.nspname NOT LIKE 'pg_toast_temp_%'
ORDER BY n.nspname, t.typname, a.attnum;

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

-- GetDomains retrieves all user-defined domains
-- name: GetDomains :many
SELECT 
    n.nspname AS domain_schema,
    t.typname AS domain_name,
    format_type(t.typbasetype, t.typtypmod) AS base_type,
    t.typnotnull AS not_null,
    t.typdefault AS default_value,
    d.description AS domain_comment
FROM pg_type t
JOIN pg_namespace n ON t.typnamespace = n.oid
LEFT JOIN pg_description d ON d.objoid = t.oid AND d.classoid = 'pg_type'::regclass
WHERE t.typtype = 'd'  -- Domain types only
    AND n.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND n.nspname NOT LIKE 'pg_temp_%'
    AND n.nspname NOT LIKE 'pg_toast_temp_%'
ORDER BY n.nspname, t.typname;

-- GetDomainConstraints retrieves constraints for domains
-- name: GetDomainConstraints :many
SELECT 
    n.nspname AS domain_schema,
    t.typname AS domain_name,
    c.conname AS constraint_name,
    pg_get_constraintdef(c.oid) AS constraint_definition
FROM pg_constraint c
JOIN pg_type t ON c.contypid = t.oid
JOIN pg_namespace n ON t.typnamespace = n.oid
WHERE t.typtype = 'd'  -- Domain types only
    AND n.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND n.nspname NOT LIKE 'pg_temp_%'
    AND n.nspname NOT LIKE 'pg_toast_temp_%'
ORDER BY n.nspname, t.typname, c.conname;