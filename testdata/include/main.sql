-- Main schema file demonstrating \i include functionality
-- This represents a modular approach to organizing database schema
-- Includes ALL supported PostgreSQL database objects

-- Include custom types folder first (dependencies for tables)
\i types/

-- Include domain types (constrained base types)
\i domains/email_address.sql
\i domains/positive_integer.sql

-- Include sequences (may be used by tables)  
\i sequences/global_id_seq.sql
\i sequences/order_number_seq.sql

-- Include trigger function (needed by users table trigger)
\i functions/update_timestamp.sql

-- Include core tables (with their constraints, indexes, and policies)
\i tables/users.sql
\i tables/orders.sql

-- Include other functions (after tables that they reference)
\i functions/get_user_count.sql
\i functions/get_order_count.sql

-- Include procedures folder
\i procedures/

-- Include views (depend on tables and functions)
\i views/user_summary.sql
\i views/order_details.sql
