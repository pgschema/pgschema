CREATE DOMAIN email_address AS text 
  DEFAULT 'example@acme.com'
  NOT NULL
  CHECK (VALUE ~ '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$');