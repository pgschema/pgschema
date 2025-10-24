--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.4.0


--
-- Name: priority_level; Type: TYPE; Schema: -; Owner: -
--

CREATE TYPE priority_level AS ENUM (
    'low',
    'medium',
    'high',
    'urgent'
);

--
-- Name: status; Type: TYPE; Schema: -; Owner: -
--

CREATE TYPE status AS ENUM (
    'active',
    'inactive'
);

--
-- Name: task_assignment; Type: TYPE; Schema: -; Owner: -
--

CREATE TYPE task_assignment AS (assignee_name text, priority priority_level, estimated_hours integer);

--
-- Name: user_role; Type: TYPE; Schema: -; Owner: -
--

CREATE TYPE user_role AS ENUM (
    'admin',
    'user'
);

--
-- Name: categories; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS categories (
    id SERIAL,
    name varchar(100) NOT NULL,
    description text,
    CONSTRAINT categories_pkey PRIMARY KEY (id),
    CONSTRAINT categories_name_key UNIQUE (name)
);

--
-- Name: users; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS users (
    id SERIAL,
    username varchar(100) NOT NULL,
    email varchar(100) NOT NULL,
    user_code text DEFAULT util.generate_id(),
    role user_role DEFAULT 'user',
    status status DEFAULT 'active',
    created_at timestamp DEFAULT now(),
    CONSTRAINT users_pkey PRIMARY KEY (id)
);

--
-- Name: idx_users_email; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

--
-- Name: posts; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS posts (
    id SERIAL,
    title varchar(200) NOT NULL,
    content text,
    author_id integer,
    status status DEFAULT 'active',
    created_at timestamp DEFAULT now(),
    CONSTRAINT posts_pkey PRIMARY KEY (id),
    CONSTRAINT posts_author_id_fkey FOREIGN KEY (author_id) REFERENCES users (id)
);

--
-- Name: create_task_assignment; Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION create_task_assignment(
    name text,
    priority priority_level,
    hours integer
)
RETURNS task_assignment
LANGUAGE plpgsql
SECURITY INVOKER
VOLATILE
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

--
-- Name: generate_task_id; Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION generate_task_id()
RETURNS text
LANGUAGE plpgsql
SECURITY INVOKER
VOLATILE
AS $$
BEGIN
    RETURN 'TASK-' || util.generate_id();
END;
$$;

--
-- Name: set_task_priority; Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION set_task_priority(
    level priority_level
)
RETURNS void
LANGUAGE plpgsql
SECURITY INVOKER
VOLATILE
AS $$
BEGIN
    RAISE NOTICE 'Setting priority to: %', level;
END;
$$;

--
-- Name: assign_task; Type: PROCEDURE; Schema: -; Owner: -
--

CREATE OR REPLACE PROCEDURE assign_task(
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

