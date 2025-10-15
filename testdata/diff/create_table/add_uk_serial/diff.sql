ALTER TABLE users
ADD COLUMN id serial CONSTRAINT users_id_key UNIQUE;
