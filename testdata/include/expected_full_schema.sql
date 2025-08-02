-- Main schema file demonstrating \i include functionality
-- This represents a modular approach to organizing database schema
-- Includes ALL supported PostgreSQL database objects

-- Include custom types first (dependencies for tables)
--
-- Name: user_status; Type: TYPE; Schema: -; Owner: -
--

CREATE TYPE user_status AS ENUM (
    'active',
    'inactive'
);
--
-- Name: order_status; Type: TYPE; Schema: -; Owner: -
--

CREATE TYPE order_status AS ENUM (
    'pending',
    'completed'
);
--
-- Name: address; Type: TYPE; Schema: -; Owner: -
--

CREATE TYPE address AS (street text, city text);

-- Include domain types (constrained base types)
--
-- Name: email_address; Type: DOMAIN; Schema: -; Owner: -
--

CREATE DOMAIN email_address AS text
  CONSTRAINT email_address_check CHECK (VALUE ~~ '%@%');
--
-- Name: positive_integer; Type: DOMAIN; Schema: -; Owner: -
--

CREATE DOMAIN positive_integer AS integer
  CONSTRAINT positive_integer_check CHECK (VALUE > 0);

-- Include sequences (may be used by tables)  
--
-- Name: global_id_seq; Type: SEQUENCE; Schema: -; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS global_id_seq;
--
-- Name: order_number_seq; Type: SEQUENCE; Schema: -; Owner: -
--

CREATE SEQUENCE IF NOT EXISTS order_number_seq;

-- Include trigger function (needed by users table trigger)
--
-- Name: update_timestamp; Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS trigger
LANGUAGE plpgsql
SECURITY INVOKER
STABLE
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

-- Include core tables (with their constraints, indexes, and policies)
--
-- Name: users; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS users (
    id integer PRIMARY KEY,
    email text NOT NULL CHECK (email LIKE '%@%'),
    name text NOT NULL
);

COMMENT ON TABLE users IS 'User accounts';

COMMENT ON COLUMN users.email IS 'User email address';

--
-- Name: idx_users_email; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

--
-- Name: idx_users_name; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_users_name ON users (name);

--
-- Name: users; Type: TABLE; Schema: -; Owner: -
--

ALTER TABLE users ENABLE ROW LEVEL SECURITY;

--
-- Name: users_policy; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY users_policy ON users TO PUBLIC USING (true);

--
-- Name: users_update_trigger; Type: TRIGGER; Schema: -; Owner: -
--

CREATE OR REPLACE TRIGGER users_update_trigger
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();
--
-- Name: orders; Type: TABLE; Schema: -; Owner: -
--

CREATE TABLE IF NOT EXISTS orders (
    id integer PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id),
    status text DEFAULT 'pending' NOT NULL CHECK (status IN('pending', 'completed')),
    amount numeric(10,2) DEFAULT 0.00
);

COMMENT ON TABLE orders IS 'Customer orders';

COMMENT ON COLUMN orders.user_id IS 'Reference to user';

--
-- Name: idx_orders_status; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);

--
-- Name: idx_orders_user_id; Type: INDEX; Schema: -; Owner: -
--

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders (user_id);

--
-- Name: orders; Type: TABLE; Schema: -; Owner: -
--

ALTER TABLE orders ENABLE ROW LEVEL SECURITY;

--
-- Name: orders_policy; Type: POLICY; Schema: -; Owner: -
--

CREATE POLICY orders_policy ON orders TO PUBLIC USING (user_id = 1);

-- Include other functions (after tables that they reference)
--
-- Name: get_user_count; Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION get_user_count()
RETURNS integer
LANGUAGE sql
SECURITY INVOKER
VOLATILE
AS $$
    SELECT COUNT(*) FROM users;
$$;
--
-- Name: get_order_count; Type: FUNCTION; Schema: -; Owner: -
--

CREATE OR REPLACE FUNCTION get_order_count(
    user_id_param integer
)
RETURNS integer
LANGUAGE sql
SECURITY INVOKER
VOLATILE
AS $$
    SELECT COUNT(*) FROM orders WHERE user_id = user_id_param;
$$;

-- Include procedures
--
-- Name: cleanup_orders; Type: PROCEDURE; Schema: -; Owner: -
--

CREATE OR REPLACE PROCEDURE cleanup_orders()
LANGUAGE sql
AS $$
    DELETE FROM orders WHERE status = 'completed';
$$;
--
-- Name: update_status; Type: PROCEDURE; Schema: -; Owner: -
--

CREATE OR REPLACE PROCEDURE update_status(
    user_id_param integer,
    new_status text
)
LANGUAGE sql
AS $$
    UPDATE orders SET status = new_status WHERE user_id = user_id_param;
$$;

-- Include views (depend on tables and functions)
--
-- Name: user_summary; Type: VIEW; Schema: -; Owner: -
--

CREATE OR REPLACE VIEW user_summary AS
 SELECT u.id,
    u.name,
    count(o.id) AS order_count
   FROM users u
     JOIN orders o ON u.id = o.user_id
  GROUP BY u.id, u.name;;

COMMENT ON VIEW user_summary IS 'User order summary';
--
-- Name: order_details; Type: VIEW; Schema: -; Owner: -
--

CREATE OR REPLACE VIEW order_details AS
 SELECT o.id,
    o.status,
    u.name AS user_name
   FROM orders o
     JOIN users u ON o.user_id = u.id;;

COMMENT ON VIEW order_details IS 'Order details with user info';
