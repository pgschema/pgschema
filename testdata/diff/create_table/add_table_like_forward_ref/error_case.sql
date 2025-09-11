-- Test error case: LIKE references non-existent table
CREATE TABLE public.orders (
    id SERIAL PRIMARY KEY,
    order_date DATE NOT NULL,
    LIKE public.nonexistent_table INCLUDING DEFAULTS
);