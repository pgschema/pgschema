ALTER TABLE "order"
ADD COLUMN tenant_id uuid CONSTRAINT "FK_order_tenant" REFERENCES tenant (id);

CREATE INDEX IF NOT EXISTS "IDX_order_tenant_order_number" ON "order" (tenant_id, order_number);
