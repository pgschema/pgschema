-- Test LIKE with template table defined first (forward reference not supported with embedded postgres)

-- This is the template table that orders references (must be defined FIRST)
CREATE TABLE public.customers (
    customer_id INTEGER NOT NULL,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);

-- orders table references customers table using LIKE
CREATE TABLE public.orders (
    id SERIAL PRIMARY KEY,
    order_date DATE NOT NULL,
    LIKE public.customers INCLUDING DEFAULTS
);