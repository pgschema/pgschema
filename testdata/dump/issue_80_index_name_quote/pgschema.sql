--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.3.0


--
-- Name: products; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS products (
    id uuid DEFAULT gen_random_uuid(),
    name text NOT NULL,
    price numeric(10,2),
    category text,
    CONSTRAINT products_pkey PRIMARY KEY (id)
);

--
-- Name: UPPER name search; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS "UPPER name search" ON products (upper(name));

--
-- Name: order; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS "order" ON products (price DESC);

--
-- Name: products_category_idx_v2; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS products_category_idx_v2 ON products (category);

--
-- Name: users; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS users (
    id uuid DEFAULT gen_random_uuid(),
    email text NOT NULL,
    username text NOT NULL,
    created_at timestamp DEFAULT now(),
    status text,
    position integer,
    department text,
    CONSTRAINT users_pkey PRIMARY KEY (id)
);

--
-- Name: UserDepartmentIndex; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS "UserDepartmentIndex" ON users (department);

--
-- Name: active users index; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS "active users index" ON users (status) WHERE (status = 'active'::text);

--
-- Name: email+username combo; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS "email+username combo" ON users (email, username);

--
-- Name: user email index; Type: INDEX; Schema: -; Owner: -
--

CREATE UNIQUE INDEX IF NOT EXISTS "user email index" ON users (email);

--
-- Name: user-status-index; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS "user-status-index" ON users (status);

--
-- Name: idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS "users.position.idx" ON users ("position");

--
-- Name: users_created_at_idx; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS users_created_at_idx ON users (created_at);

