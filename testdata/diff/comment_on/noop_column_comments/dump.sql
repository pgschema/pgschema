--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.0.3


--
-- Name: users; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username varchar(50) NOT NULL UNIQUE,
    email varchar(255) NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp DEFAULT CURRENT_TIMESTAMP
);


COMMENT ON COLUMN users.id IS 'Unique user identifier';


COMMENT ON COLUMN users.username IS 'Unique username for login';


COMMENT ON COLUMN users.email IS 'User email address';


COMMENT ON COLUMN users.created_at IS 'When the user was created';


COMMENT ON COLUMN users.updated_at IS 'When the user was last updated';

