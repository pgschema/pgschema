CREATE INDEX IF NOT EXISTS idx_users_email_status ON users (email, status DESC);
