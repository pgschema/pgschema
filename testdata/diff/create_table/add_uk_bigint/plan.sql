ALTER TABLE transactions
ADD COLUMN id bigint CONSTRAINT transactions_id_key UNIQUE;
