ALTER TABLE code
ADD CONSTRAINT code_check CHECK (code > 0 AND code < 255);
