DROP INDEX IF EXISTS idx_user_summary_email;

CREATE INDEX IF NOT EXISTS idx_user_summary_email ON user_summary (email, status);
