--
-- Complete schema definition for tenant schemas
-- This will be loaded into public, tenant1, and tenant2 schemas
--

-- Types
CREATE TYPE user_role AS ENUM ('admin', 'user');
CREATE TYPE status AS ENUM ('active', 'inactive');

-- Cross-schema type reference test cases
CREATE TYPE priority_level AS ENUM ('low', 'medium', 'high', 'urgent');

CREATE TYPE task_assignment AS (
    assignee_name text,
    priority priority_level,
    estimated_hours integer
);

-- Function to get default status as text (in same schema as table)
-- Issue #218: This tests when function and type are in the same schema as the column
-- Returns TEXT so that explicit cast to status is needed
-- Must be defined before table that uses it
CREATE FUNCTION get_default_status_text()
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT 'active'::text
$$;

-- Shared categories table
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name varchar(100) NOT NULL UNIQUE,
    description text
);

-- Users table (uses util.generate_id() for default user codes and util.extract_domain() for generated column)
-- Also tests schema-qualified type casts in default expressions (Issue #218)
CREATE TABLE users (
    id SERIAL,
    username varchar(100) NOT NULL,
    email varchar(100) NOT NULL,
    website varchar(255),
    user_code text DEFAULT util.generate_id(),
    domain text GENERATED ALWAYS AS (util.extract_domain(website)) STORED,
    role user_role DEFAULT 'user',
    -- Bug reproduction case 1: cross-schema function with type cast to local enum
    account_status status DEFAULT (util.get_default_status())::status,
    -- Bug reproduction case 2 (Issue #218): same-schema function returning TEXT with explicit cast to status
    -- When dumping from tenant1 schema, PostgreSQL stores this with qualified type cast
    -- Expected: (get_default_status_text())::tenant1.status (type qualifier preserved)
    -- Bug in v1.5.1: (get_default_status_text())::status (type qualifier stripped)
    secondary_status status DEFAULT (get_default_status_text())::status,
    created_at timestamp DEFAULT now(),
    CONSTRAINT users_pkey PRIMARY KEY (id)
);

-- Index on users
CREATE INDEX idx_users_email ON users (email);

-- Posts table
CREATE TABLE posts (
    id SERIAL,
    title varchar(200) NOT NULL,
    content text,
    author_id integer,
    status status DEFAULT 'active',
    created_at timestamp DEFAULT now(),
    CONSTRAINT posts_pkey PRIMARY KEY (id),
    CONSTRAINT posts_author_id_fkey FOREIGN KEY (author_id) REFERENCES users (id)
);

-- Functions to test cross-schema type references
CREATE FUNCTION set_task_priority(level priority_level)
RETURNS void
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE NOTICE 'Setting priority to: %', level;
END;
$$;

-- Function that uses util schema function
CREATE FUNCTION generate_task_id()
RETURNS text
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN 'TASK-' || util.generate_id();
END;
$$;

CREATE FUNCTION create_task_assignment(name text, priority priority_level, hours integer)
RETURNS task_assignment
LANGUAGE plpgsql
AS $$
DECLARE
    result task_assignment;
BEGIN
    result.assignee_name := name;
    result.priority := priority;
    result.estimated_hours := hours;
    RETURN result;
END;
$$;

-- Procedure to test cross-schema type references
CREATE PROCEDURE assign_task(
    IN task_id integer,
    IN priority priority_level,
    IN assignment task_assignment
)
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE NOTICE 'Assigning task % with priority % to %',
        task_id, priority, assignment.assignee_name;
END;
$$;