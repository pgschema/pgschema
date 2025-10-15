ALTER TABLE countries
ADD COLUMN code text CONSTRAINT countries_code_key UNIQUE;
