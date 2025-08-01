CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_created_at ON users(created_at);

COMMENT ON INDEX idx_users_email IS 'Index for fast user lookup by email';
COMMENT ON INDEX idx_users_created_at IS 'Index for chronological user queries';