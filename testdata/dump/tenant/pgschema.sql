--
-- pgschema database dump
--

-- Dumped from database version PostgreSQL 17.5
-- Dumped by pgschema version 0.1.5


--
-- Name: users; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username varchar(100) NOT NULL,
    email varchar(100) NOT NULL,
    role public.user_role DEFAULT 'user',
    status public.status DEFAULT 'active',
    created_at timestamp DEFAULT now()
);

--
-- Name: idx_users_email; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX idx_users_email ON users (email);

--
-- Name: posts; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    title varchar(200) NOT NULL,
    content text,
    author_id integer,
    status public.status DEFAULT 'active',
    created_at timestamp DEFAULT now(),
    FOREIGN KEY (author_id) REFERENCES users (id)
);