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

-- GetSchema retrieves a specific schema by name
-- name: GetSchema :one
SELECT 
    schema_name
FROM information_schema.schemata
WHERE 
    schema_name = $1
    AND schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND schema_name NOT LIKE 'pg_temp_%'
    AND schema_name NOT LIKE 'pg_toast_temp_%';

-- GetTables retrieves all tables in the database with metadata
-- name: GetTables :many
SELECT 
    t.table_schema,
    t.table_name,
    t.table_type,
    COALESCE(d.description, '') AS table_comment
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

-- GetTablesForSchema retrieves all tables in a specific schema with metadata
-- name: GetTablesForSchema :many
SELECT 
    t.table_schema,
    t.table_name,
    t.table_type,
    COALESCE(d.description, '') AS table_comment
FROM information_schema.tables t
LEFT JOIN pg_class c ON c.relname = t.table_name
LEFT JOIN pg_namespace n ON c.relnamespace = n.oid AND n.nspname = t.table_schema
LEFT JOIN pg_description d ON d.objoid = c.oid AND d.classoid = 'pg_class'::regclass AND d.objsubid = 0
WHERE 
    t.table_schema = $1
    AND t.table_type IN ('BASE TABLE', 'VIEW')
ORDER BY t.table_name;

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
    COALESCE(d.description, '') AS column_comment,
    CASE 
        WHEN dt.typtype = 'd' THEN 
            CASE WHEN dn.nspname = c.table_schema THEN dt.typname 
                 ELSE dn.nspname || '.' || dt.typname 
            END
        WHEN dt.typtype = 'e' OR dt.typtype = 'c' THEN 
            CASE WHEN dn.nspname = c.table_schema THEN dt.typname 
                 ELSE dn.nspname || '.' || dt.typname 
            END
        ELSE c.udt_name
    END AS resolved_type,
    c.is_identity,
    c.identity_generation,
    c.identity_start,
    c.identity_increment,
    c.identity_maximum,
    c.identity_minimum,
    c.identity_cycle
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

-- GetColumnsForSchema retrieves all columns for tables in a specific schema
-- name: GetColumnsForSchema :many
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
    COALESCE(d.description, '') AS column_comment,
    CASE 
        WHEN dt.typtype = 'd' THEN 
            CASE WHEN dn.nspname = c.table_schema THEN dt.typname 
                 ELSE dn.nspname || '.' || dt.typname 
            END
        WHEN dt.typtype = 'e' OR dt.typtype = 'c' THEN 
            CASE WHEN dn.nspname = c.table_schema THEN dt.typname 
                 ELSE dn.nspname || '.' || dt.typname 
            END
        ELSE c.udt_name
    END AS resolved_type,
    c.is_identity,
    c.identity_generation,
    c.identity_start,
    c.identity_increment,
    c.identity_maximum,
    c.identity_minimum,
    c.identity_cycle
FROM information_schema.columns c
LEFT JOIN pg_namespace n ON n.nspname = c.table_schema
LEFT JOIN pg_class cl ON cl.relname = c.table_name AND cl.relnamespace = n.oid
LEFT JOIN pg_description d ON d.objoid = cl.oid AND d.classoid = 'pg_class'::regclass AND d.objsubid = c.ordinal_position
LEFT JOIN pg_attribute a ON a.attrelid = cl.oid AND a.attname = c.column_name
LEFT JOIN pg_attrdef ad ON ad.adrelid = a.attrelid AND ad.adnum = a.attnum
LEFT JOIN pg_type dt ON dt.oid = a.atttypid
LEFT JOIN pg_namespace dn ON dt.typnamespace = dn.oid
WHERE 
    c.table_schema = $1
ORDER BY c.table_name, c.ordinal_position;

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
    COALESCE(fn.nspname, '') AS foreign_table_schema,
    COALESCE(fcl.relname, '') AS foreign_table_name,
    COALESCE(fa.attname, '') AS foreign_column_name,
    COALESCE(fa.attnum, 0) AS foreign_ordinal_position,
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
    idx.indisunique as is_unique,
    idx.indisprimary as is_primary,
    (idx.indpred IS NOT NULL) as is_partial,
    am.amname as method,
    pg_get_indexdef(idx.indexrelid) as indexdef,
    CASE 
        WHEN idx.indpred IS NOT NULL THEN pg_get_expr(idx.indpred, idx.indrelid)
        ELSE NULL
    END as partial_predicate,
    CASE 
        WHEN idx.indexprs IS NOT NULL THEN true
        ELSE false
    END as has_expressions
