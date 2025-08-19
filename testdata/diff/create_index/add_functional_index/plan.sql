CREATE INDEX IF NOT EXISTS idx_users_fullname_search ON users (lower(first_name), lower(last_name), lower(email));
