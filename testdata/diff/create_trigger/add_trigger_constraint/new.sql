CREATE TABLE public.products (
    id integer PRIMARY KEY,
    code text NOT NULL
);

CREATE OR REPLACE FUNCTION public.prevent_code_update()
RETURNS trigger AS $$
BEGIN
    IF OLD.code IS DISTINCT FROM NEW.code THEN
        RAISE EXCEPTION 'Product code cannot be updated';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER prevent_code_update_trigger
    AFTER UPDATE ON public.products
    DEFERRABLE INITIALLY IMMEDIATE
    FOR EACH ROW
    EXECUTE FUNCTION public.prevent_code_update();
