GRANT SELECT ON TABLE users TO readonly_role;

GRANT SELECT (id) ON TABLE users TO column_reader;
