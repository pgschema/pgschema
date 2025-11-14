CREATE TYPE status AS ENUM (
    'pending',
    'active',
    'inactive',
    'archived'
);

CREATE TABLE items (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name varchar(255) NOT NULL
);
