--
-- Test case for GitHub issue #133: Index sorting
--
-- This test case validates that indexes are dumped in alphabetical order
-- rather than in an arbitrary order.
--
-- The indexes are intentionally created in non-alphabetical order:
-- idx_users_email, idx_users_created_at, idx_users_status,
-- idx_users_department, idx_users_last_name
--
-- Expected alphabetical order in dump output:
-- idx_users_created_at, idx_users_department, idx_users_email,
-- idx_users_last_name, idx_users_status
--

CREATE TABLE users (
    id bigint NOT NULL,
    email varchar(255) NOT NULL,
    last_name varchar(100),
    department varchar(50),
    status varchar(20),
    created_at timestamp with time zone DEFAULT now()
);

-- Create indexes in non-alphabetical order to test sorting
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_department ON users(department);
CREATE INDEX idx_users_last_name ON users(last_name);

-- Add primary key
ALTER TABLE users ADD CONSTRAINT users_pkey PRIMARY KEY (id);
