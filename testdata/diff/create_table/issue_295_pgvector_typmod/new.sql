-- Desired state: add pgvector columns with dimensions (typmod)
-- Reproduces GitHub issue #295 where vector(384)/halfvec(384) dimensions were dropped
CREATE TABLE public.activity (
    id bigserial PRIMARY KEY,
    embedding halfvec(384)
);

CREATE INDEX activity_embedding_idx
    ON activity USING hnsw (embedding halfvec_cosine_ops);