FROM pg_index idx
JOIN pg_class i ON i.oid = idx.indexrelid
JOIN pg_class t ON t.oid = idx.indrelid
JOIN pg_namespace n ON n.oid = t.relnamespace
JOIN pg_am am ON am.oid = i.relam
WHERE 
    NOT idx.indisprimary
    AND NOT EXISTS (
        SELECT 1 FROM pg_constraint c 
        WHERE c.conindid = idx.indexrelid 
        AND c.contype IN ('u', 'p')
    )
    AND n.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND n.nspname NOT LIKE 'pg_temp_%'
    AND n.nspname NOT LIKE 'pg_toast_temp_%'
ORDER BY n.nspname, t.relname, i.relname;

-- GetIndexesForSchema retrieves all indexes for a specific schema
-- name: GetIndexesForSchema :many
SELECT 
    n.nspname as schemaname,
    t.relname as tablename,
    i.relname as indexname,
    idx.indisunique as is_unique,
    idx.indisprimary as is_primary,
    (idx.indpred IS NOT NULL) as is_partial,
    am.amname as method,
    pg_get_indexdef(idx.indexrelid) as indexdef,
    CASE 
        WHEN idx.indpred IS NOT NULL THEN pg_get_expr(idx.indpred, idx.indrelid)
        ELSE NULL
    END as partial_predicate,
    CASE 
        WHEN idx.indexprs IS NOT NULL THEN true
        ELSE false
    END as has_expressions,
    COALESCE(d.description, '') AS index_comment
FROM pg_index idx
JOIN pg_class i ON i.oid = idx.indexrelid
JOIN pg_class t ON t.oid = idx.indrelid
JOIN pg_namespace n ON n.oid = t.relnamespace
JOIN pg_am am ON am.oid = i.relam
LEFT JOIN pg_description d ON d.objoid = i.oid AND d.objsubid = 0
WHERE 
    NOT idx.indisprimary
    AND NOT EXISTS (
        SELECT 1 FROM pg_constraint c 
        WHERE c.conindid = idx.indexrelid 
        AND c.contype IN ('u', 'p')
    )
    AND n.nspname = $1
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
    COALESCE(desc_func.description, '') AS function_comment,
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
    COALESCE(desc_proc.description, '') AS procedure_comment,
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
    COALESCE(tf.proname, '') AS transition_function,
    COALESCE(tfn.nspname, '') AS transition_function_schema,
    -- Get state type
    format_type(a.aggtranstype, NULL) AS state_type,
    -- Get initial condition
    a.agginitval AS initial_condition,
    -- Get final function if exists
    COALESCE(ff.proname, '') AS final_function,
    COALESCE(ffn.nspname, '') AS final_function_schema,
    -- Comment
    COALESCE(d.description, '') AS aggregate_comment
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

-- GetViews retrieves all views and materialized views
-- name: GetViews :many
SELECT 
    n.nspname AS table_schema,
    c.relname AS table_name,
    pg_get_viewdef(c.oid, true) AS view_definition,
    COALESCE(d.description, '') AS view_comment,
    (c.relkind = 'm') AS is_materialized
FROM pg_class c
JOIN pg_namespace n ON c.relnamespace = n.oid
LEFT JOIN pg_description d ON d.objoid = c.oid AND d.classoid = 'pg_class'::regclass AND d.objsubid = 0
WHERE 
    c.relkind IN ('v', 'm') -- views and materialized views
    AND n.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND n.nspname NOT LIKE 'pg_temp_%'
    AND n.nspname NOT LIKE 'pg_toast_temp_%'
ORDER BY n.nspname, c.relname;


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
    COALESCE(d.description, '') AS type_comment
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
    action_statement,
    action_condition,
    action_orientation
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
    roles,
    cmd,
    qual,
    with_check
FROM pg_policies
WHERE 
    schemaname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND schemaname NOT LIKE 'pg_temp_%'
    AND schemaname NOT LIKE 'pg_toast_temp_%'
ORDER BY schemaname, tablename, policyname;

-- GetRLSTablesForSchema retrieves tables with row level security enabled for a specific schema
-- name: GetRLSTablesForSchema :many
SELECT 
    schemaname,
    tablename
FROM pg_tables
WHERE 
    schemaname = $1
    AND rowsecurity = true
ORDER BY schemaname, tablename;

-- GetRLSPoliciesForSchema retrieves all row level security policies for a specific schema
-- name: GetRLSPoliciesForSchema :many
SELECT 
    schemaname,
    tablename,
    policyname,
    permissive,
    roles,
    cmd,
    qual,
    with_check
FROM pg_policies
WHERE 
    schemaname = $1
