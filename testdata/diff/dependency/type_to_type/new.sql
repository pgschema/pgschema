-- Enum type (dependency)
CREATE TYPE public.status_type AS ENUM (
    'active',
    'inactive'
);

-- Composite type that references the enum type
CREATE TYPE public.record_type AS (
    id integer,
    status status_type
);
