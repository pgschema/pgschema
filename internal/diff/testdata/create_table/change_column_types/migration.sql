ALTER TABLE user_pending_permissions ALTER COLUMN id TYPE bigint;

ALTER TABLE user_pending_permissions ALTER COLUMN user_id TYPE bigint;

ALTER TABLE user_pending_permissions ALTER COLUMN object_ids_ints TYPE bigint[];