ORDER BY schemaname, tablename, policyname;

-- GetDomains retrieves all user-defined domains
-- name: GetDomains :many
SELECT 
    n.nspname AS domain_schema,
    t.typname AS domain_name,
    format_type(t.typbasetype, t.typtypmod) AS base_type,
    t.typnotnull AS not_null,
    t.typdefault AS default_value,
    COALESCE(d.description, '') AS domain_comment
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

-- GetPartitionedTablesForSchema retrieves partition information for partitioned tables in a specific schema
-- name: GetPartitionedTablesForSchema :many
SELECT 
    n.nspname AS table_schema,
    c.relname AS table_name,
    CASE pt.partstrat
        WHEN 'r' THEN 'RANGE'
        WHEN 'l' THEN 'LIST'
        WHEN 'h' THEN 'HASH'
        ELSE 'UNKNOWN'
    END AS partition_strategy,
    STRING_AGG(a.attname, ', ' ORDER BY a.attnum) AS partition_key
FROM pg_partitioned_table pt
JOIN pg_class c ON pt.partrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
JOIN pg_attribute a ON a.attrelid = pt.partrelid AND a.attnum = ANY(pt.partattrs)
WHERE n.nspname = $1
GROUP BY n.nspname, c.relname, pt.partstrat
ORDER BY n.nspname, c.relname;

-- GetPartitionChildren retrieves partition child tables and their attachment information
-- name: GetPartitionChildren :many
SELECT 
    pn.nspname AS parent_schema,
    pc.relname AS parent_table,
    cn.nspname AS child_schema,
    cc.relname AS child_table,
    pg_get_expr(cc.relpartbound, cc.oid) AS partition_bound
FROM pg_inherits inh
JOIN pg_class pc ON inh.inhparent = pc.oid
JOIN pg_namespace pn ON pc.relnamespace = pn.oid
JOIN pg_class cc ON inh.inhrelid = cc.oid
JOIN pg_namespace cn ON cc.relnamespace = cn.oid
WHERE pn.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND pn.nspname NOT LIKE 'pg_temp_%'
    AND pn.nspname NOT LIKE 'pg_toast_temp_%'
    AND cn.nspname NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
    AND cn.nspname NOT LIKE 'pg_temp_%'
    AND cn.nspname NOT LIKE 'pg_toast_temp_%'
    AND EXISTS (
        SELECT 1 FROM pg_partitioned_table pt 
        WHERE pt.partrelid = pc.oid
    )
ORDER BY pn.nspname, pc.relname, cn.nspname, cc.relname;


-- GetConstraintsForSchema retrieves all table constraints for a specific schema
-- name: GetConstraintsForSchema :many
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
    COALESCE(fn.nspname, '') AS foreign_table_schema,
    COALESCE(fcl.relname, '') AS foreign_table_name,
    COALESCE(fa.attname, '') AS foreign_column_name,
    COALESCE(fa.attnum, 0) AS foreign_ordinal_position,
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
WHERE n.nspname = $1
ORDER BY n.nspname, cl.relname, c.contype, c.conname, a.attnum;

-- GetSequencesForSchema retrieves all sequences for a specific schema
-- name: GetSequencesForSchema :many
SELECT 
    s.sequence_schema,
    s.sequence_name,
    s.data_type,
    s.start_value,
    s.minimum_value,
    s.maximum_value,
    s.increment,
    s.cycle_option,
    COALESCE(dep_table.relname, col_table.table_name) AS owned_by_table,
    COALESCE(dep_col.attname, col_table.column_name) AS owned_by_column
FROM information_schema.sequences s
LEFT JOIN pg_class c ON c.relname = s.sequence_name
LEFT JOIN pg_namespace n ON c.relnamespace = n.oid AND n.nspname = s.sequence_schema
-- Method 1: Try to find dependency relationship (for proper SERIAL columns)
LEFT JOIN pg_depend d ON d.objid = c.oid AND d.classid = 'pg_class'::regclass AND d.deptype = 'a'
LEFT JOIN pg_class dep_table ON d.refobjid = dep_table.oid
LEFT JOIN pg_attribute dep_col ON dep_col.attrelid = dep_table.oid AND dep_col.attnum = d.refobjsubid
-- Method 2: Find sequences used in column defaults (for nextval() patterns)
LEFT JOIN (
    SELECT 
        col.table_name,
        col.column_name,
        REGEXP_REPLACE(
            REGEXP_REPLACE(col.column_default, 'nextval\(''([^'']+)''.*\)', '\1'),
            '^[^.]*\.', ''
        ) AS sequence_name
    FROM information_schema.columns col
    WHERE col.table_schema = $1
      AND col.column_default LIKE '%nextval%'
) col_table ON col_table.sequence_name = s.sequence_name
WHERE s.sequence_schema = $1
ORDER BY s.sequence_schema, s.sequence_name;

