CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_invite_assignedTo" ON invite ("assignedTo");

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
WHERE lower(c.relname) = lower('idx_invite_assignedTo');

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_invite_created_invited ON invite ("createdAt", "invitedBy");

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
WHERE lower(c.relname) = lower('idx_invite_created_invited');
