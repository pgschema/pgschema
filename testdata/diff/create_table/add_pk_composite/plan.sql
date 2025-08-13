ALTER TABLE user_permissions
ADD CONSTRAINT user_permissions_pkey PRIMARY KEY (user_id, resource_id, permission_type);