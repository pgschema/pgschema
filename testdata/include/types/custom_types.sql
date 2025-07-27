CREATE TYPE user_status AS ENUM ('active', 'inactive');

CREATE TYPE order_status AS ENUM ('pending', 'completed');

CREATE TYPE address AS (
    street TEXT,
    city TEXT
);