CREATE TYPE item_status AS ENUM ('pending', 'active', 'done');

CREATE FUNCTION compute_status(x integer)
RETURNS item_status
LANGUAGE plpgsql
IMMUTABLE
AS $$
BEGIN
    IF x > 0 THEN RETURN 'active'::item_status; END IF;
    IF x < 0 THEN RETURN 'done'::item_status; END IF;
    RETURN 'pending'::item_status;
END;
$$;
