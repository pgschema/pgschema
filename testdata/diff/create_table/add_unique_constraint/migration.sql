ALTER TABLE user_sessions
ADD CONSTRAINT user_sessions_token_device_key UNIQUE (session_token, device_fingerprint);

ALTER TABLE user_sessions
ADD CONSTRAINT user_sessions_user_device_key UNIQUE (user_id, device_fingerprint);