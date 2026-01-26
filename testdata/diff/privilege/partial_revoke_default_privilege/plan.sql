CREATE TABLE IF NOT EXISTS readonly_data (
    id integer,
    value text,
    CONSTRAINT readonly_data_pkey PRIMARY KEY (id)
);

REVOKE DELETE, INSERT, UPDATE ON TABLE readonly_data FROM app_user;
