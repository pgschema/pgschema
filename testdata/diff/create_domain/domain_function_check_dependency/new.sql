-- Function that validates a custom ID format
CREATE OR REPLACE FUNCTION validate_custom_id(val text)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
AS $$
BEGIN
  -- Simple validation: must start with 'id_' and be at least 5 characters
  RETURN val IS NOT NULL AND val LIKE 'id_%' AND length(val) >= 5;
END
$$;

-- Domain that uses the function in its CHECK constraint
CREATE DOMAIN custom_id AS text
  CHECK (validate_custom_id(VALUE));
