CREATE FUNCTION my_default_id() RETURNS uuid
    LANGUAGE sql
    AS $$ SELECT gen_random_uuid() $$;

CREATE TABLE items (
    id uuid DEFAULT my_default_id() NOT NULL,
    name text NOT NULL,
    active boolean DEFAULT true NOT NULL,
    CONSTRAINT items_pk PRIMARY KEY (id)
);
