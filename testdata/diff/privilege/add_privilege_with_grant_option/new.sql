-- Grant with grant option - admin_user can grant to others
ALTER DEFAULT PRIVILEGES GRANT SELECT, INSERT ON TABLES TO admin_user WITH GRANT OPTION;

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);
