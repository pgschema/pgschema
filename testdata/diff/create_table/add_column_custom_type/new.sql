-- New state: Add columns using custom types from setup.sql
-- Types (email_address, user_status) are created in setup.sql
-- Use unqualified names - types will be resolved via setup.sql context

-- Modified table with new columns using custom types
CREATE TABLE public.users (
    id bigint PRIMARY KEY,
    username text NOT NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    -- New columns using types from setup.sql
    email email_address NOT NULL,
    status user_status NOT NULL DEFAULT 'active'
);
