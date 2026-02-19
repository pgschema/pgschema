ALTER TABLE activity ADD COLUMN embedding halfvec(384);
CREATE INDEX IF NOT EXISTS activity_embedding_idx ON activity USING hnsw (embedding halfvec_cosine_ops);