-- GetFunctionsForSchema retrieves all user-defined functions for a specific schema
-- name: GetFunctionsForSchema :many
SELECT 
    r.routine_schema,
    r.routine_name,
    r.routine_definition,
    r.routine_type,
    COALESCE(pg_get_function_result(p.oid), r.data_type) AS data_type,
    r.external_language,
    COALESCE(desc_func.description, '') AS function_comment,
    oidvectortypes(p.proargtypes) AS function_arguments,
    pg_get_function_arguments(p.oid) AS function_signature,
    CASE p.provolatile
        WHEN 'i' THEN 'IMMUTABLE'
        WHEN 's' THEN 'STABLE'
        WHEN 'v' THEN 'VOLATILE'
        ELSE NULL
    END AS volatility,
    p.proisstrict AS is_strict,
    p.prosecdef AS is_security_definer,
    p.proargmodes::text[] as proargmodes,
    p.proargnames,
    p.proallargtypes::oid[]::text[] as proallargtypes
FROM information_schema.routines r
LEFT JOIN pg_proc p ON p.proname = r.routine_name 
    AND p.pronamespace = (SELECT oid FROM pg_namespace WHERE nspname = r.routine_schema)
LEFT JOIN pg_depend d ON d.objid = p.oid AND d.deptype = 'e'
LEFT JOIN pg_description desc_func ON desc_func.objoid = p.oid AND desc_func.classoid = 'pg_proc'::regclass
WHERE r.routine_schema = $1
    AND r.routine_type = 'FUNCTION'
    AND d.objid IS NULL  -- Exclude functions that are extension members
ORDER BY r.routine_schema, r.routine_name;

-- GetProceduresForSchema retrieves all user-defined procedures for a specific schema
-- name: GetProceduresForSchema :many
SELECT 
    r.routine_schema,
    r.routine_name,
    r.routine_definition,
    r.routine_type,
    r.external_language,
    COALESCE(desc_proc.description, '') AS procedure_comment,
    oidvectortypes(p.proargtypes) AS procedure_arguments,
    pg_get_function_arguments(p.oid) AS procedure_signature
FROM information_schema.routines r
LEFT JOIN pg_proc p ON p.proname = r.routine_name 
    AND p.pronamespace = (SELECT oid FROM pg_namespace WHERE nspname = r.routine_schema)
LEFT JOIN pg_depend d ON d.objid = p.oid AND d.deptype = 'e'
LEFT JOIN pg_description desc_proc ON desc_proc.objoid = p.oid AND desc_proc.classoid = 'pg_proc'::regclass
WHERE r.routine_schema = $1
    AND r.routine_type = 'PROCEDURE'
    AND d.objid IS NULL  -- Exclude procedures that are extension members
ORDER BY r.routine_schema, r.routine_name;

-- GetAggregatesForSchema retrieves all user-defined aggregates for a specific schema
-- name: GetAggregatesForSchema :many
SELECT 
    n.nspname AS aggregate_schema,
    p.proname AS aggregate_name,
    pg_get_function_arguments(p.oid) AS aggregate_signature,
    oidvectortypes(p.proargtypes) AS aggregate_arguments,
    format_type(p.prorettype, NULL) AS aggregate_return_type,
    -- Get transition function
    COALESCE(tf.proname, '') AS transition_function,
    COALESCE(tfn.nspname, '') AS transition_function_schema,
    -- Get state type
    format_type(a.aggtranstype, NULL) AS state_type,
    -- Get initial condition
    a.agginitval AS initial_condition,
    -- Get final function if exists
    COALESCE(ff.proname, '') AS final_function,
    COALESCE(ffn.nspname, '') AS final_function_schema,
    -- Comment
    COALESCE(d.description, '') AS aggregate_comment
FROM pg_proc p
JOIN pg_namespace n ON p.pronamespace = n.oid
JOIN pg_aggregate a ON a.aggfnoid = p.oid
LEFT JOIN pg_proc tf ON a.aggtransfn = tf.oid
LEFT JOIN pg_namespace tfn ON tf.pronamespace = tfn.oid
LEFT JOIN pg_proc ff ON a.aggfinalfn = ff.oid
LEFT JOIN pg_namespace ffn ON ff.pronamespace = ffn.oid
LEFT JOIN pg_description d ON d.objoid = p.oid AND d.classoid = 'pg_proc'::regclass
WHERE p.prokind = 'a'  -- Only aggregates
    AND n.nspname = $1
    AND NOT EXISTS (
        SELECT 1 FROM pg_depend dep 
        WHERE dep.objid = p.oid AND dep.deptype = 'e'
    )  -- Exclude extension members
