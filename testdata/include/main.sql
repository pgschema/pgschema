-- Main schema file demonstrating \i include functionality
-- This represents a modular approach to organizing database schema
-- Includes ALL supported PostgreSQL database objects

-- Include custom types first (dependencies for tables)
\i types/custom_types.sql

-- Include domain types (constrained base types)
\i domains/custom_domains.sql

-- Include sequences (may be used by tables)  
\i sequences/sequences.sql

-- Include core tables (with their constraints, indexes, and policies)
\i tables/users.sql
\i tables/orders.sql

-- Include functions and procedures
\i functions/user_functions.sql
\i procedures/stored_procedures.sql

-- Include views (depend on tables and functions)
\i views/user_views.sql

-- Add some additional schema directly in main file to test mixed content
CREATE SEQUENCE inline_test_seq START WITH 5000;