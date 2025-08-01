CREATE DOMAIN email_address AS text
  DEFAULT 'example@acme.com'
  NOT NULL
  CONSTRAINT email_address_check CHECK (VALUE ~ '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$');