CREATE TYPE status_type AS ENUM (
    'active',
    'inactive'
);

CREATE TYPE record_type AS (id integer, status status_type);
