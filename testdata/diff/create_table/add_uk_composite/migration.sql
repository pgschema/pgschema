ALTER TABLE user_permissions
ADD CONSTRAINT user_permissions_user_id_key UNIQUE (user_id, resource_id, permission_type);