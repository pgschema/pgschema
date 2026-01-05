-- Grant USAGE on future types to app_user
ALTER DEFAULT PRIVILEGES GRANT USAGE ON TYPES TO app_user;

CREATE TYPE status AS ENUM ('pending', 'active', 'inactive');
