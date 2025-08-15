DROP INDEX IF EXISTS idx_users_email;

ALTER INDEX idx_users_email_new RENAME TO idx_users_email;

DROP INDEX IF EXISTS idx_users_status;

ALTER INDEX idx_users_status_new RENAME TO idx_users_status;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email_new ON users (email, status);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_status_new ON users (status, department);
