CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_unique_email_org ON user_profiles (email, organization_id) WHERE (deleted_at IS NULL);
