CREATE TABLE IF NOT EXISTS readonly_data (
    id integer,
    value text,
    CONSTRAINT readonly_data_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS secrets (
    id integer,
    data text,
    CONSTRAINT secrets_pkey PRIMARY KEY (id)
);

REVOKE DELETE, INSERT, UPDATE ON TABLE readonly_data FROM app_user;

REVOKE SELECT ON TABLE secrets FROM reader;
