-- Basic single column case
CREATE TABLE public.users (
    id integer,
    username text NOT NULL,
    email text,
    CONSTRAINT users_id_key UNIQUE (id)
);

-- Composite UK case
CREATE TABLE public.user_permissions (
    user_id integer NOT NULL,
    resource_id integer NOT NULL,
    permission_type text NOT NULL,
    granted_at timestamp with time zone DEFAULT now(),
    UNIQUE (user_id, resource_id, permission_type)
);

-- Identity column case
CREATE TABLE public.products (
    id integer GENERATED ALWAYS AS IDENTITY,
    name text NOT NULL,
    price numeric(10,2),
    CONSTRAINT products_id_key UNIQUE (id)
);

-- Serial column case
CREATE TABLE public.orders (
    id serial,
    customer_id integer NOT NULL,
    order_date date DEFAULT CURRENT_DATE,
    CONSTRAINT orders_id_key UNIQUE (id)
);
