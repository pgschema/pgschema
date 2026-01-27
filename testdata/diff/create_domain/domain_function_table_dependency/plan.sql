CREATE OR REPLACE FUNCTION validate_custom_id(
    val text
)
RETURNS boolean
LANGUAGE plpgsql
IMMUTABLE
AS $$
BEGIN
  RETURN val IS NOT NULL AND val LIKE 'id_%' AND length(val) >= 5;
END
$$;

CREATE DOMAIN custom_id AS text
  CONSTRAINT custom_id_check CHECK (validate_custom_id(VALUE));

CREATE TABLE IF NOT EXISTS example (
    id custom_id,
    CONSTRAINT example_pkey PRIMARY KEY (id)
);
