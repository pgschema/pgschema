ALTER TABLE documents
ADD COLUMN id uuid CONSTRAINT documents_id_key UNIQUE;
