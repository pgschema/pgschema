ALTER TABLE users
ADD COLUMN id integer CONSTRAINT users_id_key UNIQUE;
