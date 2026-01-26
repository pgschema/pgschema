CREATE TABLE IF NOT EXISTS secrets (
    id integer,
    data text,
    CONSTRAINT secrets_pkey PRIMARY KEY (id)
);

REVOKE SELECT ON TABLE secrets FROM reader;
