--
-- pgschema plan
--
-- This migration cannot run inside a transaction block.
-- Manual intervention may be required if it fails partway through.
--

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_invite_assignedTo ON invite ("assignedTo");

SELECT 
    COALESCE(i.indisvalid, false) as done,
    CASE 
        WHEN p.blocks_total > 0 THEN p.blocks_done * 100 / p.blocks_total
        ELSE 0
    END as progress
FROM pg_stat_progress_create_index p
LEFT JOIN pg_class c ON p.relid = c.oid
LEFT JOIN pg_indexes idx ON idx.tablename = c.relname
LEFT JOIN pg_class ic ON ic.relname = idx.indexname
LEFT JOIN pg_index i ON i.indexrelid = ic.oid
WHERE idx.indexname = 'idx_invite_assignedTo' OR p.index_relid = ic.oid;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_invite_created_invited ON invite ("createdAt", "invitedBy");

SELECT 
    COALESCE(i.indisvalid, false) as done,
    CASE 
        WHEN p.blocks_total > 0 THEN p.blocks_done * 100 / p.blocks_total
        ELSE 0
    END as progress
FROM pg_stat_progress_create_index p
LEFT JOIN pg_class c ON p.relid = c.oid
LEFT JOIN pg_indexes idx ON idx.tablename = c.relname
LEFT JOIN pg_class ic ON ic.relname = idx.indexname
LEFT JOIN pg_index i ON i.indexrelid = ic.oid
WHERE idx.indexname = 'idx_invite_created_invited' OR p.index_relid = ic.oid;