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
    category_id integer NOT NULL,
    status public.status DEFAULT 'active',
    created_at timestamp DEFAULT now(),
    CONSTRAINT posts_pkey PRIMARY KEY (id),
    CONSTRAINT posts_author_id_fkey FOREIGN KEY (author_id) REFERENCES users (id),
    CONSTRAINT posts_category_id_fkey FOREIGN KEY (category_id) REFERENCES public.categories (id)
);

--
-- Name: active_posts_mv; Type: MATERIALIZED VIEW; Schema: -; Owner: -
--

CREATE MATERIALIZED VIEW IF NOT EXISTS active_posts_mv AS
 SELECT p.id,
    p.title,
    p.content,
    u.username AS author_name,
    c.name AS category_name,
    c.description AS category_description,
    p.created_at
   FROM posts p
     JOIN users u ON p.author_id = u.id
     JOIN public.categories c ON p.category_id = c.id
  WHERE p.status = 'active'::public.status;

--
-- Name: idx_active_posts_category; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_active_posts_category ON active_posts_mv (category_name);

--
-- Name: user_posts_summary; Type: VIEW; Schema: -; Owner: -
--

CREATE OR REPLACE VIEW user_posts_summary AS
 SELECT u.id,
    u.username,
    u.email,
    p.title AS post_title,
    c.name AS category_name,
    p.created_at
   FROM users u
     JOIN posts p ON u.id = p.author_id
     JOIN public.categories c ON p.category_id = c.id
  WHERE u.status = 'active'::public.status;

