COMMENT ON INDEX idx_users_created_at IS 'Index for chronological user queries';

COMMENT ON INDEX idx_users_email IS 'Index for fast user lookup by email';

CREATE MATERIALIZED VIEW IF NOT EXISTS users_summary AS
SELECT
    email,
    created_at
FROM users;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_summary_email ON users_summary (email);

-- pgschema:wait
SELECT 
    COALESCE(i.indisvalid, false) as done,
    CASE 
        WHEN p.blocks_total > 0 THEN p.blocks_done * 100 / p.blocks_total
        ELSE 0
    END as progress
FROM pg_class c
LEFT JOIN pg_index i ON c.oid = i.indexrelid
LEFT JOIN pg_stat_progress_create_index p ON c.oid = p.index_relid
WHERE c.relname = 'idx_users_summary_email';

COMMENT ON INDEX idx_users_summary_email IS 'Index for email search on summary';
