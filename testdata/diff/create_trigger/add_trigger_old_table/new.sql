CREATE TABLE public.orders (
    id serial PRIMARY KEY,
    amount numeric(10,2)
);

CREATE TABLE public.orders_archive (
    id integer,
    amount numeric(10,2),
    deleted_at timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION public.archive_deleted_orders()
RETURNS trigger AS $$
BEGIN
    INSERT INTO orders_archive (id, amount)
    SELECT id, amount FROM old_orders;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER orders_delete_trigger
    AFTER DELETE ON public.orders
    REFERENCING OLD TABLE AS old_orders
    FOR EACH STATEMENT
    EXECUTE FUNCTION public.archive_deleted_orders();
