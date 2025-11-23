ALTER TABLE orders
ADD COLUMN id serial CONSTRAINT orders_id_key UNIQUE;

ALTER TABLE products
ADD COLUMN id integer GENERATED ALWAYS AS IDENTITY CONSTRAINT products_id_key UNIQUE;

ALTER TABLE user_permissions
ADD CONSTRAINT user_permissions_user_id_resource_id_permission_type_key UNIQUE (user_id, resource_id, permission_type);

ALTER TABLE users
ADD COLUMN id integer CONSTRAINT users_id_key UNIQUE;
