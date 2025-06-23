CREATE FUNCTION public.migrate_add_tenant_id() RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
    table_name text;
    tables_array text[] := ARRAY['access_tokens', 'batch_changes', 'users', 'posts'];
BEGIN
    FOREACH table_name IN ARRAY tables_array
    LOOP
        EXECUTE format('ALTER TABLE %I ADD COLUMN IF NOT EXISTS tenant_id bigint', table_name);
        COMMIT;
    END LOOP;
END;
$$;