ORDER BY n.nspname, p.proname;

-- GetViewsForSchema retrieves all views and materialized views for a specific schema
-- name: GetViewsForSchema :many
SELECT 
    n.nspname AS table_schema,
    c.relname AS table_name,
    pg_get_viewdef(c.oid, true) AS view_definition,
    COALESCE(d.description, '') AS view_comment,
    (c.relkind = 'm') AS is_materialized
FROM pg_class c
JOIN pg_namespace n ON c.relnamespace = n.oid
LEFT JOIN pg_description d ON d.objoid = c.oid AND d.classoid = 'pg_class'::regclass AND d.objsubid = 0
WHERE 
    c.relkind IN ('v', 'm') -- views and materialized views
    AND n.nspname = $1
ORDER BY n.nspname, c.relname;

-- GetTriggersForSchema retrieves all triggers for a specific schema
-- name: GetTriggersForSchema :many
SELECT 
    trigger_schema,
    trigger_name,
    event_object_table,
    action_timing,
    event_manipulation,
    action_statement,
    action_condition,
    action_orientation
FROM information_schema.triggers
WHERE trigger_schema = $1
ORDER BY trigger_schema, event_object_table, trigger_name;

-- GetTypesForSchema retrieves all user-defined types for a specific schema
-- name: GetTypesForSchema :many
SELECT 
    n.nspname AS type_schema,
    t.typname AS type_name,
    CASE t.typtype
        WHEN 'e' THEN 'ENUM'
        WHEN 'c' THEN 'COMPOSITE'
        ELSE 'OTHER'
    END AS type_kind,
    COALESCE(d.description, '') AS type_comment
FROM pg_type t
JOIN pg_namespace n ON t.typnamespace = n.oid
LEFT JOIN pg_description d ON d.objoid = t.oid AND d.classoid = 'pg_type'::regclass
LEFT JOIN pg_class c ON t.typrelid = c.oid
WHERE t.typtype IN ('e', 'c')  -- ENUM and composite types only
    AND n.nspname = $1
    AND (t.typtype = 'e' OR (t.typtype = 'c' AND c.relkind = 'c'))  -- For composite types, only include true composite types (not table types)
ORDER BY n.nspname, t.typname;

-- GetDomainsForSchema retrieves all user-defined domains for a specific schema
-- name: GetDomainsForSchema :many
SELECT 
    n.nspname AS domain_schema,
    t.typname AS domain_name,
    format_type(t.typbasetype, t.typtypmod) AS base_type,
    t.typnotnull AS not_null,
    t.typdefault AS default_value,
    COALESCE(d.description, '') AS domain_comment
FROM pg_type t
JOIN pg_namespace n ON t.typnamespace = n.oid
LEFT JOIN pg_description d ON d.objoid = t.oid AND d.classoid = 'pg_type'::regclass
WHERE t.typtype = 'd'  -- Domain types only
    AND n.nspname = $1
ORDER BY n.nspname, t.typname;

-- GetDomainConstraintsForSchema retrieves constraints for domains in a specific schema
-- name: GetDomainConstraintsForSchema :many
SELECT 
    n.nspname AS domain_schema,
    t.typname AS domain_name,
    c.conname AS constraint_name,
    pg_get_constraintdef(c.oid) AS constraint_definition
FROM pg_constraint c
JOIN pg_type t ON c.contypid = t.oid
JOIN pg_namespace n ON t.typnamespace = n.oid
WHERE t.typtype = 'd'  -- Domain types only
    AND n.nspname = $1
ORDER BY n.nspname, t.typname, c.conname;

-- GetEnumValuesForSchema retrieves enum values for ENUM types in a specific schema
-- name: GetEnumValuesForSchema :many
SELECT 
    n.nspname AS type_schema,
    t.typname AS type_name,
    e.enumlabel AS enum_value,
    e.enumsortorder AS enum_order
FROM pg_enum e
JOIN pg_type t ON e.enumtypid = t.oid
JOIN pg_namespace n ON t.typnamespace = n.oid
WHERE n.nspname = $1
ORDER BY n.nspname, t.typname, e.enumsortorder;

-- GetCompositeTypeColumnsForSchema retrieves columns for composite types in a specific schema
-- name: GetCompositeTypeColumnsForSchema :many
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
    AND n.nspname = $1
ORDER BY n.nspname, t.typname, a.attnum;