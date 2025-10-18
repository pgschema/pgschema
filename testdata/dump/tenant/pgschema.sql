--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 1.4.0


--
-- Name: users; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS users (
    id SERIAL,
    username varchar(100) NOT NULL,
    email varchar(100) NOT NULL,
    role public.user_role DEFAULT 'user',
    status public.status DEFAULT 'active',
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
    status public.status DEFAULT 'active',
    created_at timestamp DEFAULT now(),
    CONSTRAINT posts_pkey PRIMARY KEY (id),
    CONSTRAINT posts_author_id_fkey FOREIGN KEY (author_id) REFERENCES users (id)
);

