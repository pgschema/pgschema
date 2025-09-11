-- Test forward referencing: orders table references customers table that is defined later

CREATE TABLE public.orders (
    id SERIAL PRIMARY KEY,
    order_date DATE NOT NULL,
    LIKE public.customers INCLUDING DEFAULTS
);

-- This is the template table that orders references (defined AFTER orders)
CREATE TABLE public.customers (
    customer_id INTEGER NOT NULL,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);