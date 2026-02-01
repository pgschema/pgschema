ALTER TABLE people ADD CONSTRAINT created_at_not_null CHECK (created_at IS NOT NULL) NOT VALID;

ALTER TABLE people VALIDATE CONSTRAINT created_at_not_null;

ALTER TABLE people ALTER COLUMN created_at SET NOT NULL;

ALTER TABLE people DROP CONSTRAINT created_at_not_null;

ALTER TABLE people ALTER COLUMN created_at SET DEFAULT now();
