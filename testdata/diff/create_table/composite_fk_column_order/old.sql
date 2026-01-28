-- Test case: Composite FK where FK column order differs from table column order
-- This tests that constraint column ordering is preserved independently of table column order

-- Referenced table with composite primary key
CREATE TABLE public.orders (
    order_id integer NOT NULL,
    customer_id integer NOT NULL,
    name varchar(255) NOT NULL,
    CONSTRAINT orders_pkey PRIMARY KEY (customer_id, order_id)
);

-- Referencing table where columns are defined in OPPOSITE order from how FK references them
CREATE TABLE public.order_items (
    id serial PRIMARY KEY,
    -- Table defines columns in this order: order_id first (lower attnum), then customer_id (higher attnum)
    order_id integer NOT NULL,
    customer_id integer NOT NULL,
    quantity integer NOT NULL
);
