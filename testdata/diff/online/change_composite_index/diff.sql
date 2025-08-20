DROP INDEX IF EXISTS idx_users_email;

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email, status);

DROP INDEX IF EXISTS idx_users_status;

CREATE INDEX IF NOT EXISTS idx_users_status ON users (status, department);
